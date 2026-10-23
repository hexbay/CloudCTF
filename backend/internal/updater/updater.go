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

const (
	maxAssetSize             int64 = 512 << 20
	updateCheckCacheTTL            = 15 * time.Minute
	updateCheckErrorCacheTTL       = time.Minute
)

// MaxUploadSize is the maximum accepted manual update binary size.
const MaxUploadSize int64 = maxAssetSize

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
	checkCachedAt time.Time
	checkResult   CheckResult
	checkErr      error
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
	return s.check(ctx, false)
}

// Refresh bypasses the update-check cache. It is intended for explicit checks
// initiated by an administrator; automatic checks should use Check.
func (s *Service) Refresh(ctx context.Context) (CheckResult, error) {
	return s.check(ctx, true)
}

func (s *Service) check(ctx context.Context, force bool) (CheckResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if !force && !s.checkCachedAt.IsZero() {
		ttl := updateCheckCacheTTL
		if s.checkErr != nil {
			ttl = updateCheckErrorCacheTTL
		}
		if time.Since(s.checkCachedAt) < ttl {
			return s.checkResult, s.checkErr
		}
	}

	installMode := "in_place"
	if s.persistent {
		installMode = "persistent"
	}
	result := CheckResult{CurrentVersion: s.version, Repository: s.repository, InstallMode: installMode}
	if reason := s.unsupportedReason(); reason != "" {
		result.Reason = reason
		s.cacheCheck(result, nil)
		return result, nil
	}
	release, err := s.latestRelease(ctx)
	if err != nil {
		s.cacheCheck(result, err)
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
		s.cacheCheck(result, nil)
		return result, nil
	}
	if asset.Size <= 0 || asset.Size > maxAssetSize || checksum.Size <= 0 {
		result.Supported = false
		result.Reason = "release assets have an invalid size"
		s.cacheCheck(result, nil)
		return result, nil
	}
	result.AssetSize = asset.Size
	result.UpdateAvailable = isNewerVersion(release.TagName, s.version)
	s.cacheCheck(result, nil)
	return result, nil
}

func (s *Service) cacheCheck(result CheckResult, err error) {
	s.checkCachedAt = time.Now()
	s.checkResult = result
	s.checkErr = err
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

// ApplyUploaded installs a manually uploaded CloudCTF binary. The upload is
// hashed locally, so a separate checksum file is not required.
func (s *Service) ApplyUploaded(ctx context.Context, requestedVersion, uploadPath string) (ApplyResult, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	requestedVersion = normalizeVersion(requestedVersion)
	if err := s.validateUploaded(requestedVersion, uploadPath); err != nil {
		return ApplyResult{}, err
	}
	select {
	case <-ctx.Done():
		return ApplyResult{}, ctx.Err()
	default:
	}

	source, err := os.Open(uploadPath)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("打开上传文件失败：%w", err)
	}
	defer source.Close()
	sourceInfo, err := source.Stat()
	if err != nil {
		return ApplyResult{}, fmt.Errorf("读取上传文件信息失败：%w", err)
	}

	executable, err := s.executablePath()
	if err != nil {
		return ApplyResult{}, err
	}
	info, err := os.Stat(executable)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("inspect current executable: %w", err)
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
	tmp, err := os.CreateTemp(stagingDir, ".cloudctf-manual-update-*")
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

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(source, maxAssetSize+1))
	if err != nil {
		return ApplyResult{}, fmt.Errorf("保存上传文件失败：%w", err)
	}
	if written != sourceInfo.Size() || written > maxAssetSize {
		return ApplyResult{}, errors.New("上传文件不完整或超过大小限制")
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

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	previousVersion := s.version
	backupPath := executable + ".previous"
	installMode := "in_place"
	if s.persistent {
		backupPath, err = installPersistentUpdate(s.persistentDir, executable, previousVersion, requestedVersion, actualHash, tmpPath)
		installMode = "persistent"
	} else {
		err = installUpdate(executable, tmpPath, backupPath)
	}
	if err != nil {
		return ApplyResult{}, err
	}
	keepTemp = true
	s.version = requestedVersion
	return ApplyResult{
		PreviousVersion: previousVersion,
		Version:         requestedVersion,
		AssetName:       s.assetName(),
		BackupPath:      backupPath,
		Restarting:      true,
		InstallMode:     installMode,
	}, nil
}

// ValidateUploaded validates a manual update before the update job creates a
// database backup. ApplyUploaded repeats these checks before installation.
func (s *Service) ValidateUploaded(requestedVersion, uploadPath string) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return s.validateUploaded(normalizeVersion(requestedVersion), uploadPath)
}

func (s *Service) validateUploaded(requestedVersion, uploadPath string) error {
	if s.goos != "linux" || (s.goarch != "amd64" && s.goarch != "arm64") {
		return fmt.Errorf("manual updates are not available for %s/%s", s.goos, s.goarch)
	}
	if _, ok := parseSemver(requestedVersion); !ok {
		return errors.New("更新版本必须使用语义化版本号，例如 v1.0.18")
	}
	if !isNewerVersion(requestedVersion, s.version) {
		return fmt.Errorf("上传版本 %s 必须高于当前版本 %s", requestedVersion, s.version)
	}
	source, err := os.Open(uploadPath)
	if err != nil {
		return fmt.Errorf("打开上传文件失败：%w", err)
	}
	defer source.Close()
	sourceInfo, err := source.Stat()
	if err != nil {
		return fmt.Errorf("读取上传文件信息失败：%w", err)
	}
	if sourceInfo.Size() <= 0 || sourceInfo.Size() > maxAssetSize {
		return fmt.Errorf("上传文件大小无效：%d", sourceInfo.Size())
	}
	return validateLinuxBinary(source, s.goarch)
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
		return release, fmt.Errorf("获取最新 GitHub Release 失败：%w", err)
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
	message := githubErrorMessage(body)

	if response.StatusCode == http.StatusTooManyRequests ||
		(response.StatusCode == http.StatusForbidden &&
			(response.Header.Get("X-RateLimit-Remaining") == "0" || strings.Contains(strings.ToLower(message), "rate limit"))) {
		resetHint := "请稍后重试"
		if resetAt, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && resetAt > 0 {
			resetHint = fmt.Sprintf("请在 %s 后重试", time.Unix(resetAt, 0).Local().Format("2006-01-02 15:04:05"))
		}
		return fmt.Errorf("GitHub API 请求频率已达上限，%s；配置 UPDATE_GITHUB_TOKEN 可提高请求限额", resetHint)
	}

	switch response.StatusCode {
	case http.StatusUnauthorized:
		return errors.New("GitHub 更新凭据无效，请检查 UPDATE_GITHUB_TOKEN")
	case http.StatusNotFound:
		return errors.New("未找到 GitHub Release，请检查 UPDATE_REPOSITORY 配置及仓库发布状态")
	}
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("GitHub 请求失败（HTTP %d）：%s", response.StatusCode, message)
}

func githubErrorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Message) != "" {
		return strings.TrimSpace(payload.Message)
	}
	return strings.TrimSpace(string(body))
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
