import { DeleteOutlined, DownloadOutlined, EditOutlined, PlusOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography, Upload, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";

type TheoryRow = {
  id: number;
  question: string;
  question_type: "text" | "judge" | "single" | "multiple";
  points: number;
  category?: string;
  difficulty?: number;
  context?: Record<string, unknown>;
  created_at?: string;
};

const typeOptions = [
  { label: "问答题", value: "text" },
  { label: "判断题", value: "judge" },
  { label: "单选题", value: "single" },
  { label: "多选题", value: "multiple" },
];

export function AdminTheoryPage() {
  const [rows, setRows] = useState<TheoryRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<TheoryRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<TheoryRow>("/api-admin/theory/questions", { current: 1, pageSize: 200 });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载理论题失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ question_type: "single", points: 5, difficulty: 1, optionsText: "", answerText: "" });
    setModalOpen(true);
  }

  function openEdit(row: TheoryRow) {
    setEditing(row);
    const context = row.context ?? {};
    form.setFieldsValue({
      ...row,
      optionsText: Array.isArray(context.options)
        ? context.options.map((option) => String((option as { text?: unknown }).text ?? "")).join("\n")
        : "",
      answerText: Array.isArray(context.options)
        ? context.options
            .map((option, index) =>
              (option as { checked?: unknown }).checked ? String.fromCharCode(65 + index) : "",
            )
            .filter(Boolean)
            .join(",")
        : String(context.answer ?? ""),
    });
    setModalOpen(true);
  }

  async function submit() {
    try {
      setSaving(true);
      const values = await form.validateFields();
      const payload = {
        question: values.question,
        question_type: values.question_type,
        points: values.points,
        category: values.category,
        difficulty: values.difficulty,
        context: buildContext(values),
      };
      if (editing) {
        await adminApi.update(`/api-admin/theory/questions/${editing.id}`, payload);
        message.success("理论题已更新");
      } else {
        await adminApi.create("/api-admin/theory/questions", payload);
        message.success("理论题已创建");
      }
      setModalOpen(false);
      form.resetFields();
      setEditing(null);
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
    try {
      setDeletingId(id);
      await adminApi.remove(`/api-admin/theory/questions/${id}`);
      message.success("理论题已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除理论题失败");
    } finally {
      setDeletingId(null);
    }
  }

  async function upload(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    try {
      setImporting(true);
      await adminApi.create("/api-admin/theory/questions/import", formData);
      message.success("导入成功");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导入理论题失败");
    } finally {
      setImporting(false);
    }
    return false;
  }

  function handleExport() {
    setExporting(true);
    try {
      window.open("/api-admin/theory/questions/export", "_blank");
      message.success("已开始导出理论题数据");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导出理论题失败");
    } finally {
      setExporting(false);
    }
  }

  const columns: ColumnsType<TheoryRow> = [
    { title: "ID", dataIndex: "id", width: 80 },
    { title: "题干", dataIndex: "question", ellipsis: true },
    { title: "题型", dataIndex: "question_type", width: 120, render: renderType },
    { title: "分类", dataIndex: "category", width: 120, render: (value) => value || "-" },
    { title: "分值", dataIndex: "points", width: 90 },
    { title: "难度", dataIndex: "difficulty", width: 90 },
    { title: "答案", width: 220, render: (_, row) => renderAnswer(row.context) },
    {
      title: "操作",
      key: "actions",
      width: 160,
      render: (_, row) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(row)}>
            编辑
          </Button>
          <Button size="small" danger loading={deletingId === row.id} icon={<DeleteOutlined />} onClick={() => void remove(row.id)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">
        理论题库
      </Typography.Title>
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>题库管理</Typography.Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Upload showUploadList={false} beforeUpload={upload}>
              <Button icon={<UploadOutlined />} loading={importing}>导入</Button>
            </Upload>
            <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExport}>
              导出
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建题目
            </Button>
          </Space>
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} pagination={{ pageSize: 10 }} />
      </Card>

      <Modal
        title={editing ? "编辑理论题" : "新建理论题"}
        open={modalOpen}
        width={760}
        destroyOnClose
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
          form.resetFields();
        }}
        onOk={() => void submit()}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="question" label="题干" rules={[{ required: true, message: "请输入题干" }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
          <Space style={{ display: "flex" }} align="start">
            <Form.Item name="question_type" label="题型" rules={[{ required: true, message: "请选择题型" }]}>
              <Select style={{ width: 180 }} options={typeOptions} />
            </Form.Item>
            <Form.Item name="category" label="分类">
              <Input style={{ width: 180 }} />
            </Form.Item>
            <Form.Item name="points" label="分值" rules={[{ required: true, message: "请输入分值" }]}>
              <InputNumber style={{ width: 120 }} min={1} />
            </Form.Item>
            <Form.Item name="difficulty" label="难度">
              <InputNumber style={{ width: 120 }} min={1} max={5} />
            </Form.Item>
          </Space>
          <Form.Item name="optionsText" label="选项">
            <Input.TextArea rows={4} placeholder="单选/多选题每行一个选项，例如：SQL 注入" />
          </Form.Item>
          <Form.Item name="answerText" label="答案">
            <Input placeholder="判断题填 true/false，多选题用逗号分隔，其它题型填写标准答案" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function buildContext(values: Record<string, unknown>) {
  const type = String(values.question_type ?? "text");
  const options = String(values.optionsText ?? "")
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);
  const answerText = String(values.answerText ?? "").trim();
  const answerKeys = answerText.split(",").map((item) => item.trim().toUpperCase()).filter(Boolean);
  if (type === "multiple") {
    return {
      options: options.map((text, index) => ({
        text,
        checked: answerKeys.includes(String.fromCharCode(65 + index)),
      })),
    };
  }
  if (type === "judge") {
    return { answer: answerText === "true" || answerText === "1" || answerText === "是" };
  }
  if (type === "single") {
    return {
      options: options.map((text, index) => ({
        text,
        checked: answerText.toUpperCase() === String.fromCharCode(65 + index),
      })),
    };
  }
  return { answer: answerText };
}

function normalizeRows(result: unknown): TheoryRow[] {
  if (Array.isArray(result)) return result as TheoryRow[];
  const object = (result ?? {}) as { data?: TheoryRow[]; list?: TheoryRow[]; items?: TheoryRow[] };
  return object.data ?? object.list ?? object.items ?? [];
}

function renderType(value: TheoryRow["question_type"]) {
  const item = typeOptions.find((option) => option.value === value);
  return <Tag>{item?.label ?? value}</Tag>;
}

function renderAnswer(context?: Record<string, unknown>) {
  if (!context) return "-";
  if (Array.isArray(context.options)) {
    return context.options
      .map((option, index) =>
        (option as { checked?: unknown }).checked ? String.fromCharCode(65 + index) : "",
      )
      .filter(Boolean)
      .join(", ") || "-";
  }
  const answer = context.answer;
  if (Array.isArray(answer)) return answer.join(", ");
  if (typeof answer === "boolean") return answer ? "true" : "false";
  return String(answer ?? "-");
}
