package httpserver

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"

	"gorm.io/gorm"
)

type templateResourceStore struct {
	db  *gorm.DB
	srv *Server
}

func (s *templateResourceStore) List(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db unavailable")
	}
	db := s.db.Model(&models.Templates{})
	if keyword := strings.TrimSpace(firstQueryValue(query, "keyword")); keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := strings.TrimSpace(firstQueryValue(query, "status")); status != "" {
		db = db.Where("status = ?", status)
	}
	if templateType := strings.TrimSpace(firstQueryValue(query, "template_type")); templateType != "" {
		db = db.Where("template_type = ?", templateType)
	}
	if hostID := strings.TrimSpace(firstQueryValue(query, "host_id")); hostID != "" {
		db = db.Where("host_id = ?", hostID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	var items []models.Templates
	if err := db.Order("id DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, templateModelToRow(item))
	}
	return rows, int(total), nil
}

func (s *templateResourceStore) Get(id any) (map[string]any, bool, error) {
	template, found, err := s.srv.loadTemplateModelByID(id)
	if err != nil || !found {
		return nil, false, err
	}
	return templateModelToRow(*template), true, nil
}

func (s *templateResourceStore) Insert(row map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	now := time.Now()
	item := models.Templates{
		Name:          strings.TrimSpace(cleanString(row["name"])),
		ComposeFile:   defaultString(row["compose_file"], ""),
		Path:          defaultString(row["path"], ""),
		Description:   defaultString(row["description"], ""),
		HostId:        optionalUint(row["host_id"]),
		LastSyncAt:    parseTimeValue(row["last_sync_at"], now),
		RepositoryUrl: defaultString(row["repository_url"], ""),
		Services:      templateServicesValue(row["services"]),
		Status:        models.TemplateStatus(defaultString(row["status"], "pending")),
		Tags:          services.NormalizeStringSlice(row["tags"]),
		TemplateType:  defaultString(row["template_type"], "compose"),
		Ports:         services.NormalizeStringSlice(row["ports"]),
	}
	if item.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return templateModelToRow(item), nil
}

func (s *templateResourceStore) Update(id any, patch map[string]any) (map[string]any, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, fmt.Errorf("db unavailable")
	}
	template, found, err := s.srv.loadTemplateModelByID(id)
	if err != nil || !found {
		return nil, false, err
	}
	item := *template
	if value, ok := patch["name"]; ok {
		item.Name = strings.TrimSpace(cleanString(value))
	}
	if value, ok := patch["compose_file"]; ok {
		item.ComposeFile = defaultString(value, item.ComposeFile)
	}
	if value, ok := patch["path"]; ok {
		item.Path = defaultString(value, item.Path)
	}
	if value, ok := patch["description"]; ok {
		item.Description = defaultString(value, item.Description)
	}
	if value, ok := patch["host_id"]; ok {
		item.HostId = optionalUint(value)
	}
	if value, ok := patch["last_sync_at"]; ok {
		item.LastSyncAt = parseTimeValue(value, item.LastSyncAt)
	}
	if value, ok := patch["repository_url"]; ok {
		item.RepositoryUrl = defaultString(value, item.RepositoryUrl)
	}
	if value, ok := patch["services"]; ok {
		item.Services = templateServicesValue(value)
	}
	if value, ok := patch["status"]; ok {
		item.Status = models.TemplateStatus(defaultString(value, string(item.Status)))
	}
	if value, ok := patch["tags"]; ok {
		item.Tags = services.NormalizeStringSlice(value)
	}
	if value, ok := patch["template_type"]; ok {
		item.TemplateType = defaultString(value, item.TemplateType)
	}
	if value, ok := patch["ports"]; ok {
		item.Ports = services.NormalizeStringSlice(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, false, err
	}
	return templateModelToRow(item), true, nil
}

func (srv *Server) loadTemplateModelByID(id any) (*models.Templates, bool, error) {
	var template models.Templates
	if err := srv.db.Preload("Host").First(&template, fmt.Sprint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &template, true, nil
}

func (srv *Server) loadTemplateRowByID(id any) (map[string]any, bool, error) {
	template, found, err := srv.loadTemplateModelByID(id)
	if err != nil || !found {
		return nil, found, err
	}
	return templateModelToRow(*template), true, nil
}

func templateModelToRow(template models.Templates) map[string]any {
	image := ""
	if len(template.Services) > 0 {
		image = strings.TrimSpace(template.Services[0].Image)
	}
	if image == "" {
		image = services.FirstImageFromComposeBuild(template.ComposeFile)
	}
	return map[string]any{
		"id":             template.Id,
		"name":           template.Name,
		"image":          image,
		"compose_file":   template.ComposeFile,
		"path":           template.Path,
		"description":    template.Description,
		"host_id":        template.HostId,
		"last_sync_at":   template.LastSyncAt,
		"ports":          template.Ports,
		"repository_url": template.RepositoryUrl,
		"services":       template.Services,
		"status":         template.Status,
		"tags":           template.Tags,
		"template_type":  template.TemplateType,
		"created_at":     template.CreatedAt,
		"updated_at":     template.UpdatedAt,
	}
}

func firstQueryValue(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func parseTimeValue(value any, fallback time.Time) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func templateServicesValue(value any) []models.TemplateService {
	switch typed := value.(type) {
	case []models.TemplateService:
		return typed
	default:
		rows := services.NormalizeTemplateServiceRows(value)
		items := make([]models.TemplateService, 0, len(rows))
		for _, row := range rows {
			items = append(items, models.TemplateService{
				Name:       strings.TrimSpace(cleanString(row["name"])),
				Image:      strings.TrimSpace(cleanString(row["image"])),
				Ports:      services.NormalizeStringSlice(row["ports"]),
				Dockerfile: strings.TrimSpace(cleanString(row["dockerfile"])),
			})
		}
		return items
	}
}
