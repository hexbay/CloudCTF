package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

// TemplateRuntimeService 用于模板和运行时相关操作
type TemplateRuntimeService struct {
	db               *gorm.DB
	repositorySyncMu sync.Map
	proxy            string
}

var ErrResourceRepositorySyncInProgress = errors.New("resource repository sync is already in progress")

type TemplateBuildOptions struct {
	Logf func(format string, args ...any)
}

type TemplateStartOptions struct {
	GroupID             string
	Endpoint            string
	PublicHost          string
	NetworkName         string
	ManagedLabels       func(map[string]string) map[string]string
	ManagedCheck        func(map[string]string) bool
	PullMissing         func(context.Context, string, string) ([]string, error)
	ResolveComposeImage func(string) string
}

func NewTemplateRuntimeService(db *gorm.DB, proxy ...string) *TemplateRuntimeService {
	service := &TemplateRuntimeService{db: db}
	if len(proxy) > 0 {
		service.proxy = strings.TrimSpace(proxy[0])
	}
	return service
}

func StartTemplateContainers(ctx context.Context, template models.Templates, opts TemplateStartOptions) ([]map[string]any, error) {
	launchServices := templateLaunchServices(template, opts.ResolveComposeImage)
	if len(launchServices) == 0 {
		return nil, errors.New("template has no runnable services")
	}
	networkName, err := ensureTemplateRuntimeNetwork(ctx, opts, len(launchServices))
	if err != nil {
		return nil, err
	}
	started := make([]map[string]any, 0, len(launchServices))
	for index, service := range topoSortTemplateLaunchServices(launchServices) {
		row, err := startTemplateServiceContainer(ctx, template, service, index, opts, networkName, len(launchServices))
		if err != nil {
			cleanupStartedTemplateContainers(ctx, opts.Endpoint, started, opts.ManagedCheck)
			return nil, fmt.Errorf("failed to start service %s: %w", service.Name, err)
		}
		started = append(started, row)
	}
	return started, nil
}

func ensureTemplateRuntimeNetwork(ctx context.Context, opts TemplateStartOptions, serviceCount int) (string, error) {
	if serviceCount <= 1 {
		return "", nil
	}
	networkName := strings.TrimSpace(opts.NetworkName)
	err := EnsureDockerNetwork(ctx, opts.Endpoint, networkName, applyTemplateManagedLabels(opts.ManagedLabels, map[string]string{
		"com.cloudctf.resource": "network",
	}))
	if err != nil {
		return "", err
	}
	return networkName, nil
}

func startTemplateServiceContainer(
	ctx context.Context,
	template models.Templates,
	service models.TemplateService,
	index int,
	opts TemplateStartOptions,
	networkName string,
	serviceCount int,
) (map[string]any, error) {
	row, err := CreateDockerContainerWithOptions(ctx, opts.Endpoint, DockerContainerCreateOptions{
		Name:        buildTemplateContainerName(opts.GroupID, service.Name, index),
		Image:       firstNonEmpty(strings.TrimSpace(service.Image), firstTemplateImage(template, opts.ResolveComposeImage)),
		Ports:       templateServicePortMap(service, true),
		Environment: templateServiceEnvironment(service),
		Command:     templateServiceCommand(service),
		NetworkName: networkNameIfNeeded(networkName, serviceCount),
		Labels: applyTemplateManagedLabels(opts.ManagedLabels, map[string]string{
			"cloudctf.group_id":    opts.GroupID,
			"cloudctf.template_id": fmt.Sprint(template.Id),
			"cloudctf.service":     strings.TrimSpace(service.Name),
		}),
	}, opts.PullMissing)
	if err != nil {
		return nil, err
	}
	row["targets"] = challengeContainerTargetsFromDockerPorts(opts.PublicHost, dockerPortMap(row["ports"]))
	row["service_name"] = strings.TrimSpace(service.Name)
	return row, nil
}

func (s *TemplateRuntimeService) SyncResourceRepository(
	logger *slog.Logger,
	parserService *TemplateParserService,
	hostService *VirtualizationHostService,
	vulnLabService *VulnLabService,
	repo models.ResourceRepo,
	now time.Time,
	deleteMissingTemplates bool,
) (int, int, error) {
	// 仓库同步的主入口：拉取代码、解析模板、写入 templates，并按需清理失效数据。
	if s == nil || s.db == nil {
		return 0, 0, fmt.Errorf("db unavailable")
	}
	if parserService == nil {
		return 0, 0, fmt.Errorf("template parser unavailable")
	}
	if !repo.Enabled {
		logger.Error("resource repository sync skipped because repository is disabled", "repo_id", repo.ID, "repo_name", repo.Name)
		return 0, 0, fmt.Errorf("resource repository is disabled")
	}
	if strings.TrimSpace(repo.RepositoryURL) == "" {
		logger.Error("resource repository sync skipped because repository_url is empty", "repo_id", repo.ID, "repo_name", repo.Name)
		return 0, 0, fmt.Errorf("repository_url is required")
	}
	release, err := s.beginResourceRepositorySync(repo.ID)
	if err != nil {
		return 0, 0, err
	}
	defer release()

	branch := strings.TrimSpace(repo.RepositoryBranch)
	if branch == "" {
		branch = "main"
	}
	logger.Info("resource sync step started",
		"repo_id", repo.ID,
		"repo_name", repo.Name,
		"branch", branch,
		"repository_url", repo.RepositoryURL,
		"templates_path", repo.TemplatesPath,
		"delete_missing_templates", deleteMissingTemplates,
	)

	repoDir := filepath.Join("data", "resource-repositories", strconv.FormatUint(uint64(repo.ID), 10))
	logger.Info("resource sync step: cloning or updating repository snapshot", "repo_dir", repoDir, "branch", branch)
	if err := syncGitRepositoryWithProxy(logger, repoDir, repo.RepositoryURL, branch, s.proxy); err != nil {
		logger.Error("resource sync step failed during git sync", "repo_id", repo.ID, "repo_dir", repoDir, "branch", branch, "error", err)
		return 0, 0, err
	}

	scanRoot := repoDir
	if strings.TrimSpace(repo.TemplatesPath) != "" {
		scanRoot = filepath.Join(repoDir, filepath.FromSlash(strings.TrimSpace(repo.TemplatesPath)))
	}
	parser := parserService.GetRepoTemplateParser(repo.TagList())
	logger.Info("resource sync step: scanning templates", "scan_root", scanRoot, "parser_engine", parser.Engine(), "tags", repo.TagList())
	templates, err := parser.Discover(repoDir, scanRoot, repo.Name)
	if err != nil {
		logger.Error("resource sync step failed during template discovery", "repo_id", repo.ID, "scan_root", scanRoot, "error", err)
		return 0, 0, err
	}
	logger.Info("resource sync step: templates discovered", "count", len(templates), "scan_root", scanRoot)

	var resolvedHostID *uint
	if repo.HostID != nil && *repo.HostID > 0 {
		resolvedHostID = repo.HostID
	} else if hostService != nil {
		// 如果仓库没有显式绑定宿主机，则尽量回退到默认 docker 宿主机。
		if defaultHost, err := hostService.GetDefaultByType("docker"); err == nil && defaultHost != nil {
			resolvedHostID = &defaultHost.ID
			logger.Info("resource sync step: using default docker host", "host_id", defaultHost.ID, "host_name", defaultHost.Name)
		}
	}

	var existing []models.Templates
	if err := s.db.Model(&models.Templates{}).Find(&existing).Error; err != nil {
		return 0, 0, err
	}
	existingByPath := map[string]models.Templates{}
	for _, row := range existing {
		if row.Path != "" {
			existingByPath[row.Path] = row
		}
	}

	synced := 0
	seenPaths := map[string]bool{}
	for _, template := range templates {
		seenPaths[template.RepositoryPath] = true
		record, err := buildTemplateRecord(repo, template, resolvedHostID, now)
		if err != nil {
			return synced, 0, err
		}
		if len(template.Services) == 0 && shouldExtractComposeRuntime(repo.RepoType, template.ComposeFile) {
			// 某些仓库模板只有 compose 内容，这里补提取运行时服务和端口信息。
			services, ports := parserService.ExtractTemplateRuntimeDetails(template.ComposeFile)
			if len(services) > 0 || len(ports) > 0 {
			}
		}

		logger.Info("resource sync step: processing template",
			"name", template.Name,
			"repository_path", template.RepositoryPath,
			"compose_path", template.Path,
			"image", firstTemplateServiceImage(template.Services),
		)

		var templateID any
		if current, ok := existingByPath[template.RepositoryPath]; ok {
			// 同一仓库路径视为同一个模板，优先做更新。
			record.Id = current.Id
			record.CreatedAt = current.CreatedAt
			logger.Info("resource sync step: updating existing template", "template_id", current.Id, "name", template.Name)
			if err := s.db.Save(&record).Error; err != nil {
				return synced, 0, err
			}
			templateID = current.Id
		} else {
			record.CreatedAt = now
			if err := s.db.Create(&record).Error; err != nil {
				logger.Error("resource sync step failed while inserting template", "name", template.Name, "path", template.RepositoryPath, "error", err)
				return synced, 0, err
			}
			templateID = record.Id
			logger.Info("resource sync step: inserted new template", "name", template.Name, "path", template.RepositoryPath)
		}

		if id := stringifyID(templateID); id != "" {
			// 将仓库里的模板目录同步到本地工作目录，供后续构建/查看使用。
			templateDir := filepath.Join(repoDir, filepath.FromSlash(template.RepositoryPath))
			if err := syncTemplateWorkspace(templateDir, templateFilesDir(id)); err != nil {
				logger.Warn("resource sync step: failed to copy template workspace", "template_id", id, "path", template.RepositoryPath, "error", err)
			}
		}
		if shouldCreateVulnLab(repo.RepoType, repo.TagList()) {
			_ = createOrUpdateVulnLabFromTemplate(vulnLabService, templateID, template, now)
		}
		synced++
	}

	deleted := 0
	if deleteMissingTemplates {
		// 如果仓库里已经不存在该模板，则把本地对应记录和文件一并清掉。
		logger.Info("resource sync step: pruning stale templates", "count", len(existingByPath))
		for path, row := range existingByPath {
			if seenPaths[path] {
				continue
			}
			id := stringifyID(row.Id)
			if id == "" {
				continue
			}
			result := s.db.Delete(&models.Templates{}, id)
			if result.Error == nil && result.RowsAffected > 0 {
				deleted++
				_ = os.RemoveAll(templateFilesDir(id))
				logger.Info("resource sync step: pruned stale template", "template_id", id, "path", path)
			}
		}
	}
	return synced, deleted, nil
}

func buildTemplateRecord(
	repo models.ResourceRepo,
	template DiscoveredTemplate,
	resolvedHostID *uint,
	now time.Time,
) (models.Templates, error) {
	// 先组装模板基础字段，运行时细节由辅助函数统一填充。
	record := models.Templates{
		Name:          template.Name,
		TemplateType:  repo.RepoType,
		ComposeFile:   template.ComposeFile,
		Path:          template.RepositoryPath,
		Description:   template.Description,
		HostId:        resolvedHostID,
		LastSyncAt:    now,
		RepositoryUrl: repo.RepositoryURL,
		Status:        "pending",
		Tags:          templateTagsForRepo(repo, template.Tags),
		UpdatedAt:     now,
		Ports:         uniqueStrings(template.Ports),
		Services:      template.Services,
	}
	return record, nil
}

func templateTagsForRepo(repo models.ResourceRepo, templateTags []string) []string {
	var repoTypeTag string
	switch strings.ToLower(strings.TrimSpace(repo.RepoType)) {
	case "ctf":
		repoTypeTag = "ctf"
	case "vul", "vuln", "vulnerability":
		repoTypeTag = "vul"
	}

	tags := make([]string, 0, len(repo.TagList())+len(templateTags)+1)
	for _, tag := range mergeStringLists(repo.TagList(), templateTags) {
		switch strings.ToLower(strings.TrimSpace(tag)) {
		case "ctf", "vul", "vuln", "vulnerability":
			continue
		default:
			tags = append(tags, tag)
		}
	}
	if repoTypeTag != "" {
		tags = append(tags, repoTypeTag)
	}
	return tags
}

func templateLaunchServices(template models.Templates, resolveComposeImage func(string) string) []models.TemplateService {
	if shouldExtractComposeRuntime(template.TemplateType, template.ComposeFile) {
		if len(template.Services) > 0 {
			return template.Services
		}
		parser := NewTemplateParserService()
		rows, _ := parser.ExtractTemplateRuntimeDetails(template.ComposeFile)
		return templateServicesFromRows(rows)
	}
	image := firstTemplateImage(template, resolveComposeImage)
	if image == "" {
		return nil
	}
	return []models.TemplateService{{
		Name:  firstNonEmpty(strings.TrimSpace(template.Name), "app"),
		Image: image,
		Ports: uniqueStrings(template.Ports),
	}}
}

func firstTemplateImage(template models.Templates, resolveComposeImage func(string) string) string {
	for _, service := range template.Services {
		if strings.TrimSpace(service.Image) != "" {
			return strings.TrimSpace(service.Image)
		}
	}
	if resolveComposeImage != nil {
		if image := strings.TrimSpace(resolveComposeImage(template.ComposeFile)); image != "" {
			return image
		}
	}
	return ""
}

func firstTemplateServiceImage(services []models.TemplateService) string {
	for _, service := range services {
		if strings.TrimSpace(service.Image) != "" {
			return strings.TrimSpace(service.Image)
		}
	}
	return ""
}

func templateServicePortMap(service models.TemplateService, randomHostPorts bool) map[string]string {
	result := map[string]string{}
	for _, item := range uniqueStrings(service.Ports) {
		hostPort, containerPort := parseComposePortMapping(item)
		if containerPort == "" {
			continue
		}
		if randomHostPorts {
			hostPort = PortNumber(containerPort)
		}
		result[hostPort] = containerPort
	}
	return result
}

func parseComposePortMapping(value string) (string, string) {
	text := strings.TrimSpace(strings.Trim(value, `"'`))
	if text == "" {
		return "", ""
	}
	if strings.Contains(text, "/") && !strings.Contains(text, ":") {
		return "", text
	}
	parts := strings.Split(text, ":")
	if len(parts) == 1 {
		return "", strings.TrimSpace(parts[0])
	}
	container := strings.TrimSpace(parts[len(parts)-1])
	host := strings.TrimSpace(parts[len(parts)-2])
	return host, container
}

func templateServiceEnvironment(service models.TemplateService) map[string]string {
	// 当前 TemplateService 还没有 environment 字段，先保持空实现，后续统一加字段再接入。
	return map[string]string{}
}

func templateServiceCommand(service models.TemplateService) []string {
	// 当前 TemplateService 还没有 command 字段，先保持空实现，后续统一加字段再接入。
	return nil
}

func buildTemplateContainerName(groupID, serviceName string, index int) string {
	base := slugifyTemplateName(serviceName)
	if base == "" {
		base = fmt.Sprintf("svc-%d", index+1)
	}
	return fmt.Sprintf("cg-%s-%s", strings.TrimSpace(groupID), base)
}

func topoSortTemplateLaunchServices(services []models.TemplateService) []models.TemplateService {
	return services
}

func networkNameIfNeeded(name string, count int) string {
	if count <= 1 {
		return ""
	}
	return name
}

func cleanupStartedTemplateContainers(ctx context.Context, endpoint string, started []map[string]any, managedCheck func(map[string]string) bool) {
	for _, item := range started {
		containerID := strings.TrimSpace(fmt.Sprint(item["container_id"]))
		if containerID == "" {
			containerID = strings.TrimSpace(fmt.Sprint(item["id"]))
		}
		if containerID == "" {
			continue
		}
		_ = RemoveDockerContainer(ctx, endpoint, containerID, managedCheck)
	}
}

func dockerPortMap(value any) map[string]any {
	if row, ok := value.(map[string]any); ok {
		return row
	}
	return map[string]any{}
}

func applyTemplateManagedLabels(managedLabels func(map[string]string) map[string]string, labels map[string]string) map[string]string {
	if managedLabels == nil {
		return labels
	}
	return managedLabels(labels)
}

func templateServicesFromRows(rows []map[string]any) []models.TemplateService {
	services := make([]models.TemplateService, 0, len(rows))
	for _, row := range rows {
		service := models.TemplateService{
			Name:  strings.TrimSpace(fmt.Sprint(row["name"])),
			Image: strings.TrimSpace(fmt.Sprint(row["image"])),
			Ports: normalizeTemplatePortValues(row["ports"]),
		}
		services = append(services, service)
	}
	return services
}

func normalizeTemplatePortValues(value any) []string {
	switch typed := value.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				items = append(items, text)
			}
		}
		return uniqueStrings(items)
	case map[string]any:
		items := make([]string, 0, len(typed))
		for hostPort, containerPort := range typed {
			host := strings.TrimSpace(hostPort)
			container := strings.TrimSpace(fmt.Sprint(containerPort))
			if host == "" && container == "" {
				continue
			}
			if host == "" {
				items = append(items, container)
				continue
			}
			if container == "" {
				items = append(items, host)
				continue
			}
			items = append(items, host+":"+container)
		}
		return uniqueStrings(items)
	default:
		return nil
	}
}

func NormalizeStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				items = append(items, text)
			}
		}
		return items
	case map[string]any:
		items := make([]string, 0, len(typed))
		for hostPort, containerPort := range typed {
			host := strings.TrimSpace(hostPort)
			container := strings.TrimSpace(fmt.Sprint(containerPort))
			if host == "" && container == "" {
				continue
			}
			if host == "" {
				items = append(items, container)
				continue
			}
			if container == "" {
				items = append(items, host)
				continue
			}
			items = append(items, host+":"+container)
		}
		return items
	default:
		return nil
	}
}

func NormalizeTemplateServiceRows(value any) []map[string]any {
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
		return nil
	}
}

func ParseComposePortMapping(value string) (string, string) {
	text := strings.TrimSpace(strings.Trim(value, `"'`))
	if text == "" {
		return "", ""
	}
	if strings.Contains(text, "/") && !strings.Contains(text, ":") {
		return "", text
	}
	parts := strings.Split(text, ":")
	if len(parts) == 1 {
		return "", strings.TrimSpace(parts[0])
	}
	container := strings.TrimSpace(parts[len(parts)-1])
	host := strings.TrimSpace(parts[len(parts)-2])
	return host, container
}

func FirstImageFromComposeBuild(composeFile string) string {
	parser := NewTemplateParserService()
	for _, service := range parser.ParseComposeServicesForBuild(composeFile) {
		if strings.TrimSpace(service.Image) != "" {
			return strings.TrimSpace(service.Image)
		}
	}
	return ""
}

func slugifyTemplateName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if len(value) > 24 {
		value = value[:24]
	}
	return strings.Trim(value, "-")
}

func syncTemplateWorkspace(srcDir, dstDir string) error {
	// 每次同步都用仓库最新内容覆盖模板工作目录，避免残留旧文件。
	if strings.TrimSpace(srcDir) == "" || strings.TrimSpace(dstDir) == "" {
		return nil
	}
	if err := os.RemoveAll(dstDir); err != nil {
		return err
	}
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := dstDir
		if rel != "." {
			target = filepath.Join(dstDir, rel)
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode())
	})
}

func syncGitRepository(logger *slog.Logger, targetDir, repositoryURL, branch string) error {
	return syncGitRepositoryWithProxy(logger, targetDir, repositoryURL, branch, "")
}

func syncGitRepositoryWithProxy(logger *slog.Logger, targetDir, repositoryURL, branch, proxy string) error {
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}
	if gitRepositoryHasValidHead(targetDir) {
		if logger != nil {
			logger.Info("git sync using existing repository", "repo_dir", targetDir, "repository_url", repositoryURL, "branch", branch)
			logGitInspect(logger, proxy, targetDir, "remote-url", "remote", "get-url", "origin")
			logGitInspect(logger, proxy, targetDir, "remote-branches", "ls-remote", "--heads", "origin")
			logGitInspect(logger, proxy, targetDir, "local-branches", "branch", "-a")
		}
		if err := runGitWithProxy(proxy, targetDir, "fetch", "--depth", "1", "origin", branch); err != nil {
			return err
		}
		if err := runGitWithProxy(proxy, targetDir, "checkout", branch); err != nil {
			return err
		}
		return runGitWithProxy(proxy, targetDir, "reset", "--hard", "origin/"+branch)
	}
	if logger != nil {
		logger.Info("git sync cloning repository into staging directory", "repo_dir", targetDir, "repository_url", repositoryURL, "branch", branch)
	}

	stagingRoot, err := os.MkdirTemp(parentDir, "."+filepath.Base(targetDir)+".clone-")
	if err != nil {
		return fmt.Errorf("create repository clone staging directory: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	stagingRepo := filepath.Join(stagingRoot, "repo")
	if err := runGitWithProxy(proxy, "", "clone", "--depth", "1", "--branch", branch, repositoryURL, stagingRepo); err != nil {
		return err
	}
	if !gitRepositoryHasValidHead(stagingRepo) {
		return fmt.Errorf("cloned repository snapshot is incomplete: %s", stagingRepo)
	}
	if err := replaceGitRepositoryDirectory(stagingRepo, targetDir); err != nil {
		return fmt.Errorf("activate repository snapshot: %w", err)
	}
	return nil
}

func (s *TemplateRuntimeService) beginResourceRepositorySync(repoID uint) (func(), error) {
	lockValue, _ := s.repositorySyncMu.LoadOrStore(repoID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	if !lock.TryLock() {
		return nil, ErrResourceRepositorySyncInProgress
	}
	return lock.Unlock, nil
}

func gitRepositoryHasValidHead(targetDir string) bool {
	if _, err := os.Stat(filepath.Join(targetDir, ".git")); err != nil {
		return false
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = targetDir
	return cmd.Run() == nil
}

func replaceGitRepositoryDirectory(stagingDir, targetDir string) error {
	parentDir := filepath.Dir(targetDir)
	backupDir := ""
	if _, err := os.Stat(targetDir); err == nil {
		backupDir = filepath.Join(parentDir, "."+filepath.Base(targetDir)+".previous-"+strconv.FormatInt(time.Now().UnixNano(), 10))
		if err := os.Rename(targetDir, backupDir); err != nil {
			return fmt.Errorf("move previous repository snapshot: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(stagingDir, targetDir); err != nil {
		if backupDir != "" {
			if rollbackErr := os.Rename(backupDir, targetDir); rollbackErr != nil {
				return errors.Join(fmt.Errorf("activate repository snapshot: %w", err), fmt.Errorf("restore previous repository snapshot: %w", rollbackErr))
			}
		}
		return err
	}
	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}
	return nil
}

func runGit(dir string, args ...string) error {
	return runGitWithProxy("", dir, args...)
}

func runGitWithProxy(proxy, dir string, args ...string) error {
	// 统一封装 git 调用，便于保留完整报错输出。
	cmd := exec.Command("git", args...)
	if cmd.Err != nil {
		if errors.Is(cmd.Err, exec.ErrNotFound) {
			return fmt.Errorf("resource repository sync requires Git: executable \"git\" was not found in PATH; install Git, add it to PATH, and restart CloudCTF")
		}
		return fmt.Errorf("prepare git %s: %w", strings.Join(args, " "), cmd.Err)
	}
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = gitCommandEnvironment(proxy)
	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			return fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
		return fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, details)
	}
	return nil
}

func gitCommandEnvironment(proxy string) []string {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return os.Environ()
	}
	env := make([]string, 0, len(os.Environ())+6)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY":
			continue
		default:
			env = append(env, entry)
		}
	}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		env = append(env, key+"="+proxy)
	}
	return env
}

func logGitInspect(logger *slog.Logger, proxy, dir, label string, args ...string) {
	if logger == nil {
		return
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitCommandEnvironment(proxy)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		logger.Warn("git inspect failed", "label", label, "dir", dir, "args", strings.Join(args, " "), "error", err, "output", text)
		return
	}
	logger.Info("git inspect", "label", label, "dir", dir, "args", strings.Join(args, " "), "output", text)
}

func templateFilesDir(templateID string) string {
	return filepath.Join("static", "template-files", strings.TrimSpace(templateID))
}

func shouldCreateVulnLab(repoType string, tags []string) bool {
	repoType = strings.ToLower(strings.TrimSpace(repoType))
	if repoType == "vul" || repoType == "vuln" || repoType == "vulnerability" {
		return true
	}
	set := normalizedTagSet(tags)
	return set["vul"] || set["vulhub"]
}

func createOrUpdateVulnLabFromTemplate(service *VulnLabService, templateID any, template DiscoveredTemplate, now time.Time) error {
	id := uintValue(templateID)
	if id == 0 || service == nil {
		return nil
	}
	metadata := template.Metadata
	summary := firstNonEmpty(metadata.Summary, template.Description, "基于模板 "+template.Name+" 自动创建")
	description := firstNonEmpty(metadata.Description, template.Description)
	severity := firstNonEmpty(metadata.Severity, "UNKNOWN")
	return service.CreateOrUpdateByTemplate(id, template.Name, metadata.CVEID, summary, description, metadata.ReproduceSteps, severity, metadata.ReferenceLinks, now)
}

func shouldExtractComposeRuntime(_ string, composeFile string) bool {
	return strings.TrimSpace(composeFile) != ""
}

func stringifyID(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func TemplateServiceBuild(cli *client.Client, templateID int, service models.TemplateService, opts *TemplateBuildOptions) error {
	// server.Dockerfile 是文件路径 不是内容
	imageName := strings.TrimSpace(service.Image)
	if imageName == "" {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			name = "app"
		}
		imageName = fmt.Sprintf("template-%d-%s", templateID, name)
	}
	ctx := context.Background()
	templateDir := serviceTemplateDir(templateID)
	dockerfilePath := resolveTemplateDockerfilePath(templateDir, service.Dockerfile)
	templateBuildLogf(opts,
		"service prepare name=%s image=%s dockerfile_path=%s template_dir=%s",
		service.Name,
		imageName,
		dockerfilePath,
		templateDir,
	)
	if dockerfilePath != "" {
		templateBuildLogf(opts, "building service=%s image=%s", service.Name, imageName)
		if err := os.MkdirAll(templateDir, 0o755); err != nil {
			return err
		}
		if err := syncTemplateDockerfile(templateDir, service.Dockerfile, dockerfilePath); err != nil {
			return err
		}
		return BuildDockerfileByPathWithLog(ctx, cli, imageName, templateDir, dockerfilePath, func(line string) {
			templateBuildLogf(opts, "%s", line)
		})
	}
	templateBuildLogf(opts, "pulling service=%s image=%s", service.Name, imageName)
	reader, err := cli.ImagePull(ctx, imageName, imagetypes.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		templateBuildLogf(opts, "%s", scanner.Text())
	}
	return scanner.Err()
}

func serviceTemplateDir(templateID int) string {
	return filepath.Join("static", "template-files", strconv.Itoa(templateID))
}

func resolveTemplateDockerfilePath(templateDir string, dockerfilePath string) string {
	path := filepath.ToSlash(strings.TrimSpace(dockerfilePath))
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(dockerfilePath) && !strings.Contains(path, "data/resource-repositories/") {
		return path
	}
	templateDir = filepath.Clean(templateDir)
	path = filepath.Clean(filepath.FromSlash(path))
	if rel, err := filepath.Rel(templateDir, path); err == nil {
		rel = filepath.ToSlash(rel)
		if rel != "." && !strings.HasPrefix(rel, "../") {
			return rel
		}
	}
	return filepath.Base(path)
}

func syncTemplateDockerfile(templateDir string, sourcePath string, dockerfilePath string) error {
	sourcePath = strings.TrimSpace(sourcePath)
	dockerfilePath = filepath.Clean(filepath.FromSlash(strings.TrimSpace(dockerfilePath)))
	if sourcePath == "" || dockerfilePath == "" {
		return nil
	}
	if !filepath.IsAbs(sourcePath) && !strings.Contains(filepath.ToSlash(sourcePath), "data/resource-repositories/") {
		return nil
	}
	sourcePath = filepath.Clean(filepath.FromSlash(sourcePath))
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(templateDir, dockerfilePath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, content, 0o644)
}

func TemplateBuild(template *models.Templates, opts *TemplateBuildOptions) error {
	logger := slog.Default().With("template_name", template.Name)
	logger.Info("template build task started",
		"host_id_raw", template.HostId,
		"template_type", template.TemplateType,
		"compose_file_len", len(template.ComposeFile),
		"services_count", len(template.Services),
	)
	templateBuildLogf(opts, "template build started name=%s services=%d", template.Name, len(template.Services))
	ctx := context.Background()
	if template.Host == nil {
		logger.Error("template build task failed: template host is not configured", "host_id_raw", template.HostId)
		return errors.New("template host is not configured")
	}
	if len(template.Services) == 0 {
		logger.Error("template build task failed: template has no services")
		return errors.New("template has no services")
	}

	cli, err := DockerClientForEndpoint(template.Host.Endpoint)
	if err != nil {
		logger.Error("template build task failed: create docker client failed", "endpoint", template.Host.Endpoint, "error", err)
		return err
	}
	defer cli.Close()
	buildMessages := make([]string, 0)
	for index, service := range template.Services {
		templateBuildLogf(opts,
			"service dispatch index=%d name=%s image=%s dockerfile_path=%s ports=%v",
			index,
			service.Name,
			service.Image,
			service.Dockerfile,
			service.Ports,
		)
		if err := TemplateServiceBuild(cli, int(template.Id), service, opts); err != nil {
			buildMessages = append(buildMessages, service.Name+": "+err.Error())
			logger.Error("template build task service failed",
				"service_name", service.Name,
				"image_name", service.Image,
				"error", err,
			)
			templateBuildLogf(opts, "service failed name=%s error=%s", service.Name, err.Error())
			return err
		}
		image, err := cli.ImageInspect(ctx, service.Image)
		if err != nil {
			logger.Error("Image Build Not Found")
			return errors.New("image Build Not Found")
		}
		if len(service.Ports) == 0 {
			servicePorts := dockerExposedPorts(image.Config.ExposedPorts)
			template.Services[index].Ports = servicePorts
		}
		template.Ports = uniqueStrings(append(template.Ports, template.Services[index].Ports...))
		logger.Info("template build task service completed", "service_name", service.Name, "image_name", service.Image)
		templateBuildLogf(opts, "service completed name=%s image=%s", service.Name, service.Image)
	}
	templateBuildLogf(opts, "template build completed name=%s", template.Name)
	return nil
}

func templateBuildLogf(opts *TemplateBuildOptions, format string, args ...any) {
	if opts != nil && opts.Logf != nil {
		opts.Logf(format, args...)
	}
}
