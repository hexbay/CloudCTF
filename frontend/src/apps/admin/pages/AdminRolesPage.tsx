import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Checkbox, Form, Input, Modal, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { formatTime, normalizeRows } from "./adminPageUtils";

type RoleRow = {
  id: number;
  name: string;
  description?: string;
  permissions?: string[] | string;
  created_at?: string;
};

type RoleFormValues = {
  name: string;
  description?: string;
  permissions?: string[];
};

export function AdminRolesPage() {
  const [rows, setRows] = useState<RoleRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<RoleRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [permissionOptions, setPermissionOptions] = useState<string[]>([]);
  const [form] = Form.useForm<RoleFormValues>();

  useEffect(() => {
    void load();
    void adminApi
      .get<string[]>("/api-admin/roles/permissions")
      .then(setPermissionOptions)
      .catch(() => setPermissionOptions([]));
  }, []);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<RoleRow>("/api-admin/roles", {
        current: 1,
        pageSize: 200,
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载角色列表失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  }

  function openEdit(record: RoleRow) {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      permissions: normalizePermissions(record.permissions),
    });
    setModalOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    const payload = {
      name: values.name,
      description: values.description,
      permissions: values.permissions ?? [],
    };
    try {
      setSaving(true);
      if (editing) {
        await adminApi.update(`/api-admin/roles/${editing.id}`, payload);
        message.success("角色已更新");
      } else {
        await adminApi.create("/api-admin/roles", payload);
        message.success("角色已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存角色失败");
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: RoleRow) {
    try {
      setDeletingId(record.id);
      await adminApi.remove(`/api-admin/roles/${record.id}`);
      message.success("角色已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除角色失败");
    } finally {
      setDeletingId(null);
    }
  }

  const columns: ColumnsType<RoleRow> = [
    { title: "名称", dataIndex: "name", width: 180 },
    { title: "描述", dataIndex: "description", ellipsis: true, render: (value) => value || "-" },
    {
      title: "权限",
      dataIndex: "permissions",
      width: 360,
      render: (value) => (
        <Space size={[4, 4]} wrap>
          {normalizePermissions(value).length ? normalizePermissions(value).map((item) => <Tag key={item}>{item}</Tag>) : "-"}
        </Space>
      ),
    },
    { title: "创建时间", dataIndex: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "actions",
      width: 180,
      render: (_, row) => (
        <Space size={4} wrap className="admin-action-group">
          <Button type="text" size="small" className="admin-action-button" icon={<EditOutlined />} onClick={() => openEdit(row)}>
            编辑
          </Button>
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
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>角色列表</Typography.Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
          </Space>
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
      </Card>

      <Modal title={editing ? "编辑角色" : "新建角色"} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={720} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入角色名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="permissions" label="权限">
            <Checkbox.Group className="admin-permission-options">
              <Space direction="vertical" size={8}>
                {permissionOptions.map((permission) => (
                  <Checkbox key={permission} value={permission}>
                    {permission}
                  </Checkbox>
                ))}
              </Space>
            </Checkbox.Group>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function normalizePermissions(value: RoleRow["permissions"]) {
  if (Array.isArray(value)) return value.map(String);
  if (typeof value === "string") return value.split(",").map((item) => item.trim()).filter(Boolean);
  return [];
}
