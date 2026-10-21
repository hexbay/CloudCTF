package httpserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/services"
)

// ================================================
// Competition participants
// ================================================

func (srv *Server) adminCompetitionParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := relationService.ListParticipants(uintValue(competitionID), current, pageSize)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"data":     rows,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) adminAddParticipantHandler(w http.ResponseWriter, r *http.Request) {
	userService := srv.getUserService()
	teamService := srv.getTeamService()
	if userService == nil || teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}

	payload, _ := readOptionalJSONMap(r)
	userID := uintValue(payload["user_id"])
	teamID := optionalUint(payload["team_id"])
	if userID == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "user_id required"})
		return
	}

	user, err := userService.GetByID(userID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "user not found"})
		return
	}

	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	if relationService.ParticipantExists(uintValue(competitionID), userID) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "participant already exists"})
		return
	}

	teamName := ""
	if teamID != nil {
		team, err := teamService.GetByID(*teamID)
		if err != nil {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
			return
		}
		teamName = team.Name
		if err := teamService.EnsureMember(*teamID, userID, "admin add competition participant"); err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to add team member"})
			return
		}
	}

	if _, err := relationService.CreateParticipant(map[string]any{
		"competition_id": uintValue(competitionID),
		"user_id":        user.ID,
		"username":       user.Username,
		"team_id":        teamID,
		"team_name":      teamName,
	}); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save participant"})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminRemoveParticipantHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := competitionPathID(r)
	userID := r.PathValue("user_id")
	if userID == "" {
		userID = resourceID(r.URL.Path)
	}
	if competitionID == "" || userID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id and user_id required"})
		return
	}

	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	if !relationService.DeleteParticipant(uintValue(competitionID), uintValue(userID)) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "participant not found"})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminGenerateCompetitionParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	userService := srv.getUserService()
	teamService := srv.getTeamService()
	competitionService := srv.getCompetitionService()
	if userService == nil || teamService == nil || competitionService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}

	payload, _ := readOptionalJSONMap(r)
	teamCount := intValueOr(payload["team_count"], 0)
	membersPerTeam := intValueOr(payload["members_per_team"], 0)
	schoolName := cleanString(payload["school_name"])
	teamPrefix := cleanString(payload["team_prefix"])
	usernamePrefix := cleanString(payload["username_prefix"])
	defaultPassword := cleanString(payload["default_password"])

	if teamCount <= 0 || membersPerTeam <= 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_count and members_per_team must be greater than 0"})
		return
	}
	if teamPrefix == "" {
		if schoolName != "" {
			teamPrefix = schoolName
		} else {
			teamPrefix = "Team"
		}
	}
	if usernamePrefix == "" {
		usernamePrefix = sanitizeIdentifier(teamPrefix)
	}
	if defaultPassword == "" {
		defaultPassword = "CloudCTF123!"
	}

	competition, err := competitionService.GetByID(competitionID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "competition not found"})
		return
	}

	playerRole, err := userService.GetRoleByName(rolePlayer)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "player role not found"})
		return
	}

	createdUsers := 0
	teams := make([]map[string]any, 0, teamCount)
	for teamIndex := 1; teamIndex <= teamCount; teamIndex++ {
		teamName := uniqueTeamName(teamService, fmt.Sprintf("%s-%02d", teamPrefix, teamIndex))
		leaderUsername := uniqueUsername(userService, fmt.Sprintf("%s_%02d_%02d", usernamePrefix, teamIndex, 1))
		leader, err := ensureGeneratedUser(userService, leaderUsername, defaultPassword, fmt.Sprintf("%s@autogen.local", leaderUsername), fmt.Sprintf("%s Leader", teamName), &playerRole.ID)
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create leader"})
			return
		}
		createdUsers++

		teamRow, err := srv.getTeamService().Create(map[string]any{
			"name":        teamName,
			"description": fmt.Sprintf("%s generated team", competition.Name),
			"owner_id":    leader.ID,
		})
		if err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create team"})
			return
		}
		teamID := uintValue(teamRow["id"])
		memberUsernames := []string{leader.Username}

		for memberIndex := 2; memberIndex <= membersPerTeam; memberIndex++ {
			memberUsername := uniqueUsername(userService, fmt.Sprintf("%s_%02d_%02d", usernamePrefix, teamIndex, memberIndex))
			member, createErr := ensureGeneratedUser(userService, memberUsername, defaultPassword, fmt.Sprintf("%s@autogen.local", memberUsername), fmt.Sprintf("%s Member %d", teamName, memberIndex-1), &playerRole.ID)
			if createErr != nil {
				response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create member"})
				return
			}
			createdUsers++
			if err := teamService.EnsureMember(teamID, member.ID, "generated competition member"); err != nil {
				response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to add member"})
				return
			}
			memberUsernames = append(memberUsernames, member.Username)
		}

		for _, username := range memberUsernames {
			member, err := userService.GetByUsername(username)
			if err != nil {
				continue
			}
			relationService := srv.getCompetitionRelationService()
			if relationService == nil {
				response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
				return
			}
			if relationService.ParticipantExists(competition.ID, member.ID) {
				continue
			}
			if _, err := relationService.CreateParticipant(map[string]any{
				"competition_id": competition.ID,
				"user_id":        member.ID,
				"username":       member.Username,
				"team_id":        teamID,
				"team_name":      teamName,
			}); err != nil {
				response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save generated participants"})
				return
			}
		}

		teams = append(teams, map[string]any{
			"team_id":          teamID,
			"team_name":        teamName,
			"member_usernames": memberUsernames,
		})
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":          true,
		"created_teams":    len(teams),
		"created_users":    createdUsers,
		"default_password": defaultPassword,
		"teams":            teams,
	})
}

// ================================================
// Competition status
// ================================================

func (srv *Server) adminCompetitionStatusHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	status := cleanString(payload["status"])
	if status == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "status required"})
		return
	}
	service := srv.getCompetitionService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	comp, err := service.GetByID(competitionID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "competition not found"})
		return
	}
	if _, ok := service.Update(comp.ID, map[string]any{"status": status}); !ok {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to update competition status"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "status": status})
}

// ================================================
// Competition admin management
// ================================================

func (srv *Server) adminCompetitionAdminsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	userService := srv.getUserService()
	if userService == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	relationService := srv.getCompetitionRelationService()
	if relationService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	bindings, err := relationService.ListAdmins(uintValue(competitionID))
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	ids := make([]uint, 0, len(bindings))
	for _, binding := range bindings {
		if id := binding.UserID; id != 0 {
			ids = append(ids, id)
		}
	}
	modelAdmins := userService.ListCompetitionAdmins(ids)
	admins := make([]userModel, 0, len(modelAdmins))
	for _, admin := range modelAdmins {
		admins = append(admins, userModel(admin))
	}
	rows := make([]map[string]any, 0, len(admins))
	for _, a := range admins {
		rows = append(rows, userToRow(a))
	}
	response.JSON(w, http.StatusOK, rows)
}

func (srv *Server) adminCompetitionSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	competitionID := competitionPathID(r)
	if competitionID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "competition_id required"})
		return
	}
	service := srv.getSubmissionService()
	if service == nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": []map[string]any{}, "total": 0})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 100)
	rows, total, err := service.CompetitionRows(competitionID, current, pageSize)
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

// ================================================
// Team management
// ================================================

func (srv *Server) adminTeamAddMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamService := srv.getTeamService()
	if teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	teamID := resourceID(r.URL.Path)
	payload, _ := readOptionalJSONMap(r)
	userID := uintValue(payload["user_id"])
	if userID == 0 || teamID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_id and user_id required"})
		return
	}
	if teamService.HasActiveMembership(userID) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "user already belongs to another active team"})
		return
	}
	if err := ensureTeamMember(teamService, uintValue(teamID), userID, "admin add team member"); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to add team member"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminTeamRemoveMemberHandler(w http.ResponseWriter, r *http.Request) {
	teamService := srv.getTeamService()
	if teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	userID := resourceID(r.URL.Path)
	if userID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "user_id required"})
		return
	}
	_ = teamService.RemoveActiveMembership(uintValue(userID), "admin remove team member")
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminTeamBlacklistHandler(w http.ResponseWriter, r *http.Request) {
	teamService := srv.getTeamService()
	if teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	teamID := resourceID(r.URL.Path)
	if teamID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "team_id required"})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	team, err := teamService.GetByID(teamID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
		return
	}
	now := time.Now()
	_, _ = teamService.Update(team.ID, map[string]any{
		"is_blacklisted":   true,
		"blacklist_reason": cleanString(payload["reason"]),
	})
	team.BlacklistedAt = &now
	team.BlacklistedBy = optionalUint(payload["blacklisted_by"])
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminTeamUnblacklistHandler(w http.ResponseWriter, r *http.Request) {
	teamService := srv.getTeamService()
	if teamService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "db not available"})
		return
	}
	teamID := resourceID(r.URL.Path)
	team, err := teamService.GetByID(teamID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "team not found"})
		return
	}
	_, _ = teamService.Update(team.ID, map[string]any{
		"is_blacklisted":   false,
		"blacklist_reason": "",
	})
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminTeamImportHandler(w http.ResponseWriter, r *http.Request) {
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

	service := srv.getTeamService()
	imported := 0
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		parts := strings.Split(strings.TrimSpace(line), ",")
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		if service != nil {
			row := map[string]any{"name": strings.TrimSpace(parts[0]), "description": ""}
			if len(parts) > 1 {
				row["description"] = strings.TrimSpace(parts[1])
			}
			if _, err := service.Create(row); err == nil {
				imported++
			}
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "imported": imported})
}

func (srv *Server) adminTeamExportHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getTeamService()
	if service == nil {
		writeExport(w, "teams")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="teams.csv"`)
	_, _ = w.Write([]byte("name,owner,member_count,created_at\n"))
	for _, team := range service.ExportRows() {
		createdAt, _ := team["created_at"].(time.Time)
		line := fmt.Sprintf("%s,%s,%d,%s\n", stringifyID(team["name"]), stringifyID(team["owner"]), intValue(team["member_count"], 0), createdAt.Format(time.RFC3339))
		_, _ = w.Write([]byte(line))
	}
}

// ================================================
// Competition registration (player side helper)
// ================================================

func competitionPathID(r *http.Request) string {
	if id := r.PathValue("id"); id != "" {
		return id
	}
	return resourceID(r.URL.Path)
}

func intValueOr(value any, fallback int) int {
	text := cleanString(value)
	if text == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return fallback
	}
	return parsed
}

func sanitizeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(" ", "_", "-", "_").Replace(value)
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "player"
	}
	return builder.String()
}

func uniqueUsername(service *services.UserService, base string) string {
	base = sanitizeIdentifier(base)
	candidate := base
	index := 1
	for {
		if !service.UsernameExists(candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, index)
		index++
	}
}

func uniqueTeamName(service *services.TeamService, base string) string {
	candidate := strings.TrimSpace(base)
	index := 1
	for {
		if !service.NameExists(candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", strings.TrimSpace(base), index)
		index++
	}
}

func ensureGeneratedUser(service *services.UserService, username, password, email, fullName string, roleID *uint) (*userModel, error) {
	user, err := service.EnsureGeneratedUser(username, email, password, fullName, roleID)
	if err != nil {
		return nil, err
	}
	typed := userModel(*user)
	return &typed, nil
}

func ensureTeamMember(service *services.TeamService, teamID, userID uint, reason string) error {
	return service.EnsureMember(teamID, userID, reason)
}
