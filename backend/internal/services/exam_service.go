package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"
	"gorm.io/gorm"
)

type ExamService struct{ db *gorm.DB }

func NewExamService(db *gorm.DB) *ExamService { return &ExamService{db: db} }

func (s *ExamService) DB() *gorm.DB { return s.db }

type ExamSubmissionListRow struct {
	ID            uint
	CompetitionID uint
	UserID        uint
	Username      string
	TeamID        *uint
	TeamName      string
	ExamID        uint
	Score         int
	TotalScore    int
	Correct       int
	Total         int
	Status        string
	ReviewNote    string
	SubmittedAt   time.Time
	GradedAt      *time.Time
}

func (s *ExamService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	var items []models.Exam
	db := s.db.Model(&models.Exam{})
	if title := exFirst(query, "title"); title != "" {
		db = db.Where("title LIKE ?", "%"+title+"%")
	}
	if exFirst(query, "include_competition") != "true" {
		db = db.Where("competition_id IS NULL")
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
		rows = append(rows, examToRow(item))
	}
	return rows, int(total), nil
}

func (s *ExamService) GetByID(id any) (*models.Exam, error) {
	var item models.Exam
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *ExamService) GetActiveByCompetition(competitionID string) (*models.Exam, error) {
	var exam models.Exam
	query := s.db.Where("is_active = ?", true)
	if competitionID != "" {
		query = query.Where("competition_id = ?", competitionID)
	} else {
		query = query.Where("competition_id IS NULL")
	}
	if err := query.Order("id DESC").First(&exam).Error; err != nil {
		return nil, err
	}
	return &exam, nil
}
func (s *ExamService) GetCompetitionExam(competitionID uint) map[string]any {
	var exam models.Exam
	if err := s.db.Where("competition_id = ?", competitionID).Order("id DESC").First(&exam).Error; err != nil {
		return nil
	}
	return examToRow(exam)
}

func (s *ExamService) HasCompetitionSubmission(competitionID, userID uint) bool {
	if s == nil || s.db == nil {
		return false
	}
	var total int64
	_ = s.db.Model(&models.ExamSubmission{}).
		Where("competition_id = ? AND user_id = ?", competitionID, userID).
		Count(&total).Error
	return total > 0
}

func (s *ExamService) CreateSubmission(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	item := models.ExamSubmission{
		CompetitionID: exUintFromAny(row["competition_id"]),
		UserID:        exUintFromAny(row["user_id"]),
		TeamID:        exOptionalUint(row["team_id"]),
		ExamID:        exUintFromAny(row["exam_id"]),
		Score:         exInt(row["score"], 0),
		TotalScore:    exInt(row["total_score"], 0),
		Correct:       exInt(row["correct"], 0),
		Total:         exInt(row["total"], 0),
		Status:        exClean(row["status"]),
		ReviewNote:    exClean(row["review_note"]),
		Answers:       exJSON(row["answers"]),
		SubmittedAt:   exTime(row["submitted_at"], time.Now()),
		GradedAt:      exOptionalTime(row["graded_at"]),
	}
	if item.Status == "" {
		item.Status = "graded"
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return examSubmissionToRow(item), nil
}

func (s *ExamService) UpdateSubmission(id any, patch map[string]any) (map[string]any, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("db unavailable")
	}
	var item models.ExamSubmission
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false, nil
	}
	if value, ok := patch["score"]; ok {
		item.Score = exInt(value, item.Score)
	}
	if value, ok := patch["total_score"]; ok {
		item.TotalScore = exInt(value, item.TotalScore)
	}
	if value, ok := patch["status"]; ok {
		item.Status = exClean(value)
	}
	if value, ok := patch["review_note"]; ok {
		item.ReviewNote = exClean(value)
	}
	if value, ok := patch["graded_at"]; ok {
		item.GradedAt = exOptionalTime(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false, err
	}
	return examSubmissionToRow(item), true, nil
}

func (s *ExamService) GetSubmissionByID(id any) (map[string]any, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("db unavailable")
	}
	var item models.ExamSubmission
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false, nil
	}
	row := examSubmissionToRow(item)
	var answers map[string]any
	_ = json.Unmarshal([]byte(item.Answers), &answers)
	if answers == nil {
		answers = map[string]any{}
	}
	row["answers"] = answers
	if exam, err := s.GetByID(item.ExamID); err == nil {
		context := parseExamJSONObject(exam.Context)
		row["exam"] = map[string]any{
			"id":          exam.ID,
			"title":       exam.Title,
			"total_score": exam.TotalScore,
			"questions":   extractExamQuestionsFromContext(context),
		}
	}
	return row, true, nil
}

func (s *ExamService) GetCompetitionSubmission(competitionID, userID uint) (map[string]any, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("db unavailable")
	}
	var item models.ExamSubmission
	if err := s.db.Where("competition_id = ? AND user_id = ?", competitionID, userID).Order("id DESC").First(&item).Error; err != nil {
		return nil, false, nil
	}
	return examSubmissionToRow(item), true, nil
}

func (s *ExamService) CompetitionSubmissionRows(competitionID string, current, pageSize int) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	db := s.db.Table("exam_submission es").
		Select("es.id, es.competition_id, es.user_id, u.username, es.team_id, t.name as team_name, es.exam_id, es.score, es.total_score, es.correct, es.total, es.status, es.review_note, es.submitted_at, es.graded_at").
		Joins("LEFT JOIN user u ON u.id = es.user_id").
		Joins("LEFT JOIN team t ON t.id = es.team_id").
		Where("es.competition_id = ?", competitionID)
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
	var items []ExamSubmissionListRow
	if err := db.Order("es.id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		percentage := 0.0
		if item.TotalScore > 0 {
			percentage = float64(item.Score) * 100 / float64(item.TotalScore)
		}
		rows = append(rows, map[string]any{
			"id":                 item.ID,
			"competition_id":     item.CompetitionID,
			"user_id":            item.UserID,
			"username":           item.Username,
			"team_id":            item.TeamID,
			"team_name":          item.TeamName,
			"exam_id":            item.ExamID,
			"score":              item.Score,
			"total_score":        item.Score,
			"max_possible_score": item.TotalScore,
			"correct_answers":    item.Correct,
			"total_questions":    item.Total,
			"percentage":         percentage,
			"status":             item.Status,
			"review_note":        item.ReviewNote,
			"is_graded":          item.Status == "graded",
			"submitted_at":       item.SubmittedAt,
			"graded_at":          item.GradedAt,
		})
	}
	return rows, int(total), nil
}
func (s *ExamService) Create(row map[string]any) (map[string]any, error) {
	item := models.Exam{CompetitionID: exOptionalUint(row["competition_id"]), Title: exClean(row["title"]), TotalScore: exInt(row["total_score"], 0), IsActive: exBool(row["is_active"], true), Context: exJSON(normalizeExamContext(row["context"])), CreatedBy: exOptionalUint(row["created_by"])}
	if item.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return examToRow(item), nil
}
func (s *ExamService) Update(id any, patch map[string]any) (map[string]any, bool) {
	var item models.Exam
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, false
	}
	if value, ok := patch["title"]; ok {
		item.Title = exClean(value)
	}
	if value, ok := patch["competition_id"]; ok {
		item.CompetitionID = exOptionalUint(value)
	}
	if value, ok := patch["total_score"]; ok {
		item.TotalScore = exInt(value, item.TotalScore)
	}
	if value, ok := patch["is_active"]; ok {
		item.IsActive = exBool(value, item.IsActive)
	}
	if value, ok := patch["context"]; ok {
		item.Context = exJSON(normalizeExamContext(value))
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false
	}
	return examToRow(item), true
}
func (s *ExamService) Delete(id any) bool {
	result := s.db.Delete(&models.Exam{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}
func (s *ExamService) DeleteCompetitionExam(competitionID uint) bool {
	result := s.db.Where("competition_id = ?", competitionID).Delete(&models.Exam{})
	return result.Error == nil && result.RowsAffected > 0
}
func (s *ExamService) CreateCompetitionExam(competitionID uint, payload map[string]any) (map[string]any, error) {
	_ = s.DeleteCompetitionExam(competitionID)
	questions, err := s.ResolveQuestions(payload)
	if err != nil {
		return nil, err
	}
	context := map[string]any{"questions": questions}
	if value := exInt(payload["time_limit"], 0); value > 0 {
		context["time_limit"] = value
	}
	title := exClean(payload["title"])
	if title == "" {
		title = "理论考试"
	}
	row := map[string]any{"competition_id": competitionID, "title": title, "total_score": exInt(payload["total_score"], sumExamQuestionPoints(questions)), "is_active": exBool(payload["is_active"], true), "context": context}
	return s.Create(row)
}
func (s *ExamService) QuickCreateCompetitionExam(competitionID uint, payload map[string]any, questionRows []map[string]any) (map[string]any, error) {
	targets := map[string]int{"text": exInt(payload["text_count"], 0), "judge": exInt(payload["judge_count"], 0), "single": exInt(payload["single_count"], 0), "multiple": exInt(payload["multiple_count"], 0)}
	selected := make([]uint, 0)
	used := map[string]int{}
	for _, row := range questionRows {
		kind := exClean(row["question_type"])
		limit, ok := targets[kind]
		if !ok || limit <= 0 || used[kind] >= limit {
			continue
		}
		selected = append(selected, exUintFromAny(row["id"]))
		used[kind]++
	}
	if len(selected) == 0 {
		return nil, badRequestError{message: "未找到满足条件的题目"}
	}
	return s.CreateCompetitionExam(competitionID, map[string]any{"title": payload["title"], "question_ids": selected, "total_score": len(selected) * 10, "is_active": true, "time_limit": exInt(payload["time_limit"], 3600)})
}
func (s *ExamService) BindCompetitionExamFromSource(competitionID, sourceExamID uint) (map[string]any, error) {
	source, err := s.GetByID(sourceExamID)
	if err != nil {
		return nil, badRequestError{message: "source exam not found"}
	}
	if source.CompetitionID != nil {
		return nil, badRequestError{message: "只能选择公共试卷"}
	}
	context := parseExamJSONObject(source.Context)
	payload := map[string]any{"title": source.Title, "total_score": source.TotalScore, "is_active": source.IsActive, "questions": extractExamQuestionsFromContext(context)}
	if value, ok := context["time_limit"]; ok {
		payload["time_limit"] = value
	}
	return s.CreateCompetitionExam(competitionID, payload)
}
func (s *ExamService) UpdateCompetitionExam(examID uint, payload map[string]any) (map[string]any, error) {
	patch := map[string]any{}
	if _, ok := payload["title"]; ok {
		patch["title"] = payload["title"]
	}
	if _, ok := payload["is_active"]; ok {
		patch["is_active"] = payload["is_active"]
	}
	if _, ok := payload["total_score"]; ok {
		patch["total_score"] = payload["total_score"]
	}
	if questions, err := s.ResolveQuestions(payload); err == nil && len(questions) > 0 {
		context := map[string]any{"questions": questions}
		if exam, getErr := s.GetByID(examID); getErr == nil {
			existing := parseExamJSONObject(exam.Context)
			if value, ok := existing["time_limit"]; ok {
				context["time_limit"] = value
			}
		}
		patch["context"] = context
		if _, ok := payload["total_score"]; !ok {
			patch["total_score"] = sumExamQuestionPoints(questions)
		}
	}
	row, ok := s.Update(examID, patch)
	if !ok {
		return nil, badRequestError{message: "exam not found"}
	}
	return row, nil
}
func (s *ExamService) ResolveQuestions(payload map[string]any) ([]map[string]any, error) {
	if rawQuestions, ok := payload["questions"]; ok {
		context := map[string]any{"questions": rawQuestions}
		questions := extractExamQuestionsFromContext(context)
		if len(questions) == 0 {
			return nil, badRequestError{message: "questions required"}
		}
		return questions, nil
	}
	questionIDs := extractQuestionIDs(payload)
	if len(questionIDs) == 0 {
		return nil, badRequestError{message: "question_ids required"}
	}
	var items []models.TheoryQuestion
	if err := s.db.Where("id IN ?", questionIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	lookup := make(map[uint]models.TheoryQuestion, len(items))
	for _, item := range items {
		lookup[item.ID] = item
	}
	questions := make([]map[string]any, 0, len(questionIDs))
	for _, id := range questionIDs {
		item, ok := lookup[id]
		if !ok {
			continue
		}
		questions = append(questions, theoryQuestionToRow(item))
	}
	if len(questions) == 0 {
		return nil, badRequestError{message: "questions required"}
	}
	return questions, nil
}
func examToRow(item models.Exam) map[string]any {
	var context map[string]any
	_ = json.Unmarshal([]byte(item.Context), &context)
	if context == nil {
		context = map[string]any{}
	}
	return map[string]any{"id": item.ID, "competition_id": item.CompetitionID, "title": item.Title, "total_score": item.TotalScore, "is_active": item.IsActive, "context": context, "created_by": item.CreatedBy, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func (s *ExamService) ToSubmissionRow(item models.ExamSubmission) map[string]any {
	return examSubmissionToRow(item)
}

func examSubmissionToRow(item models.ExamSubmission) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"competition_id": item.CompetitionID,
		"user_id":        item.UserID,
		"team_id":        item.TeamID,
		"exam_id":        item.ExamID,
		"score":          item.Score,
		"total_score":    item.TotalScore,
		"correct":        item.Correct,
		"total":          item.Total,
		"status":         item.Status,
		"review_note":    item.ReviewNote,
		"submitted_at":   item.SubmittedAt,
		"graded_at":      item.GradedAt,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}
func exFirst(query map[string][]string, key string) string {
	if len(query[key]) == 0 {
		return ""
	}
	return query[key][0]
}
func exClean(value any) string {
	text := fmt.Sprint(value)
	if text == "<nil>" {
		return ""
	}
	return strings.TrimSpace(text)
}
func exInt(value any, fallback int) int {
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
func exTime(value any, fallback time.Time) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		if typed == "" {
			return fallback
		}
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed
		}
	}
	return fallback
}
func exOptionalTime(value any) *time.Time {
	switch typed := value.(type) {
	case time.Time:
		return &typed
	case *time.Time:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return &parsed
		}
	}
	return nil
}
func exOptionalUint(value any) *uint {
	var parsed uint
	_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
	if parsed == 0 {
		return nil
	}
	return &parsed
}
func exBool(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	if s, ok := value.(string); ok {
		return s == "true"
	}
	return fallback
}
func exJSON(value any) string {
	if text := exClean(value); strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return text
	}
	payload, _ := json.Marshal(value)
	return string(payload)
}
func normalizeExamContext(value any) map[string]any {
	context := map[string]any{}
	_ = json.Unmarshal([]byte(exJSON(value)), &context)
	questions := extractExamQuestionsFromContext(context)
	context["questions"] = questions
	if _, ok := context["question_ids"]; !ok {
		ids := make([]uint, 0, len(questions))
		for _, question := range questions {
			if id := exUintFromAny(question["id"]); id > 0 {
				ids = append(ids, id)
			}
		}
		context["question_ids"] = ids
	}
	return context
}

func extractExamQuestionsFromContext(context map[string]any) []map[string]any {
	rawQuestions, ok := context["questions"]
	if !ok {
		return []map[string]any{}
	}
	switch typed := rawQuestions.(type) {
	case []map[string]any:
		return typed
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows
	default:
		return []map[string]any{}
	}
}

func exUintFromAny(value any) uint {
	var parsed uint
	_, _ = fmt.Sscan(fmt.Sprint(value), &parsed)
	return parsed
}

func extractQuestionIDs(payload map[string]any) []uint {
	raw, ok := payload["question_ids"]
	if !ok {
		raw = payload["ids"]
	}
	switch typed := raw.(type) {
	case []any:
		ids := make([]uint, 0, len(typed))
		for _, item := range typed {
			if id := exUintFromAny(item); id > 0 {
				ids = append(ids, id)
			}
		}
		return ids
	case []uint:
		return typed
	default:
		return []uint{}
	}
}

func sumExamQuestionPoints(questions []map[string]any) int {
	total := 0
	for _, question := range questions {
		total += exInt(question["points"], 0)
	}
	return total
}

func parseExamJSONObject(raw string) map[string]any {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload
}

func theoryQuestionToRow(item models.TheoryQuestion) map[string]any {
	return map[string]any{"id": item.ID, "question": item.Question, "points": item.Points, "category": item.Category, "difficulty": item.Difficulty, "question_type": item.QuestionType, "context": item.ContextMap(), "created_by": item.CreatedBy, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

type badRequestError struct{ message string }

func (e badRequestError) Error() string { return e.message }
