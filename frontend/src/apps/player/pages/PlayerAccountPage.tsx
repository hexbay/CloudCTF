import {
  CheckCircleFilled,
  ClockCircleOutlined,
  CopyOutlined,
  IdcardOutlined,
  KeyOutlined,
  MailOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Alert, Avatar, Button, Card, Input, Popconfirm, Skeleton, Tabs, Typography, message } from 'antd';
import { useEffect, useState, type ReactNode } from 'react';
import { playerApi, type APIKeyStatus, type MCPConfig, type PlayerUser } from '@shared/api/player';
import { getErrorMessage } from '@shared/api/http';
import { PlayerPage } from './PlayerPage';

export function PlayerAccountPage() {
  const [user, setUser] = useState<PlayerUser | null>(null);
  const [keyStatus, setKeyStatus] = useState<APIKeyStatus>({ exists: false });
  const [mcpConfig, setMcpConfig] = useState<MCPConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [initializing, setInitializing] = useState(true);

  useEffect(() => {
    void Promise.all([
      playerApi.me().then(setUser),
      playerApi.apiKey().then(setKeyStatus),
      playerApi.mcpConfig().then(setMcpConfig),
    ])
      .catch((error) => message.error(getErrorMessage(error, '加载账户信息失败')))
      .finally(() => setInitializing(false));
  }, []);

  async function issueKey(reset: boolean) {
    setLoading(true);
    try {
      const result = reset ? await playerApi.resetApiKey() : await playerApi.generateApiKey();
      setKeyStatus(result);
      message.success(reset ? 'API Key 已重置，旧密钥已立即失效' : 'API Key 已生成');
    } catch (error) {
      message.error(getErrorMessage(error, 'API Key 操作失败'));
    } finally {
      setLoading(false);
    }
  }

  async function copyKey() {
    await navigator.clipboard.writeText(keyStatus.api_key ?? '');
    message.success('API Key 已复制');
  }

  const displayName = user?.full_name || user?.username || 'CTFer';
  const lastUsed = keyStatus.last_used_at
    ? new Date(keyStatus.last_used_at).toLocaleString('zh-CN')
    : '尚无使用记录';
  const mcpServerUrl = mcpConfig?.mcpServers.cloudctf?.url ?? `${window.location.origin}/mcp`;
  const apiKey = keyStatus.api_key ?? '<YOUR_API_KEY>';
  const clientConfigs = createClientConfigs(mcpServerUrl, apiKey);

  return (
    <PlayerPage
      icon={<UserOutlined />}
      title="个人中心"
      subtitle="管理你的账户资料与 MCP 访问凭据"
      showTitle
    >
      <div className="player-account-page">
        <aside className="account-sidebar">
          <Card className="player-panel-card account-profile-card">
            <div className="account-profile-glow" />
            <Avatar size={96} icon={<UserOutlined />} />
            {initializing ? (
              <Skeleton active title={{ width: 100 }} paragraph={{ rows: 1, width: 180 }} />
            ) : (
              <>
                <Typography.Title level={3}>{displayName}</Typography.Title>
                <Typography.Text>{user?.email ?? '持续训练，保持敏锐。'}</Typography.Text>
              </>
            )}
            <div className="account-role-badge">
              <SafetyCertificateOutlined />
              CloudCTF 成员
            </div>
          </Card>

          <div className="account-security-note">
            <SafetyCertificateOutlined />
            <div>
              <strong>安全提示</strong>
              <span>请勿在公开仓库、聊天记录或截图中泄露 API Key。</span>
            </div>
          </div>
        </aside>

        <div className="account-main-column">
          <Card
            className="player-panel-card account-info-card"
            title={<AccountCardTitle icon={<IdcardOutlined />} eyebrow="PROFILE" title="账户信息" />}
          >
            {initializing ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <div className="account-info-grid">
                <AccountInfoItem icon={<UserOutlined />} label="用户名" value={user?.username} />
                <AccountInfoItem icon={<MailOutlined />} label="邮箱地址" value={user?.email} />
                <AccountInfoItem icon={<IdcardOutlined />} label="姓名" value={user?.full_name} wide />
              </div>
            )}
          </Card>

          <Card
            className="player-panel-card account-key-card"
            title={<AccountCardTitle icon={<KeyOutlined />} eyebrow="MCP ACCESS" title="API Key" />}
            extra={
              <span className={`account-key-status ${keyStatus.exists ? 'is-active' : ''}`}>
                {keyStatus.exists ? <CheckCircleFilled /> : <i />}
                {keyStatus.exists ? '已启用' : '未生成'}
              </span>
            }
          >
            <Typography.Paragraph className="account-key-description">
              用于 MCP 客户端认证。每个账户仅保留一个有效密钥，重置后旧密钥会立即失效。
            </Typography.Paragraph>

            {keyStatus.api_key ? (
              <Alert
                className="account-key-alert"
                type="info"
                showIcon
                message="请立即保存新生成的密钥"
                description={
                  <div className="account-key-field">
                    <Input.Password value={keyStatus.api_key} readOnly visibilityToggle />
                    <Button icon={<CopyOutlined />} onClick={() => void copyKey()}>复制密钥</Button>
                  </div>
                }
              />
            ) : null}

            <div className="account-key-footer">
              <div className="account-key-meta">
                <ClockCircleOutlined />
                <span>最后使用</span>
                <strong>{lastUsed}</strong>
              </div>
              {!keyStatus.exists ? (
                <Button type="primary" icon={<KeyOutlined />} loading={loading} onClick={() => void issueKey(false)}>
                  生成 API Key
                </Button>
              ) : (
                <Popconfirm
                  title="重置 API Key"
                  description="旧 API Key 将立即失效，确定继续？"
                  okText="确认重置"
                  cancelText="取消"
                  onConfirm={() => void issueKey(true)}
                >
                  <Button danger icon={<ReloadOutlined />} loading={loading}>重置密钥</Button>
                </Popconfirm>
              )}
            </div>
          </Card>

          <Card
            className="player-panel-card account-client-card"
            title={<AccountCardTitle icon={<SafetyCertificateOutlined />} eyebrow="QUICK START" title="MCP 客户端配置" />}
          >
            <Typography.Paragraph className="account-client-description">
              选择你正在使用的客户端，将配置复制到对应位置即可连接 CloudCTF。
            </Typography.Paragraph>
            {!keyStatus.exists ? (
              <Alert
                className="account-client-warning"
                type="warning"
                showIcon
                message="请先生成 API Key，配置中的密钥占位符才会自动替换。"
              />
            ) : null}
            <Tabs
              className="account-client-tabs"
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
          </Card>
        </div>
      </div>
    </PlayerPage>
  );
}

async function copyClientConfig(clientName: string, content: string) {
  await navigator.clipboard.writeText(content);
  message.success(`${clientName} 配置已复制`);
}

function createClientConfigs(serverUrl: string, apiKey: string) {
  const authorization = `Bearer ${apiKey}`;
  return [
    {
      key: 'codex',
      name: 'Codex',
      location: '~/.codex/config.toml',
      hint: '保存后运行 codex mcp list，确认 cloudctf 服务已启用。',
      content: `[mcp_servers.cloudctf]\nurl = "${serverUrl}"\nhttp_headers = { Authorization = "${authorization}" }`,
    },
    {
      key: 'claude',
      name: 'Claude Code',
      location: '终端命令',
      hint: '在项目目录执行命令；如需全局使用，可在命令末尾添加 --scope user。',
      content: `claude mcp add --transport http cloudctf "${serverUrl}" --header "Authorization: ${authorization}"`,
    },
    {
      key: 'windsurf',
      name: 'Windsurf',
      location: '~/.codeium/windsurf/mcp_config.json',
      hint: '也可以在 Windsurf Settings → Cascade → MCP Servers 中打开配置文件。',
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
}

function AccountCardTitle({ icon, eyebrow, title }: { icon: ReactNode; eyebrow: string; title: string }) {
  return (
    <div className="account-card-title">
      <span className="account-card-title-icon">{icon}</span>
      <span>
        <small>{eyebrow}</small>
        <strong>{title}</strong>
      </span>
    </div>
  );
}

function AccountInfoItem({
  icon,
  label,
  value,
  wide = false,
}: {
  icon: ReactNode;
  label: string;
  value?: string | null;
  wide?: boolean;
}) {
  return (
    <div className={`account-info-item ${wide ? 'is-wide' : ''}`}>
      <span className="account-info-icon">{icon}</span>
      <span className="account-info-copy">
        <small>{label}</small>
        <strong>{value || '未设置'}</strong>
      </span>
    </div>
  );
}
