package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const persistentStateFile = "update-state.json"

type persistentState struct {
	ActiveVersion   string    `json:"active_version"`
	ActiveSHA256    string    `json:"active_sha256"`
	PreviousVersion string    `json:"previous_version,omitempty"`
	PreviousSHA256  string    `json:"previous_sha256,omitempty"`
	Pending         bool      `json:"pending"`
	Attempted       bool      `json:"attempted"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RunningInContainer reports whether CloudCTF is running in its container
// image. The explicit marker is preferred; /.dockerenv covers compatible
// third-party images.
func RunningInContainer() bool {
	if value := strings.TrimSpace(os.Getenv("CLOUDCTF_CONTAINER")); value != "" {
		return value == "1" || strings.EqualFold(value, "true")
	}
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func installPersistentUpdate(dir, currentExecutable, currentVersion, nextVersion, nextHash, updatePath string) (string, error) {
	if _, ok := parseSemver(nextVersion); !ok {
		return "", fmt.Errorf("invalid persistent update version %q", nextVersion)
	}
	if err := os.MkdirAll(filepath.Join(dir, "versions"), 0o700); err != nil {
		return "", fmt.Errorf("create persistent update directory: %w", err)
	}

	previousHash, previousPath, err := preserveCurrentExecutable(dir, currentExecutable, currentVersion)
	if err != nil {
		return "", err
	}
	targetPath, err := persistentBinaryPath(dir, nextVersion)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return "", fmt.Errorf("create version directory: %w", err)
	}
	if err := os.Rename(updatePath, targetPath); err != nil {
		return "", fmt.Errorf("install persistent update: %w", err)
	}
	if err := os.Chmod(targetPath, 0o755); err != nil {
		return "", fmt.Errorf("make persistent update executable: %w", err)
	}

	state := persistentState{
		ActiveVersion:   nextVersion,
		ActiveSHA256:    nextHash,
		PreviousVersion: normalizeVersion(currentVersion),
		PreviousSHA256:  previousHash,
		Pending:         true,
		Attempted:       false,
		UpdatedAt:       time.Now().UTC(),
	}
	if err := writePersistentState(dir, state); err != nil {
		_ = os.Remove(targetPath)
		return "", err
	}
	return previousPath, nil
}

func preserveCurrentExecutable(dir, executable, version string) (string, string, error) {
	if _, ok := parseSemver(version); !ok {
		return "", "image://builtin", nil
	}
	target, err := persistentBinaryPath(dir, version)
	if err != nil {
		return "", "", err
	}
	currentAbs, _ := filepath.Abs(executable)
	targetAbs, _ := filepath.Abs(target)
	if samePath(currentAbs, targetAbs) {
		hash, err := fileSHA256(target)
		return hash, target, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", "", fmt.Errorf("create previous version directory: %w", err)
	}
	source, err := os.Open(executable)
	if err != nil {
		return "", "", fmt.Errorf("open current executable: %w", err)
	}
	defer source.Close()
	tmp, err := os.CreateTemp(filepath.Dir(target), ".previous-*")
	if err != nil {
		return "", "", fmt.Errorf("create previous executable copy: %w", err)
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), source); err != nil {
		return "", "", fmt.Errorf("copy current executable: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		return "", "", err
	}
	if err := tmp.Sync(); err != nil {
		return "", "", err
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return "", "", fmt.Errorf("preserve current executable: %w", err)
	}
	keep = true
	return hex.EncodeToString(hasher.Sum(nil)), target, nil
}

// SelectPersistentExecutable returns the verified persistent binary that
// should replace the currently running image binary. A pending version gets
// one boot attempt; if it did not reach MarkPersistentHealthy, the next image
// start automatically selects the preserved previous version.
func SelectPersistentExecutable(currentVersion, dir, currentExecutable string) (string, bool, error) {
	state, err := readPersistentState(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	currentAbs, err := filepath.Abs(currentExecutable)
	if err != nil {
		return "", false, err
	}
	activePath, err := persistentBinaryPath(dir, state.ActiveVersion)
	if err != nil {
		return "", false, err
	}
	activeAbs, _ := filepath.Abs(activePath)
	if samePath(currentAbs, activeAbs) {
		return "", false, nil
	}

	if state.Pending && state.Attempted {
		if path, ok := verifiedPersistentBinary(dir, state.PreviousVersion, state.PreviousSHA256); ok {
			state.ActiveVersion = state.PreviousVersion
			state.ActiveSHA256 = state.PreviousSHA256
			state.PreviousVersion = ""
			state.PreviousSHA256 = ""
			state.Pending = false
			state.Attempted = false
			state.UpdatedAt = time.Now().UTC()
			if err := writePersistentState(dir, state); err != nil {
				return "", false, err
			}
			return path, true, nil
		}
		_ = os.Remove(filepath.Join(dir, persistentStateFile))
		return "", false, nil
	}

	if !isNewerVersion(state.ActiveVersion, currentVersion) {
		return "", false, nil
	}
	path, ok := verifiedPersistentBinary(dir, state.ActiveVersion, state.ActiveSHA256)
	if !ok {
		if previous, previousOK := verifiedPersistentBinary(dir, state.PreviousVersion, state.PreviousSHA256); previousOK && isNewerVersion(state.PreviousVersion, currentVersion) {
			state.ActiveVersion = state.PreviousVersion
			state.ActiveSHA256 = state.PreviousSHA256
			state.PreviousVersion = ""
			state.PreviousSHA256 = ""
			state.Pending = false
			state.Attempted = false
			state.UpdatedAt = time.Now().UTC()
			if err := writePersistentState(dir, state); err != nil {
				return "", false, err
			}
			return previous, true, nil
		}
		return "", false, nil
	}
	if state.Pending {
		state.Attempted = true
		if err := writePersistentState(dir, state); err != nil {
			return "", false, err
		}
	}
	return path, true, nil
}

func MarkPersistentHealthy(version, dir, currentExecutable string) error {
	state, err := readPersistentState(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	activePath, err := persistentBinaryPath(dir, state.ActiveVersion)
	if err != nil {
		return err
	}
	currentAbs, _ := filepath.Abs(currentExecutable)
	activeAbs, _ := filepath.Abs(activePath)
	if !samePath(currentAbs, activeAbs) || normalizeVersion(version) != normalizeVersion(state.ActiveVersion) {
		return nil
	}
	if !state.Pending && !state.Attempted {
		return nil
	}
	state.Pending = false
	state.Attempted = false
	state.UpdatedAt = time.Now().UTC()
	return writePersistentState(dir, state)
}

func verifiedPersistentBinary(dir, version, expectedHash string) (string, bool) {
	if expectedHash == "" {
		return "", false
	}
	path, err := persistentBinaryPath(dir, version)
	if err != nil {
		return "", false
	}
	hash, err := fileSHA256(path)
	return path, err == nil && strings.EqualFold(hash, expectedHash)
}

func persistentBinaryPath(dir, version string) (string, error) {
	if _, ok := parseSemver(version); !ok {
		return "", fmt.Errorf("invalid persistent version %q", version)
	}
	name := strings.TrimPrefix(normalizeVersion(version), "v")
	return filepath.Join(dir, "versions", name, "cloudctf-api"), nil
}

func readPersistentState(dir string) (persistentState, error) {
	var state persistentState
	contents, err := os.ReadFile(filepath.Join(dir, persistentStateFile))
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(contents, &state); err != nil {
		return state, fmt.Errorf("decode persistent update state: %w", err)
	}
	return state, nil
}

func writePersistentState(dir string, state persistentState) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	tmp, err := os.CreateTemp(dir, ".update-state-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(contents); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filepath.Join(dir, persistentStateFile)); err != nil {
		return err
	}
	keep = true
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
