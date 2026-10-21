import { Alert, Modal } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import { request } from "@shared/api/http";

type VulnLabShellModalProps = {
  open: boolean;
  labId?: number | null;
  title?: string;
  onClose: () => void;
};

export function VulnLabShellModal({
  open,
  labId,
  title,
  onClose,
}: VulnLabShellModalProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const resizeTimerRef = useRef<number | null>(null);
  const connectedRef = useRef(false);
  const [error, setError] = useState("");
  const [connecting, setConnecting] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [minimized, setMinimized] = useState(false);

  const shellUrl = useMemo(() => {
    if (!labId) {
      return "";
    }
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const basePath = `/api/vuln-labs/${labId}/shell`;
    const devHost = import.meta.env.DEV
      ? `${protocol}//127.0.0.1:8005`
      : `${protocol}//${window.location.host}`;
    return `${devHost}${basePath}`;
  }, [labId]);

  useEffect(() => {
    if (!open || minimized || !labId || !hostRef.current) {
      return;
    }

    setError("");
    setConnecting(true);
    connectedRef.current = false;
    const terminal = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontSize: 14,
      fontFamily: "Consolas, 'Courier New', monospace",
      theme: {
        background: "#08111c",
        foreground: "#e2e8f0",
        cursor: "#67e8f9",
        black: "#0f172a",
        red: "#f87171",
        green: "#4ade80",
        yellow: "#fbbf24",
        blue: "#60a5fa",
        magenta: "#c084fc",
        cyan: "#22d3ee",
        white: "#e2e8f0",
      },
    });
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(hostRef.current);
    fitAddon.fit();
    terminal.focus();
    terminalRef.current = terminal;
    fitAddonRef.current = fitAddon;

    let disposed = false;
    let socket: WebSocket | null = null;

    void request<{ ticket: string }>("/api/stream-ticket", {
      method: "POST",
      body: { path: `/api/vuln-labs/${labId}/shell` },
    }).then(({ ticket }) => {
      if (disposed) return;
      const url = new URL(shellUrl);
      url.searchParams.set("stream_ticket", ticket);
      socket = new WebSocket(url.toString());
      socketRef.current = socket;
      socket.onopen = () => {
        connectedRef.current = true;
        setConnecting(false);
        setError("");
        sendResize();
      };
      socket.onmessage = async (event) => {
        const payload = typeof event.data === "string" ? event.data : await event.data.text();
        const message = JSON.parse(payload) as { type?: string; data?: string };
        if (message.type === "output" && message.data) terminal.write(atob(message.data));
      };
      socket.onerror = socket.onclose = () => {
        setConnecting(false);
        if (!connectedRef.current) setError("Shell 连接失败");
      };
    }).catch(() => {
      if (!disposed) {
        setConnecting(false);
        setError("无法获取 Shell 连接凭据");
      }
    });

    terminal.onData((data) => {
      if (socket?.readyState !== WebSocket.OPEN) {
        return;
      }
      socket.send(
        JSON.stringify({
          type: "input",
          data: btoa(data),
        }),
      );
    });

    const handleResize = () => {
      if (resizeTimerRef.current) {
        window.clearTimeout(resizeTimerRef.current);
      }
      resizeTimerRef.current = window.setTimeout(() => {
        sendResize();
      }, 80);
    };

    const sendResize = () => {
      fitAddon.fit();
      if (socket?.readyState !== WebSocket.OPEN) {
        return;
      }
      socket.send(
        JSON.stringify({
          type: "resize",
          cols: terminal.cols,
          rows: terminal.rows,
        }),
      );
    };

    window.addEventListener("resize", handleResize);
    const initTimer = window.setTimeout(() => {
      sendResize();
    }, 50);

    return () => {
	  disposed = true;
      window.removeEventListener("resize", handleResize);
      window.clearTimeout(initTimer);
      if (resizeTimerRef.current) {
        window.clearTimeout(resizeTimerRef.current);
      }
      socket?.close();
      terminal.dispose();
      socketRef.current = null;
      terminalRef.current = null;
      fitAddonRef.current = null;
    };
  }, [open, labId, shellUrl, fullscreen, minimized]);

  useEffect(() => {
    if (!open) {
      setMinimized(false);
      setFullscreen(false);
    }
  }, [open]);

  return (
    <Modal
      className={`player-vuln-shell-modal${fullscreen ? " is-fullscreen" : ""}${minimized ? " is-minimized" : ""}`}
      rootClassName={`player-vuln-shell-modal-root${fullscreen ? " is-fullscreen" : ""}${minimized ? " is-minimized" : ""}`}
      open={open}
      mask={!minimized}
      title={
        <div className="player-vuln-shell-titlebar">
          <div className="player-vuln-shell-traffic">
            <button
              className="player-vuln-shell-traffic-button is-close"
              type="button"
              aria-label="关闭"
              onClick={onClose}
            />
            <button
              className="player-vuln-shell-traffic-button is-min"
              type="button"
              aria-label={minimized ? "恢复" : "最小化"}
              onClick={() => setMinimized((value) => !value)}
            />
            <button
              className="player-vuln-shell-traffic-button is-max"
              type="button"
              aria-label={fullscreen ? "退出全屏" : "全屏"}
              onClick={() => {
                setMinimized(false);
                setFullscreen((value) => !value);
              }}
            />
          </div>
          <div className="player-vuln-shell-title">{title || "环境终端"}</div>
          <div className="player-vuln-shell-titlebar-spacer" />
        </div>
      }
      footer={null}
      closable={false}
      width={fullscreen ? "100vw" : 1200}
      style={
        minimized
          ? { top: "auto", paddingBottom: 0 }
          : fullscreen
            ? { top: 0, paddingBottom: 0, maxWidth: "100vw" }
            : { top: 24 }
      }
      destroyOnHidden
      onCancel={onClose}
    >
      <div className="player-vuln-shell-body">
        {!minimized ? (
          <>
            {connecting ? (
              <Alert type="info" showIcon message="正在连接 Shell..." />
            ) : null}
            {error ? <Alert type="error" showIcon message={error} /> : null}
            <div className="player-vuln-shell-host" ref={hostRef} />
          </>
        ) : (
          <button
            type="button"
            className="player-vuln-shell-minimized-bar"
            onClick={() => setMinimized(false)}
          >
            <span>{title || "环境终端"}</span>
            <em>{connecting ? "连接中" : error ? "连接异常" : "运行中"}</em>
          </button>
        )}
      </div>
    </Modal>
  );
}
