package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Notification
	db := s.db.Model(&models.Notification{}).
		Joins("LEFT JOIN user AS receiver ON receiver.id = notification.receiver_id").
		Joins("LEFT JOIN role AS receiver_role ON receiver_role.id = receiver.role_id")
	if audience := notifFirstQuery(query, "audience"); audience != "" {
		if audience == "admin" {
			db = db.Where("notification.audience = ? OR (notification.audience = '' AND receiver_role.name <> ?)", audience, "player")
		} else {
			db = db.Where("notification.audience = ? OR (notification.audience = '' AND receiver_role.name = ?)", audience, "player")
		}
	}
	if receiverID := notifFirstQuery(query, "receiver_id"); receiverID != "" {
		db = db.Where("notification.receiver_id = ?", receiverID)
	}
	if messageType := notifFirstQuery(query, "message_type"); messageType != "" {
		db = db.Where("notification.message_type = ?", messageType)
	}
	if isRead := notifFirstQuery(query, "is_read"); isRead != "" {
		db = db.Where("notification.is_read = ?", isRead == "true")
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("notification.id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, notificationToRow(item))
	}
	return rows, int(total), nil
}

func (s *NotificationService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.Notification{
		Title:       notifCleanString(row["title"]),
		Content:     notifCleanString(row["content"]),
		MessageType: notifValueOrDefault(row["message_type"], "system"),
		Audience:    notifValueOrDefault(row["audience"], "player"),
		Priority:    notifValueOrDefault(row["priority"], "normal"),
		SenderID:    notifOptionalUint(row["sender_id"]),
		SenderName:  notifCleanString(row["sender_name"]),
		ReceiverID:  notifUintValue(row["receiver_id"]),
		RelatedType: notifCleanString(row["related_type"]),
		RelatedID:   notifOptionalUint(row["related_id"]),
		IsRead:      false,
		SendEmail:   notifBoolValue(row["send_email"], false),
	}
	if item.Title == "" || item.ReceiverID == 0 {
		return nil, fmt.Errorf("title and receiver_id are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return notificationToRow(item), nil
}

func (s *NotificationService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.Notification
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["is_read"]; ok {
		item.IsRead = notifBoolValue(value, item.IsRead)
		if item.IsRead {
			now := time.Now()
			item.ReadAt = &now
		}
	}
	if value, ok := patch["title"]; ok {
		item.Title = notifCleanString(value)
	}
	if value, ok := patch["content"]; ok {
		item.Content = notifCleanString(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return notificationToRow(item), true
}

func (s *NotificationService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Notification{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *NotificationService) CountUnread(receiverID uint, audience string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("db unavailable")
	}
	var count int64
	db := s.db.Model(&models.Notification{}).Where("receiver_id = ? AND is_read = ?", receiverID, false)
	if audience != "" {
		db = db.Where("audience = ?", audience)
	}
	return count, db.Count(&count).Error
}

func (s *NotificationService) CountUnreadSince(receiverID uint, audience string, since time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("db unavailable")
	}
	var count int64
	db := s.db.Model(&models.Notification{}).Where("receiver_id = ? AND is_read = ? AND created_at > ?", receiverID, false, since)
	if audience != "" {
		db = db.Where("audience = ?", audience)
	}
	return count, db.Count(&count).Error
}

func (s *NotificationService) Summary() (int64, int64, error) {
	if s == nil || s.db == nil {
		return 0, 0, fmt.Errorf("db unavailable")
	}
	var total, unread int64
	if err := s.db.Model(&models.Notification{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := s.db.Model(&models.Notification{}).Where("is_read = ?", false).Count(&unread).Error; err != nil {
		return 0, 0, err
	}
	return total, unread, nil
}

func (s *NotificationService) MarkAllRead(receiverID uint, audience string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	now := time.Now()
	db := s.db.Model(&models.Notification{}).Where("receiver_id = ? AND is_read = ?", receiverID, false)
	if audience != "" {
		db = db.Where("audience = ?", audience)
	}
	return db.Updates(map[string]any{"is_read": true, "read_at": &now}).Error
}

func (s *NotificationService) MarkOneRead(id any, receiverID uint, audience string) bool {
	if s == nil || s.db == nil {
		return false
	}
	now := time.Now()
	result := s.db.Model(&models.Notification{}).
		Where("id = ? AND receiver_id = ? AND audience = ?", fmt.Sprint(id), receiverID, audience).
		Updates(map[string]any{"is_read": true, "read_at": &now})
	return result.Error == nil && result.RowsAffected > 0
}

func (s *NotificationService) ReceiverIDsByAudience(audience string) []uint {
	var users []models.User
	db := s.db.Preload("Role")
	if audience == "admin" {
		db = db.Joins("Role").Where("Role.name <> ?", "player")
	} else {
		db = db.Joins("Role").Where("Role.name = ?", "player")
	}
	_ = db.Find(&users).Error
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func (s *NotificationService) ReceiverIDsByRole(roleID uint) []uint {
	var users []models.User
	_ = s.db.Where("role_id = ?", roleID).Find(&users).Error
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func (s *NotificationService) ReceiverIDsByTeam(teamID uint) []uint {
	var users []models.User
	_ = s.db.Where("team_id = ? AND is_active = ?", teamID, true).Find(&users).Error
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func notificationToRow(item models.Notification) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"title":        item.Title,
		"content":      item.Content,
		"message_type": item.MessageType,
		"audience":     item.Audience,
		"priority":     item.Priority,
		"sender_id":    item.SenderID,
		"sender_name":  item.SenderName,
		"receiver_id":  item.ReceiverID,
		"related_type": item.RelatedType,
		"related_id":   item.RelatedID,
		"is_read":      item.IsRead,
		"read_at":      item.ReadAt,
		"send_email":   item.SendEmail,
		"created_at":   item.CreatedAt,
	}
}

func notifFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func notifCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func notifValueOrDefault(value any, fallback string) string {
	text := notifCleanString(value)
	if text == "" {
		return fallback
	}
	return text
}

func notifBoolValue(value any, fallback bool) bool {
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

func notifUintValue(value any) uint {
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

func notifOptionalUint(value any) *uint {
	parsed := notifUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}
