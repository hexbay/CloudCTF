package services

import (
	"fmt"

	"cloudctf/backend/internal/models"
)

func (s *VulnLabService) PlayerDetail(id any, userID uint) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	lab, err := s.GetByID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	if !lab.IsPublic || !lab.IsActive {
		return nil, fmt.Errorf("%w: lab is not available", ErrForbidden)
	}

	row := vulnLabToRow(*lab)
	row["runtime"] = nil
	if userID == 0 {
		return row, nil
	}

	var group models.ContainerGroup
	_ = s.db.Where("user_id = ? AND vuln_lab_id = ?", userID, lab.ID).First(&group).Error
	if group.ID == 0 {
		return row, nil
	}

	targets := []models.ContainerTarget{}
	for _, container := range group.Containers {
		targets = append(targets, container.Targets...)
	}
	row["runtime"] = map[string]any{
		"targets":    targets,
		"group_id":   group.ID,
		"target":     group.TemplateName,
		"status":     group.Status,
		"expires_at": group.ExpiresAt,
	}
	return row, nil
}
