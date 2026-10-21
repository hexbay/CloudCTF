package httpserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloudctf/backend/internal/httpserver/response"
)

// ================================================
// Flag submission (admin testing)
// ================================================

type flagSubmitRequest struct {
	ChallengeID   uint   `json:"challenge_id"`
	Flag          string `json:"flag"`
	CompetitionID *uint  `json:"competition_id"`
}

func (srv *Server) adminFlagSubmitHandler(w http.ResponseWriter, r *http.Request) {
	var req flagSubmitRequest
	if err := decodeJSONBody(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	if req.ChallengeID == 0 || req.Flag == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id and flag are required"})
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	challengeService := srv.getChallengeService()
	submissionService := srv.getSubmissionService()
	scoreboardService := srv.getScoreboardService()
	if challengeService == nil || submissionService == nil || scoreboardService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	challenge, err := challengeService.GetByID(req.ChallengeID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		return
	}
	if challengeService.ExistsSolvedByUser(req.ChallengeID, user.ID) {
		response.JSON(w, http.StatusOK, map[string]any{"correct": false, "message": "已解决", "already_solved": true})
		return
	}

	isCorrect := false
	points := 0
	if challenge.IsDynamic {
		isCorrect = verifyDynamicFlag(user.ID, challenge.ID, challenge.Flag, req.Flag)
	} else {
		isCorrect = req.Flag == challenge.Flag
	}
	if isCorrect {
		points = challenge.Points
	}
	_, _ = submissionService.Create(map[string]any{
		"challenge_id":   req.ChallengeID,
		"user_id":        user.ID,
		"competition_id": req.CompetitionID,
		"flag":           req.Flag,
		"is_correct":     isCorrect,
		"points":         points,
		"ip_address":     r.RemoteAddr,
	})

	if isCorrect && req.CompetitionID != nil && *req.CompetitionID != 0 {
		scoreboardService.AddCompetitionScore(*req.CompetitionID, user.ID, nil, float64(points))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"correct":        isCorrect,
		"points":         points,
		"already_solved": false,
		"message":        map[bool]string{true: "正确!", false: "FLAG 错误"}[isCorrect],
	})
}

// ================================================
// Dynamic flag generation
// ================================================

func (srv *Server) adminDynamicFlagHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := uint(queryInt(r, "challenge_id", 0))
	if challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id is required"})
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	service := srv.getChallengeService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}

	flag := generateDynamicFlag(user.ID, challengeID)
	if challenge, err := service.GetByID(challengeID); err == nil && challenge.IsDynamic {
		_ = service.UpdateFlag(challengeID, flag)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"flag":         flag,
		"challenge_id": challengeID,
		"user_id":      user.ID,
		"expires_at":   time.Now().Add(24 * time.Hour),
	})
}

func (srv *Server) adminViewDynamicFlagHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := resourceID(r.URL.Path)
	userID := queryInt(r, "user_id", 0)
	if challengeID == "" || userID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id and user_id required"})
		return
	}
	service := srv.getChallengeService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	ch, err := service.GetByID(challengeID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		return
	}
	flag := generateDynamicFlag(uint(userID), ch.ID)
	response.JSON(w, http.StatusOK, map[string]any{"user_id": userID, "challenge_id": ch.ID, "flag": flag})
}

func generateDynamicFlag(userID, challengeID uint) string {
	salt := fmt.Sprintf("cloudctf_%d_%d_%d", userID, challengeID, time.Now().Unix()/86400)
	hash := md5.Sum([]byte(salt))
	return fmt.Sprintf("flag{%x}", hash)
}

func verifyDynamicFlag(userID, challengeID uint, expectedFlag, submittedFlag string) bool {
	today := generateDynamicFlag(userID, challengeID)
	yesterday := fmt.Sprintf("flag{%x}", md5.Sum([]byte(fmt.Sprintf("cloudctf_%d_%d_%d", userID, challengeID, time.Now().Unix()/86400-1))))
	return submittedFlag == expectedFlag || submittedFlag == today || submittedFlag == yesterday
}

// ================================================
// Challenge import / export
// ================================================

func (srv *Server) adminChallengeImportHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid multipart form"})
		return
	}
	if srv.getChallengeService() == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	competitionID := optionalUint(r.FormValue("competition_id"))

	file, header, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "file is required"})
		return
	}
	defer file.Close()

	content := make([]byte, header.Size)
	_, _ = file.Read(content)

	payload := map[string]any{}
	if err := json.Unmarshal(content, &payload); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid JSON file"})
		return
	}

	challenges, ok := payload["challenges"]
	if !ok {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "missing 'challenges' key"})
		return
	}

	imported := 0
	switch items := challenges.(type) {
	case []any:
		for _, item := range items {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if competitionID != nil {
				row["competition_id"] = *competitionID
			}
			if _, err := srv.getChallengeService().Create(row); err == nil {
				imported++
			}
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "imported": imported})
}

func (srv *Server) adminChallengeExportHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getChallengeService()
	if service == nil {
		writeExport(w, "challenges")
		return
	}
	rows := service.ExportRows(r.URL.Query().Get("category"))

	exportData := map[string]any{"challenges": rows, "exported_at": time.Now().Format(time.RFC3339)}
	payload, _ := json.MarshalIndent(exportData, "", "  ")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="challenges.json"`)
	_, _ = w.Write(payload)
}

func (srv *Server) adminTheoryImportHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "file is required"})
		return
	}
	defer file.Close()

	content := make([]byte, header.Size)
	_, _ = file.Read(content)

	payload := map[string]any{}
	if err := json.Unmarshal(content, &payload); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid JSON file"})
		return
	}

	questions, ok := payload["questions"]
	if !ok {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "missing 'questions' key"})
		return
	}

	imported := 0
	switch items := questions.(type) {
	case []any:
		for _, item := range items {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if service := srv.getTheoryService(); service != nil {
				if _, err := service.Create(row); err == nil {
					imported++
				}
			}
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "imported": imported})
}

func (srv *Server) adminTheoryExportHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTheoryService()
	if service == nil {
		writeExport(w, "theory")
		return
	}
	rows := service.ExportRows()
	exportData := map[string]any{"questions": rows, "exported_at": time.Now().Format(time.RFC3339)}
	payload, _ := json.MarshalIndent(exportData, "", "  ")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="theory_questions.json"`)
	_, _ = w.Write(payload)
}

// ================================================
// Scoreboard operations
// ================================================

func (srv *Server) adminFreezeScoreboardHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	service := srv.getCompetitionService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	_ = service.SetScoreboardFrozen(competitionID, true)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "frozen": true})
}

func (srv *Server) adminUnfreezeScoreboardHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	service := srv.getCompetitionService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	_ = service.SetScoreboardFrozen(competitionID, false)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "frozen": false})
}

func (srv *Server) adminScoreboardStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	scoreboardService := srv.getScoreboardService()
	submissionService := srv.getSubmissionService()
	if scoreboardService == nil || submissionService == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"participants":      scoreboardService.CountParticipants(competitionID),
		"total_submissions": submissionService.CountByCompetition(competitionID, false),
		"correct":           submissionService.CountByCompetition(competitionID, true),
	})
}

func (srv *Server) adminSolveStatusHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, service.SolveStatus())
}

// ================================================
// Helpers
// ================================================

func decodeJSONBody(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

// avoid unused imports
var _ = fmt.Sprintf
