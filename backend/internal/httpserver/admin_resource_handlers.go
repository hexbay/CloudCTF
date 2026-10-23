package httpserver

import (
	"net/http"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"

	"gorm.io/gorm/clause"
)

func (srv *Server) adminChallengesHandler(w http.ResponseWriter, r *http.Request) {
	var records []models.Challenge
	var total int64
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	name := queryStr(r, "name")
	category := queryStr(r, "category")
	difficulty := queryStr(r, "difficulty")
	usage := queryStr(r, "usage")
	competitionID := queryStr(r, "competition_id")
	isActive := queryStr(r, "is_active")

	db := srv.db.Model(&models.Challenge{})
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if category != "" {
		db = db.Where("category = ?", category)
	}
	if difficulty != "" {
		db = db.Where("difficulty = ?", difficulty)
	}
	if usage != "" {
		values := strings.Split(usage, ",")
		usageValues := make([]any, len(values))
		for i, value := range values {
			usageValues[i] = value
		}
		db = db.Where(clause.IN{Column: clause.Column{Name: "usage"}, Values: usageValues})
	}
	if competitionID != "" {
		if competitionID == "null" || competitionID == "none" {
			db = db.Where("competition_id IS NULL")
		} else {
			db = db.Where("competition_id = ?", competitionID)
		}
	}
	if isActive != "" {
		db = db.Where("is_active = ?", isActive == "true")
	}
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
	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     rows,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) adminChallengeCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getChallengeService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminChallengeDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getChallengeService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getChallengeService().ToRow(*item))
}

func (srv *Server) adminChallengeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getChallengeService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminChallengeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getChallengeService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminTeamsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getTeamService().List(current, pageSize, r.URL.Query())
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

func (srv *Server) adminTeamCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getTeamService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminTeamDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getTeamService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getTeamService().ToRow(*item))
}

func (srv *Server) adminTeamUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getTeamService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminTeamDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getTeamService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminAnnouncementsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getAnnouncementService().List(current, pageSize, r.URL.Query())
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

func (srv *Server) adminAnnouncementCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getAnnouncementService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminAnnouncementDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getAnnouncementService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getAnnouncementService().ToRow(*item))
}

func (srv *Server) adminAnnouncementUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getAnnouncementService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminAnnouncementDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getAnnouncementService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminVulnLabsHandler(w http.ResponseWriter, r *http.Request) {
	var records []models.VulnLab
	var total int64
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	keyword := queryStr(r, "keyword")
	severity := queryStr(r, "severity")
	isPublic := queryStr(r, "is_public")

	db := srv.db.Model(&models.VulnLab{})
	if keyword != "" {
		db = db.Where("name LIKE ? OR cve_id LIKE ? OR summary LIKE ? OR description LIKE ? OR reproduce_steps LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if severity != "" {
		db = db.Where("severity = ?", severity)
	}
	if isPublic != "" {
		db = db.Where("is_public = ?", isPublic == "true")
	}
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&records).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     records,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) adminVulnLabCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getVulnLabService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVulnLabDetailHandler(w http.ResponseWriter, r *http.Request) {
	var item models.VulnLab
	if err := srv.db.First(&item, r.PathValue("id")).Error; err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getVulnLabService().ToRow(item))
}

func (srv *Server) adminVulnLabUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getVulnLabService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVulnLabDeleteHandler(w http.ResponseWriter, r *http.Request) {
	result := srv.db.Delete(&models.VulnLab{}, r.PathValue("id"))
	if result.Error != nil || result.RowsAffected == 0 {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminResourceRepositoriesHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getResourceRepositoryService().List(current, pageSize, r.URL.Query())
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

func (srv *Server) adminResourceRepositoryCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getResourceRepositoryService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminResourceRepositoryDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getResourceRepositoryService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getResourceRepositoryService().ToRow(*item))
}

func (srv *Server) adminResourceRepositoryUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getResourceRepositoryService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminResourceRepositoryDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getResourceRepositoryService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	var total int64
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	challengeID := queryStr(r, "challenge_id")
	userID := queryStr(r, "user_id")
	competitionID := queryStr(r, "competition_id")
	keyword := queryStr(r, "keyword")

	type subRow struct {
		ID            uint      `gorm:"column:id"`
		ChallengeID   uint      `gorm:"column:challenge_id"`
		ChallengeName string    `gorm:"column:challenge_name"`
		UserID        uint      `gorm:"column:user_id"`
		Username      string    `gorm:"column:username"`
		TeamID        *uint     `gorm:"column:team_id"`
		TeamName      string    `gorm:"column:team_name"`
		CompetitionID *uint     `gorm:"column:competition_id"`
		IsCorrect     bool      `gorm:"column:is_correct"`
		Points        int       `gorm:"column:points"`
		IPAddress     string    `gorm:"column:ip_address"`
		SubmittedAt   time.Time `gorm:"column:submitted_at"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}
	var items []subRow
	db := srv.db.Table("submission").
		Select("submission.*, challenge.name AS challenge_name, user.username AS username, team.name AS team_name").
		Joins("LEFT JOIN challenge ON challenge.id = submission.challenge_id").
		Joins("LEFT JOIN user ON user.id = submission.user_id").
		Joins("LEFT JOIN team ON team.id = submission.team_id")
	if challengeID != "" {
		db = db.Where("submission.challenge_id = ?", challengeID)
	}
	if userID != "" {
		db = db.Where("submission.user_id = ?", userID)
	}
	if competitionID != "" {
		if competitionID == "null" {
			db = db.Where("submission.competition_id IS NULL")
		} else {
			db = db.Where("submission.competition_id = ?", competitionID)
		}
	}
	if keyword != "" {
		db = db.Where("(user.username LIKE ? OR challenge.name LIKE ? OR team.name LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("submission.id DESC").Find(&items).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"id":             item.ID,
			"challenge_id":   item.ChallengeID,
			"challenge_name": item.ChallengeName,
			"user_id":        item.UserID,
			"username":       item.Username,
			"team_id":        item.TeamID,
			"team_name":      item.TeamName,
			"competition_id": item.CompetitionID,
			"is_correct":     item.IsCorrect,
			"points":         item.Points,
			"ip_address":     item.IPAddress,
			"submitted_at":   item.SubmittedAt,
			"created_at":     item.CreatedAt,
		})
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminSubmissionCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getSubmissionService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getAuditLogService().List(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminTheoryQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getTheoryService().List(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminTheoryQuestionCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getTheoryService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminTheoryQuestionDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getTheoryService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": theoryToRow(*item)})
}

func (srv *Server) adminTheoryQuestionUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getTheoryService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminTheoryQuestionDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getTheoryService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminExamsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getExamService().List(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminExamResourceCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getExamService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminExamResourceUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getExamService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminExamResourceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getExamService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminCompetitionsHandler(w http.ResponseWriter, r *http.Request) {
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

func (srv *Server) adminCompetitionCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getCompetitionService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminCompetitionDetailHandler(w http.ResponseWriter, r *http.Request) {
	item, err := srv.getCompetitionService().GetByID(r.PathValue("id"))
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getCompetitionService().ToRow(*item))
}

func (srv *Server) adminCompetitionUpdateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getCompetitionService().Update(r.PathValue("id"), payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminCompetitionDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.getCompetitionService().Delete(r.PathValue("id")) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}
