package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
)

func (srv *Server) settingsInfoHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, srv.systemInfoSnapshot())
}

func (srv *Server) adminSettingsBackupCreateHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.requireAdminBackupUser(w, r)
	if !ok {
		return
	}
	service := srv.getBackupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	backupMu.Lock()
	defer backupMu.Unlock()
	info, err := service.CreateArchive(srv.currentConfig(), user.Username)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "id": info.Name, "name": info.Name, "backup": info})
}

func (srv *Server) adminSettingsBackupListHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := srv.requireAdminBackupUser(w, r); !ok {
		return
	}
	service := srv.getBackupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	items, err := service.ListFiles(srv.currentConfig().BackupDir)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"id":         item.Name,
			"name":       item.Name,
			"size":       item.Size,
			"created_at": item.CreatedAt,
			"updated_at": item.CreatedAt,
		})
	}
	response.JSON(w, http.StatusOK, rows)
}

func (srv *Server) adminSettingsBackupRestoreHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.requireAdminBackupUser(w, r)
	if !ok {
		return
	}
	service := srv.getBackupService()
	if service == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	name := filepath.Base(r.PathValue("id"))
	fullPath := filepath.Join(srv.currentConfig().BackupDir, name)
	if _, err := os.Stat(fullPath); err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "backup not found"})
		return
	}
	backupMu.Lock()
	defer backupMu.Unlock()
	manifest, err := service.RestoreArchive(srv.currentConfig(), fullPath, user.Username)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "restored_id": name, "manifest": manifest})
}

func (srv *Server) adminSettingsBackupDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := srv.requireAdminBackupUser(w, r); !ok {
		return
	}
	name := filepath.Base(r.PathValue("id"))
	fullPath := filepath.Join(srv.currentConfig().BackupDir, name)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			response.JSON(w, http.StatusNotFound, map[string]any{"detail": "backup not found"})
			return
		}
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true})
}

func (srv *Server) adminSettingsBackupDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := srv.requireAdminBackupUser(w, r); !ok {
		return
	}
	name := filepath.Base(r.PathValue("id"))
	fullPath := filepath.Join(srv.currentConfig().BackupDir, name)
	if _, err := os.Stat(fullPath); err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "backup not found"})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(name)))
	http.ServeFile(w, r, fullPath)
}

func (srv *Server) adminSettingsBackupUploadHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := srv.requireAdminBackupUser(w, r); !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid multipart form"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "file is required"})
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to read uploaded file"})
		return
	}
	name := filepath.Base(headerFilename(r))
	if name == "" {
		name = fmt.Sprintf("cloudctf-backup-%s.zip", time.Now().Format("20060102-150405"))
	}
	if !filepath.IsAbs(srv.currentConfig().BackupDir) && srv.currentConfig().BackupDir == "" {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "backup dir not configured"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "uploaded backup must be a .zip file"})
		return
	}
	if err := os.MkdirAll(srv.currentConfig().BackupDir, 0o755); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}
	target := filepath.Join(srv.currentConfig().BackupDir, name)
	if err := os.WriteFile(target, content, 0o644); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "failed to save backup"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "id": name, "name": name})
}

func headerFilename(r *http.Request) string {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return ""
	}
	if files := r.MultipartForm.File["file"]; len(files) > 0 {
		return files[0].Filename
	}
	return ""
}

func adminSettingsCacheClearHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"success": true, "cleared_at": time.Now()})
}

func (srv *Server) adminSettingsLogsHandler(w http.ResponseWriter, r *http.Request) {
	service := srv.getAuditLogService()
	if service == nil {
		response.JSON(w, http.StatusOK, []map[string]any{})
		return
	}
	response.JSON(w, http.StatusOK, service.RecentRows(200))
}
