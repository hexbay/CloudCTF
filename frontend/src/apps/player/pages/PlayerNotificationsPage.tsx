import {
  BellOutlined,
  MessageOutlined,
  NotificationOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { Button, Card, Drawer, List, Space, Tag, Typography, message } from 'antd';
import { useEffect, useState } from 'react';
import { playerApi, type PlayerNotification } from '@shared/api/player';
import { PlayerPage } from './PlayerPage';

const previewEmptyState = false;

export function PlayerNotificationsPage() {
  const [rows, setRows] = useState<PlayerNotification[]>([]);
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<PlayerNotification | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const result = await playerApi.notifications({ current: 1, pageSize: 100 });
      setRows(result.data ?? []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载通知失败');
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(item: PlayerNotification) {
    setDetail(item);
    if (!item.is_read) {
      try {
        await playerApi.markNotificationRead(item.id);
        setRows((current) =>
          current.map((row) => (row.id === item.id ? { ...row, is_read: true } : row)),
        );
      } catch {
        // Reading state is best-effort; keep the detail view available.
      }
    }
  }

  async function markAllRead() {
    setLoading(true);
    try {
      await playerApi.markAllNotificationsRead();
      setRows((current) => current.map((row) => ({ ...row, is_read: true })));
      setDetail((current) => (current ? { ...current, is_read: true } : current));
      message.success('已全部标记为已读');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '一键已读失败');
    } finally {
      setLoading(false);
    }
  }

  const showEmptyState = previewEmptyState || rows.length === 0;
  const hasUnread = rows.some((item) => !item.is_read);

  return (
    <PlayerPage icon={<BellOutlined />} title="通知中心">
      {!showEmptyState ? (
        <>
          <Card className="player-panel-card" loading={loading}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                marginBottom: 16,
              }}
            >
              <Button type="primary" onClick={() => void markAllRead()} disabled={!hasUnread} loading={loading}>
                一键已读
              </Button>
            </div>
            <List
              dataSource={rows}
              renderItem={(item) => (
                <List.Item onClick={() => void openDetail(item)} style={{ cursor: 'pointer' }}>
                  <List.Item.Meta
                    title={<Typography.Text strong={!item.is_read}>{item.title}</Typography.Text>}
                    description={formatTime(item.created_at)}
                  />
                  <Space>
                    <Tag color="cyan">{typeLabel(item.message_type)}</Tag>
                    <Tag color={item.is_read ? 'default' : 'blue'}>{item.is_read ? '已读' : '未读'}</Tag>
                  </Space>
                </List.Item>
              )}
            />
          </Card>

          <Drawer
            title={detail?.title ?? '通知详情'}
            open={Boolean(detail)}
            onClose={() => setDetail(null)}
            width={520}
          >
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <div>
                <Typography.Text type="secondary">类型</Typography.Text>
                <div>
                  <Tag>{typeLabel(detail?.message_type)}</Tag>
                </div>
              </div>
              <div>
                <Typography.Text type="secondary">时间</Typography.Text>
                <div>{formatTime(detail?.created_at)}</div>
              </div>
              <div>
                <Typography.Text type="secondary">内容</Typography.Text>
                <Typography.Paragraph style={{ marginTop: 8, whiteSpace: 'pre-wrap' }}>
                  {detail?.content || '-'}
                </Typography.Paragraph>
              </div>
            </Space>
          </Drawer>
        </>
      ) : (
        <Card className="player-empty-card player-notifications-empty-state">
          <div className="player-notifications-empty-grid" />
          <div className="player-notifications-empty-orb player-notifications-empty-orb-a" />
          <div className="player-notifications-empty-orb player-notifications-empty-orb-b" />

          <div className="player-notifications-empty-badge">Message Hub</div>

          <div className="player-notifications-empty-icon">
            <BellOutlined />
          </div>

          <Typography.Title level={2} className="player-notifications-empty-title">
            通知中心暂时很安静
          </Typography.Title>

          <Typography.Paragraph className="player-notifications-empty-description">
            当前还没有新的系统消息、赛事提醒或队伍通知。等有新的动态到来后，这里会集中展示未读状态、通知类型和详细内容。
          </Typography.Paragraph>

          <div className="player-notifications-empty-features">
            <div className="player-notifications-empty-feature">
              <NotificationOutlined className="player-notifications-empty-feature-icon" />
              <div>
                <strong>系统动态汇总</strong>
                <span>比赛、平台和个人提醒都会统一收进这里</span>
              </div>
            </div>
            <div className="player-notifications-empty-feature">
              <TeamOutlined className="player-notifications-empty-feature-icon" />
              <div>
                <strong>队伍消息同步</strong>
                <span>组队变更、邀请和协作提醒会第一时间出现</span>
              </div>
            </div>
            <div className="player-notifications-empty-feature">
              <MessageOutlined className="player-notifications-empty-feature-icon" />
              <div>
                <strong>详情即点即看</strong>
                <span>后续可直接展开查看通知正文与时间线</span>
              </div>
            </div>
          </div>

          <div className="player-notifications-empty-actions">
            <Button type="primary" size="large">
              等待新通知
            </Button>
          </div>
        </Card>
      )}
    </PlayerPage>
  );
}

function typeLabel(value?: string) {
  if (value === 'competition') return '赛事';
  if (value === 'team') return '队伍';
  if (value === 'personal') return '个人';
  return '系统';
}

function formatTime(value?: string) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}
