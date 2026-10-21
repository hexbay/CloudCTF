package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"
)

type PlayerChallengeDetailInput struct {
	ChallengeID uint
	UserID      uint
}

type PurchaseChallengeHintInput struct {
	UserID      uint
	ChallengeID uint
	HintIndex   int
}

func (s *ChallengeService) PlayerDetail(input PlayerChallengeDetailInput) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if input.ChallengeID == 0 {
		return nil, fmt.Errorf("%w: challenge_id required", ErrInvalidInput)
	}

	challenge, err := s.GetByID(input.ChallengeID)
	if err != nil {
		return nil, ErrNotFound
	}
	if !playerAccessibleChallengeModel(*challenge) {
		return nil, fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}

	row := playerChallengeDetailBaseRow(*challenge)
	if challenge.CompetitionID != nil {
		row["bloods"] = []map[string]any{}
		row["solves_count"] = 0
	} else {
		row["bloods"] = s.practiceChallengeBloods(challenge.ID, 3)
		row["solves_count"] = s.practiceChallengeSolvedCount(challenge.ID)
	}
	row["solved"] = false
	row["user_points"] = 0
	row["container_info"] = nil

	if input.UserID != 0 {
		if solved, found := s.solvedByUserWithCompetition(challenge.ID, input.UserID, challenge.CompetitionID); found {
			row["solved"] = true
			row["solved_at"] = solved.SubmittedAt
			row["user_points"] = challenge.Points
		}
		if challenge.IsDynamic {
			if group, found := s.activeChallengeContainerGroup(challenge.ID, input.UserID); found {
				row["container_info"] = challengeContainerGroupInfo(group)
			}
		}
		row["hints"] = s.practiceHintsForUser(*challenge, input.UserID)
	} else {
		row["hints"] = practiceHintsWithoutPurchase(*challenge)
	}
	return row, nil
}

func (s *ChallengeService) PurchaseHint(input PurchaseChallengeHintInput) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.ChallengeID == 0 {
		return nil, fmt.Errorf("%w: user_id and challenge_id are required", ErrInvalidInput)
	}
	if input.HintIndex < 0 {
		return nil, fmt.Errorf("%w: invalid hint index", ErrInvalidInput)
	}

	challenge, err := s.GetByID(input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("%w: challenge not found", ErrNotFound)
	}
	if !playerAccessibleChallengeModel(*challenge) {
		return nil, fmt.Errorf("%w: challenge is not available", ErrForbidden)
	}
	hints := challenge.HintList()
	if input.HintIndex >= len(hints) {
		return nil, fmt.Errorf("%w: hint not found", ErrNotFound)
	}

	hint := hints[input.HintIndex]
	cost := subIntValue(hint["cost"], 0)
	var purchase models.HintPurchase
	if s.db.Where("user_id = ? AND challenge_id = ? AND hint_index = ?", input.UserID, challenge.ID, input.HintIndex).First(&purchase).Error != nil {
		purchase = models.HintPurchase{
			UserID:      input.UserID,
			ChallengeID: challenge.ID,
			HintIndex:   input.HintIndex,
			Cost:        cost,
		}
		if err := s.db.Create(&purchase).Error; err != nil {
			return nil, err
		}
	}

	return map[string]any{
		"success": true,
		"message": "提示已解锁",
		"hint": map[string]any{
			"index":     input.HintIndex,
			"content":   fmt.Sprint(hint["content"]),
			"cost":      cost,
			"purchased": true,
		},
	}, nil
}

func playerChallengeDetailBaseRow(challenge models.Challenge) map[string]any {
	return map[string]any{
		"id":                     challenge.ID,
		"name":                   challenge.Name,
		"description":            challenge.Description,
		"category":               challenge.Category,
		"points":                 challenge.Points,
		"difficulty":             challenge.Difficulty,
		"is_active":              challenge.IsActive,
		"is_public":              challenge.IsPublic,
		"usage":                  challenge.Usage,
		"is_dynamic":             challenge.IsDynamic,
		"container_image":        challenge.ContainerImage,
		"container_port":         challenge.ContainerPort,
		"container_timeout":      challenge.ContainerTimeout,
		"host_id":                challenge.HostID,
		"template_id":            challenge.TemplateID,
		"competition_id":         challenge.CompetitionID,
		"source_challenge_id":    challenge.SourceChallengeID,
		"level_one_solution":     challenge.LevelOneSolution,
		"has_level_one_solution": strings.TrimSpace(challenge.LevelOneSolution) != "",
		"files":                  challenge.FileList(),
		"attachments":            challenge.FileList(),
		"created_by":             challenge.CreatedBy,
		"created_at":             challenge.CreatedAt,
		"updated_at":             challenge.UpdatedAt,
	}
}

func (s *ChallengeService) solvedByUserWithCompetition(challengeID, userID uint, competitionID *uint) (*models.Submission, bool) {
	var existing models.Submission
	query := s.db.Where("challenge_id = ? AND user_id = ? AND is_correct = ?", challengeID, userID, true)
	if competitionID != nil {
		query = query.Where("competition_id = ?", *competitionID)
	} else {
		query = query.Where("competition_id IS NULL")
	}
	if query.Order("submitted_at asc").First(&existing).Error != nil {
		return nil, false
	}
	return &existing, true
}

func (s *ChallengeService) practiceHintsForUser(challenge models.Challenge, userID uint) []map[string]any {
	rawHints := challenge.HintList()
	var purchases []models.HintPurchase
	_ = s.db.Where("user_id = ? AND challenge_id = ?", userID, challenge.ID).Find(&purchases).Error
	purchased := make(map[int]bool, len(purchases))
	for _, row := range purchases {
		purchased[row.HintIndex] = true
	}
	result := make([]map[string]any, 0, len(rawHints))
	for index, hint := range rawHints {
		result = append(result, map[string]any{
			"index":     index,
			"content":   fmt.Sprint(hint["content"]),
			"cost":      subIntValue(hint["cost"], 0),
			"purchased": purchased[index],
		})
	}
	return result
}

func practiceHintsWithoutPurchase(challenge models.Challenge) []map[string]any {
	rawHints := challenge.HintList()
	result := make([]map[string]any, 0, len(rawHints))
	for index, hint := range rawHints {
		result = append(result, map[string]any{
			"index":     index,
			"content":   fmt.Sprint(hint["content"]),
			"cost":      subIntValue(hint["cost"], 0),
			"purchased": false,
		})
	}
	return result
}

func (s *ChallengeService) practiceChallengeBloods(challengeID uint, limit int) []map[string]any {
	var submissions []models.Submission
	_ = s.db.Where("challenge_id = ? AND is_correct = ? AND competition_id IS NULL", challengeID, true).
		Order("submitted_at asc").
		Find(&submissions).Error
	result := make([]map[string]any, 0, limit)
	seen := map[uint]bool{}
	for _, submission := range submissions {
		if seen[submission.UserID] {
			continue
		}
		seen[submission.UserID] = true
		username := fmt.Sprintf("user_%d", submission.UserID)
		var user models.User
		if err := s.db.First(&user, submission.UserID).Error; err == nil {
			username = user.Username
		}
		result = append(result, map[string]any{
			"rank":         len(result) + 1,
			"user_id":      submission.UserID,
			"username":     username,
			"submitted_at": submission.SubmittedAt,
		})
		if len(result) >= limit {
			break
		}
	}
	return result
}

func (s *ChallengeService) practiceChallengeSolvedCount(challengeID uint) int64 {
	var total int64
	_ = s.db.Model(&models.Submission{}).
		Where("challenge_id = ? AND is_correct = ? AND competition_id IS NULL", challengeID, true).
		Distinct("user_id").
		Count(&total).Error
	return total
}

func (s *ChallengeService) activeChallengeContainerGroup(challengeID, userID uint) (models.ContainerGroup, bool) {
	var items []models.ContainerGroup
	if err := s.db.
		Where("challenge_id = ? AND user_id = ? AND status IN ?", challengeID, userID, activeContainerGroupStatuses).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return models.ContainerGroup{}, false
	}
	now := time.Now()
	for _, item := range items {
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		return item, true
	}
	return models.ContainerGroup{}, false
}

func challengeContainerGroupInfo(group models.ContainerGroup) map[string]any {
	expires := 0
	expiresAtValue := any(nil)
	if group.ExpiresAt != nil {
		expiresAtValue = *group.ExpiresAt
		expires = int(time.Until(*group.ExpiresAt).Seconds())
		if expires < 0 {
			expires = 0
		}
	}
	containers := NormalizeTemplateServiceRows(containerRuntimeRows(group.Containers))
	containerID := strings.TrimSpace(group.ContainerID)
	containerName := strings.TrimSpace(group.Name)
	targets := []map[string]any{}
	for index, container := range containers {
		if index == 0 {
			if containerID == "" {
				containerID = strings.TrimSpace(fmt.Sprint(container["container_id"]))
			}
			if containerName == "" {
				containerName = strings.TrimSpace(fmt.Sprint(container["name"]))
			}
		}
		if containerTargets := jsonMapRows(container["targets"]); len(containerTargets) > 0 {
			targets = append(targets, containerTargets...)
			continue
		}
		targets = append(targets, jsonMapRows(container["ports"])...)
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

func containerRuntimeRows(items models.ContainerRuntimeList) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	payload, err := json.Marshal(items)
	if err != nil {
		return rows
	}
	_ = json.Unmarshal(payload, &rows)
	return rows
}

func jsonMapRows(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows
	case string:
		rows := []map[string]any{}
		_ = json.Unmarshal([]byte(typed), &rows)
		return rows
	default:
		payload, err := json.Marshal(value)
		if err != nil {
			return []map[string]any{}
		}
		rows := []map[string]any{}
		_ = json.Unmarshal(payload, &rows)
		return rows
	}
}
