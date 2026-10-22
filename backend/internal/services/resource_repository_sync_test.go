package services

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResourceRepositorySyncLockIsPerRepository(t *testing.T) {
	service := &TemplateRuntimeService{}

	releaseFirst, err := service.beginResourceRepositorySync(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.beginResourceRepositorySync(1); !errors.Is(err, ErrResourceRepositorySyncInProgress) {
		releaseFirst()
		t.Fatalf("second same-repository lock error = %v, want ErrResourceRepositorySyncInProgress", err)
	}

	releaseOther, err := service.beginResourceRepositorySync(2)
	if err != nil {
		releaseFirst()
		t.Fatalf("different repository should not block: %v", err)
	}
	releaseOther()
	releaseFirst()

	releaseAgain, err := service.beginResourceRepositorySync(1)
	if err != nil {
		t.Fatalf("repository lock was not released: %v", err)
	}
	releaseAgain()
}

func TestSyncGitRepositoryReplacesIncompleteRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "master"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "CloudCTF Test"},
	} {
		if err := runGit(sourceDir, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("repository snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runGit(sourceDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGit(sourceDir, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(root, "repositories", "1")
	if err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, ".git", "HEAD"), []byte("ref: refs/heads/master\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(targetDir, ".git", "shallow.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := syncGitRepository(nil, targetDir, sourceDir, "master"); err != nil {
		t.Fatal(err)
	}
	if !gitRepositoryHasValidHead(targetDir) {
		t.Fatal("replacement repository does not have a valid HEAD")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("incomplete repository lock should be removed, stat error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "repository snapshot\n" {
		t.Fatalf("README.md = %q, want cloned repository content", content)
	}
}
