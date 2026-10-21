import {
  ArrowLeftOutlined,
  ClockCircleOutlined,
  CopyOutlined,
  LinkOutlined,
  RocketOutlined,
  StopOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Descriptions,
  Progress,
  Space,
  Spin,
  Tag,
  Typography,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { useNavigate, useParams } from "react-router-dom";
import { isPlayerLoggedIn } from "@shared/auth";
import {
  playerApi,
  type PlayerVulnLabDetail,
  type PlayerVulnLabRuntime,
} from "@shared/api/player";
import { PlayerPage } from "./PlayerPage";
import { useRef } from "react";

const severityText: Record<string, string> = {
  CRITICAL: "严重",
  HIGH: "高危",
  MEDIUM: "中危",
  LOW: "低危",
  UNKNOWN: "未知",
};

export function PlayerVulnLabDetailPage() {
  const { message } = App.useApp();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const labId = Number(id);
  const [data, setData] = useState<PlayerVulnLabDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState(false);
  const [extending, setExtending] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [copyingTarget, setCopyingTarget] = useState<string | null>(null);
  const [remaining, setRemaining] = useState(0);
  const [runtimeAffixed, setRuntimeAffixed] = useState(false);
  const [runtimeHeight, setRuntimeHeight] = useState(0);
  const stickyColRef = useRef<HTMLElement | null>(null);
  const stickyWrapRef = useRef<HTMLDivElement | null>(null);

  const runtime = data?.runtime ?? null;
  const runtimeTargets = runtime?.targets ?? [];
  const runtimeActive = Boolean(
    runtime &&
    ["running", "created", "creating"].includes(
      String(runtime.status ?? "").toLowerCase(),
    ),
  );
  const severityLabel = severityText[data?.severity || "UNKNOWN"] ?? "未知";

  const load = async () => {
    if (!labId) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const detail = await playerApi.vulnLabDetail(labId);
      if (
        !detail ||
        Array.isArray((detail as unknown as { data?: unknown }).data)
      ) {
        throw new Error("漏洞靶场详情接口返回了非详情数据");
      }
      setData(detail);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载漏洞靶场详情失败",
      );
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [labId]);

  useEffect(() => {
    const expires = runtime?.expires ?? 0;
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
  }, [runtime?.expires]);

  useEffect(() => {
    const updateSticky = () => {
      const col = stickyColRef.current;
      const wrap = stickyWrapRef.current;
      if (!col || !wrap || window.innerWidth < 1200) {
        setRuntimeAffixed(false);
        setRuntimeHeight(0);
        return;
      }

      const headerOffset = 88;
      const colTop = col.getBoundingClientRect().top + window.scrollY;
      const shouldAffix = window.scrollY + headerOffset >= colTop;
      setRuntimeAffixed(shouldAffix);
      setRuntimeHeight(wrap.offsetHeight);
    };

    updateSticky();
    window.addEventListener("scroll", updateSticky, { passive: true });
    window.addEventListener("resize", updateSticky);
    return () => {
      window.removeEventListener("scroll", updateSticky);
      window.removeEventListener("resize", updateSticky);
    };
  }, [data?.id, runtimeActive, runtimeTargets.length, remaining]);

  const timeoutPercent = useMemo(() => {
    const timeout = data?.timeout_seconds || 3600;
    return Math.max(
      0,
      Math.min(100, Math.round(((remaining || 0) / timeout) * 100)),
    );
  }, [data?.timeout_seconds, remaining]);

  const ensureAuth = () => {
    if (!isPlayerLoggedIn()) {
      window.dispatchEvent(
        new CustomEvent("show-player-auth", { detail: "login" }),
      );
      message.info("请先登录后再启动靶场环境");
      return false;
    }
    return true;
  };

  const startLab = async () => {
    if (!data || !ensureAuth()) return;
    setStarting(true);
    try {
      await playerApi.startVulnLab(data.id);
      message.success("漏洞靶场环境已启动");
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "启动漏洞靶场失败",
      );
    } finally {
      setStarting(false);
    }
  };

  const extendLab = async () => {
    if (!data || !ensureAuth()) return;
    setExtending(true);
    try {
      await playerApi.extendVulnLab(data.id);
      message.success("已延长 30 分钟");
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "延长漏洞靶场失败",
      );
    } finally {
      setExtending(false);
    }
  };

  const stopLab = async () => {
    if (!data || !ensureAuth()) return;
    setStopping(true);
    try {
      await playerApi.stopVulnLab(data.id);
      message.success("漏洞靶场环境已销毁");
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "销毁漏洞靶场失败",
      );
    } finally {
      setStopping(false);
    }
  };

  const copyTarget = async (value: string) => {
    setCopyingTarget(value);
    try {
      await navigator.clipboard.writeText(value);
      message.success("已复制目标地址");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "复制失败");
    } finally {
      setCopyingTarget(null);
    }
  };

  return (
    <PlayerPage title={data?.name ?? "漏洞靶场详情"}>
      {loading ? (
        <div className="player-loading">
          <Spin size="large" />
        </div>
      ) : null}
      {!loading && !data ? (
        <Card className="player-panel-card">
          <Typography.Text>漏洞靶场不存在或无法访问</Typography.Text>
        </Card>
      ) : null}
      {!loading && data ? (
        <div className="vuln-detail-page">
          <div className="vuln-detail-breadcrumb">漏洞靶场 / 详情页</div>
          <div className="vuln-detail-titlebar">
            <Button
              type="text"
              className="vuln-detail-back"
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate("/vuln-labs")}
            />
            <div className="vuln-detail-titlegroup">
              <Typography.Title level={2}>{data.name}</Typography.Title>
              <Space wrap>
                <Tag>{severityLabel}</Tag>
                {data.cve_id ? <Tag color="gold">{data.cve_id}</Tag> : null}
              </Space>
            </div>
          </div>

          <div className="vuln-detail-layout">
            <div className="vuln-detail-main">
              <Card className="player-panel-card vuln-detail-main-card">
                <div className="vuln-detail-section">
                  <Typography.Title level={4}>漏洞摘要</Typography.Title>
                  <Typography.Paragraph>
                    {data.summary || "暂无摘要"}
                  </Typography.Paragraph>
                </div>

                {data.description ? (
                  <div className="vuln-detail-section">
                    <Typography.Title level={4}>漏洞说明</Typography.Title>
                    <MarkdownArticle content={data.description} />
                  </div>
                ) : null}

                {data.reproduce_steps ? (
                  <div className="vuln-detail-section">
                    <Typography.Title level={4}>复现步骤</Typography.Title>
                    <Card className="vuln-detail-steps-card">
                      <MarkdownArticle content={data.reproduce_steps} />
                    </Card>
                  </div>
                ) : null}

                {data.reference_links?.length ? (
                  <div className="vuln-detail-section">
                    <Typography.Title level={4}>参考链接</Typography.Title>
                    <div className="player-vuln-reference-list">
                      {data.reference_links.map((link) => (
                        <a
                          key={link}
                          href={link}
                          target="_blank"
                          rel="noreferrer"
                        >
                          <LinkOutlined />
                          <span>{link}</span>
                        </a>
                      ))}
                    </div>
                  </div>
                ) : null}
              </Card>
            </div>

            <aside
              className="vuln-detail-aside"
              ref={stickyColRef}
              style={runtimeAffixed && runtimeHeight ? { minHeight: runtimeHeight } : undefined}
            >
              <div
                ref={stickyWrapRef}
                className={`vuln-detail-sticky-wrap${runtimeAffixed ? " is-affixed" : ""}`}
              >
                <Card className="player-panel-card vuln-detail-side-card">
                  <div className="vuln-runtime-head">
                    <div>
                      <div className="vuln-runtime-kicker">Runtime</div>
                      <Typography.Title level={4}>实验环境</Typography.Title>
                    </div>
                    <span className={`vuln-runtime-status-pill${runtimeActive ? " is-running" : ""}`}>
                      {runtimeActive ? "运行中" : "待启动"}
                    </span>
                  </div>
                  {runtimeActive ? (
                    <Space
                      direction="vertical"
                      size="large"
                      style={{ width: "100%" }}
                    >
                      <Descriptions column={1} size="small">
                        <Descriptions.Item label="环境状态">
                          {runtime?.status || "running"}
                        </Descriptions.Item>
                        <Descriptions.Item label="实例名称">
                          {runtime?.name ||
                            runtime?.container_id ||
                            "已启动实例"}
                        </Descriptions.Item>
                        <Descriptions.Item label="剩余时间">
                          {formatSeconds(remaining)}
                        </Descriptions.Item>
                      </Descriptions>

                      {runtimeTargets.length ? (
                        <div className="challenge-target-list">
                          {runtimeTargets.map((target) => (
                            <div
                              key={target.url}
                              className="challenge-target-item"
                            >
                              <Space wrap>
                                <Tag color="blue">
                                  {String(
                                    target.protocol || "http",
                                  ).toUpperCase()}
                                </Tag>
                                <Typography.Link
                                  href={target.url}
                                  target="_blank"
                                >
                                  {target.url}
                                </Typography.Link>
                              </Space>
                              <Button
                                icon={<CopyOutlined />}
                                loading={copyingTarget === target.url}
                                onClick={() => void copyTarget(target.url)}
                              />
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="player-vuln-runtime-empty">
                          目标地址正在生成，请稍后刷新。
                        </div>
                      )}

                      <Progress
                        percent={timeoutPercent}
                        showInfo={false}
                        status={remaining < 300 ? "exception" : "active"}
                      />

                      <Space className="challenge-runtime-actions">
                        <Button
                          icon={<ClockCircleOutlined />}
                          className="vuln-runtime-secondary-btn"
                          loading={extending}
                          onClick={() => void extendLab()}
                        >
                          延长时间
                        </Button>
                        <Button
                          danger
                          icon={<StopOutlined />}
                          className="vuln-runtime-danger-btn"
                          loading={stopping}
                          onClick={() => void stopLab()}
                        >
                          销毁环境
                        </Button>
                      </Space>
                    </Space>
                  ) : (
                    <Space
                      direction="vertical"
                      size="middle"
                      style={{ width: "100%" }}
                    >
                      <Descriptions
                        column={1}
                        size="small"
                        className="vuln-runtime-summary"
                      >
                        <Descriptions.Item label="环境模式">
                          独立实验实例
                        </Descriptions.Item>
                        <Descriptions.Item label="启动方式">
                          为当前账号创建隔离环境
                        </Descriptions.Item>
                      </Descriptions>
                      <Button
                        type="primary"
                        icon={<RocketOutlined />}
                        className="vuln-runtime-primary-btn"
                        loading={starting}
                        onClick={() => void startLab()}
                      >
                        启动环境
                      </Button>
                    </Space>
                  )}
                </Card>
              </div>
            </aside>
          </div>
        </div>
      ) : null}
    </PlayerPage>
  );
}

function formatSeconds(value: number) {
  const safeValue = Math.max(0, value);
  const hours = Math.floor(safeValue / 3600)
    .toString()
    .padStart(2, "0");
  const minutes = Math.floor((safeValue % 3600) / 60)
    .toString()
    .padStart(2, "0");
  const seconds = Math.floor(safeValue % 60)
    .toString()
    .padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}

function MarkdownArticle({ content }: { content: string }) {
  return (
    <div className="vuln-markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw]}
        components={{
          a: ({ ...props }) => (
            <a {...props} target="_blank" rel="noreferrer" />
          ),
          img: ({ ...props }) => (
            <img {...props} alt={props.alt ?? "markdown image"} />
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
