import {
  AimOutlined,
  AreaChartOutlined,
  AuditOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CompassOutlined,
  DashboardOutlined,
  DeploymentUnitOutlined,
  FieldTimeOutlined,
  FireOutlined,
  FlagOutlined,
  FundProjectionScreenOutlined,
  LineChartOutlined,
  RadarChartOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  TrophyOutlined,
  UserAddOutlined,
  UsergroupAddOutlined,
} from '@ant-design/icons';
import { Button, Card, Col, Empty, Row, Space, Spin, Tag, Typography } from 'antd';
import type { EChartsOption } from 'echarts';
import ReactECharts from 'echarts-for-react';
import type { ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { adminApi } from '@shared/api/admin';

type DashboardStats = {
  users?: { total?: number; new_today?: number; new_week?: number };
  competitions?: { total?: number; ongoing?: number; upcoming?: number; finished?: number };
  challenges?: { total?: number };
  teams?: { total?: number };
  submissions?: { total?: number; today?: number; correct?: number; success_rate?: number };
  participants?: { total?: number };
};

type CompetitionItem = {
  id?: number | string;
  name?: string;
  title?: string;
  status?: string;
  type?: string;
  participant_count?: number;
  start_time?: string;
  end_time?: string;
};

type ChallengeItem = {
  id?: number | string;
  name?: string;
  category?: string;
  points?: number;
  difficulty?: number;
  submission_count?: number;
  solve_count?: number;
  solve_rate?: number;
};

type ActiveUserItem = {
  id?: number | string;
  username?: string;
  submission_count?: number;
  solve_count?: number;
};

type TrendItem = {
  date?: string;
  total?: number;
  correct?: number;
  count?: number;
};

type DashboardData = {
  stats: DashboardStats;
  recentCompetitions: CompetitionItem[];
  hotChallenges: ChallengeItem[];
  activeUsers: ActiveUserItem[];
  submissionTrend: TrendItem[];
  userGrowth: TrendItem[];
};

type ApiEnvelope<T> = T | { success?: boolean; data?: T };

const emptyDashboard: DashboardData = {
  stats: {},
  recentCompetitions: [],
  hotChallenges: [],
  activeUsers: [],
  submissionTrend: [],
  userGrowth: [],
};

const radarAxes = [
  { key: 'activity', label: '活跃' },
  { key: 'quality', label: '质量' },
  { key: 'growth', label: '增长' },
  { key: 'content', label: '内容' },
  { key: 'events', label: '赛事' },
] as const;

export function AdminDashboardPage() {
  const navigate = useNavigate();
  const [data, setData] = useState<DashboardData>(emptyDashboard);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;

    async function loadDashboard() {
      setLoading(true);
      const [stats, recentCompetitions, hotChallenges, activeUsers, submissionTrend, userGrowth] = await Promise.all([
        readDashboardEndpoint<DashboardStats>('/api-admin/dashboard/stats', normalizeStats),
        readDashboardEndpoint<CompetitionItem[]>('/api-admin/dashboard/recent-competitions', asArray),
        readDashboardEndpoint<ChallengeItem[]>('/api-admin/dashboard/hot-challenges', asArray),
        readDashboardEndpoint<ActiveUserItem[]>('/api-admin/dashboard/active-users', asArray),
        readDashboardEndpoint<TrendItem[]>('/api-admin/dashboard/submission-trend', asArray, { days: 14 }),
        readDashboardEndpoint<TrendItem[]>('/api-admin/dashboard/user-growth', asArray, { days: 14 }),
      ]);

      if (mounted) {
        setData({ stats, recentCompetitions, hotChallenges, activeUsers, submissionTrend, userGrowth });
        setLoading(false);
      }
    }

    void loadDashboard().catch(() => {
      if (mounted) {
        setData(emptyDashboard);
        setLoading(false);
      }
    });

    return () => {
      mounted = false;
    };
  }, []);

  const model = buildDashboardModel(data);

  return (
    <Spin spinning={loading}>
      <div className="admin-dashboard dashboard-v2">
        <section className="dashboard-command">
          <div className="dashboard-command-copy">
            <Tag className="dashboard-command-kicker" icon={<DashboardOutlined />}>
              CloudCTF Operations
            </Tag>
            <Typography.Paragraph>
              聚合赛事、题库、用户、提交质量与增长趋势，先看整体态势，再决定今天优先处理哪条运营链路。
            </Typography.Paragraph>
            <Space wrap>
              <Button type="primary" onClick={() => navigate('/admin/competition')}>
                调度赛事
              </Button>
              <Button onClick={() => navigate('/admin/virtualization')}>查看资源</Button>
              <Button onClick={() => navigate('/admin/audit')}>审计日志</Button>
            </Space>
          </div>

          <div className="dashboard-command-grid">
            <RadialKpi
              label="平台健康指数"
              value={model.healthScore}
              icon={<SafetyCertificateOutlined />}
              caption={`${formatPercent(model.successRate)} 正确率 · ${model.liveCompetitions} 场进行中`}
            />
            <div className="dashboard-command-side">
              <SignalTile icon={<FieldTimeOutlined />} label="今日提交" value={model.todaySubmissions} />
              <SignalTile icon={<UserAddOutlined />} label="7 日新增" value={model.weekNewUsers} />
              <SignalTile icon={<FlagOutlined />} label="题库存量" value={model.challengeTotal} />
              <SignalTile icon={<TrophyOutlined />} label="待办赛事" value={model.upcomingCompetitions} />
            </div>
          </div>
        </section>

        <section className="dashboard-v2-kpi-grid">
          {model.kpis.map((item) => (
            <MetricCard key={item.label} {...item} />
          ))}
        </section>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={15}>
            <ChartCard icon={<LineChartOutlined />} title="提交与解题趋势" extra="近 14 天">
              <ComboTrendChart data={data.submissionTrend} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={9}>
            <ChartCard icon={<RadarChartOutlined />} title="平台能力雷达" extra={`健康 ${model.healthScore}`}>
              <RadarPanel values={model.radarValues} />
            </ChartCard>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12} xl={8}>
            <ChartCard icon={<AreaChartOutlined />} title="用户增长" extra="近 14 天">
              <GrowthAreaChart data={data.userGrowth} total={model.usersTotal} />
            </ChartCard>
          </Col>
          <Col xs={24} lg={12} xl={8}>
            <ChartCard icon={<BarChartOutlined />} title="赛事状态分布" extra={`${model.competitionTotal} 场`}>
              <StatusDistribution
                items={[
                  { label: '进行中', value: model.liveCompetitions, tone: 'green' },
                  { label: '待开始', value: model.upcomingCompetitions, tone: 'cyan' },
                  { label: '已结束', value: model.finishedCompetitions, tone: 'amber' },
                ]}
              />
            </ChartCard>
          </Col>
          <Col xs={24} xl={8}>
            <ChartCard icon={<FundProjectionScreenOutlined />} title="运营质量" extra="实时估算">
              <QualityMatrix items={model.qualityItems} />
            </ChartCard>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={8}>
            <ChartCard icon={<ClockCircleOutlined />} title="近期赛事" extra="最近创建">
              <RecentCompetitions data={data.recentCompetitions} onOpen={(id) => navigate(`/admin/competition/${id}`)} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={8}>
            <ChartCard icon={<FireOutlined />} title="题目热力榜" extra="按提交量">
              <HotChallenges data={data.hotChallenges} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={8}>
            <ChartCard icon={<UsergroupAddOutlined />} title="活跃用户" extra="按提交量">
              <ActiveUsers data={data.activeUsers} />
            </ChartCard>
          </Col>
        </Row>
      </div>
    </Spin>
  );
}

function buildDashboardModel(data: DashboardData) {
  const usersTotal = readNumber(data.stats.users?.total);
  const competitionTotal = readNumber(data.stats.competitions?.total);
  const challengeTotal = readNumber(data.stats.challenges?.total);
  const teamsTotal = readNumber(data.stats.teams?.total);
  const participantsTotal = readNumber(data.stats.participants?.total);
  const submissionTotal = readNumber(data.stats.submissions?.total);
  const correctSubmissions = readNumber(data.stats.submissions?.correct);
  const todaySubmissions = readNumber(data.stats.submissions?.today);
  const successRate = readNumber(data.stats.submissions?.success_rate);
  const liveCompetitions = readNumber(data.stats.competitions?.ongoing);
  const upcomingCompetitions = readNumber(data.stats.competitions?.upcoming);
  const finishedCompetitions = readNumber(data.stats.competitions?.finished);
  const weekNewUsers = readNumber(data.stats.users?.new_week);
  const activeUserCount = data.activeUsers.length;
  const hotChallengeCount = data.hotChallenges.length;
  const avgSubmissionsPerUser = usersTotal > 0 ? submissionTotal / usersTotal : 0;
  const avgSolvesPerActiveUser =
    activeUserCount > 0
      ? data.activeUsers.reduce((sum, item) => sum + readNumber(item.solve_count), 0) / activeUserCount
      : 0;
  const eventCoverage = competitionTotal > 0 ? ((liveCompetitions + upcomingCompetitions) / competitionTotal) * 100 : 0;
  const challengePressure = challengeTotal > 0 ? (submissionTotal / challengeTotal) * 10 : 0;
  const growthRate = usersTotal > 0 ? (weekNewUsers / usersTotal) * 100 : 0;
  const healthScore = clamp(
    successRate * 0.36 +
      clamp(growthRate * 8) * 0.16 +
      clamp(avgSubmissionsPerUser * 12) * 0.18 +
      clamp(eventCoverage) * 0.16 +
      clamp(challengeTotal * 8) * 0.14,
  );

  return {
    usersTotal,
    competitionTotal,
    challengeTotal,
    teamsTotal,
    participantsTotal,
    submissionTotal,
    correctSubmissions,
    todaySubmissions,
    successRate,
    liveCompetitions,
    upcomingCompetitions,
    finishedCompetitions,
    weekNewUsers,
    activeUserCount,
    hotChallengeCount,
    healthScore,
    radarValues: {
      activity: clamp(avgSubmissionsPerUser * 12),
      quality: clamp(successRate),
      growth: clamp(growthRate * 8),
      content: clamp(challengeTotal * 10),
      events: clamp(eventCoverage),
    },
    kpis: [
      {
        icon: <TeamOutlined />,
        label: '用户规模',
        value: usersTotal,
        detail: `7 日新增 ${formatNumber(weekNewUsers)} 人`,
        trend: growthRate,
        tone: 'cyan',
      },
      {
        icon: <TrophyOutlined />,
        label: '赛事运营',
        value: competitionTotal,
        detail: `${liveCompetitions} 进行中 · ${upcomingCompetitions} 待开始`,
        trend: eventCoverage,
        tone: 'amber',
      },
      {
        icon: <AimOutlined />,
        label: '题库容量',
        value: challengeTotal,
        detail: `${hotChallengeCount} 个题目产生热度`,
        trend: challengeTotal * 10,
        tone: 'green',
      },
      {
        icon: <AuditOutlined />,
        label: '提交吞吐',
        value: submissionTotal,
        detail: `今日 ${todaySubmissions} 次 · 正确 ${correctSubmissions} 次`,
        trend: successRate,
        tone: 'blue',
      },
      {
        icon: <DeploymentUnitOutlined />,
        label: '队伍结构',
        value: teamsTotal,
        detail: `${participantsTotal} 名队员参与`,
        trend: teamsTotal > 0 ? (participantsTotal / teamsTotal) * 25 : 0,
        tone: 'slate',
      },
      {
        icon: <CompassOutlined />,
        label: '活跃密度',
        value: activeUserCount,
        detail: `人均提交 ${avgSubmissionsPerUser.toFixed(1)} 次`,
        trend: avgSolvesPerActiveUser * 18,
        tone: 'rose',
      },
    ],
    qualityItems: [
      { label: '正确率', value: successRate, text: formatPercent(successRate), icon: <CheckCircleOutlined /> },
      { label: '用户增长率', value: growthRate, text: formatPercent(growthRate), icon: <UserAddOutlined /> },
      { label: '赛事覆盖', value: eventCoverage, text: formatPercent(eventCoverage), icon: <TrophyOutlined /> },
      { label: '题目压力', value: clamp(challengePressure), text: `${avgSubmissionsPerUser.toFixed(1)} / 人`, icon: <FlagOutlined /> },
    ],
  };
}

function ChartCard({ icon, title, extra, children }: { icon: ReactNode; title: string; extra?: string; children: ReactNode }) {
  return (
    <Card
      className="dashboard-v2-card"
      title={
        <span className="dashboard-v2-card-title">
          <span className="dashboard-v2-card-icon">{icon}</span>
          {title}
        </span>
      }
      extra={extra ? <span className="dashboard-v2-card-extra">{extra}</span> : null}
    >
      {children}
    </Card>
  );
}

function MetricCard({
  icon,
  label,
  value,
  detail,
  trend,
  tone,
}: {
  icon: ReactNode;
  label: string;
  value: number;
  detail: string;
  trend: number;
  tone: string;
}) {
  return (
    <div className={`dashboard-v2-metric dashboard-v2-tone-${tone}`}>
      <div className="dashboard-v2-metric-head">
        <span className="dashboard-v2-icon">{icon}</span>
        <span>{label}</span>
      </div>
      <strong>{formatNumber(value)}</strong>
      <small>{detail}</small>
      <div className="dashboard-v2-meter">
        <span style={{ width: `${clamp(trend)}%` }} />
      </div>
    </div>
  );
}

function RadialKpi({ label, value, icon, caption }: { label: string; value: number; icon: ReactNode; caption: string }) {
  const safeValue = clamp(value);
  const radius = 54;
  const circumference = 2 * Math.PI * radius;
  const dashOffset = circumference * (1 - safeValue / 100);

  return (
    <div className="dashboard-radial-kpi">
      <div className="dashboard-radial-title">
        <span>{icon}</span>
        <strong>{label}</strong>
      </div>
      <div className="dashboard-radial-ring">
        <svg viewBox="0 0 148 148" className="dashboard-radial-svg" aria-hidden="true">
          <defs>
            <linearGradient id="dashboard-radial-gradient-fixed" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#67e8f9" />
              <stop offset="100%" stopColor="#0f766e" />
            </linearGradient>
          </defs>
          <circle className="dashboard-radial-track" cx="74" cy="74" r={radius} />
          <circle
            className="dashboard-radial-progress"
            cx="74"
            cy="74"
            r={radius}
            strokeDasharray={circumference}
            strokeDashoffset={dashOffset}
          />
        </svg>
        <div className="dashboard-radial-overlay">
          <strong>{safeValue}</strong>
          <small>{caption}</small>
        </div>
      </div>
    </div>
  );
}

function SignalTile({ icon, label, value }: { icon: ReactNode; label: string; value: number }) {
  return (
    <div className="dashboard-signal-tile">
      <span>{icon}</span>
      <strong>{formatNumber(value)}</strong>
      <small>{label}</small>
    </div>
  );
}

function ComboTrendChart({ data }: { data: TrendItem[] }) {
  const series = completeTrendSeries(data, 14);
  if (!series.some((item) => readNumber(item.total) || readNumber(item.correct))) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无提交趋势" />;
  }

  const option: EChartsOption = {
    animationDuration: 700,
    animationEasing: 'quarticOut',
    grid: { top: 20, right: 8, bottom: 28, left: 8, containLabel: true },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: 'rgba(255,255,255,0.96)',
      borderColor: 'rgba(15,118,110,0.18)',
      textStyle: { color: '#475569' },
      formatter: (params) => {
        const items = Array.isArray(params) ? params : [params];
        const total = readNumber(items.find((item) => item.seriesName === '提交')?.value);
        const correct = readNumber(items.find((item) => item.seriesName === '正确')?.value);
        const rate = total > 0 ? (correct / total) * 100 : 0;
        const label = readTooltipAxisLabel(items[0]);
        return `${label}<br/>提交 ${formatNumber(total)}<br/>正确 ${formatNumber(correct)}<br/>正确率 ${formatPercent(rate)}`;
      },
    },
    legend: {
      bottom: 0,
      icon: 'circle',
      textStyle: { color: '#64748b', fontSize: 12 },
      data: ['提交', '正确'],
    },
    xAxis: {
      type: 'category',
      data: series.map((item) => formatDateLabel(item.date)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: '#e8eef4' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#e8eef4' } },
      axisTick: { show: false },
      axisLine: { show: false },
      axisLabel: { color: '#94a3b8', fontSize: 11 },
    },
    series: [
      {
        name: '提交',
        type: 'bar',
        barMaxWidth: 22,
        itemStyle: {
          color: '#c7ecf6',
          borderRadius: [8, 8, 0, 0],
        },
        emphasis: { itemStyle: { color: '#67d9ea' } },
        data: series.map((item) => readNumber(item.total)),
      },
      {
        name: '正确',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 8,
        lineStyle: { width: 4, color: '#0f766e' },
        itemStyle: { color: '#0f766e', borderColor: '#ffffff', borderWidth: 3 },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(15,118,110,0.18)' },
              { offset: 1, color: 'rgba(15,118,110,0.02)' },
            ],
          },
        },
        data: series.map((item) => readNumber(item.correct)),
      },
    ],
  };

  return (
    <div className="dashboard-combo-chart">
      <ReactECharts option={option} notMerge lazyUpdate className="dashboard-echart dashboard-echart-combo" />
    </div>
  );
}

function RadarPanel({ values }: { values: Record<(typeof radarAxes)[number]['key'], number> }) {
  const option: EChartsOption = {
    animationDuration: 700,
    radar: {
      radius: '62%',
      center: ['50%', '50%'],
      splitNumber: 3,
      axisName: {
        color: '#64748b',
        fontSize: 11,
        fontWeight: 700,
      },
      splitLine: { lineStyle: { color: '#dbe8ef' } },
      splitArea: { show: false },
      axisLine: { lineStyle: { color: '#edf4f8' } },
      indicator: radarAxes.map((axis) => ({ name: axis.label, max: 100 })),
    },
    series: [
      {
        type: 'radar',
        symbol: 'circle',
        symbolSize: 7,
        lineStyle: { width: 2, color: '#0f766e' },
        itemStyle: { color: '#0f766e' },
        areaStyle: { color: 'rgba(20, 184, 166, 0.22)' },
        data: [{ value: radarAxes.map((axis) => clamp(values[axis.key])) }],
      },
    ],
  };

  return (
    <div className="dashboard-radar-wrap">
      <ReactECharts option={option} notMerge lazyUpdate className="dashboard-echart dashboard-echart-radar" />
      <div className="dashboard-radar-list">
        {radarAxes.map((axis) => (
          <span key={axis.key}>
            {axis.label}
            <strong>{clamp(values[axis.key])}</strong>
          </span>
        ))}
      </div>
    </div>
  );
}

function GrowthAreaChart({ data, total }: { data: TrendItem[]; total: number }) {
  const series = completeTrendSeries(data, 14);
  if (!series.some((item) => readNumber(item.count))) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户增长" />;
  }

  const option: EChartsOption = {
    animationDuration: 700,
    grid: { top: 8, right: 8, bottom: 8, left: 8, containLabel: false },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: (params) => {
        const items = Array.isArray(params) ? params : [params];
        const label = readTooltipAxisLabel(items[0]);
        return `${label}<br/>新增 ${formatNumber(readNumber(items[0]?.value))}`;
      },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: series.map((item) => formatDateLabel(item.date)),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },
    series: [
      {
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 4, color: '#0f766e' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(20, 184, 166, 0.22)' },
              { offset: 1, color: 'rgba(20, 184, 166, 0.04)' },
            ],
          },
        },
        data: series.map((item) => readNumber(item.count)),
      },
    ],
  };

  return (
    <div className="dashboard-growth-chart">
      <div className="dashboard-growth-summary">
        <strong>{formatNumber(total)}</strong>
        <span>当前注册用户</span>
      </div>
      <ReactECharts option={option} notMerge lazyUpdate className="dashboard-echart dashboard-echart-growth" />
      <div className="dashboard-growth-days">
        <span>{formatDateLabel(series[0]?.date)}</span>
        <span>{formatDateLabel(series[series.length - 1]?.date)}</span>
      </div>
    </div>
  );
}

function StatusDistribution({ items }: { items: Array<{ label: string; value: number; tone: string }> }) {
  const total = Math.max(items.reduce((sum, item) => sum + item.value, 0), 1);

  const option: EChartsOption = {
    animationDuration: 700,
    tooltip: {
      trigger: 'item',
      formatter: (params) => {
        const item = params as { name?: string; value?: unknown; percent?: number };
        return `${item.name ?? '--'}<br/>${formatNumber(readNumber(item.value))} (${Math.round(item.percent ?? 0)}%)`;
      },
    },
    series: [
      {
        type: 'pie',
        radius: ['64%', '86%'],
        avoidLabelOverlap: false,
        label: { show: false },
        labelLine: { show: false },
        itemStyle: { borderColor: '#ffffff', borderWidth: 4 },
        data: items.map((item) => ({
          name: item.label,
          value: item.value,
          itemStyle: {
            color: item.tone === 'green' ? '#0f766e' : item.tone === 'cyan' ? '#06b6d4' : '#f59e0b',
          },
        })),
      },
    ],
  };

  return (
    <div className="dashboard-status-distribution">
      <div className="dashboard-donut">
        <ReactECharts option={option} notMerge lazyUpdate className="dashboard-echart dashboard-echart-donut" />
        <div>
          <strong>{formatNumber(total)}</strong>
          <span>赛事池</span>
        </div>
      </div>
      <div className="dashboard-distribution-list">
        {items.map((item) => (
          <div key={item.label} className={`dashboard-distribution-row tone-${item.tone}`}>
            <span>{item.label}</span>
            <strong>{formatNumber(item.value)}</strong>
            <i style={{ width: `${(item.value / total) * 100}%` }} />
          </div>
        ))}
      </div>
    </div>
  );
}

function QualityMatrix({ items }: { items: Array<{ label: string; value: number; text: string; icon: ReactNode }> }) {
  return (
    <div className="dashboard-quality-grid">
      {items.map((item) => (
        <div key={item.label} className="dashboard-quality-cell">
          <span>{item.icon}</span>
          <small>{item.label}</small>
          <strong>{item.text}</strong>
          <i style={{ width: `${clamp(item.value)}%` }} />
        </div>
      ))}
    </div>
  );
}

function RecentCompetitions({ data, onOpen }: { data: CompetitionItem[]; onOpen: (id: number | string) => void }) {
  if (!data.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无近期赛事" />;
  }
  return (
    <div className="dashboard-v2-list">
      {data.map((item, index) => (
        <button
          className="dashboard-v2-competition"
          key={`${item.id ?? index}-${index}`}
          type="button"
          onClick={() => item.id !== undefined && onOpen(item.id)}
        >
          <span className="dashboard-v2-list-icon">
            <TrophyOutlined />
          </span>
          <div>
            <strong>{item.name || item.title || '未命名赛事'}</strong>
            <small>{formatDateRange(item.start_time, item.end_time)}</small>
          </div>
          <Tag className={`dashboard-v2-status dashboard-v2-status-${item.status || 'draft'}`}>{statusLabel(item.status)}</Tag>
        </button>
      ))}
    </div>
  );
}

function HotChallenges({ data }: { data: ChallengeItem[] }) {
  if (!data.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无热门题目" />;
  }
  const max = Math.max(...data.map((item) => readNumber(item.submission_count)), 1);
  return (
    <div className="dashboard-v2-list">
      {data.slice(0, 6).map((item, index) => {
        const submissions = readNumber(item.submission_count);
        return (
          <div className="dashboard-v2-rank-row" key={`${item.id ?? index}-${index}`}>
            <span className="dashboard-v2-rank">{index + 1}</span>
            <div>
              <strong>{item.name || '未命名题目'}</strong>
              <small>
                {item.category || 'general'} · {formatNumber(readNumber(item.points))} pts · 解出 {formatNumber(readNumber(item.solve_count))}
              </small>
              <i>
                <b style={{ width: `${(submissions / max) * 100}%` }} />
              </i>
            </div>
            <em>{formatNumber(submissions)}</em>
          </div>
        );
      })}
    </div>
  );
}

function ActiveUsers({ data }: { data: ActiveUserItem[] }) {
  if (!data.length) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无活跃用户" />;
  }
  const max = Math.max(...data.map((item) => readNumber(item.submission_count)), 1);
  return (
    <div className="dashboard-v2-list">
      {data.slice(0, 6).map((item, index) => {
        const submissions = readNumber(item.submission_count);
        return (
          <div className="dashboard-v2-user" key={`${item.id ?? index}-${index}`}>
            <span>{(item.username || '?').slice(0, 1).toUpperCase()}</span>
            <div>
              <strong>{item.username || 'Unknown'}</strong>
              <small>
                提交 {formatNumber(submissions)} · 解出 {formatNumber(readNumber(item.solve_count))}
              </small>
              <i>
                <b style={{ width: `${(submissions / max) * 100}%` }} />
              </i>
            </div>
          </div>
        );
      })}
    </div>
  );
}

async function readDashboardEndpoint<T>(
  path: string,
  normalize: (value: unknown) => T,
  query?: Record<string, string | number | boolean | undefined>,
) {
  try {
    const payload = await adminApi.list<unknown>(path, query);
    return normalize(unwrapEnvelope(payload as ApiEnvelope<unknown>));
  } catch {
    return normalize(undefined);
  }
}

function unwrapEnvelope<T>(payload: ApiEnvelope<T>) {
  if (payload && typeof payload === 'object' && 'data' in payload) {
    return (payload as { data?: T }).data;
  }
  return payload as T;
}

function normalizeStats(value: unknown): DashboardStats {
  if (!value || typeof value !== 'object') {
    return {};
  }
  const raw = value as Record<string, unknown>;
  if (typeof raw.users === 'number' || typeof raw.competitions === 'number') {
    return {
      users: { total: readNumber(raw.users) },
      competitions: { total: readNumber(raw.competitions) },
      challenges: { total: readNumber(raw.challenges) },
      teams: { total: readNumber(raw.teams) },
      submissions: { total: readNumber(raw.submissions) },
    };
  }
  return value as DashboardStats;
}

function asArray<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

function completeTrendSeries(data: TrendItem[], days: number): TrendItem[] {
  const today = startOfLocalDay(new Date());
  const byDate = new Map<string, TrendItem>();

  data.forEach((item) => {
    const key = normalizeDateKey(item.date);
    if (key) {
      byDate.set(key, item);
    }
  });

  return Array.from({ length: days }, (_, index) => {
    const date = new Date(today);
    date.setDate(today.getDate() - (days - 1 - index));
    const key = toDateKey(date);
    return {
      date: key,
      total: 0,
      correct: 0,
      count: 0,
      ...byDate.get(key),
    };
  });
}

function normalizeDateKey(value?: string) {
  if (!value) return '';
  const date = new Date(value);
  if (!Number.isNaN(date.getTime())) return toDateKey(date);
  const match = value.match(/\d{4}-\d{2}-\d{2}/);
  return match?.[0] ?? '';
}

function startOfLocalDay(value: Date) {
  return new Date(value.getFullYear(), value.getMonth(), value.getDate());
}

function toDateKey(value: Date) {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, '0');
  const day = String(value.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function readNumber(value: unknown) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? numeric : 0;
}

function clamp(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)));
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(Math.round(value));
}

function formatPercent(value: number) {
  return `${clamp(value)}%`;
}

function formatDateLabel(value?: string) {
  if (!value) return '--';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value.slice(5, 10) || value;
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

function formatDateRange(start?: string, end?: string) {
  if (!start && !end) return '时间待定';
  return `${formatDateLabel(start)} - ${formatDateLabel(end)}`;
}

function statusLabel(status?: string) {
  const labels: Record<string, string> = {
    draft: '草稿',
    upcoming: '待开始',
    ongoing: '进行中',
    finished: '已结束',
    archived: '已归档',
  };
  return labels[status || 'draft'] || status || '草稿';
}

function readTooltipAxisLabel(param: unknown) {
  if (!param || typeof param !== 'object') {
    return '--';
  }
  const record = param as { axisValueLabel?: unknown; axisValue?: unknown; name?: unknown };
  return String(record.axisValueLabel ?? record.axisValue ?? record.name ?? '--');
}
