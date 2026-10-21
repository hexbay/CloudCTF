package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type VulnLabService struct {
	db *gorm.DB
}

func NewVulnLabService(db *gorm.DB) *VulnLabService {
	return &VulnLabService{db: db}
}

func (s *VulnLabService) GetByID(id any) (*models.VulnLab, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.VulnLab
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *VulnLabService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.VulnLab{
		Name:           vlCleanString(row["name"]),
		CVEID:          vlCleanString(row["cve_id"]),
		Severity:       vlValueOrDefault(row["severity"], "LOW"),
		Summary:        vlCleanString(row["summary"]),
		Description:    vlCleanString(row["description"]),
		ReproduceSteps: vlCleanString(row["reproduce_steps"]),
		ReferenceLinks: vlJSONString(row["reference_links"]),
		IsPublic:       vlBoolValue(row["is_public"], true),
		IsActive:       vlBoolValue(row["is_active"], true),
		TemplateID:     vlUintValue(row["template_id"]),
		TimeoutSeconds: vlIntValueDefault(row["timeout_seconds"], 3600),
	}
	if item.Name == "" || item.TemplateID == 0 {
		return nil, fmt.Errorf("name and template_id are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return vulnLabToRow(item), nil
}

func (s *VulnLabService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.VulnLab
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		item.Name = vlCleanString(value)
	}
	if value, ok := patch["cve_id"]; ok {
		item.CVEID = vlCleanString(value)
	}
	if value, ok := patch["severity"]; ok {
		item.Severity = vlCleanString(value)
	}
	if value, ok := patch["summary"]; ok {
		item.Summary = vlCleanString(value)
	}
	if value, ok := patch["description"]; ok {
		item.Description = vlCleanString(value)
	}
	if value, ok := patch["reproduce_steps"]; ok {
		item.ReproduceSteps = vlCleanString(value)
	}
	if value, ok := patch["reference_links"]; ok {
		item.ReferenceLinks = vlJSONString(value)
	}
	if value, ok := patch["is_public"]; ok {
		item.IsPublic = vlBoolValue(value, item.IsPublic)
	}
	if value, ok := patch["is_active"]; ok {
		item.IsActive = vlBoolValue(value, item.IsActive)
	}
	if value, ok := patch["template_id"]; ok {
		item.TemplateID = vlUintValue(value)
	}
	if value, ok := patch["timeout_seconds"]; ok {
		item.TimeoutSeconds = vlIntValue(value, item.TimeoutSeconds)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return vulnLabToRow(item), true
}

func (s *VulnLabService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.VulnLab{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *VulnLabService) ToRow(item models.VulnLab) map[string]any {
	return vulnLabToRow(item)
}

func (s *VulnLabService) CreateOrUpdateByTemplate(templateID uint, name string, cveID string, summary string, description string, reproduceSteps string, severity string, referenceLinks []string, now time.Time) error {
	if s == nil || s.db == nil || templateID == 0 {
		return nil
	}
	encodedLinks, _ := json.Marshal(referenceLinks)
	var existing models.VulnLab
	if err := s.db.Where("template_id = ? OR name = ?", templateID, name).First(&existing).Error; err == nil {
		existing.Name = name
		existing.CVEID = cveID
		existing.Summary = summary
		existing.Description = description
		existing.ReproduceSteps = reproduceSteps
		existing.Severity = severity
		existing.ReferenceLinks = string(encodedLinks)
		existing.TemplateID = templateID
		existing.UpdatedAt = now
		return s.db.Save(&existing).Error
	}
	lab := models.VulnLab{
		Name:           name,
		CVEID:          cveID,
		Severity:       severity,
		Summary:        summary,
		Description:    description,
		ReproduceSteps: reproduceSteps,
		ReferenceLinks: string(encodedLinks),
		IsPublic:       true,
		IsActive:       false,
		TemplateID:     templateID,
		TimeoutSeconds: 3600,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return s.db.Create(&lab).Error
}

func vulnLabToRow(item models.VulnLab) map[string]any {
	return map[string]any{
		"id":              item.ID,
		"name":            item.Name,
		"cve_id":          item.CVEID,
		"severity":        item.Severity,
		"summary":         item.Summary,
		"description":     item.Description,
		"reproduce_steps": item.ReproduceSteps,
		"reference_links": item.ReferenceLinkList(),
		"is_public":       item.IsPublic,
		"is_active":       item.IsActive,
		"template_id":     item.TemplateID,
		"timeout_seconds": item.TimeoutSeconds,
		"created_at":      item.CreatedAt,
		"updated_at":      item.UpdatedAt,
	}
}

func vlCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func vlValueOrDefault(value any, fallback string) string {
	text := vlCleanString(value)
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func vlBoolValue(value any, fallback bool) bool {
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

func vlUintValue(value any) uint {
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
		parsed, err := strconv.ParseUint(strings.TrimSpace(typed), 10, 64)
		if err == nil {
			return uint(parsed)
		}
	}
	return 0
}

func vlIntValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func vlIntValueDefault(value any, fallback int) int {
	return vlIntValue(value, fallback)
}

func vlJSONString(value any) string {
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
