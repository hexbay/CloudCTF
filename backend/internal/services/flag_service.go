package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type FlagService struct {
	db *gorm.DB
}

func NewFlagService(db *gorm.DB) *FlagService {
	return &FlagService{db: db}
}

func (s *FlagService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Challenge
	db := s.db.Model(&models.Challenge{})
	if keyword := fgFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("name LIKE ? OR flag LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
		rows = append(rows, flagRow(item))
	}
	return rows, int(total), nil
}

func (s *FlagService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	challengeID := fgUintValue(row["challenge_id"])
	if challengeID == 0 {
		return nil, fmt.Errorf("challenge_id is required")
	}
	var item models.Challenge
	if err := s.db.First(&item, challengeID).Error; err != nil {
		return nil, fmt.Errorf("challenge not found")
	}
	item.Flag = fgCleanString(row["flag"])
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	return flagRow(item), nil
}

func (s *FlagService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.Challenge
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["flag"]; ok {
		item.Flag = fgCleanString(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return flagRow(item), true
}

func (s *FlagService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	var item models.Challenge
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	item.Flag = ""
	if err := s.db.Save(&item).Error; err != nil {
		return false
	}
	return true
}

func flagRow(item models.Challenge) map[string]any {
	return map[string]any{
		"id":         item.ID,
		"name":       item.Name,
		"flag":       item.Flag,
		"category":   item.Category,
		"is_dynamic": item.IsDynamic,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	}
}

func fgFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func fgCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func fgUintValue(value any) uint {
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
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0
		}
		var parsed uint64
		if _, err := fmt.Sscan(text, &parsed); err == nil {
			return uint(parsed)
		}
	}
	return 0
}

var _ = time.Time{}
