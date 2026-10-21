package services

import (
	"testing"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testTheorySubmissionService(t *testing.T) *TheorySubmissionService {
	t.Helper()
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}
	return NewTheorySubmissionService(db)
}

func TestTheorySubmissionServiceSubmitExamGradesAndPersists(t *testing.T) {
	svc := testTheorySubmissionService(t)

	user := models.User{Username: "u1", Email: "u1@test.com", HashedPassword: "x", IsActive: true}
	if err := svc.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	comp := models.Competition{Name: "c1", Type: "theory", ParticipantMode: "individual", Status: "ongoing", IsPublic: true}
	if err := svc.db.Create(&comp).Error; err != nil {
		t.Fatalf("create comp: %v", err)
	}
	if err := svc.db.Create(&models.CompetitionParticipant{CompetitionID: comp.ID, UserID: user.ID, Username: user.Username}).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}

	exam := models.Exam{CompetitionID: &comp.ID, Title: "e1", TotalScore: 100, IsActive: true, Context: `{"questions":[{"id":1,"question_type":"single","context":{"options":[{"text":"A","checked":true},{"text":"B","checked":false}]}},{"id":2,"question_type":"single","context":{"options":[{"text":"A","checked":false},{"text":"B","checked":true}]}}]}`}
	if err := svc.db.Create(&exam).Error; err != nil {
		t.Fatalf("create exam: %v", err)
	}

	result, err := svc.SubmitExam(SubmitExamInput{UserID: user.ID, CompetitionID: comp.ID, ExamID: exam.ID, Answers: map[string]any{"1": "0", "2": "0"}})
	if err != nil {
		t.Fatalf("submit exam: %v", err)
	}
	if result.Total != 2 || result.Correct != 1 || result.TotalScore != 100 {
		t.Fatalf("unexpected result: %#v", result)
	}

	// Ensure submission is persisted.
	var total int64
	_ = svc.db.Model(&models.ExamSubmission{}).Where("competition_id = ? AND user_id = ?", comp.ID, user.ID).Count(&total).Error
	if total != 1 {
		t.Fatalf("expected 1 exam submission, got %d", total)
	}
}
