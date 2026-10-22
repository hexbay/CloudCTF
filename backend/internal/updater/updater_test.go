package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{candidate: "v1.2.4", current: "v1.2.3", want: true},
		{candidate: "v1.2.3", current: "v1.2.3", want: false},
		{candidate: "v1.2.2", current: "v1.2.3", want: false},
		{candidate: "v1.2.3", current: "dev", want: true},
		{candidate: "v1.2.3", current: "v1.2.3-rc.1", want: true},
		{candidate: "v1.2.3-rc.2", current: "v1.2.3-rc.10", want: false},
	}
	for _, test := range tests {
		if got := isNewerVersion(test.candidate, test.current); got != test.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestDevelopmentBuildDoesNotOfferOnlineUpdate(t *testing.T) {
	service := New(Config{
		Repository: "hexbay/CloudCTF",
		Enabled:    true,
		Version:    "dev",
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	result, err := service.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Supported || result.UpdateAvailable {
		t.Fatalf("development build unexpectedly supports updates: %+v", result)
	}
}

func TestCheckAndApply(t *testing.T) {
	newBinary := fakeELF("amd64")
	sum := sha256.Sum256(newBinary)
	checksum := hex.EncodeToString(sum[:]) + "  cloudctf-linux-amd64\n"
	server := releaseServer(t, newBinary, checksum)
	defer server.Close()

	dir := t.TempDir()
	executable := filepath.Join(dir, "cloudctf")
	oldBinary := []byte("old binary")
	if err := os.WriteFile(executable, oldBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	service := New(Config{
		Repository: "hexbay/CloudCTF",
		APIBaseURL: server.URL,
		Enabled:    true,
		Version:    "v0.0.5",
		Executable: executable,
		GOOS:       "linux",
		GOARCH:     "amd64",
		HTTPClient: server.Client(),
	})

	check, err := service.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !check.Supported || !check.UpdateAvailable || check.LatestVersion != "v0.0.6" {
		t.Fatalf("unexpected check result: %+v", check)
	}
	result, err := service.Apply(context.Background(), "v0.0.6")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Version != "v0.0.6" || !result.Restarting {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	installed, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(newBinary) {
		t.Fatalf("installed binary = %q", installed)
	}
	backup, err := os.ReadFile(executable + ".previous")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != string(oldBinary) {
		t.Fatalf("backup binary = %q", backup)
	}
}

func TestApplyRejectsChecksumMismatch(t *testing.T) {
	server := releaseServer(t, fakeELF("amd64"), fmt.Sprintf("%064d  cloudctf-linux-amd64\n", 0))
	defer server.Close()

	dir := t.TempDir()
	executable := filepath.Join(dir, "cloudctf")
	if err := os.WriteFile(executable, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := New(Config{
		Repository: "hexbay/CloudCTF",
		APIBaseURL: server.URL,
		Enabled:    true,
		Version:    "v0.0.5",
		Executable: executable,
		GOOS:       "linux",
		GOARCH:     "amd64",
		HTTPClient: server.Client(),
	})
	if _, err := service.Apply(context.Background(), "v0.0.6"); err == nil {
		t.Fatal("Apply() succeeded with an invalid checksum")
	}
	content, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old binary" {
		t.Fatalf("current executable changed after failed update: %q", content)
	}
}

func TestApplyRejectsWrongArchitecture(t *testing.T) {
	binary := fakeELF("arm64")
	sum := sha256.Sum256(binary)
	checksum := hex.EncodeToString(sum[:]) + "  cloudctf-linux-amd64\n"
	server := releaseServer(t, binary, checksum)
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "cloudctf")
	if err := os.WriteFile(executable, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := New(Config{
		Repository: "hexbay/CloudCTF", APIBaseURL: server.URL, Enabled: true,
		Version: "v0.0.5", Executable: executable, GOOS: "linux", GOARCH: "amd64", HTTPClient: server.Client(),
	})
	if _, err := service.Apply(context.Background(), "v0.0.6"); err == nil {
		t.Fatal("Apply() succeeded with an update for the wrong architecture")
	}
}

func TestPersistentApplySurvivesContainerRecreation(t *testing.T) {
	newBinary := fakeELF("amd64")
	sum := sha256.Sum256(newBinary)
	server := releaseServer(t, newBinary, hex.EncodeToString(sum[:])+"  cloudctf-linux-amd64\n")
	defer server.Close()

	dir := t.TempDir()
	imageExecutable := filepath.Join(dir, "image-cloudctf")
	oldBinary := []byte("image version binary")
	if err := os.WriteFile(imageExecutable, oldBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	updateDir := filepath.Join(dir, "data", "updates")
	service := New(Config{
		Repository: "hexbay/CloudCTF", APIBaseURL: server.URL, Enabled: true,
		Version: "v0.0.5", Executable: imageExecutable, Persistent: true, PersistentDir: updateDir,
		GOOS: "linux", GOARCH: "amd64", HTTPClient: server.Client(),
	})
	result, err := service.Apply(context.Background(), "v0.0.6")
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.InstallMode != "persistent" {
		t.Fatalf("install mode = %q, want persistent", result.InstallMode)
	}
	imageContents, err := os.ReadFile(imageExecutable)
	if err != nil || string(imageContents) != string(oldBinary) {
		t.Fatal("persistent update modified the image executable")
	}

	selected, ok, err := SelectPersistentExecutable("v0.0.5", updateDir, imageExecutable)
	if err != nil || !ok {
		t.Fatalf("SelectPersistentExecutable() = %q, %v, %v", selected, ok, err)
	}
	if err := MarkPersistentHealthy("v0.0.6", updateDir, selected); err != nil {
		t.Fatalf("MarkPersistentHealthy() error = %v", err)
	}
	selectedAgain, ok, err := SelectPersistentExecutable("v0.0.5", updateDir, imageExecutable)
	if err != nil || !ok || selectedAgain != selected {
		t.Fatalf("persistent update was not reused after recreation: %q, %v, %v", selectedAgain, ok, err)
	}
	if _, ok, err := SelectPersistentExecutable("v0.0.7", updateDir, imageExecutable); err != nil || ok {
		t.Fatalf("older persistent update should not override newer image: ok=%v err=%v", ok, err)
	}
}

func TestPersistentUpdateFallsBackAfterFailedFirstBoot(t *testing.T) {
	newBinary := fakeELF("amd64")
	sum := sha256.Sum256(newBinary)
	server := releaseServer(t, newBinary, hex.EncodeToString(sum[:])+"  cloudctf-linux-amd64\n")
	defer server.Close()

	dir := t.TempDir()
	imageExecutable := filepath.Join(dir, "image-cloudctf")
	if err := os.WriteFile(imageExecutable, []byte("previous binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	updateDir := filepath.Join(dir, "updates")
	service := New(Config{
		Repository: "hexbay/CloudCTF", APIBaseURL: server.URL, Enabled: true,
		Version: "v0.0.5", Executable: imageExecutable, Persistent: true, PersistentDir: updateDir,
		GOOS: "linux", GOARCH: "amd64", HTTPClient: server.Client(),
	})
	if _, err := service.Apply(context.Background(), "v0.0.6"); err != nil {
		t.Fatal(err)
	}
	first, ok, err := SelectPersistentExecutable("v0.0.5", updateDir, imageExecutable)
	if err != nil || !ok {
		t.Fatalf("first boot selection failed: %q, %v, %v", first, ok, err)
	}
	fallback, ok, err := SelectPersistentExecutable("v0.0.4", updateDir, imageExecutable)
	if err != nil || !ok {
		t.Fatalf("fallback selection failed: %q, %v, %v", fallback, ok, err)
	}
	if fallback == first || !strings.Contains(fallback, filepath.Join("versions", "0.0.5")) {
		t.Fatalf("selected fallback %q, want preserved v0.0.5", fallback)
	}
}

func fakeELF(arch string) []byte {
	binary := make([]byte, 64)
	copy(binary[:4], []byte("\x7fELF"))
	binary[4] = 2
	binary[5] = 1
	machine := uint16(62)
	if arch == "arm64" {
		machine = 183
	}
	binary[18] = byte(machine)
	binary[19] = byte(machine >> 8)
	return binary
}

func releaseServer(t *testing.T, binary []byte, checksum string) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/hexbay/CloudCTF/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Release{
				TagName: "v0.0.6",
				Name:    "v0.0.6",
				HTMLURL: server.URL + "/release",
				Assets: []Asset{
					{Name: "cloudctf-linux-amd64", Size: int64(len(binary)), BrowserDownloadURL: server.URL + "/binary"},
					{Name: "cloudctf-linux-amd64.sha256", Size: int64(len(checksum)), BrowserDownloadURL: server.URL + "/checksum"},
				},
			})
		case "/binary":
			_, _ = w.Write(binary)
		case "/checksum":
			_, _ = w.Write([]byte(checksum))
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}
