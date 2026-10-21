import {
  AppstoreOutlined,
  ArrowRightOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  FireOutlined,
  RocketOutlined,
  TeamOutlined,
  TrophyOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import { Button, Card, Col, Row, Spin, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { playerApi, type PlayerCompetition } from '@shared/api/player';
import { PlayerPage } from './PlayerPage';

const statusNames: Record<string, string> = {
  ongoing: '进行中',
  upcoming: '即将开始',
  finished: '已结束',
  draft: '草稿',
  archived: '已归档',
};

const typeNames: Record<string, string> = {
  ctf: 'CTF',
  awd: 'AWD',
  theory: '理论赛',
  mixed: '综合赛',
  king_of_the_hill: 'KOH',
  attack_defense: 'AWD',
  jeopardy: 'Jeopardy',
};

const participantModeNames: Record<string, string> = {
  individual: '个人赛',
  team: '团队赛',
};

const previewEmptyState = false;

export function PlayerCompetitionPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [data, setData] = useState<PlayerCompetition[]>([]);
  const [loading, setLoading] = useState(true);
  const activeStatus = params.get('status') || 'ongoing';

  useEffect(() => {
    setLoading(true);
    const status = activeStatus === 'all' ? undefined : activeStatus;
    playerApi
      .competitions({ current: 1, pageSize: 20, status })
      .then((res) => setData(res.data ?? []))
      .catch(() => setData([]))
      .finally(() => setLoading(false));
  }, [activeStatus]);

  const visible = useMemo(
    () =>
      activeStatus === 'all'
        ? data
        : data.filter((item) => String(item.status).toLowerCase() === activeStatus),
    [activeStatus, data],
  );

  const showEmptyState = !loading && (previewEmptyState || visible.length === 0);

  const setStatus = (status: string) => {
    setParams({ status });
  };

  return (
    <PlayerPage
      icon={<TrophyOutlined />}
      title="比赛列表"
      subtitle="公开比赛列表和入口"
      extra={
        <div className="player-button-group">
          <Button
            type={activeStatus === 'all' ? 'primary' : 'default'}
            icon={<AppstoreOutlined />}
            onClick={() => setStatus('all')}
          >
            全部比赛
          </Button>
          <Button
            type={activeStatus === 'ongoing' ? 'primary' : 'default'}
            icon={<FireOutlined />}
            onClick={() => setStatus('ongoing')}
          >
            进行中
          </Button>
          <Button
            type={activeStatus === 'upcoming' ? 'primary' : 'default'}
            icon={<CalendarOutlined />}
            onClick={() => setStatus('upcoming')}
          >
            即将开始
          </Button>
          <Button
            type={activeStatus === 'finished' ? 'primary' : 'default'}
            icon={<TrophyOutlined />}
            onClick={() => setStatus('finished')}
          >
            已结束
          </Button>
        </div>
      }
    >
      <Card className="player-panel-card">
        <Typography.Title level={4}>比赛入口</Typography.Title>
        <Spin spinning={loading}>
          {!showEmptyState ? (
            <Row gutter={[16, 16]}>
              {visible.map((competition) => {
                const competitionStatus = String(competition.status).toLowerCase();
                const isDraft = competitionStatus === 'draft';

                return (
                <Col xs={24} sm={12} md={8} xl={6} key={competition.id}>
                  <div
                    className="player-competition-card"
                    onClick={() => {
                      if (isDraft) {
                        return;
                      }
                      navigate(`/competition-arena/overview?id=${competition.id}`);
                    }}
                    style={{ cursor: isDraft ? 'not-allowed' : 'pointer', opacity: isDraft ? 0.72 : 1 }}
                  >
                    <div
                      className={`competition-cover competition-cover-${competitionStatus}`}
                      style={getCompetitionCoverStyle(competition)}
                    >
                      <div className="competition-cover-overlay" />
                      <div className="competition-cover-badges">
                        <span className={`competition-chip competition-chip-type competition-type-${normalizeCompetitionType(competition.type)}`}>
                          {typeNames[normalizeCompetitionType(competition.type)] ?? competition.type ?? 'CTF'}
                        </span>
                        <span className={`competition-chip competition-chip-status competition-status-${competitionStatus}`}>
                          {statusNames[competition.status] ?? competition.status}
                        </span>
                      </div>
                      {!competition.banner_url ? (
                        <div className="competition-cover-fallback">
                          <span className="competition-cover-monogram">
                            {getCompetitionMonogram(competition.name)}
                          </span>
                          <span className="competition-cover-kicker">
                            {typeNames[normalizeCompetitionType(competition.type)] ?? 'Competition'}
                          </span>
                        </div>
                      ) : null}
                      <div className="competition-cover-enter">
                        {isDraft ? '草稿未开放' : '进入比赛'} {!isDraft ? <ArrowRightOutlined /> : null}
                      </div>
                    </div>
                    <div className="competition-body">
                      <div className="competition-title">{competition.name}</div>
                      <div className="competition-meta">
                        <CalendarOutlined /> 开始：{formatDate(competition.start_time)}
                      </div>
                      <div className="competition-meta">
                        <ClockCircleOutlined /> 结束：{formatDate(competition.end_time)}
                      </div>
                      <div className="competition-meta">
                        <UnlockOutlined /> {competition.is_public === false ? '私有比赛' : '公开比赛'}
                      </div>
                      <div className="competition-meta">
                        <TeamOutlined /> {participantModeNames[String(competition.participant_mode ?? 'individual')] ?? '个人赛'}
                      </div>
                      {competition.max_team_members ? (
                        <div className="competition-meta">
                          <TeamOutlined /> 最多 {competition.max_team_members} 人组队
                        </div>
                      ) : null}
                    </div>
                  </div>
                </Col>
                );
              })}
            </Row>
          ) : (
            <div className="player-competition-empty-state">
              <div className="player-competition-empty-grid" />
              <div className="player-competition-empty-orb player-competition-empty-orb-a" />
              <div className="player-competition-empty-orb player-competition-empty-orb-b" />

              <div className="player-competition-empty-badge">Competition Dock</div>

              <div className="player-competition-empty-icon">
                <TrophyOutlined />
              </div>

              <Typography.Title level={2} className="player-competition-empty-title">
                比赛大厅暂时还没开放
              </Typography.Title>

              <Typography.Paragraph className="player-competition-empty-description">
                当前没有可报名或可进入的公开比赛。等赛事发布后，你可以在这里查看赛程、状态和入口，直接进入比赛空间。
              </Typography.Paragraph>

              <div className="player-competition-empty-features">
                <div className="player-competition-empty-feature">
                  <RocketOutlined className="player-competition-empty-feature-icon" />
                  <div>
                    <strong>快速入场</strong>
                    <span>比赛开启后可直接从这里进入赛事界面</span>
                  </div>
                </div>
                <div className="player-competition-empty-feature">
                  <CalendarOutlined className="player-competition-empty-feature-icon" />
                  <div>
                    <strong>赛程清晰</strong>
                    <span>进行中、即将开始、已结束状态一眼区分</span>
                  </div>
                </div>
                <div className="player-competition-empty-feature">
                  <TeamOutlined className="player-competition-empty-feature-icon" />
                  <div>
                    <strong>支持组队</strong>
                    <span>公开赛和组队人数限制都会在这里同步展示</span>
                  </div>
                </div>
              </div>

              <div className="player-competition-empty-actions">
                <Button type="primary" size="large" onClick={() => navigate('/challenges')}>
                  先去练习
                </Button>
              </div>
            </div>
          )}
        </Spin>
      </Card>
    </PlayerPage>
  );
}

function formatDate(value?: string) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function normalizeCompetitionType(value?: string) {
  return String(value ?? 'ctf').toLowerCase().replace(/\s+/g, '_');
}

function getCompetitionMonogram(name?: string) {
  const words = String(name ?? '')
    .trim()
    .split(/\s+/)
    .filter(Boolean);

  if (!words.length) return 'CTF';
  if (words.length === 1) return words[0].slice(0, 3).toUpperCase();
  return words
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? '')
    .join('');
}

function getCompetitionCoverStyle(competition: PlayerCompetition) {
  if (!competition.banner_url) return undefined;
  return {
    backgroundImage: `linear-gradient(135deg, rgba(7, 14, 28, 0.22), rgba(15, 23, 42, 0.58)), url(${competition.banner_url})`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
  };
}
