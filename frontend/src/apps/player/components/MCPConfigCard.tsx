import { ApiOutlined, CopyOutlined, DownloadOutlined } from "@ant-design/icons";
import { Button, Card, Input, Space, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { playerApi, type MCPConfig } from "@shared/api/player";

export function MCPConfigCard() {
  const [config, setConfig] = useState<MCPConfig | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    playerApi.mcpConfig().then(setConfig).catch(() => setConfig(null)).finally(() => setLoading(false));
  }, []);

  const text = config ? JSON.stringify(config, null, 2) : "";

  async function copyConfig() {
    await navigator.clipboard.writeText(text);
    message.success("MCP 配置已复制");
  }

  function downloadConfig() {
    const url = URL.createObjectURL(new Blob([text], { type: "application/json" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "cloudctf-mcp.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  return (
    <Card className="player-list-card dashboard-section-card dashboard-animate-card" title={<Space><ApiOutlined />MCP 导入配置</Space>} extra={<Button type="link" href="/account/profile">管理 API Key</Button>} loading={loading}>
      <Typography.Paragraph type="secondary">
        配置会自动带入当前用户的 API Key，可直接导入支持 Streamable HTTP 的 MCP 客户端。尚未生成密钥时，请先前往个人中心生成。
      </Typography.Paragraph>
      <Input.TextArea value={text} readOnly autoSize={{ minRows: 7, maxRows: 12 }} />
      <Space style={{ marginTop: 12 }}>
        <Button type="primary" icon={<CopyOutlined />} disabled={!text} onClick={() => void copyConfig()}>复制配置</Button>
        <Button icon={<DownloadOutlined />} disabled={!text} onClick={downloadConfig}>下载 JSON</Button>
      </Space>
    </Card>
  );
}
