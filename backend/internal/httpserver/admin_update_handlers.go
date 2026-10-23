package httpserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/models"
	"cloudctf/backend/internal/updater"
)

type updateJobSnapshot struct {
	ID          string               `json:"id"`
	Tag         string               `json:"tag"`
	Status      string               `json:"status"`
	Stage       string               `json:"stage"`
	Message     string               `json:"message"`
	StartedAt   string               `json:"started_at"`
	UpdatedAt   string               `json:"updated_at"`
	Error       string               `json:"error,omitempty"`
	FromVersion string               `json:"from_version"`
	Backup      any                  `json:"backup,omitempty"`
	Result      *updater.ApplyResult `json:"result,omitempty"`
}

type updateJobState struct {
	mu       sync.RWMutex
	job      *updateJobSnapshot
	sequence atomic.Uint64
}

func (state *updateJobState) begin(tag, currentVersion string) (updateJobSnapshot, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.job != nil {
		switch state.job.Status {
		case "running", "installed", "restarting":
			return *state.job, false
		}
	}
	now := time.Now().Format(time.RFC3339)
	state.job = &updateJobSnapshot{
		ID:          fmt.Sprintf("update-%d-%d", time.Now().UnixNano(), state.sequence.Add(1)),
		Tag:         tag,
		Status:      "running",
		Stage:       "queued",
		Message:     "更新任务已提交，正在创建备份",
		StartedAt:   now,
		UpdatedAt:   now,
		FromVersion: currentVersion,
	}
	return *state.job, true
}

func (state *updateJobState) update(jobID string, patch func(*updateJobSnapshot)) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.job == nil || state.job.ID != jobID {
		return
	}
	patch(state.job)
	state.job.UpdatedAt = time.Now().Format(time.RFC3339)
}

func (state *updateJobState) current() *updateJobSnapshot {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if state.job == nil {
		return nil
	}
	snapshot := *state.job
	return &snapshot
}

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
	job, started := srv.updateJob.begin(tag, srv.currentVersion())
	if !started {
		response.JSON(w, http.StatusConflict, map[string]any{
			"detail": "an update is already in progress",
			"job":    job,
		})
		return
	}
	go srv.runSystemUpdateJob(job.ID, tag, user, r.RemoteAddr, r.UserAgent())
	response.JSON(w, http.StatusAccepted, map[string]any{
		"success": true,
		"message": "update started",
		"job":     job,
	})
}

func (srv *Server) adminUpdateUploadHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}
	if !user.IsSuperuser {
		response.JSON(w, http.StatusForbidden, map[string]any{"detail": "只有超级管理员可以安装系统更新"})
		return
	}
	if srv.updater == nil || srv.requestRestart == nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "当前进程不支持自动更新"})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "无效的上传请求或文件超过大小限制"})
		return
	}
	version := strings.TrimSpace(r.FormValue("version"))
	if version == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "必须填写目标版本号"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "必须上传更新二进制文件"})
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > updater.MaxUploadSize {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "更新文件为空或超过 512 MiB 限制"})
		return
	}

	tmp, err := os.CreateTemp("", "cloudctf-update-upload-*")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "无法创建更新暂存文件"})
		return
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "无法保护更新暂存文件"})
		return
	}
	written, err := io.Copy(tmp, io.LimitReader(file, updater.MaxUploadSize+1))
	if err != nil || written != header.Size || written > updater.MaxUploadSize {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "更新文件上传不完整或超过大小限制"})
		return
	}
	if err := tmp.Sync(); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "无法写入更新暂存文件"})
		return
	}
	if err := tmp.Close(); err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "无法关闭更新暂存文件"})
		return
	}
	if err := srv.updater.ValidateUploaded(version, tmpPath); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}

	job, started := srv.updateJob.begin(version, srv.currentVersion())
	if !started {
		response.JSON(w, http.StatusConflict, map[string]any{"detail": "已有更新任务正在进行", "job": job})
		return
	}
	keep = true
	go srv.runManualSystemUpdateJob(job.ID, version, tmpPath, user, r.RemoteAddr, r.UserAgent())
	response.JSON(w, http.StatusAccepted, map[string]any{
		"success": true,
		"message": "manual update started",
		"job":     job,
	})
}

func (srv *Server) adminUpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"current_version": srv.currentVersion(),
		"job":             srv.updateJob.current(),
	})
}

func (srv *Server) runSystemUpdateJob(jobID, tag string, user models.User, remoteAddr, userAgent string) {
	srv.runSystemUpdateJobWith(jobID, tag, "备份完成，正在下载并校验新版本", user, remoteAddr, userAgent, func(ctx context.Context) (updater.ApplyResult, error) {
		return srv.updater.Apply(ctx, tag)
	})
}

func (srv *Server) runManualSystemUpdateJob(jobID, version, uploadPath string, user models.User, remoteAddr, userAgent string) {
	defer os.Remove(uploadPath)
	srv.runSystemUpdateJobWith(jobID, version, "备份完成，正在校验并安装上传文件", user, remoteAddr, userAgent, func(ctx context.Context) (updater.ApplyResult, error) {
		return srv.updater.ApplyUploaded(ctx, version, uploadPath)
	})
}

func (srv *Server) runSystemUpdateJobWith(
	jobID, tag, installMessage string,
	user models.User,
	remoteAddr, userAgent string,
	apply func(context.Context) (updater.ApplyResult, error),
) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	srv.updateJob.update(jobID, func(job *updateJobSnapshot) {
		job.Stage = "backup"
		job.Message = "正在创建更新前数据备份"
	})
	backup, err := srv.getBackupService().CreateArchive(srv.config, "system-update-"+user.Username)
	if err != nil {
		srv.logger.Error("pre-update backup failed", "tag", tag, "username", user.Username, "error", err)
		srv.updateJob.update(jobID, func(job *updateJobSnapshot) {
			job.Status = "failed"
			job.Stage = "backup"
			job.Message = "更新前备份失败"
			job.Error = err.Error()
		})
		return
	}
	srv.updateJob.update(jobID, func(job *updateJobSnapshot) {
		job.Backup = backup
		job.Stage = "download"
		job.Message = installMessage
	})

	result, err := apply(ctx)
	if err != nil {
		srv.logger.Error("system update failed", "tag", tag, "username", user.Username, "error", err)
		srv.updateJob.update(jobID, func(job *updateJobSnapshot) {
			job.Status = "failed"
			job.Stage = "apply"
			job.Message = "系统更新失败"
			job.Error = err.Error()
		})
		return
	}

	_, _ = srv.getAuditLogService().Create(map[string]any{
		"user_id":       user.ID,
		"username":      user.Username,
		"action_type":   "system_update",
		"resource_type": "system",
		"resource_id":   result.Version,
		"description":   fmt.Sprintf("系统在线更新: %s -> %s", result.PreviousVersion, result.Version),
		"ip_address":    remoteAddr,
		"user_agent":    userAgent,
	})
	srv.updateJob.update(jobID, func(job *updateJobSnapshot) {
		job.Status = "restarting"
		job.Stage = "restart"
		job.Message = "新版本已安装，正在重启服务"
		job.Result = &result
	})
	srv.logger.Info("system update installed", "from", result.PreviousVersion, "to", result.Version, "username", user.Username)
	time.AfterFunc(750*time.Millisecond, srv.requestRestart)
}
