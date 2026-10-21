package httpserver

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloudctf/backend/internal/httpserver/response"
	"cloudctf/backend/internal/services"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type playerShellEnvelope struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint   `json:"cols,omitempty"`
	Rows uint   `json:"rows,omitempty"`
}

func (srv *Server) playerVulnLabShellWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	slog.Default().Info("vuln lab shell request received", "path", r.URL.Path, "remote", r.RemoteAddr)
	user, ok := srv.currentUserFromRequest(r)
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"detail": "auth required"})
		return
	}

	labID := resourceID(r.URL.Path)
	if labID == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": "lab_id required"})
		return
	}

	labService := srv.getVulnLabService()
	if labService == nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{"detail": "service unavailable"})
		return
	}
	lab, err := labService.GetByID(labID)
	if err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "lab not found"})
		return
	}

	group, found := srv.findActiveVulnLabGroupForUser(user.ID, lab.ID)
	if !found {
		response.JSON(w, http.StatusNotFound, map[string]any{"detail": "active lab environment not found"})
		return
	}

	containerID, endpoint, err := srv.resolveVulnLabShellContainer(group)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	slog.Default().Info("vuln lab shell resolved container", "lab_id", lab.ID, "user_id", user.ID, "container_id", containerID, "endpoint", endpoint)

	if err := streamDockerExecShell(w, r, endpoint, containerID); err != nil {
		slog.Default().Error("vuln lab shell stream failed", "error", err, "container_id", containerID)
		if !strings.Contains(strings.ToLower(err.Error()), "hijack") {
			response.JSON(w, http.StatusBadGateway, map[string]any{"detail": err.Error()})
		}
	}
}

func (srv *Server) findActiveVulnLabGroupForUser(userID, labID uint) (map[string]any, bool) {
	service := srv.GetContainerGroupService()
	if service == nil {
		return nil, false
	}
	current, _, _ := service.ListContainerGroups(1, 1000, map[string][]string{})
	for _, row := range current {
		if uintValue(row["user_id"]) != userID || uintValue(row["vuln_lab_id"]) != labID {
			continue
		}
		status := cleanString(row["status"])
		if status != "running" && status != "created" && status != "creating" {
			continue
		}
		return row, true
	}
	return nil, false
}

func (srv *Server) resolveVulnLabShellContainer(group map[string]any) (string, string, error) {
	containerID := cleanString(group["container_id"])
	if containerID == "" {
		containers := services.NormalizeTemplateServiceRows(group["containers"])
		for _, item := range containers {
			if candidate := cleanString(item["container_id"]); candidate != "" {
				containerID = candidate
				break
			}
		}
	}
	if containerID == "" {
		return "", "", fmt.Errorf("container not found")
	}

	endpoint := defaultDockerEndpoint()
	if hostID := uintValue(group["host_id"]); hostID != 0 {
		host, err := srv.getVirtualizationHostService().GetByID(hostID)
		if err == nil && strings.TrimSpace(host.Endpoint) != "" {
			endpoint = strings.TrimSpace(host.Endpoint)
		}
	}
	return containerID, endpoint, nil
}

func streamDockerExecShell(w http.ResponseWriter, r *http.Request, endpoint, containerID string) error {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("websocket is not supported by this server")
	}

	acceptKey := websocketAcceptKey(r.Header.Get("Sec-WebSocket-Key"))
	if acceptKey == "" {
		return fmt.Errorf("invalid websocket handshake")
	}

	header := w.Header()
	header.Set("Upgrade", "websocket")
	header.Set("Connection", "Upgrade")
	header.Set("Sec-WebSocket-Accept", acceptKey)
	w.WriteHeader(http.StatusSwitchingProtocols)

	conn, rw, err := hj.Hijack()
	if err != nil {
		return err
	}

	cli, err := services.DockerClientForEndpoint(endpoint)
	if err != nil {
		_ = conn.Close()
		return err
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	execID, err := createInteractiveDockerExec(ctx, cli, containerID)
	if err != nil {
		_ = cli.Close()
		_ = conn.Close()
		return err
	}

	attachResp, err := cli.ContainerExecAttach(ctx, execID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		_ = cli.Close()
		_ = conn.Close()
		return err
	}

	go func() {
		defer cancel()
		defer attachResp.Close()
		defer cli.Close()
		defer conn.Close()

		_, _ = io.Copy(newShellWebsocketWriter(conn), attachResp.Reader)
	}()

	if err := resizeDockerExec(ctx, cli, execID, 140, 36); err != nil {
		// ignore initial resize failures
	}
	_, _ = attachResp.Conn.Write([]byte("\r"))

	reader := bufio.NewReader(rw)
	for {
		messageType, payload, err := readWebSocketFrame(reader)
		if err != nil {
			return nil
		}
		if messageType == 0x8 {
			return nil
		}
		if messageType != 0x1 {
			continue
		}

		msg := playerShellEnvelope{}
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "input":
			data, err := base64.StdEncoding.DecodeString(msg.Data)
			if err != nil {
				continue
			}
			if _, err := attachResp.Conn.Write(data); err != nil {
				return nil
			}
		case "resize":
			_ = resizeDockerExec(ctx, cli, execID, msg.Cols, msg.Rows)
		case "ping":
			_ = writeWebSocketTextFrame(conn, []byte(`{"type":"pong"}`))
		}
	}
}

func createInteractiveDockerExec(ctx context.Context, cli dockerExecClient, containerID string) (string, error) {
	resp, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh", "-lc", "export TERM=xterm-256color; (command -v bash >/dev/null 2>&1 && exec bash -il) || exec sh -il"},
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func resizeDockerExec(ctx context.Context, cli dockerExecClient, execID string, cols, rows uint) error {
	if cols == 0 {
		cols = 140
	}
	if rows == 0 {
		rows = 36
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return cli.ContainerExecResize(timeoutCtx, execID, container.ResizeOptions{
		Width:  cols,
		Height: rows,
	})
}

type dockerExecClient interface {
	ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (container.ExecCreateResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error
	Close() error
}

type shellWebsocketWriter struct {
	conn io.Writer
}

func newShellWebsocketWriter(conn io.Writer) *shellWebsocketWriter {
	return &shellWebsocketWriter{conn: conn}
}

func (w *shellWebsocketWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	framePayload, _ := json.Marshal(playerShellEnvelope{
		Type: "output",
		Data: base64.StdEncoding.EncodeToString(p),
	})
	if err := writeWebSocketTextFrame(w.conn, framePayload); err != nil {
		return 0, err
	}
	return len(p), nil
}
