package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type TheorySubmissionService struct {
	db *gorm.DB
}

func NewTheorySubmissionService(db *gorm.DB) *TheorySubmissionService {
	return &TheorySubmissionService{db: db}
}

func (s *TheorySubmissionService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	db := s.db.Model(&models.TheorySubmission{})
	if keyword := theorySubmissionFirstQuery(query, "keyword"); keyword != "" {
		db = db.Where("CAST(id AS CHAR) LIKE ? OR CAST(user_id AS CHAR) LIKE ? OR CAST(competition_id AS CHAR) LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if competitionID := theorySubmissionFirstQuery(query, "competition_id"); competitionID != "" {
		db = db.Where("competition_id = ?", competitionID)
	}
	if userID := theorySubmissionFirstQuery(query, "user_id"); userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if examID := theorySubmissionFirstQuery(query, "exam_id"); examID != "" {
		db = db.Where("exam_id = ?", examID)
	}
	if status := theorySubmissionFirstQuery(query, "status"); status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	var items []models.TheorySubmission
	if err := db.Order("id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, theorySubmissionToRow(item))
	}
	return rows, int(total), nil
}

func (s *TheorySubmissionService) GetByID(id any) (*models.TheorySubmission, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var item models.TheorySubmission
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *TheorySubmissionService) ListByCompetition(competitionID uint) ([]map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	var items []models.TheorySubmission
	if err := s.db.Where("competition_id = ?", competitionID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, theorySubmissionToRow(item))
	}
	return rows, nil
}

func (s *TheorySubmissionService) ExportRows(competitionID *uint) ([]map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	db := s.db.Model(&models.TheorySubmission{})
	if competitionID != nil {
		db = db.Where("competition_id = ?", *competitionID)
	}
	var items []models.TheorySubmission
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, theorySubmissionToRow(item))
	}
	return rows, nil
}

func (s *TheorySubmissionService) ToRow(item models.TheorySubmission) map[string]any {
	return theorySubmissionToRow(item)
}

func theorySubmissionToRow(item models.TheorySubmission) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"competition_id": item.CompetitionID,
		"user_id":        item.UserID,
		"team_id":        item.TeamID,
		"exam_id":        item.ExamID,
		"score":          item.Score,
		"status":         item.Status,
		"review_note":    item.ReviewNote,
		"answers":        theorySubmissionAnswers(item.Answers),
		"submitted_at":   item.SubmittedAt,
		"graded_at":      item.GradedAt,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func theorySubmissionAnswers(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	result := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &result); err == nil {
		return result
	}
	return map[string]any{"raw": raw}
}

func theorySubmissionFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

var _ = time.RFC3339
