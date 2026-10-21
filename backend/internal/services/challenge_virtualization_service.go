package services

import (
	"fmt"
	"strings"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type ChallengeVirtualizationService struct {
	db *gorm.DB
}

func NewChallengeVirtualizationService(db *gorm.DB) *ChallengeVirtualizationService {
	return &ChallengeVirtualizationService{db: db}
}

func (s *ChallengeVirtualizationService) ListByChallengeID(challengeID string) ([]map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	id := cvOptionalUint(challengeID)
	if id == nil {
		return []map[string]any{}, nil
	}
	var items []models.ChallengeVirtualization
	if err := s.db.Where("challenge_id = ?", *id).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, s.toRow(item))
	}
	return rows, nil
}

func (s *ChallengeVirtualizationService) ListChallenges(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Challenge
	db := s.db.Model(&models.Challenge{}).Where("is_dynamic = ?", true)
	if keyword := cvFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
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
		rows = append(rows, map[string]any{
			"id":                item.ID,
			"name":              item.Name,
			"container_image":   item.ContainerImage,
			"container_port":    item.ContainerPort,
			"container_timeout": item.ContainerTimeout,
			"host_id":           item.HostID,
			"template_id":       item.TemplateID,
			"is_dynamic":        item.IsDynamic,
			"created_at":        item.CreatedAt,
			"updated_at":        item.UpdatedAt,
		})
	}
	return rows, int(total), nil
}

func (s *ChallengeVirtualizationService) Create(challengeID string, payload map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	id := cvOptionalUint(challengeID)
	if id == nil {
		return nil, fmt.Errorf("challenge_id is required")
	}
	item := models.ChallengeVirtualization{
		ChallengeID: *id,
		Status:      cvDefault(cvCleanString(payload["status"]), "running"),
		HostID:      cvOptionalUint(payload["host_id"]),
		TemplateID:  cvOptionalUint(payload["template_id"]),
	}
	if err := s.db.Where("challenge_id = ?", item.ChallengeID).Delete(&models.ChallengeVirtualization{}).Error; err != nil {
		return nil, err
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return s.toRow(item), nil
}

func (s *ChallengeVirtualizationService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.ChallengeVirtualization{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *ChallengeVirtualizationService) FindTemplateBinding(challengeID string) (map[string]any, bool) {
	rows, err := s.ListByChallengeID(challengeID)
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	return rows[0], true
}

func (s *ChallengeVirtualizationService) toRow(item models.ChallengeVirtualization) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"challenge_id": item.ChallengeID,
		"status":       item.Status,
		"host_id":      item.HostID,
		"template_id":  item.TemplateID,
		"created_at":   item.CreatedAt,
		"updated_at":   item.UpdatedAt,
	}
}

func cvFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func cvCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func cvDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func cvOptionalUint(value any) *uint {
	switch typed := value.(type) {
	case uint:
		if typed == 0 {
			return nil
		}
		v := typed
		return &v
	case *uint:
		if typed == nil || *typed == 0 {
			return nil
		}
		v := *typed
		return &v
	case int:
		if typed <= 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case int64:
		if typed <= 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case float64:
		if typed <= 0 {
			return nil
		}
		v := uint(typed)
		return &v
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" || typed == "0" {
			return nil
		}
		var v uint
		if _, err := fmt.Sscan(typed, &v); err != nil || v == 0 {
			return nil
		}
		return &v
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" || text == "0" {
			return nil
		}
		var v uint
		if _, err := fmt.Sscan(text, &v); err != nil || v == 0 {
			return nil
		}
		return &v
	}
}
