package services

import (
	"fmt"
	"strings"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type WriteupService struct {
	db *gorm.DB
}

func NewWriteupService(db *gorm.DB) *WriteupService {
	return &WriteupService{db: db}
}

func (s *WriteupService) DB() *gorm.DB { return s.db }

func (s *WriteupService) GetByID(id any) (*models.Writeup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.Writeup
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *WriteupService) GetByCompetitionAndUser(competitionID, userID uint) (*models.Writeup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.Writeup
	if err := s.db.Where("competition_id = ? AND user_id = ?", competitionID, userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *WriteupService) Upsert(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	competitionID := writeupUintValue(row["competition_id"])
	userID := writeupUintValue(row["user_id"])
	if competitionID == 0 || userID == 0 {
		return nil, fmt.Errorf("competition_id and user_id are required")
	}
	var item models.Writeup
	err := s.db.Where("competition_id = ? AND user_id = ?", competitionID, userID).First(&item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		item = models.Writeup{
			CompetitionID: competitionID,
			UserID:        userID,
		}
	}
	item.TeamID = writeupOptionalUint(row["team_id"])
	item.Title = writeupString(row["title"])
	item.FilePath = writeupString(row["file_path"])
	item.FileName = writeupValueOrDefault(row["file_name"], writeupString(row["filename"]))
	item.FileSize = writeupInt64Value(row["file_size"], writeupInt64Value(row["size"], 0))
	if item.Title == "" {
		item.Title = item.FileName
	}
	if item.FilePath == "" || item.FileName == "" {
		return nil, fmt.Errorf("file_path and file_name are required")
	}
	if err == gorm.ErrRecordNotFound {
		if err := s.db.Create(&item).Error; err != nil {
			return nil, err
		}
		return writeupToRow(item), nil
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	return writeupToRow(item), nil
}

func (s *WriteupService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Writeup{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *WriteupService) ToRow(item models.Writeup) map[string]any {
	return writeupToRow(item)
}

func writeupToRow(item models.Writeup) map[string]any {
	row := map[string]any{
		"id":             item.ID,
		"competition_id": item.CompetitionID,
		"user_id":        item.UserID,
		"team_id":        item.TeamID,
		"title":          item.Title,
		"file_path":      item.FilePath,
		"file_name":      item.FileName,
		"filename":       item.FileName,
		"file_size":      item.FileSize,
		"size":           item.FileSize,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
	if path := strings.TrimSpace(item.FilePath); path != "" {
		row["local_path"] = path
		row["url"] = "/" + strings.TrimLeft(strings.ReplaceAll(path, "\\", "/"), "/")
	}
	return row
}

func writeupString(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func writeupValueOrDefault(value any, fallback string) string {
	text := writeupString(value)
	if text == "" {
		return fallback
	}
	return text
}

func writeupUintValue(value any) uint {
	switch typed := value.(type) {
	case uint:
		return typed
	case uint64:
		return uint(typed)
	case int:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case int64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case float64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case string:
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(typed), &parsed)
		return parsed
	default:
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(fmt.Sprint(value)), &parsed)
		return parsed
	}
}

func writeupOptionalUint(value any) *uint {
	parsed := writeupUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func writeupInt64Value(value any, fallback int64) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		var parsed int64
		if _, err := fmt.Sscan(strings.TrimSpace(typed), &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}
