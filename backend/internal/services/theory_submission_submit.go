package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type SubmitExamInput struct {
	UserID        uint
	CompetitionID uint
	ExamID        uint
	Answers       map[string]any
}

type SubmitExamResult struct {
	Score      int `json:"score"`
	TotalScore int `json:"total_score"`
	Correct    int `json:"correct"`
	Total      int `json:"total"`
}

func (s *TheorySubmissionService) SubmitExam(input SubmitExamInput) (SubmitExamResult, error) {
	if s == nil || s.db == nil {
		return SubmitExamResult{}, fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.CompetitionID == 0 || input.ExamID == 0 {
		return SubmitExamResult{}, fmt.Errorf("%w: user_id, competition_id, and exam_id are required", ErrInvalidInput)
	}

	relationService := NewCompetitionRelationService(s.db)
	if err := relationService.EnsureCanSubmitCompetition(input.UserID, input.CompetitionID); err != nil {
		return SubmitExamResult{}, err
	}

	var exam models.Exam
	if err := s.db.First(&exam, input.ExamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubmitExamResult{}, ErrNotFound
		}
		return SubmitExamResult{}, err
	}

	examService := NewExamService(s.db)
	if examService.HasCompetitionSubmission(input.CompetitionID, input.UserID) {
		return SubmitExamResult{}, fmt.Errorf("%w: 您已经提交过该比赛的试卷", ErrInvalidInput)
	}

	ctx := parseExamJSONObject(exam.Context)
	questions := extractExamQuestionsFromContext(ctx)
	total := len(questions)
	correct := 0
	answers := input.Answers
	if answers == nil {
		answers = map[string]any{}
	}
	for _, q := range questions {
		qID := strings.TrimSpace(fmt.Sprint(q["id"]))
		if qID == "" {
			continue
		}
		userAnswer, exists := answers[qID]
		if !exists {
			continue
		}
		qCtx := map[string]any{}
		_ = json.Unmarshal([]byte(exJSON(q["context"])), &qCtx)
		if isTheoryAnswerCorrect(exClean(q["question_type"]), qCtx, userAnswer) {
			correct++
		}
	}

	score := 0
	if total > 0 {
		score = correct * (exam.TotalScore / total)
	}

	var teamID *uint
	var user models.User
	if err := s.db.Select("team_id").First(&user, input.UserID).Error; err == nil {
		teamID = user.TeamID
	}

	now := time.Now()
	_, err := examService.CreateSubmission(map[string]any{
		"competition_id": input.CompetitionID,
		"user_id":        input.UserID,
		"team_id":        teamID,
		"exam_id":        input.ExamID,
		"score":          score,
		"total_score":    exam.TotalScore,
		"correct":        correct,
		"total":          total,
		"status":         "graded",
		"submitted_at":   now,
		"graded_at":      now,
		"answers":        answers,
	})
	if err != nil {
		return SubmitExamResult{}, err
	}

	return SubmitExamResult{Score: score, TotalScore: exam.TotalScore, Correct: correct, Total: total}, nil
}

func isTheoryAnswerCorrect(questionType string, context map[string]any, userAnswer any) bool {
	if questionType == "single" || questionType == "multiple" {
		correctIndexes := checkedOptionIndexes(context["options"])
		userIndexes := answerIndexes(userAnswer)
		if len(correctIndexes) != len(userIndexes) {
			return false
		}
		for index := range correctIndexes {
			if !userIndexes[index] {
				return false
			}
		}
		return true
	}
	return strings.TrimSpace(fmt.Sprint(userAnswer)) == strings.TrimSpace(fmt.Sprint(context["answer"]))
}

func checkedOptionIndexes(raw any) map[string]bool {
	result := map[string]bool{}
	options, ok := raw.([]any)
	if !ok {
		return result
	}
	for index, item := range options {
		option, ok := item.(map[string]any)
		if ok && option["checked"] == true {
			result[fmt.Sprint(index)] = true
		}
	}
	return result
}

func answerIndexes(raw any) map[string]bool {
	result := map[string]bool{}
	if values, ok := raw.([]any); ok {
		for _, value := range values {
			result[strings.TrimSpace(fmt.Sprint(value))] = true
		}
		return result
	}
	result[strings.TrimSpace(fmt.Sprint(raw))] = true
	return result
}
