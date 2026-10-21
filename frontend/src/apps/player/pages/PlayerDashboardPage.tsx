import {
  ArrowRightOutlined,
  ClockCircleOutlined,
  FireOutlined,
  FlagOutlined,
  SafetyCertificateOutlined,
  RocketOutlined,
  TeamOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import { Button, Card, Col, List, Row, Space, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  playerApi,
  type PlayerChallenge,
  type PlayerCompetition,
  type PlayerStats,
  type PublicSystemSettings,
} from "@shared/api/player";

const statusText: Record<string, string> = {
  ongoing: "进行中",
  upcoming: "即将开始",
  finished: "已结束",
};

const categoryColor: Record<string, string> = {
  web: "orange",
  crypto: "purple",
  reverse: "cyan",
  pwn: "red",
  misc: "green",
  forensics: "blue",
  mobile: "magenta",
  blockchain: "gold",
};

export function PlayerDashboardPage() {
  const navigate = useNavigate();
  const [competitions, setCompetitions] = useState<PlayerCompetition[]>([]);
  const [challenges, setChallenges] = useState<PlayerChallenge[]>([]);
  const [stats, setStats] = useState<PlayerStats>({});
  const [loggedIn, setLoggedIn] = useState(
    Boolean(localStorage.getItem("cloudctf_token")),
  );
  const [publicSettings, setPublicSettings] = useState<PublicSystemSettings>(
    {},
  );

  useEffect(() => {
    const syncAuth = () =>
      setLoggedIn(Boolean(localStorage.getItem("cloudctf_token")));
    window.addEventListener("player-auth-changed", syncAuth);
    return () => window.removeEventListener("player-auth-changed", syncAuth);
  }, []);

  useEffect(() => {
    void Promise.all([
      playerApi
        .competitions({ current: 1, pageSize: 4, status: "ongoing" })
        .then((res) => setCompetitions(res.data ?? []))
        .catch(() => setCompetitions([])),
      playerApi
        .challenges({ current: 1, pageSize: 6 })
        .then((res) => setChallenges(res.data ?? []))
        .catch(() => setChallenges([])),
      playerApi
        .publicSettings()
        .then((res) => setPublicSettings(res.data ?? {}))
        .catch(() => setPublicSettings({})),
      loggedIn
        ? playerApi
            .stats()
            .then(setStats)
            .catch(() => setStats({}))
        : Promise.resolve(),
    ]);
  }, [loggedIn]);

  const platformName =
    publicSettings.site_name?.trim() ||
    publicSettings.site_title?.trim() ||
    "CloudCTF";
  const platformDescription =
    publicSettings.site_description?.trim() ||
    "参加比赛、训练题目，并持续提升你的安全实战能力。";

  const featuredChallenges = useMemo(
    () => challenges.slice(0, 3),
    [challenges],
  );
  const statCards = useMemo(
    () => [
      {
        key: "points",
        title: "我的积分",
        value: stats.total_points ?? 0,
        accentClass: "metric-cyan",
        iconClass: "is-cyan",
        icon: <TrophyOutlined />,
        badge: "POINTS",
      },
      {
        key: "solved",
        title: "已解题目",
        value: stats.solved_challenges ?? 0,
        accentClass: "metric-green",
        iconClass: "is-green",
        icon: <FlagOutlined />,
        badge: "SOLVED",
      },
      {
        key: "rank",
        title: "当前排名",
        value: `# ${stats.rank ?? 0}`,
        accentClass: "metric-amber",
        iconClass: "is-amber",
        icon: <FireOutlined />,
        badge: "RANK",
      },
      {
        key: "team",
        title: "我的团队",
        value: stats.team_name ?? "暂无团队",
        accentClass: "metric-violet",
        iconClass: "is-violet",
        icon: <SafetyCertificateOutlined />,
        badge: "SQUAD",
      },
    ],
    [stats.rank, stats.solved_challenges, stats.team_name, stats.total_points],
  );
  const nextActions = useMemo(
    () => [
      {
        title: loggedIn ? "继续刷题" : "先登录账号",
        desc: loggedIn
          ? "从推荐题目开始，把今天的训练节奏跑起来。"
          : "登录后可以保存进度、加入团队并参与比赛。",
        action: loggedIn ? "前往练习区" : "打开登录",
        onClick: () =>
          loggedIn
            ? navigate("/challenges")
            : window.dispatchEvent(
                new CustomEvent("show-player-auth", { detail: "login" }),
              ),
      },
      {
        title: stats.team_name ? "查看团队状态" : "创建或加入团队",
        desc: stats.team_name
          ? `你当前所在团队：${stats.team_name}，可以继续邀请成员或准备报名比赛。`
          : "团队赛前先把队伍建好，邀请码加入会更快。",
        action: "进入团队中心",
        onClick: () => navigate("/team"),
      },
      {
        title: "浏览当前赛事",
        desc: "看看有哪些比赛已经开始，或哪些即将开放报名。",
        action: "查看赛事",
        onClick: () => navigate("/competition"),
      },
    ],
    [loggedIn, navigate, stats.team_name],
  );

  return (
    <div className="player-dashboard">
      <Card className="player-hero-card">
        <div className="player-hero-content">
          <div className="player-hero-copy">
            <Typography.Title level={2}>
              {loggedIn
                ? `欢迎回到 ${platformName}`
                : `欢迎来到 ${platformName}`}
            </Typography.Title>
            <Typography.Paragraph>
              {loggedIn
                ? "继续你的训练、参赛和解题进度。"
                : platformDescription}
            </Typography.Paragraph>
            <Space>
              <Button
                type="primary"
                size="large"
                icon={<RocketOutlined />}
                onClick={() => navigate("/challenges")}
              >
                开始挑战
              </Button>
              {!loggedIn ? (
                <Button
                  size="large"
                  onClick={() =>
                    window.dispatchEvent(
                      new CustomEvent("show-player-auth", {
                        detail: "register",
                      }),
                    )
                  }
                >
                  注册账号
                </Button>
              ) : null}
            </Space>
          </div>
          <div className="player-hero-visual" aria-hidden="true">
            <div className="radar-ring" />
            <div className="radar-sweep" />
            <div className="radar-ping radar-ping-1" />
            <div className="radar-ping radar-ping-2" />
            <div className="radar-core">FLAG</div>
          </div>
        </div>
      </Card>

      {loggedIn ? (
        <Row gutter={[16, 16]} className="player-stat-row">
          {statCards.map((item, index) => (
            <Col key={item.key} xs={24} sm={12} md={6}>
              <Card
                className={`metric-card metric-panel ${item.accentClass} dashboard-animate-card dashboard-stat-card`}
              >
                <div
                  className={`metric-card-glow ${item.iconClass}`}
                  aria-hidden="true"
                />
                <div className="metric-card-grid">
                  <div className="metric-card-copy">
                    <span className="metric-card-label">{item.title}</span>
                    <strong className="metric-card-value">{item.value}</strong>
                    <span className="metric-card-badge">{item.badge}</span>
                  </div>
                  <div
                    className={`metric-card-icon ${item.iconClass}`}
                    aria-hidden="true"
                  >
                    {item.icon}
                  </div>
                </div>
                <div className="metric-card-orbit" aria-hidden="true">
                  <span className="metric-card-orbit-line" />
                  <span className="metric-card-orbit-dot" />
                </div>
                <div
                  className="metric-card-scanline"
                  style={{ animationDelay: `${index * 0.12}s` }}
                  aria-hidden="true"
                />
              </Card>
            </Col>
          ))}
        </Row>
      ) : null}

      <Row gutter={[16, 16]} className="dashboard-lower-grid">
        <Col xs={24} xl={14}>
          <Card
            className="player-list-card dashboard-section-card dashboard-action-card dashboard-animate-card"
            bordered={false}
          >
            <div className="dashboard-section-head">
              <div>
                <span>今日主线</span>
                <h3>接下来做什么</h3>
              </div>
              <Button type="link" onClick={() => navigate("/team")}>
                团队中心
              </Button>
            </div>
            <div className="dashboard-action-list">
              {nextActions.map((item, index) => (
                <button
                  key={item.title}
                  type="button"
                  className="dashboard-action-item"
                  onClick={item.onClick}
                >
                  <div className="dashboard-action-index">0{index + 1}</div>
                  <div className="dashboard-action-copy">
                    <strong>{item.title}</strong>
                    <span>{item.desc}</span>
                  </div>
                  <div className="dashboard-action-cta">
                    {item.action}
                    <ArrowRightOutlined />
                  </div>
                </button>
              ))}
            </div>
          </Card>
        </Col>

        <Col xs={24} xl={10}>
          <Card
            className="player-list-card dashboard-section-card dashboard-side-highlight dashboard-animate-card dashboard-animate-delay-1"
            bordered={false}
          >
            <div className="dashboard-section-head">
              <div>
                <span>训练建议</span>
                <h3>推荐题目</h3>
              </div>
              <Link to="/challenges">查看全部</Link>
            </div>
            {featuredChallenges.length ? (
              <div className="dashboard-challenge-stack">
                {featuredChallenges.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className="dashboard-challenge-spotlight"
                    onClick={() => navigate(`/challenges/${item.id}`)}
                  >
                    <div className="dashboard-challenge-head">
                      <strong>{item.name}</strong>
                      <Tag color={categoryColor[item.category] ?? "default"}>
                        {item.category}
                      </Tag>
                    </div>
                    <p>
                      {item.description ||
                        "从这道题开始热身，快速进入今天的解题状态。"}
                    </p>
                    <div className="dashboard-challenge-meta">
                      <span>{difficultyText(item.difficulty)}</span>
                      <span>{item.points} 分</span>
                    </div>
                  </button>
                ))}
              </div>
            ) : (
              <DashboardEmpty
                title="暂无题目数据"
                description="当前还没有适合展示在首页的练习题，可以先前往题库浏览全部内容。"
              />
            )}
          </Card>
        </Col>

        <Col xs={24} xl={14}>
          <Card
            title="正在进行的比赛"
            extra={<Link to="/competition">查看全部</Link>}
            className="player-list-card dashboard-section-card dashboard-animate-card dashboard-animate-delay-2"
          >
            {competitions.length ? (
              <div className="player-dashboard-competition-list">
                {competitions.map((item) => (
                  <div
                    key={item.id}
                    className="player-dashboard-competition-item"
                  >
                    <div className="player-dashboard-competition-main">
                      <div className="player-dashboard-competition-topline">
                        <strong>{item.name}</strong>
                        <span
                          className={`player-dashboard-status-pill is-${item.status}`}
                        >
                          {statusText[item.status] ?? item.status}
                        </span>
                      </div>
                      <div className="player-dashboard-competition-progress">
                        <div className="player-dashboard-competition-progress-track">
                          <span
                            className={`player-dashboard-competition-progress-fill is-${item.status}`}
                          />
                        </div>
                      </div>
                      <div className="player-dashboard-competition-meta">
                        <span>
                          <ClockCircleOutlined />{" "}
                          {renderTimeInfo(item.start_time, item.end_time)}
                        </span>
                        {item.max_team_members ? (
                          <span>最多 {item.max_team_members} 人组队</span>
                        ) : null}
                      </div>
                    </div>
                    <Button
                      type="link"
                      onClick={() =>
                        navigate(`/competition-arena/overview?id=${item.id}`)
                      }
                    >
                      进入比赛
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <DashboardEmpty
                title="暂无比赛数据"
                description="目前还没有可展示的进行中赛事，你可以先去练习区提升状态，或者稍后再看。"
              />
            )}
          </Card>
        </Col>

        <Col xs={24} xl={10}>
          <Card
            title="练习题速览"
            extra={<Link to="/challenges">前往题库</Link>}
            className="player-list-card dashboard-section-card accent-amber dashboard-animate-card dashboard-animate-delay-3"
          >
            {challenges.length ? (
              <List
                dataSource={challenges.slice(0, 5)}
                renderItem={(item) => (
                  <List.Item
                    actions={[
                      <Button
                        key="solve"
                        type="link"
                        onClick={() => navigate(`/challenges/${item.id}`)}
                      >
                        去解题
                      </Button>,
                    ]}
                  >
                    <List.Item.Meta
                      title={
                        <Space size={[8, 8]} wrap>
                          <span>{item.name}</span>
                          <Tag color={categoryColor[item.category]}>
                            {item.category}
                          </Tag>
                        </Space>
                      }
                      description={`${difficultyText(item.difficulty)} · ${item.points} 分`}
                    />
                  </List.Item>
                )}
              />
            ) : (
              <DashboardEmpty
                title="暂无题目数据"
                description="题库内容还没有同步到首页速览区，稍后回来看看，或者直接前往完整题库。"
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}

function DashboardEmpty({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="dashboard-empty-state">
      <div className="dashboard-empty-visual" aria-hidden="true">
        <div className="dashboard-empty-orb" />
        <div className="dashboard-empty-card-shape" />
        <div className="dashboard-empty-signal" />
      </div>
      <strong>{title}</strong>
      <span>{description}</span>
    </div>
  );
}

export function difficultyText(value: number | string) {
  const n =
    typeof value === "number"
      ? value
      : value === "easy"
        ? 1
        : value === "medium"
          ? 3
          : 4;
  if (n <= 2) return "简单";
  if (n <= 3) return "中等";
  return "困难";
}

export function difficultyColor(value: number | string) {
  const text = difficultyText(value);
  return text === "简单" ? "success" : text === "中等" ? "warning" : "error";
}

function renderTimeInfo(startTime: string, endTime: string) {
  const start = new Date(startTime).getTime();
  const end = new Date(endTime).getTime();
  const now = Date.now();
  if (now < start)
    return `${Math.max(0, Math.floor((start - now) / 86400000))} 天后开始`;
  if (now < end)
    return `剩余 ${Math.max(0, Math.floor((end - now) / 86400000))} 天`;
  return "已结束";
}
