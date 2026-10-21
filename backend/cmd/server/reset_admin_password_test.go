package main

import (
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"

	"gorm.io/gorm"
)

func TestResetAdminPassword(t *testing.T) {
	db := resetPasswordTestDB(t)
	role := models.Role{Name: "superadmin", Permissions: `["*"]`}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{
		Username: "admin", Email: "admin@example.com", HashedPassword: "old",
		RoleID: &role.ID, IsSuperuser: true, IsActive: false,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	token := models.RefreshToken{UserID: user.ID, TokenHash: "token", ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&token).Error; err != nil {
		t.Fatal(err)
	}

	if err := resetAdminPassword(db, "admin", "new-password-123", true); err != nil {
		t.Fatalf("resetAdminPassword() error = %v", err)
	}
	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !services.VerifyPassword("new-password-123", updated.HashedPassword) {
		t.Fatal("new password was not stored")
	}
	if !updated.IsActive || !updated.MustChangePassword {
		t.Fatalf("unexpected account state: active=%v must_change=%v", updated.IsActive, updated.MustChangePassword)
	}
	var updatedToken models.RefreshToken
	if err := db.First(&updatedToken, token.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedToken.RevokedAt == nil {
		t.Fatal("refresh token was not revoked")
	}
}

func TestResetAdminPasswordRejectsPlayer(t *testing.T) {
	db := resetPasswordTestDB(t)
	role := models.Role{Name: "player", Permissions: `[]`}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "player", Email: "player@example.com", HashedPassword: "old", RoleID: &role.ID, IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := resetAdminPassword(db, "player", "new-password-123", true); err == nil {
		t.Fatal("resetAdminPassword() accepted a player account")
	}
}

func resetPasswordTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Role{}, &models.User{}, &models.RefreshToken{}); err != nil {
		t.Fatal(err)
	}
	return db
}
