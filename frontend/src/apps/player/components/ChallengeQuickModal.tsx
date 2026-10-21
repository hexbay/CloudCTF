import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CrownOutlined,
  DownloadOutlined,
  FlagOutlined,
  LinkOutlined,
  RocketOutlined,
  SendOutlined,
  StopOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import {
  App,
  Alert,
  Button,
  Divider,
  Input,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from "antd";
import { useEffect, useState } from "react";
import {
  playerApi,
  type PlayerChallenge,
  type PlayerChallengeDetail,
} from "@shared/api/player";
import { isPlayerLoggedIn } from "@shared/auth";
import { difficultyText } from "../pages/PlayerDashboardPage";

type ChallengeQuickModalProps = {
  challenge: PlayerChallenge | null;
  onClose: () => void;
  onSolved: (challengeId: number) => void;
};

export function ChallengeQuickModal({
  challenge,
  onClose,
  onSolved,
}: ChallengeQuickModalProps) {
  const { message } = App.useApp();
  const [detail, setDetail] = useState<PlayerChallengeDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [flag, setFlag] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [starting, setStarting] = useState(false);
  const [extending, setExtending] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [remaining, setRemaining] = useState(0);

  const open = Boolean(challenge);

  const loadDetail = async (challengeId: number) => {
    setLoading(true);
    try {
      const result = await playerApi.challengeDetail(challengeId);
      setDetail(result);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载题目失败");
      setDetail(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setFlag("");
    setDetail(null);
    setRemaining(0);
    if (challenge?.id) {
      void loadDetail(challenge.id);
    }
  }, [challenge?.id]);

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

  const ensureAuth = () => {
    if (!isPlayerLoggedIn()) {
      window.dispatchEvent(new CustomEvent("show-player-auth", { detail: "login" }));
      message.info("请先登录后再进行操作");
      return false;
    }
    return true;
  };

  const submitFlag = async () => {
    const current = detail ?? challenge;
    if (!current || !ensureAuth()) return;
    if (!flag.trim()) {
      message.warning("请输入 Flag");
      return;
    }
    setSubmitting(true);
    try {
      const result = await playerApi.submitFlag(current.id, flag.trim());
      if (result.correct) {
        message.success(result.is_first_blood ? "\u63d0\u4ea4\u6210\u529f\uff0c\u62ff\u4e0b\u4e00\u8840\uff0c\u83b7\u5f97 " + String(result.points ?? current.points ?? 0) + " \u5206" : "\u63d0\u4ea4\u6210\u529f\uff0c\u83b7\u5f97 " + String(result.points ?? current.points ?? 0) + " \u5206");
        setFlag("");
        onSolved(current.id);
        await loadDetail(current.id);
      } else if (result.already_solved) {
        message.warning(result.message ?? "\u5df2\u7ecf\u89e3\u51b3\u8fc7\u4e86");
      } else {
        message.error(result.message ?? "Flag \u9519\u8bef");
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "提交失败");
    } finally {
      setSubmitting(false);
    }
  };

  const startContainer = async () => {
    const current = detail ?? challenge;
    if (!current || !ensureAuth()) return;
    setStarting(true);
    try {
      await playerApi.startChallenge(current.id);
      message.success("环境已启动");
      await loadDetail(current.id);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "启动失败");
    } finally {
      setStarting(false);
    }
  };

  const extendContainer = async () => {
    const current = detail ?? challenge;
    if (!current || !ensureAuth()) return;
    setExtending(true);
    try {
      await playerApi.extendChallenge(current.id);
      message.success("已延长环境时间");
      await loadDetail(current.id);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "延长失败");
    } finally {
      setExtending(false);
    }
  };

  const stopContainer = async () => {
    const current = detail ?? challenge;
    if (!current || !ensureAuth()) return;
    setStopping(true);
    try {
      await playerApi.stopChallenge(current.id);
      message.success("环境已销毁");
      await loadDetail(current.id);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "销毁失败");
    } finally {
      setStopping(false);
    }
  };

  const current = detail ?? challenge;
  const solved = Boolean(detail?.solved ?? current?.is_solved);

  return (
    <Modal
      className={`challenge-quick-modal ${solved ? "is-solved" : ""}`}
      width={880}
      open={open}
      footer={null}
      closable={false}
      destroyOnHidden
      onCancel={onClose}
      title={current ? null : "题目详情"}
    >
      {loading ? (
        <div className="player-loading challenge-modal-loading">
          <Spin size="large" />
        </div>
      ) : current ? (
        <div className={`challenge-modal-content ${solved ? "is-solved" : ""}`}>
          <div className="challenge-modal-hero">
            {solved ? (
              <div className="challenge-modal-solved-ribbon" aria-label="已解决">
                <FlagOutlined />
                <span>已解决</span>
              </div>
            ) : null}
            <Space wrap size={[8, 8]}>
              <Tag
                className={`challenge-difficulty-tag difficulty-${difficultyText(current.difficulty)}`}
              >
                {difficultyText(current.difficulty)}
              </Tag>
              <Tag
                className={`challenge-category-tag category-${String(current.category).toLowerCase()}`}
              >
                {current.category}
              </Tag>
              {detail?.is_dynamic ? (
                <Tag className="challenge-state-tag is-dynamic">动态环境</Tag>
              ) : null}
            </Space>
            <Typography.Title level={2}>{current.name}</Typography.Title>
            {detail?.bloods?.length ? (
              <div className="challenge-modal-blood-strip" aria-label="题目血榜">
                {detail.bloods.slice(0, 3).map((blood, index) => (
                  <div
                    key={`${blood.user_id}-${blood.submitted_at}`}
                    className={`challenge-blood-pill rank-${index + 1}`}
                    title={blood.username}
                  >
                    <CrownOutlined />
                    <span>{blood.username}</span>
                  </div>
                ))}
              </div>
            ) : null}
            <div className="challenge-modal-score">
              <TrophyOutlined />
              <span>{current.points}</span>
              <em>分</em>
            </div>
          </div>

          <div className="challenge-modal-body">
            <section className="challenge-modal-section">
              <Typography.Title level={5}>题目描述</Typography.Title>
              <Typography.Paragraph className="challenge-modal-description">
                {current.description || "暂无描述"}
              </Typography.Paragraph>
            </section>

            {detail?.attachments?.length ? (
              <section className="challenge-modal-section">
                <Typography.Title level={5}>附件</Typography.Title>
                <Space wrap>
                  {detail.attachments.map((item, index) => {
                    const href = item.file_path || item.url || "#";
                    return (
                      <Button
                        key={`${href}-${index}`}
                        icon={<DownloadOutlined />}
                        href={href}
                        target="_blank"
                      >
                        {item.filename || `附件 ${index + 1}`}
                      </Button>
                    );
                  })}
                </Space>
              </section>
            ) : null}

            {detail?.is_dynamic ? (
              <section className="challenge-modal-section">
                <Typography.Title level={5}>动态环境</Typography.Title>
                {detail.container_info ? (
                  <div className="challenge-runtime-box">
                    <div className="challenge-runtime-status">
                      <div>
                        <span className="challenge-runtime-dot" />
                        <strong>环境运行中</strong>
                      </div>
                      <div className="challenge-runtime-countdown">
                        <ClockCircleOutlined />
                        {formatSeconds(remaining)}
                      </div>
                    </div>

                    {detail.container_info.targets?.length ? (
                      <div className="challenge-modal-targets">
                        {detail.container_info.targets.map((target) => (
                          <Typography.Link key={target.url} href={target.url} target="_blank">
                            <LinkOutlined />
                            {target.url}
                          </Typography.Link>
                        ))}
                      </div>
                    ) : (
                      <div className="challenge-runtime-target-empty">
                        目标地址会在环境启动完成后显示
                      </div>
                    )}

                    <div className="challenge-runtime-actions">
                      <Button
                        className="challenge-runtime-extend"
                        icon={<ClockCircleOutlined />}
                        loading={extending}
                        onClick={() => void extendContainer()}
                      >
                        延长时间
                      </Button>
                      <Button
                        danger
                        icon={<StopOutlined />}
                        loading={stopping}
                        onClick={() => void stopContainer()}
                      >
                        销毁环境
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="challenge-runtime-box is-idle">
                    <Typography.Text>
                      动态容器尚未启动，目标地址会在启动完成后生成。
                    </Typography.Text>
                    <Button
                      type="primary"
                      icon={<RocketOutlined />}
                      loading={starting}
                      onClick={() => void startContainer()}
                    >
                      启动环境
                    </Button>
                  </div>
                )}
              </section>
            ) : null}

            {detail?.hints?.length ? (
              <section className="challenge-modal-section">
                <Typography.Title level={5}>提示</Typography.Title>
                <div className="challenge-modal-hints">
                  {detail.hints.map((hint) => (
                    <div key={hint.index} className="challenge-modal-hint">
                      <span>{`提示 ${hint.index + 1}`}</span>
                      <Tag color={hint.purchased ? "success" : "gold"}>
                        {hint.purchased ? "已解锁" : `${hint.cost} 积分`}
                      </Tag>
                      {hint.purchased ? <p>{hint.content || "暂无内容"}</p> : null}
                    </div>
                  ))}
                </div>
              </section>
            ) : null}

            <Divider />

            <section className="challenge-modal-submit">
              {solved ? (
                <div className="challenge-modal-solved-panel">
                  <CheckCircleOutlined />
                  <span>该题已完成</span>
                </div>
              ) : (
                <div className="challenge-submit-panel">
                  <div className="challenge-submit-head">
                    <strong>提交 Flag</strong>
                    <span>输入本题 Flag 后提交校验</span>
                  </div>
                  <Input.Search
                    className="challenge-flag-search"
                    value={flag}
                    onChange={(event) => setFlag(event.target.value)}
                    onSearch={() => void submitFlag()}
                    enterButton="提交"
                    suffix={<SendOutlined />}
                    loading={submitting}
                    placeholder="flag{...}"
                  />
                </div>
              )}
            </section>
          </div>
        </div>
      ) : (
        <Alert type="error" showIcon message="题目不存在或无法访问" />
      )}
    </Modal>
  );
}

function formatSeconds(value: number) {
  const safeValue = Math.max(0, value);
  const hours = Math.floor(safeValue / 3600).toString().padStart(2, "0");
  const minutes = Math.floor((safeValue % 3600) / 60).toString().padStart(2, "0");
  const seconds = Math.floor(safeValue % 60).toString().padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}
