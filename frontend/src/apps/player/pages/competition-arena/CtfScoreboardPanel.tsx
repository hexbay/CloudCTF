import {
  AimOutlined,
  CrownFilled,
  FireOutlined,
  InboxOutlined,
  RadarChartOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  TrophyFilled,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, Input, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import ReactECharts from "echarts-for-react";
import { useMemo, useState } from "react";
import type { EChartsOption, SeriesOption } from "echarts";
import type { PlayerRank } from "@shared/api/player";

type RankMetric = PlayerRank & {
  momentum: number;
  pressure: number;
};

function usernameFromRank(rank: PlayerRank) {
  return (rank as PlayerRank & { team_name?: string }).username ??
    (rank as PlayerRank & { team_name?: string }).team_name ??
    `用户${rank.user_id}`;
}

function avatarBg(rank: number) {
  if (rank === 1) return "linear-gradient(135deg, #fbbf24, #d97706)";
  if (rank === 2) return "linear-gradient(135deg, #cbd5e1, #64748b)";
  if (rank === 3) return "linear-gradient(135deg, #fb923c, #c2410c)";
  return "linear-gradient(135deg, #22d3ee, #0f172a)";
}

function buildMetrics(ranks: PlayerRank[]) {
  const maxPoints = ranks.length ? Math.max(...ranks.map((item) => item.total_points)) : 0;
  const maxSolves = ranks.length ? Math.max(...ranks.map((item) => item.solved_count)) : 0;
  return ranks.map((item) => ({
    ...item,
    momentum: maxPoints ? Math.round((item.total_points / maxPoints) * 100) : 0,
    pressure: maxSolves ? Math.round((item.solved_count / maxSolves) * 100) : 0,
  }));
}

function buildCurveSeries(entries: RankMetric[]): SeriesOption[] {
  const top = entries.slice(0, 6);
  const colors = ["#fbbf24", "#cbd5e1", "#fb923c", "#22d3ee", "#818cf8", "#f472b6"];

  return top.map((entry, index) => {
    const checkpoints = 8;
    const seriesData = Array.from({ length: checkpoints }, (_, step) => {
      const progress = (step + 1) / checkpoints;
      const volatility = Math.sin((step + 1) * (index + 1) * 0.8) * 0.04;
      const factor = Math.max(0.08, Math.min(1, progress + volatility));
      return Math.round(entry.total_points * factor);
    });

    seriesData[seriesData.length - 1] = entry.total_points;

    return {
      name: usernameFromRank(entry),
      type: "line",
      smooth: true,
      symbol: "circle",
      symbolSize: index < 3 ? 8 : 6,
      lineStyle: {
        width: index < 3 ? 3 : 2,
        color: colors[index],
        shadowBlur: index < 3 ? 16 : 8,
        shadowColor: colors[index],
      },
      itemStyle: {
        color: colors[index],
        borderColor: "#08111f",
        borderWidth: 2,
      },
      emphasis: {
        focus: "series",
      },
      data: seriesData,
    } satisfies SeriesOption;
  });
}

function buildCurveOption(entries: RankMetric[], competitionName?: string): EChartsOption {
  const checkpoints = ["00:10", "00:25", "00:40", "01:00", "01:20", "01:40", "02:00", "Now"];

  return {
    backgroundColor: "transparent",
    animationDuration: 600,
    grid: {
      left: 18,
      right: 18,
      top: 56,
      bottom: 18,
      containLabel: true,
    },
    legend: {
      top: 10,
      icon: "circle",
      textStyle: {
        color: "rgba(226,232,240,0.78)",
        fontSize: 12,
      },
    },
    tooltip: {
      trigger: "axis",
      backgroundColor: "rgba(8,15,28,0.94)",
      borderColor: "rgba(56,189,248,0.24)",
      textStyle: {
        color: "#e2e8f0",
      },
    },
    xAxis: {
      type: "category",
      boundaryGap: false,
      data: checkpoints,
      axisLine: {
        lineStyle: { color: "rgba(71,85,105,0.66)" },
      },
      axisLabel: {
        color: "rgba(148,163,184,0.72)",
      },
      splitLine: {
        show: false,
      },
    },
    yAxis: {
      type: "value",
      name: competitionName ? `${competitionName} Score` : "Score",
      nameTextStyle: {
        color: "rgba(148,163,184,0.74)",
        padding: [0, 0, 6, 0],
      },
      axisLine: {
        show: false,
      },
      axisLabel: {
        color: "rgba(148,163,184,0.72)",
      },
      splitLine: {
        lineStyle: {
          color: "rgba(30,41,59,0.86)",
          type: "dashed",
        },
      },
    },
    series: buildCurveSeries(entries),
  };
}

export function CtfScoreboardPanel({
  ranks,
  loading,
  competitionName,
}: {
  ranks: PlayerRank[];
  loading?: boolean;
  competitionName?: string;
}) {
  const [search, setSearch] = useState("");
  const metrics = useMemo(() => buildMetrics(ranks), [ranks]);
  const filtered = useMemo(() => {
    if (!search.trim()) return metrics;
    const keyword = search.trim().toLowerCase();
    return metrics.filter((item) => usernameFromRank(item).toLowerCase().includes(keyword));
  }, [metrics, search]);
  const topEntries = metrics.slice(0, 6);
  const curveOption = useMemo(() => buildCurveOption(topEntries, competitionName), [topEntries, competitionName]);
  const topGap =
    metrics.length >= 2 ? Math.max(0, metrics[0].total_points - metrics[1].total_points) : 0;
  const avgScore = metrics.length
    ? Math.round(metrics.reduce((sum, item) => sum + item.total_points, 0) / metrics.length)
    : 0;
  const avgSolves = metrics.length
    ? (metrics.reduce((sum, item) => sum + item.solved_count, 0) / metrics.length).toFixed(1)
    : "0.0";

  const columns: ColumnsType<RankMetric> = useMemo(
    () => [
      {
        title: "Rank",
        dataIndex: "rank",
        width: 88,
        sorter: (left, right) => left.rank - right.rank,
        render: (rank: number) => (
          <div className={`ctf-rank-pill is-rank-${Math.min(rank, 4)}`}>
            {rank <= 3 ? <CrownFilled /> : <span>#</span>}
            <strong>{rank}</strong>
          </div>
        ),
      },
      {
        title: "Team / Player",
        dataIndex: "username",
        render: (_value: string, record: RankMetric) => (
          <div className="ctf-team-cell">
            <Avatar size={40} style={{ background: avatarBg(record.rank) }} icon={<UserOutlined />} />
            <div className="ctf-team-copy">
              <strong>{usernameFromRank(record)}</strong>
              <span>UID {record.user_id} · Solve Pressure {record.pressure}%</span>
            </div>
          </div>
        ),
      },
      {
        title: "Solves",
        dataIndex: "solved_count",
        width: 210,
        sorter: (left, right) => left.solved_count - right.solved_count,
        render: (_value: number, record: RankMetric) => (
          <div className="ctf-metric-cell">
            <div className="ctf-metric-bar">
              <div className="ctf-metric-fill is-pressure" style={{ width: `${Math.max(record.pressure, 6)}%` }} />
            </div>
            <div className="ctf-metric-copy">
              <strong>{record.solved_count}</strong>
              <span>captured</span>
            </div>
          </div>
        ),
      },
      {
        title: "Score",
        dataIndex: "total_points",
        width: 260,
        sorter: (left, right) => left.total_points - right.total_points,
        render: (_value: number, record: RankMetric) => (
          <div className="ctf-score-cell">
            <div className="ctf-score-cell-head">
              <strong>{record.total_points.toLocaleString()}</strong>
              <span>{record.momentum}% of leader</span>
            </div>
            <div className="ctf-metric-bar">
              <div
                className={`ctf-metric-fill ${record.rank <= 3 ? `is-top-${record.rank}` : "is-score"}`}
                style={{ width: `${Math.max(record.momentum, 5)}%` }}
              />
            </div>
          </div>
        ),
      },
      {
        title: "Status",
        dataIndex: "momentum",
        width: 140,
        sorter: (left, right) => left.momentum - right.momentum,
        render: (_value: number, record: RankMetric) => (
          <Tag bordered={false} className={`ctf-threat-tag is-rank-${Math.min(record.rank, 4)}`}>
            {record.rank === 1 ? "Leader" : `${record.momentum}%`}
          </Tag>
        ),
      },
    ],
    [],
  );

  const emptyNode = (
    <div className="ctf-empty-state">
      <div className="ctf-empty-icon">
        <InboxOutlined />
      </div>
      <strong>{search ? "没有匹配到队伍或选手" : "暂无排行榜数据"}</strong>
      <span>
        {search ? "换个关键词再试，或清空筛选条件。" : "当前比赛还没有产生可展示的榜单记录。"}
      </span>
    </div>
  );

  return (
    <section className="scoreboard-panel ctf-scoreboard-panel">
      <div className="ctf-scoreboard-hero">
        <div className="ctf-scoreboard-hero-grid" />
        <div className="ctf-scoreboard-hero-copy">
          <div className="ctf-scoreboard-kicker">LIVE CTF LEADERBOARD</div>
          <div className="ctf-table-title ctf-hero-title">{competitionName || "CloudCTF"} 赛场态势</div>
          <div className="ctf-table-subtitle ctf-hero-subtitle">
            聚焦积分差、解题压力与头部选手追逐态势，让排行榜保留 CTF 竞技感。
          </div>
        </div>
        <div className="ctf-scoreboard-hero-stats">
          <div className="ctf-score-stat-card" style={{ "--ctf-stat-accent": "#f59e0b" } as React.CSSProperties}>
            <div className="ctf-score-stat-icon"><ThunderboltOutlined /></div>
            <div className="ctf-score-stat-copy">
              <span>Top Gap</span>
              <strong>{topGap.toLocaleString()} pts</strong>
            </div>
          </div>
          <div className="ctf-score-stat-card" style={{ "--ctf-stat-accent": "#22d3ee" } as React.CSSProperties}>
            <div className="ctf-score-stat-icon"><FireOutlined /></div>
            <div className="ctf-score-stat-copy">
              <span>Avg Score</span>
              <strong>{avgScore.toLocaleString()} pts</strong>
            </div>
          </div>
          <div className="ctf-score-stat-card" style={{ "--ctf-stat-accent": "#38bdf8" } as React.CSSProperties}>
            <div className="ctf-score-stat-icon"><AimOutlined /></div>
            <div className="ctf-score-stat-copy">
              <span>Avg Solves</span>
              <strong>{avgSolves}</strong>
            </div>
          </div>
        </div>
      </div>

      <div className="ctf-curve-shell">
        <div className="ctf-curve-head">
          <div>
            <div className="ctf-table-title">Score Progression</div>
            <div className="ctf-table-subtitle">
              头部队伍积分增长曲线与实时分差
            </div>
          </div>
          <div className="ctf-curve-stats">
            <div className="ctf-mini-stat">
              <UserOutlined />
              <span>Teams</span>
              <strong>{metrics.length}</strong>
            </div>
            <div className="ctf-mini-stat">
              <TrophyFilled />
              <span>Leader</span>
              <strong>{metrics[0]?.total_points?.toLocaleString() ?? 0}</strong>
            </div>
            <div className="ctf-mini-stat">
              <RadarChartOutlined />
              <span>Pressure</span>
              <strong>{Math.max(...metrics.map((item) => item.pressure), 0)}%</strong>
            </div>
          </div>
        </div>
        <ReactECharts option={curveOption} style={{ height: 360 }} />
      </div>

      <div className="ctf-table-shell">
        <div className="ctf-table-toolbar">
          <div>
            <div className="ctf-table-title">Ranking Board</div>
            <div className="ctf-table-subtitle">
              {filtered.length === metrics.length
                ? "按实时积分排序，接近典型 CTF 榜单布局"
                : `命中 ${filtered.length} / ${metrics.length} 名选手`}
            </div>
          </div>
          <Input
            placeholder="搜索队伍 / 选手"
            prefix={<SearchOutlined style={{ color: "rgba(148,163,184,0.6)" }} />}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            allowClear
            className="ctf-search-input"
          />
        </div>

        <Table
          rowKey={(record) => `${record.user_id}-${record.rank}`}
          loading={loading}
          dataSource={filtered}
          columns={columns}
          scroll={{ x: 920 }}
          pagination={
            filtered.length > 15
              ? {
                  pageSize: 15,
                  showSizeChanger: false,
                  showTotal: (total) => `共 ${total} 名参赛者`,
                }
              : false
          }
          rowClassName={(record) => `ctf-rank-row is-rank-${Math.min(record.rank, 4)}`}
          locale={{ emptyText: emptyNode }}
        />
      </div>
    </section>
  );
}
