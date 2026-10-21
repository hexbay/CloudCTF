package services

import (
	"errors"
	"fmt"
	"strings"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type CompetitionRelationService struct {
	db *gorm.DB
}

type RegisterCompetitionInput struct {
	UserID        uint
	CompetitionID uint
	TeamID        *uint
}

func NewCompetitionRelationService(db *gorm.DB) *CompetitionRelationService {
	return &CompetitionRelationService{db: db}
}

func (s *CompetitionRelationService) EnsureCanAccessCompetition(userID, competitionID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if userID == 0 || competitionID == 0 {
		return fmt.Errorf("%w: competition id required", ErrInvalidInput)
	}
	var comp models.Competition
	if err := s.db.First(&comp, competitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: challenge is not available", ErrForbidden)
		}
		return err
	}
	if !comp.IsPublic || strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}
	if !s.ParticipantExists(competitionID, userID) {
		return fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}
	return nil
}

func (s *CompetitionRelationService) EnsureCanSubmitCompetition(userID, competitionID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if userID == 0 || competitionID == 0 {
		return fmt.Errorf("%w: competition id required", ErrInvalidInput)
	}
	var comp models.Competition
	if err := s.db.First(&comp, competitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: challenge is not available", ErrForbidden)
		}
		return err
	}
	if !comp.IsPublic {
		return fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}
	if strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("%w: challenge is not available", ErrInvalidInput)
	}
	if !s.ParticipantExists(competitionID, userID) {
		return fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}
	return nil
}

func (s *CompetitionRelationService) EnsureCanAccessExam(userID, competitionID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if userID == 0 || competitionID == 0 {
		return fmt.Errorf("%w: competition id required", ErrInvalidInput)
	}
	var comp models.Competition
	if err := s.db.First(&comp, competitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: exam is not available", ErrForbidden)
		}
		return err
	}
	if !comp.IsPublic || strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("%w: exam is not available", ErrForbidden)
	}
	if !s.ParticipantExists(competitionID, userID) {
		return fmt.Errorf("%w: exam is not available", ErrForbidden)
	}
	return nil
}

func (s *CompetitionRelationService) EnsureCanSubmitExam(userID, competitionID uint) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if userID == 0 || competitionID == 0 {
		return fmt.Errorf("%w: competition id required", ErrInvalidInput)
	}
	var comp models.Competition
	if err := s.db.First(&comp, competitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: exam is not available", ErrForbidden)
		}
		return err
	}
	if !comp.IsPublic {
		return fmt.Errorf("%w: exam is not available", ErrForbidden)
	}
	if strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("%w: exam is not available", ErrInvalidInput)
	}
	if !s.ParticipantExists(competitionID, userID) {
		return fmt.Errorf("%w: exam is not available", ErrForbidden)
	}
	return nil
}

func (s *CompetitionRelationService) RegisterParticipant(input RegisterCompetitionInput) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.CompetitionID == 0 {
		return fmt.Errorf("%w: user_id and competition_id are required", ErrInvalidInput)
	}

	var user models.User
	if err := s.db.First(&user, input.UserID).Error; err != nil {
		return fmt.Errorf("%w: auth required", ErrForbidden)
	}

	var comp models.Competition
	if err := s.db.First(&comp, input.CompetitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if !comp.RegistrationOpen {
		return fmt.Errorf("%w: 报名未开放", ErrInvalidInput)
	}
	if s.ParticipantExists(input.CompetitionID, input.UserID) {
		return nil
	}

	var teamName string
	teamID := input.TeamID
	if strings.TrimSpace(comp.ParticipantMode) == "team" {
		if user.TeamID == nil || *user.TeamID == 0 {
			return fmt.Errorf("%w: 团队赛报名需要队伍", ErrInvalidInput)
		}
		if teamID != nil && *teamID != *user.TeamID {
			return fmt.Errorf("%w: team_id must match current team", ErrForbidden)
		}
		teamID = user.TeamID
		var team models.Team
		if err := s.db.First(&team, *teamID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: team not found", ErrNotFound)
			}
			return err
		}
		teamName = team.Name
	}

	_, err := s.CreateParticipant(map[string]any{
		"competition_id": input.CompetitionID,
		"user_id":        input.UserID,
		"username":       user.Username,
		"team_id":        teamID,
		"team_name":      teamName,
	})
	return err
}

func (s *CompetitionRelationService) ListParticipants(competitionID uint, current, pageSize int) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	db := s.db.Model(&models.CompetitionParticipant{}).Where("competition_id = ?", competitionID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	var items []models.CompetitionParticipant
	if err := db.Order("id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, competitionParticipantToRow(item))
	}
	return rows, int(total), nil
}

func (s *CompetitionRelationService) ParticipantExists(competitionID, userID uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	var comp models.Competition
	if err := s.db.Select("participant_mode").First(&comp, competitionID).Error; err != nil {
		return false
	}
	if strings.TrimSpace(comp.ParticipantMode) == "team" {
		var user models.User
		if err := s.db.Select("team_id").First(&user, userID).Error; err != nil {
			return false
		}
		if user.TeamID == nil || *user.TeamID == 0 {
			return false
		}
		var count int64
		_ = s.db.Model(&models.CompetitionParticipant{}).
			Where("competition_id = ? AND team_id = ?", competitionID, *user.TeamID).
			Count(&count).Error
		return count > 0
	}
	var count int64
	_ = s.db.Model(&models.CompetitionParticipant{}).Where("competition_id = ? AND user_id = ?", competitionID, userID).Count(&count).Error
	return count > 0
}

func (s *CompetitionRelationService) CreateParticipant(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.CompetitionParticipant{
		CompetitionID: relationUintValue(row["competition_id"]),
		UserID:        relationUintValue(row["user_id"]),
		Username:      relationString(row["username"]),
		TeamID:        relationOptionalUint(row["team_id"]),
		TeamName:      relationString(row["team_name"]),
	}
	if item.CompetitionID == 0 || item.UserID == 0 || item.Username == "" {
		return nil, fmt.Errorf("competition_id, user_id and username are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return competitionParticipantToRow(item), nil
}

func (s *CompetitionRelationService) DeleteParticipant(competitionID, userID uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Where("competition_id = ? AND user_id = ?", competitionID, userID).Delete(&models.CompetitionParticipant{})
	return result.Error == nil && result.RowsAffected > 0
}

func (s *CompetitionRelationService) ListAdmins(competitionID uint) ([]models.CompetitionAdmin, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var items []models.CompetitionAdmin
	if err := s.db.Where("competition_id = ?", competitionID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CompetitionRelationService) AdminExists(competitionID, userID uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	var count int64
	_ = s.db.Model(&models.CompetitionAdmin{}).Where("competition_id = ? AND user_id = ?", competitionID, userID).Count(&count).Error
	return count > 0
}

func (s *CompetitionRelationService) CreateAdmin(competitionID, userID uint) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.CompetitionAdmin{CompetitionID: competitionID, UserID: userID}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return competitionAdminToRow(item), nil
}

func (s *CompetitionRelationService) DeleteAdmin(competitionID, userID uint) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	result := s.db.Where("competition_id = ? AND user_id = ?", competitionID, userID).Delete(&models.CompetitionAdmin{})
	if result.Error != nil {
		return 0
	}
	return result.RowsAffected
}

func (s *CompetitionRelationService) CreateTeamCompetition(teamID, competitionID uint) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.TeamCompetition{TeamID: teamID, CompetitionID: competitionID}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return teamCompetitionToRow(item), nil
}

func (s *CompetitionRelationService) DeleteTeamCompetition(teamID, competitionID uint) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	result := s.db.Where("team_id = ? AND competition_id = ?", teamID, competitionID).Delete(&models.TeamCompetition{})
	if result.Error != nil {
		return 0
	}
	return result.RowsAffected
}

func competitionParticipantToRow(item models.CompetitionParticipant) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"competition_id": item.CompetitionID,
		"user_id":        item.UserID,
		"username":       item.Username,
		"team_id":        item.TeamID,
		"team_name":      item.TeamName,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func competitionAdminToRow(item models.CompetitionAdmin) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"competition_id": item.CompetitionID,
		"user_id":        item.UserID,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func teamCompetitionToRow(item models.TeamCompetition) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"team_id":        item.TeamID,
		"competition_id": item.CompetitionID,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func relationString(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func relationUintValue(value any) uint {
	switch typed := value.(type) {
	case uint:
		return typed
	case *uint:
		if typed == nil {
			return 0
		}
		return *typed
	case uint64:
		return uint(typed)
	case *uint64:
		if typed == nil {
			return 0
		}
		return uint(*typed)
	case int:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case *int:
		if typed == nil || *typed < 0 {
			return 0
		}
		return uint(*typed)
	case int64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case *int64:
		if typed == nil || *typed < 0 {
			return 0
		}
		return uint(*typed)
	case float64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case *float64:
		if typed == nil || *typed < 0 {
			return 0
		}
		return uint(*typed)
	case string:
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(typed), &parsed)
		return parsed
	case *string:
		if typed == nil {
			return 0
		}
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(*typed), &parsed)
		return parsed
	default:
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(fmt.Sprint(value)), &parsed)
		return parsed
	}
}

func relationOptionalUint(value any) *uint {
	parsed := relationUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}
