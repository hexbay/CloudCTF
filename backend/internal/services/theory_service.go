package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"cloudctf/backend/internal/models"
	"gorm.io/gorm"
)

type TheoryService struct{ db *gorm.DB }

func NewTheoryService(db *gorm.DB) *TheoryService { return &TheoryService{db: db} }

func (s *TheoryService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	var items []models.TheoryQuestion
	db := s.db.Model(&models.TheoryQuestion{})
	if keyword := thFirst(query, "keyword"); keyword != "" {
		db = db.Where("question LIKE ?", "%"+keyword+"%")
	}
	if category := thFirst(query, "category"); category != "" {
		db = db.Where("category = ?", category)
	}
	if questionType := thFirst(query, "question_type"); questionType != "" {
		db = db.Where("question_type = ?", questionType)
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
		rows = append(rows, theoryToRow(item))
	}
	return rows, int(total), nil
}

func (s *TheoryService) GetByID(id any) (*models.TheoryQuestion, error) {
	var item models.TheoryQuestion
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *TheoryService) Create(row map[string]any) (map[string]any, error) {
	item := models.TheoryQuestion{
		Question:     thClean(row["question"]),
		Points:       thInt(row["points"], 0),
		Category:     thClean(row["category"]),
		Difficulty:   thInt(row["difficulty"], 1),
		QuestionType: thDefault(row["question_type"], "text"),
		Context:      thJSON(row["context"]),
		CreatedBy:    thOptionalUint(row["created_by"]),
	}
	if item.Question == "" {
		return nil, fmt.Errorf("question is required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return theoryToRow(item), nil
}

func (s *TheoryService) Update(id any, patch map[string]any) (map[string]any, bool) {
	var item models.TheoryQuestion
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["question"]; ok {
		item.Question = thClean(value)
	}
	if value, ok := patch["points"]; ok {
		item.Points = thInt(value, item.Points)
	}
	if value, ok := patch["category"]; ok {
		item.Category = thClean(value)
	}
	if value, ok := patch["difficulty"]; ok {
		item.Difficulty = thInt(value, item.Difficulty)
	}
	if value, ok := patch["question_type"]; ok {
		item.QuestionType = thClean(value)
	}
	if value, ok := patch["context"]; ok {
		item.Context = thJSON(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return theoryToRow(item), true
}

func (s *TheoryService) Delete(id any) bool {
	result := s.db.Delete(&models.TheoryQuestion{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *TheoryService) Count() int {
	var total int64
	_ = s.db.Model(&models.TheoryQuestion{}).Count(&total).Error
	return int(total)
}

func (s *TheoryService) ListCompetitionQuestions(competitionID uint, current, pageSize int) ([]map[string]any, int) {
	var bindings []models.CompetitionTheoryQuestion
	query := s.db.Where("competition_id = ?", competitionID).Order("id DESC")
	var total int64
	_ = query.Model(&models.CompetitionTheoryQuestion{}).Count(&total).Error
	if pageSize <= 0 {
		pageSize = 100
	}
	if current <= 0 {
		current = 1
	}
	_ = query.Offset((current - 1) * pageSize).Limit(pageSize).Find(&bindings).Error
	rows := make([]map[string]any, 0, len(bindings))
	for _, binding := range bindings {
		var question models.TheoryQuestion
		if s.db.First(&question, binding.QuestionID).Error == nil {
			row := theoryToRow(question)
			row["competition_id"] = competitionID
			rows = append(rows, row)
		}
	}
	return rows, int(total)
}

func (s *TheoryService) AddToCompetition(competitionID, questionID uint) error {
	var question models.TheoryQuestion
	if err := s.db.First(&question, questionID).Error; err != nil {
		return err
	}
	binding := models.CompetitionTheoryQuestion{CompetitionID: competitionID, QuestionID: questionID}
	return s.db.Where("competition_id = ? AND question_id = ?", competitionID, questionID).FirstOrCreate(&binding).Error
}

func (s *TheoryService) RemoveFromCompetition(competitionID, questionID uint) {
	_ = s.db.Where("competition_id = ? AND question_id = ?", competitionID, questionID).Delete(&models.CompetitionTheoryQuestion{}).Error
}

func (s *TheoryService) ExportRows() []map[string]any {
	var items []models.TheoryQuestion
	_ = s.db.Order("id DESC").Find(&items).Error
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, theoryToRow(item))
	}
	return rows
}

func theoryToRow(item models.TheoryQuestion) map[string]any {
	return map[string]any{"id": item.ID, "question": item.Question, "points": item.Points, "category": item.Category, "difficulty": item.Difficulty, "question_type": item.QuestionType, "context": item.ContextMap(), "created_by": item.CreatedBy, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}
func thFirst(query map[string][]string, key string) string {
	if len(query[key]) == 0 {
		return ""
	}
	return query[key][0]
}
func thClean(value any) string {
	text := fmt.Sprint(value)
	return strings.TrimSpace(text)
}
func thDefault(value any, fallback string) string {
	if text := thClean(value); text != "" {
		return text
	}
	return fallback
}
func thInt(value any, fallback int) int {
	var parsed int
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	}
	return fallback
}
func thOptionalUint(value any) *uint {
	var parsed uint
	_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
	if parsed == 0 {
		return nil
	}
	return &parsed
}
func thJSON(value any) string {
	if text := thClean(value); strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return text
	}
	payload, _ := json.Marshal(value)
	return string(payload)
}
