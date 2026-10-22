package services

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"

	"cloudctf/backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSyncResourceRepositoryWithCtfdbWeb2F12(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempRoot := t.TempDir()
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	dbPath := filepath.Join(tempRoot, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&models.Templates{}); err != nil {
		t.Fatal(err)
	}

	service := NewTemplateRuntimeService(db)
	parserService := NewTemplateParserService()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	now := time.Date(2026, 6, 6, 21, 33, 36, 0, time.FixedZone("UTC+8", 8*60*60))

	repo := models.ResourceRepo{
		ID:               1001,
		Name:             "ctfdb",
		RepoType:         "ctf",
		Provider:         "git",
		RepositoryURL:    "https://github.com/hexbay/ctfdb.git",
		RepositoryBranch: "main",
		TemplatesPath:    "web/web2-F12",
		Enabled:          true,
		Tags:             `["ctfdb","ctf"]`,
	}

	synced, deleted, err := service.SyncResourceRepository(logger, parserService, nil, nil, repo, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
	var templates []models.Templates
	if err := db.Order("id ASC").Find(&templates).Error; err != nil {
		t.Fatal(err)
	}
	if len(templates) != 1 {
		t.Fatalf("templates = %d, want 1", len(templates))
	}
	got := templates[0]
	if got.Name != "f12" {
		t.Fatalf("name = %q, want %q", got.Name, "f12")
	}
	if got.Description != "F12" {
		t.Fatalf("description = %q, want %q", got.Description, "F12")
	}
	if got.RepositoryUrl != "https://github.com/hexbay/ctfdb.git" {
		t.Fatalf("repository url = %q", got.RepositoryUrl)
	}
	if got.Path != "web/web2-F12" {
		t.Fatalf("path = %q, want %q", got.Path, "web/web2-F12")
	}
	if got.TemplateType != "ctf" {
		t.Fatalf("template type = %q, want %q", got.TemplateType, "ctf")
	}
	if got.Status != "pending" {
		t.Fatalf("status = %q, want %q", got.Status, "pending")
	}
	if !got.LastSyncAt.Equal(now) {
		t.Fatalf("last_sync_at = %v, want %v", got.LastSyncAt, now)
	}
	if len(got.Services) != 1 {
		t.Fatalf("expected 1 services, got %d", len(got.Services))
	}
	gotService := got.Services[0]
	if gotService.Name != "f12" {
		t.Fatalf("service name = %q, want %q", gotService.Name, "f12")

	}
	if gotService.Image != "tongchengbin/f12" {
		t.Fatalf("service image = %q, want %q", gotService.Image, "tongchengbin/f12")
	}

	if len(gotService.Dockerfile) < 10 {
		t.Fatalf("service Dockerfile Not Parsed")
	}
	workspaceDir := filepath.Join(tempRoot, "static", "template-files", stringifyID(got.Id))
	if _, err := os.Stat(filepath.Join(workspaceDir, "metadata.yml")); err != nil {
		t.Fatalf("metadata.yml not synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "Dockerfile")); err != nil {
		t.Fatalf("Dockerfile not synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "files", "html")); err != nil {
		t.Fatalf("files/html not synced: %v", err)
	}
}

func TestRunGitReportsCommandStartFailure(t *testing.T) {
	t.Setenv("PATH", "")

	err := runGit("", "version")
	if err == nil {
		t.Fatal("expected git command failure")
	}
	if !strings.Contains(err.Error(), "requires Git") ||
		!strings.Contains(err.Error(), "install Git") ||
		!strings.Contains(err.Error(), "PATH") {
		t.Fatalf("error = %q, want actionable Git installation message", err)
	}
	if strings.HasSuffix(err.Error(), ":") {
		t.Fatalf("error = %q, want complete error message", err)
	}
}

func TestSyncResourceRepositoryWithVulhubTomcat8(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempRoot := t.TempDir()
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	dbPath := filepath.Join(tempRoot, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&models.Templates{}); err != nil {
		t.Fatal(err)
	}
	service := NewTemplateRuntimeService(db)
	parserService := NewTemplateParserService()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	now := time.Date(2026, 6, 6, 21, 50, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	repo := models.ResourceRepo{
		ID:               1002,
		Name:             "vulhub",
		RepoType:         "vul",
		Provider:         "git",
		RepositoryURL:    "https://github.com/vulhub/vulhub.git",
		RepositoryBranch: "master",
		TemplatesPath:    "tomcat/tomcat8",
		Enabled:          true,
		Tags:             `["vulhub","vul"]`,
	}
	synced, deleted, err := service.SyncResourceRepository(logger, parserService, nil, nil, repo, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if synced != 1 {
		t.Fatalf("synced = %d, want 1", synced)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
	var templates []models.Templates
	if err := db.Order("id ASC").Find(&templates).Error; err != nil {
		t.Fatal(err)
	}
	if len(templates) != 1 {
		t.Fatalf("templates = %d, want 1", len(templates))
	}
	got := templates[0]
	if got.Name != "tomcat8" {
		t.Fatalf("name = %q, want %q", got.Name, "tomcat8")
	}
	if got.Path != "tomcat/tomcat8" {
		t.Fatalf("path = %q, want %q", got.Path, "tomcat/tomcat8")
	}
	if got.TemplateType != "vul" {
		t.Fatalf("template type = %q, want %q", got.TemplateType, "vul")
	}
	if got.RepositoryUrl != "https://github.com/vulhub/vulhub.git" {
		t.Fatalf("repository url = %q", got.RepositoryUrl)
	}
	if got.Status != "pending" {
		t.Fatalf("status = %q, want %q", got.Status, "pending")
	}
	if got.ComposeFile == "" {
		t.Fatal("compose_file should not be empty")
	}
	if got.Description == "" {
		t.Fatal("description should not be empty")
	}
	if len(got.Services) == 0 {
		t.Fatal("expected parsed services from docker-compose.yml")
	}
	gotService := got.Services[0]
	if len(gotService.Ports) != 1 || gotService.Ports[0] != "8080" {
		t.Fatalf("service ports = %#v, want [\"8080\"]", gotService.Ports)
	}
	if len(got.Ports) != 1 || got.Ports[0] != "8080" {
		t.Fatalf("template ports = %#v", got.Ports)
	}
	if len(got.Tags) == 0 {
		t.Fatal("expected vulhub tags")
	}
	workspaceDir := filepath.Join(tempRoot, "static", "template-files", stringifyID(got.Id))
	if _, err := os.Stat(filepath.Join(workspaceDir, "docker-compose.yml")); err != nil {
		t.Fatalf("docker-compose.yml not synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "README.md")); err != nil {
		t.Fatalf("README.md not synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "tomcat-users.xml")); err != nil {
		t.Fatalf("tomcat-users.xml not synced: %v", err)
	}
}

func getLocalCli() (*client.Client, error) {
	var uri string
	if runtime.GOOS == "windows" {
		uri = "npipe:////./pipe/docker_engine"
	} else {
		uri = "unix:///var/run/docker.sock"
	}
	cli, err := DockerClientForEndpoint(uri)
	if err != nil {
		return nil, err
	}
	// 验证 Docker daemon 是否可达
	if _, pingErr := cli.Ping(context.Background()); pingErr != nil {
		return nil, pingErr
	}
	return cli, nil
}

func TestBuildDockerfile(t *testing.T) {
	cli, err := getLocalCli()
	if err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}
	tempRoot := t.TempDir()
	templateSourceDir := filepath.Join("testdata", "templates", "f12")
	templateDir := filepath.Join(tempRoot, "template")
	if err := syncTemplateWorkspace(templateSourceDir, templateDir); err != nil {
		t.Fatal(err)
	}
	tar, err := CreateDockerTar(templateDir)
	if err != nil {
		t.Fatal(err)
	}
	err = BuildDockerfile(context.Background(), cli, "f12:latest", tar, "Dockerfile")
	if err != nil {
		t.Fatalf("Error:%s", err.Error())
	}
}

func TestTemplateBuild(t *testing.T) {
	cli, err := getLocalCli()
	if err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}
	defer cli.Close()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempRoot := t.TempDir()
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWD)

	dbPath := filepath.Join(tempRoot, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&models.VirtualizationHost{}, &models.Templates{}); err != nil {
		t.Fatal(err)
	}

	var endpoint string
	if runtime.GOOS == "windows" {
		endpoint = "npipe:////./pipe/docker_engine"
	} else {
		endpoint = "unix:///var/run/docker.sock"
	}

	hostID := uint(1)
	host := models.VirtualizationHost{
		ID:       hostID,
		Name:     "local-docker",
		HostType: "docker",
		Endpoint: endpoint,
		Status:   "online",
		IsActive: true,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}

	templateID := uint(10001)
	templateDir := filepath.Join(tempRoot, "static", "template-files", stringifyID(templateID))
	templateSourceDir := filepath.Join(originalWD, "testdata", "templates", "f12")
	if err := syncTemplateWorkspace(templateSourceDir, templateDir); err != nil {
		t.Fatal(err)
	}
	imageName := "cloudctf/test-template-build:latest"
	template := models.Templates{
		Id:           templateID,
		Name:         "test-template-build",
		HostId:       &hostID,
		Host:         &host,
		Status:       "pending",
		TemplateType: "ctf",
		Services: []models.TemplateService{
			{
				Name:       "app",
				Image:      imageName,
				Dockerfile: "Dockerfile",
				Ports:      []string{"80"},
			},
		},
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}

	err = TemplateBuild(&template, nil)
	if err != nil {
		t.Fatalf(">>")
	}
	if len(template.Services[0].Ports) != 1 || template.Services[0].Ports[0] != "80" {
		t.Fatalf("为获取到端口")
	}
}
