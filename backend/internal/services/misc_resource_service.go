package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type MiscResourceService struct {
	db *gorm.DB
}

func NewMiscResourceService(db *gorm.DB) *MiscResourceService {
	return &MiscResourceService{db: db}
}

func (s *MiscResourceService) CreateTeamJoinRequest(teamID, userID uint, status string) (map[string]any, error) {
	item := models.TeamJoinRequest{
		TeamID: teamID,
		UserID: userID,
		Status: miscDefault(status, "approved"),
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return teamJoinRequestToRow(item), nil
}

func (s *MiscResourceService) CreateBuildLog(row map[string]any) (map[string]any, error) {
	item := models.BuildLog{
		TemplateID:   miscOptionalUint(row["template_id"]),
		TemplateName: miscClean(row["template_name"]),
		TaskID:       miscClean(row["task_id"]),
		Status:       miscDefault(miscClean(row["status"]), "building"),
		Message:      miscClean(row["message"]),
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return buildLogToRow(item), nil
}

func (s *MiscResourceService) ListBuildLogs(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	db := s.db.Model(&models.BuildLog{})
	if templateID := miscFirstQuery(query, "template_id"); templateID != "" {
		db = db.Where("template_id = ?", templateID)
	}
	if keyword := miscFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("template_name LIKE ? OR message LIKE ? OR status LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
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
	var items []models.BuildLog
	if err := db.Order("id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, buildLogToRow(item))
	}
	return rows, int(total), nil
}

func (s *MiscResourceService) ListBuildLogsByTemplate(templateID uint) ([]map[string]any, error) {
	var items []models.BuildLog
	if err := s.db.Where("template_id = ?", templateID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, buildLogToRow(item))
	}
	return rows, nil
}

func (s *MiscResourceService) CreateTemplateSyncLog(row map[string]any) (map[string]any, error) {
	item := models.TemplateSyncLog{
		TemplateID: miscOptionalUint(row["template_id"]),
		Status:     miscDefault(miscClean(row["status"]), "completed"),
		Message:    miscClean(row["message"]),
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return templateSyncLogToRow(item), nil
}

func (s *MiscResourceService) ListTemplateSyncLogs() ([]map[string]any, error) {
	var items []models.TemplateSyncLog
	if err := s.db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, templateSyncLogToRow(item))
	}
	return rows, nil
}

func teamJoinRequestToRow(item models.TeamJoinRequest) map[string]any {
	return map[string]any{"id": item.ID, "team_id": item.TeamID, "user_id": item.UserID, "status": item.Status, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func buildLogToRow(item models.BuildLog) map[string]any {
	return map[string]any{"id": item.ID, "template_id": item.TemplateID, "template_name": item.TemplateName, "task_id": item.TaskID, "status": item.Status, "message": item.Message, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func templateSyncLogToRow(item models.TemplateSyncLog) map[string]any {
	return map[string]any{"id": item.ID, "template_id": item.TemplateID, "status": item.Status, "message": item.Message, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func miscClean(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func miscDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func miscOptionalUint(value any) *uint {
	parsed := relationUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func miscFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

var _ = time.RFC3339
