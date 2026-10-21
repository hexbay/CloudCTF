import {
  FireOutlined,
  FlagOutlined,
  RiseOutlined,
  StarFilled,
  ThunderboltOutlined,
  TrophyOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Avatar, Card, Empty, Segmented, Table, Tag, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { playerApi, type ChallengeRank, type PlayerRank } from '@shared/api/player';
import { PlayerPage } from './PlayerPage';

type ScoreboardTabKey = 'users' | 'challenges';

export function PlayerScoreboardPage() {
  const [activeTab, setActiveTab] = useState<ScoreboardTabKey>('users');
  const [users, setUsers] = useState<PlayerRank[]>([]);
  const [challenges, setChallenges] = useState<ChallengeRank[]>([]);
  const [myRank, setMyRank] = useState<PlayerRank | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void Promise.all([
      playerApi
        .practiceLeaderboard({ limit: 100 })
        .then((res) => setUsers(res.data ?? []))
        .catch(() => setUsers([])),
      playerApi
        .challengeLeaderboard({ limit: 100 })
        .then((res) => setChallenges(res.data ?? []))
        .catch(() => setChallenges([])),
      playerApi.myRank().then(setMyRank).catch(() => setMyRank(null)),
    ]).finally(() => setLoading(false));
  }, []);

  const topUsers = users.slice(0, 3);
  const hottestChallenges = challenges.slice(0, 3);
  const leader = users[0];
  const averageScore = users.length
    ? Math.round(users.reduce((sum, item) => sum + item.total_points, 0) / users.length)
    : 0;
  const totalSolves = users.reduce((sum, item) => sum + item.solved_count, 0);
  const leaderboardReady = users.length > 0 || challenges.length > 0;

  const userColumns = useMemo(
    () => [
      {
        title: '排名',
        dataIndex: 'rank',
        width: 110,
        render: (rank: number) => <div className={`scoreboard-rank-badge is-rank-${Math.min(rank, 4)}`}>{rank}</div>,
      },
      {
        title: '用户',
        dataIndex: 'username',
        render: (username: string, record: PlayerRank) => (
          <div className="scoreboard-user-cell">
            <Avatar
              size={42}
              icon={<UserOutlined />}
              className={`scoreboard-user-avatar is-rank-${Math.min(record.rank, 4)}`}
            />
            <div className="scoreboard-user-copy">
              <strong>{username}</strong>
              <span>UID {record.user_id}</span>
            </div>
          </div>
        ),
      },
      {
        title: '解题数',
        dataIndex: 'solved_count',
        width: 160,
        render: (value: number) => (
          <div className="scoreboard-value-stack">
            <strong>{value}</strong>
            <span>solved</span>
          </div>
        ),
      },
      {
        title: '得分',
        dataIndex: 'total_points',
        width: 180,
        render: (value: number) => <span className="scoreboard-score-text">{value}</span>,
      },
    ],
    [],
  );

  const challengeColumns = useMemo(
    () => [
      {
        title: '热度',
        dataIndex: 'rank',
        width: 110,
        render: (rank: number) => <div className={`scoreboard-rank-badge is-rank-${Math.min(rank, 4)}`}>{rank}</div>,
      },
      {
        title: '题目',
        dataIndex: 'challenge_name',
        render: (value: string) => (
          <div className="scoreboard-challenge-cell">
            <div className="scoreboard-challenge-icon">
              <FlagOutlined />
            </div>
            <strong>{value}</strong>
          </div>
        ),
      },
      {
        title: '分类',
        dataIndex: 'category',
        width: 140,
        render: (value: string) => <Tag className="scoreboard-soft-tag">{value}</Tag>,
      },
      {
        title: '难度',
        dataIndex: 'difficulty',
        width: 130,
        render: renderDifficulty,
      },
      {
        title: '分值',
        dataIndex: 'points',
        width: 120,
      },
      {
        title: '解出人数',
        dataIndex: 'solved_count',
        width: 140,
      },
      {
        title: '提交次数',
        dataIndex: 'total_submissions',
        width: 140,
      },
    ],
    [],
  );

  return (
    <PlayerPage icon={<TrophyOutlined />} title="排行榜">
      {!leaderboardReady && !loading ? (
        <Card className="player-empty-card player-scoreboard-empty-state">
          <div className="player-scoreboard-empty-grid" />
          <div className="player-scoreboard-empty-orb player-scoreboard-empty-orb-a" />
          <div className="player-scoreboard-empty-orb player-scoreboard-empty-orb-b" />

          <div className="player-scoreboard-empty-badge">Ranking Deck</div>

          <div className="player-scoreboard-empty-icon">
            <TrophyOutlined />
          </div>

          <Typography.Title level={2} className="player-scoreboard-empty-title">
            排行榜大厅暂时还没亮起来
          </Typography.Title>

          <Typography.Paragraph className="player-scoreboard-empty-description">
            当前还没有可展示的排行数据。等练习成绩、题目热度和用户分数积累后，这里会生成完整的用户榜与题目榜视图。
          </Typography.Paragraph>

          <div className="player-scoreboard-empty-features">
            <div className="player-scoreboard-empty-feature">
              <RiseOutlined className="player-scoreboard-empty-feature-icon" />
              <div>
                <strong>实时排行</strong>
                <span>用户得分与解题进度会在这里动态更新</span>
              </div>
            </div>
            <div className="player-scoreboard-empty-feature">
              <FireOutlined className="player-scoreboard-empty-feature-icon" />
              <div>
                <strong>题目热度追踪</strong>
                <span>可快速看到哪些题目最热门、解出人数最多</span>
              </div>
            </div>
            <div className="player-scoreboard-empty-feature">
              <UserOutlined className="player-scoreboard-empty-feature-icon" />
              <div>
                <strong>我的排名入口</strong>
                <span>上线后可直接查看个人名次与当前分数位置</span>
              </div>
            </div>
          </div>
        </Card>
      ) : (
        <section className="scoreboard-page-shell">
          <section className="scoreboard-hero">
            <div className="scoreboard-hero-grid" />
            <div className="scoreboard-hero-copy">
              <div className="scoreboard-kicker">Practice Intelligence Board</div>
              <Typography.Title level={2} className="scoreboard-hero-title">
                排行榜
              </Typography.Title>
              <Typography.Paragraph className="scoreboard-hero-subtitle">
                查看当前积分排名、个人位置和题目热度。
              </Typography.Paragraph>
            </div>
            <div className="scoreboard-hero-stats">
              <div className="scoreboard-stat-card is-highlight">
                <span>我的排名</span>
                <strong>{myRank?.rank ?? '-'}</strong>
                <em>{myRank ? `${myRank.total_points} 分` : '暂无个人数据'}</em>
              </div>
              <div className="scoreboard-stat-card">
                <span>榜单用户</span>
                <strong>{users.length}</strong>
                <em>平均 {averageScore} 分</em>
              </div>
              <div className="scoreboard-stat-card">
                <span>累计解题</span>
                <strong>{totalSolves}</strong>
                <em>覆盖 {challenges.length} 道题目</em>
              </div>
            </div>
          </section>

          <section className="scoreboard-overview-grid">
            <Card className="scoreboard-podium-card">
              <div className="scoreboard-panel-head">
                <div>
                  <span className="scoreboard-panel-kicker">Top 3</span>
                  <h3>头部选手</h3>
                </div>
                <TrophyOutlined />
              </div>
              <div className="scoreboard-podium-list">
                {topUsers.map((user, index) => (
                  <div key={user.user_id} className={`scoreboard-podium-item is-rank-${index + 1}`}>
                    <div className="scoreboard-podium-order">
                      <StarFilled />
                      <span>#{user.rank}</span>
                    </div>
                    <div className="scoreboard-podium-copy">
                      <strong>{user.username}</strong>
                      <span>{user.solved_count} 题 / {user.total_points} 分</span>
                    </div>
                  </div>
                ))}
              </div>
            </Card>

            <Card className="scoreboard-glance-card">
              <div className="scoreboard-panel-head">
                <div>
                  <span className="scoreboard-panel-kicker">Hot Challenges</span>
                  <h3>当前热门题目</h3>
                </div>
                <FlagOutlined />
              </div>
              <div className="scoreboard-glance-list">
                {hottestChallenges.map((challenge) => (
                  <div key={challenge.challenge_id} className="scoreboard-glance-item">
                    <div>
                      <strong>{challenge.challenge_name}</strong>
                      <span>{challenge.category} · {challenge.solved_count} 人解出</span>
                    </div>
                    <Tag className="scoreboard-soft-tag">{challenge.points} pts</Tag>
                  </div>
                ))}
              </div>
            </Card>

            <Card className="scoreboard-leader-card">
              <div className="scoreboard-panel-head">
                <div>
                  <span className="scoreboard-panel-kicker">Leader Signal</span>
                  <h3>领跑状态</h3>
                </div>
                <ThunderboltOutlined />
              </div>
              <div className="scoreboard-leader-core">
                <strong>{leader?.username ?? '暂无'}</strong>
                <span>{leader ? `当前领先 ${leader.total_points} 分，已解出 ${leader.solved_count} 道题。` : '等待榜单刷新。'}</span>
              </div>
              <div className="scoreboard-leader-metrics">
                <div>
                  <span>第一名得分</span>
                  <strong>{leader?.total_points ?? 0}</strong>
                </div>
                <div>
                  <span>第一名解题</span>
                  <strong>{leader?.solved_count ?? 0}</strong>
                </div>
              </div>
            </Card>
          </section>

          <Card className="scoreboard-table-panel">
            <div className="scoreboard-table-toolbar">
              <div>
                <span className="scoreboard-panel-kicker">Ranking Board</span>
                <h3>{activeTab === 'users' ? '用户榜' : '题目榜'}</h3>
                <p>{activeTab === 'users' ? '关注选手当前积分和解题情况。' : '观察题目热度、分类和解出分布。'}</p>
              </div>
              <Segmented<ScoreboardTabKey>
                value={activeTab}
                onChange={(value) => setActiveTab(value)}
                options={[
                  { label: '用户榜', value: 'users' },
                  { label: '题目榜', value: 'challenges' },
                ]}
                className="scoreboard-segmented"
              />
            </div>

            {activeTab === 'users' ? (
              <Table
                rowKey="user_id"
                loading={loading}
                dataSource={users}
                columns={userColumns}
                pagination={{ pageSize: 8, showSizeChanger: false }}
                scroll={{ x: 760 }}
                className="scoreboard-table"
                locale={{ emptyText: <Empty description="暂无排行数据" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              />
            ) : (
              <Table
                rowKey="challenge_id"
                loading={loading}
                dataSource={challenges}
                columns={challengeColumns}
                pagination={{ pageSize: 8, showSizeChanger: false }}
                scroll={{ x: 980 }}
                className="scoreboard-table"
                locale={{ emptyText: <Empty description="暂无题目排行数据" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              />
            )}
          </Card>
        </section>
      )}
    </PlayerPage>
  );
}

function renderDifficulty(value: number) {
  const labels = ['入门', '简单', '中等', '困难', '极难'];
  const classNames = ['is-easy', 'is-easy', 'is-medium', 'is-hard', 'is-expert'];
  return <Tag className={`scoreboard-soft-tag ${classNames[value - 1] ?? 'is-hard'}`}>{labels[value - 1] ?? '未知'}</Tag>;
}
