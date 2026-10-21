package services

import (
	"errors"
	"strings"
	"testing"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testCompetitionRelationService(t *testing.T) *CompetitionRelationService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewCompetitionRelationService(db)
}

func TestCompetitionRelationServiceEnsureCanAccessCompetitionOutsiderForbiddenMessage(t *testing.T) {
	svc := testCompetitionRelationService(t)

	user := models.User{Username: "u1", Email: "u1@test.com", HashedPassword: "x", IsActive: true}
	if err := svc.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	comp := models.Competition{Name: "c1", Type: "ctf", ParticipantMode: "individual", Status: "ongoing", IsPublic: true}
	if err := svc.db.Create(&comp).Error; err != nil {
		t.Fatalf("create comp: %v", err)
	}

	err := svc.EnsureCanAccessCompetition(user.ID, comp.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "challenge is not available") {
		t.Fatalf("expected unavailable message, got %v", err)
	}
}

func TestCompetitionRelationServiceEnsureCanAccessExamOutsiderForbiddenMessage(t *testing.T) {
	svc := testCompetitionRelationService(t)

	user := models.User{Username: "u1", Email: "u1@test.com", HashedPassword: "x", IsActive: true}
	if err := svc.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	comp := models.Competition{Name: "c1", Type: "theory", ParticipantMode: "individual", Status: "ongoing", IsPublic: true}
	if err := svc.db.Create(&comp).Error; err != nil {
		t.Fatalf("create comp: %v", err)
	}

	err := svc.EnsureCanAccessExam(user.ID, comp.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "exam is not available") {
		t.Fatalf("expected exam unavailable message, got %v", err)
	}
}

func TestCompetitionRelationServiceEnsureCanSubmitExamDraftReturnsInvalidInputMessage(t *testing.T) {
	svc := testCompetitionRelationService(t)

	user := models.User{Username: "u1", Email: "u1@test.com", HashedPassword: "x", IsActive: true}
	if err := svc.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	comp := models.Competition{Name: "c1", Type: "theory", ParticipantMode: "individual", Status: "draft", IsPublic: true}
	if err := svc.db.Create(&comp).Error; err != nil {
		t.Fatalf("create comp: %v", err)
	}
	if err := svc.db.Create(&models.CompetitionParticipant{CompetitionID: comp.ID, UserID: user.ID, Username: user.Username}).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}

	err := svc.EnsureCanSubmitExam(user.ID, comp.ID)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "exam is not available") {
		t.Fatalf("expected exam unavailable message, got %v", err)
	}
}

func TestCompetitionRelationServiceEnsureCanSubmitCompetitionDraftReturnsInvalidInputMessage(t *testing.T) {
	svc := testCompetitionRelationService(t)

	user := models.User{Username: "u1", Email: "u1@test.com", HashedPassword: "x", IsActive: true}
	if err := svc.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	comp := models.Competition{Name: "c1", Type: "ctf", ParticipantMode: "individual", Status: "draft", IsPublic: true}
	if err := svc.db.Create(&comp).Error; err != nil {
		t.Fatalf("create comp: %v", err)
	}
	if err := svc.db.Create(&models.CompetitionParticipant{CompetitionID: comp.ID, UserID: user.ID, Username: user.Username}).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}

	err := svc.EnsureCanSubmitCompetition(user.ID, comp.ID)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "challenge is not available") {
		t.Fatalf("expected unavailable message, got %v", err)
	}
}
