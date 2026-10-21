import {
  FireOutlined,
  FlagOutlined,
  RocketOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Col,
  Input,
  Row,
  Spin,
  Tabs,
  Tag,
  Typography,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { playerApi, type PlayerChallenge } from "@shared/api/player";
import { ChallengeQuickModal } from "../components/ChallengeQuickModal";
import { difficultyText } from "./PlayerDashboardPage";
import { PlayerPage } from "./PlayerPage";

const categories = ["Web", "Crypto", "Pwn", "Reverse", "Misc"];
const previewEmptyState = false;

export function PlayerChallengesPage() {
  const navigate = useNavigate();
  const [activeCategory, setActiveCategory] = useState("all");
  const [keyword, setKeyword] = useState("");
  const [data, setData] = useState<PlayerChallenge[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedChallenge, setSelectedChallenge] =
    useState<PlayerChallenge | null>(null);

  useEffect(() => {
    playerApi
      .challenges()
      .then((res) => setData(res.data ?? []))
      .catch(() => setData([]))
      .finally(() => setLoading(false));
  }, []);

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase();
    return data.filter((item) => {
      const categoryOk =
        activeCategory === "all" ||
        String(item.category).toLowerCase() === activeCategory.toLowerCase();
      const keywordOk =
        !k ||
        item.name.toLowerCase().includes(k) ||
        (item.description ?? "").toLowerCase().includes(k);
      return categoryOk && keywordOk;
    });
  }, [activeCategory, data, keyword]);

  const showEmptyState =
    !loading && (previewEmptyState || filtered.length === 0);
  const solvedCount = useMemo(
    () => data.filter((item) => item.is_solved).length,
    [data],
  );
  const availablePoints = useMemo(
    () => data.reduce((sum, item) => sum + Number(item.points || 0), 0),
    [data],
  );

  return (
    <PlayerPage
      icon={<FlagOutlined />}
      title="练习题目"
      subtitle="按方向筛选题目，持续积累解题经验与积分。"
      showTitle
    >
      <div className="player-challenges-page">
        <div className="player-challenges-overview" aria-label="题库概览">
          <div className="player-challenges-overview-item">
            <span>题目总数</span>
            <strong>{data.length}</strong>
            <FlagOutlined aria-hidden="true" />
          </div>
          <div className="player-challenges-overview-item is-solved">
            <span>已解题目</span>
            <strong>{solvedCount}</strong>
            <RocketOutlined aria-hidden="true" />
          </div>
          <div className="player-challenges-overview-item is-points">
            <span>可得积分</span>
            <strong>{availablePoints}</strong>
            <TrophyOutlined aria-hidden="true" />
          </div>
        </div>

        <div className="player-filter-bar player-challenges-filter">
          <Tabs
            activeKey={activeCategory}
            onChange={setActiveCategory}
            items={[
              { key: "all", label: "全部" },
              ...categories.map((key) => ({ key, label: key })),
            ]}
          />
          <Input
            className="player-challenges-search"
            placeholder="搜索关卡"
            prefix={<SearchOutlined />}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            allowClear
          />
          <div className="player-challenges-result-count">
            当前显示 <strong>{filtered.length}</strong> 道题
          </div>
        </div>

        {loading ? (
          <div className="player-loading">
            <Spin size="large" />
          </div>
        ) : !showEmptyState ? (
          <Row gutter={[18, 18]} className="player-challenges-grid">
            {filtered.map((challenge, index) => (
              <Col
                xs={24}
                sm={12}
                lg={8}
                key={challenge.id}
                className="player-challenges-grid-item"
                style={{ animationDelay: `${Math.min(index, 8) * 60}ms` }}
              >
                <Card
                  className={`player-challenge-card ${challenge.is_solved ? "is-solved" : ""}`}
                  hoverable
                  onClick={() => setSelectedChallenge(challenge)}
                >
                  <div className="challenge-card-accent" />
                  {challenge.is_solved ? (
                    <div
                      className="challenge-card-solved-mark"
                      aria-label="已解决"
                    >
                      <FlagOutlined />
                    </div>
                  ) : null}
                  <div className="challenge-card-tags">
                    <Tag
                      className={`challenge-difficulty-tag difficulty-${difficultyText(challenge.difficulty)}`}
                    >
                      {difficultyText(challenge.difficulty)}
                    </Tag>
                    <Tag
                      className={`challenge-category-tag category-${String(challenge.category).toLowerCase()}`}
                    >
                      {challenge.category}
                    </Tag>
                  </div>
                  <div className="challenge-card-main">
                    <Typography.Title level={4}>
                      {challenge.name}
                    </Typography.Title>
                    <Typography.Paragraph ellipsis={{ rows: 2 }}>
                      {challenge.description || "暂无描述"}
                    </Typography.Paragraph>
                  </div>
                  <div className="challenge-card-footer">
                    <span>
                      <TrophyOutlined /> {challenge.points} 分
                    </span>
                    <span className="challenge-card-open">查看详情</span>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        ) : (
          <Card className="player-empty-card player-challenges-empty-state">
            <div className="player-challenges-empty-grid" />
            <div className="player-challenges-empty-orb player-challenges-empty-orb-a" />
            <div className="player-challenges-empty-orb player-challenges-empty-orb-b" />

            <div className="player-challenges-empty-badge">Challenge Bay</div>

            <div className="player-challenges-empty-icon">
              <FlagOutlined />
            </div>

            <Typography.Title
              level={2}
              className="player-challenges-empty-title"
            >
              挑战区暂时还是空的
            </Typography.Title>

            <Typography.Paragraph className="player-challenges-empty-description">
              这里还没有可练习的题目。等题库发布后，你可以按分类筛选、搜索关键字，快速进入训练状态。
            </Typography.Paragraph>

            <div className="player-challenges-empty-features">
              <div className="player-challenges-empty-feature">
                <RocketOutlined className="player-challenges-empty-feature-icon" />
                <div>
                  <strong>一键开练</strong>
                  <span>新题发布后会直接出现在这里</span>
                </div>
              </div>
              <div className="player-challenges-empty-feature">
                <ThunderboltOutlined className="player-challenges-empty-feature-icon" />
                <div>
                  <strong>分类清晰</strong>
                  <span>Web、Crypto、Pwn 等题型自动归档</span>
                </div>
              </div>
              <div className="player-challenges-empty-feature">
                <FireOutlined className="player-challenges-empty-feature-icon" />
                <div>
                  <strong>随时开打</strong>
                  <span>准备好后就能从这里进入完整刷题流</span>
                </div>
              </div>
            </div>

            <div className="player-challenges-empty-actions">
              <Button
                type="primary"
                size="large"
                onClick={() => navigate("/competition")}
              >
                看看比赛
              </Button>
            </div>
          </Card>
        )}
      </div>
      <ChallengeQuickModal
        challenge={selectedChallenge}
        onClose={() => setSelectedChallenge(null)}
        onSolved={(challengeId) => {
          setData((current) =>
            current.map((item) =>
              item.id === challengeId ? { ...item, is_solved: true } : item,
            ),
          );
        }}
      />
    </PlayerPage>
  );
}
