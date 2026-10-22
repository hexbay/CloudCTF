package httpserver

import (
	"cloudctf/backend/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/services"

	"gorm.io/gorm"
)

type templateBuildTask struct {
	ID         string
	TemplateID string
	Template   models.Templates
	Status     string
	Logs       []string
	CreatedAt  string
	UpdatedAt  string
	Done       chan struct{}
	mu         sync.RWMutex
}

var templateBuildTasks = struct {
	mu    sync.RWMutex
	tasks map[string]*templateBuildTask
}{
	tasks: map[string]*templateBuildTask{},
}

func (srv *Server) adminVirtualizationTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	var records []models.Templates
	var total int64
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	offset := (current - 1) * pageSize
	// 常见过滤参数
	keyword := queryStr(r, "keyword")
	status := queryStr(r, "status")
	templateType := queryStr(r, "template_type")
	hostID := queryStr(r, "host_id")
	db := srv.db.Model(&models.Templates{})
	if keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if templateType != "" {
		db = db.Where("template_type = ?", templateType)
	}
	if hostID != "" {
		db = db.Where("host_id = ?", hostID)
	}
	if err := db.Count(&total).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if err := db.Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
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

func (srv *Server) adminVirtualizationTemplateCreateHandler(w http.ResponseWriter, r *http.Request) {
	store := srv.templates
	if store == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return
	}
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	now := time.Now().Format(time.RFC3339)
	payload["status"] = defaultString(payload["status"], "pending")
	payload["compose_file"] = defaultString(payload["compose_file"], "services:\n  app:\n    image: "+defaultString(payload["image"], "nginx:latest")+"\n")
	payload["created_at"] = now
	payload["updated_at"] = now
	created, err := store.Insert(payload)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, created)
}

func (srv *Server) adminVirtualizationTemplateDetailHandler(w http.ResponseWriter, r *http.Request) {
	store := srv.templates
	if store == nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	row, ok, err := store.Get(r.PathValue("id"))
	if err != nil || !ok {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationTemplateUpdateHandler(w http.ResponseWriter, r *http.Request) {
	store := srv.templates
	if store == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return
	}
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	payload["updated_at"] = time.Now().Format(time.RFC3339)
	err := srv.db.Update(r.PathValue("id"), payload).Error
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, payload)
}

func (srv *Server) adminVirtualizationTemplateDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if srv.db.Delete(&models.Templates{}, r.PathValue("id")).Error == nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	_ = os.RemoveAll(templateFilesDir(r.PathValue("id")))
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminVirtualizationTemplateBatchDeleteHandler(w http.ResponseWriter, r *http.Request) {

	var payload map[string][]string
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "ids is unavailable"})
		return
	}
	ids := payload["ids"]
	if len(ids) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "ids required"})
		return
	}
	deleted := 0
	for _, id := range ids {
		if srv.db.Delete(&models.Templates{}, id).Error == nil {
			deleted++
			_ = os.RemoveAll(templateFilesDir(fmt.Sprintf("%s", id)))
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "deleted": deleted})
}

func (srv *Server) adminVirtualizationTemplateBatchBuildHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	ids := idListFromPayload(payload["ids"])
	if len(ids) == 0 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "ids required"})
		return
	}
	tasks := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		template, found, err := srv.loadTemplateModelByID(id)
		if err != nil || !found {
			continue
		}
		task := srv.createTemplateBuildTask(id, *template)
		tasks = append(tasks, serializeTemplateBuildTask(task))
		go srv.runTemplateBuildTask(task)
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "tasks": tasks, "count": len(tasks)})
}

func idListFromPayload(value any) []string {
	ids := []string{}
	switch items := value.(type) {
	case []any:
		for _, item := range items {
			if id := stringifyID(item); id != "" {
				ids = append(ids, id)
			}
		}
	case []string:
		for _, item := range items {
			if id := strings.TrimSpace(item); id != "" {
				ids = append(ids, id)
			}
		}
	case string:
		for _, item := range strings.Split(items, ",") {
			if id := strings.TrimSpace(item); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (srv *Server) adminVirtualizationTemplateBuildHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	template, found, err := srv.loadTemplateModelByID(id)
	if err != nil || !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	task := srv.createTemplateBuildTask(id, *template)
	go srv.runTemplateBuildTask(task)
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "task": serializeTemplateBuildTask(task)})
}

func (srv *Server) requireContainerGroupService(w http.ResponseWriter) (*services.ContainerGroupService, bool) {
	service := srv.GetContainerGroupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "store unavailable"})
		return nil, false
	}
	return service, true
}

func (srv *Server) adminVirtualizationTemplateStartHandler(w http.ResponseWriter, r *http.Request) {
	containerGroupService, ok := srv.requireContainerGroupService(w)
	if !ok {
		return
	}
	currentUser, hasCurrentUser := srv.currentUserFromRequest(r)
	id := r.PathValue("id")
	template, found, err := srv.loadTemplateRowByID(id)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	templateStatus := cleanString(template["status"])
	if templateStatus != "completed" && templateStatus != "ready" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "template is not ready to start"})
		return
	}
	hostID := stringifyID(template["host_id"])
	if hostID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "template host is not configured"})
		return
	}
	if !hasCurrentUser {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "current user is required"})
		return
	}
	templateModel, foundModel, err := srv.loadTemplateModelByID(id)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	if !foundModel || templateModel == nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "template not found"})
		return
	}
	current, _, _ := containerGroupService.ListContainerGroups(1, 500, map[string][]string{})
	for _, row := range current {
		if stringifyID(row["template_id"]) == id {
			status := cleanString(row["status"])
			if status == "running" || status == "created" || status == "creating" {
				if cleanString(row["flag"]) == "" && strings.EqualFold(strings.TrimSpace(templateModel.TemplateType), "ctf") {
					updated, repairErr := containerGroupService.EnsureTemplateGroupFlag(uintValue(row["id"]), templateModel)
					if repairErr != nil {
						response.JSON(w, http.StatusBadRequest, map[string]any{"detail": repairErr.Error()})
						return
					}
					row = services.ContainerGroupToRow(updated)
				}
				response.JSON(w, http.StatusOK, map[string]any{
					"success": true,
					"group":   row,
					"message": "template already started",
				})
				return
			}
		}
	}
	started, err := containerGroupService.StartTemplateGroup(templateModel, &currentUser, nil, nil)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success": true,
		"group":   services.ContainerGroupToRow(*started),
		"message": "template started",
	})
}

func (srv *Server) adminVirtualizationContainerGroupsHandler(w http.ResponseWriter, r *http.Request) {
	containerGroupService := srv.GetContainerGroupService()
	if containerGroupService == nil {
		response.JSON(w, http.StatusOK, map[string]any{"data": []map[string]any{}, "total": 0})
		return
	}
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := containerGroupService.ListContainerGroups(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	for _, row := range rows {
		srv.populateContainerGroupDisplayFields(row)
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "total": total, "current": current, "pageSize": pageSize})
}

func (srv *Server) adminVirtualizationContainerGroupCreateHandler(w http.ResponseWriter, r *http.Request) {
	containerGroupService, ok := srv.requireContainerGroupService(w)
	if !ok {
		return
	}
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	now := time.Now().Format(time.RFC3339)
	payload["status"] = defaultString(payload["status"], "created")
	payload["created_at"] = now
	payload["updated_at"] = now
	row, err := containerGroupService.CreateContainerGroup(payload)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationContainerGroupDetailHandler(w http.ResponseWriter, r *http.Request) {
	containerGroupService, ok := srv.requireContainerGroupService(w)
	if !ok {
		return
	}
	id := pathUint(r, "id")
	row, found := containerGroupService.GetContainerGroupByID(id)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "container group not found"})
		return
	}
	payload := services.ContainerGroupToRow(row)
	srv.populateContainerGroupDisplayFields(payload)
	response.JSON(w, http.StatusOK, payload)
}

func (srv *Server) adminVirtualizationContainerGroupDeleteHandler(w http.ResponseWriter, r *http.Request) {
	containerGroupService, ok := srv.requireContainerGroupService(w)
	if !ok {
		return
	}
	id := pathUint(r, "id")
	row, found := containerGroupService.GetContainerGroupByID(id)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "container group not found"})
		return
	}
	if err := srv.applyContainerGroupAction(r.Context(), row, "remove"); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	if !containerGroupService.DeleteContainerGroup(id) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "container group not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminVirtualizationContainerGroupStartHandler(w http.ResponseWriter, r *http.Request) {
	row, err := srv.updateContainerGroupStatus(r.Context(), pathUint(r, "id"), "running", "start")
	if err != nil {
		writeContainerGroupActionError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationContainerGroupStopHandler(w http.ResponseWriter, r *http.Request) {
	row, err := srv.updateContainerGroupStatus(r.Context(), pathUint(r, "id"), "stopped", "stop")
	if err != nil {
		writeContainerGroupActionError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) updateContainerGroupStatus(ctx context.Context, id uint, status, action string) (map[string]any, error) {
	containerGroupService := srv.GetContainerGroupService()
	if containerGroupService == nil {
		return nil, errors.New("store unavailable")
	}
	if id == 0 {
		return nil, errors.New("invalid container group id")
	}
	group, found := containerGroupService.GetContainerGroupByID(id)
	if !found {
		return nil, gorm.ErrRecordNotFound
	}
	if action == "start" {
		if row, handled, err := srv.startContainerGroupRuntime(group); handled || err != nil {
			return row, err
		}
	}
	if action != "" {
		if err := srv.applyContainerGroupAction(ctx, group, action); err != nil {
			return nil, err
		}
	}
	row, found := containerGroupService.UpdateContainerGroup(id, map[string]any{
		"status":     status,
		"updated_at": time.Now().Format(time.RFC3339),
	})
	if !found {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (srv *Server) startContainerGroupRuntime(group models.ContainerGroup) (map[string]any, bool, error) {
	containerGroupService := srv.GetContainerGroupService()
	if containerGroupService == nil {
		return nil, false, errors.New("store unavailable")
	}
	if group.TemplateID == nil || *group.TemplateID == 0 {
		return nil, false, nil
	}
	if group.UserID == nil || *group.UserID == 0 {
		return nil, false, errors.New("container group user is required")
	}

	template, found, err := srv.loadTemplateModelByID(*group.TemplateID)
	if err != nil {
		return nil, true, err
	}
	if !found {
		return nil, true, gorm.ErrRecordNotFound
	}
	user, err := srv.getUserService().GetByID(*group.UserID)
	if err != nil {
		return nil, true, err
	}

	var challenge *models.Challenge
	if group.ChallengeID != nil && *group.ChallengeID != 0 {
		challenge, err = srv.getChallengeService().GetByID(*group.ChallengeID)
		if err != nil {
			return nil, true, err
		}
	}

	var vulLab *models.VulnLab
	if group.VulnLabID != nil && *group.VulnLabID != 0 {
		vulLab, err = srv.getVulnLabService().GetByID(*group.VulnLabID)
		if err != nil {
			return nil, true, err
		}
	}

	started, err := containerGroupService.StartTemplateGroup(template, user, challenge, vulLab)
	if err != nil {
		return nil, true, err
	}
	return services.ContainerGroupToRow(*started), true, nil
}

func writeContainerGroupActionError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "container group not found"})
	default:
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
	}
}

func (srv *Server) applyContainerGroupAction(ctx context.Context, group models.ContainerGroup, action string) error {
	containerGroupService := srv.GetContainerGroupService()
	if containerGroupService == nil {
		return errors.New("store unavailable")
	}
	endpoint := ""
	if group.HostID != nil && *group.HostID != 0 {
		if host, err := srv.getVirtualizationHostService().GetByID(*group.HostID); err == nil {
			endpoint = strings.TrimSpace(host.Endpoint)
		}
	}
	if endpoint == "" {
		endpoint = containerGroupService.ResolveContainerGroupEndpoint(group)
	}
	if endpoint == "" {
		return errors.New("container group endpoint is unavailable")
	}
	for _, containerRow := range services.NormalizeTemplateServiceRows(containerRuntimeRows(group.Containers)) {
		containerID := stringifyID(containerRow["container_id"])
		if containerID == "" {
			containerID = stringifyID(containerRow["id"])
		}
		if containerID == "" {
			continue
		}
		switch action {
		case "start", "stop", "restart":
			if err := services.RunDockerContainerAction(ctx, endpoint, containerID, action, srv.isManagedContainer); err != nil {
				return err
			}
		case "remove":
			if err := services.RemoveDockerContainer(ctx, endpoint, containerID, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (srv *Server) populateContainerGroupDisplayFields(row map[string]any) {
	if row == nil {
		return
	}
	if uintValue(row["template_id"]) != 0 && row["template"] == nil {
		if template, ok, _ := srv.loadTemplateRowByID(row["template_id"]); ok {
			row["template"] = template
			if cleanString(row["template_name"]) == "" {
				row["template_name"] = cleanString(template["name"])
			}
		}
	}
	if cleanString(row["template_name"]) == "" {
		if template, ok, _ := srv.loadTemplateRowByID(row["template_id"]); ok {
			row["template_name"] = cleanString(template["name"])
			row["template"] = template
		}
	}
	if uintValue(row["user_id"]) != 0 {
		if user, err := srv.getUserService().GetByID(row["user_id"]); err == nil {
			userRow := srv.getUserService().ToRowWithTeam(*user)
			if row["user"] == nil {
				row["user"] = userRow
			}
			if cleanString(row["user_name"]) == "" {
				row["user_name"] = cleanString(userRow["username"])
			}
		}
	}
	if uintValue(row["host_id"]) != 0 && row["host"] == nil {
		if host, err := srv.getVirtualizationHostService().GetByID(row["host_id"]); err == nil {
			row["host"] = srv.getVirtualizationHostService().ToRow(*host)
			if cleanString(row["host_name"]) == "" {
				row["host_name"] = host.Name
			}
		}
	}
	if cleanString(row["host_name"]) == "" && uintValue(row["host_id"]) != 0 {
		if host, err := srv.getVirtualizationHostService().GetByID(row["host_id"]); err == nil {
			row["host_name"] = host.Name
			row["host"] = srv.getVirtualizationHostService().ToRow(*host)
		}
	}
}

func containerRuntimeListValue(value any) models.ContainerRuntimeList {
	text := jsonString(value)
	if text == "" {
		return models.ContainerRuntimeList{}
	}
	var items models.ContainerRuntimeList
	if err := items.Scan(text); err != nil {
		return models.ContainerRuntimeList{}
	}
	return items
}

func templateFilesDir(templateID string) string {
	return filepath.Join("static", "template-files", cleanPathSegment(templateID))
}

func cleanPathSegment(value string) string {
	value = strings.TrimSpace(filepath.Base(value))
	if value == "." || value == "/" || value == `\` {
		return ""
	}
	return value
}

func defaultString(value any, fallback string) string {
	text := strings.TrimSpace(stringifyID(value))
	if text == "" {
		return fallback
	}
	return text
}

// adminVirtualizationTemplateBuildStreamHandler 提供模板构建的实时日志流 (SSE)
func adminVirtualizationTemplateBuildStreamHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task := latestTemplateBuildTask(id)
	if task == nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "build task not found"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		println("[SSE] ERROR: streaming unsupported")
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "streaming unsupported"})
		return
	}
	send := func(data map[string]any) {
		payload, err := json.Marshal(data)
		if err != nil {
			return
		}
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(payload)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	sent := 0
	for {
		logs, status, done := snapshotTemplateBuildTask(task)
		for sent < len(logs) {
			send(map[string]any{"status": status, "log": logs[sent], "template_id": id, "task_id": task.ID})
			sent++
		}
		if done {
			if status == "completed" {
				send(map[string]any{"status": "completed", "log": "模板构建完成", "template_id": id, "task_id": task.ID})
			} else if status == "failed" || status == "error" {
				send(map[string]any{"status": "failed", "log": "模板构建失败", "template_id": id, "task_id": task.ID})
			}
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(400 * time.Millisecond):
		}
	}
}

// completeTemplateBuild 完成模板构建并创建日志和通知
func (srv *Server) completeTemplateBuild(templateID string, template models.Templates, success bool, message string) {
	now := time.Now().Format(time.RFC3339)
	status := "completed"
	buildStatus := "completed"
	if !success {
		status = "error"
		buildStatus = "failed"
	}

	template.Status = models.TemplateStatus(status)
	template.UpdatedAt = parseRFC3339OrNow(now)
	_ = srv.saveTemplateBuildResult(templateID, template)

	// 创建构建日志
	if service := srv.getMiscResourceService(); service != nil {
		_, _ = service.CreateBuildLog(map[string]any{
			"template_id":   template.Id,
			"template_name": template.Name,
			"status":        buildStatus,
			"message":       message,
		})
	}

	// 发送通知给管理员
	notificationTitle := "模板构建成功"
	notificationType := "success"
	if !success {
		notificationTitle = "模板构建失败"
		notificationType = "error"
	}
	srv.createAdminNotification(notificationTitle, template.Name, templateID, notificationType, message)
}

func (srv *Server) createTemplateBuildTask(templateID string, template models.Templates) *templateBuildTask {
	now := time.Now().Format(time.RFC3339)
	task := &templateBuildTask{
		ID:         fmt.Sprintf("tpl-build-%s-%d", templateID, time.Now().UnixNano()),
		TemplateID: templateID,
		Template:   template,
		Status:     "building",
		Logs:       []string{},
		CreatedAt:  now,
		UpdatedAt:  now,
		Done:       make(chan struct{}),
	}
	templateBuildTasks.mu.Lock()
	templateBuildTasks.tasks[templateID] = task
	templateBuildTasks.mu.Unlock()
	_ = srv.markTemplateBuilding(templateID, now)
	if service := srv.getMiscResourceService(); service != nil {
		_, _ = service.CreateBuildLog(map[string]any{
			"template_id":   template.Id,
			"template_name": template.Name,
			"task_id":       task.ID,
			"status":        "building",
			"message":       "template build started",
		})
	}
	return task
}

func latestTemplateBuildTask(templateID string) *templateBuildTask {
	templateBuildTasks.mu.RLock()
	defer templateBuildTasks.mu.RUnlock()
	return templateBuildTasks.tasks[templateID]
}

func appendTemplateBuildTaskLog(task *templateBuildTask, line string) {
	task.mu.Lock()
	defer task.mu.Unlock()
	task.Logs = append(task.Logs, line)
	task.UpdatedAt = time.Now().Format(time.RFC3339)
}

func finishTemplateBuildTask(task *templateBuildTask, status string) {
	task.mu.Lock()
	task.Status = status
	task.UpdatedAt = time.Now().Format(time.RFC3339)
	task.mu.Unlock()
	select {
	case <-task.Done:
	default:
		close(task.Done)
	}
}

func snapshotTemplateBuildTask(task *templateBuildTask) ([]string, string, bool) {
	task.mu.RLock()
	defer task.mu.RUnlock()
	logs := append([]string(nil), task.Logs...)
	done := task.Status == "completed" || task.Status == "failed" || task.Status == "error"
	return logs, task.Status, done
}

func serializeTemplateBuildTask(task *templateBuildTask) map[string]any {
	task.mu.RLock()
	defer task.mu.RUnlock()
	return map[string]any{
		"id":          task.ID,
		"template_id": task.TemplateID,
		"status":      task.Status,
		"created_at":  task.CreatedAt,
		"updated_at":  task.UpdatedAt,
		"log_count":   len(task.Logs),
	}
}

func (srv *Server) runTemplateBuildTask(task *templateBuildTask) {
	template := task.Template
	err := services.TemplateBuild(&template, &services.TemplateBuildOptions{
		Logf: func(format string, args ...any) {
			appendTemplateBuildTaskLog(task, fmt.Sprintf(format, args...))
		},
	})
	if err != nil {
		appendTemplateBuildTaskLog(task, "模板构建失败: "+err.Error())
		srv.completeTemplateBuild(task.TemplateID, template, false, err.Error())
		finishTemplateBuildTask(task, "failed")
		return
	}

	task.Template = template
	appendTemplateBuildTaskLog(task, "模板构建完成")
	srv.completeTemplateBuild(task.TemplateID, template, true, "模板构建完成")
	finishTemplateBuildTask(task, "completed")
}

func (srv *Server) saveTemplateBuildResult(templateID string, template models.Templates) error {
	db := srv.db
	if db == nil {
		return fmt.Errorf("db unavailable")
	}
	updateModel := models.Templates{
		UpdatedAt: template.UpdatedAt,
		Status:    template.Status,
		Ports:     services.NormalizeStringSlice(template.Ports),
		Services:  template.Services,
	}
	return srv.db.Model(&models.Templates{}).Where("id = ?", templateID).Updates(&updateModel).Error
}

func (srv *Server) markTemplateBuilding(templateID string, now string) error {
	return srv.db.Model(&models.Templates{}).Where("id = ?", templateID).Updates(&models.Templates{
		Status:    "building",
		UpdatedAt: parseRFC3339OrNow(now),
	}).Error
}

func parseRFC3339OrNow(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}

// createAdminNotification 创建管理员通知
func (srv *Server) createAdminNotification(title string, templateName any, templateID string, notificationType string, message string) {
	service := srv.getNotificationService()
	if service == nil {
		return
	}
	templateNameStr := stringifyID(templateName)
	for _, receiverID := range service.ReceiverIDsByAudience("admin") {
		_, _ = service.Create(map[string]any{
			"title":        title,
			"content":      "模板 [" + templateNameStr + "] " + message,
			"message_type": notificationType,
			"audience":     "admin",
			"receiver_id":  receiverID,
			"related_type": "template",
			"related_id":   uintValue(templateID),
		})
	}
}
