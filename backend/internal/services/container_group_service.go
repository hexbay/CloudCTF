package services

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

var activeContainerGroupStatuses = []string{"running", "creating", "created"}
var reusableContainerGroupStatuses = []string{"running", "creating", "created", "expired"}

type ContainerGroupRuntime interface {
	RemoveContainer(ctx context.Context, endpoint, containerID string) error
	CreatePracticeContainer(ctx context.Context, spec PracticeContainerSpec) (map[string]any, error)
	InspectImageExposedPorts(ctx context.Context, endpoint, image string) ([]string, error)
	TemplatePortSpecs(templateID uint) []string
	TemplateImageFromMetadata(templateID uint) string
}

type ContainerGroupService struct {
	db      *gorm.DB
	runtime ContainerGroupRuntime
}

type PracticeContainerSpec struct {
	Endpoint    string
	Image       string
	Name        string
	PortSpecs   []string
	ChallengeID uint
	UserID      uint
}

func NewContainerGroupService(db *gorm.DB, runtime ContainerGroupRuntime) *ContainerGroupService {
	return &ContainerGroupService{db: db, runtime: runtime}
}

func (s *ContainerGroupService) ContainerGroupExpires(row map[string]any) (time.Time, bool) {
	raw, _ := row["expires_at"].(string)
	if raw == "" {
		if value, ok := row["expires_at"].(time.Time); ok {
			return value, true
		}
		if value, ok := row["expires_at"].(*time.Time); ok && value != nil {
			return *value, true
		}
		return time.Time{}, false
	}
	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		expiresAt, err = time.Parse(time.RFC3339Nano, raw)
	}
	if err != nil {
		return time.Time{}, false
	}
	return expiresAt, true
}

func (s *ContainerGroupService) ContainerGroupIsExpired(row map[string]any, now time.Time) bool {
	if expiresAt, ok := s.ContainerGroupExpires(row); ok {
		return !expiresAt.After(now)
	}
	return false
}

func (s *ContainerGroupService) GetActiveChallengeContainerGroup(challengeID, userID uint) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var items []models.ContainerGroup
	if err := s.db.
		Where("challenge_id = ? AND user_id = ? AND status IN ?", challengeID, userID, activeContainerGroupStatuses).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, false
	}
	now := time.Now()
	for _, item := range items {
		row := containerGroupToRow(item)
		if s.ContainerGroupIsExpired(row, now) {
			s.markContainerGroupExpired(item.ID, now)
			continue
		}
		return row, true
	}
	return nil, false
}

func (s *ContainerGroupService) GetReusableExpiredChallengeContainerGroup(challengeID, userID uint) (map[string]any, bool) {
	if s == nil || s.db == nil {
		return nil, false
	}
	var items []models.ContainerGroup
	if err := s.db.
		Where("challenge_id = ? AND user_id = ? AND status IN ?", challengeID, userID, reusableContainerGroupStatuses).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, false
	}
	now := time.Now()
	for _, item := range items {
		row := containerGroupToRow(item)
		if !s.ContainerGroupIsExpired(row, now) {
			continue
		}
		return row, true
	}
	return nil, false
}

func (s *ContainerGroupService) ListContainerGroups(current, pageSize int, query map[string][]string) ([]map[string]any, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, nil
	}
	var items []models.ContainerGroup
	db := s.db.Model(&models.ContainerGroup{})
	if status := firstQuery(query, "status"); status != "" {
		db = db.Where("status = ?", status)
	}
	if hostID := firstQuery(query, "host_id"); hostID != "" {
		db = db.Where("host_id = ?", hostID)
	}
	if templateID := firstQuery(query, "template_id"); templateID != "" {
		db = db.Where("template_id = ?", templateID)
	}
	if challengeID := firstQuery(query, "challenge_id"); challengeID != "" {
		db = db.Where("challenge_id = ?", challengeID)
	}
	if userID := firstQuery(query, "user_id"); userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if challengeName := firstQuery(query, "challenge_name"); challengeName != "" {
		db = db.Where("challenge_name LIKE ?", "%"+challengeName+"%")
	}
	if userName := firstQuery(query, "user_name"); userName != "" {
		db = db.Where("user_name LIKE ?", "%"+userName+"%")
	}
	if vulnLabName := firstQuery(query, "vuln_lab_name"); vulnLabName != "" {
		db = db.Where("vuln_lab_name LIKE ?", "%"+vulnLabName+"%")
	}
	if sourceType := firstQuery(query, "source_type"); sourceType != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	if keyword := firstQuery(query, "keyword"); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR template_name LIKE ? OR challenge_name LIKE ? OR user_name LIKE ? OR vuln_lab_name LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if current <= 0 {
		current = 1
	}
	if err := db.Offset((current - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, containerGroupToRow(item))
	}
	return rows, int(total), nil
}

func (s *ContainerGroupService) GetContainerGroupByID(id uint) (models.ContainerGroup, bool) {
	var item models.ContainerGroup
	if s == nil || s.db == nil {
		return item, false
	}
	if id == 0 {
		return item, false
	}
	if err := s.db.Where("id = ?", id).First(&item).Error; err != nil {
		return item, false
	}
	return item, true
}

func (s *ContainerGroupService) CreateContainerGroup(payload map[string]any) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	item := buildContainerGroupFromPayload(payload)
	if item.Name == "" {
		item.Name = "Container Group"
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return containerGroupToRow(item), nil
}

func (s *ContainerGroupService) UpdateContainerGroup(id any, patch map[string]any) (map[string]any, bool) {
	row, err := s.updateContainerGroupRow(id, patch)
	return row, err == nil
}

func (s *ContainerGroupService) DeleteContainerGroup(id any) bool {
	if s == nil || s.db == nil {
		return false
	}
	result := s.db.Delete(&models.ContainerGroup{}, fmt.Sprint(id))
	return result.Error == nil && result.RowsAffected > 0
}

func (s *ContainerGroupService) CleanupExpiredContainerGroups() {
	if s == nil || s.db == nil {
		return
	}
	var items []models.ContainerGroup
	now := time.Now()
	if err := s.db.
		Where("status IN ? AND expires_at IS NOT NULL AND expires_at <= ?", reusableContainerGroupStatuses, now).
		Order("id ASC").
		Find(&items).Error; err != nil {
		return
	}
	for _, item := range items {
		row := containerGroupToRow(item)
		if item.Status != "expired" {
			s.markContainerGroupExpired(item.ID, now)
			row["status"] = "expired"
			row["updated_at"] = now
		}
		if err := s.DestroyContainerGroupRuntime(row); err != nil {
			continue
		}
		_ = s.db.Delete(&models.ContainerGroup{}, item.ID).Error
	}
}

func (s *ContainerGroupService) DestroyContainerGroupRuntime(group map[string]any) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	endpoint := s.resolveContainerGroupEndpoint(group)
	for _, containerRow := range normalizeTemplateServices(group["containers"]) {
		containerID := cleanString(containerRow["container_id"])
		if containerID == "" {
			continue
		}
		if err := s.runtime.RemoveContainer(context.Background(), endpoint, containerID); err != nil {
			lower := strings.ToLower(err.Error())
			if !strings.Contains(lower, "no such") && !strings.Contains(lower, "not found") {
				return err
			}
		}
	}
	return nil
}

func (s *ContainerGroupService) markContainerGroupExpired(id uint, now time.Time) {
	_ = s.db.Model(&models.ContainerGroup{}).Where("id = ?", id).Updates(map[string]any{
		"status":     "expired",
		"updated_at": now,
	}).Error
}

func (s *ContainerGroupService) resolveContainerGroupEndpoint(group map[string]any) string {
	endpoint := defaultDockerEndpoint()
	if s == nil || s.db == nil {
		return endpoint
	}
	if groupHostID := uintValue(group["host_id"]); groupHostID != 0 {
		var host models.VirtualizationHost
		if err := s.db.First(&host, groupHostID).Error; err == nil && strings.TrimSpace(host.Endpoint) != "" {
			return strings.TrimSpace(host.Endpoint)
		}
	}
	var host models.VirtualizationHost
	if err := s.db.Where("is_active = ?", true).Order("is_default DESC, id ASC").First(&host).Error; err == nil && strings.TrimSpace(host.Endpoint) != "" {
		return strings.TrimSpace(host.Endpoint)
	}
	return endpoint
}

func (s *ContainerGroupService) ResolveContainerGroupEndpoint(group models.ContainerGroup) string {
	return s.resolveContainerGroupEndpoint(ContainerGroupToRow(group))
}

func GenerateDynamicFlag() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("flag{%d}", time.Now().UnixNano())
	}
	return fmt.Sprintf("flag{%s}", hex.EncodeToString(buf))
}

func (s *ContainerGroupService) UpdateContainersFlag(cli *client.Client, containers []map[string]any, flag string) error {
	for _, container := range containers {
		if err := ExecDockerContainer(context.Background(), cli, container["id"].(string), []string{"sh", "-lc", "/start.sh " + flag}); err != nil {
			continue
		}
		return nil
	}
	return errors.New("write Flag Error")
}

func (s *ContainerGroupService) StartTemplateGroup(template *models.Templates, user *models.User, challenge *models.Challenge, vulLab *models.VulnLab) (*models.ContainerGroup, error) {
	if err := s.db.Preload("Host").First(&template, template.Id).Error; err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("未指定用户")
	}
	// 查找当前已存在的记录
	var existing models.ContainerGroup
	query := s.db.Where(
		"template_id = ? AND status IN ? AND user_id = ?",
		template.Id,
		[]string{"running", "creating", "created"}, user.ID,
	)
	if challenge != nil {
		query = query.Where("challenge_id = ?", challenge.ID)
	} else {
		query = query.Where("challenge_id IS NULL")
	}
	if vulLab != nil {
		query = query.Where("vuln_lab_id = ?", vulLab.ID)
	} else {
		query = query.Where("vuln_lab_id IS NULL")
	}
	err := query.First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	host := template.Host
	if host == nil {
		return nil, errors.New("当前模板未绑定主机")
	}
	if template.Status != models.TemplateStatusCompleted {
		return nil, errors.New("模板未就绪、请联系管理员")
	}
	if existing.ID == 0 {
		// 创建记录
		var name string
		if challenge != nil {
			name = fmt.Sprintf("CTF_%s_%s", user.Username, challenge.Name)
		} else if vulLab != nil {
			name = fmt.Sprintf("CTF_%s_%s", user.Username, vulLab.Name)
		}
		existing = models.ContainerGroup{
			Name:         name,
			Status:       "pending",
			SourceType:   "template",
			HostID:       optionalUintFromValue(host.ID),
			TemplateID:   optionalUintFromValue(template.Id),
			TemplateName: template.Name,
			UserID:       &user.ID,
		}
		if challenge != nil {
			existing.ChallengeID = &challenge.ID
		} else if vulLab != nil {
			existing.VulnLabID = &vulLab.ID
		}
		if err := s.db.Create(&existing).Error; err != nil {
			return nil, err
		}
	}
	now := time.Now()
	timeoutSeconds := 3600
	if challenge != nil {
		timeoutSeconds = s.settingInt("ctf_ttl_seconds", 3600)
	} else if vulLab != nil {
		timeoutSeconds = s.settingInt("vuln_ttl_seconds", 3600)
	}

	expiresAt := now.Add(time.Duration(timeoutSeconds) * time.Second)
	// create group
	existing.ExpiresAt = &expiresAt
	publicHost := "127.0.0.1"
	if strings.TrimSpace(host.OutIP) != "" {
		publicHost = strings.TrimSpace(host.OutIP)
	}
	groupID := existing.ID
	endpoint := strings.TrimSpace(host.Endpoint)
	networkName := "cg-" + fmt.Sprint(groupID)
	containers, startErr := StartTemplateContainers(context.Background(), *template, TemplateStartOptions{
		GroupID:       fmt.Sprint(groupID),
		Endpoint:      endpoint,
		PublicHost:    publicHost,
		NetworkName:   networkName,
		PullMissing:   PullDockerImage,
		ManagedCheck:  nil,
		ManagedLabels: nil,
		ResolveComposeImage: func(composeFile string) string {
			return FirstImageFromComposeBuild(composeFile)
		},
	})
	if startErr != nil {
		_, _ = s.UpdateContainerGroup(groupID, map[string]any{
			"status":     "error",
			"updated_at": time.Now(),
		})
		return nil, startErr
	}
	flag := ""
	updates := map[string]any{
		"status":     "running",
		"containers": containers,
		"updated_at": time.Now(),
		"started_at": now,
		"expires_at": expiresAt,
	}
	if challenge != nil {
		cli, _ := DockerClientForEndpoint(host.Endpoint)
		flag = GenerateDynamicFlag()
		err = s.UpdateContainersFlag(cli, containers, flag)
		if err != nil {
			slog.Warn("更新FLAG失败")
		} else {
			updates["flag"] = flag
		}
	}
	row, err := s.updateContainerGroupRow(groupID, updates)
	if err != nil {
		return nil, err
	}
	updated, found := s.GetContainerGroupByID(uintValue(row["id"]))
	if !found {
		return nil, gorm.ErrRecordNotFound
	}
	return &updated, nil
}

func ContainerGroupToRow(item models.ContainerGroup) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"name":           item.Name,
		"description":    item.Description,
		"status":         item.Status,
		"source_type":    item.SourceType,
		"host_id":        nullableUint(item.HostID),
		"template_id":    nullableUint(item.TemplateID),
		"template_name":  item.TemplateName,
		"challenge_id":   nullableUint(item.ChallengeID),
		"challenge_name": item.ChallengeName,
		"competition_id": nullableUint(item.CompetitionID),
		"user_id":        nullableUint(item.UserID),
		"user_name":      item.UserName,
		"vuln_lab_id":    nullableUint(item.VulnLabID),
		"vuln_lab_name":  item.VulnLabName,
		"vul_lab_name":   item.VulnLabName,
		"container_id":   item.ContainerID,
		"containers":     containerRuntimeListToRows(item.Containers),
		"flag":           item.Flag,
		"started_at":     item.StartedAt,
		"expires_at":     item.ExpiresAt,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func containerGroupToRow(item models.ContainerGroup) map[string]any {
	return ContainerGroupToRow(item)
}

func nullableUint(value *uint) any {
	if value == nil {
		return nil
	}
	return *value
}

func parseJSONArray(raw string) []map[string]any {
	if strings.TrimSpace(raw) == "" {
		return []map[string]any{}
	}
	items := []map[string]any{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []map[string]any{}
	}
	return items
}

func normalizeTemplateServices(value any) []map[string]any {
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
	default:
		return []map[string]any{}
	}
}

func buildContainerGroupFromPayload(payload map[string]any) models.ContainerGroup {
	return models.ContainerGroup{
		Name:          valueOrDefault(payload["name"], "Container Group"),
		Description:   cleanString(payload["description"]),
		Status:        valueOrDefault(payload["status"], "pending"),
		SourceType:    cleanString(payload["source_type"]),
		HostID:        optionalUint(payload["host_id"]),
		TemplateID:    optionalUint(payload["template_id"]),
		TemplateName:  cleanString(payload["template_name"]),
		ChallengeID:   optionalUint(payload["challenge_id"]),
		ChallengeName: cleanString(payload["challenge_name"]),
		CompetitionID: optionalUint(payload["competition_id"]),
		UserID:        optionalUint(payload["user_id"]),
		UserName:      cleanString(payload["user_name"]),
		VulnLabID:     optionalUint(payload["vuln_lab_id"]),
		VulnLabName:   firstNonEmpty(cleanString(payload["vuln_lab_name"]), cleanString(payload["vul_lab_name"])),
		ContainerID:   cleanString(payload["container_id"]),
		Containers:    containerRuntimeListValue(payload["containers"]),
		Flag:          cleanString(payload["flag"]),
		StartedAt:     parseOptionalTime(payload["started_at"]),
		ExpiresAt:     parseOptionalTime(payload["expires_at"]),
	}
}

func cleanString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "<nil>" {
			return ""
		}
		return text
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func uintValue(value any) uint {
	switch typed := value.(type) {
	case uint:
		return typed
	case uint64:
		return uint(typed)
	case uint32:
		return uint(typed)
	case int:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case int64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case float64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	default:
		return 0
	}
}

func defaultDockerEndpoint() string {
	return "http://127.0.0.1:2375"
}

func (s *ContainerGroupService) updateContainerGroupRow(id any, patch map[string]any) (map[string]any, error) {
	var item models.ContainerGroup
	if err := s.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return nil, err
	}
	if value, ok := patch["name"]; ok {
		item.Name = cleanString(value)
	}
	if value, ok := patch["description"]; ok {
		item.Description = cleanString(value)
	}
	if value, ok := patch["status"]; ok {
		item.Status = cleanString(value)
	}
	if value, ok := patch["source_type"]; ok {
		item.SourceType = cleanString(value)
	}
	if value, ok := patch["host_id"]; ok {
		item.HostID = optionalUint(value)
	}
	if value, ok := patch["template_id"]; ok {
		item.TemplateID = optionalUint(value)
	}
	if value, ok := patch["template_name"]; ok {
		item.TemplateName = cleanString(value)
	}
	if value, ok := patch["challenge_id"]; ok {
		item.ChallengeID = optionalUint(value)
	}
	if value, ok := patch["challenge_name"]; ok {
		item.ChallengeName = cleanString(value)
	}
	if value, ok := patch["competition_id"]; ok {
		item.CompetitionID = optionalUint(value)
	}
	if value, ok := patch["user_id"]; ok {
		item.UserID = optionalUint(value)
	}
	if value, ok := patch["user_name"]; ok {
		item.UserName = cleanString(value)
	}
	if value, ok := patch["vuln_lab_id"]; ok {
		item.VulnLabID = optionalUint(value)
	}
	if value, ok := patch["vuln_lab_name"]; ok {
		item.VulnLabName = cleanString(value)
	}
	if value, ok := patch["vul_lab_name"]; ok && item.VulnLabName == "" {
		item.VulnLabName = cleanString(value)
	}
	if value, ok := patch["container_id"]; ok {
		item.ContainerID = cleanString(value)
	}
	if value, ok := patch["containers"]; ok {
		item.Containers = containerRuntimeListValue(value)
	}
	if value, ok := patch["flag"]; ok {
		item.Flag = cleanString(value)
	}
	if value, ok := patch["expires_at"]; ok {
		item.ExpiresAt = parseOptionalTime(value)
	}
	if value, ok := patch["started_at"]; ok {
		item.StartedAt = parseOptionalTime(value)
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	return containerGroupToRow(item), nil
}

func (s *ContainerGroupService) resolveChallengeContainerRuntime(challenge models.Challenge) (string, string, uint, string, []string, error) {
	host := "127.0.0.1"
	endpoint := ""
	hostID := uint(0)
	if s != nil && s.db != nil {
		var virtualHost models.VirtualizationHost
		if err := s.db.Where("is_active = ?", true).Order("is_default DESC, id ASC").First(&virtualHost).Error; err == nil {
			hostID = virtualHost.ID
			if strings.TrimSpace(virtualHost.OutIP) != "" {
				host = strings.TrimSpace(virtualHost.OutIP)
			}
			endpoint = strings.TrimSpace(virtualHost.Endpoint)
		}
	}
	if endpoint == "" {
		endpoint = defaultDockerEndpoint()
	}
	image := strings.TrimSpace(challenge.ContainerImage)
	if image == "<nil>" {
		image = ""
	}
	if image == "" && challenge.TemplateID != nil && s.runtime != nil {
		image = s.runtime.TemplateImageFromMetadata(*challenge.TemplateID)
	}
	if image == "" {
		return "", "", 0, "", nil, fmt.Errorf("challenge container image is not configured")
	}
	portSpecs := make([]string, 0, 2)
	if challenge.TemplateID != nil && s.runtime != nil {
		portSpecs = append(portSpecs, s.runtime.TemplatePortSpecs(*challenge.TemplateID)...)
	}
	if len(portSpecs) == 0 && s.runtime != nil {
		if exposedPorts, err := s.runtime.InspectImageExposedPorts(context.Background(), endpoint, image); err == nil {
			portSpecs = append(portSpecs, exposedPorts...)
		}
	}
	portSpecs = uniqueSortedPortValues(portSpecs)
	if len(portSpecs) == 0 {
		return "", "", 0, "", nil, fmt.Errorf("challenge container ports are not configured")
	}
	return endpoint, host, hostID, image, portSpecs, nil
}

func challengeContainerTargetsFromDockerPorts(host string, ports map[string]any) []map[string]any {
	targets := make([]map[string]any, 0, len(ports))
	for rawHostPort, rawContainerPort := range ports {
		hostPort := strings.TrimSpace(rawHostPort)
		containerSpec := strings.TrimSpace(fmt.Sprint(rawContainerPort))
		if hostPort == "" || containerSpec == "" {
			continue
		}
		containerPort := portNumber(containerSpec)
		if containerPort == "" {
			containerPort = containerSpec
		}
		protocol := portProtocol(containerSpec)
		if protocol == "" || protocol == "tcp" {
			protocol = "http"
		}
		targets = append(targets, map[string]any{
			"port":           hostPort,
			"protocol":       protocol,
			"container_port": containerPort,
			"url":            fmt.Sprintf("%s://%s:%s", protocol, host, hostPort),
		})
	}
	sort.Slice(targets, func(i, j int) bool {
		return fmt.Sprint(targets[i]["port"]) < fmt.Sprint(targets[j]["port"])
	})
	return targets
}

func mustJSONString(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func containerRuntimeListValue(value any) models.ContainerRuntimeList {
	switch typed := value.(type) {
	case nil:
		return models.ContainerRuntimeList{}
	case models.ContainerRuntimeList:
		return typed
	case []models.ContainerRuntime:
		return models.ContainerRuntimeList(typed)
	case string:
		text := strings.TrimSpace(typed)
		if text == "" || text == "<nil>" {
			return models.ContainerRuntimeList{}
		}
		var items models.ContainerRuntimeList
		if err := items.Scan(text); err == nil {
			return items
		}
	case []map[string]any:
		return mapRowsToContainerRuntimeList(typed)
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return mapRowsToContainerRuntimeList(rows)
	}
	return mapRowsToContainerRuntimeList(parseJSONArray(mustJSONString(value)))
}

func mapRowsToContainerRuntimeList(rows []map[string]any) models.ContainerRuntimeList {
	if len(rows) == 0 {
		return models.ContainerRuntimeList{}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return models.ContainerRuntimeList{}
	}
	items := models.ContainerRuntimeList{}
	if err := items.Scan(payload); err != nil {
		return models.ContainerRuntimeList{}
	}
	return items
}

func containerRuntimeListToRows(items models.ContainerRuntimeList) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return []map[string]any{}
	}
	rows := []map[string]any{}
	if err := json.Unmarshal(payload, &rows); err != nil {
		return []map[string]any{}
	}
	return rows
}

func optionalUintFromValue(value uint) *uint {
	if value == 0 {
		return nil
	}
	copy := value
	return &copy
}

func optionalUint(value any) *uint {
	switch typed := value.(type) {
	case nil:
		return nil
	case uint:
		copy := typed
		return &copy
	case uint64:
		copy := uint(typed)
		return &copy
	case int:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case int64:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case float64:
		if typed <= 0 {
			return nil
		}
		copy := uint(typed)
		return &copy
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if parsed, err := strconv.ParseUint(text, 10, 64); err == nil && parsed > 0 {
			copy := uint(parsed)
			return &copy
		}
	}
	return nil
}

func parseOptionalTime(value any) *time.Time {
	switch typed := value.(type) {
	case time.Time:
		copy := typed
		return &copy
	case *time.Time:
		return typed
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, text); err == nil {
			return &parsed
		}
	}
	return nil
}

func valueOrZero(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func valueOrDefault(value any, fallback string) string {
	text := cleanString(value)
	if text == "" {
		return fallback
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstQuery(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func normalizeContainerPort(value string) string {
	text := strings.TrimSpace(strings.Trim(value, `"'`))
	if text == "" {
		return ""
	}
	if strings.Contains(text, "/") {
		return text
	}
	return text + "/tcp"
}

func portNumber(value string) string {
	text := normalizeContainerPort(value)
	parts := strings.SplitN(text, "/", 2)
	return parts[0]
}

func portProtocol(value string) string {
	text := normalizeContainerPort(value)
	parts := strings.SplitN(text, "/", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return parts[1]
	}
	return "tcp"
}

func uniqueSortedPortValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeContainerPort(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Slice(result, func(i, j int) bool {
		left := portNumber(result[i])
		right := portNumber(result[j])
		if left == right {
			return portProtocol(result[i]) < portProtocol(result[j])
		}
		return left < right
	})
	return result
}

func generateDynamicFlag(userID, challengeID uint, now time.Time) string {
	salt := fmt.Sprintf("cloudctf_%d_%d_%d", userID, challengeID, now.Unix()/86400)
	hash := md5.Sum([]byte(salt))
	return fmt.Sprintf("flag{%x}", hash)
}

func (s *ContainerGroupService) settingInt(key string, fallback int) int {
	if s.db == nil {
		return fallback
	}
	var val models.SystemSetting
	if err := s.db.Where("`key` = ?", key).First(&val).Error; err != nil {
		return fallback
	}
	n, err := strconv.Atoi(val.Value)
	if err != nil {
		return fallback
	}
	return n
}
