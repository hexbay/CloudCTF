package httpserver

import (
	"net/http"

	"cloudctf/backend/internal/httpserver/response"
)

func (srv *Server) adminCompetitionTheoryQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := r.PathValue("id")
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	service := srv.getTheoryService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"data": []map[string]any{}, "total": 0})
		return
	}

	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 100)
	rows, total := service.ListCompetitionQuestions(uintValue(competitionID), current, pageSize)
	response.JSON(w, http.StatusOK, map[string]any{"data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminAddTheoryQuestionToCompetitionHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := r.PathValue("id")
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	questionID := uintValue(payload["question_id"])
	if questionID == 0 {
		questionID = uintValue(r.URL.Query().Get("question_id"))
	}
	if questionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "question_id required"})
		return
	}

	service := srv.getTheoryService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true})
		return
	}
	if err := service.AddToCompetition(uintValue(competitionID), questionID); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "competition_id": uintValue(competitionID), "question_id": questionID})
}

func (srv *Server) adminRemoveTheoryQuestionFromCompetitionHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := uintValue(r.PathValue("id"))
	questionID := uintValue(r.PathValue("qid"))
	if competitionID == 0 || questionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id and question_id required"})
		return
	}
	service := srv.getTheoryService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true})
		return
	}
	service.RemoveFromCompetition(competitionID, questionID)
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminQuestionStatsHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTheoryService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"stats": map[string]int{}, "questions": []map[string]any{}})
		return
	}

	rows, _, err := service.List(1, 5000, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	stats := map[string]int{"text": 0, "judge": 0, "single": 0, "multiple": 0, "total": len(rows)}
	for _, row := range rows {
		kind := cleanString(row["question_type"])
		if _, exists := stats[kind]; exists {
			stats[kind]++
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"stats": stats, "questions": rows})
}

func (srv *Server) adminCompetitionExamHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := uintValue(r.PathValue("id"))
	if competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"exam": nil})
		return
	}

	switch r.Method {
	case http.MethodGet:
		exam := service.GetCompetitionExam(competitionID)
		response.JSON(w, http.StatusOK, exam)
	case http.MethodPost:
		payload, _ := readOptionalJSONMap(r)
		exam, err := service.CreateCompetitionExam(competitionID, payload)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusOK, exam)
	case http.MethodDelete:
		if !service.DeleteCompetitionExam(competitionID) {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "exam not found"})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"message": "试卷删除成功"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (srv *Server) adminCompetitionExamQuickCreateHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := uintValue(r.PathValue("id"))
	payload, _ := readOptionalJSONMap(r)
	examService := srv.getExamService()
	theoryService := srv.getTheoryService()
	if examService == nil || theoryService == nil || competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	rows, _ := theoryService.ListCompetitionQuestions(competitionID, 1, 5000)
	if len(rows) == 0 {
		var err error
		rows, _, err = theoryService.List(1, 5000, map[string][]string{})
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
	}
	exam, err := examService.QuickCreateCompetitionExam(competitionID, payload, rows)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, exam)
}

func (srv *Server) adminCompetitionExamBindHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := uintValue(r.PathValue("id"))
	if competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	sourceExamID := uintValue(payload["source_exam_id"])
	if sourceExamID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "source_exam_id required"})
		return
	}
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	exam, err := service.BindCompetitionExamFromSource(competitionID, sourceExamID)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, exam)
}

func (srv *Server) adminUpdateCompetitionExamHandler(w http.ResponseWriter, r *http.Request) {
	examID := uintValue(r.PathValue("id"))
	if examID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "exam_id required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	exam, err := service.UpdateCompetitionExam(examID, payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, exam)
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
			if id := uintValue(item); id > 0 {
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
