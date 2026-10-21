package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type VirtualizationHostService struct {
	db *gorm.DB
}

func NewVirtualizationHostService(db *gorm.DB) *VirtualizationHostService {
	return &VirtualizationHostService{db: db}
}

func (s *VirtualizationHostService) GetByID(id any) (*models.VirtualizationHost, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.VirtualizationHost
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *VirtualizationHostService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.VirtualizationHost
	db := s.db.Model(&models.VirtualizationHost{})
	if hostType := firstQuery(query, "host_type"); hostType != "" {
		db = db.Where("host_type = ?", hostType)
	}
	if status := firstQuery(query, "status"); status != "" {
		db = db.Where("status = ?", status)
	}
	if keyword := firstQuery(query, "keyword"); keyword != "" {
		db = db.Where("name LIKE ? OR endpoint LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
		rows = append(rows, virtualizationHostToRow(item))
	}
	return rows, int(total), nil
}

func (s *VirtualizationHostService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.VirtualizationHost{
		Name:        cleanString(row["name"]),
		Description: cleanString(row["description"]),
		HostType:    valueOrDefault(row["host_type"], "docker"),
		IsDefault:   boolValue(row["is_default"], false),
		Endpoint:    cleanString(row["endpoint"]),
		OutIP:       cleanString(row["out_ip"]),
		APIKey:      cleanString(row["api_key"]),
		Username:    cleanString(row["username"]),
		Password:    cleanString(row["password"]),
		Status:      "offline",
		IsActive:    boolValue(row["is_active"], true),
		Config:      jsonString(row["config"]),
	}
	if item.HostType == "docker" && item.Endpoint == "" {
		item.Endpoint = defaultDockerEndpoint()
	}
	if item.Name == "" || item.Endpoint == "" {
		return nil, fmt.Errorf("name and endpoint are required")
	}
	if item.IsDefault {
		if err := s.clearDefault(item.HostType, 0); err != nil {
			return nil, err
		}
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return virtualizationHostToRow(item), nil
}

func (s *VirtualizationHostService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.VirtualizationHost
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		item.Name = cleanString(value)
	}
	if value, ok := patch["description"]; ok {
		item.Description = cleanString(value)
	}
	if value, ok := patch["host_type"]; ok {
		item.HostType = cleanString(value)
	}
	if value, ok := patch["is_default"]; ok {
		item.IsDefault = boolValue(value, item.IsDefault)
	}
	if value, ok := patch["endpoint"]; ok {
		item.Endpoint = cleanString(value)
	}
	if value, ok := patch["out_ip"]; ok {
		item.OutIP = cleanString(value)
	}
	if value, ok := patch["api_key"]; ok && cleanString(value) != "" {
		item.APIKey = cleanString(value)
	}
	if value, ok := patch["username"]; ok {
		item.Username = cleanString(value)
	}
	if value, ok := patch["password"]; ok && cleanString(value) != "" {
		item.Password = cleanString(value)
	}
	if value, ok := patch["status"]; ok {
		item.Status = cleanString(value)
	}
	if value, ok := patch["is_active"]; ok {
		item.IsActive = boolValue(value, item.IsActive)
	}
	if value, ok := patch["config"]; ok {
		item.Config = jsonString(value)
	}
	if item.IsDefault {
		if err := s.clearDefault(item.HostType, item.ID); err != nil {
			return nil, false
		}
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return virtualizationHostToRow(item), true
}

func (s *VirtualizationHostService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.VirtualizationHost{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *VirtualizationHostService) GetDefaultByType(hostType string) (*models.VirtualizationHost, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.VirtualizationHost
	if err := s.db.
		Where("host_type = ? AND is_default = ?", strings.TrimSpace(hostType), true).
		Order("id ASC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *VirtualizationHostService) UpdateRuntime(host *models.VirtualizationHost, status string, resources map[string]any) error {
	if s == nil || s.db == nil || host == nil {
		return fmt.Errorf("host unavailable")
	}
	now := time.Now()
	host.Status = status
	host.LastCheck = &now
	if resources != nil {
		payload, err := json.Marshal(resources)
		if err != nil {
			return err
		}
		host.Resources = string(payload)
	}
	return s.db.Save(host).Error
}

func (s *VirtualizationHostService) ListDockerHosts(hostID int) ([]models.VirtualizationHost, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var hosts []models.VirtualizationHost
	db := s.db.Model(&models.VirtualizationHost{}).Where("host_type = ?", "docker")
	if hostID > 0 {
		db = db.Where("id = ?", hostID)
	}
	if err := db.Order("id DESC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no docker hosts found")
	}
	return hosts, nil
}

func (s *VirtualizationHostService) ResolveDockerHostByContainerID(
	ctx context.Context,
	containerID string,
	inspector func(context.Context, models.VirtualizationHost, string) error,
) (*models.VirtualizationHost, error) {
	hosts, err := s.ListDockerHosts(0)
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		inspectErr := inspector(timeoutCtx, host, containerID)
		cancel()
		if inspectErr == nil {
			copy := host
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("container not found")
}

func (s *VirtualizationHostService) clearDefault(hostType string, excludeID uint) error {
	query := s.db.Model(&models.VirtualizationHost{}).
		Where("host_type = ? AND is_default = ?", hostType, true)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	return query.Update("is_default", false).Error
}

func virtualizationHostToRow(item models.VirtualizationHost) map[string]any {
	var resources map[string]any
	if item.Resources != "" {
		_ = json.Unmarshal([]byte(item.Resources), &resources)
	}
	return map[string]any{
		"id":          item.ID,
		"name":        item.Name,
		"description": item.Description,
		"host_type":   item.HostType,
		"is_default":  item.IsDefault,
		"endpoint":    item.Endpoint,
		"out_ip":      item.OutIP,
		"username":    item.Username,
		"status":      item.Status,
		"last_check":  item.LastCheck,
		"resources":   resources,
		"is_active":   item.IsActive,
		"created_at":  item.CreatedAt,
		"updated_at":  item.UpdatedAt,
	}
}

func (s *VirtualizationHostService) ToRow(item models.VirtualizationHost) map[string]any {
	return virtualizationHostToRow(item)
}

func boolValue(value any, fallback bool) bool {
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

func jsonString(value any) string {
	if value == nil {
		return "{}"
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" || text == "<nil>" {
			return "{}"
		}
		return text
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return "{}"
		}
		return string(payload)
	}
}
