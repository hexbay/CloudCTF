package httpserver

import (
	"log/slog"
	"testing"

	"cloudctf/backend/internal/bootstrap"
)

func TestEnsureCompetitionChallengeStartAllowed(t *testing.T) {
	cfg := testConfig()
	db, err := bootstrap.OpenDB(cfg.DatabaseURI)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("failed to prepare schema: %v", err)
	}
	srv := NewServer(Dependencies{DB: db, Config: cfg, Logger: slog.Default()})

	userService := srv.getUserService()
	teamService := srv.getTeamService()
	competitionService := srv.getCompetitionService()
	relationService := srv.getCompetitionRelationService()
	if userService == nil || teamService == nil || competitionService == nil || relationService == nil {
		t.Fatal("required services unavailable")
	}

	playerRole, err := userService.GetRoleByName(rolePlayer)
	if err != nil {
		t.Fatalf("get player role: %v", err)
	}
	userRow, err := userService.Create("users", map[string]any{
		"username": "ci-player",
		"email":    "ci-player@test.com",
		"password": "CloudFlag123!",
		"role_id":  playerRole.ID,
	})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	participantID := uintValue(userRow["id"])
	participant, err := userService.GetByID(participantID)
	if err != nil {
		t.Fatalf("load participant: %v", err)
	}

	teamRow, err := teamService.Create(map[string]any{"name": "ci-team", "owner_id": participantID})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	teamID := uintValue(teamRow["id"])

	competitionRow, err := competitionService.Create(map[string]any{
		"name":             "CTF",
		"type":             "ctf",
		"participant_mode": "team",
		"status":           "ongoing",
		"is_public":        true,
	})
	if err != nil {
		t.Fatalf("create competition: %v", err)
	}
	competitionID := uintValue(competitionRow["id"])
	draftTheoryRow, err := competitionService.Create(map[string]any{
		"name":             "Theory Draft",
		"type":             "theory",
		"participant_mode": "individual",
		"status":           "draft",
		"is_public":        true,
	})
	if err != nil {
		t.Fatalf("create draft theory competition: %v", err)
	}
	draftTheoryID := uintValue(draftTheoryRow["id"])
	if _, err := relationService.CreateParticipant(map[string]any{
		"competition_id": competitionID,
		"user_id":        participantID,
		"username":       participant.Username,
		"team_id":        teamID,
		"team_name":      "ci-team",
	}); err != nil {
		t.Fatalf("create participant: %v", err)
	}

	practiceChallenge := challengeModel{ID: 1, IsActive: true, IsPublic: true, Usage: "PRACTICE"}
	if err := srv.ensureCompetitionChallengeStartAllowed(practiceChallenge, *participant); err != nil {
		t.Fatalf("practice challenge should be allowed, got %v", err)
	}

	competitionChallenge := challengeModel{ID: 2, IsActive: true, IsPublic: true, Usage: "COMPETITION", CompetitionID: &competitionID}
	if err := srv.ensureCompetitionChallengeStartAllowed(competitionChallenge, *participant); err != nil {
		t.Fatalf("competition participant should be allowed, got %v", err)
	}
	if err := srv.checkCompetitionExamAllowed(competitionID, *participant); err != nil {
		t.Fatalf("competition exam participant should be allowed, got %v", err)
	}
	if err := srv.checkCompetitionExamAllowed(draftTheoryID, *participant); err == nil {
		t.Fatal("draft public exam should not be readable")
	}
	if err := srv.checkCompetitionExamSubmitAllowed(draftTheoryID, *participant); err == nil {
		t.Fatal("draft public exam should not allow submissions")
	}

	outsider, err := userService.GetByID(2)
	if err != nil {
		t.Fatalf("load outsider: %v", err)
	}
	if err := srv.ensureCompetitionChallengeStartAllowed(competitionChallenge, *outsider); err == nil {
		t.Fatal("competition outsider should be rejected")
	}
	if err := srv.checkCompetitionExamAllowed(competitionID, *outsider); err == nil {
		t.Fatal("competition exam outsider should be rejected")
	}
}
