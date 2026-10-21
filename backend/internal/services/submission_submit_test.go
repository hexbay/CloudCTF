package services

import (
	"errors"
	"testing"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testSubmissionService(t *testing.T) *SubmissionService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewSubmissionService(db)
}

func TestSubmissionServiceSubmitFlagInvalidInput(t *testing.T) {
	service := testSubmissionService(t)
	if _, err := service.SubmitFlag(SubmitFlagInput{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSubmissionServiceSubmitFlagChallengeNotFound(t *testing.T) {
	service := testSubmissionService(t)
	_, err := service.SubmitFlag(SubmitFlagInput{UserID: 1, ChallengeID: 999, Flag: "flag{1}"})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSubmissionServiceSubmitFlagPracticeCorrectAndAlreadySolved(t *testing.T) {
	service := testSubmissionService(t)

	challenge := models.Challenge{
		Name:     "c1",
		Category: "misc",
		Points:   100,
		IsActive: true,
		IsPublic: true,
		Usage:    "PRACTICE",
		Flag:     "flag{ok}",
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	result, err := service.SubmitFlag(SubmitFlagInput{UserID: 10, ChallengeID: challenge.ID, Flag: "flag{ok}", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("submit flag: %v", err)
	}
	if !result.Correct || result.Points != 100 || result.AlreadySolved {
		t.Fatalf("unexpected result: %#v", result)
	}

	again, err := service.SubmitFlag(SubmitFlagInput{UserID: 10, ChallengeID: challenge.ID, Flag: "flag{ok}", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("submit again: %v", err)
	}
	if again.AlreadySolved != true {
		t.Fatalf("expected already_solved, got %#v", again)
	}
}

func TestSubmissionServiceSubmitFlagPracticeWrong(t *testing.T) {
	service := testSubmissionService(t)

	challenge := models.Challenge{
		Name:     "c1",
		Category: "misc",
		Points:   100,
		IsActive: true,
		IsPublic: true,
		Usage:    "PRACTICE",
		Flag:     "flag{ok}",
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	result, err := service.SubmitFlag(SubmitFlagInput{UserID: 11, ChallengeID: challenge.ID, Flag: "flag{no}", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("submit wrong flag: %v", err)
	}
	if result.Correct || result.Points != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestSubmissionServiceSubmitFlagDraftCompetitionReturnsInvalidInput(t *testing.T) {
	service := testSubmissionService(t)

	competition := models.Competition{
		Name:     "draft ctf",
		Type:     "ctf",
		Status:   "draft",
		IsPublic: true,
	}
	if err := service.db.Create(&competition).Error; err != nil {
		t.Fatalf("create competition: %v", err)
	}
	challenge := models.Challenge{
		Name:          "competition challenge",
		Category:      "misc",
		Points:        100,
		IsActive:      true,
		IsPublic:      true,
		Usage:         "COMPETITION",
		Flag:          "flag{ok}",
		CompetitionID: &competition.ID,
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if err := service.db.Create(&models.CompetitionParticipant{
		CompetitionID: competition.ID,
		UserID:        10,
		Username:      "player",
	}).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}

	_, err := service.SubmitFlag(SubmitFlagInput{UserID: 10, ChallengeID: challenge.ID, Flag: "flag{ok}", IPAddress: "127.0.0.1"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for draft competition, got %v", err)
	}
}
