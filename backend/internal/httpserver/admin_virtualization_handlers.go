package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/services"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func (srv *Server) adminVirtualizationHostsHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	rows, total, err := srv.getVirtualizationHostService().List(current, pageSize, r.URL.Query())
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	for _, row := range rows {
		hostID, err := strconv.Atoi(stringifyID(row["id"]))
		if err != nil {
			continue
		}
		host, err := srv.getVirtualizationHostService().GetByID(hostID)
		if err != nil || host.HostType != "docker" {
			continue
		}

		ctx, cancel := context.WithTimeout(r.Context(), dockerRuntimeTimeout)
		resources, resolvedEndpoint, inspectErr := srv.inspectDockerHost(ctx, host.Endpoint)
		cancel()

		if inspectErr != nil {
			_ = srv.getVirtualizationHostService().UpdateRuntime(host, "error", map[string]any{"error": inspectErr.Error()})
			row["status"] = "error"
			row["last_check"] = host.LastCheck
			row["resources"] = map[string]any{"error": inspectErr.Error()}
			continue
		}

		if host.Endpoint != resolvedEndpoint {
			host.Endpoint = resolvedEndpoint
		}
		_ = srv.getVirtualizationHostService().UpdateRuntime(host, "online", resources)
		row["status"] = "online"
		row["endpoint"] = host.Endpoint
		row["last_check"] = host.LastCheck
		row["resources"] = resources
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     rows,
		"total":    total,
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) adminVirtualizationHostCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, err := srv.getVirtualizationHostService().Create(payload)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationHostUpdateHandler(w http.ResponseWriter, r *http.Request) {
	hostID := r.PathValue("id")
	if hostID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "id is required"})
		return
	}
	payload, ok := readJSONMap(w, r)
	if !ok {
		return
	}
	row, found := srv.getVirtualizationHostService().Update(hostID, payload)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationHostDeleteHandler(w http.ResponseWriter, r *http.Request) {
	hostID := r.PathValue("id")
	if hostID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "id is required"})
		return
	}
	if !srv.getVirtualizationHostService().Delete(hostID) {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminVirtualizationHostDetailHandler(w http.ResponseWriter, r *http.Request) {
	hostID := r.PathValue("id")
	if hostID == "" {
		hostID = resourceID(r.URL.Path)
	}
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}
	response.JSON(w, http.StatusOK, srv.getVirtualizationHostService().ToRow(*host))
}

func (srv *Server) adminVirtualizationHostCheckHandler(w http.ResponseWriter, r *http.Request) {
	hostID := r.PathValue("id")
	if hostID == "" {
		hostID = resourceID(r.URL.Path)
	}
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}

	if host.HostType != "docker" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "unsupported host type"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dockerRuntimeTimeout)
	resources, resolvedEndpoint, inspectErr := srv.inspectDockerHost(ctx, host.Endpoint)
	cancel()

	if inspectErr != nil {
		_ = srv.getVirtualizationHostService().UpdateRuntime(host, "error", map[string]any{"error": inspectErr.Error()})
		response.JSON(w, http.StatusOK, map[string]any{
			"status":  "error",
			"message": inspectErr.Error(),
		})
		return
	}

	if host.Endpoint != resolvedEndpoint {
		host.Endpoint = resolvedEndpoint
	}
	_ = srv.getVirtualizationHostService().UpdateRuntime(host, "online", resources)
	response.JSON(w, http.StatusOK, map[string]any{
		"status": "online",
		"info":   resources,
		"host":   srv.getVirtualizationHostService().ToRow(*host),
	})
}

func (srv *Server) adminVirtualizationHostImagesHandler(w http.ResponseWriter, r *http.Request) {
	hostID := r.PathValue("id")
	if hostID == "" {
		hostID = resourceID(r.URL.Path)
	}
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "resource not found"})
		return
	}

	if host.HostType != "docker" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "unsupported host type"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dockerRuntimeTimeout)
	rows, listErr := listDockerImages(ctx, host.Endpoint)
	cancel()
	if listErr != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": listErr.Error()})
		return
	}

	response.JSON(w, http.StatusOK, rows)
}

const dockerRuntimeTimeout = 5 * time.Second

func defaultDockerEndpoint() string {
	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); host != "" {
		return host
	}
	if runtime.GOOS == "windows" {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

func dockerEndpointCandidates(endpoint string) []string {
	seen := map[string]struct{}{}
	add := func(target string, result *[]string) {
		target = strings.TrimSpace(target)
		if target == "" {
			return
		}
		if _, ok := seen[target]; ok {
			return
		}
		seen[target] = struct{}{}
		*result = append(*result, target)
	}

	candidates := make([]string, 0, 4)
	isAuto := endpoint == "" || strings.EqualFold(endpoint, "auto")
	if isAuto {
		add(defaultDockerEndpoint(), &candidates)
	} else {
		add(endpoint, &candidates)
	}

	if !isAuto {
		return candidates
	}

	// WSL / Docker Desktop 常见兼容路径：
	// 1. 优先尊重 DOCKER_HOST
	// 2. Linux/容器内尝试 docker.sock
	// 3. Windows 原生尝试 named pipe
	envHost := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	add(envHost, &candidates)
	add("unix:///var/run/docker.sock", &candidates)
	add("npipe:////./pipe/docker_engine", &candidates)

	return candidates
}

func newDockerRuntimeClient(endpoint string) (*client.Client, error) {
	opts := []client.Opt{
		client.WithHost(endpoint),
		client.WithAPIVersionNegotiation(),
	}
	return client.NewClientWithOpts(opts...)
}

func (srv *Server) inspectDockerHost(ctx context.Context, endpoint string) (map[string]any, string, error) {
	var lastErr error
	for _, candidate := range dockerEndpointCandidates(endpoint) {
		resources, err := srv.inspectDockerHostOnce(ctx, candidate)
		if err == nil {
			return resources, candidate, nil
		}
		lastErr = err
	}
	return nil, "", lastErr
}

func (srv *Server) inspectDockerHostOnce(ctx context.Context, endpoint string) (map[string]any, error) {
	cli, err := newDockerRuntimeClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	info, err := cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: services.MapToDockerFilters("label", srv.managedContainerFilter()),
	})
	if err != nil {
		return nil, err
	}

	diskUsage, err := cli.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return nil, err
	}

	containersRunning := 0
	for _, c := range containers {
		if c.State == "running" {
			containersRunning++
		}
	}

	diskUsed := int64(0)
	for _, item := range diskUsage.Images {
		diskUsed += item.Size
	}
	for _, item := range diskUsage.Containers {
		diskUsed += item.SizeRw
	}
	for _, item := range diskUsage.Volumes {
		diskUsed += item.UsageData.Size
	}

	return map[string]any{
		"name":               info.Name,
		"os":                 strings.TrimSpace(fmt.Sprintf("%s %s", info.OperatingSystem, info.KernelVersion)),
		"os_type":            info.OSType,
		"architecture":       info.Architecture,
		"docker_version":     info.ServerVersion,
		"cpu_count":          info.NCPU,
		"cpu_usage":          0,
		"memory_total":       info.MemTotal,
		"memory_used":        0,
		"disk_total":         diskUsage.LayersSize,
		"disk_used":          diskUsed,
		"containers_running": containersRunning,
		"containers_total":   len(containers),
		"images_count":       len(diskUsage.Images),
	}, nil
}

func listDockerImages(ctx context.Context, endpoint string) ([]map[string]any, error) {
	var lastErr error
	for _, candidate := range dockerEndpointCandidates(endpoint) {
		rows, err := listDockerImagesOnce(ctx, candidate)
		if err == nil {
			return rows, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func listDockerImagesOnce(ctx context.Context, endpoint string) ([]map[string]any, error) {
	cli, err := newDockerRuntimeClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	images, err := cli.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]any, 0, len(images))
	for _, img := range images {
		repoTags := img.RepoTags
		if len(repoTags) == 0 {
			repoTags = []string{"<none>:<none>"}
		}

		// 解析第一个 tag 为 repository 和 tag
		firstTag := repoTags[0]
		repository := "<none>"
		tag := "<none>"
		if firstTag != "<none>:<none>" {
			parts := strings.SplitN(firstTag, ":", 2)
			if len(parts) >= 1 && parts[0] != "" {
				repository = parts[0]
			}
			if len(parts) >= 2 && parts[1] != "" {
				tag = parts[1]
			}
		}

		rows = append(rows, map[string]any{
			"id":          img.ID,
			"image_id":    strings.TrimPrefix(img.ID, "sha256:")[:12],
			"repository":  repository,
			"tag":         tag,
			"tags":        repoTags,
			"created":     img.Created,
			"size":        img.Size,
			"shared_size": img.SharedSize,
		})
	}

	return rows, nil
}

func (srv *Server) adminVirtualizationHostImagePullHandler(w http.ResponseWriter, r *http.Request) {
	host, err := srv.resolveDockerHostForRequest(r)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	imageName := cleanString(payload["image_name"])
	if imageName == "" {
		imageName = cleanString(payload["imageName"])
	}
	if imageName == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "image_name is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	logs, err := services.PullDockerImage(ctx, host.Endpoint, imageName)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"status": "completed",
		"image":  imageName,
		"logs":   logs,
	})
}

func (srv *Server) adminVirtualizationHostImageBuildHandler(w http.ResponseWriter, r *http.Request) {
	host, err := srv.resolveDockerHostForRequest(r)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	payload, _ := readOptionalJSONMap(r)
	tag := cleanString(payload["tag"])
	dockerfile := cleanString(payload["dockerfile"])
	if tag == "" || dockerfile == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "tag and dockerfile are required"})
		return
	}

	buildArgs := map[string]*string{}
	if rawArgs, ok := payload["build_args"].(map[string]any); ok {
		for key, value := range rawArgs {
			text := stringifyID(value)
			buildArgs[key] = &text
		}
	}
	if rawArgs, ok := payload["buildArgs"].(map[string]any); ok {
		for key, value := range rawArgs {
			text := stringifyID(value)
			buildArgs[key] = &text
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	logs, err := services.BuildDockerImage(ctx, host.Endpoint, dockerfile, tag, buildArgs)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"status": "completed",
		"tag":    tag,
		"logs":   logs,
	})
}

func (srv *Server) adminVirtualizationHostImageLoadHandler(w http.ResponseWriter, r *http.Request) {
	host, err := srv.resolveDockerHostForRequest(r)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "file is required"})
		return
	}
	defer file.Close()

	tag := r.FormValue("tag")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	logs, err := services.LoadDockerImage(ctx, host.Endpoint, file, tag)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"status": "completed",
		"tag":    tag,
		"logs":   logs,
	})
}

func (srv *Server) adminVirtualizationContainersHandler(w http.ResponseWriter, r *http.Request) {
	current := queryInt(r, "current", 1)
	pageSize := queryInt(r, "pageSize", 10)
	hostID := queryInt(r, "host_id", 0)
	nameQuery := strings.TrimSpace(r.URL.Query().Get("name"))
	statusQuery := strings.TrimSpace(r.URL.Query().Get("status"))

	hosts, err := srv.getVirtualizationHostService().ListDockerHosts(hostID)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}

	allRows := make([]map[string]any, 0)
	for _, host := range hosts {
		ctx, cancel := context.WithTimeout(r.Context(), dockerRuntimeTimeout)
		rows, listErr := services.ListDockerContainers(
			ctx,
			host.Endpoint,
			host.ID,
			host.Name,
			host.OutIP,
			services.MapToDockerFilters("label", srv.managedContainerFilter()),
		)
		cancel()
		if listErr != nil {
			continue
		}
		for _, row := range rows {
			if nameQuery != "" {
				text := strings.ToLower(stringifyID(row["name"]) + " " + stringifyID(row["image"]) + " " + stringifyID(row["id"]))
				if !strings.Contains(text, strings.ToLower(nameQuery)) {
					continue
				}
			}
			if statusQuery != "" && !strings.EqualFold(stringifyID(row["status"]), statusQuery) {
				continue
			}
			allRows = append(allRows, row)
		}
	}

	start := (current - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	end := start + pageSize
	if start > len(allRows) {
		start = len(allRows)
	}
	if end > len(allRows) {
		end = len(allRows)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"data":     allRows[start:end],
		"total":    len(allRows),
		"current":  current,
		"pageSize": pageSize,
	})
}

func (srv *Server) adminVirtualizationContainerCreateHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	hostID := stringifyID(payload["host_id"])
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "host not found"})
		return
	}
	name := cleanString(payload["name"])
	imageName := cleanString(payload["image"])
	if name == "" || imageName == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "name and image are required"})
		return
	}
	ports := map[string]string{}
	if raw, ok := payload["ports"].(map[string]any); ok {
		for hostPort, containerPort := range raw {
			ports[hostPort] = stringifyID(containerPort)
		}
	}
	envs := map[string]string{}
	if raw, ok := payload["environment"].(map[string]any); ok {
		for key, value := range raw {
			envs[key] = stringifyID(value)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	row, err := srv.createDockerContainerWithOptions(ctx, host.Endpoint, services.DockerContainerCreateOptions{
		Name:        name,
		Image:       imageName,
		Ports:       ports,
		Environment: envs,
		Labels:      srv.managedContainerLabels(nil),
	})
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	row["host_id"] = host.ID
	row["host_name"] = host.Name
	row["out_ip"] = host.OutIP
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationContainerDetailHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	row, err := srv.inspectDockerContainerAcrossHosts(r.Context(), containerID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationContainerUpdateHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	payload, ok := readOptionalJSONMap(r)
	if !ok {
		return
	}
	hostID := stringifyID(payload["host_id"])
	var host *virtualizationHostModel
	var err error
	if hostID != "" {
		host, err = srv.getVirtualizationHostService().GetByID(hostID)
	} else {
		host, err = srv.resolveDockerHostByContainerID(r.Context(), containerID)
	}
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := services.UpdateDockerContainer(ctx, host.Endpoint, containerID, payload); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	row, err := srv.inspectDockerContainerAcrossHosts(r.Context(), containerID)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]any{"success": true, "container_id": containerID})
		return
	}
	response.JSON(w, http.StatusOK, row)
}

func (srv *Server) adminVirtualizationContainerDeleteHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	host, err := srv.resolveDockerHostByContainerID(r.Context(), containerID)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := services.RemoveDockerContainer(ctx, host.Endpoint, containerID, srv.isManagedContainer); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "message": "删除成功"})
}

func (srv *Server) adminVirtualizationContainerActionHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	action := cleanString(payload["action"])
	containerID := stringifyID(payload["container_id"])
	hostID := stringifyID(payload["host_id"])
	var host *virtualizationHostModel
	var err error
	if hostID != "" {
		host, err = srv.getVirtualizationHostService().GetByID(hostID)
	} else {
		host, err = srv.resolveDockerHostByContainerID(r.Context(), containerID)
	}
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := services.RunDockerContainerAction(ctx, host.Endpoint, containerID, action, srv.isManagedContainer); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "action": action})
}

func (srv *Server) adminVirtualizationContainerBatchActionHandler(w http.ResponseWriter, r *http.Request) {
	payload, _ := readOptionalJSONMap(r)
	action := cleanString(payload["action"])
	hostID := stringifyID(payload["host_id"])
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "host not found"})
		return
	}
	rawIDs, _ := payload["container_ids"].([]any)
	results := make([]map[string]any, 0, len(rawIDs))
	successCount := 0
	failedCount := 0
	for _, rawID := range rawIDs {
		containerID := stringifyID(rawID)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		err := services.RunDockerContainerAction(ctx, host.Endpoint, containerID, action, srv.isManagedContainer)
		cancel()
		if err != nil {
			failedCount++
			results = append(results, map[string]any{
				"container_id": containerID,
				"success":      false,
				"message":      err.Error(),
			})
			continue
		}
		successCount++
		results = append(results, map[string]any{
			"container_id": containerID,
			"success":      true,
			"message":      "ok",
		})
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"success_count": successCount,
		"failed_count":  failedCount,
		"results":       results,
	})
}

func (srv *Server) resolveDockerHostForRequest(r *http.Request) (*virtualizationHostModel, error) {
	hostID := r.PathValue("id")
	if hostID == "" {
		hostID = resourceID(r.URL.Path)
	}
	host, err := srv.getVirtualizationHostService().GetByID(hostID)
	if err != nil {
		return nil, fmt.Errorf("host not found")
	}
	if host.HostType != "docker" {
		return nil, fmt.Errorf("unsupported host type")
	}
	return host, nil
}

func (srv *Server) inspectDockerContainerAcrossHosts(ctx context.Context, containerID string) (map[string]any, error) {
	host, err := srv.resolveDockerHostByContainerID(ctx, containerID)
	if err != nil {
		return nil, err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return services.InspectDockerContainer(timeoutCtx, host.Endpoint, containerID, host.ID, host.Name, host.OutIP)
}

func (srv *Server) resolveDockerHostByContainerID(ctx context.Context, containerID string) (*virtualizationHostModel, error) {
	return srv.getVirtualizationHostService().ResolveDockerHostByContainerID(ctx, containerID, func(timeoutCtx context.Context, host virtualizationHostModel, resolvedContainerID string) error {
		_, err := services.InspectDockerContainer(timeoutCtx, host.Endpoint, resolvedContainerID, host.ID, host.Name, host.OutIP)
		return err
	})
}

func (srv *Server) createDockerContainerWithOptions(ctx context.Context, endpoint string, options services.DockerContainerCreateOptions) (map[string]any, error) {
	options.Labels = srv.managedContainerLabels(options.Labels)
	return services.CreateDockerContainerWithOptions(ctx, endpoint, options, services.PullDockerImage)
}

type containerGroupRuntimeAdapter struct {
	srv *Server
}

func (a containerGroupRuntimeAdapter) RemoveContainer(ctx context.Context, endpoint, containerID string) error {
	return services.RemoveDockerContainer(ctx, endpoint, containerID, nil)
}

func (a containerGroupRuntimeAdapter) CreatePracticeContainer(ctx context.Context, spec services.PracticeContainerSpec) (map[string]any, error) {
	return services.CreateDockerContainerWithOptions(ctx, spec.Endpoint, services.DockerContainerCreateOptions{
		Name:  spec.Name,
		Image: spec.Image,
		Ports: challengeContainerPortMap(spec.PortSpecs),
		Labels: a.srv.managedContainerLabels(map[string]string{
			"cloudctf.kind":         "practice",
			"cloudctf.challenge_id": fmt.Sprint(spec.ChallengeID),
			"cloudctf.user_id":      fmt.Sprint(spec.UserID),
		}),
	}, services.PullDockerImage)
}

func (a containerGroupRuntimeAdapter) InspectImageExposedPorts(ctx context.Context, endpoint, image string) ([]string, error) {
	return services.InspectDockerImageExposedPorts(ctx, endpoint, image)
}

func (a containerGroupRuntimeAdapter) TemplatePortSpecs(templateID uint) []string {
	return a.srv.templatePortSpecs(templateID)
}

func (a containerGroupRuntimeAdapter) TemplateImageFromMetadata(templateID uint) string {
	return a.srv.templateImageFromMetadata(templateID)
}

func (srv *Server) GetContainerGroupService() *services.ContainerGroupService {
	return srv.currentServices().containerGroupService
}

func (srv *Server) managedContainerLabels(extra map[string]string) map[string]string {
	cfg := srv.currentConfig()
	labels := map[string]string{
		cfg.ContainerLabelKey:   cfg.ContainerLabelValue,
		cfg.ContainerManagedKey: cfg.ContainerManagedVal,
	}
	for key, value := range extra {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		labels[key] = value
	}
	return labels
}

func (srv *Server) managedContainerFilter() []string {
	cfg := srv.currentConfig()
	items := make([]string, 0, 2)
	if key := strings.TrimSpace(cfg.ContainerLabelKey); key != "" {
		items = append(items, key+"="+cfg.ContainerLabelValue)
	}
	if key := strings.TrimSpace(cfg.ContainerManagedKey); key != "" {
		items = append(items, key+"="+cfg.ContainerManagedVal)
	}
	return items
}

func (srv *Server) isManagedContainer(labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}
	cfg := srv.currentConfig()
	return labels[cfg.ContainerLabelKey] == cfg.ContainerLabelValue &&
		labels[cfg.ContainerManagedKey] == cfg.ContainerManagedVal
}
