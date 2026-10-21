package services

import (
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
)

type testTokenIssuer struct{}

func (testTokenIssuer) IssueAuthTokens(userID uint) (string, string, time.Time, error) {
	return "access", "refresh", time.Unix(100, 0), nil
}

func testAuthService(t *testing.T) *AuthService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewAuthService(NewUserService(db), testTokenIssuer{})
}

func TestAuthServiceLoginAdmin(t *testing.T) {
	service := testAuthService(t)
	result, err := service.LoginAdmin("admin", "admin123456")
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	if result.AccessToken != "access" || result.RefreshToken != "refresh" {
		t.Fatalf("unexpected tokens: %#v", result)
	}
	if result.User.Role.Name != "superadmin" {
		t.Fatalf("expected superadmin role, got %#v", result.User.Role)
	}
}

func TestAuthServiceRejectsPlayerForAdminLogin(t *testing.T) {
	service := testAuthService(t)
	if _, err := service.LoginAdmin("player1", "password123"); err != ErrAuthForbidden {
		t.Fatalf("expected ErrAuthForbidden, got %v", err)
	}
}
