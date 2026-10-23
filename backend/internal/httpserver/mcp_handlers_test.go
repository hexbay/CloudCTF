package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"
)

func testMCPServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	cfg := testConfig()
	db, err := bootstrap.OpenDB(cfg.DatabaseURI)
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(Dependencies{DB: db, Config: cfg, Logger: slog.Default()})
	return srv, srv.Handler()
}

func TestMCPRequiresAPIKey(t *testing.T) {
	_, handler := testMCPServer(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMCPToolsRespectRole(t *testing.T) {
	srv, handler := testMCPServer(t)
	var player models.User
	if err := srv.db.Preload("Role").Joins("Role").Where("Role.name = ?", rolePlayer).First(&player).Error; err != nil {
		t.Fatal(err)
	}
	playerKey := createTestAPIKey(t, srv, player.ID)
	playerBody := callMCP(t, handler, playerKey, "tools/list", map[string]any{})
	if bytes.Contains(playerBody, []byte("compile_vulnerability_environment")) {
		t.Fatalf("player must not receive admin compile tool: %s", playerBody)
	}

	var admin models.User
	if err := srv.db.Preload("Role").Where("is_superuser = ?", true).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	adminKey := createTestAPIKey(t, srv, admin.ID)
	adminBody := callMCP(t, handler, adminKey, "tools/list", map[string]any{})
	if !bytes.Contains(adminBody, []byte("compile_vulnerability_environment")) {
		t.Fatalf("admin should receive compile tool: %s", adminBody)
	}
}

func createTestAPIKey(t *testing.T, srv *Server, userID uint) string {
	t.Helper()
	plain := "ctf_mcp_test_key_" + string(rune('a'+userID))
	record := models.APIKey{UserID: userID, Key: plain}
	if err := srv.db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	return plain
}

func callMCP(t *testing.T, handler http.Handler, key, method string, params any) []byte {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}
