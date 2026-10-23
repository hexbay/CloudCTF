package services

import (
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubmissionService struct {
	db *gorm.DB
}

func NewSubmissionService(db *gorm.DB) *SubmissionService {
	return &SubmissionService{db: db}
}

func (s *SubmissionService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Submission
	db := s.db.Model(&models.Submission{})
	if challengeID := subFirstQuery(query, "challenge_id"); challengeID != "" {
		db = db.Where("challenge_id = ?", challengeID)
	}
	if userID := subFirstQuery(query, "user_id"); userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if competitionID := subFirstQuery(query, "competition_id"); competitionID != "" {
		db = db.Where("competition_id = ?", competitionID)
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
		rows = append(rows, submissionToRow(item))
	}
	return rows, int(total), nil
}

func (s *SubmissionService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.Submission{
		ChallengeID:   subUintValue(row["challenge_id"]),
		UserID:        subUintValue(row["user_id"]),
		TeamID:        subOptionalUint(row["team_id"]),
		CompetitionID: subOptionalUint(row["competition_id"]),
		Flag:          fmt.Sprint(row["flag"]),
		IsCorrect:     subBoolValue(row["is_correct"], false),
		Points:        subIntValue(row["points"], 0),
		IPAddress:     fmt.Sprint(row["ip_address"]),
		SubmittedAt:   time.Now(),
	}
	if item.ChallengeID == 0 || item.UserID == 0 || item.Flag == "" {
		return nil, fmt.Errorf("challenge_id, user_id, and flag are required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return submissionToRow(item), nil
}

func (s *SubmissionService) Delete(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.Submission{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *SubmissionService) ExistsSolvedByUserWithCompetition(challengeID, userID uint, competitionID *uint) (*models.Submission, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var existing models.Submission
	query := s.db.Where("challenge_id = ? AND user_id = ? AND is_correct = ?", challengeID, userID, true)
	if competitionID != nil {
		query = query.Where("competition_id = ?", *competitionID)
	} else {
		query = query.Where("competition_id IS NULL")
	}
	if query.Order("submitted_at asc").First(&existing).Error != nil {
		return nil, false
	}
	return &existing, true
}

func (s *SubmissionService) FirstBloodExists(challengeID uint, competitionID *uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	var total int64
	query := s.db.Model(&models.Submission{}).Where("challenge_id = ? AND is_correct = ?", challengeID, true)
	if competitionID != nil {
		query = query.Where("competition_id = ?", *competitionID)
	} else {
		query = query.Where("competition_id IS NULL")
	}
	_ = query.Count(&total).Error
	return total > 0
}

func (s *SubmissionService) PracticeHintsForUser(challenge models.Challenge, userID uint) []map[string]any {
	rawHints := challenge.HintList()
	var purchases []models.HintPurchase
	_ = s.db.Where("user_id = ? AND challenge_id = ?", userID, challenge.ID).Find(&purchases).Error
	purchased := make(map[int]bool, len(purchases))
	for _, row := range purchases {
		purchased[row.HintIndex] = true
	}
	result := make([]map[string]any, 0, len(rawHints))
	for index, hint := range rawHints {
		result = append(result, map[string]any{
			"index":     index,
			"content":   fmt.Sprint(hint["content"]),
			"cost":      subIntValue(hint["cost"], 0),
			"purchased": purchased[index],
		})
	}
	return result
}

func (s *SubmissionService) UnlockHint(userID, challengeID uint, hintIndex, cost int) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	var purchase models.HintPurchase
	if s.db.Where("user_id = ? AND challenge_id = ? AND hint_index = ?", userID, challengeID, hintIndex).First(&purchase).Error == nil {
		return nil
	}
	purchase = models.HintPurchase{
		UserID:      userID,
		ChallengeID: challengeID,
		HintIndex:   hintIndex,
		Cost:        cost,
	}
	return s.db.Create(&purchase).Error
}

func (s *SubmissionService) PracticeChallengeBloods(challengeID uint, limit int) []map[string]any {
	var submissions []models.Submission
	_ = s.db.Where("challenge_id = ? AND is_correct = ? AND competition_id IS NULL", challengeID, true).
		Order("submitted_at asc").
		Find(&submissions).Error
	result := make([]map[string]any, 0, limit)
	seen := map[uint]bool{}
	for _, submission := range submissions {
		if seen[submission.UserID] {
			continue
		}
		seen[submission.UserID] = true
		username := fmt.Sprintf("user_%d", submission.UserID)
		var user models.User
		if err := s.db.First(&user, submission.UserID).Error; err == nil {
			username = user.Username
		}
		result = append(result, map[string]any{
			"rank":         len(result) + 1,
			"user_id":      submission.UserID,
			"username":     username,
			"submitted_at": submission.SubmittedAt,
		})
		if len(result) >= limit {
			break
		}
	}
	return result
}

func (s *SubmissionService) PracticeChallengeSolvedCount(challengeID uint) int64 {
	var total int64
	_ = s.db.Model(&models.Submission{}).
		Where("challenge_id = ? AND is_correct = ? AND competition_id IS NULL", challengeID, true).
		Distinct("user_id").
		Count(&total).Error
	return total
}

func (s *SubmissionService) PracticeChallengeRows(current, pageSize int, category string, difficulty int, competitionID string, userID uint) ([]map[string]any, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	var items []models.Challenge
	db := s.db.Model(&models.Challenge{}).Where("is_active = ? AND is_public = ?", true, true)
	if competitionID != "" {
		db = db.Where("competition_id = ?", competitionID)
		db = db.Where(clause.IN{Column: clause.Column{Name: "usage"}, Values: []any{"", "COMPETITION", "BOTH", "PRACTICE"}})
	} else {
		db = db.Where("competition_id IS NULL")
		db = db.Where(clause.IN{Column: clause.Column{Name: "usage"}, Values: []any{"", "PRACTICE", "BOTH"}})
	}
	if category != "" {
		db = db.Where("LOWER(category) = LOWER(?)", category)
	}
	if difficulty > 0 {
		db = db.Where("difficulty = ?", difficulty)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("points asc, id asc").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"id":                     item.ID,
			"name":                   item.Name,
			"description":            item.Description,
			"category":               item.Category,
			"points":                 item.Points,
			"difficulty":             item.Difficulty,
			"is_active":              item.IsActive,
			"is_public":              item.IsPublic,
			"usage":                  item.Usage,
			"container_image":        item.ContainerImage,
			"container_port":         item.ContainerPort,
			"container_timeout":      item.ContainerTimeout,
			"host_id":                item.HostID,
			"template_id":            item.TemplateID,
			"competition_id":         item.CompetitionID,
			"source_challenge_id":    item.SourceChallengeID,
			"has_level_one_solution": strings.TrimSpace(item.LevelOneSolution) != "",
			"created_by":             item.CreatedBy,
			"created_at":             item.CreatedAt,
			"updated_at":             item.UpdatedAt,
			"is_solved":              false,
		}
		if userID != 0 {
			var solved int64
			solvedQuery := s.db.Model(&models.Submission{}).
				Where("challenge_id = ? AND user_id = ? AND is_correct = ?", item.ID, userID, true)
			if competitionID != "" {
				solvedQuery = solvedQuery.Where("competition_id = ?", competitionID)
			} else {
				solvedQuery = solvedQuery.Where("competition_id IS NULL")
			}
			_ = solvedQuery.Count(&solved).Error
			row["is_solved"] = solved > 0
		}
		rows = append(rows, row)
	}
	return rows, total, nil
}

func (s *SubmissionService) SubmissionTrend(days int) []map[string]any {
	type row struct {
		Date    string
		Total   int64
		Correct int64
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("DATE(submitted_at) as date, COUNT(*) as total, SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) as correct").
		Where("submitted_at >= ?", time.Now().AddDate(0, 0, -days)).
		Group("DATE(submitted_at)").Order("date ASC").Find(&results).Error
	data := make([]map[string]any, 0, len(results))
	for _, r := range results {
		data = append(data, map[string]any{"date": r.Date, "total": r.Total, "correct": r.Correct})
	}
	return data
}

func (s *SubmissionService) CountByCompetition(competitionID any, correctOnly bool) int64 {
	var total int64
	db := s.db.Model(&models.Submission{})
	if competitionID == nil {
		db = db.Where("competition_id IS NULL")
	} else {
		db = db.Where("competition_id = ?", competitionID)
	}
	if correctOnly {
		db = db.Where("is_correct = ?", true)
	}
	_ = db.Count(&total).Error
	return total
}

func (s *SubmissionService) CountSince(since time.Time) int64 {
	var total int64
	_ = s.db.Model(&models.Submission{}).Where("submitted_at >= ?", since).Count(&total).Error
	return total
}

func (s *SubmissionService) CompetitionRows(competitionID string, current, pageSize int) ([]map[string]any, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	var items []models.Submission
	db := s.db.Model(&models.Submission{}).Where("competition_id = ?", competitionID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("submitted_at DESC, id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, submissionToRow(item))
	}
	return rows, total, nil
}

func (s *SubmissionService) HotChallenges(limit int) []map[string]any {
	type row struct {
		ChallengeID uint
		SubmitCount int64
		SolvedCount int64
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("challenge_id, COUNT(*) as submit_count, SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) as solved_count").
		Group("challenge_id").Order("submit_count DESC").Limit(limit).Find(&results).Error
	data := make([]map[string]any, 0, len(results))
	for _, r := range results {
		var ch models.Challenge
		solveRate := float64(0)
		if r.SubmitCount > 0 {
			solveRate = float64(r.SolvedCount) / float64(r.SubmitCount) * 100
		}
		chName := "Unknown"
		if err := s.db.First(&ch, r.ChallengeID).Error; err == nil {
			chName = ch.Name
		}
		data = append(data, map[string]any{
			"id":               r.ChallengeID,
			"name":             chName,
			"category":         ch.Category,
			"points":           ch.Points,
			"difficulty":       ch.Difficulty,
			"submission_count": r.SubmitCount,
			"solve_count":      r.SolvedCount,
			"solve_rate":       solveRate,
		})
	}
	return data
}

func (s *SubmissionService) ActiveUsers(limit int) []map[string]any {
	type row struct {
		UserID      uint
		SubmitCount int64
		SolvedCount int64
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("user_id, COUNT(*) as submit_count, SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) as solved_count").
		Group("user_id").Order("submit_count DESC").Limit(limit).Find(&results).Error
	data := make([]map[string]any, 0, len(results))
	for _, r := range results {
		var user models.User
		username := "Unknown"
		if err := s.db.First(&user, r.UserID).Error; err == nil {
			username = user.Username
		}
		data = append(data, map[string]any{
			"id":               r.UserID,
			"username":         username,
			"email":            "",
			"submission_count": r.SubmitCount,
			"solve_count":      r.SolvedCount,
		})
	}
	return data
}

func (s *SubmissionService) SolveStatus() []map[string]any {
	type row struct {
		TeamID      *uint
		UserID      *uint
		ChallengeID uint
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("team_id, user_id, challenge_id").
		Where("is_correct = ?", true).
		Group("team_id, user_id, challenge_id").
		Find(&results).Error
	data := make([]map[string]any, 0, len(results))
	for _, r := range results {
		data = append(data, map[string]any{
			"team_id": r.TeamID, "user_id": r.UserID, "challenge_id": r.ChallengeID, "solved": true,
		})
	}
	return data
}

func submissionToRow(item models.Submission) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"challenge_id":   item.ChallengeID,
		"user_id":        item.UserID,
		"team_id":        item.TeamID,
		"competition_id": item.CompetitionID,
		"is_correct":     item.IsCorrect,
		"points":         item.Points,
		"ip_address":     item.IPAddress,
		"submitted_at":   item.SubmittedAt,
		"created_at":     item.CreatedAt,
	}
}

func subFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func subUintValue(value any) uint {
	var parsed uint
	_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
	return parsed
}

func subOptionalUint(value any) *uint {
	switch typed := value.(type) {
	case *uint:
		if typed == nil || *typed == 0 {
			return nil
		}
		parsed := *typed
		return &parsed
	}
	parsed := subUintValue(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func subBoolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	}
	return fallback
}

func subIntValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		var parsed int
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	}
	return fallback
}
