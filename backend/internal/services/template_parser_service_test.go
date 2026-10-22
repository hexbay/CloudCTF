package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRepoMetadataAcceptsCVEFormatVariants(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "sequence",
			content: `cve:
  - CVE-2024-0001
  - CVE-2024-0002`,
			want: []string{"CVE-2024-0001", "CVE-2024-0002"},
		},
		{
			name:    "scalar",
			content: `cve: CVE-2024-0001, CVE-2024-0002`,
			want:    []string{"CVE-2024-0001", "CVE-2024-0002"},
		},
		{
			name:    "missing",
			content: `name: template without CVE`,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metadata RepoMetaData
			if err := yaml.Unmarshal([]byte(tt.content), &metadata); err != nil {
				t.Fatalf("unmarshal metadata: %v", err)
			}
			if got := []string(metadata.Cve); strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("cve = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCtfdbDiscoverAcceptsCVESequence(t *testing.T) {
	repoDir := t.TempDir()
	templateDir := filepath.Join(repoDir, "apache", "CVE-2021-41773")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeParserTestFile(t, templateDir, "metadata.yml", `name: CVE-2021-41773
image: cve202141773
cve:
  - CVE-2021-41773
`)
	writeParserTestFile(t, templateDir, "Dockerfile", "FROM scratch\n")

	parser := NewTemplateParserService().GetRepoTemplateParser([]string{"ctfdb"})
	templates, err := parser.Discover(repoDir, repoDir, "VULDB")
	if err != nil {
		t.Fatalf("discover templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("templates = %d, want 1", len(templates))
	}
	if !containsString(templates[0].Tags, "CVE-2021-41773") {
		t.Fatalf("tags = %#v, want CVE tag", templates[0].Tags)
	}
}

func TestCtfdbDiscoverFallsBackToDirectoryCVE(t *testing.T) {
	repoDir := t.TempDir()
	templateDir := filepath.Join(repoDir, "tomcat", "CVE-2024-50379")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeParserTestFile(t, templateDir, "metadata.yml", `name: Tomcat conditional race
image: cve_2024_50379
`)

	parser := NewTemplateParserService().GetRepoTemplateParser([]string{"ctfdb"})
	templates, err := parser.Discover(repoDir, repoDir, "VULDB")
	if err != nil {
		t.Fatalf("discover templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("templates = %d, want 1", len(templates))
	}
	if !containsString(templates[0].Tags, "CVE-2024-50379") {
		t.Fatalf("tags = %#v, want directory CVE fallback", templates[0].Tags)
	}
}

func TestReadRepoMetadataReportsPathAndYAMLError(t *testing.T) {
	templateDir := t.TempDir()
	writeParserTestFile(t, templateDir, "metadata.yml", `name: invalid
cve:
  id: CVE-2024-0001
`)

	_, err := readRepoMetadata(templateDir)
	if err == nil {
		t.Fatal("expected metadata parse error")
	}
	if !strings.Contains(err.Error(), "metadata.yml") || !strings.Contains(err.Error(), "cve") {
		t.Fatalf("error = %q, want metadata path and cve field context", err)
	}
}

func writeParserTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
