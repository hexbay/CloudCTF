package services

import (
	"fmt"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type AuditLogService struct {
	db *gorm.DB
}

func NewAuditLogService(db *gorm.DB) *AuditLogService {
	return &AuditLogService{db: db}
}

func (s *AuditLogService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.AuditLog
	db := s.db.Model(&models.AuditLog{})
	if actionType := auditFirstQuery(query, "action_type"); actionType != "" {
		db = db.Where("action_type = ?", actionType)
	}
	if resourceType := auditFirstQuery(query, "resource_type"); resourceType != "" {
		db = db.Where("resource_type = ?", resourceType)
	}
	if username := auditFirstQuery(query, "username"); username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}
	if keyword := auditFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("description LIKE ? OR resource_type LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
		rows = append(rows, auditToRow(item))
	}
	return rows, int(total), nil
}

func (s *AuditLogService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.AuditLog{
		UserID:       auditOptionalUint(row["user_id"]),
		Username:     fmt.Sprint(row["username"]),
		ActionType:   fmt.Sprint(row["action_type"]),
		ResourceType: fmt.Sprint(row["resource_type"]),
		ResourceID:   fmt.Sprint(row["resource_id"]),
		Description:  fmt.Sprint(row["description"]),
		IPAddress:    fmt.Sprint(row["ip_address"]),
		UserAgent:    fmt.Sprint(row["user_agent"]),
	}
	if item.ActionType == "" || item.ResourceType == "" {
		return nil, nil
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return auditToRow(item), nil
}

func (s *AuditLogService) Recent(limit int) []models.AuditLog {
	if s == nil || s.db == nil {
		return []models.AuditLog{}
	}
	var items []models.AuditLog
	_ = s.db.Order("id DESC").Limit(limit).Find(&items).Error
	return items
}

func (s *AuditLogService) DashboardSummary(days int) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	type row struct {
		Date  string
		Count int64
	}
	var results []row
	_ = s.db.Model(&models.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -days)).
		Group("DATE(created_at)").Order("date ASC").Find(&results).Error
	data := make([]map[string]any, 0, len(results))
	for _, r := range results {
		data = append(data, map[string]any{"date": r.Date, "count": r.Count})
	}
	return data
}

func (s *AuditLogService) RecentRows(limit int) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	var logs []models.AuditLog
	_ = s.db.Order("id DESC").Limit(limit).Find(&logs).Error
	rows := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, map[string]any{
			"id":            log.ID,
			"level":         "info",
			"message":       log.Description,
			"action_type":   log.ActionType,
			"resource_type": log.ResourceType,
			"created_at":    log.CreatedAt,
		})
	}
	return rows
}

func (s *AuditLogService) CountBetween(start, end time.Time) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var total int64
	_ = s.db.Model(&models.AuditLog{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&total).Error
	return total
}

func (s *AuditLogService) VisitHistory(limit int) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	type row struct {
		Date   string
		Visits int64
	}
	var rows []row
	_ = s.db.Model(&models.AuditLog{}).
		Select("DATE(created_at) as date, COUNT(*) as visits").
		Group("DATE(created_at)").
		Order("date DESC").
		Limit(limit).
		Scan(&rows).Error
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{"date": row.Date, "visits": row.Visits})
	}
	return result
}

func (s *AuditLogService) SummaryCounts() (int64, int64) {
	if s == nil || s.db == nil {
		return 0, 0
	}
	var total int64
	_ = s.db.Model(&models.AuditLog{}).Count(&total).Error
	var sensitive int64
	_ = s.db.Model(&models.AuditLog{}).Where("action_type IN ?", []string{"delete", "ban", "reset_password", "backup_restore"}).Count(&sensitive).Error
	return total, sensitive
}

func auditToRow(item models.AuditLog) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"user_id":       item.UserID,
		"username":      item.Username,
		"action_type":   item.ActionType,
		"resource_type": item.ResourceType,
		"resource_id":   item.ResourceID,
		"description":   item.Description,
		"ip_address":    item.IPAddress,
		"user_agent":    item.UserAgent,
		"created_at":    item.CreatedAt,
	}
}

func auditFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func auditOptionalUint(value any) *uint {
	var parsed uint
	_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
	if parsed == 0 {
		return nil
	}
	return &parsed
}
