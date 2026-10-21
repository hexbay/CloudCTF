import { DeleteOutlined, EditOutlined, KeyOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Form, Input, Modal, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";
import { formatTime, normalizeFilters, normalizeRows } from "./adminPageUtils";

type AdminRow = {
  id: number;
  username: string;
  email: string;
  full_name?: string;
  role?: string;
  role_id?: number;
  is_active?: boolean;
  is_superuser?: boolean;
  created_at?: string;
};

type RoleRow = {
  id: number;
  name: string;
  description?: string;
};

type AdminFormValues = {
  username: string;
  email: string;
  password?: string;
  full_name?: string;
  role?: string;
  is_active?: boolean;
};

export function AdminAdminsPage() {
  const [rows, setRows] = useState<AdminRow[]>([]);
  const [roles, setRoles] = useState<RoleRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AdminRow | null>(null);
  const [passwordRecord, setPasswordRecord] = useState<AdminRow | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form] = Form.useForm<AdminFormValues>();
  const [passwordForm] = Form.useForm<{ password: string; confirmPassword: string }>();

  const adminRoleOptions = useMemo(
    () =>
      roles
        .filter((role) => role.name !== "player")
        .map((role) => ({
          label: role.description ? `${role.description} (${role.name})` : role.name,
          value: role.name,
        })),
    [roles],
  );

  useEffect(() => {
    void load();
    void loadRoles();
  }, []);

  async function load(nextFilters = filters) {
    setLoading(true);
    try {
      const result = await adminApi.list<AdminRow>("/api-admin/admins", {
        current: 1,
        pageSize: 200,
        ...normalizeFilters(nextFilters),
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载管理员列表失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadRoles() {
    try {
      const result = await adminApi.list<RoleRow>("/api-admin/roles", {
        current: 1,
        pageSize: 200,
      });
      setRoles(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载角色列表失败");
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      role: "admin",
      is_active: true,
    });
    setModalOpen(true);
  }

  function openEdit(record: AdminRow) {
    setEditing(record);
    form.setFieldsValue({
      username: record.username,
      email: record.email,
      full_name: record.full_name,
      role: record.role || "admin",
      is_active: record.is_active ?? true,
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
      role: values.role || "admin",
      is_active: values.is_active ?? true,
    };
    try {
      setSaving(true);
      if (editing) {
        await adminApi.update(`/api-admin/admins/${editing.id}`, payload);
        message.success("管理员已更新");
      } else {
        await adminApi.create("/api-admin/admins", payload);
        message.success("管理员已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存管理员失败");
    } finally {
      setSaving(false);
    }
  }

  async function submitPassword() {
    if (!passwordRecord) {
      return;
    }
    const values = await passwordForm.validateFields();
    try {
      setPasswordSaving(true);
      await adminApi.update(`/api-admin/admins/${passwordRecord.id}`, {
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

  async function remove(record: AdminRow) {
    try {
      setDeletingId(record.id);
      await adminApi.remove(`/api-admin/admins/${record.id}`);
      message.success("管理员已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除管理员失败");
    } finally {
      setDeletingId(null);
    }
  }

  const columns: ColumnsType<AdminRow> = [
    { title: "用户名", dataIndex: "username", width: 160 },
    { title: "邮箱", dataIndex: "email", width: 220 },
    { title: "姓名", dataIndex: "full_name", width: 140, render: (value) => value || "-" },
    {
      title: "角色",
      dataIndex: "role",
      width: 160,
      render: (value) => <Tag color={value === "superadmin" ? "red" : "blue"}>{roleLabel(String(value ?? "admin"))}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "is_active",
      width: 90,
      render: (value) => <Tag color={value ? "green" : "default"}>{value ? "启用" : "禁用"}</Tag>,
    },
    {
      title: "超级管理员",
      dataIndex: "is_superuser",
      width: 110,
      render: (value) => <Tag color={value ? "red" : "default"}>{value ? "是" : "否"}</Tag>,
    },
    { title: "创建时间", dataIndex: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "actions",
      width: 240,
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
          <Button
            type="text"
            size="small"
            danger
            loading={deletingId === row.id}
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
        管理员管理
      </Typography.Title>

      <AdminDataTableCard
        title="管理员列表"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>关键词：</label>
              <Input allowClear value={filters.keyword ?? ""} onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))} />
            </div>
            <div className="admin-search-field">
              <label>角色：</label>
              <Select
                allowClear
                options={adminRoleOptions}
                value={filters.role || undefined}
                onChange={(value) => setFilters((current) => ({ ...current, role: value ? String(value) : "" }))}
              />
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
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
          </>
        }
      >
        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
      </AdminDataTableCard>

      <Modal title={editing ? "编辑管理员" : "新建管理员"} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={720} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, message: "请输入邮箱" }]}>
            <Input />
          </Form.Item>
          {!editing ? (
            <Form.Item name="password" label="密码" rules={[{ required: true, message: "请输入密码" }, { min: 6, message: "密码至少 6 位" }]}>
              <Input.Password />
            </Form.Item>
          ) : null}
          <Form.Item name="full_name" label="姓名">
            <Input />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true, message: "请选择角色" }]}>
            <Select options={adminRoleOptions} />
          </Form.Item>
          <Form.Item name="is_active" label="状态">
            <Select options={[{ label: "启用", value: true }, { label: "禁用", value: false }]} />
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

function roleLabel(value: string) {
  if (value === "superadmin") return "超级管理员";
  if (value === "admin") return "管理员";
  return value;
}
