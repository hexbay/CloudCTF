import { ClockCircleOutlined, SendOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Checkbox, Col, Empty, Form, Input, Progress, Radio, Row, Space, Table, Tag, Typography, message } from "antd";
import type { FormInstance } from "antd/es/form";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import type { PlayerCompetition, PlayerExam, PlayerExamQuestion, PlayerRank } from "@shared/api/player";

type ArenaTabKey =
  | "overview"
  | "register"
  | "challenges"
  | "scoreboard"
  | "announcements";
export function TheoryCompetitionExamPanel({
  exam,
  endTime,
  examForm,
  examAnswers,
  examSubmitting,
  examResult,
  onBackToCompetitions,
  onSubmit,
}: {
  exam: PlayerExam | null;
  endTime?: string;
  examForm: FormInstance;
  examAnswers: Record<string, unknown>;
  examSubmitting: boolean;
  examResult: {
    correct?: number;
    total?: number;
    score?: number;
    total_score?: number;
  } | null;
  onBackToCompetitions: () => void;
  onSubmit: (values: Record<string, unknown>) => Promise<void>;
}) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const orderedQuestions = orderExamQuestions(exam?.questions ?? []);
  const answeredCount = countAnsweredValues(examAnswers, orderedQuestions);
  const examCountdown = getExamCountdown(endTime, now);

  if (examResult) {
    const completedResult = examResult;
    return (
      <div className="theory-exam-finished-shell">
        <Card className="player-panel-card theory-exam-finished-card">
          <div className="theory-exam-finished-state">
            <Typography.Title level={2}>考试已完成</Typography.Title>
            <div className="theory-exam-finished-score">
              <strong>
                {completedResult.score ?? 0}
              </strong>
              <span>/ {completedResult.total_score ?? exam?.total_score ?? 0} 分</span>
            </div>
            <div className="theory-exam-finished-meta">
              <div>正确题数：{completedResult.correct ?? 0} / {completedResult.total ?? 0}</div>
              <div>正确率：{formatExamAccuracy(completedResult.correct, completedResult.total)}</div>
            </div>
            <Button
              className="theory-exam-finished-button"
              type="primary"
              onClick={onBackToCompetitions}
            >
              返回比赛列表
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} xl={18}>
        <Card className="player-panel-card theory-exam-paper-card">
          <div className="theory-exam-paper-head">
            <div className="theory-exam-paper-title">理论试题</div>
          </div>
          {orderedQuestions.length ? (
            <Form
              form={examForm}
              layout="vertical"
              onFinish={onSubmit}
              onFinishFailed={() => message.error("请检查答题表单后重试")}
            >
              <div className="theory-exam-question-list">
                {orderedQuestions.map((question, index) => (
                  <div
                    key={question.id}
                    id={`theory-question-${question.id}`}
                    className="theory-exam-question-card"
                  >
                    <div className="theory-exam-question-head">
                      <div className="theory-exam-question-tags">
                        <Tag color="cyan">
                          {questionTypeText(question.question_type)}
                        </Tag>
                        <Tag color="gold">{question.points} 分</Tag>
                      </div>
                    </div>
                    <Typography.Title
                      level={5}
                      className="theory-exam-question-title"
                    >
                      {index + 1}. {question.question}
                    </Typography.Title>
                    <Form.Item
                      name={String(question.id)}
                      className="theory-exam-form-item"
                    >
                      {renderExamAnswerInput(question)}
                    </Form.Item>
                  </div>
                ))}
              </div>

              <div className="theory-exam-submitbar">
                <Button
                  className="theory-exam-submit-button"
                  type="primary"
                  icon={<SendOutlined />}
                  loading={examSubmitting}
                  onClick={() => examForm.submit()}
                >
                  提交试卷
                </Button>
              </div>
            </Form>
          ) : (
            <Empty description="当前赛事暂无理论试卷" />
          )}
        </Card>
      </Col>
      <Col xs={24} xl={6}>
        <Card
          className="player-panel-card theory-exam-nav-card theory-exam-nav-sticky"
          title="答题导航"
        >
          <div className="theory-exam-nav-countdown">
            <ClockCircleOutlined />
            <span>剩余时间：{examCountdown}</span>
          </div>
          <div className="theory-exam-nav-grid">
            {orderedQuestions.map((question, index) => {
              const answered = isQuestionAnswered(
                examAnswers[String(question.id)],
                question.question_type
              );
              return (
                <button
                  key={question.id}
                  type="button"
                  className={
                    answered
                      ? "theory-exam-nav-item is-answered"
                      : "theory-exam-nav-item"
                  }
                  onClick={() => {
                    const element = document.getElementById(
                      `theory-question-${question.id}`
                    );
                    element?.scrollIntoView({
                      behavior: "smooth",
                      block: "start",
                    });
                  }}
                >
                  {index + 1}
                </button>
              );
            })}
          </div>
          <div className="theory-exam-nav-legend">
            <span>
              <i className="is-answered" /> 已作答
            </span>
            <span>
              <i /> 未作答
            </span>
          </div>
          <div className="theory-exam-progress">
            <div className="theory-exam-progress-head">
              <span>答题进度</span>
              <strong>
                {answeredCount} / {orderedQuestions.length}
              </strong>
            </div>
            <Progress
              percent={
                orderedQuestions.length
                  ? Math.round((answeredCount / orderedQuestions.length) * 100)
                  : 0
              }
              showInfo={false}
              strokeColor="#14b8a6"
            />
          </div>
        </Card>
      </Col>
    </Row>
  );
}

export const scoreboardColumns: ColumnsType<PlayerRank> = [
  {
    title: "排名",
    dataIndex: "rank",
    width: 90,
    render: (rank: number) => (
      <span className={rank <= 3 ? `rank-top rank-${rank}` : undefined}>
        {rank}
      </span>
    ),
  },
  { title: "用户", dataIndex: "username" },
  { title: "解题数", dataIndex: "solved_count", width: 120 },
  {
    title: "得分",
    dataIndex: "total_points",
    width: 120,
    render: (value: number) => <span className="rank-score">{value}</span>,
  },
];

export function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

export function getArenaTab(pathname: string): ArenaTabKey {
  const tab = pathname.split("/")[2] as ArenaTabKey | undefined;
  if (
    tab === "register" ||
    tab === "challenges" ||
    tab === "scoreboard" ||
    tab === "announcements"
  ) {
    return tab;
  }
  return "overview";
}

export function getCountdown(endTime?: string) {
  if (!endTime) return { value: "-", label: "暂无结束时间" };
  const diff = new Date(endTime).getTime() - Date.now();
  if (Number.isNaN(diff)) return { value: "-", label: "结束时间异常" };
  if (diff <= 0) return { value: "已结束", label: "比赛已到结束时间" };
  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  return {
    value:
      days > 0 ? `${days} 天 ${hours} 小时` : `${hours} 小时 ${minutes} 分钟`,
    label: "距离比赛结束",
  };
}

export function getExamCountdown(endTime: string | undefined, now: number) {
  if (!endTime) return "--:--:--";
  const diff = new Date(endTime).getTime() - now;
  if (Number.isNaN(diff) || diff <= 0) return "00:00:00";
  const totalSeconds = Math.floor(diff / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return [hours, minutes, seconds].map((value) => String(value).padStart(2, "0")).join(":");
}

export function formatExamAccuracy(correct?: number, total?: number) {
  if (!total) return "0.00%";
  return `${((Number(correct ?? 0) / Number(total)) * 100).toFixed(2)}%`;
}

export function getRegistrationState(
  competition: PlayerCompetition | null,
  loggedIn: boolean
) {
  if (!loggedIn) {
    return {
      type: "info" as const,
      title: "需要先登录",
      description: "登录后才可以报名或进入比赛。",
    };
  }
  if (!competition) {
    return {
      type: "warning" as const,
      title: "比赛信息加载失败",
      description: "请稍后刷新重试。",
    };
  }
  if (competition.is_participant) {
    return {
      type: "success" as const,
      title: "已完成报名",
      description: "你现在可以直接进入比赛赛场。",
    };
  }
  if (competition.registration_open === false) {
    return {
      type: "warning" as const,
      title: "报名暂未开放",
      description: "当前比赛未开放报名，请关注赛事公告。",
    };
  }
  return {
    type: "success" as const,
    title: "当前可报名",
    description: "完成报名后将直接进入比赛赛场。",
  };
}

export function renderExamAnswerInput(question: PlayerExamQuestion) {
  if (question.question_type === "single") {
    return (
      <Radio.Group className="theory-exam-option-group">
        {(question.context?.options ?? []).map((option, index) => (
          <Radio
            key={`${index}-${option.text}`}
            value={String(index)}
            className="theory-exam-option"
          >
            {String.fromCharCode(65 + index)}. {option.text}
          </Radio>
        ))}
      </Radio.Group>
    );
  }

  if (question.question_type === "multiple") {
    return (
      <Checkbox.Group
        className="theory-exam-option-group theory-exam-checkbox-group"
        options={(question.context?.options ?? []).map((option, index) => ({
          label: `${String.fromCharCode(65 + index)}. ${option.text}`,
          value: String(index),
        }))}
      />
    );
  }

  if (question.question_type === "judge") {
    return (
      <Radio.Group className="theory-exam-option-group">
        <Radio value="true" className="theory-exam-option">
          正确
        </Radio>
        <Radio value="false" className="theory-exam-option">
          错误
        </Radio>
      </Radio.Group>
    );
  }

  return (
    <Input.TextArea
      rows={4}
      placeholder="请输入答案"
      className="theory-exam-textarea"
    />
  );
}

export function questionTypeText(type: PlayerExamQuestion["question_type"]) {
  if (type === "single") return "单选题";
  if (type === "multiple") return "多选题";
  if (type === "judge") return "判断题";
  return "问答题";
}

export function countAnsweredValues(
  values: Record<string, unknown>,
  questions: PlayerExamQuestion[]
) {
  return questions.reduce(
    (count, question) =>
      isQuestionAnswered(values[String(question.id)], question.question_type)
        ? count + 1
        : count,
    0
  );
}

export function isQuestionAnswered(
  value: unknown,
  type: PlayerExamQuestion["question_type"]
) {
  if (type === "multiple") return Array.isArray(value) && value.length > 0;
  if (typeof value === "string") return value.trim().length > 0;
  return value !== undefined && value !== null && value !== "";
}

export function normalizeExamAnswers(
  values: Record<string, unknown>,
  questions: PlayerExamQuestion[]
) {
  return questions.reduce<Record<string, unknown>>((result, question) => {
    const value = values[String(question.id)];
    if (value === undefined || value === null || value === "") {
      return result;
    }
    result[String(question.id)] =
      question.question_type === "text" ? String(value).trim() : value;
    return result;
  }, {});
}

export function orderExamQuestions(questions: PlayerExamQuestion[]) {
  const typeOrder: Record<PlayerExamQuestion["question_type"], number> = {
    single: 0,
    multiple: 1,
    judge: 2,
    text: 3,
  };
  return [...questions].sort((left, right) => {
    const orderDiff = typeOrder[left.question_type] - typeOrder[right.question_type];
    if (orderDiff !== 0) return orderDiff;
    return Number(left.id) - Number(right.id);
  });
}



