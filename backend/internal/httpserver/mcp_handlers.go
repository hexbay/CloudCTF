package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/services"

	"gorm.io/gorm"
)

const mcpProtocolVersion = "2025-06-18"

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (srv *Server) playerAPIKeyStatusHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	var key models.APIKey
	err := srv.db.Where("user_id = ?", user.ID).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(w, http.StatusOK, map[string]any{"exists": false})
		return
	}
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to load API key"})
		return
	}
	response.JSON(w, http.StatusOK, apiKeyMetadata(key))
}

func (srv *Server) playerAPIKeyGenerateHandler(w http.ResponseWriter, r *http.Request) {
	srv.issuePlayerAPIKey(w, r, false)
}

func (srv *Server) playerAPIKeyResetHandler(w http.ResponseWriter, r *http.Request) {
	srv.issuePlayerAPIKey(w, r, true)
}

func (srv *Server) issuePlayerAPIKey(w http.ResponseWriter, r *http.Request, reset bool) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	var existing models.APIKey
	err := srv.db.Where("user_id = ?", user.ID).First(&existing).Error
	if err == nil && !reset {
		response.JSON(w, http.StatusConflict, map[string]any{"detail": "API key already exists; reset it to create a new key"})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to load API key"})
		return
	}
	random, err := randomToken(32)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to generate API key"})
		return
	}
	plain := "ctf_mcp_" + random
	record := models.APIKey{UserID: user.ID, Key: plain}
	if existing.ID != 0 {
		record.ID = existing.ID
		record.CreatedAt = existing.CreatedAt
		if err := srv.db.Save(&record).Error; err != nil {
			response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to reset API key"})
			return
		}
	} else if err := srv.db.Create(&record).Error; err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to create API key"})
		return
	}
	result := apiKeyMetadata(record)
	result["api_key"] = plain
	response.JSON(w, http.StatusOK, result)
}

func apiKeyMetadata(key models.APIKey) map[string]any {
	return map[string]any{
		"exists":       true,
		"api_key":      key.Key,
		"created_at":   key.CreatedAt,
		"updated_at":   key.UpdatedAt,
		"last_used_at": key.LastUsedAt,
	}
}

func (srv *Server) playerMCPConfigHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" {
		host = forwarded
	}
	var key models.APIKey
	if err := srv.db.Where("user_id = ?", user.ID).First(&key).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to load API key"})
		return
	}
	apiKey := key.Key
	if apiKey == "" {
		apiKey = "<YOUR_API_KEY>"
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"mcpServers": map[string]any{
			"cloudctf": map[string]any{
				"type": "http",
				"url":  scheme + "://" + host + "/mcp",
				"headers": map[string]string{
					"Authorization": "Bearer " + apiKey,
				},
			},
		},
	})
}

func (srv *Server) userFromAPIKey(r *http.Request) (userModel, models.APIKey, bool) {
	plain := bearerToken(r.Header.Get("Authorization"))
	if !strings.HasPrefix(plain, "ctf_mcp_") {
		return userModel{}, models.APIKey{}, false
	}
	var key models.APIKey
	if err := srv.db.Where("api_key = ?", plain).First(&key).Error; err != nil {
		return userModel{}, models.APIKey{}, false
	}
	user, ok := srv.findUserByID(key.UserID)
	if !ok || !user.IsActive {
		return userModel{}, models.APIKey{}, false
	}
	now := time.Now()
	_ = srv.db.Model(&key).Update("last_used_at", &now).Error
	key.LastUsedAt = &now
	return user, key, true
}

func (srv *Server) mcpHandler(w http.ResponseWriter, r *http.Request) {
	user, _, ok := srv.userFromAPIKey(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid or missing API key"})
		return
	}
	var req mcpRequest
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil || req.JSONRPC != "2.0" || req.Method == "" {
		mcpWriteError(w, req.ID, -32600, "invalid request")
		return
	}
	if len(req.ID) == 0 || string(req.ID) == "null" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch req.Method {
	case "initialize":
		mcpWriteResult(w, req.ID, map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "CloudCTF", "version": srv.currentVersion()},
		})
	case "ping":
		mcpWriteResult(w, req.ID, map[string]any{})
	case "tools/list":
		mcpWriteResult(w, req.ID, map[string]any{"tools": mcpToolsForUser(user)})
	case "tools/call":
		var params mcpToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			mcpWriteError(w, req.ID, -32602, "invalid tool arguments")
			return
		}
		result, err := srv.callMCPTool(user, params)
		if err != nil {
			mcpWriteResult(w, req.ID, mcpToolResult(map[string]any{"error": err.Error()}, true))
			return
		}
		mcpWriteResult(w, req.ID, mcpToolResult(result, false))
	default:
		mcpWriteError(w, req.ID, -32601, "method not found")
	}
}

func mcpToolsForUser(user userModel) []map[string]any {
	tools := []map[string]any{
		mcpTool("search_vulnerabilities", "检索可用的漏洞复现环境", map[string]any{
			"type": "object", "properties": map[string]any{
				"keyword":  map[string]any{"type": "string", "description": "漏洞名称、CVE 或关键词"},
				"severity": map[string]any{"type": "string", "description": "可选严重级别"},
				"limit":    map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 20},
			},
		}),
		mcpTool("start_vulnerability_environment", "为当前用户启动漏洞环境", idSchema("vuln_lab_id", "漏洞环境 ID")),
		mcpTool("destroy_vulnerability_environment", "销毁当前用户的漏洞环境", idSchema("vuln_lab_id", "漏洞环境 ID")),
	}
	if isMCPAdmin(user) {
		tools = append(tools, mcpTool("compile_vulnerability_environment", "编译漏洞环境模板（仅管理员）", idSchema("vuln_lab_id", "漏洞环境 ID")))
	}
	return tools
}

func mcpTool(name, description string, schema map[string]any) map[string]any {
	return map[string]any{"name": name, "description": description, "inputSchema": schema}
}

func idSchema(name, description string) map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{name: map[string]any{"type": "integer", "minimum": 1, "description": description}}, "required": []string{name}}
}

func isMCPAdmin(user userModel) bool {
	return user.IsSuperuser || !strings.EqualFold(strings.TrimSpace(user.Role.Name), rolePlayer)
}

func (srv *Server) callMCPTool(user userModel, call mcpToolCallParams) (any, error) {
	switch call.Name {
	case "search_vulnerabilities":
		limit := intArg(call.Arguments, "limit", 20)
		if limit < 1 {
			limit = 1
		}
		if limit > 100 {
			limit = 100
		}
		db := srv.db.Model(&models.VulnLab{}).Where("is_public = ? AND is_active = ?", true, true)
		keyword := strings.TrimSpace(fmt.Sprint(call.Arguments["keyword"]))
		if keyword != "" && keyword != "<nil>" {
			like := "%" + keyword + "%"
			db = db.Where("name LIKE ? OR cve_id LIKE ? OR summary LIKE ? OR description LIKE ?", like, like, like, like)
		}
		severity := strings.TrimSpace(fmt.Sprint(call.Arguments["severity"]))
		if severity != "" && severity != "<nil>" {
			db = db.Where("severity = ?", severity)
		}
		var labs []models.VulnLab
		if err := db.Order("id DESC").Limit(limit).Find(&labs).Error; err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(labs))
		for _, lab := range labs {
			rows = append(rows, srv.getVulnLabService().ToRow(lab))
		}
		return map[string]any{"items": rows, "count": len(rows)}, nil
	case "start_vulnerability_environment":
		labID := uintArg(call.Arguments, "vuln_lab_id")
		group, err := srv.GetContainerGroupService().StartVulnLabRuntime(services.StartVulnLabRuntimeInput{UserID: user.ID, VulnLabID: labID})
		if err != nil {
			return nil, err
		}
		return map[string]any{"lab_id": labID, "status": "running", "environment": challengeContainerGroupToInfo(group)}, nil
	case "destroy_vulnerability_environment":
		labID := uintArg(call.Arguments, "vuln_lab_id")
		if err := srv.GetContainerGroupService().StopVulnLabRuntime(services.StopVulnLabRuntimeInput{UserID: user.ID, VulnLabID: labID}); err != nil {
			return nil, err
		}
		return map[string]any{"lab_id": labID, "status": "stopped"}, nil
	case "compile_vulnerability_environment":
		if !isMCPAdmin(user) {
			return nil, fmt.Errorf("administrator permission required")
		}
		labID := uintArg(call.Arguments, "vuln_lab_id")
		lab, err := srv.getVulnLabService().GetByID(labID)
		if err != nil {
			return nil, fmt.Errorf("vulnerability environment not found")
		}
		template, found, err := srv.loadTemplateModelByID(strconv.FormatUint(uint64(lab.TemplateID), 10))
		if err != nil || !found || template == nil {
			return nil, fmt.Errorf("vulnerability template not found")
		}
		if !strings.EqualFold(strings.TrimSpace(template.TemplateType), "vul") {
			return nil, fmt.Errorf("only vulnerability templates can be compiled")
		}
		task := srv.createTemplateBuildTask(strconv.FormatUint(uint64(template.Id), 10), *template)
		go srv.runTemplateBuildTask(task)
		return map[string]any{"lab_id": labID, "template_id": template.Id, "task": serializeTemplateBuildTask(task)}, nil
	default:
		return nil, fmt.Errorf("unknown or unauthorized tool: %s", call.Name)
	}
}

func uintArg(args map[string]any, key string) uint {
	value := intArg(args, key, 0)
	if value < 0 {
		return 0
	}
	return uint(value)
}

func intArg(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		if err == nil {
			return parsed
		}
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func mcpToolResult(value any, isError bool) map[string]any {
	payload, _ := json.MarshalIndent(value, "", "  ")
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(payload)}}, "structuredContent": value, "isError": isError}
}

func mcpWriteResult(w http.ResponseWriter, id json.RawMessage, result any) {
	response.JSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id), "result": result})
}

func mcpWriteError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	response.JSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id), "error": map[string]any{"code": code, "message": message}})
}
