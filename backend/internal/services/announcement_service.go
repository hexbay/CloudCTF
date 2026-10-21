package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type AnnouncementService struct {
	db *gorm.DB
}

func NewAnnouncementService(db *gorm.DB) *AnnouncementService {
	return &AnnouncementService{db: db}
}

func (s *AnnouncementService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Announcement
	db := s.db.Model(&models.Announcement{})
	if keyword := annFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("title LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if isPublished := annFirstQuery(query, "is_published"); isPublished != "" {
		db = db.Where("is_published = ?", isPublished == "true")
	}
	if targetType := annFirstQuery(query, "target_type"); targetType != "" {
		db = db.Where("target_type = ?", targetType)
	}
	if targetID := annFirstQuery(query, "target_id"); targetID != "" {
		db = db.Where("target_id = ?", targetID)
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("is_pinned DESC, pin_order DESC, id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, announcementToRow(item))
	}
	return rows, int(total), nil
}

func (s *AnnouncementService) GetByID(id any) (*models.Announcement, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.Announcement
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *AnnouncementService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.Announcement{
		Title:       annCleanString(row["title"]),
		Content:     annCleanString(row["content"]),
		Summary:     annCleanString(row["summary"]),
		AuthorID:    annUintValue(row["author_id"]),
		IsPinned:    annBoolValue(row["is_pinned"], false),
		PinOrder:    annIntValue(row["pin_order"], 0),
		IsPublished: annBoolValue(row["is_published"], false),
		TargetType:  annValueOrDefault(row["target_type"], "all"),
		TargetID:    annOptionalUint(row["target_id"]),
		Attachments: annJSONString(row["attachments"]),
	}
	if item.IsPublished {
		now := time.Now()
		item.PublishedAt = &now
	}
	if item.Title == "" || item.AuthorID == 0 {
		return nil, fmt.Errorf("title and author_id are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return announcementToRow(item), nil
}

func (s *AnnouncementService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.Announcement
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["title"]; ok {
		item.Title = annCleanString(value)
	}
	if value, ok := patch["content"]; ok {
		item.Content = annCleanString(value)
	}
	if value, ok := patch["summary"]; ok {
		item.Summary = annCleanString(value)
	}
	if value, ok := patch["is_pinned"]; ok {
		item.IsPinned = annBoolValue(value, item.IsPinned)
	}
	if value, ok := patch["pin_order"]; ok {
		item.PinOrder = annIntValue(value, item.PinOrder)
	}
	if value, ok := patch["is_published"]; ok {
		item.IsPublished = annBoolValue(value, item.IsPublished)
		if item.IsPublished && item.PublishedAt == nil {
			now := time.Now()
			item.PublishedAt = &now
		}
	}
	if value, ok := patch["target_type"]; ok {
		item.TargetType = annCleanString(value)
	}
	if value, ok := patch["target_id"]; ok {
		item.TargetID = annOptionalUint(value)
	}
	if value, ok := patch["attachments"]; ok {
		item.Attachments = annJSONString(value)
	}
	if value, ok := patch["view_count"]; ok {
		item.ViewCount = annIntValue(value, item.ViewCount)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return announcementToRow(item), true
}

func (s *AnnouncementService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Announcement{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *AnnouncementService) SetPublished(id any, published bool) bool {
	return s.updateState(id, func(item *models.Announcement) {
		item.IsPublished = published
		if published && item.PublishedAt == nil {
			now := time.Now()
			item.PublishedAt = &now
		}
	})
}

func (s *AnnouncementService) SetPinned(id any, pinned bool) bool {
	return s.updateState(id, func(item *models.Announcement) {
		item.IsPinned = pinned
	})
}

func (s *AnnouncementService) ToRow(item models.Announcement) map[string]any {
	return announcementToRow(item)
}

func (s *AnnouncementService) updateState(id any, apply func(*models.Announcement)) bool {
	if s == nil || s.db == nil {
		return false
	}
	var item models.Announcement
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	apply(&item)
	return s.db.Save(&item).Error == nil
}

func announcementToRow(item models.Announcement) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"title":        item.Title,
		"content":      item.Content,
		"summary":      item.Summary,
		"author_id":    item.AuthorID,
		"is_pinned":    item.IsPinned,
		"pin_order":    item.PinOrder,
		"is_published": item.IsPublished,
		"target_type":  item.TargetType,
		"target_id":    item.TargetID,
		"attachments":  annParseJSONArray(item.Attachments),
		"view_count":   item.ViewCount,
		"published_at": item.PublishedAt,
		"created_at":   item.CreatedAt,
		"updated_at":   item.UpdatedAt,
	}
}

func annFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func annCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func annValueOrDefault(value any, fallback string) string {
	text := annCleanString(value)
	if text == "" {
		return fallback
	}
	return text
}

func annBoolValue(value any, fallback bool) bool {
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

func annIntValue(value any, fallback int) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		var parsed int
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	}
	return fallback
}

func annUintValue(value any) uint {
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
	case float64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case string:
		var parsed uint
		_, _ = fmt.Sscan(strings.TrimSpace(typed), &parsed)
		return parsed
	}
	return 0
}

func annOptionalUint(value any) *uint {
	parsed := annUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func annJSONString(value any) string {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return "[]"
		}
		return trimmed
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return "[]"
		}
		return string(payload)
	}
}

func annParseJSONArray(raw string) []map[string]any {
	if strings.TrimSpace(raw) == "" {
		return []map[string]any{}
	}
	var result []map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err == nil {
		return result
	}
	var generic []any
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return []map[string]any{}
	}
	rows := make([]map[string]any, 0, len(generic))
	for _, item := range generic {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
			continue
		}
		rows = append(rows, map[string]any{"value": item})
	}
	return rows
}
