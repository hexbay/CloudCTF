import {
  CopyOutlined,
  LoginOutlined,
  PlusOutlined,
  SearchOutlined,
  TeamOutlined,
  TrophyOutlined,
  UserOutlined,
} from "@ant-design/icons";
import {
  Avatar,
  Button,
  Card,
  Col,
  Divider,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Row,
  Space,
  Spin,
  Statistic,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import {
  playerApi,
  type PlayerTeam,
  type PlayerUser,
} from "@shared/api/player";
import { PlayerPage } from "./PlayerPage";

const { TextArea } = Input;
const MIN_JOINING_FEEDBACK_MS = 240;

const competitionStatusText: Record<string, string> = {
  ongoing: "进行中",
  upcoming: "即将开始",
  finished: "已结束",
};

const competitionStatusColor: Record<string, string> = {
  ongoing: "green",
  upcoming: "blue",
  finished: "default",
};

function updateStoredUser(patch: Partial<PlayerUser>) {
  const raw = localStorage.getItem("cloudctf_user");
  const current = raw ? (JSON.parse(raw) as PlayerUser) : ({} as PlayerUser);
  const next = { ...current, ...patch };
  localStorage.setItem("cloudctf_user", JSON.stringify(next));
  window.dispatchEvent(
    new CustomEvent("player-auth-changed", {
      detail: { loggedIn: true, user: next },
    }),
  );
}

function wait(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

export function PlayerTeamPage() {
  const [team, setTeam] = useState<PlayerTeam | null>(null);
  const [loadingTeam, setLoadingTeam] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [joinOpen, setJoinOpen] = useState(false);
  const [teamSearch, setTeamSearch] = useState("");
  const [teamOptions, setTeamOptions] = useState<PlayerTeam[]>([]);
  const [quickJoinCode, setQuickJoinCode] = useState("");
  const [joiningCode, setJoiningCode] = useState("");
  const [joining, setJoining] = useState(false);
  const [creating, setCreating] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const [loadingTeams, setLoadingTeams] = useState(false);
  const [form] = Form.useForm();
  const loggedIn = Boolean(localStorage.getItem("cloudctf_token"));

  useEffect(() => {
    if (!loggedIn) {
      setTeam(null);
      setLoadingTeam(false);
      return;
    }

    setLoadingTeam(true);
    playerApi
      .team()
      .then((value) => {
        if (value && value.id) {
          setTeam(value);
          updateStoredUser({ team_id: value.id, team_name: value.name });
        } else {
          setTeam(null);
          updateStoredUser({ team_id: 0, team_name: "" });
        }
      })
      .catch(() => setTeam(null))
      .finally(() => setLoadingTeam(false));
  }, [loggedIn]);

  useEffect(() => {
    if (!joinOpen) return;
    setLoadingTeams(true);
    playerApi
      .teams({ current: 1, pageSize: 20, name: teamSearch || undefined })
      .then((res) => setTeamOptions(res.data ?? []))
      .catch(() => {
        setTeamOptions([]);
        message.error("加载队伍列表失败");
      })
      .finally(() => setLoadingTeams(false));
  }, [joinOpen, teamSearch]);

  const memberCount = team?.member_count ?? team?.members?.length ?? 0;
  const owner = useMemo(
    () => team?.members?.find((item) => item.is_owner),
    [team],
  );
  const joinedCompetitions = team?.competitions ?? [];

  const createTeam = async () => {
    try {
      setCreating(true);
      const values = await form.validateFields();
      const created = await playerApi.createTeam(values);
      const latestTeam = created?.id ? created : await playerApi.team();
      setTeam(latestTeam);
      updateStoredUser({ team_id: latestTeam.id, team_name: latestTeam.name });
      setCreateOpen(false);
      form.resetFields();
      message.success("团队创建成功");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建团队失败");
    } finally {
      setCreating(false);
    }
  };

  const leaveTeam = async () => {
    try {
      setLeaving(true);
      await playerApi.leaveTeam();
      setTeam(null);
      updateStoredUser({ team_id: 0, team_name: "" });
      message.success("已退出团队");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "退出团队失败");
    } finally {
      setLeaving(false);
    }
  };

  const joinTeam = async (invitationCode: string) => {
    const normalizedCode = invitationCode.trim();
    if (!normalizedCode) {
      message.warning("请输入邀请码后再加入");
      return;
    }
    setJoining(true);
    setJoiningCode(normalizedCode);
    try {
      await Promise.all([
        playerApi.joinTeam({ invitation_code: normalizedCode }),
        wait(MIN_JOINING_FEEDBACK_MS),
      ]);
      const latestTeam = await playerApi.team();
      setTeam(latestTeam);
      updateStoredUser({ team_id: latestTeam.id, team_name: latestTeam.name });
      setQuickJoinCode("");
      setJoinOpen(false);
      message.success("已加入团队");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加入团队失败");
    } finally {
      setJoining(false);
      setJoiningCode("");
    }
  };

  const joinTeamById = async (teamId: number) => {
    if (!teamId) {
      message.warning("请选择要加入的团队");
      return;
    }
    setJoining(true);
    setJoiningCode(`team:${teamId}`);
    try {
      await Promise.all([
        playerApi.joinTeam({ team_id: teamId }),
        wait(MIN_JOINING_FEEDBACK_MS),
      ]);
      const latestTeam = await playerApi.team();
      setTeam(latestTeam);
      updateStoredUser({ team_id: latestTeam.id, team_name: latestTeam.name });
      setQuickJoinCode("");
      setJoinOpen(false);
      message.success("已加入团队");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加入团队失败");
    } finally {
      setJoining(false);
      setJoiningCode("");
    }
  };

  const copyInviteCode = async () => {
    if (!team?.invitation_code) return;
    try {
      await navigator.clipboard.writeText(team.invitation_code);
      message.success("邀请码已复制");
    } catch {
      message.error("复制失败，请手动复制");
    }
  };

  if (!loggedIn) {
    return (
      <PlayerPage
        icon={<TeamOutlined />}
        title="我的团队"
        subtitle="登录后即可创建团队、邀请队友或加入已有队伍，一起参加比赛。"
        showTitle
      >
        <Card className="player-empty-card team-login-card">
          <TeamOutlined className="team-login-icon" />
          <Typography.Title level={3}>先登录，再组建你的战队</Typography.Title>
          <Typography.Paragraph>
            你可以先注册账号，再创建团队或通过邀请码加入队友的战队。
          </Typography.Paragraph>
          <Space wrap>
            <Button
              type="primary"
              size="large"
              onClick={() =>
                window.dispatchEvent(
                  new CustomEvent("show-player-auth", { detail: "register" }),
                )
              }
            >
              注册账号
            </Button>
            <Button
              size="large"
              icon={<LoginOutlined />}
              onClick={() =>
                window.dispatchEvent(
                  new CustomEvent("show-player-auth", { detail: "login" }),
                )
              }
            >
              已有账号，去登录
            </Button>
          </Space>
        </Card>
      </PlayerPage>
    );
  }

  return (
    <PlayerPage
      icon={<TeamOutlined />}
      title="我的团队"
      subtitle="从这里完成创建、加入、邀请和查看队伍状态的全部操作。"
      showTitle
      extra={
        team ? (
          <Space wrap>
            <Button
              icon={<CopyOutlined />}
              onClick={() => void copyInviteCode()}
              disabled={!team.invitation_code}
            >
              复制邀请码
            </Button>
            <Button icon={<SearchOutlined />} onClick={() => setJoinOpen(true)}>
              查找其他队伍
            </Button>
            <Button danger loading={leaving} onClick={() => void leaveTeam()}>
              退出团队
            </Button>
          </Space>
        ) : (
          <Space wrap>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateOpen(true)}
            >
              创建团队
            </Button>
            <Button icon={<SearchOutlined />} onClick={() => setJoinOpen(true)}>
              浏览并加入
            </Button>
          </Space>
        )
      }
    >
      <Spin spinning={loadingTeam}>
        {team ? (
          <div className="team-page-stack">
            <Card className="player-panel-card team-hero-card">
              <div className="team-hero-layout">
                <div className="team-hero-main">
                  <div className="team-hero-head">
                    <Avatar
                      size={72}
                      icon={<TeamOutlined />}
                      className="team-hero-avatar"
                    />
                    <div className="team-hero-copy">
                      <Space size={[8, 8]} wrap className="team-hero-badges">
                        <Typography.Title level={2}>
                          {team.name}
                        </Typography.Title>
                        <Tag color="cyan">团队空间</Tag>
                        {owner ? (
                          <Tag color="gold">队长：{owner.username}</Tag>
                        ) : null}
                      </Space>
                      <Typography.Paragraph className="team-hero-description">
                        {team.description ||
                          "这支队伍还没有填写介绍，可以补充团队方向、擅长题型和协作方式。"}
                      </Typography.Paragraph>
                    </div>
                  </div>
                  <div className="team-info-grid">
                    <div>
                      <span>创建时间</span>
                      <strong>{formatDate(team.created_at)}</strong>
                    </div>
                    <div>
                      <span>成员数量</span>
                      <strong>{memberCount} 人</strong>
                    </div>
                    <div>
                      <span>参与赛事</span>
                      <strong>{joinedCompetitions.length} 场</strong>
                    </div>
                    <div>
                      <span>邀请码</span>
                      <strong>{team.invitation_code || "暂未生成"}</strong>
                    </div>
                  </div>
                </div>
                <div className="team-hero-side">
                  <div className="team-invite-card">
                    <span>团队邀请码</span>
                    <strong>{team.invitation_code || "暂未生成邀请码"}</strong>
                    <Typography.Paragraph>
                      新成员最直接的加入方式就是注册后输入邀请码，所以这里应该始终清晰可见。
                    </Typography.Paragraph>
                    <Button
                      type="primary"
                      icon={<CopyOutlined />}
                      onClick={() => void copyInviteCode()}
                      disabled={!team.invitation_code}
                    >
                      复制邀请码
                    </Button>
                  </div>
                </div>
              </div>
            </Card>

            <Row gutter={[16, 16]}>
              <Col xs={24} md={8}>
                <Card className="player-panel-card metric-card metric-cyan team-metric-card">
                  <Statistic
                    title="当前成员"
                    value={memberCount}
                    prefix={<UserOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card className="player-panel-card metric-card metric-green team-metric-card">
                  <Statistic
                    title="报名赛事"
                    value={joinedCompetitions.length}
                    prefix={<TrophyOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={24} md={8}>
                <Card className="player-panel-card metric-card metric-amber team-metric-card">
                  <Statistic
                    title="邀请码状态"
                    value={team.invitation_code ? "已就绪" : "未生成"}
                  />
                </Card>
              </Col>
            </Row>

            <Row gutter={[16, 16]}>
              <Col xs={24} xl={14}>
                <Card
                  className="player-panel-card team-detail-card"
                  title={
                    <Space className="team-section-title">
                      <UserOutlined />
                      团队成员
                    </Space>
                  }
                >
                  <List
                    className="team-member-list"
                    dataSource={team.members ?? []}
                    locale={{ emptyText: "还没有成员数据" }}
                    renderItem={(item) => (
                      <List.Item className="team-member-item">
                        <List.Item.Meta
                          avatar={
                            <Avatar
                              icon={<UserOutlined />}
                              className="team-member-avatar"
                            />
                          }
                          title={
                            <Space
                              size={[8, 8]}
                              wrap
                              className="team-member-title"
                            >
                              <span className="team-member-name">
                                {item.full_name || item.username}
                              </span>
                              {item.full_name &&
                              item.full_name !== item.username ? (
                                <Typography.Text className="team-member-handle">
                                  @{item.username}
                                </Typography.Text>
                              ) : null}
                              {item.is_owner ? (
                                <Tag color="red">队长</Tag>
                              ) : (
                                <Tag>成员</Tag>
                              )}
                            </Space>
                          }
                          description={item.email || "暂未公开邮箱"}
                        />
                      </List.Item>
                    )}
                  />
                </Card>
              </Col>
              <Col xs={24} xl={10}>
                <Card
                  className="player-panel-card team-detail-card"
                  title={
                    <Space className="team-section-title">
                      <TrophyOutlined />
                      参赛状态
                    </Space>
                  }
                >
                  {joinedCompetitions.length ? (
                    <List
                      dataSource={joinedCompetitions}
                      renderItem={(item) => (
                        <List.Item className="team-competition-item">
                          <List.Item.Meta
                            avatar={<Avatar icon={<TrophyOutlined />} />}
                            title={item.name}
                            description={
                              <Tag
                                color={
                                  competitionStatusColor[item.status] ??
                                  "default"
                                }
                              >
                                {competitionStatusText[item.status] ??
                                  item.status}
                              </Tag>
                            }
                          />
                        </List.Item>
                      )}
                    />
                  ) : (
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description="当前团队还没有参与任何比赛"
                    />
                  )}
                  <Divider />
                  <Typography.Paragraph className="team-side-note">
                    下一步通常是邀请队友加入，然后再去赛事页统一报名团队赛。
                  </Typography.Paragraph>
                </Card>
              </Col>
            </Row>
          </div>
        ) : (
          <div className="team-page-stack">
            <Card className="player-panel-card team-empty-state">
              <div className="team-empty-hero">
                <div>
                  <Tag color="cyan">团队协作</Tag>
                  <Typography.Title level={2}>
                    你还没有加入任何团队
                  </Typography.Title>
                  <Typography.Paragraph>
                    注册完成后的第一步通常是二选一：自己创建团队，或者输入队友发来的邀请码直接加入。
                  </Typography.Paragraph>
                  <Space wrap>
                    <Button
                      type="primary"
                      size="large"
                      icon={<PlusOutlined />}
                      onClick={() => setCreateOpen(true)}
                    >
                      我来创建团队
                    </Button>
                    <Button
                      size="large"
                      icon={<SearchOutlined />}
                      onClick={() => setJoinOpen(true)}
                    >
                      我有邀请码 / 去查找
                    </Button>
                  </Space>
                </div>
                <div className="team-empty-metrics">
                  <div>
                    <strong>1. 注册并登录</strong>
                    <span>新用户先完成账号注册，系统会自动进入已登录状态</span>
                  </div>
                  <div>
                    <strong>2. 创建或加入团队</strong>
                    <span>可以自己建队，也可以输入邀请码直接加入已有队伍</span>
                  </div>
                  <div>
                    <strong>3. 再去报名比赛</strong>
                    <span>
                      团队建立完成后，再去赛事页进行团队赛报名会更顺畅
                    </span>
                  </div>
                </div>
              </div>
            </Card>

            <Card className="player-panel-card">
              <Space direction="vertical" size={16} style={{ width: "100%" }}>
                <div>
                  <Typography.Title level={4}>
                    通过邀请码快速加入
                  </Typography.Title>
                  <Typography.Paragraph className="team-side-note">
                    如果队长已经把邀请码发给你，直接在这里输入会比先搜索更快。
                  </Typography.Paragraph>
                </div>
                <Space.Compact style={{ width: "100%" }}>
                  <Input
                    value={quickJoinCode}
                    placeholder="输入团队邀请码，例如 TEAM-123456"
                    onChange={(event) => setQuickJoinCode(event.target.value)}
                    onPressEnter={() => void joinTeam(quickJoinCode)}
                  />
                  <Button
                    type="primary"
                    loading={joining && joiningCode === quickJoinCode.trim()}
                    onClick={() => void joinTeam(quickJoinCode)}
                  >
                    立即加入
                  </Button>
                </Space.Compact>
              </Space>
            </Card>
          </div>
        )}
      </Spin>

      <Modal
        title="创建新团队"
        className="player-team-modal"
        open={createOpen}
        onOk={() => void createTeam()}
        confirmLoading={creating}
        onCancel={() => setCreateOpen(false)}
        okText="创建团队"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="player-team-form">
          <Form.Item
            name="name"
            label="团队名称"
            rules={[{ required: true, message: "请输入团队名称" }]}
          >
            <Input placeholder="例如：CloudCTF Red Team" maxLength={32} />
          </Form.Item>
          <Form.Item name="description" label="团队介绍">
            <TextArea
              rows={4}
              maxLength={200}
              placeholder="可以介绍团队方向、擅长题型、协作风格等"
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="查找并加入团队"
        className="player-team-modal"
        open={joinOpen}
        footer={null}
        onCancel={() => setJoinOpen(false)}
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <Input.Search
            placeholder="输入团队名称搜索"
            enterButton
            value={teamSearch}
            loading={loadingTeams}
            onChange={(event) => setTeamSearch(event.target.value)}
          />
          <Space.Compact style={{ width: "100%" }}>
            <Input
              value={quickJoinCode}
              placeholder="或者直接输入邀请码加入"
              onChange={(event) => setQuickJoinCode(event.target.value)}
              onPressEnter={() => void joinTeam(quickJoinCode)}
            />
            <Button
              type="primary"
              loading={joining && joiningCode === quickJoinCode.trim()}
              onClick={() => void joinTeam(quickJoinCode)}
            >
              用邀请码加入
            </Button>
          </Space.Compact>
        </Space>
        <List
          className="team-search-list"
          loading={loadingTeams}
          dataSource={teamOptions}
          locale={{ emptyText: "暂无可加入的团队" }}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button
                  key="join"
                  type="primary"
                  size="small"
                  loading={joining && joiningCode === `team:${item.id}`}
                  onClick={() => void joinTeamById(item.id)}
                >
                  加入团队
                </Button>,
              ]}
            >
              <List.Item.Meta
                avatar={<Avatar icon={<TeamOutlined />} />}
                title={
                  <Space size={[8, 8]} wrap>
                    <span>{item.name}</span>
                    <Tag>
                      {item.member_count ?? item.members?.length ?? 0} 人
                    </Tag>
                  </Space>
                }
                description={
                  item.invitation_code
                    ? `邀请码：${item.invitation_code}${item.description ? ` · ${item.description}` : ""}`
                    : item.description || "暂无团队介绍"
                }
              />
            </List.Item>
          )}
        />
      </Modal>
    </PlayerPage>
  );
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleDateString();
}
