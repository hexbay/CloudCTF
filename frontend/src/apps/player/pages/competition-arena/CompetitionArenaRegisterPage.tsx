import { Alert, Button, Card, Col, Row } from "antd";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

export function CompetitionArenaRegisterPage() {
  const {
    competition,
    registrationState,
    loggedIn,
    storedUser,
    canEnter,
    registering,
    handleRegister,
    navigateToTab,
  } = useCompetitionArena();

  if (!competition) return null;

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} lg={16}>
        <Card className="player-panel-card arena-overview-card" title="报名说明">
          <div className="arena-register-stack">
            <Alert
              type={registrationState.type}
              showIcon
              message={registrationState.title}
              description={
                competition.is_public ? "公开赛可直接进入，无需额外报名。" : registrationState.description
              }
            />
            <div className="arena-info-grid">
              <div>
                <span>报名状态</span>
                <strong>
                  {competition.is_public && !competition.registration_open ? "公开赛免报名" : competition.is_participant ? "已报名" : "未报名"}
                </strong>
              </div>
              <div>
                <span>登录状态</span>
                <strong>{loggedIn ? "已登录" : "未登录"}</strong>
              </div>
              <div>
                <span>参赛身份</span>
                <strong>{storedUser?.team_name || "个人参赛 / 默认队伍"}</strong>
              </div>
              <div>
                <span>报名入口</span>
                <strong>
                  {competition.is_public
                    ? "无需报名"
                    : competition.registration_open === false
                      ? "暂未开放"
                      : "当前可用"}
                </strong>
              </div>
            </div>
          </div>
        </Card>
      </Col>
      <Col xs={24} lg={8}>
        <Card className="player-panel-card arena-overview-card" title="比赛入口">
          <div className="arena-register-actions">
            <Button
              type="primary"
              block
              loading={registering}
              onClick={canEnter ? () => navigateToTab("challenges") : () => void handleRegister()}
            >
              {canEnter ? "进入比赛" : "立即报名"}
            </Button>
          </div>
        </Card>
      </Col>
    </Row>
  );
}
