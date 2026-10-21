package services

import (
	"fmt"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

type ExtendChallengeRuntimeInput struct {
	UserID      uint
	ChallengeID uint
}

type StopChallengeRuntimeInput struct {
	UserID      uint
	ChallengeID uint
}

type ExtendVulnLabRuntimeInput struct {
	UserID    uint
	VulnLabID uint
}

type StopVulnLabRuntimeInput struct {
	UserID    uint
	VulnLabID uint
}

type StartVulnLabRuntimeInput struct {
	UserID    uint
	VulnLabID uint
}

func (s *ContainerGroupService) StartVulnLabRuntime(input StartVulnLabRuntimeInput) (models.ContainerGroup, error) {
	var zero models.ContainerGroup
	if s == nil || s.db == nil {
		return zero, fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.VulnLabID == 0 {
		return zero, fmt.Errorf("%w: user_id and vuln_lab_id are required", ErrInvalidInput)
	}

	var user models.User
	if err := s.db.First(&user, input.UserID).Error; err != nil {
		return zero, ErrForbidden
	}
	var lab models.VulnLab
	if err := s.db.First(&lab, input.VulnLabID).Error; err != nil {
		return zero, fmt.Errorf("%w: lab not found", ErrNotFound)
	}
	if lab.TemplateID == 0 {
		return zero, fmt.Errorf("%w: lab template not configured", ErrInvalidInput)
	}
	var template models.Templates
	if err := s.db.Preload("Host").First(&template, lab.TemplateID).Error; err != nil {
		return zero, fmt.Errorf("%w: template not found", ErrInvalidInput)
	}
	if template.Status != models.TemplateStatusCompleted && template.Status != models.TemplateStatusReady {
		return zero, fmt.Errorf("%w: 当前环境模板未就绪、请联系管理员!", ErrInvalidInput)
	}
	if template.HostId == nil || *template.HostId == 0 {
		return zero, fmt.Errorf("%w: template host is not configured", ErrInvalidInput)
	}

	group, err := s.StartTemplateGroup(&template, &user, nil, &lab)
	if err != nil {
		return zero, err
	}
	return *group, nil
}

func (s *ContainerGroupService) ExtendChallengeRuntime(input ExtendChallengeRuntimeInput) (models.ContainerGroup, error) {
	var zero models.ContainerGroup
	if s == nil || s.db == nil {
		return zero, fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.ChallengeID == 0 {
		return zero, fmt.Errorf("%w: user_id and challenge_id are required", ErrInvalidInput)
	}

	var challenge models.Challenge
	if err := s.db.First(&challenge, input.ChallengeID).Error; err != nil {
		return zero, ErrNotFound
	}

	group, found := s.GetActiveChallengeContainerGroup(input.ChallengeID, input.UserID)
	if !found {
		return zero, ErrNotFound
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(containerTimeoutOrDefault(challenge.ContainerTimeout)) * time.Second)
	if currentExpiresAt, ok := s.ContainerGroupExpires(group); ok {
		expiresAt = currentExpiresAt.Add(30 * time.Minute)
	} else {
		expiresAt = now.Add(30 * time.Minute)
	}

	updatedRow, ok := s.UpdateContainerGroup(group["id"], map[string]any{
		"expires_at": expiresAt.Format(time.RFC3339),
		"updated_at": now.Format(time.RFC3339),
	})
	if !ok {
		return zero, fmt.Errorf("failed to extend container")
	}

	updated, found := s.GetContainerGroupByID(uintValue(updatedRow["id"]))
	if !found {
		return zero, gorm.ErrRecordNotFound
	}
	return updated, nil
}

func (s *ContainerGroupService) StopChallengeRuntime(input StopChallengeRuntimeInput) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.ChallengeID == 0 {
		return fmt.Errorf("%w: user_id and challenge_id are required", ErrInvalidInput)
	}

	group, found := s.GetActiveChallengeContainerGroup(input.ChallengeID, input.UserID)
	if !found {
		return ErrNotFound
	}
	if err := s.DestroyContainerGroupRuntime(group); err != nil {
		return err
	}
	if !s.DeleteContainerGroup(group["id"]) {
		return fmt.Errorf("failed to stop container")
	}
	return nil
}

func (s *ContainerGroupService) ExtendVulnLabRuntime(input ExtendVulnLabRuntimeInput) (models.ContainerGroup, error) {
	var zero models.ContainerGroup
	if s == nil || s.db == nil {
		return zero, fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.VulnLabID == 0 {
		return zero, fmt.Errorf("%w: user_id and vuln_lab_id are required", ErrInvalidInput)
	}
	var lab models.VulnLab
	if err := s.db.First(&lab, input.VulnLabID).Error; err != nil {
		return zero, fmt.Errorf("%w: lab not found", ErrNotFound)
	}
	group, found := s.activeVulnLabContainerGroup(input.VulnLabID, input.UserID)
	if !found {
		return zero, fmt.Errorf("%w: active lab environment not found", ErrNotFound)
	}

	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)
	if group.ExpiresAt != nil {
		expiresAt = group.ExpiresAt.Add(30 * time.Minute)
	}
	if err := s.db.Model(&models.ContainerGroup{}).Where("id = ?", group.ID).Updates(map[string]any{
		"expires_at": expiresAt,
		"updated_at": now,
	}).Error; err != nil {
		return zero, err
	}
	group.ExpiresAt = &expiresAt
	group.UpdatedAt = now
	return group, nil
}

func (s *ContainerGroupService) StopVulnLabRuntime(input StopVulnLabRuntimeInput) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db unavailable")
	}
	if input.UserID == 0 || input.VulnLabID == 0 {
		return fmt.Errorf("%w: user_id and vuln_lab_id are required", ErrInvalidInput)
	}
	var lab models.VulnLab
	if err := s.db.First(&lab, input.VulnLabID).Error; err != nil {
		return fmt.Errorf("%w: lab not found", ErrNotFound)
	}
	group, found := s.activeVulnLabContainerGroup(input.VulnLabID, input.UserID)
	if !found {
		return fmt.Errorf("%w: active lab environment not found", ErrNotFound)
	}
	if err := s.DestroyContainerGroupRuntime(ContainerGroupToRow(group)); err != nil {
		return err
	}
	if !s.DeleteContainerGroup(group.ID) {
		return fmt.Errorf("failed to stop lab runtime")
	}
	return nil
}

func (s *ContainerGroupService) activeVulnLabContainerGroup(vulnLabID, userID uint) (models.ContainerGroup, bool) {
	var group models.ContainerGroup
	if err := s.db.
		Where("user_id = ? AND vuln_lab_id = ? AND status IN ?", userID, vulnLabID, activeContainerGroupStatuses).
		Order("id DESC").
		First(&group).Error; err != nil {
		return models.ContainerGroup{}, false
	}
	return group, true
}

func playerAccessibleChallengeModel(ch models.Challenge) bool {
	if !ch.IsActive || !ch.IsPublic {
		return false
	}
	if ch.CompetitionID != nil {
		return ch.Usage == "" || ch.Usage == "COMPETITION" || ch.Usage == "BOTH"
	}
	return ch.Usage == "" || ch.Usage == "PRACTICE" || ch.Usage == "BOTH"
}

func containerTimeoutOrDefault(timeout int) int {
	if timeout <= 0 {
		return 3600
	}
	return timeout
}
