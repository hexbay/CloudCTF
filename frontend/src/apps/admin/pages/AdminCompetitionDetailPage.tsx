import {
  ArrowLeftOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Typography,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { TabsProps } from "antd";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import { getErrorMessage } from "@shared/api/http";
import { AdminCompetitionChallengesPage } from "./AdminCompetitionChallengesPage";

type Competition = {
  id: number;
  name: string;
  type?: string;
  status?: string;
  start_time?: string;
  end_time?: string;
  is_public?: boolean;
  registration_open?: boolean;
  scoreboard_frozen?: boolean;
  participant_mode?: string;
  description?: string;
};

type AnnouncementRow = {
  id: number;
  title: string;
  summary?: string;
  content?: string;
  created_at?: string;
};

type ParticipantRow = {
  id?: number;
  user_id: number;
  username?: string;
  team_id?: number | null;
  team_name?: string;
};

type ChallengeRow = {
  id: number;
  name?: string;
};

type SubmissionRow = {
  id: number;
  user_id?: number | null;
  team_id?: number | null;
  challenge_id?: number | null;
  is_correct?: boolean;
  points?: number;
  submitted_at?: string;
};

type ExamSubmissionRow = {
  id: number;
  user_id?: number | null;
  username?: string;
  team_id?: number | null;
  team_name?: string | null;
  total_score?: number;
  max_possible_score?: number;
  correct_answers?: number;
  total_questions?: number;
  percentage?: number;
  submitted_at?: string;
  is_graded?: boolean;
  answers?: Record<string, unknown>;
  exam?: {
    id?: number;
    title?: string;
    total_score?: number;
    questions?: Array<{
      id?: number;
      question?: string;
      question_type?: string;
      points?: number;
      context?: {
        options?: Array<{ text?: string; checked?: boolean }>;
        answer?: string | string[];
      };
    }>;
  };
};

type RowData = Record<string, unknown>;

const statusLabels: Record<string, string> = {
  draft: "草稿",
  upcoming: "即将开始",
  ongoing: "进行中",
  finished: "已结束",
  archived: "已归档",
};

const typeLabels: Record<string, string> = {
  ctf: "CTF",
  awd: "AWD",
  theory: "理论赛",
  mixed: "混合赛",
};

export function AdminCompetitionDetailPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const { competitionId = "", tabKey } = useParams();
  const [competition, setCompetition] = useState<Competition | null>(null);
  const [loading, setLoading] = useState(false);
  const [statusUpdating, setStatusUpdating] = useState(false);

  useEffect(() => {
    void loadCompetition();
  }, [competitionId]);

  async function loadCompetition() {
    setLoading(true);
    try {
      const result = await adminApi.list<Competition>(
        "/api-admin/competitions",
        { current: 1, pageSize: 500 },
      );
      const rows = normalizeRows<Competition>(result);
      setCompetition(
        rows.find((item) => String(item.id) === competitionId) ?? null,
      );
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载赛事详情失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function updateCompetitionStatus(status: string) {
    if (!competition) {
      return;
    }
    setStatusUpdating(true);
    try {
      await adminApi.action(
        `/api-admin/competitions/${competition.id}/status`,
        "PATCH",
        { status },
      );
      message.success("比赛状态已更新");
      await loadCompetition();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "更新比赛状态失败",
      );
    } finally {
      setStatusUpdating(false);
    }
  }

  const competitionType = competition?.type;

  // 根据比赛类型构建对应的 tab 项
  const typeTabs: NonNullable<TabsProps["items"]> = [];
  if (competition && (competitionType === "ctf" || competitionType === "awd")) {
    typeTabs.push(
      {
        key: "challenges",
        label: "题目设置",
        children: <AdminCompetitionChallengesPage embedded />,
      },
      {
        key: "submissions",
        label: "提交与成绩",
        children: (
          <CompetitionSubmissionsPanel
            competitionId={competition.id}
            participantMode={competition.participant_mode}
          />
        ),
      },
      {
        key: "writeups",
        label: "Writeup",
        children: <CompetitionWriteupsPanel competitionId={competition.id} />,
      },
    );
  } else if (competition && competitionType === "theory") {
    typeTabs.push(
      {
        key: "exam",
        label: "理论考试",
        children: <CompetitionExamSummary competitionId={competition.id} />,
      },
      {
        key: "submissions",
        label: "提交与成绩",
        children: (
          <CompetitionExamSubmissionsPanel competitionId={competition.id} />
        ),
      },
    );
  } else if (competition && competitionType === "mixed") {
    typeTabs.push(
      {
        key: "challenges",
        label: "题目设置",
        children: <AdminCompetitionChallengesPage embedded />,
      },
      {
        key: "submissions",
        label: "提交与成绩",
        children: (
          <CompetitionSubmissionsPanel
            competitionId={competition.id}
            participantMode={competition.participant_mode}
          />
        ),
      },
      {
        key: "exam",
        label: "理论考试",
        children: <CompetitionExamSummary competitionId={competition.id} />,
      },
    );
  }

  const tabItems: NonNullable<TabsProps["items"]> = [
    {
      key: "announcements",
      label: "比赛公告",
      children: competition ? (
        <CompetitionAnnouncementsPanel competitionId={competition.id} />
      ) : null,
    },
    ...typeTabs,
    {
      key: "participants",
      label: "参赛单位",
      children: competition ? (
        <CompetitionParticipantsPanel
          competitionId={competition.id}
          participantMode={competition.participant_mode}
        />
      ) : null,
    },
    {
      key: "admins",
      label: "管理员",
      children: (
        <BoundTable
          endpoint={`/api-admin/competitions/${competitionId}/admins`}
          columns={["id", "username", "email", "role"]}
        />
      ),
    },
  ];
  const allowedTabKeys = new Set(tabItems.map((item) => item.key));
  const activeTabKey =
    tabKey && allowedTabKeys.has(tabKey) ? tabKey : "announcements";

  useEffect(() => {
    if (!competitionId) {
      return;
    }
    if (!competition) {
      return;
    }
    if (!tabKey) {
      navigate(`/admin/competition/${competitionId}/${activeTabKey}`, {
        replace: true,
      });
      return;
    }
    if (!allowedTabKeys.has(tabKey)) {
      navigate(`/admin/competition/${competitionId}/announcements`, {
        replace: true,
      });
    }
  }, [
    activeTabKey,
    allowedTabKeys,
    competition,
    competitionId,
    navigate,
    tabKey,
  ]);

  if (!competition) {
    return (
      <div className="admin-page">
        <Card className="admin-table-card" bordered={false} loading={loading}>
          <Space direction="vertical">
            <Typography.Title level={4}>赛事不存在或尚未加载</Typography.Title>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate("/admin/competition")}
            >
              返回赛事列表
            </Button>
          </Space>
        </Card>
      </div>
    );
  }

  return (
    <div className="admin-page">
      <div className="admin-page-heading">
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadCompetition()}
          >
            刷新
          </Button>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate("/admin/competition")}
          >
            返回赛事列表
          </Button>
        </Space>
      </div>

      <Card className="admin-table-card" bordered={false} loading={loading}>
        <Descriptions bordered size="small" column={3}>
          <Descriptions.Item label="ID">{competition.id}</Descriptions.Item>
          <Descriptions.Item label="类型">
            {typeLabels[String(competition.type)] ?? competition.type}
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColor(String(competition.status))}>
              {statusLabels[String(competition.status)] ?? competition.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="开始时间">
            {formatValue(competition.start_time)}
          </Descriptions.Item>
          <Descriptions.Item label="结束时间">
            {formatValue(competition.end_time)}
          </Descriptions.Item>
          <Descriptions.Item label="参赛模式">
            {competition.participant_mode === "team" ? "团队" : "个人"}
          </Descriptions.Item>
          <Descriptions.Item label="公开">
            {competition.is_public ? "是" : "否"}
          </Descriptions.Item>
          <Descriptions.Item label="报名">
            {competition.registration_open ? "开放" : "关闭"}
          </Descriptions.Item>
          <Descriptions.Item label="冻结榜单">
            {competition.scoreboard_frozen ? "已冻结" : "未冻结"}
          </Descriptions.Item>
          <Descriptions.Item label="描述" span={3}>
            {competition.description || "-"}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5} style={{ margin: 0 }}>
            比赛操作
          </Typography.Title>
          <Space wrap>
            {renderCompetitionStatusActions(
              competition,
              statusUpdating,
              updateCompetitionStatus,
            )}
          </Space>
        </div>
      </Card>

      <Card className="admin-table-card" bordered={false}>
        <Tabs
          activeKey={activeTabKey}
          onChange={(nextKey) =>
            navigate(`/admin/competition/${competitionId}/${nextKey}`)
          }
          items={tabItems}
        />
      </Card>
    </div>
  );
}

function CompetitionSubmissionsPanel({
  competitionId,
  participantMode,
}: {
  competitionId: number;
  participantMode?: string;
}) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [submissions, setSubmissions] = useState<SubmissionRow[]>([]);
  const [participants, setParticipants] = useState<ParticipantRow[]>([]);
  const [challenges, setChallenges] = useState<ChallengeRow[]>([]);

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const [submissionResult, participantResult, challengeResult] =
        await Promise.all([
          adminApi.list<SubmissionRow>(
            `/api-admin/competitions/${competitionId}/submissions`,
            { current: 1, pageSize: 100 },
          ),
          adminApi.list<ParticipantRow>(
            `/api-admin/competitions/${competitionId}/participants`,
            { current: 1, pageSize: 100 },
          ),
          adminApi.list<ChallengeRow>(
            `/api-admin/competitions/${competitionId}/challenges`,
            { current: 1, pageSize: 100 },
          ),
        ]);
      setSubmissions(normalizeRows(submissionResult));
      setParticipants(normalizeRows(participantResult));
      setChallenges(normalizeRows(challengeResult));
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载提交与成绩失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function removeSubmission(id: number) {
    try {
      await adminApi.remove(`/api-admin/submissions/${id}`);
      message.success("提交记录已删除");
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "删除提交记录失败",
      );
    }
  }

  const isTeamMode = participantMode === "team";
  const userNameMap = new Map<number, string>();
  const teamNameMap = new Map<number, string>();
  participants.forEach((item) => {
    if (typeof item.user_id === "number" && item.username) {
      userNameMap.set(item.user_id, item.username);
    }
    if (typeof item.team_id === "number" && item.team_name) {
      teamNameMap.set(item.team_id, item.team_name);
    }
  });
  const challengeNameMap = new Map<number, string>();
  challenges.forEach((item) => {
    if (typeof item.id === "number" && item.name) {
      challengeNameMap.set(item.id, item.name);
    }
  });

  const submissionColumns: ColumnsType<SubmissionRow> = [
    { title: "提交 ID", dataIndex: "id", width: 100 },
    {
      title: isTeamMode ? "参赛队伍" : "提交用户",
      key: "participant",
      render: (_, record) =>
        renderParticipant(
          record.user_id,
          record.team_id,
          userNameMap,
          teamNameMap,
          isTeamMode,
        ),
    },
    {
      title: "题目",
      key: "challenge",
      render: (_, record) =>
        renderNamedValue(record.challenge_id, challengeNameMap, "题目"),
    },
    {
      title: "结果",
      dataIndex: "is_correct",
      width: 100,
      render: (value: boolean | undefined) => (
        <Tag color={value ? "green" : "red"}>{value ? "正确" : "错误"}</Tag>
      ),
    },
    {
      title: "得分",
      dataIndex: "points",
      width: 100,
      render: (value: number | undefined) => value ?? 0,
    },
    {
      title: "提交时间",
      dataIndex: "submitted_at",
      width: 180,
      render: (value: string | undefined) => formatValue(value),
    },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, record) => (
        <Popconfirm
          title="确认删除这条提交记录？"
          onConfirm={() => void removeSubmission(record.id)}
        >
          <Button size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <div className="resource-toolbar">
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>
          刷新
        </Button>
      </div>

      <Card size="small" title="提交记录">
        <Table
          rowKey="id"
          loading={loading}
          columns={submissionColumns}
          dataSource={submissions}
          pagination={{ pageSize: 10 }}
        />
      </Card>
    </Space>
  );
}

function CompetitionExamSummary({ competitionId }: { competitionId: number }) {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [exam, setExam] = useState<Record<string, unknown> | null>(null);
  const [availableExams, setAvailableExams] = useState<
    Array<{ id: number; title: string; total_score: number }>
  >([]);
  const [selectVisible, setSelectVisible] = useState(false);
  const [quickCreateVisible, setQuickCreateVisible] = useState(false);
  const [bindingExam, setBindingExam] = useState(false);
  const [creatingExam, setCreatingExam] = useState(false);
  const [selectForm] = Form.useForm();
  const [quickCreateForm] = Form.useForm();

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const [result, examsResult] = await Promise.all([
        adminApi.get<
          Record<string, unknown> | { exam?: Record<string, unknown> | null }
        >(`/api-admin/exam/competitions/${competitionId}/exam`),
        adminApi.list<{ id: number; title: string; total_score: number }>(
          "/api-admin/exam",
          { current: 1, pageSize: 500 },
        ),
      ]);
      setExam(normalizeCompetitionExam(result));
      setAvailableExams(
        normalizeRows<{ id: number; title: string; total_score: number }>(
          examsResult,
        ),
      );
    } catch (error) {
      message.error(getErrorMessage(error, "加载试卷失败"));
    } finally {
      setLoading(false);
    }
  }

  async function bindSelectedExam() {
    try {
      const values = await selectForm.validateFields();
      setBindingExam(true);
      const createdExam = await adminApi.create<Record<string, unknown>>(
        `/api-admin/exam/competitions/${competitionId}/exam/bind`,
        {
          source_exam_id: values.source_exam_id,
        },
      );
      setExam(normalizeCompetitionExam(createdExam));
      message.success("试卷已复制并绑定到赛事");
      setSelectVisible(false);
      selectForm.resetFields();
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "关联试卷失败"));
    } finally {
      setBindingExam(false);
    }
  }

  async function quickCreateExam() {
    try {
      const values = await quickCreateForm.validateFields();
      setCreatingExam(true);
      const createdExam = await adminApi.create<Record<string, unknown>>(
        `/api-admin/exam/competitions/${competitionId}/exam/quick-create`,
        {
          title: values.title,
          single_count: Number(values.single_count ?? 0),
          multiple_count: Number(values.multiple_count ?? 0),
          judge_count: Number(values.judge_count ?? 0),
          text_count: Number(values.text_count ?? 0),
        },
      );
      setExam(normalizeCompetitionExam(createdExam));
      message.success("赛事试卷创建成功");
      setQuickCreateVisible(false);
      quickCreateForm.resetFields();
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "创建试卷失败"));
    } finally {
      setCreatingExam(false);
    }
  }

  async function removeExam() {
    await adminApi.remove(`/api-admin/exam/competitions/${competitionId}/exam`);
    message.success("赛事试卷已删除");
    await load();
  }

  return (
    <>
      <Card size="small" loading={loading}>
        {!exam ? (
          <div className="competition-exam-empty">
            <Typography.Text type="secondary">
              当前赛事还没有关联试卷。
            </Typography.Text>
            <Space wrap>
              <Button type="primary" onClick={() => setSelectVisible(true)}>
                选择已有试卷
              </Button>
              <Button onClick={() => setQuickCreateVisible(true)}>
                快速创建试卷
              </Button>
            </Space>
          </div>
        ) : (
          <div className="competition-exam-summary">
            <div className="competition-exam-summary-main">
              <div className="competition-exam-summary-header">
                <Typography.Title level={5} style={{ margin: 0 }}>
                  {String(exam.title ?? "-")}
                </Typography.Title>
                <Tag className="competition-tag competition-tag-type-theory">
                  赛事试卷
                </Tag>
              </div>
              <div className="competition-exam-summary-meta">
                <span>试卷 ID：{String(exam.id ?? "-")}</span>
                <span>总分：{String(exam.total_score ?? "-")} 分</span>
                <span>题目数：{extractExamQuestionCount(exam)} 题</span>
              </div>
            </div>
            <Space
              wrap
              className="admin-action-group competition-exam-summary-actions"
            >
              <Button
                type="text"
                size="small"
                className="admin-action-button"
                onClick={() =>
                  navigate(
                    `/admin/competition/${competitionId}/exams/${exam.id}`,
                  )
                }
              >
                查看/编辑
              </Button>
              <Button
                type="text"
                size="small"
                className="admin-action-button"
                onClick={() => setSelectVisible(true)}
              >
                替换试卷
              </Button>
              <Button
                type="text"
                size="small"
                className="admin-action-button"
                onClick={() => setQuickCreateVisible(true)}
              >
                重新组卷
              </Button>
              <Popconfirm
                title="确认删除当前赛事试卷？"
                onConfirm={() => removeExam()}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  className="admin-action-button admin-action-button-danger"
                >
                  删除
                </Button>
              </Popconfirm>
            </Space>
          </div>
        )}
      </Card>

      <Modal
        title="选择已有试卷"
        open={selectVisible}
        onCancel={() => {
          setSelectVisible(false);
          selectForm.resetFields();
        }}
        onOk={() => void bindSelectedExam()}
        confirmLoading={bindingExam}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary">
          选择公共试卷后，会复制一份到当前赛事，后续修改不会影响原试卷。
        </Typography.Paragraph>
        <Form form={selectForm} layout="vertical">
          <Form.Item
            name="source_exam_id"
            label="试卷"
            rules={[{ required: true, message: "请选择试卷" }]}
          >
            <Select
              showSearch
              placeholder="请选择试卷"
              optionFilterProp="label"
              options={availableExams.map((item) => ({
                value: item.id,
                label: `${item.title}（${item.total_score}分）`,
              }))}
              notFoundContent={
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="暂无可选择的公共试卷"
                />
              }
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="快速创建赛事试卷"
        open={quickCreateVisible}
        onCancel={() => {
          setQuickCreateVisible(false);
          quickCreateForm.resetFields();
        }}
        onOk={() => void quickCreateExam()}
        confirmLoading={creatingExam}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary">
          创建出的试卷会直接绑定到当前赛事，并从全局试卷管理中隐藏。
        </Typography.Paragraph>
        <Form form={quickCreateForm} layout="vertical">
          <Form.Item
            name="title"
            label="试卷标题"
            rules={[{ required: true, message: "请输入试卷标题" }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="single_count" label="单选题数量" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="multiple_count" label="多选题数量" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="judge_count" label="判断题数量" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="text_count" label="问答题数量" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

function CompetitionAnnouncementsPanel({
  competitionId,
}: {
  competitionId: number;
}) {
  const { message } = App.useApp();
  const [rows, setRows] = useState<AnnouncementRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AnnouncementRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<AnnouncementRow>(
        `/api-admin/competitions/${competitionId}/announcements`,
        { current: 1, pageSize: 50 },
      );
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载公告失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  }

  function openEdit(record: AnnouncementRow) {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  }

  async function submit() {
    try {
      const values = await form.validateFields();
      setSaving(true);
      const payload = {
        ...values,
        author_id: 1,
        is_published: true,
        target_type: "competition",
        target_id: competitionId,
      };
      if (editing) {
        await adminApi.update(
          `/api-admin/announcements/${editing.id}`,
          payload,
        );
        message.success("公告已更新");
      } else {
        await adminApi.create("/api-admin/announcements", payload);
        message.success("公告已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  }

  async function remove(id: number) {
    await adminApi.remove(`/api-admin/announcements/${id}`);
    message.success("公告已删除");
    await load();
  }

  const columns: ColumnsType<AnnouncementRow> = [
    { title: "标题", dataIndex: "title" },
    { title: "摘要", dataIndex: "summary", render: (value) => value || "-" },
    {
      title: "创建时间",
      dataIndex: "created_at",
      render: (value) => formatValue(value),
    },
    {
      title: "操作",
      key: "actions",
      width: 180,
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确认删除这条公告？"
            onConfirm={() => remove(record.id)}
          >
            <Button size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div className="resource-toolbar" style={{ marginBottom: 16 }}>
        <Typography.Title level={5} style={{ margin: 0 }}>
          比赛公告
        </Typography.Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            发布公告
          </Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={rows}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title={editing ? "编辑公告" : "发布公告"}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
          form.resetFields();
        }}
        onOk={() => void submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="title"
            label="公告标题"
            rules={[{ required: true, message: "请输入公告标题" }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="summary" label="公告摘要">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item
            name="content"
            label="公告内容"
            rules={[{ required: true, message: "请输入公告内容" }]}
          >
            <Input.TextArea rows={6} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

function CompetitionParticipantsPanel({
  competitionId,
  participantMode,
}: {
  competitionId: number;
  participantMode?: string;
}) {
  const { message } = App.useApp();
  const [rows, setRows] = useState<ParticipantRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [generateModalOpen, setGenerateModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [form] = Form.useForm();
  const [generateForm] = Form.useForm();

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<ParticipantRow>(
        `/api-admin/competitions/${competitionId}/participants`,
        {
          current: 1,
          pageSize: 100,
        },
      );
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载参赛单位失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function submit() {
    try {
      const values = await form.validateFields();
      setSaving(true);
      await adminApi.create(
        `/api-admin/competitions/${competitionId}/participants`,
        values,
      );
      message.success("参赛单位已添加");
      setModalOpen(false);
      form.resetFields();
      await load();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  }

  async function generateTeams() {
    try {
      const values = await generateForm.validateFields();
      setGenerating(true);
      const result = await adminApi.create<{
        created_teams?: number;
        created_users?: number;
        default_password?: string;
      }>(
        `/api-admin/competitions/${competitionId}/participants/generate`,
        values,
      );
      message.success(
        `已生成 ${result.created_teams ?? 0} 支队伍、${result.created_users ?? 0} 名选手，默认密码：${result.default_password ?? "-"}`,
      );
      setGenerateModalOpen(false);
      generateForm.resetFields();
      await load();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setGenerating(false);
    }
  }

  async function remove(userId: number) {
    await adminApi.remove(
      `/api-admin/competitions/${competitionId}/participants/${userId}`,
    );
    message.success("参赛单位已移除");
    await load();
  }

  const isTeamMode = participantMode === "team";

  const columns: ColumnsType<ParticipantRow> = [
    ...(isTeamMode
      ? [
          {
            title: "队伍 ID",
            dataIndex: "team_id",
            width: 120,
            render: (value: number | null | undefined) => value ?? "-",
          },
          {
            title: "队伍名称",
            dataIndex: "team_name",
            render: (value: string | undefined) => value || "-",
          },
          {
            title: "队长/成员用户名",
            dataIndex: "username",
            render: (value: string | undefined) => value || "-",
          },
        ]
      : [
          { title: "参赛个人 ID", dataIndex: "user_id", width: 140 },
          {
            title: "参赛个人",
            dataIndex: "username",
            render: (value: string | undefined) => value || "-",
          },
        ]),
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record) => (
        <Popconfirm
          title="确认移除该参赛单位？"
          onConfirm={() => remove(record.user_id)}
        >
          <Button size="small" danger icon={<DeleteOutlined />}>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <div className="resource-toolbar" style={{ marginBottom: 16 }}>
        <Typography.Title level={5} style={{ margin: 0 }}>
          {isTeamMode ? "参赛队伍" : "参赛个人"}
        </Typography.Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          {isTeamMode ? (
            <Button onClick={() => setGenerateModalOpen(true)}>
              批量生成队伍
            </Button>
          ) : null}
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setModalOpen(true)}
          >
            {isTeamMode ? "添加参赛队伍成员" : "添加参赛个人"}
          </Button>
        </Space>
      </div>
      <Table
        rowKey={(record) =>
          String(
            (record as ParticipantRow & { id?: number }).id ??
              `${record.user_id}-${record.team_id ?? "none"}`,
          )
        }
        loading={loading}
        columns={columns}
        dataSource={rows}
        pagination={{ pageSize: 10 }}
      />

      {isTeamMode ? (
        <Modal
          title="批量生成参赛队伍"
          open={generateModalOpen}
          onCancel={() => {
            setGenerateModalOpen(false);
            generateForm.resetFields();
          }}
          onOk={() => void generateTeams()}
          confirmLoading={generating}
          destroyOnClose
        >
          <Form
            form={generateForm}
            layout="vertical"
            initialValues={{
              team_count: 30,
              members_per_team: 4,
              school_name: "",
              team_prefix: "",
              username_prefix: "",
              default_password: "CloudFlag123!",
            }}
          >
            <Form.Item name="school_name" label="学校名称">
              <Input placeholder="例如：某职业学院" />
            </Form.Item>
            <Form.Item name="team_prefix" label="队伍名前缀">
              <Input placeholder="为空时默认使用学校名称" />
            </Form.Item>
            <Form.Item name="username_prefix" label="账号前缀">
              <Input placeholder="为空时自动生成" />
            </Form.Item>
            <Form.Item
              name="team_count"
              label="生成队伍数"
              rules={[{ required: true, message: "请输入队伍数" }]}
            >
              <InputNumber min={1} max={500} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="members_per_team"
              label="每队人数（含队长）"
              rules={[{ required: true, message: "请输入每队人数" }]}
            >
              <InputNumber min={1} max={20} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="default_password"
              label="默认密码"
              rules={[{ required: true, message: "请输入默认密码" }]}
            >
              <Input />
            </Form.Item>
          </Form>
        </Modal>
      ) : null}

      <Modal
        title={isTeamMode ? "添加参赛队伍成员" : "添加参赛个人"}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          form.resetFields();
        }}
        onOk={() => void submit()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="user_id"
            label={isTeamMode ? "用户 ID" : "参赛个人 ID"}
            rules={[{ required: true, message: "请输入用户 ID" }]}
          >
            <InputNumber style={{ width: "100%" }} min={1} />
          </Form.Item>
          {isTeamMode ? (
            <Form.Item
              name="team_id"
              label="队伍 ID"
              rules={[{ required: true, message: "请输入队伍 ID" }]}
            >
              <InputNumber style={{ width: "100%" }} min={1} />
            </Form.Item>
          ) : null}
        </Form>
      </Modal>
    </>
  );
}

function BoundTable({
  endpoint,
  query,
  columns,
}: {
  endpoint: string;
  query?: Record<string, string>;
  columns: string[];
}) {
  const { message } = App.useApp();
  const [rows, setRows] = useState<RowData[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void load();
  }, [endpoint, JSON.stringify(query ?? {})]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(endpoint, {
        current: 1,
        pageSize: 50,
        ...query,
      });
      setRows(normalizeRows<RowData>(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载数据失败");
    } finally {
      setLoading(false);
    }
  }

  const tableColumns: ColumnsType<RowData> = columns.map((key) => ({
    title: labelize(key),
    dataIndex: key,
    ellipsis: true,
    render: (value) => renderValue(value),
  }));

  return (
    <Table
      rowKey={(record) =>
        String(record.id ?? record.user_id ?? JSON.stringify(record))
      }
      loading={loading}
      columns={tableColumns}
      dataSource={rows}
      pagination={{ pageSize: 10 }}
    />
  );
}

function normalizeRows<T>(result: unknown): T[] {
  if (Array.isArray(result)) return result as T[];
  const object = (result ?? {}) as { data?: T[]; items?: T[]; list?: T[] };
  return object.data ?? object.items ?? object.list ?? [];
}

function normalizeCompetitionExam(
  result:
    | Record<string, unknown>
    | { exam?: Record<string, unknown> | null }
    | null
    | undefined,
): Record<string, unknown> | null {
  if (!isRecord(result)) {
    return null;
  }
  if ("exam" in result) {
    return isRecord(result.exam) ? result.exam : null;
  }
  const record = result as Record<string, unknown>;
  return typeof record.id === "number" ? record : null;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function renderCompetitionStatusActions(
  competition: Competition,
  loading: boolean,
  onUpdateStatus: (status: string) => Promise<void>,
) {
  if (competition.status === "draft") {
    return (
      <Button
        type="primary"
        loading={loading}
        onClick={() => void onUpdateStatus("upcoming")}
      >
        发布比赛
      </Button>
    );
  }
  if (competition.status === "upcoming") {
    return (
      <>
        <Button
          type="primary"
          loading={loading}
          onClick={() => void onUpdateStatus("ongoing")}
        >
          开始比赛
        </Button>
        <Button loading={loading} onClick={() => void onUpdateStatus("draft")}>
          撤回发布
        </Button>
      </>
    );
  }
  if (competition.status === "ongoing") {
    return (
      <Button
        danger
        loading={loading}
        onClick={() => void onUpdateStatus("finished")}
      >
        停止比赛
      </Button>
    );
  }
  if (competition.status === "finished") {
    return (
      <>
        <Button
          type="primary"
          loading={loading}
          onClick={() => void onUpdateStatus("archived")}
        >
          归档比赛
        </Button>
        <Button
          loading={loading}
          onClick={() => void onUpdateStatus("ongoing")}
        >
          重新开启
        </Button>
      </>
    );
  }
  if (competition.status === "archived") {
    return (
      <Button loading={loading} onClick={() => void onUpdateStatus("finished")}>
        取消归档
      </Button>
    );
  }
  return (
    <Typography.Text type="secondary">当前状态暂无可用操作</Typography.Text>
  );
}

function statusColor(status: string) {
  if (status === "ongoing") return "green";
  if (status === "finished") return "orange";
  if (status === "archived") return "purple";
  if (status === "upcoming") return "blue";
  return "default";
}

function formatValue(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function renderValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "是" : "否";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function CompetitionExamSubmissionsPanel({
  competitionId,
}: {
  competitionId: number;
}) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [submissions, setSubmissions] = useState<ExamSubmissionRow[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailSubmission, setDetailSubmission] =
    useState<ExamSubmissionRow | null>(null);

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<ExamSubmissionRow>(
        "/api-admin/exam/submissions",
        {
          current: 1,
          pageSize: 100,
          competition_id: String(competitionId),
        },
      );
      setSubmissions(normalizeRows(result));
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载提交记录失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(record: ExamSubmissionRow) {
    setDetailLoading(true);
    try {
      const result = await adminApi.get<ExamSubmissionRow>(
        `/api-admin/exam/submissions/${record.id}`,
      );
      setDetailSubmission(result);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载答卷详情失败",
      );
    } finally {
      setDetailLoading(false);
    }
  }

  const columns: ColumnsType<ExamSubmissionRow> = [
    { title: "提交 ID", dataIndex: "id", width: 100 },
    {
      title: "用户",
      dataIndex: "username",
      render: (value: string | undefined) => value || "-",
    },
    {
      title: "队伍",
      dataIndex: "team_name",
      render: (value: string | null | undefined) => value || "-",
    },
    {
      title: "得分",
      key: "score",
      render: (_, record) =>
        `${Number(record.total_score ?? 0)} / ${Number(record.max_possible_score ?? 0)}`,
    },
    {
      title: "正确题数",
      key: "answers",
      render: (_, record) =>
        `${Number(record.correct_answers ?? 0)} / ${Number(record.total_questions ?? 0)}`,
    },
    {
      title: "得分率",
      dataIndex: "percentage",
      width: 100,
      render: (value: number | undefined) => `${Number(value ?? 0)}%`,
    },
    {
      title: "状态",
      dataIndex: "is_graded",
      width: 100,
      render: (value: boolean | undefined) => (
        <Tag color={value ? "green" : "orange"}>
          {value ? "已评分" : "待评分"}
        </Tag>
      ),
    },
    {
      title: "提交时间",
      dataIndex: "submitted_at",
      width: 180,
      render: (value: string | undefined) => formatValue(value),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record) => (
        <Button size="small" onClick={() => void openDetail(record)}>
          查看试卷
        </Button>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <div className="resource-toolbar">
        <Typography.Title level={5} style={{ margin: 0 }}>
          提交与成绩
        </Typography.Title>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>
          刷新
        </Button>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={submissions}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title={
          detailSubmission ? `答卷详情 #${detailSubmission.id}` : "答卷详情"
        }
        open={Boolean(detailSubmission)}
        onCancel={() => setDetailSubmission(null)}
        footer={null}
        width={960}
        destroyOnClose
      >
        <Spin spinning={detailLoading}>
          {detailSubmission?.exam?.questions?.length ? (
            <Space direction="vertical" size={16} style={{ width: "100%" }}>
              <Typography.Text type="secondary">
                {detailSubmission.exam.title || "理论考试"} · 得分{" "}
                {Number(detailSubmission.total_score ?? 0)} /{" "}
                {Number(detailSubmission.max_possible_score ?? 0)}
              </Typography.Text>
              {orderExamReviewQuestions(detailSubmission.exam.questions).map(
                (question, index) => {
                  const answerKey = String(question.id ?? "");
                  const userAnswer = detailSubmission.answers?.[answerKey];
                  const correctAnswer = question.context?.answer;
                  return (
                    <Card key={answerKey || String(index)} size="small">
                      <Space
                        direction="vertical"
                        size={8}
                        style={{ width: "100%" }}
                      >
                        <div
                          style={{
                            display: "flex",
                            alignItems: "center",
                            gap: 8,
                            flexWrap: "wrap",
                          }}
                        >
                          <Tag color="cyan">
                            {reviewQuestionTypeText(question.question_type)}
                          </Tag>
                          <Tag color="gold">
                            {Number(question.points ?? 0)} 分
                          </Tag>
                        </div>
                        <Typography.Title level={5} style={{ margin: 0 }}>
                          {index + 1}. {question.question || "-"}
                        </Typography.Title>
                        {renderExamSubmissionReview(
                          question,
                          userAnswer,
                          correctAnswer,
                        )}
                      </Space>
                    </Card>
                  );
                },
              )}
            </Space>
          ) : (
            <Empty description="暂无答卷详情" />
          )}
        </Spin>
      </Modal>
    </Space>
  );
}

function CompetitionWriteupsPanel({
  competitionId,
}: {
  competitionId: number;
}) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<Array<Record<string, unknown>>>([]);
  const [search, setSearch] = useState("");
  const [removingID, setRemovingID] = useState<number | null>(null);

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<Record<string, unknown>>(
        `/api-admin/competitions/${competitionId}/writeups`,
        { current: 1, pageSize: 200 },
      );
      setRows(normalizeRows(result));
    } catch {
      setRows([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(id: number) {
    setRemovingID(id);
    try {
      await adminApi.remove(`/api-admin/writeups/${id}`);
      message.success("Writeup 已删除");
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "删除失败"));
    } finally {
      setRemovingID(null);
    }
  }

  const filtered = search
    ? rows.filter((row) =>
        [row.username, row.team_name, row.title, row.file_name].some((value) =>
          String(value ?? "")
            .toLowerCase()
            .includes(search.toLowerCase()),
        ),
      )
    : rows;

  const columns: ColumnsType<Record<string, unknown>> = [
    {
      title: "用户名",
      dataIndex: "username",
      render: (value: unknown) => String(value ?? "-"),
    },
    {
      title: "队伍",
      dataIndex: "team_name",
      render: (value: unknown) => String(value ?? "-"),
    },
    {
      title: "标题",
      dataIndex: "title",
      render: (value: unknown) => String(value ?? "-"),
    },
    {
      title: "文件",
      key: "file",
      render: (_, record) => {
        const name = String(record.file_name ?? "-");
        const url = String(record.url ?? "");
        return url ? (
          <a href={url} target="_blank" rel="noreferrer">
            {name}
          </a>
        ) : (
          <span>{name}</span>
        );
      },
    },
    {
      title: "大小",
      dataIndex: "file_size",
      width: 100,
      render: (value: unknown) => {
        const bytes = Number(value ?? 0);
        if (bytes < 1024) return `${bytes} B`;
        if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
      },
    },
    {
      title: "上传时间",
      dataIndex: "created_at",
      width: 180,
      render: (value: unknown) => formatValue(String(value ?? "")),
    },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, record) => (
        <Popconfirm
          title="确认删除该 Writeup？"
          onConfirm={() => void handleDelete(Number(record.id))}
        >
          <Button
            size="small"
            danger
            icon={<DeleteOutlined />}
            loading={removingID === Number(record.id)}
          >
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <div className="resource-toolbar">
        <Typography.Title level={5} style={{ margin: 0 }}>
          Writeup 管理
        </Typography.Title>
        <Space>
          <Input.Search
            placeholder="搜索用户名、队伍、标题..."
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onSearch={() => {}}
            style={{ width: 300 }}
            allowClear
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={filtered}
        pagination={{ pageSize: 10 }}
        locale={{ emptyText: <Empty description="暂无 Writeup 提交" /> }}
      />
    </Space>
  );
}

function renderNamedValue(
  id: number | null | undefined,
  nameMap: Map<number, string>,
  label: string,
) {
  if (typeof id !== "number") return "-";
  const name = nameMap.get(id);
  return name ? `${name} (#${id})` : `${label} #${id}`;
}

function renderParticipant(
  userId: number | null | undefined,
  teamId: number | null | undefined,
  userNameMap: Map<number, string>,
  teamNameMap: Map<number, string>,
  isTeamMode: boolean,
) {
  if (isTeamMode && typeof teamId === "number") {
    const teamName = teamNameMap.get(teamId);
    if (teamName) return `${teamName} (#${teamId})`;
    return `队伍 #${teamId}`;
  }
  if (typeof userId === "number") {
    const username = userNameMap.get(userId);
    if (username) return `${username} (#${userId})`;
    return `用户 #${userId}`;
  }
  return "-";
}

function extractExamQuestionCount(exam: Record<string, unknown>) {
  const context = exam.context as { questions?: unknown[] } | undefined;
  return Array.isArray(context?.questions) ? context.questions.length : 0;
}

function reviewQuestionTypeText(type?: string) {
  if (type === "single") return "单选题";
  if (type === "multiple") return "多选题";
  if (type === "judge") return "判断题";
  return "问答题";
}

function orderExamReviewQuestions(
  questions: NonNullable<ExamSubmissionRow["exam"]>["questions"],
) {
  const typeOrder: Record<string, number> = {
    single: 0,
    multiple: 1,
    judge: 2,
    text: 3,
  };
  const safeQuestions = Array.isArray(questions) ? questions : [];
  return [...safeQuestions].sort((left, right) => {
    const orderDiff =
      (typeOrder[left.question_type ?? "text"] ?? 99) -
      (typeOrder[right.question_type ?? "text"] ?? 99);
    if (orderDiff !== 0) return orderDiff;
    return Number(left.id ?? 0) - Number(right.id ?? 0);
  });
}

function renderExamSubmissionReview(
  question: NonNullable<
    NonNullable<ExamSubmissionRow["exam"]>["questions"]
  >[number],
  userAnswer: unknown,
  correctAnswer: unknown,
) {
  const type = question.question_type;
  if (type === "single" || type === "multiple") {
    const options = question.context?.options ?? [];
    const selectedValues = normalizeAnswerValues(userAnswer);
    return (
      <div>
        <Typography.Text strong>选项：</Typography.Text>
        <Space
          direction="vertical"
          size={8}
          style={{ width: "100%", marginTop: 8 }}
        >
          {options.map((option, index) => {
            const typedOption = option as { text?: string; checked?: boolean };
            const value = String(index);
            const selected = selectedValues.includes(value);
            const correct = Boolean(typedOption.checked);
            return (
              <div
                key={`${index}-${typedOption.text ?? ""}`}
                style={{
                  padding: "10px 12px",
                  borderRadius: 8,
                  border: `1px solid ${correct ? "#52c41a" : selected ? "#1677ff" : "#d9d9d9"}`,
                  background:
                    correct && selected
                      ? "#f6ffed"
                      : correct
                        ? "#f6ffed"
                        : selected
                          ? "#e6f4ff"
                          : "#fff",
                }}
              >
                <Space size={8} wrap>
                  <span>
                    {String.fromCharCode(65 + index)}. {typedOption.text ?? ""}
                  </span>
                  {correct ? <Tag color="green">正确答案</Tag> : null}
                  {selected ? <Tag color="blue">选手答案</Tag> : null}
                </Space>
              </div>
            );
          })}
        </Space>
      </div>
    );
  }

  if (type === "judge") {
    const selected = renderJudgeAnswer(userAnswer);
    const correct = renderJudgeAnswer(correctAnswer);
    return (
      <Space direction="vertical" size={8} style={{ width: "100%" }}>
        <div>
          <Typography.Text strong>正确答案：</Typography.Text>
          <Tag color="green" style={{ marginInlineStart: 8 }}>
            {correct}
          </Tag>
        </div>
        <div>
          <Typography.Text strong>选手答案：</Typography.Text>
          <Tag
            color={selected === correct ? "cyan" : "blue"}
            style={{ marginInlineStart: 8 }}
          >
            {selected}
          </Tag>
        </div>
      </Space>
    );
  }

  return (
    <Space direction="vertical" size={8} style={{ width: "100%" }}>
      <div>
        <Typography.Text strong>正确答案：</Typography.Text>
        <div
          style={{
            marginTop: 6,
            padding: "10px 12px",
            borderRadius: 8,
            background: "#f6ffed",
            border: "1px solid #b7eb8f",
          }}
        >
          {renderValue(correctAnswer)}
        </div>
      </div>
      <div>
        <Typography.Text strong>选手答案：</Typography.Text>
        <div
          style={{
            marginTop: 6,
            padding: "10px 12px",
            borderRadius: 8,
            background: "#e6f4ff",
            border: "1px solid #91caff",
          }}
        >
          {renderValue(userAnswer)}
        </div>
      </div>
    </Space>
  );
}

function normalizeAnswerValues(value: unknown) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item));
  }
  if (value === null || value === undefined || value === "") {
    return [];
  }
  return [String(value)];
}

function renderJudgeAnswer(value: unknown) {
  const text = String(value ?? "").toLowerCase();
  if (text === "true") return "正确";
  if (text === "false") return "错误";
  return renderValue(value);
}

function labelize(key: string) {
  const labels: Record<string, string> = {
    id: "ID",
    title: "标题",
    summary: "摘要",
    user_id: "用户 ID",
    username: "用户名",
    team_id: "队伍 ID",
    team_name: "参赛单位",
    challenge_id: "题目 ID",
    flag: "Flag",
    is_correct: "正确",
    points: "分值",
    submitted_at: "提交时间",
    name: "名称",
    status: "状态",
    host_id: "宿主机 ID",
    created_at: "创建时间",
    email: "邮箱",
    role: "角色",
    total_score: "总分",
    is_active: "启用",
  };
  return labels[key] ?? key;
}
