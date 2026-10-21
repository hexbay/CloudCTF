package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type CompetitionService struct {
	db *gorm.DB
}

func NewCompetitionService(db *gorm.DB) *CompetitionService {
	return &CompetitionService{db: db}
}

func (s *CompetitionService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var competitions []models.Competition
	db := s.db.Model(&models.Competition{})
	if name := compFirstQuery(query, "name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if typ := compFirstQuery(query, "type"); typ != "" {
		db = db.Where("type = ?", typ)
	}
	if participantMode := compFirstQuery(query, "participant_mode"); participantMode != "" {
		db = db.Where("participant_mode = ?", participantMode)
	}
	if status := compFirstQuery(query, "status"); status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if current <= 0 {
		current = 1
	}
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&competitions).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(competitions))
	for _, competition := range competitions {
		rows = append(rows, competitionToRow(competition))
	}
	return rows, int(total), nil
}

func (s *CompetitionService) GetByID(id any) (*models.Competition, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var competition models.Competition
	if err := s.db.First(&competition, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &competition, nil
}

func (s *CompetitionService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	competition := models.Competition{
		Name:             compCleanString(row["name"]),
		Description:      compCleanString(row["description"]),
		Type:             compValueOrDefault(row["type"], "ctf"),
		ParticipantMode:  compValueOrDefault(row["participant_mode"], "individual"),
		Status:           compValueOrDefault(row["status"], "draft"),
		StartTime:        compParseOptionalTime(row["start_time"]),
		EndTime:          compParseOptionalTime(row["end_time"]),
		IsPublic:         compBoolValue(row["is_public"], true),
		RegistrationOpen: compBoolValue(row["registration_open"], false),
		Config:           "{}",
	}
	if competition.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := s.db.Create(&competition).Error; err != nil {
		return nil, err
	}
	return competitionToRow(competition), nil
}

func (s *CompetitionService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var competition models.Competition
	if err := s.db.First(&competition, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		competition.Name = compCleanString(value)
	}
	if value, ok := patch["description"]; ok {
		competition.Description = compCleanString(value)
	}
	if value, ok := patch["type"]; ok {
		competition.Type = compCleanString(value)
	}
	if value, ok := patch["participant_mode"]; ok {
		competition.ParticipantMode = compCleanString(value)
	}
	if value, ok := patch["status"]; ok {
		competition.Status = compCleanString(value)
	}
	if value, ok := patch["start_time"]; ok {
		competition.StartTime = compParseOptionalTime(value)
	}
	if value, ok := patch["end_time"]; ok {
		competition.EndTime = compParseOptionalTime(value)
	}
	if value, ok := patch["is_public"]; ok {
		competition.IsPublic = compBoolValue(value, competition.IsPublic)
	}
	if value, ok := patch["registration_open"]; ok {
		competition.RegistrationOpen = compBoolValue(value, competition.RegistrationOpen)
	}
	if err := s.db.Save(&competition).Error; err != nil {
		return nil, false
	}
	return competitionToRow(competition), true
}

func (s *CompetitionService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Competition{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *CompetitionService) ToRow(item models.Competition) map[string]any {
	return competitionToRow(item)
}

func (s *CompetitionService) Recent(limit int) []map[string]any {
	var competitions []models.Competition
	_ = s.db.Order("id DESC").Limit(limit).Find(&competitions).Error
	rows := make([]map[string]any, 0, len(competitions))
	for _, competition := range competitions {
		rows = append(rows, competitionToRow(competition))
	}
	return rows
}

func (s *CompetitionService) DashboardCounts(now time.Time) (int64, int64, int64, int64) {
	var total, ongoing, upcoming, finished int64
	_ = s.db.Model(&models.Competition{}).Count(&total).Error
	_ = s.db.Model(&models.Competition{}).Where("status = ? OR (start_time <= ? AND end_time >= ?)", "ongoing", now, now).Count(&ongoing).Error
	_ = s.db.Model(&models.Competition{}).Where("status = ?", "upcoming").Count(&upcoming).Error
	_ = s.db.Model(&models.Competition{}).Where("status = ? OR end_time < ?", "finished", now).Count(&finished).Error
	return total, ongoing, upcoming, finished
}

func (s *CompetitionService) SetScoreboardFrozen(id any, frozen bool) bool {
	return s.db.Model(&models.Competition{}).Where("id = ?", fmt.Sprint(id)).Update("scoreboard_frozen", frozen).Error == nil
}

func (s *CompetitionService) CountAnnouncements(competitionID uint) int64 {
	var total int64
	_ = s.db.Model(&models.Announcement{}).
		Where("target_type = ? AND target_id = ? AND is_published = ?", "competition", competitionID, true).
		Count(&total).Error
	return total
}

func competitionToRow(competition models.Competition) map[string]any {
	return map[string]any{
		"id":                competition.ID,
		"name":              competition.Name,
		"description":       competition.Description,
		"type":              competition.Type,
		"participant_mode":  competition.ParticipantMode,
		"status":            competition.Status,
		"start_time":        competition.StartTime,
		"end_time":          competition.EndTime,
		"is_public":         competition.IsPublic,
		"registration_open": competition.RegistrationOpen,
		"scoreboard_frozen": competition.ScoreboardFrozen,
		"banner_url":        competition.BannerURL,
		"created_at":        competition.CreatedAt,
		"updated_at":        competition.UpdatedAt,
	}
}

func compFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func compCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func compValueOrDefault(value any, fallback string) string {
	text := compCleanString(value)
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func compBoolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}

func compParseOptionalTime(value any) *time.Time {
	text := compCleanString(value)
	if text == "" || text == "<nil>" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return &parsed
		}
	}
	return nil
}
