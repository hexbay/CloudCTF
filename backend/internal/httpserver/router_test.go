package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/models"
)

func testConfig() config.Config {
	cfg := config.Load()
	cfg.DatabaseURI = "sqlite://:memory:"
	baseDir, _ := os.MkdirTemp("", "cloudctf-router-test-*")
	cfg.StaticDir = filepath.Join(baseDir, "static")
	cfg.UploadDir = filepath.Join(baseDir, "static", "uploads")
	cfg.BackupDir = filepath.Join(baseDir, "backups")
	return cfg
}

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := testConfig()
	db, err := bootstrap.OpenDB(cfg.DatabaseURI)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("failed to prepare schema: %v", err)
	}
	return NewRouter(Dependencies{
		DB:     db,
		Config: cfg,
		Logger: slog.Default(),
	})
}

func jsonID(body string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return ""
	}
	return stringifyID(payload["id"])
}

func testAccessToken(t *testing.T, userID uint) string {
	t.Helper()
	srv := NewServer(Dependencies{Config: testConfig()})
	token, _, err := srv.issueAccessToken(userID, time.Hour)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}
	return token
}

func addBearer(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

func addAdminBearer(t *testing.T, req *http.Request) {
	t.Helper()
	addBearer(req, testAccessToken(t, 1))
}

func TestRouterHealthz(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Fatalf("expected %s=%q, got %q", header, want, got)
		}
	}
}

func TestSecurityHeadersSetHSTSForForwardedHTTPS(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected HSTS header for HTTPS request")
	}
}

func TestRequestProtectionAddsRequestID(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); !requestIDPattern.MatchString(got) {
		t.Fatalf("expected generated request id, got %q", got)
	}
}

func TestRequestProtectionRejectsOversizedBody(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(strings.Repeat("x", int(defaultRequestBodyLimit+1))))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminLogin(t *testing.T) {
	router := testRouter(t)

	body := strings.NewReader("username=admin&password=admin123456")
	req := httptest.NewRequest(http.MethodPost, "/api-admin/auth/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "access_token") {
		t.Fatalf("expected token response, got %s", rec.Body.String())
	}
}

func TestSensitiveEndpointsAreRateLimited(t *testing.T) {
	router := testRouter(t)
	for attempt := 1; attempt <= 11; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/auth/login", strings.NewReader("username=missing&password=wrong"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "192.0.2.10:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if attempt <= 10 && rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was limited too early", attempt)
		}
		if attempt == 11 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected request 11 to return 429, got %d", rec.Code)
		}
	}
}

func TestAdminRoutesRequireAdminAuth(t *testing.T) {
	router := testRouter(t)

	cases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "player token", token: testAccessToken(t, 2), wantStatus: http.StatusForbidden},
		{name: "admin token", token: testAccessToken(t, 1), wantStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api-admin/dashboard/stats", nil)
			if tc.token != "" {
				addBearer(req, tc.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminRoutesRequireConfiguredPermission(t *testing.T) {
	cfg := testConfig()
	db, err := bootstrap.OpenDB(cfg.DatabaseURI)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("failed to prepare schema: %v", err)
	}
	role := models.Role{Name: "auditor", Permissions: `["admin:audit"]`}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	user := models.User{
		Username:       "auditor",
		Email:          "auditor@test.com",
		HashedPassword: "unused",
		RoleID:         &role.ID,
		IsActive:       true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	srv := NewServer(Dependencies{DB: db, Config: cfg, Logger: slog.Default()})
	token, _, err := srv.issueAccessToken(user.ID, time.Hour)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}
	router := srv.Handler()

	allowedReq := httptest.NewRequest(http.MethodGet, "/api-admin/audit/logs", nil)
	addBearer(allowedReq, token)
	allowedRec := httptest.NewRecorder()
	router.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusOK {
		t.Fatalf("expected audit permission to allow request, got %d: %s", allowedRec.Code, allowedRec.Body.String())
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/api-admin/users", nil)
	addBearer(deniedReq, token)
	deniedRec := httptest.NewRecorder()
	router.ServeHTTP(deniedRec, deniedReq)
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("expected missing users permission to return 403, got %d: %s", deniedRec.Code, deniedRec.Body.String())
	}
	if !strings.Contains(deniedRec.Body.String(), "admin:users") {
		t.Fatalf("expected required permission in response, got %s", deniedRec.Body.String())
	}
}

func TestAdminVirtualizationMutationRequiresAdminAuth(t *testing.T) {
	router := testRouter(t)

	body := `{"name":"Blocked Host","host_type":"docker","endpoint":"unix:///var/run/docker.sock"}`
	req := httptest.NewRequest(http.MethodPost, "/api-admin/virtualization/hosts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated virtualization mutation to return 401, got %d: %s", rec.Code, rec.Body.String())
	}

	playerReq := httptest.NewRequest(http.MethodPost, "/api-admin/virtualization/hosts", bytes.NewBufferString(body))
	playerReq.Header.Set("Content-Type", "application/json")
	addBearer(playerReq, testAccessToken(t, 2))
	playerRec := httptest.NewRecorder()
	router.ServeHTTP(playerRec, playerReq)
	if playerRec.Code != http.StatusForbidden {
		t.Fatalf("expected player virtualization mutation to return 403, got %d: %s", playerRec.Code, playerRec.Body.String())
	}
}

func TestRemovedCompatibilityRoutesReturnNotFound(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)

	cases := []struct {
		name      string
		method    string
		path      string
		adminAuth bool
	}{
		{name: "admin singular admins alias", method: http.MethodGet, path: "/api-admin/admin", adminAuth: true},
		{name: "admin theory collection alias", method: http.MethodGet, path: "/api-admin/theory", adminAuth: true},
		{name: "player singular competitions alias", method: http.MethodGet, path: "/api/competition/"},
		{name: "player my team alias", method: http.MethodGet, path: "/api/my-team"},
		{name: "player misspelled exam alias", method: http.MethodGet, path: "/api/competion/exam?id=1"},
		{name: "player scoreboard prefix alias", method: http.MethodGet, path: "/api/scoreboard/practice"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.adminAuth {
				addBearer(req, adminToken)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected removed compatibility route to return 404, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminResourceCreateAndList(t *testing.T) {
	router := testRouter(t)

	createReq := httptest.NewRequest(http.MethodPost, "/api-admin/users", bytes.NewBufferString(`{"username":"alice","email":"alice@test.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	addAdminBearer(t, createReq)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected create 200, got %d: %s", createRec.Code, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api-admin/users?keyword=alice", nil)
	addAdminBearer(t, listReq)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "alice") {
		t.Fatalf("expected list to contain alice, got %s", listRec.Body.String())
	}
}

func TestRouterPublicSettings(t *testing.T) {
	router := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/public/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFrontendAdminResourceRoutes(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)

	createCompetitionReq := httptest.NewRequest(http.MethodPost, "/api-admin/competitions", bytes.NewBufferString(`{"name":"Status Action Competition","type":"ctf"}`))
	createCompetitionReq.Header.Set("Content-Type", "application/json")
	addBearer(createCompetitionReq, adminToken)
	createCompetitionRec := httptest.NewRecorder()
	router.ServeHTTP(createCompetitionRec, createCompetitionReq)
	if createCompetitionRec.Code != http.StatusOK {
		t.Fatalf("create competition: expected 200, got %d: %s", createCompetitionRec.Code, createCompetitionRec.Body.String())
	}
	competitionID := jsonID(createCompetitionRec.Body.String())
	if competitionID == "" {
		t.Fatalf("create competition response has no id: %s", createCompetitionRec.Body.String())
	}

	createDynamicChallenge := httptest.NewRequest(http.MethodPost, "/api-admin/challenges", bytes.NewBufferString(`{"name":"Dynamic","category":"web","points":100,"flag":"flag{dynamic}","is_dynamic":true}`))
	createDynamicChallenge.Header.Set("Content-Type", "application/json")
	addBearer(createDynamicChallenge, adminToken)
	createDynamicChallengeRec := httptest.NewRecorder()
	router.ServeHTTP(createDynamicChallengeRec, createDynamicChallenge)
	if createDynamicChallengeRec.Code != http.StatusOK {
		t.Fatalf("create dynamic challenge: expected 200, got %d: %s", createDynamicChallengeRec.Code, createDynamicChallengeRec.Body.String())
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "users", method: http.MethodGet, path: "/api-admin/users?current=1&pageSize=10"},
		{name: "admins", method: http.MethodGet, path: "/api-admin/admins?current=1&pageSize=10"},
		{name: "roles", method: http.MethodGet, path: "/api-admin/roles?current=1&pageSize=10"},
		{name: "competitions", method: http.MethodGet, path: "/api-admin/competitions?current=1&pageSize=10"},
		{name: "challenges", method: http.MethodGet, path: "/api-admin/challenges?current=1&pageSize=10"},
		{name: "theory", method: http.MethodGet, path: "/api-admin/theory/questions?current=1&pageSize=10"},
		{name: "teams", method: http.MethodGet, path: "/api-admin/teams?current=1&pageSize=10"},
		{name: "vuln labs", method: http.MethodGet, path: "/api-admin/vuln-labs?current=1&pageSize=10"},
		{name: "virtualization hosts", method: http.MethodGet, path: "/api-admin/virtualization/hosts?current=1&pageSize=10"},
		{name: "resource repositories", method: http.MethodGet, path: "/api-admin/resource-repositories?current=1&pageSize=10"},
		{name: "announcements", method: http.MethodGet, path: "/api-admin/announcements?current=1&pageSize=10"},
		{name: "notifications", method: http.MethodGet, path: "/api-admin/notifications?current=1&pageSize=10"},
		{name: "audit", method: http.MethodGet, path: "/api-admin/audit/logs?current=1&pageSize=10"},
		{name: "settings", method: http.MethodGet, path: "/api-admin/settings?current=1&pageSize=10"},
		{name: "enable user action", method: http.MethodPatch, path: "/api-admin/users/1/enable", body: `{}`},
		{name: "reset user password action", method: http.MethodPost, path: "/api-admin/users/1/reset-password", body: `{"password":"newpass123"}`},
		{name: "competition status action", method: http.MethodPatch, path: "/api-admin/competitions/" + competitionID + "/status", body: `{"status":"ongoing"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *strings.Reader
			if tc.body == "" {
				body = strings.NewReader("")
			} else {
				body = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			addBearer(req, adminToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPracticeChallengesReturnsCompetitionBoundRows(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)

	createCompetition := func(name string) string {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/competitions", bytes.NewBufferString(`{"name":"`+name+`","type":"ctf"}`))
		req.Header.Set("Content-Type", "application/json")
		addBearer(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create competition expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		return jsonID(rec.Body.String())
	}

	createChallenge := func(name string) string {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/challenges", bytes.NewBufferString(`{"name":"`+name+`","category":"web","points":100,"flag":"flag{`+name+`}"}`))
		req.Header.Set("Content-Type", "application/json")
		addBearer(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create challenge expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		return jsonID(rec.Body.String())
	}

	addToCompetition := func(competitionID, challengeID string) {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/competitions/"+competitionID+"/challenges", bytes.NewBufferString(`{"challenge_id":`+challengeID+`}`))
		req.Header.Set("Content-Type", "application/json")
		addBearer(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("add challenge to competition expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	competitionID := createCompetition("Practice API Competition")
	challengeID := createChallenge("Competition Bound Challenge")
	addToCompetition(competitionID, challengeID)

	req := httptest.NewRequest(http.MethodGet, "/api/practice/challenges?current=1&pageSize=100&competition_id="+competitionID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("practice challenges expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Data) != 1 {
		t.Fatalf("expected exactly 1 competition-bound challenge, got total=%d body=%s", payload.Total, rec.Body.String())
	}
	if stringifyID(payload.Data[0]["competition_id"]) != competitionID {
		t.Fatalf("expected competition_id %s, got body=%s", competitionID, rec.Body.String())
	}
}

func TestAdminCompetitionChallengesFiltersByCompetition(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)

	createCompetition := func(name string) string {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/competitions", bytes.NewBufferString(`{"name":"`+name+`","type":"ctf"}`))
		req.Header.Set("Content-Type", "application/json")
		addBearer(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create competition expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		id := jsonID(rec.Body.String())
		if id == "" {
			t.Fatalf("create competition response has no id: %s", rec.Body.String())
		}
		return id
	}

	createChallenge := func(name, competitionID string) {
		req := httptest.NewRequest(http.MethodPost, "/api-admin/challenges", bytes.NewBufferString(`{"name":"`+name+`","category":"web","points":100,"flag":"flag{`+name+`}","competition_id":`+competitionID+`}`))
		req.Header.Set("Content-Type", "application/json")
		addBearer(req, adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create challenge expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	competitionA := createCompetition("Filter Competition A")
	competitionB := createCompetition("Filter Competition B")

	createChallenge("challenge-a1", competitionA)
	createChallenge("challenge-b1", competitionB)

	req := httptest.NewRequest(http.MethodGet, "/api-admin/competitions/"+competitionA+"/challenges?current=1&pageSize=20", nil)
	addBearer(req, adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list competition challenges expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Data) != 1 {
		t.Fatalf("expected exactly 1 challenge for competition %s, got total=%d body=%s", competitionA, payload.Total, rec.Body.String())
	}
	if stringifyID(payload.Data[0]["competition_id"]) != competitionA {
		t.Fatalf("expected competition_id %s, got body=%s", competitionA, rec.Body.String())
	}
}

func TestFrontendAdminDeleteButtons(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)

	cases := []struct {
		name     string
		endpoint string
		body     string
	}{
		{name: "users", endpoint: "/api-admin/users", body: `{"username":"delete_user","email":"delete_user@test.com","password":"pass123456"}`},
		{name: "roles", endpoint: "/api-admin/roles", body: `{"name":"delete_role","description":"temporary role"}`},
		{name: "competitions", endpoint: "/api-admin/competitions", body: `{"name":"Delete Competition","type":"ctf"}`},
		{name: "challenges", endpoint: "/api-admin/challenges", body: `{"name":"Delete Challenge","category":"web","points":100,"flag":"flag{delete}"}`},
		{name: "theory", endpoint: "/api-admin/theory/questions", body: `{"question":"Delete question?","points":10,"question_type":"text"}`},
		{name: "teams", endpoint: "/api-admin/teams", body: `{"name":"Delete Team","owner_id":1}`},
		{name: "vuln labs", endpoint: "/api-admin/vuln-labs", body: `{"name":"Delete Vuln","cve_id":"CVE-2026-DELETE","template_id":1}`},
		{name: "virtualization hosts", endpoint: "/api-admin/virtualization/hosts", body: `{"name":"Delete Host","host_type":"docker","endpoint":"unix:///var/run/docker.sock"}`},
		{name: "resource repositories", endpoint: "/api-admin/resource-repositories", body: `{"name":"Delete Repo","repo_type":"vuln","provider":"git","repository_url":"https://example.com/delete.git"}`},
		{name: "announcements", endpoint: "/api-admin/announcements", body: `{"title":"Delete Announcement","content":"temporary","author_id":1}`},
		{name: "notifications", endpoint: "/api-admin/notifications", body: `{"title":"Delete Notification","content":"temporary","receiver_id":1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			createReq := httptest.NewRequest(http.MethodPost, tc.endpoint, bytes.NewBufferString(tc.body))
			createReq.Header.Set("Content-Type", "application/json")
			addBearer(createReq, adminToken)
			createRec := httptest.NewRecorder()
			router.ServeHTTP(createRec, createReq)
			if createRec.Code != http.StatusOK {
				t.Fatalf("create expected 200, got %d: %s", createRec.Code, createRec.Body.String())
			}
			id := jsonID(createRec.Body.String())
			if id == "" {
				t.Fatalf("create response has no id: %s", createRec.Body.String())
			}

			deleteReq := httptest.NewRequest(http.MethodDelete, tc.endpoint+"/"+id, nil)
			addBearer(deleteReq, adminToken)
			deleteRec := httptest.NewRecorder()
			router.ServeHTTP(deleteRec, deleteReq)
			if deleteRec.Code != http.StatusOK {
				t.Fatalf("delete expected 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
			}
		})
	}
}

func TestAdminVirtualizationTemplateBatchDelete(t *testing.T) {
	router := testRouter(t)
	adminToken := testAccessToken(t, 1)
	ids := []string{}
	for _, name := range []string{"Batch Template A", "Batch Template B"} {
		createReq := httptest.NewRequest(http.MethodPost, "/api-admin/virtualization/templates", bytes.NewBufferString(`{"name":"`+name+`","host_id":1,"template_type":"vul","image":"nginx:latest"}`))
		createReq.Header.Set("Content-Type", "application/json")
		addBearer(createReq, adminToken)
		createRec := httptest.NewRecorder()
		router.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusOK {
			t.Fatalf("create template expected 200, got %d: %s", createRec.Code, createRec.Body.String())
		}
		id := jsonID(createRec.Body.String())
		if id == "" {
			t.Fatalf("create template response has no id: %s", createRec.Body.String())
		}
		ids = append(ids, id)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/api-admin/virtualization/templates/batch/delete", bytes.NewBufferString(`{"ids":["`+ids[0]+`","`+ids[1]+`"]}`))
	deleteReq.Header.Set("Content-Type", "application/json")
	addBearer(deleteReq, adminToken)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("batch delete expected 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), `"deleted":2`) {
		t.Fatalf("batch delete expected deleted=2, got: %s", deleteRec.Body.String())
	}
}

func TestFrontendPlayerRoutes(t *testing.T) {
	router := testRouter(t)

	cases := []struct {
		name string
		path string
	}{
		{name: "competitions", path: "/api/competitions?current=1&pageSize=10"},
		{name: "practice challenges", path: "/api/practice/challenges?current=1&pageSize=10"},
		{name: "vuln labs", path: "/api/vuln-labs?current=1&pageSize=10"},
		{name: "practice leaderboard", path: "/api/leaderboard/practice"},
		{name: "challenge leaderboard", path: "/api/leaderboard/challenges"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRemovedUnusedFrontendRoutesReturnNotFound(t *testing.T) {
	router := testRouter(t)
	token := testAccessToken(t, 1)

	cases := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/system-announcements"},
		{method: http.MethodGet, path: "/api/user/profile"},
		{method: http.MethodGet, path: "/api/exam/1/scoreboard"},
		{method: http.MethodPost, path: "/api/public/referral/track"},
		{method: http.MethodGet, path: "/api-admin/virtualization"},
		{method: http.MethodGet, path: "/api-admin/competitions/"},
		{method: http.MethodGet, path: "/api-admin/notifications/"},
		{method: http.MethodPost, path: "/api-admin/virtualization/templates/batch-delete"},
		{method: http.MethodGet, path: "/api-admin/virtualization/templates_sync_logs"},
		{method: http.MethodGet, path: "/api-admin/system/backups"},
		{method: http.MethodPost, path: "/api-admin/challenges/1/start"},
		{method: http.MethodGet, path: "/api-admin/flags"},
		{method: http.MethodGet, path: "/api-admin/scoreboard"},
		{method: http.MethodGet, path: "/api-admin/permissions"},
		{method: http.MethodPost, path: "/api-admin/notifications/send"},
		{method: http.MethodGet, path: "/api-admin/notifications/logs"},
		{method: http.MethodGet, path: "/api-admin/challenge-virtualization"},
		{method: http.MethodGet, path: "/api-admin/virtualization/templates_build_logs"},
		{method: http.MethodGet, path: "/api-admin/virtualization/containers/1/logs"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if strings.HasPrefix(tc.path, "/api-admin/") {
				addBearer(req, token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminBackupLifecycle(t *testing.T) {
	router := testRouter(t)
	token := testAccessToken(t, 1)

	createUserReq := httptest.NewRequest(http.MethodPost, "/api-admin/users", bytes.NewBufferString(`{"username":"backup-user","email":"backup-user@test.com","password":"pass123456"}`))
	createUserReq.Header.Set("Content-Type", "application/json")
	addBearer(createUserReq, token)
	createUserRec := httptest.NewRecorder()
	router.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusOK {
		t.Fatalf("seed user expected 200, got %d: %s", createUserRec.Code, createUserRec.Body.String())
	}

	backupReq := httptest.NewRequest(http.MethodPost, "/api-admin/settings/backup", nil)
	backupReq.Header.Set("Authorization", "Bearer "+token)
	backupRec := httptest.NewRecorder()
	router.ServeHTTP(backupRec, backupReq)
	if backupRec.Code != http.StatusOK {
		t.Fatalf("backup create expected 200, got %d: %s", backupRec.Code, backupRec.Body.String())
	}

	var backupResp struct {
		Backup backupFileInfo `json:"backup"`
	}
	if err := json.Unmarshal(backupRec.Body.Bytes(), &backupResp); err != nil {
		t.Fatalf("decode backup response: %v", err)
	}
	if backupResp.Backup.Name == "" {
		t.Fatalf("expected backup name, got %s", backupRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api-admin/settings/backups", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), backupResp.Backup.Name) {
		t.Fatalf("backup list expected created file, got %d: %s", listRec.Code, listRec.Body.String())
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api-admin/settings/restore/"+backupResp.Backup.Name, nil)
	restoreReq.Header.Set("Authorization", "Bearer "+token)
	restoreRec := httptest.NewRecorder()
	router.ServeHTTP(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("restore expected 200, got %d: %s", restoreRec.Code, restoreRec.Body.String())
	}
}
