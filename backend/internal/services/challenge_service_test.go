package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testChallengeService(t *testing.T) *ChallengeService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewChallengeService(db)
}

func TestNormalizeChallengeContainerTimeout(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "positive value kept", input: 1800, want: 1800},
		{name: "zero falls back to default", input: 0, want: 3600},
		{name: "negative falls back to default", input: -1, want: 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeChallengeContainerTimeout(tt.input); got != tt.want {
				t.Fatalf("normalizeChallengeContainerTimeout(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestChallengeServicePlayerDetailUnavailableChallengeReturnsForbiddenMessage(t *testing.T) {
	service := testChallengeService(t)
	challenge := models.Challenge{
		Name:     "hidden",
		Category: "web",
		Points:   100,
		IsActive: true,
		IsPublic: false,
		Usage:    "PRACTICE",
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if err := service.db.Model(&challenge).Update("is_public", false).Error; err != nil {
		t.Fatalf("hide challenge: %v", err)
	}

	_, err := service.PlayerDetail(PlayerChallengeDetailInput{ChallengeID: challenge.ID})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "challenge is not available") {
		t.Fatalf("expected unavailable message, got %v", err)
	}
}

func TestChallengeServicePlayerDetailIncludesSolvedHintsAndContainerInfo(t *testing.T) {
	service := testChallengeService(t)
	user := models.User{Username: "player", Email: "player@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	challenge := models.Challenge{
		Name:          "dynamic",
		Category:      "web",
		Points:        200,
		IsActive:      true,
		IsPublic:      true,
		IsDynamic:     true,
		Usage:         "PRACTICE",
		Hints:         `[{"content":"look","cost":10}]`,
		ContainerPort: 80,
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if err := service.db.Create(&models.Submission{
		ChallengeID: challenge.ID,
		UserID:      user.ID,
		Flag:        "flag{ok}",
		IsCorrect:   true,
		Points:      challenge.Points,
		SubmittedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := service.db.Create(&models.HintPurchase{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		HintIndex:   0,
		Cost:        10,
	}).Error; err != nil {
		t.Fatalf("create hint purchase: %v", err)
	}
	expiresAt := time.Now().Add(time.Hour)
	if err := service.db.Create(&models.ContainerGroup{
		Name:        "runtime",
		Status:      "running",
		ChallengeID: &challenge.ID,
		UserID:      &user.ID,
		Containers: models.ContainerRuntimeList{
			{ContainerID: "abc", Name: "web", Targets: []models.ContainerTarget{{URL: "http://127.0.0.1:8080"}}},
		},
		ExpiresAt: &expiresAt,
	}).Error; err != nil {
		t.Fatalf("create container group: %v", err)
	}

	row, err := service.PlayerDetail(PlayerChallengeDetailInput{ChallengeID: challenge.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("player detail: %v", err)
	}
	if row["solved"] != true || row["user_points"] != challenge.Points {
		t.Fatalf("unexpected solve fields: %#v", row)
	}
	hints, ok := row["hints"].([]map[string]any)
	if !ok || len(hints) != 1 || hints[0]["purchased"] != true {
		t.Fatalf("unexpected hints: %#v", row["hints"])
	}
	containerInfo, ok := row["container_info"].(map[string]any)
	if !ok || containerInfo["container_id"] != "abc" {
		t.Fatalf("unexpected container_info: %#v", row["container_info"])
	}
}

func TestChallengeServicePurchaseHintCreatesPurchaseAndReturnsHint(t *testing.T) {
	service := testChallengeService(t)
	user := models.User{Username: "hint-player", Email: "hint-player@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	challenge := models.Challenge{
		Name:     "hinted",
		Category: "web",
		Points:   100,
		IsActive: true,
		IsPublic: true,
		Usage:    "PRACTICE",
		Hints:    `[{"content":"inspect source","cost":20}]`,
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	result, err := service.PurchaseHint(PurchaseChallengeHintInput{UserID: user.ID, ChallengeID: challenge.ID, HintIndex: 0})
	if err != nil {
		t.Fatalf("purchase hint: %v", err)
	}
	if result["success"] != true || result["message"] != "提示已解锁" {
		t.Fatalf("unexpected result: %#v", result)
	}
	hint, ok := result["hint"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected hint: %#v", result["hint"])
	}
	if hint["content"] != "inspect source" || hint["cost"] != 20 || hint["purchased"] != true {
		t.Fatalf("unexpected hint: %#v", hint)
	}
	var count int64
	_ = service.db.Model(&models.HintPurchase{}).Where("user_id = ? AND challenge_id = ? AND hint_index = ?", user.ID, challenge.ID, 0).Count(&count).Error
	if count != 1 {
		t.Fatalf("expected one hint purchase, got %d", count)
	}
}

func TestChallengeServicePurchaseHintMissingHintReturnsNotFoundMessage(t *testing.T) {
	service := testChallengeService(t)
	user := models.User{Username: "missing-hint-player", Email: "missing-hint-player@example.com", HashedPassword: "x"}
	if err := service.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	challenge := models.Challenge{
		Name:     "no hints",
		Category: "web",
		Points:   100,
		IsActive: true,
		IsPublic: true,
		Usage:    "PRACTICE",
	}
	if err := service.db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	_, err := service.PurchaseHint(PurchaseChallengeHintInput{UserID: user.ID, ChallengeID: challenge.ID, HintIndex: 0})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "hint not found") {
		t.Fatalf("expected hint not found message, got %v", err)
	}
}
