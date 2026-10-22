package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
)

// ================================================
// Dashboard extended stats
// ================================================

func (srv *Server) adminDashboardFullStatsHandler(w http.ResponseWriter, r *http.Request) {
	userService := srv.getUserService()
	competitionService := srv.getCompetitionService()
	challengeService := srv.getChallengeService()
	teamService := srv.getTeamService()
	submissionService := srv.getSubmissionService()
	if userService == nil || competitionService == nil || challengeService == nil || teamService == nil || submissionService == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	totalUsers := int64(userService.Count(""))
	totalCompetitions, ongoingComps, upcomingComps, finishedComps := competitionService.DashboardCounts(now)
	totalChallenges := int64(challengeService.Count())
	totalTeams := int64(teamService.Count())
	totalSubmissions := int64(submissionService.CountByCompetition(nil, false))
	correctSubmissions := int64(submissionService.CountByCompetition(nil, true))
	todaySubmissions := submissionService.CountSince(today)
	weekNewUsers := userService.CountCreatedSince(now.AddDate(0, 0, -7))
	var successRate float64
	if totalSubmissions > 0 {
		successRate = float64(correctSubmissions) / float64(totalSubmissions) * 100
	}
	participantTotal := teamService.CountActiveMembers()

	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"users":        map[string]any{"total": totalUsers, "new_today": 0, "new_week": weekNewUsers},
		"competitions": map[string]any{"total": totalCompetitions, "ongoing": ongoingComps, "upcoming": upcomingComps, "finished": finishedComps},
		"challenges":   map[string]any{"total": totalChallenges},
		"teams":        map[string]any{"total": totalTeams},
		"submissions":  map[string]any{"total": totalSubmissions, "today": todaySubmissions, "correct": correctSubmissions, "success_rate": successRate},
		"participants": map[string]any{"total": participantTotal},
	}})
}

func (srv *Server) adminRecentCompetitionsHandler(w http.ResponseWriter, r *http.Request) {
	competitionService := srv.getCompetitionService()
	teamService := srv.getTeamService()
	if competitionService == nil || teamService == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	rows := competitionService.Recent(5)
	participantCount := teamService.CountActiveMembers()
	for _, row := range rows {
		row["participant_count"] = participantCount
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows})
}

func (srv *Server) adminHotChallengesHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": service.HotChallenges(10)})
}

func (srv *Server) adminActiveUsersHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": service.ActiveUsers(10)})
}

func (srv *Server) adminSubmissionTrendHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	days := queryInt(r, "days", 7)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": service.SubmissionTrend(days)})
}

func (srv *Server) adminUserGrowthHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getAuditLogService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	days := queryInt(r, "days", 7)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": service.DashboardSummary(days)})
}

// ================================================
// User custom actions
// ================================================

func (srv *Server) adminBanUserHandler(w http.ResponseWriter, r *http.Request) {
	srv.adminUserAction(w, r, func(currentUser userModel, target *userModel) error {
		if target.ID == currentUser.ID {
			return fmt.Errorf("不能封禁自己")
		}
		if target.IsSuperuser {
			return fmt.Errorf("不能封禁超级管理员")
		}
		if _, ok := srv.getUserService().SetActive(target.ID, false); !ok {
			return fmt.Errorf("failed to update user status")
		}
		return nil
	})
}

func (srv *Server) adminUnbanUserHandler(w http.ResponseWriter, r *http.Request) {
	srv.adminUserAction(w, r, func(currentUser userModel, target *userModel) error {
		if _, ok := srv.getUserService().SetActive(target.ID, true); !ok {
			return fmt.Errorf("failed to update user status")
		}
		return nil
	})
}

func (srv *Server) adminResetUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	newPassword := cleanString(payload["password"])
	if len(newPassword) < 6 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "密码至少6位"})
		return
	}
	srv.adminUserAction(w, r, func(currentUser userModel, target *userModel) error {
		if !srv.getUserService().UpdatePassword(target.ID, newPassword) {
			return fmt.Errorf("failed to update password")
		}
		return nil
	})
}

func readIDArray(r *http.Request) []uint {
	if r.Body == nil {
		return nil
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil
	}
	// Try array: [1,2,3]
	var arr []uint
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	// Try object: {"ids":[1,2]} or {"user_ids":[1,2]}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err == nil {
		return extractIDs(obj)
	}
	return nil
}

func (srv *Server) adminBatchDeleteUsersHandler(w http.ResponseWriter, r *http.Request) {
	ids := readIDArray(r)
	if len(ids) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "ids are required"})
		return
	}
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	deleted := service.BatchDeleteNonSuperusers(ids)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "deleted": deleted})
}

func (srv *Server) adminBatchBanUsersHandler(w http.ResponseWriter, r *http.Request) {
	ids := readIDArray(r)
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	currentUser, _ := srv.currentUserFromRequest(r)
	count := service.BatchSetActive(ids, false, currentUser.ID, true)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "count": count})
}

func (srv *Server) adminBatchUnbanUsersHandler(w http.ResponseWriter, r *http.Request) {
	ids := readIDArray(r)
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	count := service.BatchSetActive(ids, true, 0, false)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "count": count})
}

// ================================================
// Settings
// ================================================

func (srv *Server) adminSettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response.JSON(w, http.StatusOK, srv.loadSystemSettings())
	case http.MethodPut:
		payload, ok := readJSONMap(w, r)
		if !ok {
			return
		}
		if err := srv.saveSystemSettings(payload); err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusOK, payload)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func defaultSettings() map[string]any {
	return map[string]any{
		"site_name":            "CloudCTF",
		"site_title":           "CloudCTF",
		"site_logo":            "",
		"site_description":     "CTF SaaS 平台",
		"allow_registration":   true,
		"email_verification":   false,
		"max_team_size":        4,
		"ctf_ttl_seconds":      3800,
		"vuln_ttl_seconds":     3800,
		"enable_writeup":       true,
		"enable_notifications": true,
		"show_github_link":     true,
	}
}

// ================================================
// Writeup
// ================================================

func (srv *Server) adminListWriteupsHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getWriteupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	competitionID := uintValue(resourceID(r.URL.Path))
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	db := service.DB()
	var total int64
	if err := db.Model(&models.Writeup{}).Where("competition_id = ?", competitionID).Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	type writeupWithNames struct {
		models.Writeup
		Username string `gorm:"column:username"`
		TeamName string `gorm:"column:team_name"`
	}
	var items []writeupWithNames
	if err := db.
		Table("writeups").
		Select("writeups.*, user.username, team.name AS team_name").
		Joins("LEFT JOIN user ON user.id = writeups.user_id").
		Joins("LEFT JOIN team ON team.id = writeups.team_id").
		Where("writeups.competition_id = ?", competitionID).
		Order("writeups.id DESC").
		Offset((current - 1) * pageSize).
		Limit(pageSize).
		Scan(&items).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := service.ToRow(item.Writeup)
		row["username"] = item.Username
		row["team_name"] = item.TeamName
		rows = append(rows, row)
	}
	response.JSON(w, http.StatusOK, map[string]any{"data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminDeleteWriteupHandler(w http.ResponseWriter, r *http.Request) {
	id := uintValue(r.PathValue("id"))
	if id == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "writeup id required"})
		return
	}
	service := srv.getWriteupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	if !service.Delete(id) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "writeup not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"message": "删除成功"})
}

// ================================================
// Audit export
// ================================================

func (srv *Server) adminAuditExportHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getAuditLogService()
	if service == nil {
		writeExport(w, "audit_logs")
		return
	}
	logs := service.Recent(10000)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="audit_logs.csv"`)
	_, _ = w.Write([]byte("id,username,action_type,resource_type,resource_id,description,ip_address,user_agent,created_at\n"))
	for _, lg := range logs {
		line := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s\n",
			lg.ID, lg.Username, lg.ActionType, lg.ResourceType, lg.ResourceID, escapeCSV(lg.Description), lg.IPAddress, lg.UserAgent, lg.CreatedAt.Format(time.RFC3339))
		_, _ = w.Write([]byte(line))
	}
}

func escapeCSV(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ================================================
// Helpers
// ================================================

func (srv *Server) adminUserAction(w http.ResponseWriter, r *http.Request, fn func(userModel, *userModel) error) {
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	currentUser, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	targetID := resourceID(r.URL.Path)
	if targetID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "user id is required"})
		return
	}
	targetUser, err := service.GetByID(targetID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "user not found"})
		return
	}
	if err := fn(currentUser, targetUser); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func extractIDs(payload map[string]any) []uint {
	raw, ok := payload["ids"]
	if !ok {
		raw, ok = payload["user_ids"]
	}
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []any:
		ids := make([]uint, 0, len(typed))
		for _, item := range typed {
			ids = append(ids, uintValue(item))
		}
		return ids
	case []float64:
		ids := make([]uint, 0, len(typed))
		for _, item := range typed {
			ids = append(ids, uint(item))
		}
		return ids
	}
	return nil
}

func (srv *Server) adminBatchResetUserPasswordsHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	newPassword := cleanString(payload["password"])
	if len(newPassword) < 6 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "password must be at least 6 characters"})
		return
	}
	ids := extractIDs(payload)
	if len(ids) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "ids are required"})
		return
	}
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	updated := 0
	for _, id := range ids {
		if service.UpdatePassword(id, newPassword) {
			updated++
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "updated": updated})
}

func (srv *Server) adminChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getUserService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	currentUser, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	oldPassword := cleanString(payload["old_password"])
	if oldPassword == "" {
		oldPassword = cleanString(payload["current_password"])
	}
	newPassword := cleanString(payload["new_password"])
	if newPassword == "" {
		newPassword = cleanString(payload["password"])
	}
	if len(newPassword) < 6 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "password must be at least 6 characters"})
		return
	}
	if !verifyPassword(oldPassword, currentUser.HashedPassword) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "current password is incorrect"})
		return
	}
	if !service.ChangeOwnPassword(currentUser.ID, newPassword) {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to update password"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "must_change_password": false})
}

func (srv *Server) adminUserTagsMutationHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	if srv.getUserService() == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	targetID := uintValue(r.PathValue("id"))
	if targetID == 0 {
		targetID = uintValue(resourceID(r.URL.Path))
	}
	if targetID != 0 {
		if err := srv.adminApplyUserTags(targetID, payload["tags"]); err != nil {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "user not found"})
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"success": true})
		return
	}
	ids := extractIDs(payload)
	if len(ids) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "user id or ids are required"})
		return
	}
	updated := 0
	for _, id := range ids {
		if err := srv.adminApplyUserTags(id, payload["tags"]); err == nil {
			updated++
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "updated": updated})
}

func (srv *Server) adminApplyUserTags(userID uint, tags any) error {
	if ok := srv.getUserService().ApplyTags(userID, tags); !ok {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (srv *Server) adminTeamMemberHistoryHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTeamService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	teamID := uintValue(r.PathValue("id"))
	if teamID == 0 {
		teamID = uintValue(resourceID(r.URL.Path))
	}
	response.JSON(w, http.StatusOK, service.MemberHistory(teamID))
}

func (srv *Server) adminUserTeamHistoryHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTeamService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	userID := uintValue(r.PathValue("user_id"))
	if userID == 0 {
		userID = uintValue(resourceID(r.URL.Path))
	}
	response.JSON(w, http.StatusOK, service.UserHistory(userID))
}

// avoid unused
var _ = fmt.Sprintf
