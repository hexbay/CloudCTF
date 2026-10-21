package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testVulnLabService(t *testing.T) *VulnLabService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewVulnLabService(db)
}

func TestVulnLabServicePlayerDetailIncludesRuntime(t *testing.T) {
	service := testVulnLabService(t)
	user := models.User{Username: "lab-player", Email: "lab-player@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	lab := models.VulnLab{Name: "lab", TemplateID: 1, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}
	expiresAt := time.Now().Add(time.Hour)
	if err := service.db.Create(&models.ContainerGroup{
		Name:         "runtime",
		Status:       "running",
		TemplateName: "template",
		UserID:       &user.ID,
		VulnLabID:    &lab.ID,
		Containers: models.ContainerRuntimeList{
			{Targets: []models.ContainerTarget{{URL: "http://127.0.0.1:8080"}}},
		},
		ExpiresAt: &expiresAt,
	}).Error; err != nil {
		t.Fatalf("create container group: %v", err)
	}

	row, err := service.PlayerDetail(lab.ID, user.ID)
	if err != nil {
		t.Fatalf("player detail: %v", err)
	}
	runtime, ok := row["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime, got %#v", row["runtime"])
	}
	if runtime["status"] != "running" || runtime["target"] != "template" {
		t.Fatalf("unexpected runtime: %#v", runtime)
	}
}

func TestVulnLabServicePlayerDetailUnavailableReturnsForbiddenMessage(t *testing.T) {
	service := testVulnLabService(t)
	lab := models.VulnLab{Name: "hidden lab", TemplateID: 1, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}
	if err := service.db.Model(&lab).Updates(map[string]any{"is_public": false}).Error; err != nil {
		t.Fatalf("hide lab: %v", err)
	}

	_, err := service.PlayerDetail(lab.ID, 0)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "lab is not available") {
		t.Fatalf("expected unavailable message, got %v", err)
	}
}
