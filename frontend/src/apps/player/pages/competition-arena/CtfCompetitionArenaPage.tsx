import {
  CheckCircleFilled,
  CopyOutlined,
  CrownFilled,
  FlagOutlined,
  LinkOutlined,
  ReloadOutlined,
  RocketOutlined,
  SendOutlined,
} from "@ant-design/icons";
import { App, Button, Card, Empty, Form, Input, Modal, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
import { useEffect, useMemo, useState } from "react";
import { playerApi, type PlayerChallenge, type PlayerChallengeDetail } from "@shared/api/player";
import { difficultyText } from "../PlayerDashboardPage";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";

function getDifficultyTone(value: number | string | undefined) {
  const normalized = String(value ?? "");
  if (normalized === "1" || normalized === "easy") return "is-easy";
  if (normalized === "2" || normalized === "medium") return "is-medium";
  if (normalized === "3" || normalized === "hard") return "is-hard";
  return "is-expert";
}

function formatSeconds(value: number) {
  const hours = Math.floor(value / 3600).toString().padStart(2, "0");
  const minutes = Math.floor((value % 3600) / 60).toString().padStart(2, "0");
  const seconds = Math.floor(value % 60).toString().padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}

function ChallengeDetailModal({
  open,
  challenge,
  flagForm,
  onClose,
  onSubmitFlag,
  submittingID,
}: {
  open: boolean;
  challenge: PlayerChallenge | null;
  flagForm: FormInstance<{ flag: string }>;
  onClose: () => void;
  onSubmitFlag: (challengeId: number, values: { flag: string }) => Promise<void>;
  submittingID: number | null;
}) {
  const { message } = App.useApp();
  const [modal, contextHolder] = Modal.useModal();
  const [detail, setDetail] = useState<PlayerChallengeDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [starting, setStarting] = useState(false);
  const [extending, setExtending] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [copyingTarget, setCopyingTarget] = useState<string | null>(null);
  const [remaining, setRemaining] = useState(0);

  const loadDetail = async (challengeId: number) => {
    setLoading(true);
    try {
      setDetail(await playerApi.challengeDetail(challengeId));
    } catch (error) {
      setDetail(null);
      message.error(error instanceof Error ? error.message : "加载题目失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open && challenge?.id) void loadDetail(challenge.id);
    if (!open) {
      setDetail(null);
      flagForm.resetFields();
    }
  }, [open, challenge?.id, flagForm]);

  useEffect(() => {
    const expires = detail?.container_info?.expires ?? 0;
    setRemaining(expires);
    if (!expires) return;
    const timer = window.setInterval(() => {
      setRemaining((value) => {
        if (value <= 1) {
          window.clearInterval(timer);
          if (challenge?.id) void loadDetail(challenge.id);
          return 0;
        }
        return value - 1;
      });
    }, 1000);
    return () => window.clearInterval(timer);
  }, [detail?.container_info?.expires, challenge?.id]);

  const copyTarget = async (value: string) => {
    setCopyingTarget(value);
    try {
      await navigator.clipboard.writeText(value);
      message.success("已复制");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "复制失败");
    } finally {
      setCopyingTarget(null);
    }
  };

  const startContainer = async () => {
    if (!detail) return;
    setStarting(true);
    try {
      await playerApi.startChallenge(detail.id);
      message.success("环境已启动");
      await loadDetail(detail.id);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "启动失败");
    } finally {
      setStarting(false);
    }
  };

  const extendContainer = async () => {
    if (!detail) return;
    setExtending(true);
    try {
      await playerApi.extendChallenge(detail.id);
      message.success("环境已延长");
      await loadDetail(detail.id);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "延长失败");
    } finally {
      setExtending(false);
    }
  };

  const stopContainer = async () => {
    if (!detail) return;
    modal.confirm({
      title: "销毁环境",
      content: "销毁后当前环境会停止并失效，确认继续吗？",
      okType: "danger",
      okText: "确认销毁",
      cancelText: "取消",
      onOk: async () => {
        setStopping(true);
        try {
          await playerApi.stopChallenge(detail.id);
          message.success("环境已销毁");
          await loadDetail(detail.id);
        } catch (error) {
          message.error(error instanceof Error ? error.message : "销毁失败");
        } finally {
          setStopping(false);
        }
      },
    });
  };

  const displayChallenge = detail ?? challenge;

  return (
    <>
      {contextHolder}
      <Modal open={open && Boolean(challenge)} onCancel={onClose} footer={null} width={920} centered className="arena-detail-modal">
        {displayChallenge ? (
          <div className="arena-detail-modal-body">
            <div className="arena-detail-topline">
              <div className="arena-detail-tags">
                <span className="arena-chip arena-chip-category">{displayChallenge.category}</span>
                <span className={`arena-chip arena-chip-difficulty ${getDifficultyTone(displayChallenge.difficulty)}`}>
                  {difficultyText(displayChallenge.difficulty)}
                </span>
                <span className="arena-chip arena-chip-score">{displayChallenge.points} 分</span>
              </div>
              {displayChallenge.is_solved || detail?.solved ? (
                <div className="arena-detail-solved">
                  <CheckCircleFilled />
                  <span>已作答</span>
                </div>
              ) : null}
            </div>

            <div className="arena-detail-head arena-detail-head-rich">
              <div>
                <Typography.Title level={3}>{displayChallenge.name}</Typography.Title>
                <Typography.Paragraph>
                  {detail?.description || displayChallenge.description || "暂无题目描述"}
                </Typography.Paragraph>
              </div>
            </div>

            {(detail?.bloods?.length ?? 0) > 0 ? (
              <div className="arena-blood-board">
                {detail?.bloods?.slice(0, 3).map((blood, index) => (
                  <div key={`${blood.user_id}-${blood.rank}`} className="arena-blood-card">
                    <div className={`arena-blood-crown is-rank-${index + 1}`}>
                      <CrownFilled />
                    </div>
                    <strong>{blood.username}</strong>
                    <span>{index === 0 ? "一血" : index === 1 ? "二血" : "三血"}</span>
                  </div>
                ))}
              </div>
            ) : null}

            {detail?.attachments?.length ? (
              <div className="arena-detail-section-block">
                <div className="arena-detail-section-title">附件资料</div>
                <div className="arena-attachment-list">
                  {detail.attachments.map((item, index) => {
                    const href = item.file_path || item.url || "#";
                    return (
                      <a key={`${href}-${index}`} className="arena-attachment-item" href={href} target="_blank" rel="noreferrer">
                        <LinkOutlined />
                        <span>{item.filename || `附件 ${index + 1}`}</span>
                      </a>
                    );
                  })}
                </div>
              </div>
            ) : null}

            {detail?.is_dynamic ? (
              <div className="arena-detail-section-block">
                <div className="arena-detail-section-title">靶机环境</div>
                {detail.container_info ? (
                  <div className="arena-runtime-shell">
                    <div className="arena-runtime-countdown">
                      <span>剩余时间</span>
                      <strong>{formatSeconds(remaining)}</strong>
                      <em>{remaining <= 300 ? "环境即将到期，请及时延长" : "环境运行中，可按需延长或销毁"}</em>
                    </div>
                    <div className="arena-runtime-targets">
                      {detail.container_info.targets?.map((target) => (
                        <div key={target.url} className="arena-runtime-target">
                          <div>
                            <span>{target.protocol.toUpperCase()}</span>
                            <a href={target.url} target="_blank" rel="noreferrer">{target.url}</a>
                          </div>
                          <Button className="arena-action-button arena-action-button-secondary" icon={<CopyOutlined />} loading={copyingTarget === target.url} onClick={() => void copyTarget(target.url)}>
                            复制
                          </Button>
                        </div>
                      ))}
                    </div>
                    <div className="arena-runtime-actions">
                      <Button className="arena-action-button arena-action-button-primary" icon={<RocketOutlined />} loading={extending} onClick={() => void extendContainer()}>
                        延长环境
                      </Button>
                      <Button danger className="arena-action-button" loading={stopping} onClick={() => void stopContainer()}>
                        销毁环境
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="arena-runtime-empty">
                    <Typography.Paragraph>当前题目支持动态环境，启动后即可获得访问地址。</Typography.Paragraph>
                    <Button className="arena-action-button arena-action-button-primary" icon={<RocketOutlined />} loading={starting || loading} onClick={() => void startContainer()}>
                      启动环境
                    </Button>
                  </div>
                )}
              </div>
            ) : null}

            <Form form={flagForm} layout="vertical" onFinish={(values) => onSubmitFlag(displayChallenge.id, values)} className="arena-flag-form">
              <Form.Item name="flag" label="Flag" rules={[{ required: true, message: "请输入 Flag" }]}>
                <Input placeholder="flag{...}" autoComplete="off" />
              </Form.Item>
              <Button className="arena-action-button arena-action-button-primary" type="primary" htmlType="submit" icon={<SendOutlined />} loading={submittingID === displayChallenge.id} block>
                提交 Flag
              </Button>
            </Form>
          </div>
        ) : null}
      </Modal>
    </>
  );
}

export function CtfCompetitionArenaPage() {
  const { competition, challenges, activeChallenge, setActiveChallenge, loadArena, reloading, flagForm, submitFlag, submittingID } = useCompetitionArena();
  const [detailOpen, setDetailOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState("all");

  const categories = useMemo(
    () => ["all", ...Array.from(new Set(challenges.map((item) => item.category).filter(Boolean)))],
    [challenges],
  );

  const filteredChallenges = useMemo(
    () => challenges.filter((item) => activeCategory === "all" || item.category === activeCategory),
    [activeCategory, challenges],
  );

  if (!competition) return null;

  return (
    <div className="arena-mode-shell">
      <div className="arena-mode-head arena-mode-head-compact">
        <div>
          <div className="arena-mode-kicker">Challenge Mode</div>
        </div>
        <Button className="arena-action-button arena-action-button-secondary" icon={<ReloadOutlined />} loading={reloading} onClick={() => void loadArena(true)}>
          刷新
        </Button>
      </div>

      <div className="arena-filter-bar arena-challenges-filter">
        <div className="arena-filter-group">
          {categories.map((category) => (
            <button key={category} type="button" className={activeCategory === category ? "arena-filter-chip is-active" : "arena-filter-chip"} onClick={() => setActiveCategory(category)}>
              {category === "all" ? "全部类型" : category}
            </button>
          ))}
        </div>
      </div>

      <div className="arena-classic-grid">
        {filteredChallenges.length ? (
          filteredChallenges.map((challenge) => (
            <Card
              key={challenge.id}
              hoverable
              className={activeChallenge?.id === challenge.id ? "arena-classic-card is-active" : "arena-classic-card"}
              onClick={() => {
                setActiveChallenge(challenge);
                setDetailOpen(true);
              }}
            >
              <div className="challenge-card-accent" />
              {challenge.is_solved ? (
                <div className="challenge-card-solved-mark" aria-label="已解决">
                  <FlagOutlined />
                </div>
              ) : null}
              <div className="arena-classic-card-tags">
                <span className="arena-chip arena-chip-category">{challenge.category}</span>
                <span className={`arena-chip arena-chip-difficulty ${getDifficultyTone(challenge.difficulty)}`}>
                  {difficultyText(challenge.difficulty)}
                </span>
              </div>
              <div className="challenge-card-main">
                <Typography.Title level={4}>{challenge.name}</Typography.Title>
                <Typography.Paragraph ellipsis={{ rows: 2 }}>{challenge.description || "暂无题目描述"}</Typography.Paragraph>
              </div>
              <div className="arena-classic-card-footer">
                <span>
                  <FlagOutlined /> {challenge.points} 分
                </span>
                <span className="challenge-card-open">查看详情</span>
              </div>
            </Card>
          ))
        ) : (
          <Card className="player-panel-card arena-overview-card">
            <Empty description={`当前筛选条件下暂无${competition.name}比赛题目`} />
          </Card>
        )}
      </div>

      <ChallengeDetailModal
        open={detailOpen}
        challenge={activeChallenge}
        flagForm={flagForm}
        onClose={() => setDetailOpen(false)}
        onSubmitFlag={submitFlag}
        submittingID={submittingID}
      />
    </div>
  );
}
