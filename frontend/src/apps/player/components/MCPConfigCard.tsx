import { ApiOutlined, CheckCircleFilled, CopyOutlined, EyeOutlined, KeyOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Modal, Space, Tabs, Tag, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { playerApi, type MCPConfig } from "@shared/api/player";

export function MCPConfigCard() {
  const navigate = useNavigate();
  const [config, setConfig] = useState<MCPConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    setLoading(true);
    playerApi.mcpConfig().then(setConfig).catch(() => setConfig(null)).finally(() => setLoading(false));
  }, []);

  const text = config ? JSON.stringify(config, null, 2) : "";
  const server = config?.mcpServers.cloudctf;
  const hasAPIKey = useMemo(
    () => Boolean(server?.headers.Authorization && !server.headers.Authorization.includes("<YOUR_API_KEY>")),
    [server],
  );
  const clientConfigs = useMemo(() => {
    const serverUrl = server?.url ?? `${window.location.origin}/mcp`;
    const authorization = server?.headers.Authorization ?? "Bearer <YOUR_API_KEY>";
    return [
      {
        key: "codex",
        name: "Codex",
        location: "~/.codex/config.toml",
        hint: "保存后运行 codex mcp list，确认 cloudctf 服务已启用。",
        content: `[mcp_servers.cloudctf]\nurl = "${serverUrl}"\nhttp_headers = { Authorization = "${authorization}" }`,
      },
      {
        key: "claude",
        name: "Claude Code",
        location: "终端命令",
        hint: "在项目目录执行命令；如需全局使用，可在末尾添加 --scope user。",
        content: `claude mcp add --transport http cloudctf "${serverUrl}" --header "Authorization: ${authorization}"`,
      },
      {
        key: "windsurf",
        name: "Windsurf",
        location: "~/.codeium/windsurf/mcp_config.json",
        hint: "也可以在 Windsurf Settings → Cascade → MCP Servers 中打开配置文件。",
        content: JSON.stringify({
          mcpServers: {
            cloudctf: {
              serverUrl,
              headers: { Authorization: authorization },
            },
          },
        }, null, 2),
      },
    ];
  }, [server]);

  async function copyConfig() {
    await navigator.clipboard.writeText(text);
    message.success("MCP 配置已复制");
  }

  async function copyClientConfig(clientName: string, content: string) {
    await navigator.clipboard.writeText(content);
    message.success(`${clientName} 配置已复制`);
  }

  return (
    <>
      <Card className="mcp-config-card dashboard-animate-card" loading={loading}>
        <div className="mcp-config-mark" aria-hidden="true"><ApiOutlined /></div>
        <div className="mcp-config-copy">
          <div className="mcp-config-kicker">MODEL CONTEXT PROTOCOL</div>
          <div className="mcp-config-title-row">
            <h3>MCP 接入配置</h3>
            <Tag className={hasAPIKey ? "mcp-status-ready" : "mcp-status-pending"}>
              {hasAPIKey ? <CheckCircleFilled /> : <KeyOutlined />}
              {hasAPIKey ? "已就绪" : "待配置密钥"}
            </Tag>
          </div>
          <p>连接安全工具与漏洞靶场，支持检索、启动和销毁环境。</p>
          <div className="mcp-config-meta">
            <span><i /> Streamable HTTP</span>
            <code>{server?.url ?? "正在获取服务地址..."}</code>
          </div>
        </div>
        <div className="mcp-config-actions">
          <Button type="primary" icon={<CopyOutlined />} disabled={!text} onClick={() => void copyConfig()}>复制配置</Button>
          <Button icon={<EyeOutlined />} disabled={!text} onClick={() => setOpen(true)}>查看配置</Button>
          <Button className="mcp-key-button" icon={<KeyOutlined />} onClick={() => navigate("/account/profile")}>管理 API Key</Button>
        </div>
      </Card>

      <Modal
        className="mcp-config-modal"
        title={<Space><ApiOutlined />MCP 客户端配置</Space>}
        open={open}
        onCancel={() => setOpen(false)}
        width={680}
        footer={null}
      >
        <p className="mcp-modal-hint">选择客户端，将配置复制到对应位置即可连接 CloudCTF。</p>
        {!hasAPIKey ? (
          <Alert
            className="account-client-warning"
            type="warning"
            showIcon
            message="请先生成 API Key，配置中的密钥占位符才会自动替换。"
          />
        ) : null}
        <Tabs
          className="account-client-tabs mcp-client-modal-tabs"
          items={clientConfigs.map((client) => ({
            key: client.key,
            label: client.name,
            children: (
              <div className="account-client-config">
                <div className="account-client-config-head">
                  <span>{client.location}</span>
                  <Button
                    type="text"
                    icon={<CopyOutlined />}
                    onClick={() => void copyClientConfig(client.name, client.content)}
                  >
                    复制配置
                  </Button>
                </div>
                <pre><code>{client.content}</code></pre>
                <p>{client.hint}</p>
              </div>
            ),
          }))}
        />
      </Modal>
    </>
  );
}
