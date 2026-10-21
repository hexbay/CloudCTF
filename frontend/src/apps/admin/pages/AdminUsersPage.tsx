import { DeleteOutlined, DownloadOutlined, EditOutlined, KeyOutlined, PlusOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Form, Input, Modal, Select, Space, Table, Tag, Tooltip, Typography, Upload, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";
import { formatTime, normalizeFilters, normalizeRows } from "./adminPageUtils";

type UserRow = {
  id: number;
  username: string;
  email: string;
  full_name?: string;
  tags?: string[] | string;
  teams?: Array<{ id: number; name: string }>;
  is_active?: boolean;
  created_at?: string;
};

type UserFormValues = {
  username: string;
  email: string;
  password?: string;
  full_name?: string;
  tags?: string;
};

export function AdminUsersPage() {
  const [rows, setRows] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<UserRow | null>(null);
  const [passwordRecord, setPasswordRecord] = useState<UserRow | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [removingId, setRemovingId] = useState<number | null>(null);
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const [form] = Form.useForm<UserFormValues>();
  const [passwordForm] = Form.useForm<{ password: string; confirmPassword: string }>();

  useEffect(() => {
    void load();
  }, []);

  async function load(nextFilters = filters) {
    setLoading(true);
    try {
      const result = await adminApi.list<UserRow>("/api-admin/users", {
        current: 1,
        pageSize: 200,
        ...normalizeFilters(nextFilters),
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载用户列表失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  }

  function openEdit(record: UserRow) {
    setEditing(record);
    form.setFieldsValue({
      username: record.username,
      email: record.email,
      full_name: record.full_name,
      tags: normalizeTags(record.tags).join(", "),
    });
    setModalOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    const payload = {
      username: values.username,
      email: values.email,
      password: values.password,
      full_name: values.full_name,
      tags: values.tags ? values.tags.split(",").map((item) => item.trim()).filter(Boolean) : [],
    };
    try {
      setSaving(true);
      if (editing) {
        await adminApi.update(`/api-admin/users/${editing.id}`, payload);
        message.success("用户已更新");
      } else {
        await adminApi.create("/api-admin/users", payload);
        message.success("用户已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存用户失败");
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: UserRow) {
    try {
      setRemovingId(record.id);
      await adminApi.remove(`/api-admin/users/${record.id}`);
      message.success("用户已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除用户失败");
    } finally {
      setRemovingId(null);
    }
  }

  async function updateEnabled(record: UserRow, enabled: boolean) {
    try {
      setTogglingId(record.id);
      await adminApi.action(`/api-admin/users/${record.id}/${enabled ? "enable" : "disable"}`, "PATCH");
      message.success(enabled ? "用户已启用" : "用户已禁用");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "更新用户状态失败");
    } finally {
      setTogglingId(null);
    }
  }

  async function submitPassword() {
    if (!passwordRecord) return;
    const values = await passwordForm.validateFields();
    try {
      setPasswordSaving(true);
      await adminApi.action(`/api-admin/users/${passwordRecord.id}/reset-password`, "POST", {
        password: values.password,
      });
      message.success("密码已重置");
      setPasswordRecord(null);
      passwordForm.resetFields();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "重置密码失败");
    } finally {
      setPasswordSaving(false);
    }
  }

  async function handleImport(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    try {
      setImporting(true);
      await adminApi.create("/api-admin/users/import", formData);
      message.success("导入成功");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导入用户失败");
    } finally {
      setImporting(false);
    }
    return false;
  }

  function handleExport() {
    setExporting(true);
    try {
      window.open("/api-admin/users/export", "_blank");
      message.success("已开始导出用户数据");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导出用户失败");
    } finally {
      setExporting(false);
    }
  }

  const columns: ColumnsType<UserRow> = [
    { title: "用户名", dataIndex: "username", width: 160 },
    { title: "邮箱", dataIndex: "email", width: 220 },
    { title: "姓名", dataIndex: "full_name", width: 140, render: (value) => value || "-" },
    {
      title: "标签",
      dataIndex: "tags",
      width: 220,
      render: (value) => (
        <Space size={[4, 4]} wrap>
          {normalizeTags(value).length ? normalizeTags(value).map((tag) => <Tag key={tag}>{tag}</Tag>) : "-"}
        </Space>
      ),
    },
    {
      title: "队伍",
      dataIndex: "teams",
      width: 220,
      render: (value: UserRow["teams"]) => (
        <Space size={[4, 4]} wrap>
          {value?.length ? value.map((team) => <Tag key={team.id}>{team.name}</Tag>) : "-"}
        </Space>
      ),
    },
    {
      title: "状态",
      dataIndex: "is_active",
      width: 90,
      render: (value) => <Tag color={value ? "green" : "default"}>{value ? "启用" : "禁用"}</Tag>,
    },
    { title: "创建时间", dataIndex: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "actions",
      width: 280,
      render: (_, row) => (
        <Space size={4} wrap className="admin-action-group">
          <Button type="text" size="small" className="admin-action-button" icon={<EditOutlined />} onClick={() => openEdit(row)}>
            编辑
          </Button>
          <Tooltip title="重置密码">
            <Button type="text" size="small" className="admin-action-button" icon={<KeyOutlined />} onClick={() => setPasswordRecord(row)}>
              密码
            </Button>
          </Tooltip>
          <Button type="text" size="small" loading={togglingId === row.id} className="admin-action-button" onClick={() => void updateEnabled(row, !row.is_active)}>
            {row.is_active ? "禁用" : "启用"}
          </Button>
          <Button
            type="text"
            size="small"
            danger
            loading={removingId === row.id}
            className="admin-action-button admin-action-button-danger"
            icon={<DeleteOutlined />}
            onClick={() => void remove(row)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">
        用户管理
      </Typography.Title>

      <AdminDataTableCard
        title="用户列表"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>用户名：</label>
              <Input allowClear value={filters.username ?? ""} onChange={(event) => setFilters((current) => ({ ...current, username: event.target.value }))} />
            </div>
            <div className="admin-search-field">
              <label>邮箱：</label>
              <Input allowClear value={filters.email ?? ""} onChange={(event) => setFilters((current) => ({ ...current, email: event.target.value }))} />
            </div>
            <div className="admin-search-field">
              <label>状态：</label>
              <Select
                allowClear
                value={filters.is_active || undefined}
                options={[
                  { label: "启用", value: "true" },
                  { label: "禁用", value: "false" },
                ]}
                onChange={(value) => setFilters((current) => ({ ...current, is_active: value ? String(value) : "" }))}
              />
            </div>
            <Space>
              <Button onClick={() => { setFilters({}); void load({}); }}>重置</Button>
              <Button type="primary" onClick={() => void load()}>查询</Button>
            </Space>
          </div>
        }
        actions={
          <>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>刷新</Button>
            <Upload showUploadList={false} beforeUpload={(file) => handleImport(file)}>
              <Button icon={<UploadOutlined />} loading={importing}>导入</Button>
            </Upload>
            <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExport}>导出</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建</Button>
          </>
        }
      >
        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
      </AdminDataTableCard>

      <Modal title={editing ? "编辑用户" : "新建用户"} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={720} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, message: "请输入邮箱" }]}>
            <Input />
          </Form.Item>
          {!editing ? (
            <Form.Item name="password" label="密码" rules={[{ required: true, message: "请输入密码" }]}>
              <Input.Password />
            </Form.Item>
          ) : null}
          <Form.Item name="full_name" label="姓名">
            <Input />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Input placeholder="多个标签用逗号分隔" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`重置密码：${passwordRecord?.username ?? ""}`} open={Boolean(passwordRecord)} onCancel={() => setPasswordRecord(null)} onOk={() => void submitPassword()} confirmLoading={passwordSaving} destroyOnClose>
        <Form form={passwordForm} layout="vertical">
          <Form.Item name="password" label="新密码" rules={[{ required: true, message: "请输入新密码" }, { min: 6, message: "密码至少 6 位" }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={["password"]}
            rules={[
              { required: true, message: "请再次输入新密码" },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue("password") === value) return Promise.resolve();
                  return Promise.reject(new Error("两次输入的密码不一致"));
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function normalizeTags(value: UserRow["tags"]) {
  if (Array.isArray(value)) return value.map(String);
  if (typeof value === "string") return value.split(",").map((item) => item.trim()).filter(Boolean);
  return [];
}
