package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/services"
)

// ================================================
// Notification send
// ================================================

func (srv *Server) adminUnreadCountHandler(w http.ResponseWriter, r *http.Request) {
	if notificationAudienceFromRequest(r) == "player" && !srv.systemSettingBool("enable_notifications", true) {
		response.JSON(w, http.StatusOK, map[string]any{"count": 0})
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusOK, map[string]any{"count": 0})
		return
	}
	service := srv.getNotificationService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"count": 0})
		return
	}
	audience := notificationAudienceFromRequest(r)
	count, _ := service.CountUnread(user.ID, audience)
	response.JSON(w, http.StatusOK, map[string]any{"count": count})
}

func (srv *Server) adminNotificationsInboxHandler(w http.ResponseWriter, r *http.Request) {
	srv.notificationsInboxHandler(w, r, "admin")
}

func (srv *Server) playerNotificationsInboxHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.systemSettingBool("enable_notifications", true) {
		current := queryInt(r, "current", 1)
		pageSize := queryInt(r, "pageSize", 10)
		response.JSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"data":     []map[string]any{},
			"total":    0,
			"current":  current,
			"pageSize": pageSize,
		})
		return
	}
	srv.notificationsInboxHandler(w, r, "player")
}

func (srv *Server) adminMarkAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	_ = srv.getNotificationService().MarkAllRead(user.ID, notificationAudienceFromRequest(r))
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminMarkOneReadHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	messageID := resourceID(r.URL.Path)
	if !srv.getNotificationService().MarkOneRead(messageID, user.ID, notificationAudienceFromRequest(r)) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "notification not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func notificationAudience(payload map[string]any, fallback string) string {
	value := cleanString(payload["audience"])
	if value == "" {
		value = cleanString(payload["target_audience"])
	}
	if value != "admin" && value != "player" {
		return fallback
	}
	return value
}

func notificationAudienceFromRequest(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, "/api-admin/") {
		return "admin"
	}
	return "player"
}

func cloneQuery(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, item := range values {
		cloned[key] = append([]string(nil), item...)
	}
	return cloned
}

func questionPointsTotal(rows []map[string]any) int {
	total := 0
	for _, row := range rows {
		total += intValue(row["points"], 0)
	}
	return total
}

// ================================================
// Announcement custom actions
// ================================================

func (srv *Server) adminPublishAnnouncementHandler(w http.ResponseWriter, r *http.Request) {
	announcementStateAction(w, r, func(id string) bool {
		return srv.getAnnouncementService().SetPublished(id, true)
	})
}

func (srv *Server) adminUnpublishAnnouncementHandler(w http.ResponseWriter, r *http.Request) {
	announcementStateAction(w, r, func(id string) bool {
		return srv.getAnnouncementService().SetPublished(id, false)
	})
}

func (srv *Server) adminPinAnnouncementHandler(w http.ResponseWriter, r *http.Request) {
	announcementStateAction(w, r, func(id string) bool {
		return srv.getAnnouncementService().SetPinned(id, true)
	})
}

func (srv *Server) adminUnpinAnnouncementHandler(w http.ResponseWriter, r *http.Request) {
	announcementStateAction(w, r, func(id string) bool {
		return srv.getAnnouncementService().SetPinned(id, false)
	})
}

func announcementStateAction(w http.ResponseWriter, r *http.Request, apply func(string) bool) {
	annID := resourceID(r.URL.Path)
	if annID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "id is required"})
		return
	}
	if !apply(annID) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "announcement not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

// ================================================
// Theory / Exam custom actions
// ================================================

func adminTheorySubmitHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"correct": false, "message": "theory submission accepted"})
}

func (srv *Server) adminExamQuickCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("id") != "" {
		srv.adminCompetitionExamQuickCreateHandler(w, r)
		return
	}
	theoryService := srv.getTheoryService()
	examService := srv.getExamService()
	if theoryService == nil || examService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	competitionID := resourceID(r.URL.Path)
	_ = competitionID
	payload, _ := readOptionalJSONMap(r)

	count := intValue(payload["count"], 10)
	rows, _, err := theoryService.List(1, 5000, map[string][]string{})
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if count > 0 && len(rows) > count {
		rows = rows[:count]
	}
	exam, err := examService.Create(map[string]any{
		"title":       valueOrDefault(payload["title"], "自动生成试卷"),
		"total_score": questionPointsTotal(rows),
		"is_active":   true,
		"context":     map[string]any{"questions": rows, "time_limit": 3600},
	})
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"exam_id":     exam["id"],
		"title":       exam["title"],
		"total_score": exam["total_score"],
		"questions":   rows,
		"count":       len(rows),
	})
}

func (srv *Server) adminExamCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("id") != "" {
		srv.adminCompetitionExamHandler(w, r)
		return
	}
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	examService := srv.getExamService()
	if examService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}

	// Manual exam creation with specified questions
	questionIDs := extractIDs(payload)
	title := cleanString(payload["title"])
	if title == "" {
		title = "鎵嬪姩鍒涘缓璇曞嵎"
	}
	if len(questionIDs) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "question ids required"})
		return
	}

	exam, err := examService.Create(map[string]any{
		"title":       title,
		"total_score": intValue(payload["total_score"], 0),
		"is_active":   true,
		"context": map[string]any{
			"question_ids": questionIDs,
			"time_limit":   intValue(payload["time_limit"], 3600),
		},
	})
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"exam_id":     exam["id"],
		"total_score": exam["total_score"],
		"questions":   len(questionIDs),
	})
}

func (srv *Server) adminExamStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	submissionService := srv.getSubmissionService()
	if submissionService == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	competitionID := resourceID(r.URL.Path)
	submissions := int64(submissionService.CountByCompetition(competitionID, false))
	correct := int64(submissionService.CountByCompetition(competitionID, true))
	avgScore := 0.0
	if submissions > 0 {
		avgScore = float64(correct) / float64(submissions)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"total_submissions": submissions,
		"avg_score":         avgScore,
	})
}

// ================================================
// Resource repository sync
// ================================================

func (srv *Server) adminResourceRepoSyncHandler(w http.ResponseWriter, r *http.Request) {
	repoID := resourceID(r.URL.Path)
	if repoID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "repo id required"})
		return
	}
	resourceRepoService := srv.getResourceRepositoryService()
	miscService := srv.getMiscResourceService()
	vulnLabService := srv.getVulnLabService()
	templateService := srv.getTemplateRuntimeService()
	if resourceRepoService == nil || miscService == nil || vulnLabService == nil || templateService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	repo, err := resourceRepoService.GetByID(repoID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource repository not found"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	pruneMissing := boolValue(payload["force"], false) || boolValue(payload["prune_missing"], false)
	logger := slog.Default().With("repo_id", repo.ID, "repo_name", repo.Name)
	now := time.Now()
	logger.Info("resource sync started", "prune_missing", pruneMissing, "branch", repo.RepositoryBranch)
	synced, deleted, err := templateService.SyncResourceRepository(
		logger,
		services.NewTemplateParserService(),
		srv.getVirtualizationHostService(),
		vulnLabService,
		*repo,
		now,
		pruneMissing,
	)
	status := "completed"
	messageText := fmt.Sprintf("synced %d templates", synced)
	if deleted > 0 {
		messageText = fmt.Sprintf("%s, deleted %d stale templates", messageText, deleted)
	}
	if err != nil {
		status = "failed"
		messageText = err.Error()
		logger.Error("resource sync failed", "error", err, "synced", synced, "deleted", deleted)
	} else {
		logger.Info("resource sync completed", "synced", synced, "deleted", deleted, "branch", repo.RepositoryBranch)
	}
	_, _ = miscService.CreateTemplateSyncLog(map[string]any{
		"status":  status,
		"message": fmt.Sprintf("%s (repo=%s, synced=%d, deleted=%d)", messageText, repo.Name, synced, deleted),
	})
	_ = resourceRepoService.TouchLastRun(repo.ID, now)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{
			"detail":         err.Error(),
			"success":        false,
			"template_count": synced,
			"deleted_count":  deleted,
			"synced_at":      now,
		})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"message":        messageText,
		"template_count": synced,
		"deleted_count":  deleted,
		"synced_at":      now,
	})
}

// ================================================
// Challenge container management (admin)
// ================================================

func (srv *Server) adminChallengeStartContainerHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := resourceID(r.URL.Path)
	if challengeID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
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
	if !ch.IsDynamic {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge is not dynamic"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":          true,
		"container_status": "running",
		"container_port":   ch.ContainerPort,
		"host":             ch.HostID,
	})
}

func adminChallengeStopContainerHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := resourceID(r.URL.Path)
	if challengeID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "container_status": "stopped"})
}

// ================================================
// Role permissions
// ================================================

func (srv *Server) adminRolePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	roleID := resourceID(r.URL.Path)
	service := srv.getRoleService()
	if service == nil {
		response.JSON(w, http.StatusOK, []string{})
		return
	}
	role, err := service.GetByID(roleID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "role not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"permissions": role.PermissionList()})
}

func adminAllPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, []string{
		"admin:*", "admin:users", "admin:roles", "admin:competitions", "admin:challenges",
		"admin:theory", "admin:exam", "admin:teams", "admin:submissions", "admin:scoreboard",
		"admin:notifications", "admin:announcements", "admin:audit", "admin:virtualization",
		"admin:vuln_labs", "admin:resource_repositories", "admin:settings", "admin:writeup",
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (srv *Server) adminNotificationCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getNotificationService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminNotificationUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getNotificationService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminNotificationDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getNotificationService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) notificationsInboxHandler(w http.ResponseWriter, r *http.Request, audience string) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		if audience == "admin" {
			current := queryInt(r, "current", 1)
			pageSize := queryInt(r, "pageSize", 10)
			response.JSON(w, http.StatusOK, map[string]any{
				"success":  true,
				"data":     []map[string]any{},
				"total":    0,
				"current":  current,
				"pageSize": pageSize,
			})
			return
		}
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	query := cloneQuery(r.URL.Query())
	query.Set("receiver_id", fmt.Sprint(user.ID))
	query.Set("audience", audience)
	rows, total, err := srv.getNotificationService().List(current, pageSize, query)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     rows,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}
