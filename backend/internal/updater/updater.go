package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxAssetSize int64 = 512 << 20

var (
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	semverPattern     = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)
)

type Config struct {
	Repository    string
	APIBaseURL    string
	Token         string
	Proxy         string
	Enabled       bool
	Version       string
	HTTPClient    *http.Client
	Executable    string
	Persistent    bool
	PersistentDir string
	GOOS          string
	GOARCH        string
}

type Service struct {
	repository    string
	apiBaseURL    string
	token         string
	enabled       bool
	version       string
	client        *http.Client
	executable    string
	persistent    bool
	persistentDir string
	goos          string
	goarch        string
	applyMu       sync.Mutex
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type CheckResult struct {
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version,omitempty"`
	UpdateAvailable bool      `json:"update_available"`
	Supported       bool      `json:"supported"`
	Reason          string    `json:"reason,omitempty"`
	ReleaseName     string    `json:"release_name,omitempty"`
	ReleaseNotes    string    `json:"release_notes,omitempty"`
	ReleaseURL      string    `json:"release_url,omitempty"`
	PublishedAt     time.Time `json:"published_at,omitempty"`
	AssetName       string    `json:"asset_name,omitempty"`
	AssetSize       int64     `json:"asset_size,omitempty"`
	Repository      string    `json:"repository"`
	InstallMode     string    `json:"install_mode"`
}

type ApplyResult struct {
	PreviousVersion string `json:"previous_version"`
	Version         string `json:"version"`
	AssetName       string `json:"asset_name"`
	BackupPath      string `json:"backup_path"`
	Restarting      bool   `json:"restarting"`
	InstallMode     string `json:"install_mode"`
}

func New(cfg Config) *Service {
	client := cfg.HTTPClient
	if client == nil {
		client = newHTTPClient(cfg.Proxy)
	}
	apiBaseURL := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}
	goos := strings.TrimSpace(cfg.GOOS)
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := strings.TrimSpace(cfg.GOARCH)
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return &Service{
		repository:    strings.TrimSpace(cfg.Repository),
		apiBaseURL:    apiBaseURL,
		token:         strings.TrimSpace(cfg.Token),
		enabled:       cfg.Enabled,
		version:       normalizeVersion(cfg.Version),
		client:        client,
		executable:    strings.TrimSpace(cfg.Executable),
		persistent:    cfg.Persistent,
		persistentDir: strings.TrimSpace(cfg.PersistentDir),
		goos:          goos,
		goarch:        goarch,
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func newHTTPClient(proxy string) *http.Client {
	client := &http.Client{Timeout: 10 * time.Minute}
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return client
	}
	proxyURL, err := url.Parse(proxy)
	if err != nil || proxyURL.Host == "" {
		client.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("invalid network proxy URL")
		})
		return client
	}
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		clone := transport.Clone()
		clone.Proxy = http.ProxyURL(proxyURL)
		client.Transport = clone
	}
	return client
}

func (s *Service) Check(ctx context.Context) (CheckResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	installMode := "in_place"
	if s.persistent {
		installMode = "persistent"
	}
	result := CheckResult{CurrentVersion: s.version, Repository: s.repository, InstallMode: installMode}
	if reason := s.unsupportedReason(); reason != "" {
		result.Reason = reason
		return result, nil
	}
	release, err := s.latestRelease(ctx)
	if err != nil {
		return result, err
	}
	assetName := s.assetName()
	asset, checksum, ok := releaseAssets(release, assetName)
	result.LatestVersion = normalizeVersion(release.TagName)
	result.ReleaseName = release.Name
	result.ReleaseNotes = release.Body
	result.ReleaseURL = release.HTMLURL
	result.PublishedAt = release.PublishedAt
	result.AssetName = assetName
	result.Supported = ok
	if !ok {
		result.Reason = fmt.Sprintf("release %s is missing %s or %s.sha256", release.TagName, assetName, assetName)
		return result, nil
	}
	if asset.Size <= 0 || asset.Size > maxAssetSize || checksum.Size <= 0 {
		result.Supported = false
		result.Reason = "release assets have an invalid size"
		return result, nil
	}
	result.AssetSize = asset.Size
	result.UpdateAvailable = isNewerVersion(release.TagName, s.version)
	return result, nil
}

func (s *Service) Apply(ctx context.Context, requestedTag string) (ApplyResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	if reason := s.unsupportedReason(); reason != "" {
		return ApplyResult{}, errors.New(reason)
	}
	release, err := s.latestRelease(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	requestedTag = strings.TrimSpace(requestedTag)
	if requestedTag == "" || normalizeVersion(requestedTag) != normalizeVersion(release.TagName) {
		return ApplyResult{}, fmt.Errorf("latest release changed to %s; check for updates again", release.TagName)
	}
	if !isNewerVersion(release.TagName, s.version) {
		return ApplyResult{}, fmt.Errorf("%s is not newer than current version %s", release.TagName, s.version)
	}

	previousVersion := s.version
	assetName := s.assetName()
	asset, checksumAsset, ok := releaseAssets(release, assetName)
	if !ok {
		return ApplyResult{}, fmt.Errorf("release %s is missing required assets for %s/%s", release.TagName, s.goos, s.goarch)
	}
	executable, err := s.executablePath()
	if err != nil {
		return ApplyResult{}, err
	}
	info, err := os.Stat(executable)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("inspect current executable: %w", err)
	}

	expectedHash, err := s.downloadChecksum(ctx, checksumAsset, assetName)
	if err != nil {
		return ApplyResult{}, err
	}
	stagingDir := filepath.Dir(executable)
	if s.persistent {
		if s.persistentDir == "" {
			return ApplyResult{}, errors.New("persistent update directory is not configured")
		}
		stagingDir = filepath.Join(s.persistentDir, ".staging")
		if err := os.MkdirAll(stagingDir, 0o700); err != nil {
			return ApplyResult{}, fmt.Errorf("create persistent update staging directory: %w", err)
		}
	}
	tmp, err := os.CreateTemp(stagingDir, ".cloudctf-update-*")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("create update file: %w", err)
	}
	tmpPath := tmp.Name()
	keepTemp := false
	defer func() {
		_ = tmp.Close()
		if !keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	actualHash, err := s.downloadBinary(ctx, tmp, asset)
	if err != nil {
		return ApplyResult{}, err
	}
	if !strings.EqualFold(actualHash, expectedHash) {
		return ApplyResult{}, fmt.Errorf("checksum mismatch for %s", assetName)
	}
	if err := validateLinuxBinary(tmp, s.goarch); err != nil {
		return ApplyResult{}, err
	}
	mode := info.Mode().Perm()
	if mode&0o111 == 0 {
		mode |= 0o755
	}
	if err := tmp.Chmod(mode); err != nil {
		return ApplyResult{}, fmt.Errorf("make update executable: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return ApplyResult{}, fmt.Errorf("flush update file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return ApplyResult{}, fmt.Errorf("close update file: %w", err)
	}

	backupPath := executable + ".previous"
	installMode := "in_place"
	if s.persistent {
		backupPath, err = installPersistentUpdate(s.persistentDir, executable, previousVersion, normalizeVersion(release.TagName), actualHash, tmpPath)
		installMode = "persistent"
	} else {
		err = installUpdate(executable, tmpPath, backupPath)
	}
	if err != nil {
		return ApplyResult{}, err
	}
	keepTemp = true
	s.version = normalizeVersion(release.TagName)
	return ApplyResult{
		PreviousVersion: previousVersion,
		Version:         normalizeVersion(release.TagName),
		AssetName:       assetName,
		BackupPath:      backupPath,
		Restarting:      true,
		InstallMode:     installMode,
	}, nil
}

func validateLinuxBinary(file *os.File, arch string) error {
	header := make([]byte, 20)
	if _, err := file.ReadAt(header, 0); err != nil {
		return fmt.Errorf("read update executable header: %w", err)
	}
	if string(header[:4]) != "\x7fELF" || header[4] != 2 || header[5] != 1 {
		return errors.New("downloaded update is not a 64-bit Linux ELF executable")
	}
	machine := binary.LittleEndian.Uint16(header[18:20])
	wantMachine := uint16(62)
	if arch == "arm64" {
		wantMachine = 183
	}
	if machine != wantMachine {
		return fmt.Errorf("downloaded update architecture does not match %s", arch)
	}
	return nil
}

func (s *Service) unsupportedReason() string {
	if !s.enabled {
		return "online updates are disabled by configuration"
	}
	if !repositoryPattern.MatchString(s.repository) {
		return "UPDATE_REPOSITORY must use the owner/repository format"
	}
	if _, ok := parseSemver(s.version); !ok {
		return "online updates are unavailable in development builds"
	}
	if s.goos != "linux" || (s.goarch != "amd64" && s.goarch != "arm64") {
		return fmt.Sprintf("online updates are not available for %s/%s", s.goos, s.goarch)
	}
	return ""
}

func (s *Service) latestRelease(ctx context.Context) (Release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", s.apiBaseURL, s.repository)
	var release Release
	if err := s.getJSON(ctx, endpoint, &release); err != nil {
		return release, fmt.Errorf("get latest GitHub release: %w", err)
	}
	if release.Draft || release.Prerelease || strings.TrimSpace(release.TagName) == "" {
		return release, errors.New("GitHub returned an invalid latest release")
	}
	if _, ok := parseSemver(release.TagName); !ok {
		return release, fmt.Errorf("release tag %q is not semantic versioning", release.TagName)
	}
	return release, nil
}

func (s *Service) getJSON(ctx context.Context, endpoint string, target any) error {
	response, err := s.doRequest(ctx, endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return githubStatusError(response)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}

func (s *Service) downloadChecksum(ctx context.Context, asset Asset, binaryName string) (string, error) {
	response, err := s.doRequest(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", githubStatusError(response)
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 64<<10))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) != binaryName {
			continue
		}
		hash := strings.ToLower(fields[0])
		decoded, err := hex.DecodeString(hash)
		if err == nil && len(decoded) == sha256.Size {
			return hash, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read checksum: %w", err)
	}
	return "", fmt.Errorf("checksum for %s was not found", binaryName)
}

func (s *Service) downloadBinary(ctx context.Context, target *os.File, asset Asset) (string, error) {
	if asset.Size <= 0 || asset.Size > maxAssetSize {
		return "", fmt.Errorf("invalid update size: %d", asset.Size)
	}
	response, err := s.doRequest(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", githubStatusError(response)
	}
	if response.ContentLength > maxAssetSize {
		return "", errors.New("update exceeds the maximum allowed size")
	}
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(target, hasher), io.LimitReader(response.Body, maxAssetSize+1))
	if err != nil {
		return "", fmt.Errorf("save update: %w", err)
	}
	if written > maxAssetSize {
		return "", errors.New("update exceeds the maximum allowed size")
	}
	if asset.Size > 0 && written != asset.Size {
		return "", fmt.Errorf("downloaded update size is %d; expected %d", written, asset.Size)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *Service) doRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return nil, errors.New("invalid GitHub URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "CloudCTF-Updater/"+s.version)
	apiURL, _ := url.Parse(s.apiBaseURL)
	if s.token != "" && apiURL != nil && strings.EqualFold(parsed.Host, apiURL.Host) {
		request.Header.Set("Authorization", "Bearer "+s.token)
	}
	return s.client.Do(request)
}

func (s *Service) executablePath() (string, error) {
	path := s.executable
	if path == "" {
		var err error
		path, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("locate current executable: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	return abs, nil
}

func (s *Service) assetName() string {
	return "cloudctf-" + s.goos + "-" + s.goarch
}

func releaseAssets(release Release, binaryName string) (Asset, Asset, bool) {
	var binary, checksum Asset
	for _, asset := range release.Assets {
		switch asset.Name {
		case binaryName:
			binary = asset
		case binaryName + ".sha256":
			checksum = asset
		}
	}
	return binary, checksum, binary.BrowserDownloadURL != "" && checksum.BrowserDownloadURL != ""
}

func githubStatusError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("GitHub returned HTTP %d: %s", response.StatusCode, message)
}

type semver struct {
	major, minor, patch uint64
	prerelease          []string
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "dev"
	}
	return value
}

func parseSemver(value string) (semver, bool) {
	match := semverPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return semver{}, false
	}
	major, err1 := strconv.ParseUint(match[1], 10, 64)
	minor, err2 := strconv.ParseUint(match[2], 10, 64)
	patch, err3 := strconv.ParseUint(match[3], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return semver{}, false
	}
	parsed := semver{major: major, minor: minor, patch: patch}
	if match[4] != "" {
		parsed.prerelease = strings.Split(match[4], ".")
	}
	return parsed, true
}

func isNewerVersion(candidate, current string) bool {
	candidateVersion, candidateOK := parseSemver(candidate)
	if !candidateOK {
		return false
	}
	currentVersion, currentOK := parseSemver(current)
	if !currentOK {
		return true
	}
	return compareSemver(candidateVersion, currentVersion) > 0
}

func compareSemver(left, right semver) int {
	for _, pair := range [][2]uint64{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}
	limit := len(left.prerelease)
	if len(right.prerelease) < limit {
		limit = len(right.prerelease)
	}
	for index := 0; index < limit; index++ {
		leftPart, rightPart := left.prerelease[index], right.prerelease[index]
		leftNumber, leftErr := strconv.ParseUint(leftPart, 10, 64)
		rightNumber, rightErr := strconv.ParseUint(rightPart, 10, 64)
		switch {
		case leftErr == nil && rightErr == nil && leftNumber < rightNumber:
			return -1
		case leftErr == nil && rightErr == nil && leftNumber > rightNumber:
			return 1
		case leftErr == nil && rightErr != nil:
			return -1
		case leftErr != nil && rightErr == nil:
			return 1
		case leftErr != nil && rightErr != nil && leftPart < rightPart:
			return -1
		case leftErr != nil && rightErr != nil && leftPart > rightPart:
			return 1
		}
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	return 0
}
