package services

import (
	"encoding/json"
	"fmt"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type SystemSettingService struct {
	db *gorm.DB
}

func NewSystemSettingService(db *gorm.DB) *SystemSettingService {
	return &SystemSettingService{db: db}
}

func (s *SystemSettingService) Load(defaults map[string]any) map[string]any {
	result := make(map[string]any, len(defaults))
	for key, value := range defaults {
		result[key] = value
	}
	if s == nil || s.db == nil {
		return result
	}
	var items []models.SystemSetting
	if err := s.db.Find(&items).Error; err != nil {
		return result
	}
	for _, item := range items {
		var decoded any
		if err := json.Unmarshal([]byte(item.Value), &decoded); err != nil {
			result[item.Key] = item.Value
			continue
		}
		result[item.Key] = decoded
	}
	return result
}

func (s *SystemSettingService) Save(payload map[string]any, defaults map[string]any) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	settings := s.Load(defaults)
	for key, value := range payload {
		settings[key] = value
	}
	for key, value := range settings {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		item := models.SystemSetting{Key: key}
		if err := s.db.Where(&models.SystemSetting{Key: key}).First(&item).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			item = models.SystemSetting{Key: key, Value: string(encoded)}
			if err := s.db.Create(&item).Error; err != nil {
				return err
			}
			continue
		}
		item.Value = string(encoded)
		if err := s.db.Save(&item).Error; err != nil {
			return err
		}
	}
	return nil
}
