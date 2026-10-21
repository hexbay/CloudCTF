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

type ChallengeService struct {
	db *gorm.DB
}

func NewChallengeService(db *gorm.DB) *ChallengeService {
	return &ChallengeService{db: db}
}

func (s *ChallengeService) GetByID(id any) (*models.Challenge, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.Challenge
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ChallengeService) CountByCompetition(competitionID uint) int64 {
	var total int64
	_ = s.db.Model(&models.Challenge{}).Where("competition_id = ?", competitionID).Count(&total).Error
	return total
}

func (s *ChallengeService) Count() int {
	var total int64
	_ = s.db.Model(&models.Challenge{}).Count(&total).Error
	return int(total)
}

func (s *ChallengeService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.Challenge{
		Name:              chCleanString(row["name"]),
		Description:       chCleanString(row["description"]),
		Category:          chValueOrDefault(row["category"], "misc"),
		Points:            chIntValue(row["points"], 0),
		Difficulty:        chIntValue(row["difficulty"], 1),
		IsActive:          chBoolValue(row["is_active"], true),
		IsPublic:          chBoolValue(row["is_public"], true),
		Usage:             chValueOrDefault(row["usage"], "PRACTICE"),
		Flag:              chCleanString(row["flag"]),
		IsDynamic:         chBoolValue(row["is_dynamic"], false),
		ContainerImage:    chCleanString(row["container_image"]),
		ContainerPort:     chIntValue(row["container_port"], 0),
		ContainerTimeout:  normalizeChallengeContainerTimeout(chIntValueDefault(row["container_timeout"], 3600)),
		HostID:            chOptionalUint(row["host_id"]),
		TemplateID:        chOptionalUint(row["template_id"]),
		CompetitionID:     chOptionalUint(row["competition_id"]),
		SourceChallengeID: chOptionalUint(row["source_challenge_id"]),
		LevelOneSolution:  chCleanString(row["level_one_solution"]),
		Hints:             chJSONString(row["hints"], "[]"),
		Files:             chJSONString(row["files"], "[]"),
		CreatedBy:         chOptionalUint(row["created_by"]),
	}
	if item.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return adminChallengeToRow(item), nil
}

func (s *ChallengeService) Update(id any, patch map[string]any) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var item models.Challenge
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["name"]; ok {
		item.Name = chCleanString(value)
	}
	if value, ok := patch["description"]; ok {
		item.Description = chCleanString(value)
	}
	if value, ok := patch["category"]; ok {
		item.Category = chCleanString(value)
	}
	if value, ok := patch["points"]; ok {
		item.Points = chIntValue(value, item.Points)
	}
	if value, ok := patch["difficulty"]; ok {
		item.Difficulty = chIntValue(value, item.Difficulty)
	}
	if value, ok := patch["is_active"]; ok {
		item.IsActive = chBoolValue(value, item.IsActive)
	}
	if value, ok := patch["is_public"]; ok {
		item.IsPublic = chBoolValue(value, item.IsPublic)
	}
	if value, ok := patch["usage"]; ok {
		item.Usage = chCleanString(value)
	}
	if value, ok := patch["flag"]; ok && chCleanString(value) != "" {
		item.Flag = chCleanString(value)
	}
	if value, ok := patch["is_dynamic"]; ok {
		item.IsDynamic = chBoolValue(value, item.IsDynamic)
	}
	if value, ok := patch["container_image"]; ok {
		item.ContainerImage = chCleanString(value)
	}
	if value, ok := patch["container_port"]; ok {
		item.ContainerPort = chIntValue(value, item.ContainerPort)
	}
	if value, ok := patch["container_timeout"]; ok {
		item.ContainerTimeout = normalizeChallengeContainerTimeout(chIntValue(value, item.ContainerTimeout))
	}
	if value, ok := patch["host_id"]; ok {
		item.HostID = chOptionalUint(value)
	}
	if value, ok := patch["template_id"]; ok {
		item.TemplateID = chOptionalUint(value)
	}
	if value, ok := patch["competition_id"]; ok {
		item.CompetitionID = chOptionalUint(value)
	}
	if value, ok := patch["source_challenge_id"]; ok {
		item.SourceChallengeID = chOptionalUint(value)
	}
	if value, ok := patch["level_one_solution"]; ok {
		item.LevelOneSolution = chCleanString(value)
	}
	if value, ok := patch["hints"]; ok {
		item.Hints = chJSONString(value, "[]")
	}
	if value, ok := patch["files"]; ok {
		item.Files = chJSONString(value, "[]")
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return adminChallengeToRow(item), true
}

func (s *ChallengeService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Challenge{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *ChallengeService) CloneToCompetition(challengeID, competitionID uint) (*models.Challenge, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var challenge models.Challenge
	if err := s.db.First(&challenge, challengeID).Error; err != nil {
		return nil, err
	}
	newUsage := challenge.Usage
	if newUsage == "" || newUsage == "PRACTICE" {
		newUsage = "COMPETITION"
	}
	cloned := models.Challenge{
		Name:              challenge.Name,
		Description:       challenge.Description,
		Category:          challenge.Category,
		Points:            challenge.Points,
		Difficulty:        challenge.Difficulty,
		IsActive:          challenge.IsActive,
		IsPublic:          challenge.IsPublic,
		Usage:             newUsage,
		Flag:              challenge.Flag,
		IsDynamic:         challenge.IsDynamic,
		ContainerImage:    challenge.ContainerImage,
		ContainerPort:     challenge.ContainerPort,
		ContainerTimeout:  challenge.ContainerTimeout,
		HostID:            challenge.HostID,
		TemplateID:        challenge.TemplateID,
		CompetitionID:     &competitionID,
		SourceChallengeID: &challenge.ID,
		LevelOneSolution:  challenge.LevelOneSolution,
		Hints:             challenge.Hints,
		Files:             challenge.Files,
		CreatedBy:         challenge.CreatedBy,
	}
	if err := s.db.Create(&cloned).Error; err != nil {
		return nil, err
	}
	return &cloned, nil
}

func (s *ChallengeService) BatchCloneToCompetition(challengeIDs []uint, competitionID uint) int {
	if s == nil || s.db == nil {
		return 0
	}
	added := 0
	for _, challengeID := range challengeIDs {
		if challengeID == 0 {
			continue
		}
		if _, err := s.CloneToCompetition(challengeID, competitionID); err == nil {
			added++
		}
	}
	return added
}

func (s *ChallengeService) DeleteInCompetition(challengeID, competitionID uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Where("id = ? AND competition_id = ?", challengeID, competitionID).Delete(&models.Challenge{})
	return result.Error == nil && result.RowsAffected > 0
}

func (s *ChallengeService) ToRow(item models.Challenge) map[string]any {
	return adminChallengeToRow(item)
}

func (s *ChallengeService) ExistsSolvedByUser(challengeID, userID uint) bool {
	var existing models.Submission
	return s.db.Where("challenge_id = ? AND user_id = ? AND is_correct = ?", challengeID, userID, true).First(&existing).Error == nil
}

func (s *ChallengeService) UpdateFlag(id any, flag string) bool {
	var item models.Challenge
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return false
	}
	return s.db.Model(&item).Update("flag", flag).Error == nil
}

func (s *ChallengeService) ExportRows(category string) []map[string]any {
	var items []models.Challenge
	db := s.db.Model(&models.Challenge{})
	if category != "" {
		db = db.Where("category = ?", category)
	}
	_ = db.Order("id DESC").Find(&items).Error
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		r := challengeToRow(item)
		r["flag"] = item.Flag
		rows = append(rows, r)
	}
	return rows
}

func (s *ChallengeService) UpdateFiles(id any, files []map[string]any) bool {
	_, ok := s.Update(id, map[string]any{"files": files})
	return ok
}

func (s *ChallengeService) AppendFile(id any, file map[string]any) error {
	item, err := s.GetByID(id)
	if err != nil {
		return err
	}
	files := chParseJSONArray(item.Files)
	files = append(files, file)
	if ok := s.UpdateFiles(id, files); !ok {
		return fmt.Errorf("failed to update challenge files")
	}
	return nil
}

func (s *ChallengeService) AddHint(id any, hint map[string]any) (map[string]any, bool) {
	item, err := s.GetByID(id)
	if err != nil {
		return nil, false
	}
	hints := chParseJSONArray(item.Hints)
	hints = append(hints, hint)
	if _, ok := s.Update(id, map[string]any{"hints": hints}); !ok {
		return nil, false
	}
	return hint, true
}

func (s *ChallengeService) RemoveHint(id any, hintID string) bool {
	item, err := s.GetByID(id)
	if err != nil {
		return false
	}
	hints := chParseJSONArray(item.Hints)
	filtered := make([]map[string]any, 0, len(hints))
	removed := false
	for _, hint := range hints {
		if fmt.Sprint(hint["id"]) == hintID {
			removed = true
			continue
		}
		filtered = append(filtered, hint)
	}
	if !removed {
		return false
	}
	_, ok := s.Update(id, map[string]any{"hints": filtered})
	return ok
}

func adminChallengeToRow(item models.Challenge) map[string]any {
	row := challengeToRow(item)
	row["flag"] = item.Flag
	return row
}

func challengeToRow(item models.Challenge) map[string]any {
	return map[string]any{
		"id":                     item.ID,
		"name":                   item.Name,
		"description":            item.Description,
		"category":               item.Category,
		"points":                 item.Points,
		"difficulty":             item.Difficulty,
		"is_active":              item.IsActive,
		"is_public":              item.IsPublic,
		"usage":                  item.Usage,
		"is_dynamic":             item.IsDynamic,
		"container_image":        item.ContainerImage,
		"container_port":         item.ContainerPort,
		"container_timeout":      item.ContainerTimeout,
		"host_id":                item.HostID,
		"template_id":            item.TemplateID,
		"competition_id":         item.CompetitionID,
		"source_challenge_id":    item.SourceChallengeID,
		"level_one_solution":     item.LevelOneSolution,
		"has_level_one_solution": chCleanString(item.LevelOneSolution) != "",
		"hints":                  chParseJSONArray(item.Hints),
		"files":                  chParseJSONArray(item.Files),
		"created_by":             item.CreatedBy,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
	}
}

func chCleanString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func chValueOrDefault(value any, fallback string) string {
	text := chCleanString(value)
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func chBoolValue(value any, fallback bool) bool {
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

func chIntValue(value any, fallback int) int {
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

func chIntValueDefault(value any, fallback int) int {
	parsed := chIntValue(value, 0)
	if parsed == 0 {
		return fallback
	}
	return parsed
}

func normalizeChallengeContainerTimeout(timeout int) int {
	if timeout <= 0 {
		return 3600
	}
	return timeout
}

func chOptionalUint(value any) *uint {
	switch typed := value.(type) {
	case nil:
		return nil
	case uint:
		if typed == 0 {
			return nil
		}
		copy := typed
		return &copy
	case uint64:
		if typed == 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case int:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case float64:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		parsed, err := strconv.ParseUint(text, 10, 64)
		if err == nil && parsed > 0 {
			copy := uint(parsed)
			return &copy
		}
	}
	return nil
}

func chJSONString(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" || text == "<nil>" {
			return fallback
		}
		return text
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return fallback
		}
		return string(payload)
	}
}

func chParseJSONArray(raw string) []map[string]any {
	if strings.TrimSpace(raw) == "" {
		return []map[string]any{}
	}
	items := []map[string]any{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []map[string]any{}
	}
	return items
}

var _ = time.Time{}
