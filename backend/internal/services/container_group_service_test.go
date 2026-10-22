package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testContainerGroupService(t *testing.T) *ContainerGroupService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewContainerGroupService(db, nil)
}

func TestContainerGroupExpiresSupportsTimePointer(t *testing.T) {
	service := &ContainerGroupService{}
	expiresAt := time.Now().Add(time.Hour).Round(time.Second)

	got, ok := service.ContainerGroupExpires(map[string]any{
		"expires_at": &expiresAt,
	})
	if !ok {
		t.Fatalf("expected expires_at pointer to be recognized")
	}
	if !got.Equal(expiresAt) {
		t.Fatalf("got %v, want %v", got, expiresAt)
	}
}

func TestContainerGroupNameFallsBackToTemplateName(t *testing.T) {
	template := &models.Templates{Id: 7, Name: "Web Challenge"}
	user := &models.User{Username: "alice"}

	got := containerGroupNameFor(template, user, nil, nil)
	if got != "CTF_alice_Web Challenge" {
		t.Fatalf("got %q, want template-based group name", got)
	}
}

func TestTemplateRequiresDynamicFlag(t *testing.T) {
	ctfTemplate := &models.Templates{TemplateType: "ctf"}
	composeTemplate := &models.Templates{TemplateType: "compose"}
	challenge := &models.Challenge{Name: "Challenge"}

	if !templateRequiresDynamicFlag(ctfTemplate, nil) {
		t.Fatal("expected CTF template to require a dynamic flag")
	}
	if !templateRequiresDynamicFlag(composeTemplate, challenge) {
		t.Fatal("expected challenge-backed template to require a dynamic flag")
	}
	if templateRequiresDynamicFlag(composeTemplate, nil) {
		t.Fatal("did not expect a standalone non-CTF template to require a dynamic flag")
	}
}

func TestStartVulnLabRuntimeValidatesTemplate(t *testing.T) {
	service := testContainerGroupService(t)
	user := models.User{Username: "lab-start-user", Email: "lab-start-user@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	lab := models.VulnLab{Name: "lab", IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}

	_, err := service.StartVulnLabRuntime(StartVulnLabRuntimeInput{UserID: user.ID, VulnLabID: lab.ID})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "lab template not configured") {
		t.Fatalf("expected template configured error, got %v", err)
	}
}

func TestStartVulnLabRuntimeRejectsTemplateNotReady(t *testing.T) {
	service := testContainerGroupService(t)
	user := models.User{Username: "lab-template-user", Email: "lab-template-user@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	template := models.Templates{Name: "tpl", Status: models.TemplateStatusPending}
	if err := service.db.Create(&template).Error; err != nil {
		t.Fatalf("create template: %v", err)
	}
	lab := models.VulnLab{Name: "lab", TemplateID: template.Id, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}

	_, err := service.StartVulnLabRuntime(StartVulnLabRuntimeInput{UserID: user.ID, VulnLabID: lab.ID})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "当前环境模板未就绪") {
		t.Fatalf("expected template not ready error, got %v", err)
	}
}

func TestExtendVulnLabRuntimeExtendsActiveGroup(t *testing.T) {
	service := testContainerGroupService(t)
	user := models.User{Username: "lab-user", Email: "lab-user@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	lab := models.VulnLab{Name: "lab", TemplateID: 1, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}
	expiresAt := time.Now().Add(time.Hour).Round(time.Second)
	group := models.ContainerGroup{
		Name:      "runtime",
		Status:    "running",
		UserID:    &user.ID,
		VulnLabID: &lab.ID,
		ExpiresAt: &expiresAt,
	}
	if err := service.db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	updated, err := service.ExtendVulnLabRuntime(ExtendVulnLabRuntimeInput{UserID: user.ID, VulnLabID: lab.ID})
	if err != nil {
		t.Fatalf("extend lab runtime: %v", err)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.Equal(expiresAt.Add(30*time.Minute)) {
		t.Fatalf("unexpected expires_at: %v", updated.ExpiresAt)
	}
}

func TestStopVulnLabRuntimeDeletesActiveGroup(t *testing.T) {
	service := testContainerGroupService(t)
	user := models.User{Username: "lab-stop-user", Email: "lab-stop-user@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	lab := models.VulnLab{Name: "lab", TemplateID: 1, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}
	group := models.ContainerGroup{Name: "runtime", Status: "running", UserID: &user.ID, VulnLabID: &lab.ID}
	if err := service.db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	if err := service.StopVulnLabRuntime(StopVulnLabRuntimeInput{UserID: user.ID, VulnLabID: lab.ID}); err != nil {
		t.Fatalf("stop lab runtime: %v", err)
	}
	var count int64
	_ = service.db.Model(&models.ContainerGroup{}).Where("id = ?", group.ID).Count(&count).Error
	if count != 0 {
		t.Fatalf("expected group deleted, got count %d", count)
	}
}

func TestExtendVulnLabRuntimeMissingActiveGroupReturnsNotFoundMessage(t *testing.T) {
	service := testContainerGroupService(t)
	user := models.User{Username: "lab-missing-user", Email: "lab-missing-user@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	lab := models.VulnLab{Name: "lab", TemplateID: 1, IsPublic: true, IsActive: true}
	if err := service.db.Create(&lab).Error; err != nil {
		t.Fatalf("create lab: %v", err)
	}

	_, err := service.ExtendVulnLabRuntime(ExtendVulnLabRuntimeInput{UserID: user.ID, VulnLabID: lab.ID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "active lab environment not found") {
		t.Fatalf("expected active lab not found message, got %v", err)
	}
}
