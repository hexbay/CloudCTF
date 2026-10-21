import { BookOutlined, InboxOutlined, SearchOutlined, TrophyOutlined, UserOutlined } from "@ant-design/icons";
import { Avatar, Empty, Input, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo, useState } from "react";
import type { PlayerRank } from "@shared/api/player";
import "./TheoryScoreboardPanel.css";

type TheoryRankRow = PlayerRank & {
  accuracy: number;
};

function theoryName(row: PlayerRank) {
  return (row as PlayerRank & { team_name?: string }).username ??
    (row as PlayerRank & { team_name?: string }).team_name ??
    `用户${row.user_id}`;
}

function buildTheoryRows(ranks: PlayerRank[]) {
  const maxSolves = ranks.length ? Math.max(...ranks.map((item) => item.solved_count), 0) : 0;
  return ranks.map((item) => ({
    ...item,
    accuracy: maxSolves ? Math.round((item.solved_count / maxSolves) * 100) : 0,
  }));
}

function avatarGradient(rank: number) {
  if (rank === 1) return "linear-gradient(135deg, #f59e0b, #b45309)";
  if (rank === 2) return "linear-gradient(135deg, #94a3b8, #475569)";
  if (rank === 3) return "linear-gradient(135deg, #fb7185, #be123c)";
  return "linear-gradient(135deg, #0f766e, #0284c7)";
}

export function TheoryExamScoreboardPanel({
  ranks,
  loading,
  competitionName,
}: {
  ranks: PlayerRank[];
  loading?: boolean;
  competitionName?: string;
}) {
  const [search, setSearch] = useState("");
  const rows = useMemo(() => buildTheoryRows(ranks), [ranks]);
  const filtered = useMemo(() => {
    if (!search.trim()) return rows;
    const keyword = search.trim().toLowerCase();
    return rows.filter((item) => theoryName(item).toLowerCase().includes(keyword));
  }, [rows, search]);

  const avgScore = rows.length
    ? Math.round(rows.reduce((sum, item) => sum + item.total_points, 0) / rows.length)
    : 0;
  const avgCorrect = rows.length
    ? (rows.reduce((sum, item) => sum + item.solved_count, 0) / rows.length).toFixed(1)
    : "0.0";
  const leader = rows[0];

  const columns: ColumnsType<TheoryRankRow> = [
    {
      title: "排名",
      dataIndex: "rank",
      width: 90,
      render: (rank: number) => (
        <Tag bordered={false} className={`theory-rank-tag is-rank-${Math.min(rank, 4)}`}>
          #{rank}
        </Tag>
      ),
    },
    {
      title: "选手",
      dataIndex: "username",
      render: (_value, record) => (
        <div className="theory-player-cell">
          <Avatar
            size={42}
            className="theory-player-avatar"
            style={{ background: avatarGradient(record.rank) }}
            icon={<UserOutlined />}
          />
          <div className="theory-player-copy">
            <div className="theory-player-name">{theoryName(record)}</div>
            <div className="theory-player-meta">UID {record.user_id}</div>
          </div>
        </div>
      ),
    },
    {
      title: "理论得分",
      dataIndex: "total_points",
      width: 140,
      sorter: (left, right) => left.total_points - right.total_points,
      render: (value: number) => <span className="theory-score-strong">{value.toLocaleString()}</span>,
    },
    {
      title: "答对题数",
      dataIndex: "solved_count",
      width: 180,
      sorter: (left, right) => left.solved_count - right.solved_count,
      render: (value: number, record) => (
        <div className="theory-correct-cell">
          <span>{value}</span>
          <div className="theory-correct-bar">
            <div className="theory-correct-fill" style={{ width: `${Math.max(record.accuracy, 4)}%` }} />
          </div>
          <Tag bordered={false} className="theory-accuracy-tag">{record.accuracy}%</Tag>
        </div>
      ),
    },
  ];

  return (
    <section className="theory-board-page">
      <div className="theory-board-shell">
        <div className="theory-board-hero">
          <div className="theory-board-hero-content">
            <div className="theory-board-hero-head">
            <div>
              <div className="theory-board-kicker">Theory Ranking</div>
              <Typography.Title level={2} className="theory-board-title">
                {competitionName || "理论赛"} 成绩榜
              </Typography.Title>
              <Typography.Paragraph className="theory-board-copy">
                理论赛按答题得分和正确题数展示排名。
              </Typography.Paragraph>
            </div>
            <div className="theory-board-badges">
              <Tag className="theory-board-badge is-theory" bordered={false} icon={<BookOutlined />}>理论赛</Tag>
              <Tag className="theory-board-badge is-rank" bordered={false} icon={<TrophyOutlined />}>按成绩排序</Tag>
            </div>
            </div>

            <div className="theory-board-summary">
              <div className="theory-board-stat">
                <div className="theory-board-stat-label">当前第一名</div>
                <div className="theory-board-stat-value">{leader ? `#${leader.rank}` : "--"}</div>
                <div className="theory-board-stat-note">{leader ? theoryName(leader) : "暂无成绩记录"}</div>
              </div>
              <div className="theory-board-stat">
                <div className="theory-board-stat-label">平均得分</div>
                <div className="theory-board-stat-value">{avgScore}</div>
                <div className="theory-board-stat-note">基于当前已交卷选手</div>
              </div>
              <div className="theory-board-stat">
                <div className="theory-board-stat-label">平均答对题数</div>
                <div className="theory-board-stat-value">{avgCorrect}</div>
                <div className="theory-board-stat-note">反映整体完成质量</div>
              </div>
            </div>

            <div className="theory-board-toolbar">
              <Input.Search
                className="theory-board-search"
                placeholder="搜索选手"
                prefix={<SearchOutlined />}
                allowClear
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
            </div>
          </div>
        </div>

        <div className="theory-board-table-wrap">
          <div className="theory-board-table-head">
            <div>
              <div className="theory-board-table-title">Theory Ranking Board</div>
              <div className="theory-board-table-copy">
                {filtered.length === rows.length
                  ? "按理论答题得分实时排序，并展示当前答题正确率。"
                  : `命中 ${filtered.length} / ${rows.length} 名选手`}
              </div>
            </div>
          </div>
          <div className="theory-board-table">
          <Table
            rowKey={(record) => `${record.user_id}-${record.rank}`}
            loading={loading}
            columns={columns}
            dataSource={filtered}
            pagination={filtered.length > 10 ? { pageSize: 10, showSizeChanger: false } : false}
            rowClassName={(record) => `theory-rank-row is-rank-${Math.min(record.rank, 4)}`}
            locale={{
              emptyText: (
                <div className="theory-board-empty">
                  <div className="theory-board-empty-icon">
                    <InboxOutlined />
                  </div>
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={false} />
                  <div className="theory-board-empty-title">
                    {search ? "没有匹配到选手" : "当前理论赛还没有成绩记录"}
                  </div>
                  <div className="theory-board-empty-copy">
                    {search ? "请尝试其他关键词。" : "等待首批选手完成交卷后，这里将生成成绩单。"}
                  </div>
                </div>
              ),
            }}
          />
          </div>
        </div>
      </div>
    </section>
  );
}
