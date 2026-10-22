import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type React from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";

type CompetitionRow = {
  id: number;
  name: string;
  type: "ctf" | "awd" | "theory" | "mixed";
  participant_mode?: "individual" | "team";
  status: "draft" | "upcoming" | "ongoing" | "finished" | "archived";
  start_time?: string;
  end_time?: string;
  is_public?: boolean;
  registration_open?: boolean;
  created_at?: string;
  description?: string;
};

const competitionTypeOptions = [
  { label: "CTF", value: "ctf" },
  { label: "AWD", value: "awd" },
  { label: "理论赛", value: "theory" },
  { label: "混合赛", value: "mixed" },
];

const competitionStatusOptions = [
  { label: "草稿", value: "draft" },
  { label: "未开始", value: "upcoming" },
  { label: "进行中", value: "ongoing" },
  { label: "已结束", value: "finished" },
  { label: "已归档", value: "archived" },
];

const participantModeOptions = [
  { label: "个人赛", value: "individual" },
  { label: "团队赛", value: "team" },
];

export function AdminCompetitionListPage() {
  const navigate = useNavigate();
  const [rows, setRows] = useState<CompetitionRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState<CompetitionRow | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [removingId, setRemovingId] = useState<number | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<number | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    void load();
  }, [current, pageSize]);

  const columns = useMemo<ColumnsType<CompetitionRow>>(
    () => [
      {
        title: "名称",
        dataIndex: "name",
        width: 260,
        render: (value, record) => (
          <Button
            type="link"
            className="competition-name-link"
            onClick={() => navigate(`/admin/competition/${record.id}`)}
          >
            {value}
          </Button>
        ),
      },
      {
        title: "类型",
        dataIndex: "type",
        width: 110,
        render: (value) => (
          <Tag className={`competition-tag competition-tag-type-${value}`}>
            {typeLabel(value)}
          </Tag>
        ),
      },
      {
        title: "参赛类型",
        dataIndex: "participant_mode",
        width: 110,
        render: (value) => <Tag>{participantModeLabel(value)}</Tag>,
      },
      {
        title: "状态",
        dataIndex: "status",
        width: 120,
        render: (value) => (
          <Tag className={`competition-tag competition-tag-status-${value}`}>
            {statusLabel(value)}
          </Tag>
        ),
      },
      {
        title: "开始时间",
        dataIndex: "start_time",
        width: 190,
        render: (value) => formatTime(value),
      },
      {
        title: "结束时间",
        dataIndex: "end_time",
        width: 190,
        render: (value) => formatTime(value),
      },
      {
        title: "公开",
        dataIndex: "is_public",
        width: 80,
        render: (value) => (
          <Tag
            className={`competition-tag ${value ? "competition-tag-bool-true" : "competition-tag-bool-false"}`}
          >
            {value ? "公开" : "私有"}
          </Tag>
        ),
      },
      {
        title: "开放报名",
        dataIndex: "registration_open",
        width: 100,
        render: (value) => (
          <Tag
            className={`competition-tag ${value ? "competition-tag-bool-true" : "competition-tag-bool-false"}`}
          >
            {value ? "开放" : "关闭"}
          </Tag>
        ),
      },
      {
        title: "创建时间",
        dataIndex: "created_at",
        width: 190,
        render: (value) => formatTime(value),
      },
      {
        title: "操作",
        key: "actions",
        width: 300,
        render: (_, record) => (
          <Space size={4} wrap className="admin-action-group">
            <ActionLink
              icon={<EyeOutlined />}
              label="详情"
              onClick={() => navigate(`/admin/competition/${record.id}`)}
            />
            <ActionLink
              icon={<EditOutlined />}
              label="编辑"
              onClick={() => openEdit(record)}
            />
            {renderStatusAction(record)}
            <ActionLink
              danger
              loading={removingId === record.id}
              icon={<DeleteOutlined />}
              label="删除"
              onClick={() => void remove(record)}
            />
          </Space>
        ),
      },
    ],
    [navigate, removingId, statusUpdatingId],
  );

  async function load(nextFilters = filters, nextCurrent = current) {
    setLoading(true);
    try {
      const result = await adminApi.list<CompetitionRow>(
        "/api-admin/competitions",
        {
          current: nextCurrent,
          pageSize,
          ...nextFilters,
        },
      );
      const normalized = normalizePageResult(result);
      setRows(normalized.rows);
      setTotal(normalized.total);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载比赛列表失败",
      );
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      type: "ctf",
      participant_mode: "individual",
      is_public: true,
      registration_open: true,
    });
    setModalOpen(true);
  }

  function openEdit(record: CompetitionRow) {
    setEditing(record);
    form.setFieldsValue({
      ...record,
      start_time: record.start_time ? dayjs(record.start_time) : undefined,
      end_time: record.end_time ? dayjs(record.end_time) : undefined,
    });
    setModalOpen(true);
  }

  async function submit() {
    try {
      setSaving(true);
      const values = await form.validateFields();
      const payload = {
        ...values,
        start_time: values.start_time
          ? values.start_time.toISOString()
          : undefined,
        end_time: values.end_time ? values.end_time.toISOString() : undefined,
      };
      if (editing) {
        await adminApi.update(`/api-admin/competitions/${editing.id}`, payload);
        message.success("比赛已更新");
      } else {
        await adminApi.create("/api-admin/competitions", payload);
        message.success("比赛已创建");
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

  async function remove(record: CompetitionRow) {
    try {
      setRemovingId(record.id);
      await adminApi.remove(`/api-admin/competitions/${record.id}`);
      message.success("比赛已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除比赛失败");
    } finally {
      setRemovingId(null);
    }
  }

  async function updateStatus(
    record: CompetitionRow,
    status: CompetitionRow["status"],
  ) {
    try {
      setStatusUpdatingId(record.id);
      await adminApi.action(
        `/api-admin/competitions/${record.id}/status`,
        "PATCH",
        { status },
      );
      message.success("状态已更新");
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "更新比赛状态失败",
      );
    } finally {
      setStatusUpdatingId(null);
    }
  }

  function renderStatusAction(record: CompetitionRow) {
    if (record.status === "draft") {
      return (
        <ActionLink
          loading={statusUpdatingId === record.id}
          label="发布"
          onClick={() => void updateStatus(record, "upcoming")}
        />
      );
    }
    if (record.status === "upcoming") {
      return (
        <ActionLink
          loading={statusUpdatingId === record.id}
          label="开始"
          onClick={() => void updateStatus(record, "ongoing")}
        />
      );
    }
    if (record.status === "ongoing") {
      return (
        <ActionLink
          loading={statusUpdatingId === record.id}
          label="结束"
          onClick={() => void updateStatus(record, "finished")}
        />
      );
    }
    if (record.status === "finished") {
      return (
        <ActionLink
          loading={statusUpdatingId === record.id}
          label="归档"
          onClick={() => void updateStatus(record, "archived")}
        />
      );
    }
    return null;
  }

  return (
    <div className="admin-page competition-list-page">
      <AdminDataTableCard
        title="比赛列表"
        className="competition-list-card"
        search={
          <div className="admin-search-form competition-search-form">
            <div className="admin-search-field">
              <label>名称：</label>
              <Input
                allowClear
                placeholder="请输入比赛名称"
                value={filters.name ?? ""}
                onChange={(event) =>
                  setFilters((currentFilters) => ({
                    ...currentFilters,
                    name: event.target.value,
                  }))
                }
              />
            </div>
            <div className="admin-search-field">
              <label>类型：</label>
              <Select
                allowClear
                placeholder="请选择类型"
                options={competitionTypeOptions}
                value={filters.type || undefined}
                onChange={(value) =>
                  setFilters((currentFilters) => ({
                    ...currentFilters,
                    type: value ? String(value) : "",
                  }))
                }
              />
            </div>
            <div className="admin-search-field">
              <label>参赛类型：</label>
              <Select
                allowClear
                placeholder="请选择参赛类型"
                options={participantModeOptions}
                value={filters.participant_mode || undefined}
                onChange={(value) =>
                  setFilters((currentFilters) => ({
                    ...currentFilters,
                    participant_mode: value ? String(value) : "",
                  }))
                }
              />
            </div>
            <div className="admin-search-field">
              <label>状态：</label>
              <Select
                allowClear
                placeholder="请选择状态"
                options={competitionStatusOptions}
                value={filters.status || undefined}
                onChange={(value) =>
                  setFilters((currentFilters) => ({
                    ...currentFilters,
                    status: value ? String(value) : "",
                  }))
                }
              />
            </div>
            <Space wrap>
              <Button
                onClick={() => {
                  setFilters({});
                  setCurrent(1);
                  void load({}, 1);
                }}
              >
                重置
              </Button>
              <Button
                type="primary"
                onClick={() => {
                  setCurrent(1);
                  void load(filters, 1);
                }}
              >
                查询
              </Button>
            </Space>
          </div>
        }
        actions={
          <>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
          </>
        }
      >
        <Table
          className="competition-table"
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
            },
          }}
        />
      </AdminDataTableCard>

      <Modal
        title={editing ? "编辑比赛" : "新建比赛"}
        open={modalOpen}
        width={720}
        destroyOnHidden
        onCancel={() => {
          setModalOpen(false);
          setEditing(null);
          form.resetFields();
        }}
        onOk={() => void submit()}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: "请输入比赛名称" }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: "请选择比赛类型" }]}
          >
            <Select options={competitionTypeOptions} />
          </Form.Item>
          <Form.Item
            name="participant_mode"
            label="参赛类型"
            rules={[{ required: true, message: "请选择参赛类型" }]}
          >
            <Select options={participantModeOptions} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Space style={{ display: "flex" }} align="start">
            <Form.Item name="start_time" label="开始时间">
              <DatePicker showTime style={{ width: 250 }} />
            </Form.Item>
            <Form.Item name="end_time" label="结束时间">
              <DatePicker showTime style={{ width: 250 }} />
            </Form.Item>
          </Space>
          <Space style={{ display: "flex" }} align="start">
            <Form.Item name="is_public" label="公开" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item
              name="registration_open"
              label="开放报名"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  );
}

function ActionLink({
  icon,
  label,
  danger = false,
  loading = false,
  onClick,
}: {
  icon?: React.ReactNode;
  label: string;
  danger?: boolean;
  loading?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="text"
      size="small"
      danger={danger}
      loading={loading}
      className={`admin-action-button${danger ? " admin-action-button-danger" : ""}`}
      icon={icon}
      onClick={onClick}
    >
      {label}
    </Button>
  );
}

function normalizePageResult(result: unknown) {
  if (Array.isArray(result)) {
    return { rows: result as CompetitionRow[], total: result.length };
  }
  const object = (result ?? {}) as {
    data?: CompetitionRow[];
    items?: CompetitionRow[];
    list?: CompetitionRow[];
    total?: number;
  };
  const rows = object.data ?? object.items ?? object.list ?? [];
  return { rows, total: object.total ?? rows.length };
}

function formatTime(value?: string) {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

function typeLabel(value: CompetitionRow["type"]) {
  return (
    competitionTypeOptions.find((item) => item.value === value)?.label ?? value
  );
}

function statusLabel(value: CompetitionRow["status"]) {
  return (
    competitionStatusOptions.find((item) => item.value === value)?.label ??
    value
  );
}

function participantModeLabel(value?: CompetitionRow["participant_mode"]) {
  return (
    participantModeOptions.find((item) => item.value === value)?.label ??
    value ??
    "-"
  );
}
