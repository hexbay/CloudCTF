import { ReloadOutlined, SendOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Form, Input, Row, Col, Typography } from "antd";
import { difficultyText } from "../PlayerDashboardPage";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

function getDifficultyTone(value: number | string | undefined) {
  const normalized = String(value ?? "");
  if (normalized === "1" || normalized === "easy") return "is-easy";
  if (normalized === "2" || normalized === "medium") return "is-medium";
  if (normalized === "3" || normalized === "hard") return "is-hard";
  return "is-expert";
}

export function AwdCompetitionArenaPage() {
  const {
    competition,
    challenges,
    activeChallenge,
    setActiveChallenge,
    loadArena,
    reloading,
    flagForm,
    submitFlag,
    submittingID,
  } = useCompetitionArena();

  if (!competition) return null;

  return (
    <div className="arena-mode-shell">
      <Card className="player-panel-card arena-awd-hero">
        <div className="arena-awd-hero-grid" />
        <div className="arena-awd-hero-content">
          <div>
            <div className="arena-mode-kicker">Attack Defense Arena</div>
            <Typography.Title level={3}>{competition.name} 攻防视图</Typography.Title>
            <Typography.Paragraph>
              AWD 页面强化目标切换与提交路径，适合快速进入攻防任务。
            </Typography.Paragraph>
          </div>
          <Button
            className="arena-action-button arena-action-button-secondary"
            icon={<ReloadOutlined />}
            loading={reloading}
            onClick={() => void loadArena(true)}
          >
            刷新态势
          </Button>
        </div>
      </Card>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={15}>
          <Card className="player-panel-card arena-awd-targets" title="目标服务">
            {challenges.length ? (
              <div className="arena-awd-target-grid">
                {challenges.map((challenge) => (
                  <button
                    key={challenge.id}
                    type="button"
                    className={activeChallenge?.id === challenge.id ? "arena-awd-target is-active" : "arena-awd-target"}
                    onClick={() => setActiveChallenge(challenge)}
                  >
                    <strong>{challenge.name}</strong>
                    <span>{challenge.category}</span>
                    <em>{challenge.points} 分</em>
                  </button>
                ))}
              </div>
            ) : (
              <Empty description="暂无攻防目标" />
            )}
          </Card>
        </Col>

        <Col xs={24} xl={9}>
          <Card className="player-panel-card arena-submit-panel" title="目标详情与提交">
            {activeChallenge ? (
              <>
                <div className="arena-detail-head">
                  <Typography.Title level={4}>{activeChallenge.name}</Typography.Title>
                </div>
                <Typography.Paragraph>{activeChallenge.description || "暂无目标说明"}</Typography.Paragraph>
                <div className="arena-detail-tags">
                  <span className="arena-chip arena-chip-category">{activeChallenge.category}</span>
                  <span className={`arena-chip arena-chip-difficulty ${getDifficultyTone(activeChallenge.difficulty)}`}>
                    {difficultyText(activeChallenge.difficulty)}
                  </span>
                  <span className="arena-chip arena-chip-score">{activeChallenge.points} 分</span>
                </div>
                <Form
                  form={flagForm}
                  layout="vertical"
                  onFinish={(values) => submitFlag(activeChallenge.id, values)}
                  className="arena-flag-form"
                >
                  <Form.Item name="flag" label="Flag / 结果" rules={[{ required: true, message: "请输入提交内容" }]}>
                    <Input placeholder="请输入提交内容" autoComplete="off" />
                  </Form.Item>
                  <Button
                    className="arena-action-button arena-action-button-primary"
                    type="primary"
                    htmlType="submit"
                    icon={<SendOutlined />}
                    loading={submittingID === activeChallenge.id}
                    block
                  >
                    提交结果
                  </Button>
                </Form>
              </>
            ) : (
              <Empty description="请选择一个目标服务" />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
