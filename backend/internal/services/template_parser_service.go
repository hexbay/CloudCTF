package services

import (
	"cloudctf/backend/internal/models"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComposeServiceBuildInfo struct {
	Name       string
	Image      string
	Dockerfile string
}

type TemplateMetadata struct {
	CVEID          string
	Summary        string
	Description    string
	ReproduceSteps string
	Severity       string
	ReferenceLinks []string
}

type DiscoveredTemplate struct {
	Name           string
	RepositoryPath string
	Path           string
	ComposeFile    string
	Description    string
	Tags           []string
	Metadata       TemplateMetadata
	Services       []models.TemplateService
	Ports          []string
}

type RepoTemplateParser interface {
	Engine() string
	Discover(repoDir, scanRoot, repoName string) ([]DiscoveredTemplate, error)
}

type TemplateParserService struct{}

func NewTemplateParserService() *TemplateParserService {
	return &TemplateParserService{}
}

func (s *TemplateParserService) GetRepoTemplateParser(tags []string) RepoTemplateParser {
	normalized := normalizedTagSet(tags)
	base := defaultRepoTemplateParser{service: s, tags: tags}
	if normalized["vulhub"] {
		return vulhubRepoTemplateParser{defaultRepoTemplateParser: base}
	} else {
		return ctfdbRepoTemplateParser{defaultRepoTemplateParser: base}
	}
}

func (s *TemplateParserService) ParseComposeServicesForBuild(composeFile string) []ComposeServiceBuildInfo {
	if composeFile == "" {
		return nil
	}

	var compose struct {
		Services map[string]struct {
			Image            string `yaml:"image"`
			Dockerfile       string `yaml:"dockerfile"`
			DockerfileInline string `yaml:"dockerfile_inline"`
			Build            any    `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(composeFile), &compose); err != nil {
		return nil
	}

	services := make([]ComposeServiceBuildInfo, 0, len(compose.Services))
	for name, service := range compose.Services {
		dockerfile := strings.TrimSpace(service.DockerfileInline)
		if dockerfile == "" {
			dockerfile = strings.TrimSpace(service.Dockerfile)
		}
		if dockerfile == "" {
			switch build := service.Build.(type) {
			case string:
				dockerfile = strings.TrimSpace(build)
			case map[string]any:
				if inline, ok := build["dockerfile_inline"].(string); ok {
					dockerfile = strings.TrimSpace(inline)
				} else if file, ok := build["dockerfile"].(string); ok {
					dockerfile = strings.TrimSpace(file)
				}
			}
		}
		services = append(services, ComposeServiceBuildInfo{
			Name:       name,
			Image:      strings.TrimSpace(service.Image),
			Dockerfile: dockerfile,
		})
	}

	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	return services
}

func (s *TemplateParserService) ExtractTemplateRuntimeDetails(composeFile string) ([]map[string]any, []string) {
	var doc struct {
		Services map[string]map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(composeFile), &doc); err != nil || len(doc.Services) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	services := make([]map[string]any, 0, len(names))
	ports := []string{}
	for _, name := range names {
		def := doc.Services[name]
		service := map[string]any{"name": name}
		if image := cleanScalar(def["image"]); image != "" {
			service["image"] = image
		}
		rawPorts := normalizeTemplatePortList(def["ports"])
		if len(rawPorts) > 0 {
			containerPorts := normalizeServicePortList(rawPorts)
			service["ports"] = containerPorts
			ports = append(ports, containerPorts...)
		}
		services = append(services, service)
	}
	return services, uniqueStrings(ports)
}

func (s *TemplateParserService) ExtractMetadataRuntimeDetails(dir string, metadata map[string]any, scalars map[string]string) ([]map[string]any, []string) {
	ports := normalizeTemplatePortList(firstNonNil(
		metadata["ports"],
		metadata["port"],
		metadata["container_port"],
		metadata["containerPort"],
		scalars["ports"],
		scalars["port"],
		scalars["container_port"],
		scalars["containerPort"],
	))
	containerPorts := normalizeServicePortList(ports)
	if len(ports) == 0 {
		return nil, containerPorts
	}

	service := map[string]any{
		"name":  firstNonEmptyString(scalars["name"], filepath.Base(dir)),
		"ports": containerPorts,
	}
	if image := firstNonEmptyString(scalars["image"], cleanScalar(metadata["image"])); image != "" {
		service["image"] = image
	}
	return []map[string]any{service}, containerPorts
}

func (s *TemplateParserService) MergeTemplateRuntimeDetails(primaryServices []map[string]any, primaryPorts []string, fallbackServices []map[string]any, fallbackPorts []string) ([]map[string]any, []string) {
	if len(primaryServices) == 0 {
		return fallbackServices, fallbackPorts
	}
	if len(primaryPorts) > 0 || len(fallbackPorts) == 0 {
		return primaryServices, primaryPorts
	}

	fallbackPortList := []string{}
	if len(fallbackServices) > 0 {
		fallbackPortList = normalizeStringSlice(fallbackServices[0]["ports"])
	}
	if len(fallbackPortList) == 0 {
		fallbackPortList = normalizeStringSlice(fallbackPorts)
	}
	if len(fallbackPortList) > 0 {
		primaryServices[0]["ports"] = fallbackPortList
	}
	return primaryServices, uniqueStrings(fallbackPorts)
}

func cleanScalar(value any) string {
	return strings.Trim(strings.TrimSpace(stringifyAny(value)), `"'`)
}

func stringifyAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return strings.TrimSpace(strings.ReplaceAll(fmt.Sprint(typed), "\r", ""))
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		return value
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeTemplatePortList(value any) []string {
	result := []string{}
	switch typed := value.(type) {
	case nil:
	case []any:
		for _, item := range typed {
			result = append(result, normalizeTemplatePortList(item)...)
		}
	case []string:
		for _, item := range typed {
			if port := normalizePortString(item); port != "" {
				result = append(result, port)
			}
		}
	case map[string]any:
		target := normalizePortValue(typed["target"])
		published := normalizePortValue(typed["published"])
		protocol := firstNonEmptyString(cleanScalar(typed["protocol"]), "tcp")
		if target != "" {
			if !strings.Contains(target, "/") {
				target += "/" + protocol
			}
			if published != "" {
				result = append(result, published+":"+target)
			} else {
				result = append(result, target)
			}
		}
	default:
		if port := normalizePortItem(typed); port != "" {
			result = append(result, port)
		}
	}
	return uniqueStrings(result)
}

func normalizeServicePortList(ports []string) []string {
	result := make([]string, 0, len(ports))
	for _, item := range ports {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		segments := strings.Split(clean, ":")
		containerPort := strings.TrimSpace(segments[len(segments)-1])
		containerPort = parserPortNumber(containerPort)
		if containerPort != "" {
			result = append(result, containerPort)
		}
	}
	return uniqueStrings(result)
}

func normalizePortItem(value any) string {
	switch typed := value.(type) {
	case string:
		return normalizePortString(typed)
	case int, int64, uint64, float64:
		return cleanScalar(typed)
	default:
		return normalizePortValue(typed)
	}
}

func normalizePortString(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

func normalizePortValue(value any) string {
	text := cleanScalar(value)
	if text == "" {
		return ""
	}
	if strings.Contains(text, ":") {
		parts := strings.Split(text, ":")
		if len(parts) >= 2 {
			text = parts[len(parts)-2]
		} else {
			text = parts[0]
		}
	}
	if strings.Contains(text, "/") {
		text = strings.Split(text, "/")[0]
	}
	return strings.TrimSpace(text)
}

func mergePortMappings(target map[string]any, ports []string) {
	for _, item := range ports {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		segments := strings.Split(clean, ":")
		containerPort := strings.TrimSpace(segments[len(segments)-1])
		hostPort := ""
		if len(segments) > 1 {
			hostPort = strings.TrimSpace(segments[len(segments)-2])
		}
		if hostPort == "" {
			hostPort = parserPortNumber(containerPort)
		}
		if hostPort != "" && containerPort != "" {
			target[hostPort] = parserNormalizeContainerPort(containerPort)
		}
	}
}

func normalizeStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := cleanScalar(item); text != "" {
				result = append(result, text)
			}
		}
		return uniqueStrings(result)
	case map[string]any:
		result := make([]string, 0, len(typed))
		for key := range typed {
			if text := cleanScalar(key); text != "" {
				result = append(result, text)
			}
		}
		return uniqueStrings(result)
	default:
		if text := cleanScalar(value); text != "" {
			return []string{text}
		}
		return nil
	}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func parserPortNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "/"); idx >= 0 {
		value = value[:idx]
	}
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		value = parts[len(parts)-1]
	}
	return strings.TrimSpace(value)
}

func parserNormalizeContainerPort(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "/") {
		return value + "/tcp"
	}
	return value
}

func scanFile(directory string, files []string) []string {
	//	查找指定文件
	var targets []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		fileName := strings.ToLower(info.Name())
		for _, target := range files {
			if fileName == target {
				targets = append(targets, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to scan files", "directory", directory, "error", err)
	}
	return targets

}

type defaultRepoTemplateParser struct {
	service *TemplateParserService
	tags    []string
}

func (p defaultRepoTemplateParser) Engine() string {
	return "default"
}

type ctfdbRepoTemplateParser struct {
	defaultRepoTemplateParser
}

func (p ctfdbRepoTemplateParser) Engine() string {
	return "ctfdb"
}

func (p ctfdbRepoTemplateParser) Discover(repoDir, scanRoot, repoName string) ([]DiscoveredTemplate, error) {
	if _, err := os.Stat(scanRoot); err != nil {
		return nil, fmt.Errorf("templates path not found: %s", scanRoot)
	}
	var results []DiscoveredTemplate
	metaPaths := scanFile(scanRoot, []string{"metadata.yml", "metadata.yaml"})
	for _, metaPath := range metaPaths {
		dir := filepath.Dir(metaPath)
		metadata := readRepoMetadata(dir)
		if metadata == nil {
			slog.Warn("Parser MetaData Error")
			continue
		}
		relDir, err := filepath.Rel(repoDir, dir)
		if err != nil {
			slog.Warn("Parse File Rel Error")
			continue
		}
		relDir = filepath.ToSlash(relDir)
		// 判断是否有Dockerfile
		DockerFilePath := firstExistingFile(dir, "Dockerfile")
		results = append(results, DiscoveredTemplate{
			Name:           metadata.Name,
			RepositoryPath: relDir,
			Path:           dir,
			Description:    metadata.Description,
			Tags:           append(strings.Split(metadata.Cve, ","), "ctfdb", "ctf"),
			Services: []models.TemplateService{
				{
					Name:       metadata.Name,
					Image:      metadata.Image,
					Ports:      []string{},
					Dockerfile: DockerFilePath,
				},
			},
		})
	}
	return results, nil
}

type vulhubRepoTemplateParser struct {
	defaultRepoTemplateParser
}

func (p vulhubRepoTemplateParser) Engine() string {
	return "vulhub"
}

func (p vulhubRepoTemplateParser) Discover(repoDir, scanRoot, repoName string) ([]DiscoveredTemplate, error) {
	if _, err := os.Stat(scanRoot); err != nil {
		return nil, fmt.Errorf("templates path not found: %s", scanRoot)
	}
	var results []DiscoveredTemplate
	composeFiles := scanFile(scanRoot, []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"})
	for _, composeFile := range composeFiles {
		contentBytes, err := os.ReadFile(composeFile)
		if err != nil {
			return nil, err
		}
		content := string(contentBytes)
		dir := filepath.Dir(composeFile)
		relDir, err := filepath.Rel(repoDir, dir)
		if err != nil {
			return nil, err
		}
		relDir = filepath.ToSlash(relDir)
		if relDir == "." {
			relDir = ""
		}
		composeRel, err := filepath.Rel(repoDir, composeFile)
		if err != nil {
			return nil, err
		}
		metadata := readRepoMetadata(dir)
		if metadata == nil {
			metadata = &RepoMetaData{}
			slog.Warn("repo metadata not found", "dir", dir)
		}
		services, ports := parserDockerComposeFromContent(content)
		if len(services) == 0 && strings.TrimSpace(metadata.Image) != "" {
			services = []models.TemplateService{
				{
					Name:  buildTemplateName(repoName, relDir),
					Image: metadata.Image,
				},
			}
		}
		name := firstNonEmptyString(metadata.Name, buildTemplateName(repoName, relDir))
		item := DiscoveredTemplate{
			Name:           name,
			RepositoryPath: relDir,
			Path:           filepath.ToSlash(composeRel),
			ComposeFile:    content,
			Description:    metadata.Description,
			Tags:           append(strings.Split(metadata.Cve, ","), "vulhub", "vul"),
			Services:       services,
			Ports:          ports,
		}
		item.Metadata = extractVulhubMetadata(dir, item.Name)
		if item.Description == "" {
			item.Description = firstNonEmptyString(item.Metadata.Summary, item.Metadata.Description)
		}
		results = append(results, item)
	}
	return results, nil
}

func toModelTemplateServices(items []map[string]any) []models.TemplateService {
	services := make([]models.TemplateService, 0, len(items))
	for _, item := range items {
		service := models.TemplateService{
			Name:  cleanScalar(item["name"]),
			Image: cleanScalar(item["image"]),
			Ports: normalizeStringSlice(item["ports"]),
		}
		services = append(services, service)
	}
	return services
}

func buildTemplateName(repoName, relDir string) string {
	if strings.TrimSpace(relDir) == "" {
		return repoName
	}
	return filepath.Base(relDir)
}

func firstComposeImage(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "image:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "image:")), `"'`)
		}
	}
	return ""
}

type RepoMetaData struct {
	Image       string `yaml:"image"`
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
	Cve         string `yaml:"cve"`
}

func readRepoMetadata(dir string) *RepoMetaData {
	path := firstExistingFile(dir, "metadata.yml", "metadata.yaml")
	if path == "" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var metadata RepoMetaData
	if err := yaml.Unmarshal(content, &metadata); err != nil {
		return nil
	}
	return &metadata
}

func readMetadataMap(dir string) map[string]any {
	path := firstExistingFile(dir, "metadata.yml", "metadata.yaml")
	if path == "" {
		return map[string]any{}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var metadata map[string]any
	if err := yaml.Unmarshal(content, &metadata); err != nil {
		return map[string]any{}
	}
	return metadata
}

func firstExistingFile(dir string, names ...string) string {
	lowerToName := map[string]string{}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lowerToName[strings.ToLower(entry.Name())] = entry.Name()
	}
	for _, name := range names {
		if actual, ok := lowerToName[strings.ToLower(name)]; ok {
			return filepath.Join(dir, actual)
		}
	}
	return ""
}

func extractVulhubMetadata(templateDir, templateName string) TemplateMetadata {
	metadata := TemplateMetadata{Severity: "UNKNOWN"}
	idPattern := regexp.MustCompile(`(?i)(CVE-\d{4}-\d{4,7}|CNVD-\d{4}-\d{4,7})`)
	if match := idPattern.FindString(templateName + " " + templateDir); match != "" {
		metadata.CVEID = strings.ToUpper(match)
		if strings.HasPrefix(metadata.CVEID, "CVE") {
			metadata.ReferenceLinks = append(metadata.ReferenceLinks, "https://cve.mitre.org/cgi-bin/cvename.cgi?name="+metadata.CVEID)
		} else if strings.HasPrefix(metadata.CVEID, "CNVD") {
			metadata.ReferenceLinks = append(metadata.ReferenceLinks, "https://www.cnvd.org.cn/flaw/show/"+metadata.CVEID)
		}
	}
	readmePath := firstExistingFile(templateDir, "README.zh-cn.md", "README.zh_CN.md", "README-zh.md", "README.md", "README.MD")
	if readmePath == "" {
		return metadata
	}
	contentBytes, err := os.ReadFile(readmePath)
	if err != nil {
		return metadata
	}
	content := string(contentBytes)
	metadata.ReproduceSteps = content
	metadata.Summary = markdownSummary(content)
	metadata.ReferenceLinks = mergeStringLists(metadata.ReferenceLinks, markdownLinks(content))
	metadata.Severity = inferSeverity(content)
	return metadata
}

func markdownSummary(content string) string {
	lines := []string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "![") {
			if len(lines) > 0 {
				break
			}
			continue
		}
		lines = append(lines, line)
		if len(strings.Join(lines, " ")) > 200 {
			break
		}
	}
	summary := strings.Join(lines, " ")
	if len(summary) > 500 {
		return summary[:500]
	}
	return summary
}

func markdownLinks(content string) []string {
	linkPattern := regexp.MustCompile(`\[[^\]]+\]\((https?://[^)]+)\)`)
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	links := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, strings.TrimSpace(match[1]))
		}
	}
	return links
}

func inferSeverity(content string) string {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "critical") || strings.Contains(content, "严重") || strings.Contains(content, "危急"):
		return "CRITICAL"
	case strings.Contains(lower, "high") || strings.Contains(content, "高危"):
		return "HIGH"
	case strings.Contains(lower, "medium") || strings.Contains(content, "中危") || strings.Contains(content, "中等"):
		return "MEDIUM"
	case strings.Contains(lower, "low") || strings.Contains(content, "低危"):
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

func normalizedTagSet(tags []string) map[string]bool {
	result := map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" {
			result[tag] = true
		}
	}
	return result
}

func mergeStringLists(lists ...[]string) []string {
	seen := map[string]bool{}
	merged := []string{}
	for _, list := range lists {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			key := strings.ToLower(item)
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, item)
		}
	}
	return merged
}

type Doc struct {
	Services map[string]map[string]any `yaml:"services"`
}

func parserDockerComposeFromContent(content string) ([]models.TemplateService, []string) {
	var doc Doc
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, []string{}
	}
	return parserDockerCompose(doc)
}

func parserDockerCompose(doc Doc) ([]models.TemplateService, []string) {
	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	services := make([]models.TemplateService, 0, len(names))
	ports := []string{}
	for _, name := range names {
		def := doc.Services[name]
		service := models.TemplateService{Name: name}
		service.Image = cleanScalar(def["image"])
		rawPorts := normalizeTemplatePortList(def["ports"])
		if len(rawPorts) > 0 {
			containerPorts := normalizeServicePortList(rawPorts)
			service.Ports = containerPorts
			ports = append(ports, containerPorts...)
		}
		services = append(services, service)
	}
	return services, uniqueStrings(ports)
}
