package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type ScoreboardService struct {
	db *gorm.DB
}

func NewScoreboardService(db *gorm.DB) *ScoreboardService {
	return &ScoreboardService{db: db}
}

func (s *ScoreboardService) Create(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	competitionID := fmt.Sprint(row["competition_id"])
	if competitionID == "" {
		return nil, fmt.Errorf("competition_id is required")
	}
	rows, _, err := s.listCompetitionScoreboard(competitionID, 1, 1000000)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success":        true,
		"competition_id": competitionID,
		"data":           rows,
	}, nil
}

type PracticeLeaderboardEntry struct {
	Rank        int     `json:"rank"`
	UserID      uint    `json:"user_id"`
	Username    string  `json:"username"`
	SolvedCount int     `json:"solved_count"`
	TotalPoints float64 `json:"total_points"`
}

type competitionLeaderboardRow struct {
	ID             uint
	Name           string
	SolvedCount    int
	TotalPoints    int
	LastSubmission *time.Time
}

type theorySubmissionLeaderboardRow struct {
	UserID      uint
	TeamID      *uint
	Score       int
	Correct     int
	SubmittedAt time.Time
}

func (s *ScoreboardService) PracticeLeaderboard(limit int) []PracticeLeaderboardEntry {
	type row struct {
		UserID      uint
		SolvedCount int
		TotalPoints float64
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("user_id, COUNT(DISTINCT challenge_id) as solved_count, SUM(points) as total_points").
		Where("is_correct = ? AND competition_id IS NULL", true).
		Group("user_id").
		Order("total_points DESC, solved_count DESC").
		Limit(limit).
		Find(&results).Error
	entries := make([]PracticeLeaderboardEntry, 0, len(results))
	for i, r := range results {
		var user models.User
		username := fmt.Sprintf("user_%d", r.UserID)
		if err := s.db.First(&user, r.UserID).Error; err == nil {
			username = user.Username
		}
		entries = append(entries, PracticeLeaderboardEntry{
			Rank:        i + 1,
			UserID:      r.UserID,
			Username:    username,
			SolvedCount: r.SolvedCount,
			TotalPoints: r.TotalPoints,
		})
	}
	return entries
}

func (s *ScoreboardService) ChallengeLeaderboard(challengeID uint, limit int) []map[string]any {
	if challengeID == 0 {
		type aggregateRow struct {
			ChallengeID      uint
			ChallengeName    string
			Category         string
			Difficulty       int
			Points           int
			SolvedCount      int64
			TotalSubmissions int64
		}
		var rows []aggregateRow
		_ = s.db.Model(&models.Challenge{}).
			Select(`
				challenge.id as challenge_id,
				challenge.name as challenge_name,
				challenge.category,
				challenge.difficulty,
				challenge.points,
				COUNT(DISTINCT CASE WHEN submission.is_correct = 1 AND submission.competition_id IS NULL THEN submission.user_id END) as solved_count,
				COUNT(submission.id) as total_submissions
			`).
			Joins("LEFT JOIN submission ON submission.challenge_id = challenge.id AND submission.competition_id IS NULL").
			Where("challenge.is_active = ? AND challenge.is_public = ? AND challenge.competition_id IS NULL", true, true).
			Group("challenge.id, challenge.name, challenge.category, challenge.difficulty, challenge.points").
			Order("solved_count DESC, total_submissions DESC, challenge.points DESC, challenge.id ASC").
			Limit(limit).
			Scan(&rows).Error

		result := make([]map[string]any, 0, len(rows))
		for i, row := range rows {
			result = append(result, map[string]any{
				"rank":              i + 1,
				"challenge_id":      row.ChallengeID,
				"challenge_name":    row.ChallengeName,
				"category":          row.Category,
				"difficulty":        row.Difficulty,
				"points":            row.Points,
				"solved_count":      row.SolvedCount,
				"total_submissions": row.TotalSubmissions,
			})
		}
		return result
	}

	var challenge models.Challenge
	if s.db.First(&challenge, challengeID).Error != nil {
		return []map[string]any{}
	}
	var solvedCount int64
	_ = s.db.Model(&models.Submission{}).Where("challenge_id = ? AND is_correct = ? AND competition_id IS NULL", challengeID, true).Distinct("user_id").Count(&solvedCount).Error
	var totalSubmissions int64
	_ = s.db.Model(&models.Submission{}).Where("challenge_id = ? AND competition_id IS NULL", challengeID).Count(&totalSubmissions).Error
	return []map[string]any{{
		"rank":              1,
		"challenge_id":      challenge.ID,
		"challenge_name":    challenge.Name,
		"category":          challenge.Category,
		"difficulty":        challenge.Difficulty,
		"points":            challenge.Points,
		"solved_count":      solvedCount,
		"total_submissions": totalSubmissions,
	}}
}

func (s *ScoreboardService) UserRank(userID uint) map[string]any {
	type row struct {
		UserID      uint
		TotalPoints float64
		SolvedCount int
	}
	var results []row
	_ = s.db.Model(&models.Submission{}).
		Select("user_id, SUM(points) as total_points, COUNT(DISTINCT challenge_id) as solved_count").
		Where("is_correct = ? AND competition_id IS NULL", true).
		Group("user_id").
		Order("total_points DESC, solved_count DESC").
		Find(&results).Error
	for i, r := range results {
		if r.UserID == userID {
			return map[string]any{
				"rank":         i + 1,
				"user_id":      userID,
				"solved_count": r.SolvedCount,
				"total_points": r.TotalPoints,
			}
		}
	}
	return map[string]any{
		"rank":         0,
		"user_id":      userID,
		"solved_count": 0,
		"total_points": 0,
	}
}

func (s *ScoreboardService) UserStats(userID uint) map[string]any {
	var totalPoints float64
	var solvedCount int64
	_ = s.db.Model(&models.Submission{}).
		Where("user_id = ? AND is_correct = ? AND competition_id IS NULL", userID, true).
		Select("COALESCE(SUM(points), 0)").
		Row().Scan(&totalPoints)
	_ = s.db.Model(&models.Submission{}).
		Where("user_id = ? AND is_correct = ? AND competition_id IS NULL", userID, true).
		Distinct("challenge_id").
		Count(&solvedCount).Error
	rankInfo := s.UserRank(userID)
	return map[string]any{
		"total_points":      totalPoints,
		"solved_challenges": solvedCount,
		"rank":              rankInfo["rank"],
	}
}

func (s *ScoreboardService) AddCompetitionScore(competitionID, userID uint, teamID *uint, points float64) {
	// Competition scoreboard is now computed from submission in real time.
}

func (s *ScoreboardService) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	competitionID := scoreFirstQuery(query, "competition_id")
	if competitionID == "" {
		return []map[string]any{}, 0, nil
	}
	return s.listCompetitionScoreboard(competitionID, current, pageSize)
}

func (s *ScoreboardService) CountParticipants(competitionID any) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	rows, total, err := s.listCompetitionScoreboard(fmt.Sprint(competitionID), 1, 1000000)
	if err != nil {
		return 0
	}
	if len(rows) == 0 {
		return 0
	}
	return int64(total)
}

func (s *ScoreboardService) ListByCompetition(competitionID any, limit int) ([]map[string]any, int) {
	if s == nil || s.db == nil {
		return []map[string]any{}, 0
	}
	rows, total, err := s.listCompetitionScoreboard(fmt.Sprint(competitionID), 1, limit)
	if err != nil {
		return []map[string]any{}, 0
	}
	return rows, total
}

func (s *ScoreboardService) listCompetitionScoreboard(competitionID string, current, pageSize int) ([]map[string]any, int, error) {
	if competitionID == "" {
		return []map[string]any{}, 0, nil
	}

	var competition models.Competition
	if err := s.db.First(&competition, competitionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []map[string]any{}, 0, nil
		}
		return nil, 0, err
	}

	isTeamMode := isCompetitionTeamMode(competition.ParticipantMode)
	isTheoryCompetition := strings.EqualFold(strings.TrimSpace(competition.Type), "theory")

	var leaderboard []competitionLeaderboardRow
	var err error
	if isTheoryCompetition {
		if isTeamMode {
			leaderboard, err = s.buildCompetitionTheoryTeamLeaderboard(competition.ID)
		} else {
			leaderboard, err = s.buildCompetitionTheoryUserLeaderboard(competition.ID)
		}
	} else {
		if isTeamMode {
			leaderboard, err = s.buildCompetitionTeamLeaderboard(competition.ID)
		} else {
			leaderboard, err = s.buildCompetitionUserLeaderboard(competition.ID)
		}
	}
	if err != nil {
		return nil, 0, err
	}

	total := len(leaderboard)
	if pageSize <= 0 {
		pageSize = total
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if current <= 0 {
		current = 1
	}

	start := (current - 1) * pageSize
	if start >= total {
		return []map[string]any{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	result := make([]map[string]any, 0, end-start)
	for index, row := range leaderboard[start:end] {
		rank := start + index + 1
		item := map[string]any{
			"rank":         rank,
			"solved_count": row.SolvedCount,
			"total_points": row.TotalPoints,
		}
		if isTeamMode {
			item["team_id"] = row.ID
			item["team_name"] = row.Name
			item["user_id"] = row.ID
			item["username"] = row.Name
		} else {
			item["user_id"] = row.ID
			item["username"] = row.Name
		}
		result = append(result, item)
	}
	return result, total, nil
}

func (s *ScoreboardService) buildCompetitionUserLeaderboard(competitionID uint) ([]competitionLeaderboardRow, error) {
	type solvedSubmission struct {
		UserID      uint
		ChallengeID uint
		SubmittedAt time.Time
		Points      int
	}
	var submissions []solvedSubmission
	if err := s.db.Model(&models.Submission{}).
		Select("user_id, challenge_id, submitted_at, points").
		Where("competition_id = ? AND is_correct = ?", competitionID, true).
		Order("submitted_at ASC").
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	type aggregate struct {
		solved   map[uint]struct{}
		points   int
		lastTime *time.Time
	}
	userAgg := map[uint]*aggregate{}
	for _, submission := range submissions {
		agg := userAgg[submission.UserID]
		if agg == nil {
			agg = &aggregate{solved: map[uint]struct{}{}}
			userAgg[submission.UserID] = agg
		}
		if _, exists := agg.solved[submission.ChallengeID]; exists {
			continue
		}
		agg.solved[submission.ChallengeID] = struct{}{}
		agg.points += submission.Points
		submittedAt := submission.SubmittedAt
		agg.lastTime = &submittedAt
	}

	if len(userAgg) == 0 {
		return []competitionLeaderboardRow{}, nil
	}

	userIDs := make([]uint, 0, len(userAgg))
	for userID := range userAgg {
		userIDs = append(userIDs, userID)
	}
	var users []models.User
	if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	usernames := map[uint]string{}
	for _, user := range users {
		usernames[user.ID] = user.Username
	}

	rows := make([]competitionLeaderboardRow, 0, len(userAgg))
	for userID, agg := range userAgg {
		rows = append(rows, competitionLeaderboardRow{
			ID:             userID,
			Name:           scoreDisplayName(usernames[userID], userID, "user"),
			SolvedCount:    len(agg.solved),
			TotalPoints:    agg.points,
			LastSubmission: agg.lastTime,
		})
	}

	sortCompetitionRows(rows)
	return rows, nil
}

func (s *ScoreboardService) buildCompetitionTeamLeaderboard(competitionID uint) ([]competitionLeaderboardRow, error) {
	type solvedSubmission struct {
		TeamID      *uint
		UserID      uint
		ChallengeID uint
		SubmittedAt time.Time
		Points      int
	}
	var submissions []solvedSubmission
	if err := s.db.Model(&models.Submission{}).
		Select("team_id, user_id, challenge_id, submitted_at, points").
		Where("competition_id = ? AND is_correct = ? AND team_id IS NOT NULL", competitionID, true).
		Order("submitted_at ASC").
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	type aggregate struct {
		solved   map[uint]struct{}
		points   int
		lastTime *time.Time
	}
	teamAgg := map[uint]*aggregate{}
	for _, submission := range submissions {
		if submission.TeamID == nil || *submission.TeamID == 0 {
			continue
		}
		teamID := *submission.TeamID
		agg := teamAgg[teamID]
		if agg == nil {
			agg = &aggregate{solved: map[uint]struct{}{}}
			teamAgg[teamID] = agg
		}
		if _, exists := agg.solved[submission.ChallengeID]; exists {
			continue
		}
		agg.solved[submission.ChallengeID] = struct{}{}
		agg.points += submission.Points
		submittedAt := submission.SubmittedAt
		agg.lastTime = &submittedAt
	}

	if len(teamAgg) == 0 {
		return []competitionLeaderboardRow{}, nil
	}

	teamIDs := make([]uint, 0, len(teamAgg))
	for teamID := range teamAgg {
		teamIDs = append(teamIDs, teamID)
	}
	var teams []models.Team
	if err := s.db.Where("id IN ?", teamIDs).Find(&teams).Error; err != nil {
		return nil, err
	}
	teamNames := map[uint]string{}
	for _, team := range teams {
		teamNames[team.ID] = team.Name
	}

	rows := make([]competitionLeaderboardRow, 0, len(teamAgg))
	for teamID, agg := range teamAgg {
		rows = append(rows, competitionLeaderboardRow{
			ID:             teamID,
			Name:           scoreDisplayName(teamNames[teamID], teamID, "team"),
			SolvedCount:    len(agg.solved),
			TotalPoints:    agg.points,
			LastSubmission: agg.lastTime,
		})
	}

	sortCompetitionRows(rows)
	return rows, nil
}

func (s *ScoreboardService) buildCompetitionTheoryUserLeaderboard(competitionID uint) ([]competitionLeaderboardRow, error) {
	var submissions []theorySubmissionLeaderboardRow
	if err := s.db.Model(&models.ExamSubmission{}).
		Select("user_id, score, correct, submitted_at").
		Where("competition_id = ?", competitionID).
		Order("submitted_at ASC, id ASC").
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	if len(submissions) == 0 {
		return []competitionLeaderboardRow{}, nil
	}

	bestByUser := map[uint]theorySubmissionLeaderboardRow{}
	for _, submission := range submissions {
		existing, ok := bestByUser[submission.UserID]
		if !ok || theorySubmissionBetter(submission, existing) {
			bestByUser[submission.UserID] = submission
		}
	}

	userIDs := make([]uint, 0, len(bestByUser))
	for userID := range bestByUser {
		userIDs = append(userIDs, userID)
	}
	var users []models.User
	if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	usernames := map[uint]string{}
	for _, user := range users {
		usernames[user.ID] = user.Username
	}

	rows := make([]competitionLeaderboardRow, 0, len(bestByUser))
	for userID, submission := range bestByUser {
		submittedAt := submission.SubmittedAt
		rows = append(rows, competitionLeaderboardRow{
			ID:             userID,
			Name:           scoreDisplayName(usernames[userID], userID, "user"),
			SolvedCount:    submission.Correct,
			TotalPoints:    submission.Score,
			LastSubmission: &submittedAt,
		})
	}

	sortCompetitionRows(rows)
	return rows, nil
}

func (s *ScoreboardService) buildCompetitionTheoryTeamLeaderboard(competitionID uint) ([]competitionLeaderboardRow, error) {
	var submissions []theorySubmissionLeaderboardRow
	if err := s.db.Model(&models.ExamSubmission{}).
		Select("team_id, score, correct, submitted_at").
		Where("competition_id = ? AND team_id IS NOT NULL", competitionID).
		Order("submitted_at ASC, id ASC").
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	if len(submissions) == 0 {
		return []competitionLeaderboardRow{}, nil
	}

	bestByTeam := map[uint]theorySubmissionLeaderboardRow{}
	for _, submission := range submissions {
		if submission.TeamID == nil || *submission.TeamID == 0 {
			continue
		}
		teamID := *submission.TeamID
		existing, ok := bestByTeam[teamID]
		if !ok || theorySubmissionBetter(submission, existing) {
			bestByTeam[teamID] = submission
		}
	}

	if len(bestByTeam) == 0 {
		return []competitionLeaderboardRow{}, nil
	}

	teamIDs := make([]uint, 0, len(bestByTeam))
	for teamID := range bestByTeam {
		teamIDs = append(teamIDs, teamID)
	}
	var teams []models.Team
	if err := s.db.Where("id IN ?", teamIDs).Find(&teams).Error; err != nil {
		return nil, err
	}
	teamNames := map[uint]string{}
	for _, team := range teams {
		teamNames[team.ID] = team.Name
	}

	rows := make([]competitionLeaderboardRow, 0, len(bestByTeam))
	for teamID, submission := range bestByTeam {
		submittedAt := submission.SubmittedAt
		rows = append(rows, competitionLeaderboardRow{
			ID:             teamID,
			Name:           scoreDisplayName(teamNames[teamID], teamID, "team"),
			SolvedCount:    submission.Correct,
			TotalPoints:    submission.Score,
			LastSubmission: &submittedAt,
		})
	}

	sortCompetitionRows(rows)
	return rows, nil
}

func sortCompetitionRows(rows []competitionLeaderboardRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalPoints != rows[j].TotalPoints {
			return rows[i].TotalPoints > rows[j].TotalPoints
		}
		if rows[i].SolvedCount != rows[j].SolvedCount {
			return rows[i].SolvedCount > rows[j].SolvedCount
		}
		leftNil := rows[i].LastSubmission == nil
		rightNil := rows[j].LastSubmission == nil
		if leftNil != rightNil {
			return !leftNil
		}
		if !leftNil && !rows[i].LastSubmission.Equal(*rows[j].LastSubmission) {
			return rows[i].LastSubmission.Before(*rows[j].LastSubmission)
		}
		return rows[i].ID < rows[j].ID
	})
}

func theorySubmissionBetter(left, right theorySubmissionLeaderboardRow) bool {
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	if left.Correct != right.Correct {
		return left.Correct > right.Correct
	}
	if !left.SubmittedAt.Equal(right.SubmittedAt) {
		return left.SubmittedAt.Before(right.SubmittedAt)
	}
	return false
}

func isCompetitionTeamMode(mode string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	return normalized == "team" || normalized == "mixed"
}

func scoreDisplayName(name string, id uint, prefix string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fmt.Sprintf("%s_%d", prefix, id)
}

func scoreFirstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
