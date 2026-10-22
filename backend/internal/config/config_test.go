package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGeneratesAndPersistsProductionSecrets(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "data", ".env")
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

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	for _, expected := range []string{"JWT_SECRET=" + cfg.JWTSecret, "FIRST_SUPERUSER_PASSWORD=" + cfg.FirstSuperuserPassword} {
		if !strings.Contains(string(contents), expected) {
			t.Errorf("generated config does not contain %q", expected)
		}
	}

	// Loading again must reuse durable values instead of rotating credentials.
	_ = os.Unsetenv("JWT_SECRET")
	_ = os.Unsetenv("FIRST_SUPERUSER_PASSWORD")
	cfgAgain := Load()
	if cfgAgain.JWTSecret != cfg.JWTSecret || cfgAgain.FirstSuperuserPassword != cfg.FirstSuperuserPassword {
		t.Fatal("generated credentials changed between loads")
	}
}
