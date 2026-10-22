import { DeleteOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Form, Input, InputNumber, Modal, Popconfirm, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "@shared/api/admin";

type ExamRecord = {
  id: number;
  competition_id?: number | null;
  title: string;
  total_score: number;
  is_active: boolean;
};

type TheoryQuestion = {
  id: number;
  question: string;
  question_type: "text" | "judge" | "single" | "multiple";
  points: number;
  context?: Record<string, unknown>;
};

export function AdminExamPage() {
  const navigate = useNavigate();
  const [rows, setRows] = useState<ExamRecord[]>([]);
  const [questions, setQuestions] = useState<TheoryQuestion[]>([]);
  const [loading, setLoading] = useState(false);
  const [quickCreateModalVisible, setQuickCreateModalVisible] = useState(false);
  const [manualCreateModalVisible, setManualCreateModalVisible] = useState(false);
  const [quickCreating, setQuickCreating] = useState(false);
  const [manualCreating, setManualCreating] = useState(false);
  const [quickCreateForm] = Form.useForm();
  const [manualCreateForm] = Form.useForm();

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [examResult, statsResult] = await Promise.all([
        adminApi.list<ExamRecord>("/api-admin/exam", { current: 1, pageSize: 200 }),
        adminApi.list<{ questions: TheoryQuestion[] }>("/api-admin/theory/question/stats"),
      ]);
      setRows(normalizeRows(examResult));
      setQuestions((statsResult as unknown as { questions?: TheoryQuestion[] }).questions ?? []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载试卷列表失败");
    } finally {
      setLoading(false);
    }
  }

  async function handleQuickCreate() {
    try {
      const values = await quickCreateForm.validateFields();
      setQuickCreating(true);
      await adminApi.create("/api-admin/exam/quick-create", {
        title: values.title,
        count:
          Number(values.single_count ?? 0) +
          Number(values.multiple_count ?? 0) +
          Number(values.judge_count ?? 0) +
          Number(values.text_count ?? 0),
      });
      message.success("试卷创建成功");
      setQuickCreateModalVisible(false);
      quickCreateForm.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建试卷失败");
    } finally {
      setQuickCreating(false);
    }
  }

  async function handleManualCreate() {
    try {
      const values = await manualCreateForm.validateFields();
      setManualCreating(true);
      await adminApi.create("/api-admin/exam", {
        title: values.title,
        total_score: sumQuestionPoints(values.question_ids, questions),
        is_active: true,
        context: {
          questions: questions
            .filter((question) => (values.question_ids ?? []).includes(question.id))
            .map((question) => ({ ...question, context: { ...(question.context ?? {}) } })),
        },
      });
      message.success("试卷创建成功");
      setManualCreateModalVisible(false);
      manualCreateForm.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建试卷失败");
    } finally {
      setManualCreating(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await adminApi.remove(`/api-admin/exam/${id}`);
      message.success("试卷已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除试卷失败");
    }
  }

  return (
    <div className="admin-page">
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>已创建试卷</Typography.Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setQuickCreateModalVisible(true)}>
              快速创建
            </Button>
            <Button icon={<PlusOutlined />} onClick={() => setManualCreateModalVisible(true)}>
              手动创建
            </Button>
          </Space>
        </div>
        <Table
          rowKey="id"
          loading={loading}
          columns={[
            { title: "标题", dataIndex: "title" },
            { title: "总分", dataIndex: "total_score", width: 90 },
            {
              title: "启用",
              dataIndex: "is_active",
              width: 80,
              render: (value: boolean) => <Tag color={value ? "green" : "default"}>{value ? "是" : "否"}</Tag>,
            },
            {
              title: "操作",
              key: "actions",
              width: 220,
              render: (_, row) => (
                <Space>
                  <Button size="small" onClick={() => navigate(`/admin/theory-exams/${row.id}`)}>
                    查看/编辑
                  </Button>
                  <Popconfirm title="确认删除该试卷？" onConfirm={() => void handleDelete(row.id)}>
                    <Button size="small" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
          dataSource={rows}
          pagination={{ pageSize: 10 }}
          locale={{ emptyText: <Empty description="暂无试卷" /> }}
        />
      </Card>

      <Modal
        title="快速创建试卷"
        open={quickCreateModalVisible}
        onCancel={() => setQuickCreateModalVisible(false)}
        onOk={() => void handleQuickCreate()}
        confirmLoading={quickCreating}
        width={600}
      >
        <Form form={quickCreateForm} layout="vertical">
          <Form.Item label="试卷标题" name="title" rules={[{ required: true, message: "请输入试卷标题" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="单选题数量" name="single_count" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="多选题数量" name="multiple_count" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="判断题数量" name="judge_count" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="问答题数量" name="text_count" initialValue={0}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="手动创建试卷"
        open={manualCreateModalVisible}
        onCancel={() => setManualCreateModalVisible(false)}
        onOk={() => void handleManualCreate()}
        confirmLoading={manualCreating}
        width={900}
      >
        <Form form={manualCreateForm} layout="vertical">
          <Form.Item label="试卷标题" name="title" rules={[{ required: true, message: "请输入试卷标题" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="选择题目" name="question_ids" rules={[{ required: true, message: "请选择题目" }]}>
            <Table
              rowSelection={{ type: "checkbox", onChange: (keys) => manualCreateForm.setFieldsValue({ question_ids: keys }) }}
              columns={[
                { title: "ID", dataIndex: "id", width: 70 },
                { title: "题目", dataIndex: "question", ellipsis: true },
                { title: "类型", dataIndex: "question_type", width: 90 },
                { title: "分值", dataIndex: "points", width: 70 },
              ]}
              dataSource={questions}
              rowKey="id"
              pagination={{ pageSize: 6 }}
              size="small"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function normalizeRows<T>(result: unknown): T[] {
  if (Array.isArray(result)) return result as T[];
  const object = (result ?? {}) as { data?: T[]; items?: T[]; list?: T[] };
  return object.data ?? object.items ?? object.list ?? [];
}

function sumQuestionPoints(ids: number[], questions: TheoryQuestion[]) {
  const lookup = new Map(questions.map((item) => [item.id, item.points]));
  return ids.reduce((sum, id: number) => sum + Number(lookup.get(id) ?? 0), 0);
}
