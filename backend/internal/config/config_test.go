package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGeneratesAndPersistsProductionSecrets(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "data", ".env")
	template := "APP_NAME=Template CloudCTF\nAPP_ENV=development\nJWT_SECRET=change-me\nFIRST_SUPERUSER_USERNAME=admin\nFIRST_SUPERUSER_PASSWORD=change-me\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".env.example"), []byte(template), 0o600); err != nil {
		t.Fatalf("write env template: %v", err)
	}
	t.Chdir(tempDir)
	t.Setenv("CONFIG_FILE", configPath)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("FIRST_SUPERUSER_PASSWORD", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg := Load()
	if len(cfg.JWTSecret) < 32 {
		t.Fatalf("expected generated JWT secret, got length %d", len(cfg.JWTSecret))
	}
	if len(cfg.FirstSuperuserPassword) < 12 {
		t.Fatalf("expected generated admin password, got length %d", len(cfg.FirstSuperuserPassword))
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated production config should validate: %v", err)
	}
	if cfg.AppName != "Template CloudCTF" {
		t.Fatalf("expected template content to be copied, got app name %q", cfg.AppName)
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	for _, expected := range []string{"JWT_SECRET=" + cfg.JWTSecret, "FIRST_SUPERUSER_PASSWORD=" + cfg.FirstSuperuserPassword} {
		if !strings.Contains(string(contents), expected) {
			t.Errorf("generated config does not contain %q", expected)
		}
	}
	if strings.Contains(string(contents), "change-me") {
		t.Fatal("generated config retained a placeholder secret")
	}
	if !strings.Contains(string(contents), "APP_ENV=production") {
		t.Fatal("generated config did not persist the effective APP_ENV")
	}

	// Loading again must reuse durable values instead of rotating credentials.
	_ = os.Unsetenv("JWT_SECRET")
	_ = os.Unsetenv("FIRST_SUPERUSER_PASSWORD")
	cfgAgain := Load()
	if cfgAgain.JWTSecret != cfg.JWTSecret || cfgAgain.FirstSuperuserPassword != cfg.FirstSuperuserPassword {
		t.Fatal("generated credentials changed between loads")
	}
}

func TestEnsureEnvFileDoesNotOverwriteExistingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".env")
	original := []byte("JWT_SECRET=keep-existing-secret\nFIRST_SUPERUSER_PASSWORD=keep-existing-password\n")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	t.Setenv("CONFIG_FILE", configPath)

	ensureEnvFile()

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(contents) != string(original) {
		t.Fatal("existing config was unexpectedly modified")
	}
}

func TestLoadProxySupportsHTTPAndSOCKS5(t *testing.T) {
	for _, proxy := range []string{
		"http://127.0.0.1:10808",
		"socks5://127.0.0.1:10808",
	} {
		t.Run(proxy, func(t *testing.T) {
			t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), ".env"))
			t.Setenv("APP_ENV", "development")
			t.Setenv("PROXY", proxy)

			cfg := Load()
			if cfg.Proxy != proxy {
				t.Fatalf("Proxy = %q, want %q", cfg.Proxy, proxy)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("proxy configuration should validate: %v", err)
			}
		})
	}
}

func TestValidateRejectsUnsupportedProxyScheme(t *testing.T) {
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), ".env"))
	t.Setenv("APP_ENV", "development")
	t.Setenv("PROXY", "ftp://127.0.0.1:10808")

	cfg := Load()
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "PROXY scheme") {
		t.Fatalf("Validate() error = %v, want unsupported proxy scheme", err)
	}
}
