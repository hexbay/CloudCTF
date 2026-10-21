package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type TeamService struct {
	db *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{db: db}
}

func (s *TeamService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var teams []models.Team
	db := s.db.Preload("Owner").Model(&models.Team{})
	if name := teamFirstQuery(query, "name"); name != "" {
		db = db.Where("team.name LIKE ?", "%"+name+"%")
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
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("team.id DESC").Find(&teams).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(teams))
	for _, team := range teams {
		rows = append(rows, s.ToRow(team))
	}
	return rows, int(total), nil
}

func (s *TeamService) GetByID(id any) (*models.Team, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var team models.Team
	if err := s.db.Preload("Owner").First(&team, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

func (s *TeamService) GetByInvitationCode(code string) (*models.Team, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var team models.Team
	if err := s.db.Preload("Owner").Where("invitation_code = ?", strings.TrimSpace(code)).First(&team).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

func (s *TeamService) ActiveMemberCount(teamID uint) int64 {
	var total int64
	_ = s.db.Model(&models.User{}).Where("team_id = ? AND is_active = ?", teamID, true).Count(&total).Error
	return total
}

func (s *TeamService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	ownerID := teamUintValue(row["owner_id"])
	if ownerID == 0 {
		var player models.User
		if err := s.db.Where("username = ?", "player1").First(&player).Error; err == nil {
			ownerID = player.ID
		}
	}
	if ownerID == 0 {
		return nil, fmt.Errorf("owner_id is required")
	}
	invitationCode := teamCleanString(row["invitation_code"])
	if invitationCode == "" {
		invitationCode = fmt.Sprintf("TEAM-%d", time.Now().UnixNano())
	}
	team := models.Team{
		Name:           teamCleanString(row["name"]),
		Description:    teamCleanString(row["description"]),
		OwnerID:        ownerID,
		InvitationCode: invitationCode,
	}
	if team.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := s.db.Create(&team).Error; err != nil {
		return nil, err
	}
	_ = s.db.Model(&models.User{}).Where("id = ?", ownerID).Update("team_id", team.ID).Error
	_ = s.db.Preload("Owner").First(&team, team.ID).Error
	return s.ToRow(team), nil
}

func (s *TeamService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var team models.Team
	if err := s.db.Preload("Owner").First(&team, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		team.Name = teamCleanString(value)
	}
	if value, ok := patch["description"]; ok {
		team.Description = teamCleanString(value)
	}
	if value, ok := patch["owner_id"]; ok && teamUintValue(value) != 0 {
		team.OwnerID = teamUintValue(value)
	}
	if value, ok := patch["is_blacklisted"]; ok {
		team.IsBlacklisted = teamBoolValue(value, team.IsBlacklisted)
	}
	if value, ok := patch["blacklist_reason"]; ok {
		team.BlacklistReason = teamCleanString(value)
	}
	if err := s.db.Save(&team).Error; err != nil {
		return nil, false
	}
	_ = s.db.Preload("Owner").First(&team, team.ID).Error
	return s.ToRow(team), true
}

func (s *TeamService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Team{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *TeamService) Members(teamID uint) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	var users []models.User
	_ = s.db.Preload("Role").Where("team_id = ? AND is_active = ?", teamID, true).Find(&users).Error
	rows := make([]map[string]any, 0, len(users))
	for _, user := range users {
		row := map[string]any{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"full_name":    user.FullName,
			"is_active":    user.IsActive,
			"is_superuser": user.IsSuperuser,
			"role_id":      user.RoleID,
			"created_at":   user.CreatedAt,
			"updated_at":   user.UpdatedAt,
			"is_owner":     false,
		}
		rows = append(rows, row)
	}
	return rows
}

func (s *TeamService) GetActiveTeamByUserID(userID uint) (*models.Team, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.TeamID == nil || *user.TeamID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.GetByID(*user.TeamID)
}

func (s *TeamService) MemberHistory(teamID uint) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	var members []models.TeamMember
	_ = s.db.Where("team_id = ?", teamID).Order("joined_at DESC").Find(&members).Error
	rows := make([]map[string]any, 0, len(members))
	for _, member := range members {
		var user models.User
		_ = s.db.First(&user, member.UserID).Error
		rows = append(rows, map[string]any{
			"id":        member.ID,
			"team_id":   member.TeamID,
			"user_id":   member.UserID,
			"username":  user.Username,
			"email":     user.Email,
			"joined_at": member.JoinedAt,
			"left_at":   member.LeftAt,
			"is_active": member.IsActive,
			"reason":    member.Reason,
		})
	}
	return rows
}

func (s *TeamService) UserHistory(userID uint) []map[string]any {
	if s == nil || s.db == nil {
		return []map[string]any{}
	}
	var members []models.TeamMember
	_ = s.db.Where("user_id = ?", userID).Order("joined_at DESC").Find(&members).Error
	rows := make([]map[string]any, 0, len(members))
	for _, member := range members {
		var team models.Team
		_ = s.db.First(&team, member.TeamID).Error
		rows = append(rows, map[string]any{
			"id":        member.ID,
			"team_id":   member.TeamID,
			"team_name": team.Name,
			"user_id":   member.UserID,
			"joined_at": member.JoinedAt,
			"left_at":   member.LeftAt,
			"is_active": member.IsActive,
			"reason":    member.Reason,
		})
	}
	return rows
}

func (s *TeamService) Count() int {
	var total int64
	_ = s.db.Model(&models.Team{}).Count(&total).Error
	return int(total)
}

func (s *TeamService) CountActiveMembers() int64 {
	var total int64
	_ = s.db.Model(&models.User{}).Where("team_id IS NOT NULL AND is_active = ?", true).Count(&total).Error
	return total
}

func (s *TeamService) NameExists(name string) bool {
	var total int64
	_ = s.db.Model(&models.Team{}).Where("name = ?", strings.TrimSpace(name)).Count(&total).Error
	return total > 0
}

func (s *TeamService) EnsureMember(teamID, userID uint, reason string) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("team_id", teamID).Error
}

func (s *TeamService) HasActiveMember(teamID, userID uint) bool {
	var user models.User
	return s.db.Where("id = ? AND team_id = ? AND is_active = ?", userID, teamID, true).First(&user).Error == nil
}

func (s *TeamService) HasActiveMembership(userID uint) bool {
	var user models.User
	return s.db.Where("id = ? AND team_id IS NOT NULL AND is_active = ?", userID, true).First(&user).Error == nil
}

func (s *TeamService) RemoveActiveMembership(userID uint, reason string) error {
	return s.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("team_id", nil).Error
}

func (s *TeamService) ExportRows() []map[string]any {
	var teams []models.Team
	_ = s.db.Preload("Owner").Order("id DESC").Find(&teams).Error
	rows := make([]map[string]any, 0, len(teams))
	for _, team := range teams {
		var memberCount int64
		_ = s.db.Model(&models.User{}).Where("team_id = ? AND is_active = ?", team.ID, true).Count(&memberCount).Error
		rows = append(rows, map[string]any{
			"name":         team.Name,
			"owner":        team.Owner.Username,
			"member_count": memberCount,
			"created_at":   team.CreatedAt,
		})
	}
	return rows
}

func (s *TeamService) ToRow(team models.Team) map[string]any {
	var memberCount int64
	_ = s.db.Model(&models.User{}).Where("team_id = ? AND is_active = ?", team.ID, true).Count(&memberCount).Error
	owner := map[string]any{
		"id":           team.Owner.ID,
		"username":     team.Owner.Username,
		"email":        team.Owner.Email,
		"full_name":    team.Owner.FullName,
		"is_active":    team.Owner.IsActive,
		"is_superuser": team.Owner.IsSuperuser,
		"role_id":      team.Owner.RoleID,
		"created_at":   team.Owner.CreatedAt,
		"updated_at":   team.Owner.UpdatedAt,
	}
	return map[string]any{
		"id":               team.ID,
		"name":             team.Name,
		"description":      team.Description,
		"owner_id":         team.OwnerID,
		"owner":            owner,
		"invitation_code":  team.InvitationCode,
		"member_count":     memberCount,
		"is_blacklisted":   team.IsBlacklisted,
		"blacklist_reason": team.BlacklistReason,
		"blacklisted_at":   team.BlacklistedAt,
		"blacklisted_by":   team.BlacklistedBy,
		"created_at":       team.CreatedAt,
		"updated_at":       team.UpdatedAt,
	}
}

func teamFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func teamCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func teamBoolValue(value any, fallback bool) bool {
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

func teamUintValue(value any) uint {
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
