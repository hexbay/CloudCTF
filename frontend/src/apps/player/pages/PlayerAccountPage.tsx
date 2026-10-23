import { KeyOutlined, UserOutlined } from '@ant-design/icons';
import { Alert, Avatar, Button, Card, Col, Descriptions, Input, Popconfirm, Row, Space, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { playerApi, type APIKeyStatus, type PlayerUser } from '@shared/api/player';
import { getErrorMessage } from '@shared/api/http';
import { PlayerPage } from './PlayerPage';

export function PlayerAccountPage() {
  const [user, setUser] = useState<PlayerUser | null>(null);
  const [keyStatus, setKeyStatus] = useState<APIKeyStatus>({ exists: false });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void Promise.all([playerApi.me().then(setUser), playerApi.apiKey().then(setKeyStatus)])
      .catch((error) => message.error(getErrorMessage(error, '加载账户信息失败')));
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

  return (
    <PlayerPage icon={<UserOutlined />} title="个人中心" subtitle="管理个人资料、API Key 和 MCP 访问">
      <Row gutter={[18, 18]} className="player-account-page">
        <Col xs={24} md={8}>
          <Card className="player-panel-card account-profile-card">
            <Avatar size={88} icon={<UserOutlined />} />
            <h3>{user?.username ?? 'CTFer'}</h3>
            <p>{user?.email ?? '持续训练，保持敏锐。'}</p>
          </Card>
        </Col>
        <Col xs={24} md={16}>
          <Card className="player-panel-card account-edit-card" title="账户信息">
            <Descriptions column={1}>
              <Descriptions.Item label="用户名">{user?.username ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="邮箱">{user?.email ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="姓名">{user?.full_name ?? '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24}>
          <Card className="player-panel-card" title={<Space><KeyOutlined />MCP API Key</Space>}>
            <Typography.Paragraph type="secondary">
              此密钥用于 MCP 客户端认证。每个用户只维护一个 API Key，重置时会覆盖旧密钥。
            </Typography.Paragraph>
            {keyStatus.api_key ? (
              <Alert type="info" showIcon message="当前 API Key" description={<Space.Compact style={{ width: '100%', marginTop: 10 }}><Input.Password value={keyStatus.api_key} readOnly visibilityToggle /><Button onClick={() => void copyKey()}>复制</Button></Space.Compact>} style={{ marginBottom: 16 }} />
            ) : null}
            <Descriptions column={1}>
              <Descriptions.Item label="状态">{keyStatus.exists ? '已启用' : '未生成'}</Descriptions.Item>
              {keyStatus.last_used_at ? <Descriptions.Item label="最后使用">{new Date(keyStatus.last_used_at).toLocaleString()}</Descriptions.Item> : null}
            </Descriptions>
            {!keyStatus.exists ? (
              <Button type="primary" loading={loading} onClick={() => void issueKey(false)}>随机生成 API Key</Button>
            ) : (
              <Popconfirm title="重置后旧 API Key 将立即失效，确定继续？" onConfirm={() => void issueKey(true)}>
                <Button danger loading={loading}>重置 API Key</Button>
              </Popconfirm>
            )}
          </Card>
        </Col>
      </Row>
    </PlayerPage>
  );
}
