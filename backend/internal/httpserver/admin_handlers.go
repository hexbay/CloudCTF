package httpserver

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"
)

func (srv *Server) adminUserSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, service.Search(q, 20))
}

func adminUserImportHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "imported": 0, "message": "导入成功"})
}

func (srv *Server) adminCompetitionChallengesHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := r.PathValue("id")
	if competitionID == "" {
		competitionID = resourceID(r.URL.Path)
	}
	var records []models.Challenge
	var total int64
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 20)
	db := srv.db.Model(&models.Challenge{}).Where("competition_id = ?", competitionID)
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&records).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	rows := make([]map[string]any, 0, len(records))
	svc := srv.getChallengeService()
	for _, item := range records {
		rows = append(rows, svc.ToRow(item))
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminAddChallengeToCompetitionHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := r.PathValue("id")
	payload, _ := readOptionalJSONMap(r)
	challengeID := uintValue(payload["challenge_id"])
	if challengeID == 0 {
		challengeID = uintValue(payload["id"])
	}
	if competitionID == "" || challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id and challenge_id required"})
		return
	}
	compID := uintValue(competitionID)
	cloned, err := srv.getChallengeService().CloneToCompetition(challengeID, compID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "challenge_id": cloned.ID, "competition_id": compID, "source_challenge_id": challengeID})
}

func (srv *Server) adminRemoveChallengeFromCompetitionHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := r.PathValue("id")
	challengeID := r.PathValue("challenge_id")
	if competitionID == "" || challengeID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id and challenge_id required"})
		return
	}
	if !srv.getChallengeService().DeleteInCompetition(uintValue(challengeID), uintValue(competitionID)) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) playerJoinTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	invitationCode := cleanString(payload["invitation_code"])
	teamID := uintValue(payload["team_id"])
	teamService := srv.getTeamService()
	if teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	var team *teamModel
	var err error
	switch {
	case teamID != 0:
		team, err = teamService.GetByID(teamID)
		if err != nil {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
			return
		}
	case invitationCode != "":
		team, err = teamService.GetByInvitationCode(invitationCode)
		if err != nil {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
			return
		}
	default:
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_id or invitation_code required"})
		return
	}
	if team.IsBlacklisted {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team is unavailable"})
		return
	}
	if team.ID == 0 {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
		return
	}
	if teamService.HasActiveMembership(user.ID) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "already in a team, leave it first"})
		return
	}
	memberCount := teamService.ActiveMemberCount(team.ID)
	if int(memberCount) >= srv.systemSettingInt("max_team_size", 4) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team is full"})
		return
	}
	_ = teamService.EnsureMember(team.ID, user.ID, "joined via invitation code")
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "team_id": team.ID, "team_name": team.Name})
}

func (srv *Server) adminSubmissionDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getSubmissionService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "submission not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminExamSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	db := srv.db.Model(&models.ExamSubmission{}).Preload("User").Preload("Team")
	competitionID := r.URL.Query().Get("competition_id")
	if competitionID != "" {
		db = db.Where("competition_id = ?", competitionID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	var items []models.ExamSubmission
	if err := db.Order("id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{
			"id":                 item.ID,
			"competition_id":     item.CompetitionID,
			"user_id":            item.UserID,
			"username":           item.User.Username,
			"team_id":            item.TeamID,
			"team_name":          "",
			"exam_id":            item.ExamID,
			"total_score":        item.TotalScore,
			"correct_answers":    item.Correct,
			"total_questions":    item.Total,
			"percentage":         0,
			"submitted_at":       item.SubmittedAt,
			"is_graded":          item.Status == "graded",
		}
		if item.Team != nil {
			row["team_name"] = item.Team.Name
		}
		if item.TotalScore > 0 {
			row["percentage"] = float64(item.Correct) * 100 / float64(item.Total)
		}
		rows = append(rows, row)
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminExamSubmissionDetailHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	row, found, err := service.GetSubmissionByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "exam submission not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func adminReferralDomainStatsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": []map[string]any{}, "total": 0, "current": current, "pageSize": pageSize})
}

func adminResourcePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": []map[string]any{}, "total": 0, "current": queryInt(r, "current", 1), "pageSize": queryInt(r, "pageSize", 10)})
}

func adminUsersTemplateHandler(w http.ResponseWriter, r *http.Request) {
	writeCSVTemplate(w, "users_template.csv", []string{"username", "email", "password"}, [][]string{
		{"player1", "player1@example.com", "password123"},
	})
}

func adminTeamsTemplateHandler(w http.ResponseWriter, r *http.Request) {
	writeCSVTemplate(w, "teams_template.csv", []string{"name", "description"}, [][]string{
		{"Blue Team", "示例战队"},
	})
}

func writeCSVTemplate(w http.ResponseWriter, filename string, header []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(w)
	_ = writer.Write(header)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
}

func (srv *Server) adminChallengeFileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getChallengeService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return
	}
	challenge, err := service.GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		return
	}
	fileID := r.PathValue("file_id")
	files := make([]map[string]any, 0)
	removed := false
	for _, file := range challenge.FileList() {
		if matchesChallengeFile(file, fileID) {
			if path := stringifyID(file["file_path"]); path != "" {
				_ = os.Remove(strings.TrimPrefix(path, "/"))
			}
			removed = true
			continue
		}
		files = append(files, file)
	}
	_ = service.UpdateFiles(challenge.ID, files)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "removed": removed})
}

func matchesChallengeFile(file map[string]any, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, candidate := range []string{
		stringifyID(file["id"]),
		stringifyID(file["uid"]),
		stringifyID(file["file_path"]),
		stringifyID(file["url"]),
		stringifyID(file["filename"]),
	} {
		if strings.TrimSpace(candidate) == target {
			return true
		}
	}
	return false
}

func (srv *Server) adminChallengeHintCreateHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getChallengeService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return
	}
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	hint := map[string]any{
		"id":      time.Now().UnixNano(),
		"content": cleanString(payload["content"]),
		"cost":    intValue(payload["cost"], 0),
	}
	if _, ok := service.AddHint(r.PathValue("id"), hint); !ok {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		return
	}
	response.JSON(w, http.StatusOK, hint)
}

func (srv *Server) adminChallengeHintDeleteHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getChallengeService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return
	}
	removed := service.RemoveHint(r.PathValue("id"), r.PathValue("hint_id"))
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "removed": removed})
}

func (srv *Server) mutateTeamCompetitionBinding(w http.ResponseWriter, r *http.Request, bind bool) {
	teamID := uintValue(r.PathValue("id"))
	competitionID := uintValue(r.PathValue("cid"))
	if teamID == 0 || competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_id and competition_id required"})
		return
	}
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	if bind {
		row, err := relationService.CreateTeamCompetition(teamID, competitionID)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "binding": row})
		return
	}
	removed := relationService.DeleteTeamCompetition(teamID, competitionID)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "removed": removed})
}

func (srv *Server) adminUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "file is required"})
		return
	}
	defer file.Close()

	uploadDir := strings.TrimSpace(os.Getenv("UPLOAD_DIR"))
	if uploadDir == "" {
		uploadDir = "static/uploads"
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to prepare upload dir"})
		return
	}

	filename := buildUploadFilename(header.Filename)
	targetPath := filepath.Join(uploadDir, filename)
	targetFile, err := os.Create(targetPath)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create upload file"})
		return
	}
	defer targetFile.Close()

	size, err := io.Copy(targetFile, file)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save upload file"})
		return
	}

	result := map[string]any{
		"id":        time.Now().UnixNano(),
		"filename":  header.Filename,
		"file_path": "/static/uploads/" + filename,
		"url":       "/static/uploads/" + filename,
		"size":      size,
	}

	if strings.Contains(r.URL.Path, "/api-admin/challenges/") {
		if err := srv.appendChallengeFile(resourceID(r.URL.Path), result); err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
	}

	response.JSON(w, http.StatusOK, result)
}

func (srv *Server) appendChallengeFile(challengeID string, fileInfo map[string]any) error {
	service := srv.getChallengeService()
	if service == nil {
		return fmt.Errorf("challenge service unavailable")
	}
	return service.AppendFile(challengeID, fileInfo)
}

func buildUploadFilename(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(filepath.Base(original), ext)
	base = strings.TrimSpace(base)
	if base == "" {
		base = "upload"
	}
	base = strings.ReplaceAll(base, " ", "_")
	return fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), base, ext)
}

var backupMu sync.Mutex

type backupFileInfo = services.BackupFileInfo

func (srv *Server) requireAdminBackupUser(w http.ResponseWriter, r *http.Request) (userModel, bool) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return userModel{}, false
	}
	if user.Role.Name == rolePlayer && !user.IsSuperuser {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "admin permissions required"})
		return userModel{}, false
	}
	return user, true
}

func (srv *Server) adminUsersHandler(resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current := queryInt(r, "current", 1)
		pageSize := queryInt(r, "pageSize", 10)
		rows, total, err := srv.getUserService().List(resource, current, pageSize, r.URL.Query())
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
	}
}

func (srv *Server) adminUserCreateHandler(resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, ok := readJSONMap(w, r)
		if !ok {
			return
		}
		row, err := srv.getUserService().Create(resource, payload)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		response.JSON(w, http.StatusOK, row)
	}
}

func (srv *Server) adminUserDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getUserService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getUserService().ToRowWithTeam(*item))
}

func (srv *Server) adminUserUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getUserService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminUserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getUserService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminRolesHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getRoleService().List(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminRoleCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getRoleService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminRoleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getRoleService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminRoleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getRoleService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminUserEnableHandler(w http.ResponseWriter, r *http.Request) {
	row, found := srv.getUserService().Update(r.PathValue("id"), map[string]any{"is_active": true})
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminUserDisableHandler(w http.ResponseWriter, r *http.Request) {
	row, found := srv.getUserService().Update(r.PathValue("id"), map[string]any{"is_active": false})
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

type demoDataSummary struct {
	Scale        string   `json:"scale"`
	Users        int      `json:"users"`
	Teams        int      `json:"teams"`
	Competitions int      `json:"competitions"`
	Challenges   int      `json:"challenges"`
	Submissions  int      `json:"submissions"`
	Types        []string `json:"types"`
}

func (srv *Server) adminGenerateDemoDataHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getDemoDataService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	if _, ok := srv.currentUserFromRequest(r); !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}

	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	scale := cleanString(payload["scale"])
	if scale == "" {
		scale = "10-100"
	}

	summary, err := service.Generate(scale)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf(
			"demo data generated: scale=%s users=%d teams=%d competitions=%d challenges=%d submissions=%d",
			summary.Scale, summary.Users, summary.Teams, summary.Competitions, summary.Challenges, summary.Submissions,
		),
		"data": demoDataSummary(summary),
	})
}
