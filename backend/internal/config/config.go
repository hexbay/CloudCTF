package config

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
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
	UpdateRepository       string
	UpdateAPIURL           string
	UpdateGitHubToken      string
}

func Load() Config {
	loadEnvFiles()
	return Config{
		AppName:                env("APP_NAME", "CloudCTF API"),
		Host:                   env("HOST", "0.0.0.0"),
		Port:                   envInt("PORT", 8005),
		Env:                    env("APP_ENV", "development"),
		LogLevel:               envLogLevel("LOG_LEVEL", slog.LevelInfo),
		LogFile:                env("LOG_FILE", "logs/backend.log"),
		DatabaseURI:            env("DATABASE_URI", "sqlite://data/cloudctf.db"),
		RedisAddr:              env("REDIS_ADDR", "localhost:6379"),
		JWTSecret:              env("JWT_SECRET", "change-me"),
		FirstSuperuserUsername: env("FIRST_SUPERUSER_USERNAME", "admin"),
		FirstSuperuserPassword: env("FIRST_SUPERUSER_PASSWORD", ""),
		UploadDir:              env("UPLOAD_DIR", "static/uploads"),
		StaticDir:              env("STATIC_DIR", "static"),
		FrontendDir:            env("FRONTEND_DIR", "../frontend/dist"),
		BackupDir:              env("BACKUP_DIR", "data/backups"),
		BackupRetention:        envInt("BACKUP_RETENTION", 7),
		AllowedOrigins:         envList("ALLOWED_ORIGINS", []string{"*"}),
		ContainerLabelKey:      env("CONTAINER_LABEL_KEY", "com.cloudctf.project"),
		ContainerLabelValue:    env("CONTAINER_LABEL_VALUE", "cloudctf"),
		ContainerManagedKey:    env("CONTAINER_MANAGED_LABEL_KEY", "com.cloudctf.managed"),
		ContainerManagedVal:    env("CONTAINER_MANAGED_LABEL_VALUE", "true"),
		UpdateEnabled:          envBool("UPDATE_ENABLED", true),
		UpdateRepository:       env("UPDATE_REPOSITORY", "hexbay/CloudCTF"),
		UpdateAPIURL:           env("UPDATE_API_URL", "https://api.github.com"),
		UpdateGitHubToken:      env("UPDATE_GITHUB_TOKEN", ""),
	}
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

	if c.IsProduction() {
		if strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == "change-me" {
			issues = append(issues, "JWT_SECRET must be set to a non-default value in production")
		}
		if len(c.AllowedOrigins) == 0 || containsStringFold(c.AllowedOrigins, "*") {
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
// environment. The file path comes from CONFIG_FILE (default ".env"). Existing
// environment variables always win, so the file only provides fallbacks. A
// missing file is ignored silently.
func loadEnvFiles() {
	path := strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	if path == "" {
		path = ".env"
	}
	applyEnvFile(path)
}

func applyEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

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
		if _, exists := os.LookupEnv(key); exists {
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
