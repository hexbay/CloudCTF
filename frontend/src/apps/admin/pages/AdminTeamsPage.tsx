import { DeleteOutlined, DownloadOutlined, EditOutlined, PlusOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Form, Input, InputNumber, Modal, Space, Table, Tag, Upload, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";
import { formatTime, normalizeFilters, normalizeRows } from "./adminPageUtils";

type TeamRow = {
  id: number;
  name: string;
  owner?: { id: number; username?: string; name?: string } | string;
  owner_id?: number;
  member_count?: number;
  is_blacklisted?: boolean;
  description?: string;
  created_at?: string;
};

type TeamFormValues = {
  name: string;
  owner_id?: number;
  description?: string;
};

export function AdminTeamsPage() {
  const [rows, setRows] = useState<TeamRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<TeamRow | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form] = Form.useForm<TeamFormValues>();

  useEffect(() => {
    void load();
  }, []);

  async function load(nextFilters = filters) {
    setLoading(true);
    try {
      const result = await adminApi.list<TeamRow>("/api-admin/teams", {
        current: 1,
        pageSize: 200,
        ...normalizeFilters(nextFilters),
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载队伍列表失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  }

  function openEdit(record: TeamRow) {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      owner_id: record.owner_id ?? (typeof record.owner === "object" ? record.owner?.id : undefined),
      description: record.description,
    });
    setModalOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    try {
      setSaving(true);
      if (editing) {
        await adminApi.update(`/api-admin/teams/${editing.id}`, values);
        message.success("队伍已更新");
      } else {
        await adminApi.create("/api-admin/teams", values);
        message.success("队伍已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存队伍失败");
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: TeamRow) {
    try {
      setDeletingId(record.id);
      await adminApi.remove(`/api-admin/teams/${record.id}`);
      message.success("队伍已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除队伍失败");
    } finally {
      setDeletingId(null);
    }
  }

  async function handleImport(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    try {
      setImporting(true);
      await adminApi.create("/api-admin/teams/import", formData);
      message.success("导入成功");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导入队伍失败");
    } finally {
      setImporting(false);
    }
    return false;
  }

  function handleExport() {
    setExporting(true);
    try {
      window.open("/api-admin/teams/export", "_blank");
      message.success("已开始导出队伍数据");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "导出队伍失败");
    } finally {
      setExporting(false);
    }
  }

  const columns: ColumnsType<TeamRow> = [
    { title: "名称", dataIndex: "name", width: 220 },
    {
      title: "队长",
      dataIndex: "owner",
      width: 160,
      render: (value, row) => {
        if (typeof value === "object" && value) return value.username || value.name || value.id;
        return value || row.owner_id || "-";
      },
    },
    { title: "成员数", dataIndex: "member_count", width: 90, render: (value) => value ?? 0 },
    {
      title: "状态",
      dataIndex: "is_blacklisted",
      width: 100,
      render: (value) => <Tag color={value ? "red" : "green"}>{value ? "黑名单" : "正常"}</Tag>,
    },
    { title: "描述", dataIndex: "description", ellipsis: true, render: (value) => value || "-" },
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
      <AdminDataTableCard
        title="队伍列表"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>队伍名称：</label>
              <Input allowClear value={filters.name ?? ""} onChange={(event) => setFilters((current) => ({ ...current, name: event.target.value }))} />
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

      <Modal title={editing ? "编辑队伍" : "新建队伍"} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={720} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="队伍名称" rules={[{ required: true, message: "请输入队伍名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="owner_id" label="队长 ID">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
