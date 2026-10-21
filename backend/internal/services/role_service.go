package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"cloudctf/backend/internal/models"
	"gorm.io/gorm"
)

type RoleService struct{ db *gorm.DB }

func NewRoleService(db *gorm.DB) *RoleService { return &RoleService{db: db} }

func (s *RoleService) GetByID(id any) (*models.Role, error) {
	var role models.Role
	if err := s.db.First(&role, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	var roles []models.Role
	db := s.db.Model(&models.Role{})
	if name := roleFirstQuery(query, "name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, roleToRow(role))
	}
	return rows, int(total), nil
}

func (s *RoleService) Create(row map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(fmt.Sprint(row["name"]))
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	role := models.Role{Name: name, Description: roleCleanString(row["description"]), Permissions: rolePermissionsFromAny(row["permissions"]), IsSystem: false}
	if err := s.db.Create(&role).Error; err != nil {
		return nil, err
	}
	return roleToRow(role), nil
}

func (s *RoleService) Update(id any, patch map[string]any) (map[string]any, bool) {
	var role models.Role
	if err := s.db.First(&role, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if role.IsSystem {
		return roleToRow(role), true
	}
	if value, ok := patch["name"]; ok {
		role.Name = roleCleanString(value)
	}
	if value, ok := patch["description"]; ok {
		role.Description = roleCleanString(value)
	}
	if value, ok := patch["permissions"]; ok {
		role.Permissions = rolePermissionsFromAny(value)
	}
	if err := s.db.Save(&role).Error; err != nil {
		return nil, false
	}
	return roleToRow(role), true
}

func (s *RoleService) Delete(id any) bool {
	result := s.db.Where("is_system = ?", false).Delete(&models.Role{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func roleToRow(role models.Role) map[string]any {
	return map[string]any{"id": role.ID, "name": role.Name, "description": role.Description, "permissions": rolePermissionList(role.Permissions), "is_system": role.IsSystem, "created_at": role.CreatedAt, "updated_at": role.UpdatedAt}
}
func roleFirstQuery(query map[string][]string, key string) string {
	if len(query[key]) == 0 {
		return ""
	}
	return query[key][0]
}
func roleCleanString(value any) string {
	text := fmt.Sprint(value)
	if text == "<nil>" {
		return ""
	}
	return strings.TrimSpace(text)
}
func rolePermissionsFromAny(value any) string {
	if text := roleCleanString(value); strings.HasPrefix(text, "[") {
		return text
	}
	switch typed := value.(type) {
	case []string:
		payload, _ := json.Marshal(typed)
		return string(payload)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := roleCleanString(item); text != "" {
				items = append(items, text)
			}
		}
		payload, _ := json.Marshal(items)
		return string(payload)
	}
	if text := roleCleanString(value); text != "" {
		payload, _ := json.Marshal([]string{text})
		return string(payload)
	}
	return "[]"
}
func rolePermissionList(raw string) []string {
	var items []string
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}
