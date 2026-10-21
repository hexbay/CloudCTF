import {
  CalendarOutlined,
  ClockCircleOutlined,
  DownOutlined,
  FileTextOutlined,
  InfoCircleOutlined,
  LogoutOutlined,
  NotificationOutlined,
  PlayCircleOutlined,
  RocketOutlined,
  TeamOutlined,
  TrophyOutlined,
  UserOutlined,
  FlagOutlined,
} from "@ant-design/icons";
import { App, Avatar, Button, Card, Dropdown, Spin, Typography } from "antd";
import { Form } from "antd";
import {
  createContext,
  useContext,
  useEffect,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from "react";
import {
  Navigate,
  Outlet,
  useLocation,
  useNavigate,
  useOutletContext,
  useSearchParams,
} from "react-router-dom";
import { getStoredPlayerUser, isPlayerLoggedIn } from "@shared/auth";
import {
  playerApi,
  type PlayerAnnouncement,
  type PlayerChallenge,
  type PlayerCompetition,
  type PlayerExam,
  type PlayerRank,
} from "@shared/api/player";
import { fallbackCompetitions } from "./playerData";
import {
  formatDate,
  getCountdown,
  getRegistrationState,
  normalizeExamAnswers,
} from "./competition-arena/TheoryCompetitionExamPanel";

type ArenaTabKey =
  | "overview"
  | "register"
  | "challenges"
  | "scoreboard"
  | "writeup"
  | "announcements";

type ArenaContextValue = {
  competitionID: number;
  competition: PlayerCompetition | null;
  challenges: PlayerChallenge[];
  ranks: PlayerRank[];
  announcements: PlayerAnnouncement[];
  activeChallenge: PlayerChallenge | null;
  setActiveChallenge: Dispatch<SetStateAction<PlayerChallenge | null>>;
  exam: PlayerExam | null;
  examResult: {
    correct?: number;
    total?: number;
    score?: number;
    total_score?: number;
  } | null;
  challengesLoading: boolean;
  examLoading: boolean;
  scoreboardLoading: boolean;
  examSubmitting: boolean;
  submittingID: number | null;
  registering: boolean;
  loading: boolean;
  reloading: boolean;
  loggedIn: boolean;
  storedUser: ReturnType<typeof getStoredPlayerUser>;
  canEnter: boolean;
  registrationState: ReturnType<typeof getRegistrationState>;
  isTheoryCompetition: boolean;
  challengeMode: "ctf" | "awd" | "mixed" | "theory";
  countdown: { value: string; label: string };
  overviewDescription: string;
  participantModeText: string;
  activeTab: ArenaTabKey;
  examForm: ReturnType<typeof Form.useForm>[0];
  flagForm: ReturnType<typeof Form.useForm<{ flag: string }>>[0];
  examAnswers: Record<string, unknown>;
  loadArena: (withIndicator?: boolean) => Promise<void>;
  loadChallenges: () => Promise<void>;
  loadExam: () => Promise<void>;
  loadScoreboard: () => Promise<void>;
  submitFlag: (challengeId: number, values: { flag: string }) => Promise<void>;
  submitExam: (values: Record<string, unknown>) => Promise<void>;
  handleRegister: () => Promise<void>;
  handleLeaveArena: () => void;
  navigateToTab: (tab: ArenaTabKey) => void;
};

const ArenaContext = createContext<ArenaContextValue | null>(null);

const statusNames: Record<string, string> = {
  ongoing: "进行中",
  upcoming: "即将开始",
  finished: "已结束",
  draft: "草稿",
  archived: "已归档",
};

const typeNames: Record<string, string> = {
  ctf: "CTF",
  awd: "AWD",
  theory: "理论赛",
  mixed: "混合赛",
};

function cleanNullableText(value: unknown) {
  const text = String(value ?? "").trim();
  if (!text || text === "<nil>" || text.toLowerCase() === "null") return "";
  return text;
}

function getArenaTabs(
  type?: string,
): Array<{ key: ArenaTabKey; label: string; icon: ReactNode }> {
  const mode = getArenaChallengeMode(type);
  const tabs: Array<{ key: ArenaTabKey; label: string; icon: ReactNode }> = [
    { key: "overview", label: "比赛概述", icon: <InfoCircleOutlined /> },
    {
      key: "challenges",
      label: mode === "theory" ? "试卷" : "赛题",
      icon: mode === "theory" ? <FileTextOutlined /> : <FlagOutlined />,
    },
    { key: "scoreboard", label: "排行榜", icon: <TrophyOutlined /> },
  ];
  tabs.push({ key: "announcements", label: "公告", icon: <NotificationOutlined /> });
  return tabs;
}

function getArenaChallengeMode(type?: string) {
  const normalized = String(type ?? "").toLowerCase();
  if (normalized === "theory") return "theory";
  if (normalized === "awd") return "awd";
  if (normalized === "mixed") return "mixed";
  return "ctf";
}

function getParticipantModeText(mode?: string, maxTeamMembers?: number | null) {
  const normalized = String(mode ?? "").toLowerCase();
  if (normalized === "individual" || normalized === "solo") {
    return "个人参赛";
  }
  if (normalized === "team") {
    return maxTeamMembers
      ? `支持队伍参赛，最多 ${maxTeamMembers} 人`
      : "支持队伍参赛";
  }
  if (normalized === "mixed") {
    return maxTeamMembers
      ? `支持个人或队伍参赛，队伍最多 ${maxTeamMembers} 人`
      : "支持个人或队伍参赛";
  }
  return maxTeamMembers
    ? `支持组队参赛，最多 ${maxTeamMembers} 人`
    : "支持个人或队伍参赛";
}

function getArenaTab(pathname: string): ArenaTabKey {
  const tab = pathname.split("/")[2] as ArenaTabKey | undefined;
  if (
    tab === "register" ||
    tab === "challenges" ||
    tab === "scoreboard" ||
    tab === "writeup" ||
    tab === "announcements"
  ) {
    return tab;
  }
  return "overview";
}

function useArenaContextValue(): ArenaContextValue {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const location = useLocation();
  const [params] = useSearchParams();
  const competitionID = Number(params.get("id") || 0);
  const activeTab = getArenaTab(location.pathname);
  const [loading, setLoading] = useState(true);
  const [reloading, setReloading] = useState(false);
  const [competition, setCompetition] = useState<PlayerCompetition | null>(
    null,
  );
  const [challenges, setChallenges] = useState<PlayerChallenge[]>([]);
  const [ranks, setRanks] = useState<PlayerRank[]>([]);
  const [announcements, setAnnouncements] = useState<PlayerAnnouncement[]>([]);
  const [activeChallenge, setActiveChallenge] =
    useState<PlayerChallenge | null>(null);
  const [exam, setExam] = useState<PlayerExam | null>(null);
  const [challengesLoading, setChallengesLoading] = useState(false);
  const [examLoading, setExamLoading] = useState(false);
  const [scoreboardLoading, setScoreboardLoading] = useState(false);
  const [examSubmitting, setExamSubmitting] = useState(false);
  const [examResult, setExamResult] = useState<{
    correct?: number;
    total?: number;
    score?: number;
    total_score?: number;
  } | null>(null);
  const [submittingID, setSubmittingID] = useState<number | null>(null);
  const [registering, setRegistering] = useState(false);
  const [loggedIn, setLoggedIn] = useState(isPlayerLoggedIn());
  const [storedUser, setStoredUser] = useState(getStoredPlayerUser());
  const [examForm] = Form.useForm();
  const [flagForm] = Form.useForm<{ flag: string }>();
  const examAnswers =
    (Form.useWatch([], examForm) as Record<string, unknown> | undefined) ?? {};

  useEffect(() => {
    const syncAuth = () => {
      setLoggedIn(isPlayerLoggedIn());
      setStoredUser(getStoredPlayerUser());
    };
    window.addEventListener("player-auth-changed", syncAuth);
    return () => window.removeEventListener("player-auth-changed", syncAuth);
  }, []);

  async function loadArena(withIndicator = false) {
    if (!competitionID) {
      setLoading(false);
      return;
    }

    if (withIndicator) {
      setReloading(true);
    } else {
      setLoading(true);
    }

    try {
      const [competitionRes, announcementRes] = await Promise.all([
        playerApi.competitionDetail(competitionID),
        playerApi.competitionAnnouncements(competitionID, {
          current: 1,
          pageSize: 20,
        }),
      ]);

      const selectedCompetition = competitionRes?.id
        ? competitionRes
        : (fallbackCompetitions.find((item) => item.id === competitionID) ??
          null);

      setCompetition(selectedCompetition);
      setAnnouncements(announcementRes.data ?? []);

      // 理论赛试卷在进入赛题 tab 时按需加载，不在 overview 预请求
      if (String(selectedCompetition?.type).toLowerCase() !== "theory") {
        setExam(null);
        examForm.resetFields();
        setExamResult(null);
      }

      if (withIndicator) {
        message.success("比赛数据已刷新");
      }
    } catch (error) {
      const selectedCompetition =
        fallbackCompetitions.find((item) => item.id === competitionID) ??
        fallbackCompetitions[0] ??
        null;
      setCompetition(selectedCompetition);
      setChallenges([]);
      setRanks([]);
      setAnnouncements([]);
      setExam(null);
      examForm.resetFields();
      setExamResult(null);
      setActiveChallenge(null);
      if (withIndicator) {
        message.error(
          error instanceof Error ? error.message : "刷新比赛数据失败",
        );
      }
    } finally {
      setLoading(false);
      setReloading(false);
    }
  }

  useEffect(() => {
    void loadArena();
  }, [competitionID]);

  async function loadChallenges() {
    if (!competitionID || challengesLoading) return;
    setChallengesLoading(true);
    try {
      const res = await playerApi.challenges({
        current: 1,
        pageSize: 100,
        competition_id: competitionID || undefined,
      });
      const rows = res.data?.length ? res.data : [];
      setChallenges(rows);
      setActiveChallenge((current) => {
        if (!rows.length) return null;
        return rows.find((item) => item.id === current?.id) ?? rows[0];
      });
    } catch {
      setChallenges([]);
      setActiveChallenge(null);
    } finally {
      setChallengesLoading(false);
    }
  }

  async function loadExam() {
    if (!competitionID || examLoading) return;
    setExamLoading(true);
    try {
      const examRes = await playerApi.competitionExam(competitionID);
      if (examRes?.exam_id) {
        setExam(examRes);
        examForm.resetFields();
        setExamResult(examRes.submission ?? null);
      } else {
        setExam(null);
        examForm.resetFields();
        setExamResult(null);
      }
    } catch {
      setExam(null);
      examForm.resetFields();
      setExamResult(null);
    } finally {
      setExamLoading(false);
    }
  }

  async function loadScoreboard() {
    if (!competitionID || scoreboardLoading) return;
    setScoreboardLoading(true);
    try {
      const rankRes = await playerApi.competitionScoreboard(competitionID);
      setRanks(rankRes.data ?? []);
    } catch {
      setRanks([]);
    } finally {
      setScoreboardLoading(false);
    }
  }

  async function submitFlag(challengeId: number, values: { flag: string }) {
    const targetChallenge =
      challenges.find((item) => item.id === challengeId) ?? null;
    if (!targetChallenge) {
      message.warning("请先选择一道赛题");
      return;
    }
    if (!values.flag?.trim()) {
      message.warning("请输入 Flag");
      return;
    }
    setSubmittingID(challengeId);
    try {
      const result = await playerApi.submitFlag(
        challengeId,
        values.flag.trim(),
      );
      if (result.correct) {
        message.success(
          result.is_first_blood
            ? `提交成功，拿下一血，获得 ${String(result.points ?? targetChallenge.points ?? 0)} 分`
            : `提交成功，获得 ${String(result.points ?? targetChallenge.points ?? 0)} 分`,
        );
        flagForm.resetFields();
        setChallenges((current) =>
          current.map((item) =>
            item.id === challengeId ? { ...item, is_solved: true } : item,
          ),
        );
        setActiveChallenge((current) =>
          current?.id === challengeId
            ? { ...current, is_solved: true }
            : current,
        );
        if (competitionID) {
          const rankRes = await playerApi.competitionScoreboard(competitionID);
          setRanks(rankRes.data ?? []);
        }
      } else if (result.already_solved) {
        message.warning(result.message || "已经解决过了");
      } else {
        message.error(result.message || "Flag 不正确");
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "提交失败");
    } finally {
      setSubmittingID(null);
    }
  }

  async function handleRegister() {
    if (!competition) return;
    if (!loggedIn) {
      window.dispatchEvent(
        new CustomEvent("show-player-auth", { detail: "login" }),
      );
      message.info("请先登录后再参与比赛");
      return;
    }
    setRegistering(true);
    try {
      const result = await playerApi.registerCompetition(
        competition.id,
        storedUser?.team_id ? { team_id: storedUser.team_id } : undefined,
      );
      message.success(result.message || "报名成功");
      setCompetition((current) =>
        current ? { ...current, is_participant: true } : current,
      );
      navigate(`/competition-arena/challenges?id=${competition.id}`);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "报名失败");
    } finally {
      setRegistering(false);
    }
  }

  function handleLeaveArena() {
    navigate("/");
  }

  async function submitExam(values: Record<string, unknown>) {
    if (!exam || !competitionID) {
      message.warning("当前没有可提交的试卷");
      return;
    }
    setExamSubmitting(true);
    try {
      const answers = normalizeExamAnswers(values, exam.questions);
      const result = await playerApi.submitCompetitionExam(competitionID, {
        exam_id: exam.exam_id,
        answers,
      });
      setExamResult(result);
      await loadArena();
      message.success("试卷已提交");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "交卷失败");
    } finally {
      setExamSubmitting(false);
    }
  }

  const canEnter = Boolean(
    competition?.is_participant ||
    (competition?.is_public && !competition?.registration_open),
  );
  const registrationState = getRegistrationState(competition, loggedIn);
  const countdown = getCountdown(competition?.end_time);
  const isTheoryCompetition =
    String(competition?.type ?? "").toLowerCase() === "theory";
  const challengeMode = getArenaChallengeMode(competition?.type);
  const overviewDescription =
    cleanNullableText(competition?.description) ||
    "这里展示比赛介绍、时间安排、参赛方式、规则说明，以及主办与支持信息。";
  const participantModeText = getParticipantModeText(
    competition?.participant_mode,
    competition?.max_team_members,
  );

  return {
    competitionID,
    competition,
    challenges,
    ranks,
    announcements,
    activeChallenge,
    setActiveChallenge,
    exam,
    examResult,
    challengesLoading,
    examLoading,
    scoreboardLoading,
    examSubmitting,
    submittingID,
    registering,
    loading,
    reloading,
    loggedIn,
    storedUser,
    canEnter,
    registrationState,
    isTheoryCompetition,
    challengeMode,
    countdown,
    overviewDescription,
    participantModeText,
    activeTab,
    examForm,
    flagForm,
    examAnswers,
    loadArena,
    loadChallenges,
    loadExam,
    loadScoreboard,
    submitFlag,
    submitExam,
    handleRegister,
    handleLeaveArena,
    navigateToTab: (tab) =>
      navigate(`/competition-arena/${tab}?id=${competitionID}`),
  };
}

export function useCompetitionArena() {
  const context =
    useContext(ArenaContext) ?? useOutletContext<ArenaContextValue>();
  if (!context) {
    throw new Error("Competition arena context is not available");
  }
  return context;
}

export function ArenaOverviewPage() {
  const { competition, overviewDescription, participantModeText } =
    useCompetitionArena();
  if (!competition) return null;

  return (
    <div className="arena-overview-layout">
      <div className="arena-overview-main">
        <section className="arena-overview-section">
          <div className="arena-overview-eyebrow">Competition Overview</div>
          <Typography.Title level={2}>比赛介绍</Typography.Title>
          <Typography.Paragraph>{overviewDescription}</Typography.Paragraph>
        </section>

        <section className="arena-overview-section">
          <Typography.Title level={3}>基本信息</Typography.Title>
          <div className="arena-overview-table">
            <div className="arena-overview-row">
              <span>开始时间</span>
              <strong>{formatDate(competition.start_time)}</strong>
            </div>
            <div className="arena-overview-row">
              <span>结束时间</span>
              <strong>{formatDate(competition.end_time)}</strong>
            </div>
            <div className="arena-overview-row">
              <span>比赛类型</span>
              <strong>
                {typeNames[String(competition.type ?? "ctf").toLowerCase()] ??
                  "CTF"}
              </strong>
            </div>
            <div className="arena-overview-row">
              <span>比赛状态</span>
              <strong>
                {statusNames[competition.status] ?? competition.status}
              </strong>
            </div>
            <div className="arena-overview-row">
              <span>开放方式</span>
              <strong>
                {competition.is_public === false ? "私有比赛" : "公开比赛"}
              </strong>
            </div>
            <div className="arena-overview-row">
              <span>参赛模式</span>
              <strong>{participantModeText}</strong>
            </div>
          </div>
        </section>

        <section className="arena-overview-section">
          <Typography.Title level={3}>参赛规则</Typography.Title>
          <div className="arena-overview-copy-list">
            <div className="arena-overview-copy-item">
              <i />
              <Typography.Paragraph>
                请在比赛开放时间内完成参赛与作答，最终以平台记录和赛事公告为准。
              </Typography.Paragraph>
            </div>
            <div className="arena-overview-copy-item">
              <i />
              <Typography.Paragraph>
                禁止攻击非授权目标、破坏平台稳定性或通过违规方式影响比赛公平性。
              </Typography.Paragraph>
            </div>
            <div className="arena-overview-copy-item">
              <i />
              <Typography.Paragraph>
                若比赛支持动态环境，请按需启动、延长和销毁，避免占用无效资源。
              </Typography.Paragraph>
            </div>
          </div>
        </section>

        <section className="arena-overview-section is-last">
          <Typography.Title level={3}>主办方与支持</Typography.Title>
          <div className="arena-overview-support">
            <div>
              <span>主办方</span>
              <strong>CloudCTF 平台</strong>
              <p>负责赛事发布、平台运行与活动承载。</p>
            </div>
            <div>
              <span>赛事组织</span>
              <strong>比赛组委会</strong>
              <p>负责赛程安排、公告说明与规则执行。</p>
            </div>
            <div>
              <span>技术支持</span>
              <strong>技术支持团队</strong>
              <p>负责环境保障、赛题运营与比赛过程支持。</p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

export function PlayerCompetitionArenaPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const contextValue = useArenaContextValue();
  const {
    competition,
    competitionID,
    loading,
    activeTab,
    canEnter,
    registering,
    countdown,
    participantModeText,
    handleRegister,
    navigateToTab,
    loggedIn,
    storedUser,
    handleLeaveArena,
    isTheoryCompetition,
    overviewDescription,
  } = contextValue;
  const navigationTabs = getArenaTabs(competition?.type);

  if (!competitionID) {
    return <Navigate to="/competition" replace />;
  }

  const showArenaEntryAction = !(canEnter && activeTab === "challenges");
  const showTheoryCompletedState =
    contextValue.isTheoryCompetition &&
    activeTab === "challenges" &&
    Boolean(contextValue.examResult);
  const hideSharedHero =
    activeTab === "scoreboard" || activeTab === "challenges";
  const userMenuItems = [
    {
      key: "account",
      icon: <UserOutlined />,
      label: "个人中心",
      onClick: () => navigate("/account"),
    },
    {
      key: "team",
      icon: <TeamOutlined />,
      label: "我的队伍",
      onClick: () => navigate("/team"),
    },
    {
      type: "divider" as const,
    },
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: "退出比赛区",
      danger: true,
      onClick: handleLeaveArena,
    },
  ];

  return (
    <ArenaContext.Provider value={contextValue}>
      <div className="arena-shell">
        <header className="arena-topbar">
          <div
            className="arena-brand"
            onClick={() => contextValue.navigateToTab("overview")}
          >
            <img src="/logo.svg" alt="CloudCTF" className="arena-brand-logo" />
            <div>
              <strong>{competition?.name || "CloudCTF 比赛中心"}</strong>
            </div>
          </div>

          <nav className="arena-nav">
            {navigationTabs.map((tab) => (
              <button
                key={tab.key}
                type="button"
                className={
                  activeTab === tab.key
                    ? "arena-nav-item is-active"
                    : "arena-nav-item"
                }
                onClick={() => navigateToTab(tab.key)}
              >
                {tab.icon}
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>

          <div className="arena-userbar">
            {!isTheoryCompetition ? (
              <Button
                className="arena-top-action"
                icon={<FileTextOutlined />}
                onClick={() => navigateToTab("writeup")}
              >
                Writeup
              </Button>
            ) : null}
            {loggedIn ? (
              <>
                <Dropdown
                  menu={{ items: userMenuItems }}
                  trigger={["click"]}
                  placement="bottomRight"
                >
                  <button
                    type="button"
                    className="arena-userchip arena-userchip-trigger"
                  >
                    <Avatar size={30} icon={<UserOutlined />} />
                    <span>{storedUser?.username || "当前用户"}</span>
                    <DownOutlined className="arena-userchip-arrow" />
                  </button>
                </Dropdown>
              </>
            ) : (
              <Button
                type="primary"
                onClick={() =>
                  window.dispatchEvent(
                    new CustomEvent("show-player-auth", { detail: "login" }),
                  )
                }
              >
                登录后参赛
              </Button>
            )}
          </div>
        </header>

        <main className="arena-main">
          <Spin spinning={loading}>
            {competition ? (
              <div className="arena-screen">
                {!showTheoryCompletedState && !hideSharedHero ? (
                  <Card className="arena-screen-hero">
                    <div className="arena-screen-hero-grid" />
                    <div className="arena-screen-hero-stars" />
                    <div className="arena-screen-hero-orb arena-screen-hero-orb-a" />
                    <div className="arena-screen-hero-orb arena-screen-hero-orb-b" />
                    <div className="arena-screen-hero-layout">
                      <div className="arena-screen-hero-copy">
                        <Typography.Title level={2}>
                          {competition.name}
                        </Typography.Title>
                        <Typography.Paragraph>
                          {overviewDescription}
                        </Typography.Paragraph>
                      </div>
                      <div className="arena-screen-hero-side">
                        <div>
                          <CalendarOutlined />
                          开始时间：{formatDate(competition.start_time)}
                        </div>
                        <div>
                          <ClockCircleOutlined />
                          结束时间：{formatDate(competition.end_time)}
                        </div>
                        <div>
                          <TeamOutlined />
                          {participantModeText}
                        </div>
                        <div className="arena-screen-hero-deadline">
                          <span>{countdown.label}</span>
                          <strong>{countdown.value}</strong>
                        </div>
                        {showArenaEntryAction ? (
                          <Button
                            type="primary"
                            icon={
                              canEnter ? (
                                <PlayCircleOutlined />
                              ) : (
                                <RocketOutlined />
                              )
                            }
                            loading={registering}
                            onClick={
                              canEnter
                                ? () => navigateToTab("challenges")
                                : () => void handleRegister()
                            }
                          >
                            {canEnter ? "进入比赛" : "立即报名"}
                          </Button>
                        ) : (
                          <div className="arena-screen-hero-status">
                            <PlayCircleOutlined />
                            <span>已进入比赛区，当前可直接答题</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </Card>
                ) : null}

                <Outlet context={contextValue} />
              </div>
            ) : (
              <Card>
                <Typography.Text>暂无可进入的比赛</Typography.Text>
              </Card>
            )}
          </Spin>
        </main>
      </div>
    </ArenaContext.Provider>
  );
}
