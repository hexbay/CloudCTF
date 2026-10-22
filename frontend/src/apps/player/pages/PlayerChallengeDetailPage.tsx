import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  CopyOutlined,
  FlagOutlined,
  RocketOutlined,
  TeamOutlined,
  TrophyOutlined,
} from '@ant-design/icons';
import { App, Alert, Button, Card, Col, Descriptions, Divider, Input, Modal, Progress, Row, Space, Spin, Statistic, Tag, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { playerApi, type PlayerChallengeDetail } from '@shared/api/player';
import { isPlayerLoggedIn } from '@shared/auth';
import { difficultyColor, difficultyText } from './PlayerDashboardPage';
import { PlayerPage } from './PlayerPage';

function formatSeconds(value: number) {
  const hours = Math.floor(value / 3600).toString().padStart(2, '0');
  const minutes = Math.floor((value % 3600) / 60).toString().padStart(2, '0');
  const seconds = Math.floor(value % 60).toString().padStart(2, '0');
  return `${hours}:${minutes}:${seconds}`;
}

export function PlayerChallengeDetailPage() {
  const { message } = App.useApp();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [modal, contextHolder] = Modal.useModal();
  const challengeId = Number(id);
  const [data, setData] = useState<PlayerChallengeDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [starting, setStarting] = useState(false);
  const [extending, setExtending] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [copyingTarget, setCopyingTarget] = useState<string | null>(null);
  const [flag, setFlag] = useState('');
  const [remaining, setRemaining] = useState(0);
  const [hintModalOpen, setHintModalOpen] = useState(false);
  const [activeHintIndex, setActiveHintIndex] = useState<number | null>(null);
  const [purchasingHint, setPurchasingHint] = useState(false);
  const [relatedChallenges, setRelatedChallenges] = useState<PlayerChallengeDetail[]>([]);

  const load = async () => {
    if (!challengeId) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const detail = await playerApi.challengeDetail(challengeId);
      setData(detail);
      const related = await loadRelatedChallenges(detail);
      setRelatedChallenges(related);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载题目失败');
      setData(null);
      setRelatedChallenges([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [challengeId]);

  useEffect(() => {
    const expires = data?.container_info?.expires ?? 0;
    setRemaining(expires);
    if (!expires) {
      return;
    }
    const timer = window.setInterval(() => {
      setRemaining((value) => {
        if (value <= 1) {
          window.clearInterval(timer);
          void load();
          return 0;
        }
        return value - 1;
      });
    }, 1000);
    return () => window.clearInterval(timer);
  }, [data?.container_info?.expires]);

  const difficulty = useMemo(() => difficultyText(data?.difficulty ?? 1), [data?.difficulty]);

  const ensureAuth = () => {
    if (!isPlayerLoggedIn()) {
      window.dispatchEvent(new CustomEvent('show-player-auth', { detail: 'login' }));
      message.info('请先登录后再进行操作');
      return false;
    }
    return true;
  };

  const submitFlag = async () => {
    if (!ensureAuth() || !data) {
      return;
    }
    if (!flag.trim()) {
      message.warning('请输入 Flag');
      return;
    }
    setSubmitting(true);
    try {
      const result = await playerApi.submitFlag(data.id, flag.trim());
      if (result.correct) {
        message.success(result.is_first_blood ? '\u63d0\u4ea4\u6210\u529f\uff0c\u62ff\u4e0b\u4e00\u8840\uff0c\u83b7\u5f97 ' + String(result.points ?? data.points ?? 0) + ' \u5206' : '\u63d0\u4ea4\u6210\u529f\uff0c\u83b7\u5f97 ' + String(result.points ?? data.points ?? 0) + ' \u5206');
        setFlag('');
        await load();
      } else if (result.already_solved) {
        message.warning(result.message ?? '\u5df2\u7ecf\u89e3\u51b3\u8fc7\u4e86');
      } else {
        message.error(result.message ?? 'Flag \u9519\u8bef');
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '提交失败');
    } finally {
      setSubmitting(false);
    }
  };

  const startContainer = async () => {
    if (!ensureAuth() || !data) {
      return;
    }
    setStarting(true);
    try {
      await playerApi.startChallenge(data.id);
      message.success('环境已启动');
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '启动失败');
    } finally {
      setStarting(false);
    }
  };

  const extendContainer = async () => {
    if (!ensureAuth() || !data) {
      return;
    }
    setExtending(true);
    try {
      await playerApi.extendChallenge(data.id);
      message.success('已延长 30 分钟');
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '延长失败');
    } finally {
      setExtending(false);
    }
  };

  const stopContainer = async () => {
    if (!ensureAuth() || !data) {
      return;
    }
    modal.confirm({
      title: '停止环境',
      content: '停止后当前实例会被销毁。',
      okType: 'danger',
      onOk: async () => {
        setStopping(true);
        try {
          await playerApi.stopChallenge(data.id);
          message.success('环境已停止');
          await load();
        } catch (error) {
          message.error(error instanceof Error ? error.message : '停止失败');
        } finally {
          setStopping(false);
        }
      },
    });
  };

  const purchaseHint = async () => {
    if (activeHintIndex === null) {
      message.warning('请选择要解锁的提示');
      return;
    }
    if (!ensureAuth() || !data) {
      return;
    }
    setPurchasingHint(true);
    try {
      await playerApi.purchaseHint(data.id, activeHintIndex);
      message.success('提示已解锁');
      setHintModalOpen(false);
      setActiveHintIndex(null);
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '解锁失败');
    } finally {
      setPurchasingHint(false);
    }
  };

  const currentHint = activeHintIndex === null ? null : data?.hints?.find((hint) => hint.index === activeHintIndex) ?? null;

  const revealHint = (hintIndex: number) => {
    if (!ensureAuth()) {
      return;
    }
    setActiveHintIndex(hintIndex);
    setHintModalOpen(true);
  };

  const copyTarget = async (value: string) => {
    setCopyingTarget(value);
    try {
      await navigator.clipboard.writeText(value);
      message.success('已复制');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '复制失败');
    } finally {
      setCopyingTarget(null);
    }
  };

  const loadRelatedChallenges = async (detail: PlayerChallengeDetail) => {
    try {
      const result = await playerApi.challenges({
        current: 1,
        pageSize: 20,
        category: detail.category,
      });
      const rows = result.data ?? [];
      return rows.filter((item) => item.id !== detail.id).slice(0, 3) as PlayerChallengeDetail[];
    } catch {
      return [];
    }
  };

  return (
    <PlayerPage icon={<FlagOutlined />} title={data?.name ?? '题目详情'} subtitle={undefined} extra={undefined}>
      {contextHolder}
      {loading ? <div className="player-loading"><Spin size="large" /></div> : null}
      {!loading && !data ? <Alert type="error" message="题目不存在或无法访问" showIcon /> : null}
      {!loading && data ? (
        <div className="challenge-detail-layout">
          <div className="challenge-detail-breadcrumb">
            <Button type="text" className="challenge-detail-back" icon={<ArrowLeftOutlined />} onClick={() => navigate('/challenges')}>
              返回题库
            </Button>
            <span aria-hidden="true">/</span>
            <span>题目详情</span>
          </div>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={16}>
              <Card className="challenge-detail-hero">
                <div className="challenge-detail-accent" />
                <div className="challenge-detail-hero-top">
                  <Space wrap size={[8, 8]} className="challenge-detail-tags">
                    <Tag className={`challenge-difficulty-tag difficulty-${difficulty}`}>{difficulty}</Tag>
                    <Tag className={`challenge-category-tag category-${String(data.category).toLowerCase()}`}>{data.category}</Tag>
                    {data.solved ? <Tag className="challenge-state-tag is-solved" icon={<CheckCircleOutlined />}>已解决</Tag> : null}
                    {data.is_dynamic ? <Tag className="challenge-state-tag is-dynamic">动态环境</Tag> : <Tag className="challenge-state-tag">静态题目</Tag>}
                  </Space>
                  <div className="challenge-detail-points" aria-label={`${data.points} 积分`}>
                    <TrophyOutlined />
                    <div>
                      <span>题目积分</span>
                      <strong>{data.points}</strong>
                    </div>
                  </div>
                </div>
                <div className="challenge-detail-intro">
                  <span className="challenge-detail-kicker">{data.category} Challenge</span>
                  <Typography.Title level={2} className="challenge-detail-heading">{data.name}</Typography.Title>
                  <div className="challenge-detail-description" dangerouslySetInnerHTML={{ __html: (data.description || '暂无描述').replace(/\n/g, '<br />') }} />
                </div>
                <div className="challenge-detail-sections">
                  {data.attachments?.length ? (
                    <div className="challenge-detail-section">
                      <Divider orientation="left">附件</Divider>
                      <Space direction="vertical" style={{ width: '100%' }}>
                        {data.attachments.map((item, index) => {
                          const href = item.file_path || item.url || '#';
                          return (
                            <Button key={`${href}-${index}`} href={href} target="_blank">
                              {item.filename || `附件 ${index + 1}`}
                            </Button>
                          );
                        })}
                      </Space>
                    </div>
                  ) : null}

                  {data.hints?.length ? (
                    <div className="challenge-detail-section">
                      <Divider orientation="left">提示</Divider>
                      <div className="challenge-hint-grid">
                        {data.hints.map((hint) => (
                          <Card key={hint.index} size="small" className="challenge-hint-card">
                            <Space direction="vertical" size={10} style={{ width: '100%' }}>
                              <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                                <Typography.Text strong>{`提示 ${hint.index + 1}`}</Typography.Text>
                                <Tag color={hint.purchased ? 'success' : 'gold'}>{hint.purchased ? '已解锁' : `${hint.cost} 积分`}</Tag>
                              </Space>
                              {hint.purchased ? (
                                <Typography.Paragraph className="challenge-hint-content">
                                  {hint.content || '暂无内容'}
                                </Typography.Paragraph>
                              ) : (
                                <Button type="dashed" block onClick={() => revealHint(hint.index)}>
                                  {`解锁提示 ${hint.index + 1}`}
                                </Button>
                              )}
                            </Space>
                          </Card>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  {data.is_dynamic ? (
                    <div className="challenge-detail-section">
                      <Divider orientation="left">动态环境</Divider>
                      {data.container_info ? (
                        <Space direction="vertical" style={{ width: '100%' }} size="large" className="challenge-runtime-panel">
                          <Space wrap>
                            <Tag color="success">运行中</Tag>
                            <Tag>{data.container_info.status}</Tag>
                            {data.container_info.name ? <Tag>{data.container_info.name}</Tag> : null}
                          </Space>
                          <Descriptions column={1} size="small">
                            <Descriptions.Item label="状态">{data.container_info.status}</Descriptions.Item>
                            <Descriptions.Item label="容器 ID">{data.container_info.container_id}</Descriptions.Item>
                            <Descriptions.Item label="剩余时间">{formatSeconds(remaining)}</Descriptions.Item>
                          </Descriptions>
                          <div className="challenge-target-list">
                            {data.container_info.targets?.map((target) => (
                              <div key={target.url} className="challenge-target-item">
                                <Space wrap>
                                  <Tag color="blue">{target.protocol.toUpperCase()}</Tag>
                                  <Typography.Link href={target.url} target="_blank">{target.url}</Typography.Link>
                                </Space>
                                <Button icon={<CopyOutlined />} loading={copyingTarget === target.url} onClick={() => void copyTarget(target.url)} />
                              </div>
                            ))}
                          </div>
                          <Progress percent={Math.max(0, Math.min(100, Math.round(((remaining || 0) / (data.container_timeout || 3600)) * 100)))} showInfo={false} status={remaining < 300 ? 'exception' : 'active'} />
                          <Space className="challenge-runtime-actions">
                            <Button type="primary" icon={<RocketOutlined />} loading={extending} onClick={() => void extendContainer()}>
                              延长时间
                            </Button>
                            <Button danger loading={stopping} onClick={() => void stopContainer()}>
                              停止环境
                            </Button>
                          </Space>
                        </Space>
                      ) : (
                        <Space direction="vertical" size="middle" className="challenge-runtime-empty">
                          <Typography.Text type="secondary">此题目需要先启动动态环境后才能访问。</Typography.Text>
                          <Button type="primary" icon={<RocketOutlined />} loading={starting} onClick={() => void startContainer()}>
                            启动环境
                          </Button>
                        </Space>
                      )}
                    </div>
                  ) : null}

                  <div className="challenge-detail-section challenge-submit-panel">
                    <Divider orientation="left">提交 Flag</Divider>
                    <Input.Search
                      className="challenge-flag-search"
                      value={flag}
                      onChange={(event) => setFlag(event.target.value)}
                      onSearch={() => void submitFlag()}
                      enterButton="提交"
                      loading={submitting}
                      placeholder="flag{...}"
                      disabled={Boolean(data.solved)}
                    />
                  </div>
                </div>
              </Card>
            </Col>

            <Col xs={24} lg={8} className="challenge-sidebar">
              <Card title="挑战数据" className="challenge-detail-card challenge-stats-card">
                <div className="challenge-stat-primary">
                  <Statistic title="解题人数" value={data.solves_count ?? 0} prefix={<TeamOutlined />} />
                </div>
                <div className="challenge-first-blood">
                  <span>一血记录</span>
                  <strong>{data.bloods?.[0]?.username ?? '暂无'}</strong>
                </div>
              </Card>

              <Card title="相似题目" className="challenge-detail-card challenge-related-card">
                {relatedChallenges.length ? (
                  <div className="challenge-related-list">
                    {relatedChallenges.map((item) => (
                      <button key={item.id} type="button" className="challenge-related-item" onClick={() => navigate(`/challenges/${item.id}`)}>
                        <div className="challenge-related-head">
                          <Typography.Text>{item.name}</Typography.Text>
                          <Tag color={difficultyColor(Number(item.difficulty))}>{difficultyText(item.difficulty)}</Tag>
                        </div>
                        <div className="challenge-related-meta">{`${item.category} · ${item.points}分`}</div>
                      </button>
                    ))}
                  </div>
                ) : (
                  <div className="challenge-related-empty">暂无相似题目</div>
                )}
              </Card>
            </Col>
          </Row>
        </div>
      ) : null}

      <Modal
        title="解锁提示"
        open={hintModalOpen}
        okText="确认解锁"
        cancelText="取消"
        confirmLoading={purchasingHint}
        onOk={() => void purchaseHint()}
        onCancel={() => {
          setHintModalOpen(false);
          setActiveHintIndex(null);
        }}
      >
        <Typography.Paragraph>
          解锁这个提示需要消耗 <Typography.Text strong>{currentHint?.cost ?? 0}</Typography.Text> 积分。
        </Typography.Paragraph>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          确认要继续吗？
        </Typography.Paragraph>
      </Modal>
    </PlayerPage>
  );
}
