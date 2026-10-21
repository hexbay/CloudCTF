package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"
)

type SubmitFlagInput struct {
	UserID        uint
	ChallengeID   uint
	CompetitionID *uint
	Flag          string
	IPAddress     string
}

type SubmitFlagResult struct {
	Correct       bool   `json:"correct"`
	Points        int    `json:"points"`
	AlreadySolved bool   `json:"already_solved"`
	FirstBlood    bool   `json:"is_first_blood"`
	Message       string `json:"message"`
}

func (s *SubmissionService) SubmitFlag(input SubmitFlagInput) (SubmitFlagResult, error) {
	if s == nil || s.db == nil {
		return SubmitFlagResult{}, fmt.Errorf("db unavailable")
	}
	flag := strings.TrimSpace(input.Flag)
	if input.UserID == 0 || input.ChallengeID == 0 || flag == "" {
		return SubmitFlagResult{}, fmt.Errorf("%w: user_id, challenge_id, and flag are required", ErrInvalidInput)
	}

	var challenge models.Challenge
	if err := s.db.First(&challenge, input.ChallengeID).Error; err != nil {
		return SubmitFlagResult{}, ErrNotFound
	}
	if !challenge.IsActive {
		return SubmitFlagResult{}, fmt.Errorf("%w: challenge is not active", ErrInvalidInput)
	}
	if !playerAccessibleChallenge(challenge) {
		return SubmitFlagResult{}, ErrForbidden
	}

	if challenge.CompetitionID != nil {
		if err := s.ensureCanSubmitCompetitionChallenge(input.UserID, *challenge.CompetitionID); err != nil {
			return SubmitFlagResult{}, err
		}
	}

	if _, found := s.ExistsSolvedByUserWithCompetition(challenge.ID, input.UserID, challenge.CompetitionID); found {
		return SubmitFlagResult{Correct: false, AlreadySolved: true, Message: "已经解决过了"}, nil
	}

	isCorrect := false
	points := 0
	if challenge.IsDynamic {
		group, found := (&ContainerGroupService{db: s.db}).GetActiveChallengeContainerGroup(challenge.ID, input.UserID)
		if !found {
			return SubmitFlagResult{}, fmt.Errorf("%w: dynamic challenge requires an active container", ErrInvalidInput)
		}
		expected := strings.TrimSpace(fmt.Sprint(group["flag"]))
		isCorrect = verifyDynamicFlag(input.UserID, challenge.ID, expected, flag)
	} else {
		isCorrect = flag == challenge.Flag
	}
	if isCorrect {
		points = challenge.Points
	}

	isFirstBlood := false
	if isCorrect {
		isFirstBlood = !s.FirstBloodExists(challenge.ID, challenge.CompetitionID)
	}

	var competitionID any
	var teamID any
	if challenge.CompetitionID != nil {
		competitionID = *challenge.CompetitionID
		var user models.User
		if err := s.db.Select("team_id").First(&user, input.UserID).Error; err == nil && user.TeamID != nil && *user.TeamID != 0 {
			teamID = user.TeamID
		}
	}
	if _, err := s.Create(map[string]any{
		"challenge_id":   challenge.ID,
		"user_id":        input.UserID,
		"team_id":        teamID,
		"competition_id": competitionID,
		"flag":           flag,
		"is_correct":     isCorrect,
		"points":         points,
		"ip_address":     input.IPAddress,
	}); err != nil {
		return SubmitFlagResult{}, err
	}

	message := "FLAG 错误"
	if isCorrect {
		message = "正确!"
		if isFirstBlood {
			message = "一血! 正确!"
		}
	}
	return SubmitFlagResult{
		Correct:       isCorrect,
		Points:        points,
		AlreadySolved: false,
		FirstBlood:    isFirstBlood,
		Message:       message,
	}, nil
}

func (s *SubmissionService) ensureCanSubmitCompetitionChallenge(userID, competitionID uint) error {
	var comp models.Competition
	if err := s.db.First(&comp, competitionID).Error; err != nil {
		return ErrForbidden
	}
	if !comp.IsPublic {
		return ErrForbidden
	}
	if strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("%w: challenge is not available", ErrInvalidInput)
	}
	if strings.TrimSpace(comp.ParticipantMode) == "team" {
		var user models.User
		if err := s.db.Select("team_id").First(&user, userID).Error; err != nil {
			return ErrForbidden
		}
		if user.TeamID == nil || *user.TeamID == 0 {
			return ErrForbidden
		}
		var count int64
		_ = s.db.Model(&models.CompetitionParticipant{}).
			Where("competition_id = ? AND team_id = ?", competitionID, *user.TeamID).
			Count(&count).Error
		if count <= 0 {
			return ErrForbidden
		}
		return nil
	}
	var count int64
	_ = s.db.Model(&models.CompetitionParticipant{}).
		Where("competition_id = ? AND user_id = ?", competitionID, userID).
		Count(&count).Error
	if count <= 0 {
		return ErrForbidden
	}
	return nil
}

func playerAccessibleChallenge(ch models.Challenge) bool {
	if !ch.IsActive || !ch.IsPublic {
		return false
	}
	if ch.CompetitionID != nil {
		return ch.Usage == "" || ch.Usage == "COMPETITION" || ch.Usage == "BOTH"
	}
	return ch.Usage == "" || ch.Usage == "PRACTICE" || ch.Usage == "BOTH"
}

func verifyDynamicFlag(userID, challengeID uint, expectedFlag, submittedFlag string) bool {
	now := time.Now()
	today := generateDynamicFlag(userID, challengeID, now)
	yesterday := generateDynamicFlag(userID, challengeID, now.Add(-24*time.Hour))
	return submittedFlag == expectedFlag || submittedFlag == today || submittedFlag == yesterday
}
