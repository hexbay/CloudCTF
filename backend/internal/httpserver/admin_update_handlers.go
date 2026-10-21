package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
)

func (srv *Server) adminUpdateCheckHandler(w http.ResponseWriter, r *http.Request) {
	if srv.updater == nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "update service unavailable"})
		return
	}
	result, err := srv.updater.Check(r.Context())
	if err != nil {
		srv.logger.Error("update check failed", "error", err)
		response.JSON(w, http.StatusBadGateway, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (srv *Server) adminUpdateApplyHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	if !user.IsSuperuser {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "only a super administrator can install system updates"})
		return
	}
	if srv.requestRestart == nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "this process does not support automatic restart"})
		return
	}
	payload, valid := readOptionalJSONMap(r)
	if !valid {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid request body"})
		return
	}
	tag := strings.TrimSpace(fmt.Sprint(payload["tag"]))
	if tag == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "release tag is required"})
		return
	}
	result, err := srv.updater.Apply(r.Context(), tag)
	if err != nil {
		srv.logger.Error("system update failed", "tag", tag, "username", user.Username, "error", err)
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}

	_, _ = srv.getAuditLogService().Create(map[string]any{
		"user_id":       user.ID,
		"username":      user.Username,
		"action_type":   "system_update",
		"resource_type": "system",
		"resource_id":   result.Version,
		"description":   fmt.Sprintf("系统在线更新: %s -> %s", result.PreviousVersion, result.Version),
		"ip_address":    r.RemoteAddr,
		"user_agent":    r.UserAgent(),
	})
	response.JSON(w, http.StatusAccepted, map[string]any{
		"success": true,
		"message": "update installed; service is restarting",
		"data":    result,
	})
	srv.logger.Info("system update installed", "from", result.PreviousVersion, "to", result.Version, "username", user.Username)
	time.AfterFunc(750*time.Millisecond, srv.requestRestart)
}
