package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type ResourceRepositoryService struct {
	db *gorm.DB
}

func NewResourceRepositoryService(db *gorm.DB) *ResourceRepositoryService {
	return &ResourceRepositoryService{db: db}
}

func (s *ResourceRepositoryService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.ResourceRepo
	db := s.db.Model(&models.ResourceRepo{})
	if repoType := rrFirstQuery(query, "repo_type"); repoType != "" {
		db = db.Where("repo_type = ?", repoType)
	}
	if keyword := rrFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("name LIKE ? OR repository_url LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, resourceRepositoryToRow(item))
	}
	return rows, int(total), nil
}

func (s *ResourceRepositoryService) GetByID(id any) (*models.ResourceRepo, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.ResourceRepo
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ResourceRepositoryService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.ResourceRepo{
		Name:             rrCleanString(row["name"]),
		RepoType:         rrValueOrDefault(row["repo_type"], "vul"),
		Provider:         rrValueOrDefault(row["provider"], "git"),
		RepositoryURL:    rrCleanString(row["repository_url"]),
		RepositoryBranch: rrValueOrDefault(row["repository_branch"], "main"),
		TemplatesPath:    rrCleanString(row["templates_path"]),
		HostID:           rrOptionalUint(row["host_id"]),
		Enabled:          rrBoolValue(row["enabled"], true),
		Tags:             rrJSONString(row["tags"]),
		Config:           rrJSONString(row["config"]),
	}
	if item.Name == "" || item.RepositoryURL == "" {
		return nil, fmt.Errorf("name and repository_url are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return resourceRepositoryToRow(item), nil
}

func (s *ResourceRepositoryService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.ResourceRepo
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		item.Name = rrCleanString(value)
	}
	if value, ok := patch["repo_type"]; ok {
		item.RepoType = rrCleanString(value)
	}
	if value, ok := patch["provider"]; ok {
		item.Provider = rrCleanString(value)
	}
	if value, ok := patch["repository_url"]; ok {
		item.RepositoryURL = rrCleanString(value)
	}
	if value, ok := patch["repository_branch"]; ok {
		item.RepositoryBranch = rrCleanString(value)
	}
	if value, ok := patch["templates_path"]; ok {
		item.TemplatesPath = rrCleanString(value)
	}
	if value, ok := patch["host_id"]; ok {
		item.HostID = rrOptionalUint(value)
	}
	if value, ok := patch["enabled"]; ok {
		item.Enabled = rrBoolValue(value, item.Enabled)
	}
	if value, ok := patch["last_run_at"]; ok {
		item.LastRunAt = rrParseOptionalTime(value)
	}
	if value, ok := patch["tags"]; ok {
		item.Tags = rrJSONString(value)
	}
	if value, ok := patch["config"]; ok {
		item.Config = rrJSONString(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return resourceRepositoryToRow(item), true
}

func (s *ResourceRepositoryService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.ResourceRepo{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *ResourceRepositoryService) TouchLastRun(id uint, now time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	return s.db.Model(&models.ResourceRepo{}).Where("id = ?", id).Update("last_run_at", &now).Error
}

func (s *ResourceRepositoryService) ToRow(item models.ResourceRepo) map[string]any {
	return resourceRepositoryToRow(item)
}

func resourceRepositoryToRow(item models.ResourceRepo) map[string]any {
	return map[string]any{
		"id":                item.ID,
		"name":              item.Name,
		"repo_type":         item.RepoType,
		"provider":          item.Provider,
		"repository_url":    item.RepositoryURL,
		"repository_branch": item.RepositoryBranch,
		"templates_path":    item.TemplatesPath,
		"host_id":           item.HostID,
		"enabled":           item.Enabled,
		"last_run_at":       item.LastRunAt,
		"tags":              item.TagList(),
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
	}
}

func rrOptionalUint(value any) *uint {
	switch typed := value.(type) {
	case nil:
		return nil
	case uint:
		copy := typed
		return &copy
	case uint64:
		copy := uint(typed)
		return &copy
	case int:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case int64:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case float64:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		var parsed uint64
		if _, err := fmt.Sscan(text, &parsed); err == nil && parsed > 0 {
			copy := uint(parsed)
			return &copy
		}
	}
	return nil
}

func rrParseOptionalTime(value any) *time.Time {
	switch typed := value.(type) {
	case time.Time:
		copy := typed
		return &copy
	case *time.Time:
		return typed
	case string:
		text := strings.TrimSpace(typed)
		if text == "" || text == "<nil>" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, text); err == nil {
			return &parsed
		}
	}
	return nil
}

func rrFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func rrCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func rrValueOrDefault(value any, fallback string) string {
	text := rrCleanString(value)
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func rrBoolValue(value any, fallback bool) bool {
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

func rrJSONString(value any) string {
	if value == nil {
		return "[]"
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" || text == "<nil>" {
			return "[]"
		}
		return text
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return "[]"
		}
		return string(payload)
	}
}
