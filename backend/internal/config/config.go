package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppName                string
	Host                   string
	Port                   int
	Env                    string
	LogLevel               slog.Level
	LogFile                string
	DatabaseURI            string
	RedisAddr              string
	JWTSecret              string
	FirstSuperuserUsername string
	FirstSuperuserPassword string
	UploadDir              string
	StaticDir              string
	FrontendDir            string
	BackupDir              string
	BackupRetention        int
	AllowedOrigins         []string
	ContainerLabelKey      string
	ContainerLabelValue    string
	ContainerManagedKey    string
	ContainerManagedVal    string
	UpdateEnabled          bool
	UpdateDir              string
	UpdateRepository       string
	UpdateAPIURL           string
	UpdateGitHubToken      string
	Proxy                  string
}

func Load() Config {
	ensureEnvFile()
	loadEnvFiles()
	ensureGeneratedSecrets()
	return Config{
		AppName:                env("APP_NAME", "CloudCTF API"),
		Host:                   env("HOST", "0.0.0.0"),
		Port:                   envInt("PORT", 8005),
		Env:                    env("APP_ENV", "development"),
		LogLevel:               envLogLevel("LOG_LEVEL", slog.LevelInfo),
		LogFile:                env("LOG_FILE", "logs/backend.log"),
		DatabaseURI:            env("DATABASE_URI", "sqlite://data/cloudctf.db"),
		RedisAddr:              env("REDIS_ADDR", "redis://localhost:6379/0"),
		JWTSecret:              env("JWT_SECRET", "change-me"),
		FirstSuperuserUsername: env("FIRST_SUPERUSER_USERNAME", "admin"),
		FirstSuperuserPassword: env("FIRST_SUPERUSER_PASSWORD", ""),
		UploadDir:              env("UPLOAD_DIR", "static/uploads"),
		StaticDir:              env("STATIC_DIR", "static"),
		FrontendDir:            env("FRONTEND_DIR", "../frontend/dist"),
		BackupDir:              env("BACKUP_DIR", "data/backups"),
		BackupRetention:        envInt("BACKUP_RETENTION", 7),
		AllowedOrigins:         envList("ALLOWED_ORIGINS", nil),
		ContainerLabelKey:      env("CONTAINER_LABEL_KEY", "com.cloudctf.project"),
		ContainerLabelValue:    env("CONTAINER_LABEL_VALUE", "cloudctf"),
		ContainerManagedKey:    env("CONTAINER_MANAGED_LABEL_KEY", "com.cloudctf.managed"),
		ContainerManagedVal:    env("CONTAINER_MANAGED_LABEL_VALUE", "true"),
		UpdateEnabled:          envBool("UPDATE_ENABLED", true),
		UpdateDir:              env("UPDATE_DIR", "data/updates"),
		UpdateRepository:       env("UPDATE_REPOSITORY", "hexbay/CloudCTF"),
		UpdateAPIURL:           env("UPDATE_API_URL", "https://api.github.com"),
		UpdateGitHubToken:      env("UPDATE_GITHUB_TOKEN", ""),
		Proxy:                  env("PROXY", ""),
	}
}

const fallbackEnvTemplate = `# CloudCTF configuration
APP_NAME=CloudCTF API
APP_ENV=development
HOST=0.0.0.0
PORT=8005
LOG_LEVEL=info
LOG_FILE=logs/backend.log
DATABASE_URI=sqlite://data/cloudctf.db
REDIS_ADDR=redis://localhost:6379/0
JWT_SECRET=
FIRST_SUPERUSER_USERNAME=admin
FIRST_SUPERUSER_PASSWORD=
UPLOAD_DIR=static/uploads
STATIC_DIR=static
FRONTEND_DIR=../frontend/dist
BACKUP_DIR=data/backups
BACKUP_RETENTION=7
UPDATE_ENABLED=true
UPDATE_DIR=data/updates
UPDATE_REPOSITORY=hexbay/CloudCTF
UPDATE_API_URL=https://api.github.com
# GIT 资源仓库同步和在线更新共用代理，支持 HTTP 和 SOCKS5。
# PROXY=http://127.0.0.1:10808
# PROXY=socks5://127.0.0.1:10808
`

func configFilePath() string {
	if path := strings.TrimSpace(os.Getenv("CONFIG_FILE")); path != "" {
		return path
	}
	return filepath.Join("data", ".env")
}

// ensureEnvFile creates the first-run configuration from .env.example. The
// generated file lives under data/ by default so Docker volume mounts preserve
// credentials when the container is recreated.
func ensureEnvFile() {
	path := configFilePath()
	if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
		return
	}

	template := []byte(fallbackEnvTemplate)
	for _, candidate := range []string{".env.example", filepath.Join("backend", ".env.example")} {
		if contents, err := os.ReadFile(candidate); err == nil {
			template = contents
			break
		}
	}

	jwtSecret, err := randomSecret()
	if err != nil {
		return
	}
	adminPassword, err := randomSecret()
	if err != nil {
		return
	}
	contents := replaceEnvValue(string(template), "JWT_SECRET", jwtSecret)
	contents = replaceEnvValue(contents, "FIRST_SUPERUSER_PASSWORD", adminPassword)
	if appEnv := strings.TrimSpace(os.Getenv("APP_ENV")); appEnv != "" {
		contents = replaceEnvValue(contents, "APP_ENV", appEnv)
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	if _, err = file.WriteString(contents); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(path)
	}
}

func randomSecret() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func replaceEnvValue(contents, key, value string) string {
	lines := strings.Split(strings.ReplaceAll(contents, "\r\n", "\n"), "\n")
	prefix := key + "="
	replaced := false
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[index] = prefix + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, prefix+value)
	}
	result := strings.Join(lines, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c Config) Validate() error {
	issues := []string{}
	if strings.TrimSpace(c.Host) == "" {
		issues = append(issues, "HOST must not be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		issues = append(issues, "PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(c.DatabaseURI) == "" {
		issues = append(issues, "DATABASE_URI must not be empty")
	}
	if strings.TrimSpace(c.StaticDir) == "" {
		issues = append(issues, "STATIC_DIR must not be empty")
	}
	if strings.TrimSpace(c.UploadDir) == "" {
		issues = append(issues, "UPLOAD_DIR must not be empty")
	}
	if c.BackupRetention < 0 {
		issues = append(issues, "BACKUP_RETENTION must be greater than or equal to 0")
	}
	if c.UpdateEnabled && strings.TrimSpace(c.UpdateDir) == "" {
		issues = append(issues, "UPDATE_DIR must not be empty when online updates are enabled")
	}
	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil || proxyURL.Host == "" {
			issues = append(issues, "PROXY must be a valid proxy URL")
		} else {
			switch strings.ToLower(proxyURL.Scheme) {
			case "http", "https", "socks5", "socks5h":
			default:
				issues = append(issues, "PROXY scheme must be http, https, socks5, or socks5h")
			}
		}
	}

	if c.IsProduction() {
		if strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == "change-me" {
			issues = append(issues, "JWT_SECRET could not be generated or loaded; check CONFIG_FILE permissions")
		}
		if containsStringFold(c.AllowedOrigins, "*") {
			issues = append(issues, "ALLOWED_ORIGINS must not contain * in production")
		}
		if len(c.FirstSuperuserPassword) < 12 {
			issues = append(issues, "FIRST_SUPERUSER_PASSWORD must be at least 12 characters in production")
		}
	}

	return joinValidationIssues(issues)
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.Env), "production")
}

func (c Config) EnsureRuntimePaths() error {
	dirs := []string{
		strings.TrimSpace(c.StaticDir),
		strings.TrimSpace(c.UploadDir),
		strings.TrimSpace(c.BackupDir),
		strings.TrimSpace(c.UpdateDir),
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create runtime dir %s: %w", dir, err)
		}
	}
	logDir := strings.TrimSpace(filepath.Dir(c.LogFile))
	if logDir != "" && logDir != "." {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return fmt.Errorf("create log dir %s: %w", logDir, err)
		}
	}
	return nil
}

func joinValidationIssues(issues []string) error {
	if len(issues) == 0 {
		return nil
	}
	return errors.New(strings.Join(issues, "; "))
}

func containsStringFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

// loadEnvFiles loads key=value pairs from a config file into the process
// environment. The file path comes from CONFIG_FILE (default "data/.env"). Existing
// environment variables always win, so the file only provides fallbacks. A
// missing file is ignored silently.
func loadEnvFiles() {
	applyEnvFile(configFilePath())
}

// ensureGeneratedSecrets creates durable production credentials on first boot.
// Explicit environment variables and values loaded from CONFIG_FILE always win.
func ensureGeneratedSecrets() {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return
	}
	path := configFilePath()
	generated := make(map[string]string, 2)
	for _, key := range []string{"JWT_SECRET", "FIRST_SUPERUSER_PASSWORD"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		value, err := randomSecret()
		if err != nil {
			continue
		}
		_ = os.Setenv(key, value)
		generated[key] = value
	}
	if len(generated) == 0 {
		return
	}
	if err := persistGeneratedValues(path, generated); err != nil {
		// Forget values that were not made durable: a restart must not rotate them
		// unexpectedly, and startup validation will report any missing secret.
		for key := range generated {
			_ = os.Unsetenv(key)
		}
	}
}

func persistGeneratedValues(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	for key, value := range values {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return file.Sync()
}

func applyEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	externalEnv := make(map[string]struct{})
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.TrimSpace(value) != "" {
			externalEnv[key] = struct{}{}
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := externalEnv[key]; exists {
			continue
		}
		_ = os.Setenv(key, unquoteEnv(strings.TrimSpace(value)))
	}
}

func unquoteEnv(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func envLogLevel(key string, fallback slog.Level) slog.Level {
	switch strings.ToLower(env(key, "")) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}
