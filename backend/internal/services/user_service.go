package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) List(resource string, current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	var users []models.User
	db := s.db.Preload("Role").Model(&models.User{})
	if resource == "users" {
		db = db.Joins("Role").Where("Role.name = ?", "player")
	}
	if resource == "admins" {
		db = db.Joins("Role").Where("Role.name <> ?", "player")
	}
	if keyword := userFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("user.username LIKE ? OR user.email LIKE ? OR user.full_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if username := userFirstQuery(query, "username"); username != "" {
		db = db.Where("user.username LIKE ?", "%"+username+"%")
	}
	if email := userFirstQuery(query, "email"); email != "" {
		db = db.Where("user.email LIKE ?", "%"+email+"%")
	}
	if role := userFirstQuery(query, "role"); role != "" {
		db = db.Where("Role.name = ?", role)
	}
	if active := userFirstQuery(query, "is_active"); active != "" {
		db = db.Where("user.is_active = ?", active == "true")
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("user.id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(users))
	for _, user := range users {
		rows = append(rows, s.ToRowWithTeam(user))
	}
	return rows, int(total), nil
}

func (s *UserService) GetByID(id any) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Role").First(&user, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Create(resource string, row map[string]any) (map[string]any, error) {
	username := strings.TrimSpace(fmt.Sprint(row["username"]))
	email := strings.TrimSpace(fmt.Sprint(row["email"]))
	password := fmt.Sprint(row["password"])
	if username == "" || username == "<nil>" || email == "" || email == "<nil>" {
		return nil, fmt.Errorf("username and email are required")
	}
	if password == "" || password == "<nil>" {
		password = "password123"
	}
	roleName := fmt.Sprint(row["role"])
	if roleName == "" || roleName == "<nil>" {
		if resource == "admins" {
			roleName = "admin"
		} else {
			roleName = "player"
		}
	}
	var role models.Role
	if err := s.db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return nil, fmt.Errorf("role %q not found", roleName)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashed),
		FullName:       userCleanString(row["full_name"]),
		IsActive:       userBoolValue(row["is_active"], true),
		IsSuperuser:    role.Name == "superadmin",
		Tags:           userTagsJSON(row["tags"]),
		MustChangePassword: resource == "admins",
		RoleID:         &role.ID,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}
	user.Role = role
	return s.ToRow(user), nil
}

func (s *UserService) Update(id any, patch map[string]any) (map[string]any, bool) {
	var user models.User
	if err := s.db.Preload("Role").First(&user, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["username"]; ok {
		user.Username = userCleanString(value)
	}
	if value, ok := patch["email"]; ok {
		user.Email = userCleanString(value)
	}
	if value, ok := patch["full_name"]; ok {
		user.FullName = userCleanString(value)
	}
	if value, ok := patch["is_active"]; ok {
		user.IsActive = userBoolValue(value, user.IsActive)
	}
	if value, ok := patch["tags"]; ok {
		user.Tags = userTagsJSON(value)
	}
	if value, ok := patch["password"]; ok && userCleanString(value) != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(userCleanString(value)), bcrypt.DefaultCost)
		if err != nil {
			return nil, false
		}
		user.HashedPassword = string(hashed)
	}
	if value, ok := patch["role"]; ok && userCleanString(value) != "" {
		var role models.Role
		if err := s.db.Where("name = ?", userCleanString(value)).First(&role).Error; err != nil {
			return nil, false
		}
		user.RoleID = &role.ID
		user.IsSuperuser = role.Name == "superadmin"
	}
	if err := s.db.Save(&user).Error; err != nil {
		return nil, false
	}
	_ = s.db.Preload("Role").First(&user, user.ID).Error
	return s.ToRow(user), true
}

func (s *UserService) Delete(id any) bool {
	result := s.db.Delete(&models.User{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *UserService) SetActive(id any, active bool) (map[string]any, bool) {
	var user models.User
	if err := s.db.Preload("Role").First(&user, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	user.IsActive = active
	if err := s.db.Save(&user).Error; err != nil {
		return nil, false
	}
	_ = s.db.Preload("Role").First(&user, user.ID).Error
	return s.ToRow(user), true
}

func (s *UserService) UpdatePassword(id any, password string) bool {
	if strings.TrimSpace(password) == "" {
		return false
	}
	var user models.User
	if err := s.db.First(&user, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false
	}
	user.HashedPassword = string(hashed)
	user.MustChangePassword = true
	return s.db.Save(&user).Error == nil
}

func (s *UserService) ChangeOwnPassword(id any, password string) bool {
	if strings.TrimSpace(password) == "" {
		return false
	}
	var user models.User
	if err := s.db.First(&user, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false
	}
	user.HashedPassword = string(hashed)
	user.MustChangePassword = false
	return s.db.Save(&user).Error == nil
}

func (s *UserService) ApplyTags(id any, tags any) bool {
	var user models.User
	if err := s.db.First(&user, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	user.Tags = userTagsJSON(tags)
	return s.db.Save(&user).Error == nil
}

func (s *UserService) LeaveActiveTeams(userID uint, reason string, now time.Time) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"team_id": nil}).Error
}

func (s *UserService) SetCurrentTeam(userID uint, teamID *uint) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("team_id", teamID).Error
}

func (s *UserService) BatchDeleteNonSuperusers(ids []uint) int {
	deleted := 0
	for _, id := range ids {
		var user models.User
		if s.db.First(&user, id).Error == nil && !user.IsSuperuser {
			_ = s.db.Where("owner_id = ?", user.ID).Delete(&models.Team{}).Error
			if s.db.Delete(&user).Error == nil {
				deleted++
			}
		}
	}
	return deleted
}

func (s *UserService) BatchSetActive(ids []uint, active bool, skipUserID uint, protectSuperuser bool) int {
	updated := 0
	for _, id := range ids {
		if skipUserID != 0 && id == skipUserID {
			continue
		}
		var user models.User
		if s.db.First(&user, id).Error != nil {
			continue
		}
		if protectSuperuser && user.IsSuperuser {
			continue
		}
		user.IsActive = active
		if s.db.Save(&user).Error == nil {
			updated++
		}
	}
	return updated
}

func (s *UserService) Count(resource string) int {
	var total int64
	db := s.db.Model(&models.User{})
	if resource == "users" {
		db = db.Joins("Role").Where("Role.name = ?", "player")
	}
	if resource == "admins" {
		db = db.Joins("Role").Where("Role.name <> ?", "player")
	}
	_ = db.Count(&total).Error
	return int(total)
}

func (s *UserService) CountCreatedSince(since time.Time) int64 {
	var total int64
	_ = s.db.Model(&models.User{}).Where("created_at >= ?", since).Count(&total).Error
	return total
}

func (s *UserService) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Role").Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) FindByLogin(login string) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Role").Where("username = ? OR email = ?", login, login).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Search(keyword string, limit int) []map[string]any {
	if s == nil || s.db == nil || strings.TrimSpace(keyword) == "" {
		return []map[string]any{}
	}
	if limit <= 0 {
		limit = 20
	}
	var users []models.User
	_ = s.db.Preload("Role").
		Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).
		Find(&users).Error
	rows := make([]map[string]any, 0, len(users))
	for _, user := range users {
		rows = append(rows, map[string]any{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"full_name": user.FullName,
		})
	}
	return rows
}

func (s *UserService) UsernameExists(username string) bool {
	var total int64
	_ = s.db.Model(&models.User{}).Where("username = ?", username).Count(&total).Error
	return total > 0
}

func (s *UserService) ExistsByUsernameOrEmail(username, email string) bool {
	var total int64
	_ = s.db.Model(&models.User{}).Where("username = ? OR email = ?", username, email).Count(&total).Error
	return total > 0
}

func (s *UserService) RegisterPlayer(username, email, password, fullName string, roleID *uint) (*models.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashed),
		FullName:       fullName,
		IsActive:       true,
		IsSuperuser:    false,
		Tags:           "[]",
		RoleID:         roleID,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return s.GetByID(user.ID)
}

func (s *UserService) EnsureGeneratedUser(username, email, password, fullName string, roleID *uint) (*models.User, error) {
	if existing, err := s.GetByUsername(username); err == nil {
		return existing, nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{
		Username:       username,
		Email:          email,
		HashedPassword: string(hashed),
		FullName:       fullName,
		IsActive:       true,
		IsSuperuser:    false,
		Tags:           "[]",
		RoleID:         roleID,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return s.GetByID(user.ID)
}

func (s *UserService) GetRoleByName(name string) (*models.Role, error) {
	var role models.Role
	if err := s.db.Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *UserService) ListCompetitionAdmins(ids []uint) []models.User {
	var admins []models.User
	if len(ids) == 0 {
		_ = s.db.Preload("Role").Where("is_superuser = ? OR id = ?", true, 1).Find(&admins).Error
		return admins
	}
	_ = s.db.Preload("Role").Where("id IN ?", ids).Find(&admins).Error
	return admins
}

func (s *UserService) ToRow(user models.User) map[string]any {
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"full_name":    user.FullName,
		"is_active":    user.IsActive,
		"is_superuser": user.IsSuperuser,
		"tags":         userTagList(user.Tags),
		"role":         user.Role.Name,
		"role_id":      user.RoleID,
		"avatar":       user.Avatar,
		"team_id":      user.TeamID,
		"must_change_password": user.MustChangePassword,
		"teams":        []map[string]any{},
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	}
}

func (s *UserService) ToRowWithTeam(user models.User) map[string]any {
	row := s.ToRow(user)
	if user.TeamID != nil && *user.TeamID != 0 {
		var team models.Team
		if err := s.db.First(&team, *user.TeamID).Error; err == nil {
			row["teams"] = []map[string]any{{"id": team.ID, "name": team.Name}}
		}
	}
	return row
}

func userFirstQuery(query map[string][]string, key string) string {
	if len(query[key]) == 0 {
		return ""
	}
	return query[key][0]
}
func userCleanString(value any) string {
	text := fmt.Sprint(value)
	return strings.TrimSpace(text)
}
func userBoolValue(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	if s, ok := value.(string); ok {
		return s == "true"
	}
	return fallback
}
func userTagsJSON(value any) string {
	if text := userCleanString(value); text != "" && text != "[]" && !strings.HasPrefix(text, "[") {
		payload, _ := json.Marshal([]string{text})
		return string(payload)
	}
	if text := userCleanString(value); strings.HasPrefix(text, "[") {
		return text
	}
	if arr, ok := value.([]string); ok {
		payload, _ := json.Marshal(arr)
		return string(payload)
	}
	if arr, ok := value.([]any); ok {
		items := make([]string, 0, len(arr))
		for _, item := range arr {
			if text := userCleanString(item); text != "" {
				items = append(items, text)
			}
		}
		payload, _ := json.Marshal(items)
		return string(payload)
	}
	return "[]"
}
func userTagList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}
