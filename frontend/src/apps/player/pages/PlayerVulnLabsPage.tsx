import {
  BugOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  ExperimentOutlined,
  EyeOutlined,
  RadarChartOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Col,
  Input,
  Modal,
  Row,
  Spin,
  Tabs,
  Typography,
} from "antd";
import { useEffect, useRef, useState } from "react";
import { isPlayerLoggedIn } from "@shared/auth";
import {
  playerApi,
  type PlayerVulnLabDetail,
  type PlayerVulnLabRuntime,
  type VulnLab,
} from "@shared/api/player";
import { useNavigate } from "react-router-dom";
import { VulnLabShellModal } from "../components/VulnLabShellModal";
import { PlayerPage } from "./PlayerPage";

const severities = [
  { key: "all", label: "全部" },
  { key: "CRITICAL", label: "严重" },
  { key: "HIGH", label: "高危" },
  { key: "MEDIUM", label: "中危" },
  { key: "LOW", label: "低危" },
  { key: "UNKNOWN", label: "未知" },
];

const severityText: Record<string, string> = {
  CRITICAL: "严重",
  HIGH: "高危",
  MEDIUM: "中危",
  LOW: "低危",
  UNKNOWN: "未知",
};

const severityTone: Record<string, string> = {
  CRITICAL: "critical",
  HIGH: "high",
  MEDIUM: "medium",
  LOW: "low",
  UNKNOWN: "unknown",
};

export function PlayerVulnLabsPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const loaderRef = useRef<HTMLDivElement | null>(null);
  const [activeSeverity, setActiveSeverity] = useState("all");
  const [keyword, setKeyword] = useState("");
  const [data, setData] = useState<VulnLab[]>([]);
  const [loading, setLoading] = useState(true);
  const [current, setCurrent] = useState(1);
  const [pageSize] = useState(12);
  const [total, setTotal] = useState(0);
  const [loadingMore, setLoadingMore] = useState(false);
  const [activeLab, setActiveLab] = useState<VulnLab | null>(null);
  const [activeLabDetail, setActiveLabDetail] =
    useState<PlayerVulnLabDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [runtime, setRuntime] = useState<PlayerVulnLabRuntime | null>(null);
  const [starting, setStarting] = useState(false);
  const [extending, setExtending] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [remaining, setRemaining] = useState(0);
  const [keywordInput, setKeywordInput] = useState("");
  const [shellOpen, setShellOpen] = useState(false);

  useEffect(() => {
    const severity = activeSeverity === "all" ? undefined : activeSeverity;
    const isFirstPage = current === 1;

    if (isFirstPage) {
      setLoading(true);
    } else {
      setLoadingMore(true);
    }

    playerApi
      .vulnLabs({
        current,
        pageSize,
        severity,
        keyword: keyword || undefined,
      })
      .then((res) => {
        const nextPageData = Array.isArray(res.data) ? res.data : [];
        setData((prev) =>
          isFirstPage ? nextPageData : [...prev, ...nextPageData],
        );
        setTotal(typeof res.total === "number" ? res.total : 0);
      })
      .catch(() => {
        if (isFirstPage) {
          setData([]);
          setTotal(0);
        }
      })
      .finally(() => {
        setLoading(false);
        setLoadingMore(false);
      });
  }, [activeSeverity, current, keyword, pageSize]);

  useEffect(() => {
    setData([]);
    setTotal(0);
    setCurrent(1);
  }, [activeSeverity, keyword]);

  const hasMore = data.length < total;

  useEffect(() => {
    if (loading || loadingMore || !hasMore) {
      return;
    }

    const node = loaderRef.current;
    if (!node) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry?.isIntersecting) {
          setCurrent((prev) => prev + 1);
        }
      },
      { rootMargin: "240px 0px" },
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [hasMore, loading, loadingMore]);

  const showEmptyState = !loading && data.length === 0;
  const visibleCount = data.length;
  const totalCount = total;
  const displayLab = activeLabDetail ?? activeLab;
  const activeSeverityText =
    severityText[displayLab?.severity || "UNKNOWN"] ?? "未知";
  const activeTone =
    severityTone[displayLab?.severity || "UNKNOWN"] ?? "unknown";
  const currentRuntime = activeLabDetail?.runtime ?? runtime;
  const runtimeTargets = currentRuntime?.targets ?? [];
  const primaryTarget = runtimeTargets[0];
  const hasMultipleTargets = runtimeTargets.length > 1;
  const runtimeActive = Boolean(
    currentRuntime &&
    ["running", "created", "creating"].includes(
      String(currentRuntime.status ?? "").toLowerCase(),
    ),
  );

  useEffect(() => {
    if (!activeLab) {
      setActiveLabDetail(null);
      setRuntime(null);
      setRemaining(0);
      return;
    }
    setActiveLabDetail(null);
    setRuntime(null);
    setRemaining(0);
  }, [activeLab?.id]);

  useEffect(() => {
    if (!activeLab) {
      return;
    }

    let cancelled = false;
    setDetailLoading(true);

    playerApi
      .vulnLabDetail(activeLab.id)
      .then((detail) => {
        if (cancelled) {
          return;
        }
        setActiveLabDetail(detail);
        if (detail.runtime) {
          setRuntime(detail.runtime);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setActiveLabDetail(null);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setDetailLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [activeLab]);

  useEffect(() => {
    const expires = resolveRuntimeRemainingSeconds(currentRuntime);
    setRemaining(expires);
    if (!expires) {
      return;
    }

    const timer = window.setInterval(() => {
      setRemaining((value) => {
        if (value <= 1) {
          window.clearInterval(timer);
          return 0;
        }
        return value - 1;
      });
    }, 1000);

    return () => window.clearInterval(timer);
  }, [currentRuntime?.expires, currentRuntime?.expires_at]);

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

  const handleStartLab = async () => {
    if (!activeLab || !ensureAuth()) return;
    setStarting(true);
    try {
      const result = await playerApi.startVulnLab(activeLab.id);
      setRuntime(result.group ?? null);
      setData((prev) => {
        const next = prev.map((item) =>
          item.id === activeLab.id ? { ...item, runtime: result.group ?? null } : item,
        );
        const started = next.find((item) => item.id === activeLab.id);
        if (!started) {
          return next;
        }
        return [started, ...next.filter((item) => item.id !== activeLab.id)];
      });
      message.success(result.message ?? "漏洞靶场环境已启动");
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "启动漏洞靶场失败",
      );
    } finally {
      setStarting(false);
    }
  };

  const handleExtendLab = async () => {
    if (!activeLab || !ensureAuth()) return;
    setExtending(true);
    try {
      const result = await playerApi.extendVulnLab(activeLab.id);
      if (result.container_info) {
        setRuntime(result.container_info);
      }
      message.success(result.message ?? "已延长漏洞靶场环境时间");
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "延长漏洞靶场失败",
      );
    } finally {
      setExtending(false);
    }
  };

  const handleStopLab = async () => {
    if (!activeLab || !ensureAuth()) return;
    setStopping(true);
    try {
      await playerApi.stopVulnLab(activeLab.id);
      setRuntime(null);
      setRemaining(0);
      setData((prev) =>
        prev.map((item) =>
          item.id === activeLab.id ? { ...item, runtime: null } : item,
        ),
      );
      message.success("漏洞靶场环境已销毁");
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "销毁漏洞靶场失败",
      );
    } finally {
      setStopping(false);
    }
  };

  const handleCloseDetail = () => {
    setShellOpen(false);
    setActiveLab(null);
  };

  return (
    <PlayerPage
      icon={<BugOutlined />}
      title="漏洞靶场"
      subtitle="真实靶场数据与实例操作入口"
    >
      <div className="player-filter-bar player-vuln-filter">
        <Tabs
          activeKey={activeSeverity}
          onChange={setActiveSeverity}
          items={severities}
        />
        <Input
          className="player-challenges-search player-vuln-search"
          placeholder="搜索标题 / CVE / 摘要"
          prefix={<SearchOutlined />}
          value={keywordInput}
          onChange={(e) => setKeywordInput(e.target.value)}
          onPressEnter={() => setKeyword(keywordInput.trim())}
          onBlur={() => setKeyword(keywordInput.trim())}
          allowClear
          onClear={() => {
            setKeywordInput("");
            setKeyword("");
          }}
        />
      </div>
      {!loading ? (
        <div className="player-vuln-toolbar">
          <div className="player-vuln-toolbar-stat">
            <DatabaseOutlined />
            <span>当前展示 {visibleCount} 个靶场</span>
          </div>
          <div className="player-vuln-toolbar-stat">
            <ExperimentOutlined />
            <span>共收录 {totalCount} 个复现条目</span>
          </div>
        </div>
      ) : null}
      {loading ? (
        <div className="player-loading">
          <Spin size="large" />
        </div>
      ) : !showEmptyState ? (
        <Row gutter={[16, 16]}>
          {data.map((item) => {
            const severity = item.severity || "UNKNOWN";
            const tone = severityTone[severity] ?? "unknown";
            const isRunning = Boolean(item.runtime);
            return (
              <Col xs={12} md={8} xl={6} key={item.id}>
                <Card
                  className={`vuln-card severity-${severity.toLowerCase()} severity-tone-${tone}`}
                  hoverable
                  onClick={() => setActiveLab(item)}
                >
                  <div className="vuln-card-grid" />
                  <div className="vuln-card-head">
                    <div className="vuln-card-kicker">Exploit Lab</div>
                    <div className="vuln-card-badges">
                      {isRunning ? (
                        <span className="vuln-runtime-pill">运行中</span>
                      ) : null}
                      <span className="vuln-severity-pill">
                        {severityText[severity] ?? "未知"}
                      </span>
                    </div>
                  </div>
                  <div className="vuln-card-main">
                    <Typography.Title level={5}>{item.name}</Typography.Title>
                    {item.cve_id ? (
                      <div className="vuln-cve">{item.cve_id}</div>
                    ) : null}
                  </div>
                  <Typography.Paragraph
                    className="vuln-summary"
                    ellipsis={{ rows: 3 }}
                  >
                    {item.summary || "暂无摘要"}
                  </Typography.Paragraph>
                  <div className="vuln-card-footer">
                    <div className="vuln-card-signal">
                      <span className="vuln-card-signal-dot" />
                      <span>创建于 {formatDate(item.created_at)}</span>
                    </div>
                    <span className="vuln-card-action">查看实验</span>
                  </div>
                </Card>
              </Col>
            );
          })}
        </Row>
      ) : (
        <Card className="player-empty-card player-vuln-empty-state">
          <div className="player-vuln-empty-grid" />
          <div className="player-vuln-empty-orb player-vuln-empty-orb-a" />
          <div className="player-vuln-empty-orb player-vuln-empty-orb-b" />

          <div className="player-vuln-empty-badge">Lab Matrix</div>

          <div className="player-vuln-empty-icon">
            <BugOutlined />
          </div>

          <Typography.Title level={2} className="player-vuln-empty-title">
            漏洞靶场正在准备中
          </Typography.Title>

          <Typography.Paragraph className="player-vuln-empty-description">
            当前还没有可启动的靶场环境。等实验资源上线后，你可以在这里按风险等级筛选、搜索
            CVE， 然后直接进入漏洞复现与验证流程。
          </Typography.Paragraph>

          <div className="player-vuln-empty-features">
            <div className="player-vuln-empty-feature">
              <RadarChartOutlined className="player-vuln-empty-feature-icon" />
              <div>
                <strong>风险分级清晰</strong>
                <span>严重、高危、中危等等级会自动归类展示</span>
              </div>
            </div>
            <div className="player-vuln-empty-feature">
              <SafetyCertificateOutlined className="player-vuln-empty-feature-icon" />
              <div>
                <strong>真实复现路径</strong>
                <span>支持从 CVE 到环境实例的一站式靶场体验</span>
              </div>
            </div>
            <div className="player-vuln-empty-feature">
              <ThunderboltOutlined className="player-vuln-empty-feature-icon" />
              <div>
                <strong>即开即测</strong>
                <span>资源发布后可直接从这里进入漏洞验证空间</span>
              </div>
            </div>
          </div>

          <div className="player-vuln-empty-actions">
            <Button type="primary" size="large" href="/challenges">
              先去练习
            </Button>
          </div>
        </Card>
      )}
      {!loading && !showEmptyState ? (
        <div className="player-vuln-loadmore" ref={loaderRef}>
          {loadingMore ? (
            <>
              <Spin size="small" />
              <span>正在加载更多靶场...</span>
            </>
          ) : hasMore ? (
            <span>继续下滑加载更多</span>
          ) : (
            <span>已经到底了</span>
          )}
        </div>
      ) : null}
      <Modal
        className="player-vuln-detail-modal"
        open={Boolean(activeLab)}
        footer={null}
        width={760}
        destroyOnHidden
        onCancel={handleCloseDetail}
      >
        {activeLab ? (
          <div className={`player-vuln-detail severity-tone-${activeTone}`}>
            <div className="player-vuln-detail-hero">
              <div className="player-vuln-detail-kicker">
                Vulnerability Replay
              </div>
              <div className="player-vuln-detail-head">
                <Typography.Title level={3}>
                  {displayLab?.name}
                </Typography.Title>
              </div>
              {displayLab?.cve_id || activeSeverityText !== "未知" ? (
                <div className="player-vuln-detail-submeta">
                  {displayLab?.cve_id ? (
                    <span className="vuln-cve">{displayLab.cve_id}</span>
                  ) : null}
                  {activeSeverityText !== "未知" ? (
                    <span className="player-vuln-detail-submeta-severity">
                      {activeSeverityText}
                    </span>
                  ) : null}
                </div>
              ) : null}
            </div>

            <div className="player-vuln-detail-section">
              <Typography.Title level={5}>漏洞摘要</Typography.Title>
              <Typography.Paragraph>
                {displayLab?.summary || "当前条目暂未提供摘要信息。"}
              </Typography.Paragraph>
            </div>

            <div className="player-vuln-runtime-panel">
              <div className="player-vuln-runtime-panel-head">
                <div>
                  <span className="player-vuln-runtime-panel-kicker">
                    Runtime Access
                  </span>
                </div>
              </div>

              <div className="player-vuln-runtime-primary">
                {runtimeActive ? (
                  <>
                    <div className="player-vuln-runtime-primary-card is-target">
                      <span className="player-vuln-runtime-primary-label">
                        环境地址
                      </span>
                      {primaryTarget?.url ? (
                        <a
                          className="player-vuln-runtime-primary-link"
                          href={primaryTarget.url}
                          target="_blank"
                          rel="noreferrer"
                        >
                          {primaryTarget.url}
                        </a>
                      ) : (
                        <strong>暂无入口</strong>
                      )}
                      {primaryTarget ? (
                        <Typography.Text type="secondary">
                          {primaryTarget.protocol?.toUpperCase() || "HTTP"} /
                          外部端口 {primaryTarget.port} / 容器端口{" "}
                          {primaryTarget.container_port}
                        </Typography.Text>
                      ) : null}
                    </div>
                    <div className="player-vuln-runtime-primary-card">
                      <span className="player-vuln-runtime-primary-label">
                        剩余时间
                      </span>
                      <strong>{formatSeconds(remaining)}</strong>
                      <Typography.Text type="secondary">
                        {`到期 ${formatDateTime(currentRuntime?.expires_at)}`}
                      </Typography.Text>
                    </div>
                  </>
                ) : (
                  <div className="player-vuln-runtime-empty-cta">
                    <div className="player-vuln-runtime-empty-copy">
                      <strong>环境尚未启动</strong>
                      <span>启动后即可进入目标环境进行复现。</span>
                    </div>
                    <Button
                      type="primary"
                      size="large"
                      icon={<RocketOutlined />}
                      loading={starting}
                      onClick={() => void handleStartLab()}
                    >
                      启动环境
                    </Button>
                  </div>
                )}
              </div>

              {runtimeActive && hasMultipleTargets ? (
                <div className="player-vuln-runtime-target-list">
                  {runtimeTargets.map((target, index) => (
                    <div
                      className="player-vuln-runtime-target-item"
                      key={`${target.url}-${index}`}
                    >
                      <div>
                        <span>目标 {index + 1}</span>
                        <a
                          className="player-vuln-runtime-target-link"
                          href={target.url}
                          target="_blank"
                          rel="noreferrer"
                        >
                          {target.url}
                        </a>
                      </div>
                      <div className="player-vuln-runtime-target-actions">
                        <em>
                          {target.protocol?.toUpperCase() || "HTTP"} ·{" "}
                          {target.port} → {target.container_port}
                        </em>
                      </div>
                    </div>
                  ))}
                </div>
              ) : null}

              {runtimeActive ? (
                <div className="player-vuln-runtime-inline-actions">
                  <Button onClick={() => setShellOpen(true)}>打开 Shell</Button>
                  <Button
                    icon={<ClockCircleOutlined />}
                    loading={extending}
                    onClick={() => void handleExtendLab()}
                  >
                    延长时间
                  </Button>
                  <Button
                    danger
                    icon={<StopOutlined />}
                    loading={stopping}
                    onClick={() => void handleStopLab()}
                  >
                    销毁环境
                  </Button>
                </div>
              ) : null}
            </div>

            <div className="player-vuln-detail-actions">
              <Button onClick={handleCloseDetail}>关闭</Button>
              <Button
                icon={<EyeOutlined />}
                onClick={() => {
                  setShellOpen(false);
                  setActiveLab(null);
                  navigate(`/vuln-labs/${activeLab.id}`);
                }}
              >
                查看完整详情
              </Button>
            </div>
          </div>
        ) : null}
      </Modal>
      <VulnLabShellModal
        key={`${activeLab?.id ?? "none"}-${shellOpen ? "open" : "closed"}`}
        open={shellOpen}
        labId={activeLab?.id ?? null}
        title={displayLab?.name || activeLab?.name || "环境终端"}
        onClose={() => setShellOpen(false)}
      />
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

function formatDate(value?: string) {
  if (!value) {
    return "未知时间";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "未知时间";
  }

  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatDateTime(value?: string) {
  if (!value) {
    return "未设置";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "未设置";
  }

  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  const hours = `${date.getHours()}`.padStart(2, "0");
  const minutes = `${date.getMinutes()}`.padStart(2, "0");
  const seconds = `${date.getSeconds()}`.padStart(2, "0");
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

function resolveRuntimeRemainingSeconds(runtime?: PlayerVulnLabRuntime | null) {
  if (!runtime) {
    return 0;
  }

  if (typeof runtime.expires === "number" && runtime.expires > 0) {
    return runtime.expires;
  }

  if (!runtime.expires_at) {
    return 0;
  }

  const expiresAt = new Date(runtime.expires_at);
  if (Number.isNaN(expiresAt.getTime())) {
    return 0;
  }

  return Math.max(0, Math.floor((expiresAt.getTime() - Date.now()) / 1000));
}
