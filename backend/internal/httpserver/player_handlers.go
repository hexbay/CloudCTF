package httpserver

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
)

type playerLoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         any       `json:"user"`
}

func (srv *Server) playerLoginHandler(w http.ResponseWriter, r *http.Request) {
	username, password, err := readPlayerCredentials(r)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	if username == "" || password == "" {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": "username and password are required"})
		return
	}

	user, ok := srv.findUserByLogin(username)
	if !ok || !verifyPassword(password, user.HashedPassword) {
		srv.auditLoginFailed(username, "player", r)
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Incorrect username or password"})
		return
	}
	if !user.IsActive {
		srv.auditLoginFailed(username, "player", r)
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "Inactive user"})
		return
	}

	accessToken, refreshToken, expiresAt, err := srv.issueAuthTokens(user.ID)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to issue auth tokens"})
		return
	}

	srv.auditLoginSuccess(user.ID, user.Username, "player", r)
	response.JSON(w, http.StatusOK, playerLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
		ExpiresAt:    expiresAt,
		User:         srv.playerUserResponse(user),
	})
}

func (srv *Server) playerMeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	response.JSON(w, http.StatusOK, srv.playerUserResponse(user))
}

func (srv *Server) playerRefreshHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	token := stringifyID(payload["refresh_token"])
	user, accessToken, refreshToken, expiresAt, ok := srv.rotateRefreshToken(token)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_at":    expiresAt,
		"user":          srv.playerUserResponse(user),
	})
}

func (srv *Server) playerLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if user, ok := srv.currentUserFromRequest(r); ok {
		_ = srv.revokeAllUserRefreshTokens(user.ID)
	}
	if payload, ok := readOptionalJSONMap(r); ok {
		_ = srv.revokeRefreshToken(stringifyID(payload["refresh_token"]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"msg": "ok", "success": true})
}

func (srv *Server) playerPracticeChallengesHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"data":     []map[string]any{},
			"total":    0,
			"current":  1,
			"pageSize": 100,
		})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 100)
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	difficulty := queryInt(r, "difficulty", 0)
	competitionID := strings.TrimSpace(r.URL.Query().Get("competition_id"))
	var userID uint
	if user, ok := srv.currentUserFromRequest(r); ok {
		userID = user.ID
	}
	rows, total, err := service.PracticeChallengeRows(current, pageSize, category, difficulty, competitionID, userID)
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

func (srv *Server) playerStatsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	scoreboardService := srv.getScoreboardService()
	teamService := srv.getTeamService()
	if scoreboardService == nil {
		response.JSON(w, http.StatusOK, map[string]any{"total_points": 0, "solved_challenges": 0, "rank": 0, "team_name": ""})
		return
	}
	stats := scoreboardService.UserStats(user.ID)
	if teamService != nil {
		if team, err := teamService.GetActiveTeamByUserID(user.ID); err == nil {
			stats["team_id"] = team.ID
			stats["team_name"] = team.Name
		}
	}
	response.JSON(w, http.StatusOK, stats)
}

func (srv *Server) playerPracticeLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getScoreboardService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "total": 0, "data": []map[string]any{}})
		return
	}
	entries := service.PracticeLeaderboard(20)
	data := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		data = append(data, map[string]any{
			"rank":         entry.Rank,
			"user_id":      entry.UserID,
			"username":     entry.Username,
			"solved_count": entry.SolvedCount,
			"total_points": entry.TotalPoints,
		})
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"total":   len(data),
		"data":    data,
	})
}

func (srv *Server) playerVulnLabsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	keyword := queryStr(r, "keyword")
	severity := queryStr(r, "severity")

	db := srv.db.Model(&models.VulnLab{}).Where("is_public = ? AND is_active = ?", true, true)
	if keyword != "" {
		db = db.Where("name LIKE ? OR cve_id LIKE ? OR summary LIKE ? OR description LIKE ? OR reproduce_steps LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if severity != "" {
		db = db.Where("severity = ?", severity)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	var items []models.VulnLab
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&items).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	labService := srv.getVulnLabService()
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, labService.ToRow(item))
	}

	if user, ok := srv.currentUserFromRequest(r); ok {
		activeGroups := make(map[uint]map[string]any)
		currentGroups, _, _ := srv.GetContainerGroupService().ListContainerGroups(1, 1000, map[string][]string{})
		for _, group := range currentGroups {
			if uintValue(group["user_id"]) != user.ID {
				continue
			}
			status := cleanString(group["status"])
			if status != "running" && status != "created" && status != "creating" {
				continue
			}
			labID := uintValue(group["vuln_lab_id"])
			if labID == 0 {
				continue
			}
			activeGroups[labID] = challengeContainerGroupToInfo(containerGroupFromRow(group))
		}
		for _, row := range rows {
			row["runtime"] = activeGroups[uintValue(row["id"])]
		}
		sort.SliceStable(rows, func(i, j int) bool {
			leftRuntime := rows[i]["runtime"] != nil
			rightRuntime := rows[j]["runtime"] != nil
			if leftRuntime == rightRuntime {
				return false
			}
			return leftRuntime && !rightRuntime
		})
	} else {
		for _, row := range rows {
			row["runtime"] = nil
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     rows,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) playerChallengeLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	challengeID := uint(queryInt(r, "challenge_id", 0))
	service := srv.getScoreboardService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "total": 0, "data": []map[string]any{}})
		return
	}
	data := service.ChallengeLeaderboard(challengeID, 20)
	response.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"total":   len(data),
		"data":    data,
	})
}

func (srv *Server) playerMyRankHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	service := srv.getScoreboardService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"rank": 0, "user_id": user.ID, "username": user.Username})
		return
	}
	rankInfo := service.UserRank(user.ID)
	rankInfo["username"] = user.Username
	response.JSON(w, http.StatusOK, rankInfo)
}

func (srv *Server) playerTeamHandler(w http.ResponseWriter, r *http.Request) {
	if teamID := resourceID(r.URL.Path); teamID != "" {
		srv.playerTeamDetailHandler(w, r, teamID)
		return
	}
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	service := srv.getTeamService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	team, err := service.GetActiveTeamByUserID(user.ID)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]any{})
		return
	}
	row := service.ToRow(*team)
	row["members"] = service.Members(team.ID)
	row["competitions"] = []map[string]any{}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) playerTeamsListHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTeamService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": []map[string]any{}, "total": 0, "current": 1, "pageSize": 10})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := service.List(current, pageSize, r.URL.Query())
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

func (srv *Server) playerTeamDetailHandler(w http.ResponseWriter, r *http.Request, teamID string) {
	service := srv.getTeamService()
	if service == nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
		return
	}
	team, err := service.GetByID(teamID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
		return
	}
	row := service.ToRow(*team)
	row["members"] = service.Members(team.ID)
	row["competitions"] = []map[string]any{}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) playerCreateTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "Could not validate credentials"})
		return
	}
	if user.TeamID != nil && *user.TeamID != 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "already in a team, leave it first"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	name := cleanString(payload["name"])
	if name == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "name is required"})
		return
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	team := models.Team{
		Name:           name,
		Description:    cleanString(payload["description"]),
		OwnerID:        user.ID,
		InvitationCode: fmt.Sprintf("TEAM-%X", b),
	}
	if err := srv.db.Create(&team).Error; err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	_ = srv.db.Model(&models.User{}).Where("id = ?", user.ID).Update("team_id", team.ID).Error
	_ = srv.db.Preload("Owner").First(&team, team.ID).Error

	var memberCount int64
	_ = srv.db.Model(&models.User{}).Where("team_id = ? AND is_active = ?", team.ID, true).Count(&memberCount).Error
	response.JSON(w, http.StatusOK, map[string]any{
		"id":              team.ID,
		"name":            team.Name,
		"description":     team.Description,
		"owner_id":        team.OwnerID,
		"owner":           srv.playerUserResponse(team.Owner),
		"invitation_code": team.InvitationCode,
		"member_count":    memberCount,
		"created_at":      team.CreatedAt,
		"updated_at":      team.UpdatedAt,
	})
}

func (srv *Server) playerLeaveTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if ok {
		if service := srv.getUserService(); service != nil {
			_ = service.LeaveActiveTeams(user.ID, "用户主动退出", time.Now())
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "message": "已退出队伍"})
}

func playerSSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte("event: ping\ndata: ok\n\n"))
}

func readPlayerCredentials(r *http.Request) (string, string, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
			return "", "", err
		}
		return r.FormValue("username"), r.FormValue("password"), nil
	}
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	return payload.Username, payload.Password, nil
}

func readOptionalJSONMap(r *http.Request) (map[string]any, bool) {
	if r.Body == nil || r.ContentLength == 0 {
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return map[string]any{}, false
	}
	return payload, true
}

func defaultPlayerUser(username string) map[string]any {
	return map[string]any{
		"id":          1,
		"username":    username,
		"email":       username + "@cloudctf.local",
		"full_name":   username,
		"role":        "player",
		"is_active":   true,
		"team_id":     1,
		"team_name":   "Red Team",
		"permissions": []string{},
	}
}

func (srv *Server) playerCompetitionsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getCompetitionService().List(current, pageSize, r.URL.Query())
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

func (srv *Server) playerUserResponse(user userModel) map[string]any {
	teamID := 0
	teamName := ""
	if service := srv.getTeamService(); service != nil {
		if team, err := service.GetActiveTeamByUserID(user.ID); err == nil {
			teamID = int(team.ID)
			teamName = team.Name
		}
	}
	return map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"full_name":   user.FullName,
		"role":        user.Role.Name,
		"is_active":   user.IsActive,
		"team_id":     teamID,
		"team_name":   teamName,
		"permissions": user.Role.PermissionList(),
	}
}
