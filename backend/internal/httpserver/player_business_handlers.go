package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"
)

func isPlayerAccessibleChallenge(ch challengeModel) bool {
	if !ch.IsActive || !ch.IsPublic {
		return false
	}
	if ch.CompetitionID != nil {
		return ch.Usage == "" || ch.Usage == "COMPETITION" || ch.Usage == "BOTH"
	}
	return ch.Usage == "" || ch.Usage == "PRACTICE" || ch.Usage == "BOTH"
}

func challengeSubmissionScope(ch challengeModel) string {
	if ch.CompetitionID != nil {
		return "competition"
	}
	return "practice"
}

func challengeContainerTimeoutOrDefault(timeout int) int {
	if timeout <= 0 {
		return 3600
	}
	return timeout
}

func (srv *Server) templateImageFromMetadata(templateID uint) string {
	if row, ok, _ := srv.loadTemplateRowByID(templateID); ok && row != nil {
		if image := strings.TrimSpace(cleanString(row["image"])); image != "" {
			return image
		}
		if servicesList := services.NormalizeTemplateServiceRows(row["services"]); len(servicesList) > 0 {
			for _, service := range servicesList {
				if image := strings.TrimSpace(cleanString(service["image"])); image != "" {
					return image
				}
			}
		}
	}
	path := filepath.Join("static", "template-files", fmt.Sprintf("%d", templateID), "metadata.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(strings.ReplaceAll(rawLine, "\r", ""))
		if !strings.HasPrefix(line, "image:") {
			continue
		}
		image := strings.TrimSpace(strings.Trim(strings.TrimPrefix(line, "image:"), `"'`))
		if image != "" && image != "<nil>" {
			return image
		}
	}
	return ""
}

func (srv *Server) templatePortSpecs(templateID uint) []string {
	row, ok, _ := srv.loadTemplateRowByID(templateID)
	if !ok {
		return nil
	}
	specs := make([]string, 0, 4)
	switch typed := row["ports"].(type) {
	case map[string]any:
		for _, rawPorts := range typed {
			for _, item := range services.NormalizeStringSlice(rawPorts) {
				if normalized := services.NormalizeContainerPort(item); normalized != "" {
					specs = append(specs, normalized)
				}
			}
		}
	case map[string]string:
		for _, item := range typed {
			if normalized := services.NormalizeContainerPort(item); normalized != "" {
				specs = append(specs, normalized)
			}
		}
	default:
		for _, item := range services.NormalizeStringSlice(row["ports"]) {
			_, containerPort := services.ParseComposePortMapping(item)
			if containerPort == "" {
				containerPort = item
			}
			if normalized := services.NormalizeContainerPort(containerPort); normalized != "" {
				specs = append(specs, normalized)
			}
		}
	}
	if len(specs) == 0 {
		return nil
	}
	sort.Strings(specs)
	return specs
}

func challengeContainerPortMap(portSpecs []string) map[string]string {
	ports := make(map[string]string, len(portSpecs))
	for _, spec := range portSpecs {
		normalized := services.NormalizeContainerPort(spec)
		if normalized == "" {
			continue
		}
		containerPort := services.PortNumber(normalized)
		if containerPort == "" {
			continue
		}
		ports[containerPort] = normalized
	}
	return ports
}

// ================================================
// Player flag submission
// ================================================

func (srv *Server) playerFlagSubmitHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	submissionService := srv.getSubmissionService()
	if submissionService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}

	challengeID := uintValue(r.PathValue("id"))
	if challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}

	payload, _ := readOptionalJSONMap(r)
	flag := cleanString(payload["flag"])
	if flag == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "flag required"})
		return
	}

	result, err := submissionService.SubmitFlag(services.SubmitFlagInput{
		UserID:      user.ID,
		ChallengeID: challengeID,
		Flag:        flag,
		IPAddress:   r.RemoteAddr,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// ================================================
// Player registration (real)
// ================================================

func (srv *Server) playerRealRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.systemSettingBool("allow_registration", true) {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "registration is disabled"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	username := cleanString(payload["username"])
	email := cleanString(payload["email"])
	password := cleanString(payload["password"])

	if username == "" || email == "" || password == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "username, email, and password are required"})
		return
	}
	if len(password) < 6 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "password must be at least 6 characters"})
		return
	}

	userService := srv.getUserService()
	if userService == nil {
		// Fallback for memory store
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": defaultPlayerUser(username)})
		return
	}

	if userService.ExistsByUsernameOrEmail(username, email) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "username or email already exists"})
		return
	}

	playerRole, err := userService.GetRoleByName(rolePlayer)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "system role not found"})
		return
	}

	registered, err := userService.RegisterPlayer(username, email, password, cleanString(payload["full_name"]), &playerRole.ID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create user"})
		return
	}

	_, _ = srv.getAuditLogService().Create(map[string]any{
		"user_id":       registered.ID,
		"username":      registered.Username,
		"action_type":   "register",
		"resource_type": "user",
		"resource_id":   fmt.Sprintf("%d", registered.ID),
		"description":   "用户注册",
		"ip_address":    r.RemoteAddr,
	})

	token, refreshToken, expiresAt, err := srv.issueAuthTokens(registered.ID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to issue auth tokens"})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"data":          srv.playerUserResponse(*registered),
		"access_token":  token,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_at":    expiresAt,
	})
}

// ================================================
// Player challenge detail with hints and solve status
// ================================================

func (srv *Server) playerChallengeDetailHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := resourceID(r.URL.Path)
	if challengeID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}
	challengeService := srv.getChallengeService()
	if challengeService == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	userID := uint(0)
	if user, ok := srv.currentUserFromRequest(r); ok {
		userID = user.ID
	}
	result, err := challengeService.PlayerDetail(services.PlayerChallengeDetailInput{
		ChallengeID: uintValue(challengeID),
		UserID:      userID,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "challenge not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ================================================
// Player container management
// ================================================

// playerStartContainerHandler 启动CTF题目
func (srv *Server) playerStartContainerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	challengeID := uintValue(r.PathValue("id"))
	if challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}

	// 获取challenge信息，记录日志
	var challenge models.Challenge
	if err := srv.db.Preload("Template").Preload("Template.Host").First(&challenge, challengeID).Error; err != nil {
		slog.Error("failed to load challenge", "challenge_id", challengeID, "error", err)
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	// 判断是否是比赛
	if challenge.CompetitionID != nil {
		slog.Info("player starting competition challenge runtime", "user_id", user.ID, "challenge_id", challengeID, "competition_id", *challenge.CompetitionID)
	} else {
		slog.Info("player starting practice challenge runtime", "user_id", user.ID, "challenge_id", challengeID)
	}
	// 校验题目是否可用
	if !challenge.IsDynamic {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge is not dynamic"})
		return
	}
	if !challenge.IsActive {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "challenge is not active"})
		return
	}
	if !challenge.IsPublic {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "challenge is not public"})
		return
	}
	// 校验模板
	if challenge.Template == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "当前题目环境未配置，请联系管理员!"})
		return
	}
	// 如果是比赛题目，检查比赛访问权限
	if challenge.CompetitionID != nil {
		if err := srv.checkCompetitionAccess(user.ID, *challenge.CompetitionID); err != nil {
			slog.Error("competition access denied", "user_id", user.ID, "challenge_id", challengeID, "competition_id", *challenge.CompetitionID, "error", err)
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
			return
		}
	}
	slog.Info("challenge template loaded", "challenge_id", challengeID, "template_id", challenge.Template.Id, "template_name", challenge.Template.Name, "host_id", challenge.Template.HostId)

	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	group, err := service.StartTemplateGroup(challenge.Template, &user, &challenge, nil)
	if err != nil {
		slog.Error("failed to start challenge runtime", "user_id", user.ID, "challenge_id", challengeID, "error", err)
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "container_info": challengeContainerGroupToInfo(*group)})
}

func (srv *Server) checkCompetitionAccess(userID, competitionID uint) error {
	var comp models.Competition
	if err := srv.db.First(&comp, competitionID).Error; err != nil {
		return fmt.Errorf("competition not found")
	}
	if !comp.IsPublic {
		return fmt.Errorf("competition is not public")
	}
	if strings.TrimSpace(comp.Status) != "ongoing" {
		return fmt.Errorf("competition is not ongoing")
	}
	// 判断比赛是团队赛还是个人赛，如果是团队赛就校验团队权限、如果是个人赛就验证个人权限
	if comp.ParticipantMode == "team" {
		var user models.User
		if err := srv.db.Select("team_id").First(&user, userID).Error; err != nil {
			return fmt.Errorf("user is not in a team")
		}
		if user.TeamID == nil || *user.TeamID == 0 {
			return fmt.Errorf("user is not in a team")
		}
		var count int64
		srv.db.Model(&models.CompetitionParticipant{}).
			Where("competition_id = ? AND team_id = ?", competitionID, *user.TeamID).
			Count(&count)
		if count <= 0 {
			return fmt.Errorf("team is not a participant")
		}
		return nil
	}
	var count int64
	srv.db.Model(&models.CompetitionParticipant{}).
		Where("competition_id = ? AND user_id = ?", competitionID, userID).
		Count(&count)
	if count <= 0 {
		return fmt.Errorf("user is not a participant")
	}
	return nil
}

func (srv *Server) playerExtendContainerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	challengeID := uintValue(r.PathValue("id"))
	if challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	group, err := service.ExtendChallengeRuntime(services.ExtendChallengeRuntimeInput{UserID: user.ID, ChallengeID: challengeID})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "message": "已延长30分钟", "container_info": challengeContainerGroupToInfo(group)})
}

func (srv *Server) playerStopContainerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	challengeID := uintValue(r.PathValue("id"))
	if challengeID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "challenge_id required"})
		return
	}
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	err := service.StopChallengeRuntime(services.StopChallengeRuntimeInput{UserID: user.ID, ChallengeID: challengeID})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "container_status": "stopped"})
}

// ================================================
// Player hint purchase
// ================================================

func (srv *Server) playerPurchaseHintHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	challengeID := resourceID(r.URL.Path)
	challengeService := srv.getChallengeService()
	if challengeService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	hintIndex, err := strconv.Atoi(r.PathValue("hint_index"))
	if err != nil || hintIndex < 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid hint index"})
		return
	}
	result, err := challengeService.PurchaseHint(services.PurchaseChallengeHintInput{
		UserID:      user.ID,
		ChallengeID: uintValue(challengeID),
		HintIndex:   hintIndex,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			detail := "challenge not found"
			if strings.Contains(err.Error(), "hint not found") {
				detail = "hint not found"
			}
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": detail})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": "challenge is not available"})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid hint index"})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to unlock hint"})
		}
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// ================================================
// Player competition registration
// ================================================

func (srv *Server) playerRegisterCompetitionHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	competitionID := resourceID(r.URL.Path)
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	teamID := optionalUint(payload["team_id"])
	if teamService := srv.getTeamService(); teamService != nil {
		if team, err := teamService.GetActiveTeamByUserID(user.ID); err == nil {
			currentTeamID := team.ID
			if teamID != nil && *teamID != currentTeamID {
				response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_id must match current team"})
				return
			}
			teamID = &currentTeamID
		}
	}
	err := relationService.RegisterParticipant(services.RegisterCompetitionInput{
		UserID:        user.ID,
		CompetitionID: uintValue(competitionID),
		TeamID:        teamID,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}

	// Create audit log
	_, _ = srv.getAuditLogService().Create(map[string]any{
		"user_id":       user.ID,
		"username":      user.Username,
		"action_type":   "register",
		"resource_type": "competition",
		"resource_id":   competitionID,
		"description":   "报名比赛: " + competitionID,
		"ip_address":    r.RemoteAddr,
	})

	response.JSON(w, http.StatusOK, map[string]any{"success": true, "message": "报名成功"})
}

// playerCompetitionDetailHandler 获取比赛详情
func (srv *Server) playerCompetitionDetailHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := pathUint(r, "id")
	var comp models.Competition
	if err := srv.db.First(&comp, competitionID).Error; err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "competition not found"})
		return
	}
	row := map[string]any{
		"id":                comp.ID,
		"name":              comp.Name,
		"description":       comp.Description,
		"type":              comp.Type,
		"participant_mode":  comp.ParticipantMode,
		"status":            comp.Status,
		"start_time":        comp.StartTime,
		"end_time":          comp.EndTime,
		"is_public":         comp.IsPublic,
		"registration_open": comp.RegistrationOpen,
		"scoreboard_frozen": comp.ScoreboardFrozen,
		"banner_url":        comp.BannerURL,
		"created_at":        comp.CreatedAt,
		"updated_at":        comp.UpdatedAt,
	}
	row["is_participant"] = false
	if user, ok := srv.currentUserFromRequest(r); ok {
		row["is_participant"] = srv.checkCompetitionAccess(user.ID, comp.ID) == nil
	}

	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) playerCompetitionAnnouncementsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getAnnouncementService().List(current, pageSize, map[string][]string{
		"is_published": {"true"},
		"target_type":  {"competition"},
		"target_id":    {competitionID},
	})
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

func (srv *Server) playerCompetitionScoreboardHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	service := srv.getScoreboardService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": []map[string]any{}, "total": 0})
		return
	}
	rows, total := service.ListByCompetition(competitionID, 100)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total})
}

// ================================================
// Player SSE notifications
// ================================================

var _ = playerSSEHandler // keep original

func (srv *Server) playerRealSSEHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.systemSettingBool("enable_notifications", true) {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "notifications are disabled"})
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"ok\"}\n\n"))
	flusher.Flush()

	lastCheck := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if service := srv.getNotificationService(); service != nil {
				count, _ := service.CountUnreadSince(user.ID, "player", lastCheck)
				if count > 0 {
					_, _ = w.Write([]byte(fmt.Sprintf("event: notification\ndata: {\"count\":%d}\n\n", count)))
					flusher.Flush()
				}
			}
			lastCheck = time.Now()
		}
	}
}

// ================================================
// Player vuln lab
// ================================================

func (srv *Server) playerVulnLabStartHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	labID := resourceID(r.URL.Path)
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	group, err := service.StartVulnLabRuntime(services.StartVulnLabRuntimeInput{UserID: user.ID, VulnLabID: uintValue(labID)})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "lab not found"})
		case errors.Is(err, services.ErrInvalidInput):
			detail := err.Error()
			if _, value, ok := strings.Cut(detail, ": "); ok {
				detail = value
			}
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": detail})
		default:
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":          true,
		"lab_id":           uintValue(labID),
		"user_id":          user.ID,
		"group":            challengeContainerGroupToInfo(group),
		"container_status": "running",
		"expires_at":       group.ExpiresAt,
	})
}

// 玩家获取靶场详情，包括当前环境状态
func (srv *Server) playerVulnLabDetailHandler(w http.ResponseWriter, r *http.Request) {
	labID := resourceID(r.URL.Path)
	if labID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "lab_id required"})
		return
	}
	labService := srv.getVulnLabService()
	if labService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	userID := uint(0)
	if user, ok := srv.currentUserFromRequest(r); ok {
		userID = user.ID
	}
	result, err := labService.PlayerDetail(labID, userID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": "lab is not available"})
			return
		}
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "lab not found"})
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (srv *Server) playerVulnLabExtendHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	labID := resourceID(r.URL.Path)
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	group, err := service.ExtendVulnLabRuntime(services.ExtendVulnLabRuntimeInput{UserID: user.ID, VulnLabID: uintValue(labID)})
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			detail := "active lab environment not found"
			if strings.Contains(err.Error(), "lab not found") {
				detail = "lab not found"
			}
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": detail})
			return
		}
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to extend lab runtime"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"message":        "已延长30分钟",
		"container_info": challengeContainerGroupToInfo(group),
	})
}

func (srv *Server) playerVulnLabStopHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	labID := resourceID(r.URL.Path)
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	err := service.StopVulnLabRuntime(services.StopVulnLabRuntimeInput{UserID: user.ID, VulnLabID: uintValue(labID)})
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			detail := "active lab environment not found"
			if strings.Contains(err.Error(), "lab not found") {
				detail = "lab not found"
			}
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": detail})
			return
		}
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to stop lab runtime"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "container_status": "stopped"})
}

// ================================================
// Player exam
// ================================================

func (srv *Server) playerExamGetHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	competitionID := resourceID(r.URL.Path)
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	if err := srv.checkCompetitionExamAllowed(uintValue(competitionID), user); err != nil {
		if err.Error() == "service unavailable" {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		return
	}
	exam, err := service.GetActiveByCompetition(competitionID)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]any{"detail": "no exam available"})
		return
	}

	ctx := parseJSONObject(exam.Context)
	questions := extractExamQuestionsFromContext(ctx)
	if len(questions) == 0 {
		response.JSON(w, http.StatusOK, map[string]any{"detail": "exam has no questions"})
		return
	}
	rows := make([]map[string]any, 0, len(questions))
	for _, q := range questions {
		row := normalizeExamQuestionRow(q)
		context := parseJSONObject(jsonString(row["context"]))
		row["context"] = sanitizePlayerExamQuestionContext(context)
		rows = append(rows, row)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"exam_id":        exam.ID,
		"title":          exam.Title,
		"total_score":    exam.TotalScore,
		"questions":      rows,
		"competition_id": competitionID,
		"user_id":        user.ID,
	})
}

func (srv *Server) playerCompetitionExamHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	competitionID := uintValue(r.PathValue("id"))
	if competitionID == 0 {
		competitionID = uintValue(r.URL.Query().Get("id"))
	}
	if competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition id required"})
		return
	}
	service := srv.getExamService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	if err := srv.checkCompetitionExamAllowed(competitionID, user); err != nil {
		if err.Error() == "service unavailable" {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
			return
		}
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		return
	}
	exam, err := service.GetActiveByCompetition(fmt.Sprint(competitionID))
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]any{"detail": "no exam available"})
		return
	}

	ctx := parseJSONObject(exam.Context)
	questions := extractExamQuestionsFromContext(ctx)
	if len(questions) == 0 {
		response.JSON(w, http.StatusOK, map[string]any{"detail": "exam has no questions"})
		return
	}
	rows := make([]map[string]any, 0, len(questions))
	for _, q := range questions {
		row := normalizeExamQuestionRow(q)
		context := parseJSONObject(jsonString(row["context"]))
		row["context"] = sanitizePlayerExamQuestionContext(context)
		rows = append(rows, row)
	}

	var submissionResult map[string]any
	if submission, found, getErr := service.GetCompetitionSubmission(competitionID, user.ID); getErr == nil && found {
		submissionResult = map[string]any{
			"correct":     intValue(submission["correct"], 0),
			"total":       intValue(submission["total"], 0),
			"score":       intValue(submission["score"], 0),
			"total_score": intValue(submission["total_score"], 0),
		}
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"exam_id":        exam.ID,
		"title":          exam.Title,
		"total_score":    exam.TotalScore,
		"questions":      rows,
		"competition_id": competitionID,
		"user_id":        user.ID,
		"submission":     submissionResult,
	})
}

func (srv *Server) playerExamSubmitHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	examID := uintValue(payload["exam_id"])
	competitionID := uintValue(r.PathValue("id"))
	if competitionID == 0 {
		competitionID = uintValue(resourceID(r.URL.Path))
	}
	if competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition id required"})
		return
	}

	theoryService := srv.getTheorySubmissionService()
	if theoryService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}

	answers, _ := payload["answers"].(map[string]any)
	result, err := theoryService.SubmitExam(services.SubmitExamInput{
		UserID:        user.ID,
		CompetitionID: competitionID,
		ExamID:        examID,
		Answers:       answers,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		case errors.Is(err, services.ErrForbidden):
			response.JSON(w, http.StatusForbidden, map[string]any{"detail": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		default:
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"exam_id":     examID,
		"user_id":     user.ID,
		"correct":     result.Correct,
		"total":       result.Total,
		"score":       result.Score,
		"total_score": result.TotalScore,
	})
}

func (srv *Server) playerExamScoreboardHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := resourceID(r.URL.Path)
	service := srv.getScoreboardService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	rows, total := service.ListByCompetition(competitionID, 50)
	response.JSON(w, http.StatusOK, map[string]any{"data": rows, "total": total})
}

func practiceHintsWithoutPurchase(ch challengeModel) []map[string]any {
	rawHints := ch.HintList()
	result := make([]map[string]any, 0, len(rawHints))
	for index, hint := range rawHints {
		result = append(result, map[string]any{
			"index":     index,
			"content":   cleanString(hint["content"]),
			"cost":      intValue(hint["cost"], 0),
			"purchased": false,
		})
	}
	return result
}

func challengeContainerGroupToInfo(group models.ContainerGroup) map[string]any {
	expires := 0
	expiresAtValue := any(nil)
	if group.ExpiresAt != nil {
		expiresAtValue = *group.ExpiresAt
		expires = int(time.Until(*group.ExpiresAt).Seconds())
		if expires < 0 {
			expires = 0
		}
	}
	containers := services.NormalizeTemplateServiceRows(containerRuntimeRows(group.Containers))
	containerID := strings.TrimSpace(group.ContainerID)
	containerName := strings.TrimSpace(group.Name)
	targets := []map[string]any{}
	for index, container := range containers {
		if index == 0 {
			if containerID == "" {
				containerID = cleanString(container["container_id"])
			}
			if containerName == "" {
				containerName = cleanString(container["name"])
			}
		}

		containerTargets := parseJSONArray(jsonString(container["targets"]))
		if len(containerTargets) > 0 {
			targets = append(targets, containerTargets...)
			continue
		}

		containerPorts := parseJSONArray(jsonString(container["ports"]))
		targets = append(targets, containerPorts...)
	}
	return map[string]any{
		"group_id":     group.ID,
		"container_id": containerID,
		"status":       group.Status,
		"created_at":   group.CreatedAt,
		"updated_at":   group.UpdatedAt,
		"expires_at":   expiresAtValue,
		"expires":      expires,
		"name":         containerName,
		"targets":      targets,
	}
}

func containerGroupFromRow(row map[string]any) models.ContainerGroup {
	group := models.ContainerGroup{
		ID:            uintValue(row["id"]),
		Name:          cleanString(row["name"]),
		Description:   cleanString(row["description"]),
		Status:        cleanString(row["status"]),
		SourceType:    cleanString(row["source_type"]),
		HostID:        optionalUint(row["host_id"]),
		TemplateID:    optionalUint(row["template_id"]),
		TemplateName:  cleanString(row["template_name"]),
		ChallengeID:   optionalUint(row["challenge_id"]),
		ChallengeName: cleanString(row["challenge_name"]),
		CompetitionID: optionalUint(row["competition_id"]),
		UserID:        optionalUint(row["user_id"]),
		UserName:      cleanString(row["user_name"]),
		VulnLabID:     optionalUint(row["vuln_lab_id"]),
		VulnLabName:   cleanString(row["vuln_lab_name"]),
		ContainerID:   cleanString(row["container_id"]),
		Containers:    containerRuntimeListValue(row["containers"]),
		Flag:          cleanString(row["flag"]),
		StartedAt:     parseOptionalTime(row["started_at"]),
		ExpiresAt:     parseOptionalTime(row["expires_at"]),
	}
	if createdAt := parseOptionalTime(row["created_at"]); createdAt != nil {
		group.CreatedAt = *createdAt
	}
	if updatedAt := parseOptionalTime(row["updated_at"]); updatedAt != nil {
		group.UpdatedAt = *updatedAt
	}
	return group
}

func containerRuntimeRows(items models.ContainerRuntimeList) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	payload, err := json.Marshal(items)
	if err != nil {
		return rows
	}
	_ = json.Unmarshal(payload, &rows)
	return rows
}

// ================================================
// Player writeup
// ================================================

func (srv *Server) playerWriteupHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.systemSettingBool("enable_writeup", true) {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "writeup is disabled"})
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	service := srv.getWriteupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	competitionID := uintValue(r.PathValue("id"))
	if competitionID == 0 {
		competitionID = uintValue(resourceID(r.URL.Path))
	}
	if competitionID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition id required"})
		return
	}
	if r.Method == http.MethodGet {
		item, err := service.GetByCompetitionAndUser(competitionID, user.ID)
		if err != nil {
			response.JSON(w, http.StatusOK, map[string]any{"data": nil, "message": "暂无 writeup"})
			return
		}
		response.JSON(w, http.StatusOK, service.ToRow(*item))
		return
	}
	if r.Method == http.MethodPost {
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
		dir := filepath.Join("static", "uploads", "writeups")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to prepare writeup dir"})
			return
		}
		filename := buildUploadFilename(header.Filename)
		localPath := filepath.Join(dir, filename)
		out, err := os.Create(localPath)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save writeup"})
			return
		}
		size, err := io.Copy(out, file)
		_ = out.Close()
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save writeup"})
			return
		}
		payload := map[string]any{
			"competition_id": competitionID,
			"user_id":        user.ID,
			"title":          strings.TrimSpace(r.FormValue("title")),
			"file_name":      header.Filename,
			"file_size":      size,
			"file_path":      localPath,
		}
		if cleanString(payload["title"]) == "" {
			payload["title"] = header.Filename
		}
		if teamService := srv.getTeamService(); teamService != nil {
			if team, err := teamService.GetActiveTeamByUserID(user.ID); err == nil {
				payload["team_id"] = team.ID
			}
		}
		row, err := service.Upsert(payload)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save writeup"})
			return
		}
		response.JSON(w, http.StatusOK, row)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}
