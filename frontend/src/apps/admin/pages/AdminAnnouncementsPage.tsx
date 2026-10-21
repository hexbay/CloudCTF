import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { formatTime, normalizeRows } from "./adminPageUtils";

type AnnouncementRow = {
  id: number;
  title: string;
  summary?: string;
  content?: string;
  author_id?: number;
  published_at?: string;
  created_at?: string;
  updated_at?: string;
  is_published?: boolean;
  target_type?: string;
  target_id?: number;
  view_count?: number;
};

export function AdminAnnouncementsPage() {
  const [rows, setRows] = useState<AnnouncementRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AnnouncementRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form] = Form.useForm<AnnouncementRow>();

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<AnnouncementRow>("/api-admin/announcements", {
        current: 1,
        pageSize: 200,
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载公告列表失败");
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      is_published: true,
      target_type: "global",
    });
    setModalOpen(true);
  }

  function openEdit(record: AnnouncementRow) {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditing(null);
    form.resetFields();
  }

  async function submit() {
    try {
      const values = await form.validateFields();
      setSaving(true);
      const payload = {
        title: values.title,
        summary: values.summary,
        content: values.content,
        is_published: true,
        target_type: values.target_type || "global",
        target_id: values.target_id,
        author_id: values.author_id,
      };
      if (editing) {
        await adminApi.update(`/api-admin/announcements/${editing.id}`, payload);
        message.success("公告已更新");
      } else {
        await adminApi.create("/api-admin/announcements", payload);
        message.success("公告已创建");
      }
      closeModal();
      await load();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: AnnouncementRow) {
    try {
      setDeletingId(record.id);
      await adminApi.remove(`/api-admin/announcements/${record.id}`);
      message.success("公告已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除公告失败");
    } finally {
      setDeletingId(null);
    }
  }

  const columns: ColumnsType<AnnouncementRow> = [
    { title: "标题", dataIndex: "title", ellipsis: true },
    { title: "摘要", dataIndex: "summary", ellipsis: true, render: (value) => value || "-" },
    {
      title: "目标",
      key: "target",
      width: 140,
      render: (_, row) => (
        <Tag color={row.target_type === "competition" ? "blue" : "default"}>
          {row.target_type === "competition" ? `比赛 ${row.target_id ?? "-"}` : "全站"}
        </Tag>
      ),
    },
    { title: "浏览量", dataIndex: "view_count", width: 90, render: (value) => value ?? 0 },
    { title: "发布时间", dataIndex: "published_at", width: 180, render: formatTime },
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
          <Popconfirm title={`确认删除公告 ${row.title}？`} onConfirm={() => remove(row)}>
            <Button type="text" size="small" danger loading={deletingId === row.id} className="admin-action-button admin-action-button-danger" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">
        公告管理
      </Typography.Title>

      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>公告列表</Typography.Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建公告
            </Button>
          </Space>
        </div>

        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
      </Card>

      <Modal
        title={editing ? "编辑公告" : "新建公告"}
        open={modalOpen}
        onCancel={closeModal}
        onOk={() => void submit()}
        confirmLoading={saving}
        width={760}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="标题" rules={[{ required: true, message: "请输入公告标题" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="summary" label="摘要">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="content" label="内容" rules={[{ required: true, message: "请输入公告内容" }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
          <Form.Item name="target_type" label="目标类型">
            <Input placeholder="默认 global，可选 competition" />
          </Form.Item>
          <Form.Item name="target_id" label="目标 ID">
            <Input placeholder="可为空；competition 类型时填比赛 ID" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
