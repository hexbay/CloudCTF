import {
  CalendarOutlined,
  ClockCircleOutlined,
  FileTextOutlined,
  LoginOutlined,
  RocketOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Button, Typography } from "antd";
import { formatDate } from "./TheoryCompetitionExamPanel";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

export function CompetitionArenaRegistrationGate() {
  const {
    competition,
    loggedIn,
    storedUser,
    registering,
    registrationState,
    participantModeText,
    handleRegister,
    navigateToTab,
    challengeMode,
  } = useCompetitionArena();

  if (!competition) return null;

  const isTheory = challengeMode === "theory";

  return (
    <section className="arena-registration-gate">
      <div className="arena-registration-gate-grid" />
      <div className="arena-registration-gate-copy">
        <div className="arena-registration-kicker">
          {isTheory ? "Theory Exam Access" : "Competition Access"}
        </div>
        <Typography.Title level={2}>
          {isTheory ? "报名后进入理论试卷" : "报名后进入比赛赛题"}
        </Typography.Title>
        <Typography.Paragraph>
          {registrationState.description}
        </Typography.Paragraph>

        <div className="arena-registration-meta">
          <div>
            <CalendarOutlined />
            <span>开始时间</span>
            <strong>{formatDate(competition.start_time)}</strong>
          </div>
          <div>
            <ClockCircleOutlined />
            <span>结束时间</span>
            <strong>{formatDate(competition.end_time)}</strong>
          </div>
          <div>
            <TeamOutlined />
            <span>参赛模式</span>
            <strong>{participantModeText}</strong>
          </div>
          <div>
            <FileTextOutlined />
            <span>当前身份</span>
            <strong>{loggedIn ? storedUser?.team_name || storedUser?.username || "个人参赛" : "未登录"}</strong>
          </div>
        </div>
      </div>

      <aside className="arena-registration-action">
        <div className="arena-registration-status">
          <span>{registrationState.title}</span>
          <strong>{competition.is_participant ? "已报名" : "未报名"}</strong>
        </div>
        <Button
          type="primary"
          size="large"
          icon={loggedIn ? <RocketOutlined /> : <LoginOutlined />}
          loading={registering}
          onClick={() => void handleRegister()}
          block
        >
          {loggedIn ? "立即报名" : "登录后报名"}
        </Button>
        <Button type="link" onClick={() => navigateToTab("overview")}>
          返回比赛概述
        </Button>
      </aside>
    </section>
  );
}
