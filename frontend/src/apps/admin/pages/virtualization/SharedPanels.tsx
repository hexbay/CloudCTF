import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Button,
  Descriptions,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from "antd";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import type { RowData } from "./types";
import { columnLabel, normalizeRows, renderValue } from "./utils";

export function DetailModal({
  title,
  record,
  onClose,
}: {
  title: string;
  record: RowData | null;
  onClose: () => void;
}) {
  return (
    <Modal
      title={title}
      open={Boolean(record)}
      footer={null}
      onCancel={onClose}
      width={760}
    >
      <Descriptions column={1} size="small" bordered>
        {record
          ? Object.entries(record).map(([key, value]) => (
              <Descriptions.Item key={key} label={columnLabel(key)}>
                {renderValue(value, key)}
              </Descriptions.Item>
            ))
          : null}
      </Descriptions>
    </Modal>
  );
}

export function ResourceRepositoriesModal({
  open,
  onClose,
  onSynced,
}: {
  open: boolean;
  onClose: () => void;
  onSynced?: () => void | Promise<void>;
}) {
  const [rows, setRows] = useState<RowData[]>([]);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<RowData | null>(null);
  const [detail, setDetail] = useState<RowData | null>(null);
  const [syncingId, setSyncingId] = useState<string | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    if (!open) {
      return;
    }
    void load();
  }, [open]);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        "/api-admin/resource-repositories",
        {
          current: 1,
          pageSize: 200,
        },
      );
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载资源仓库失败",
      );
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing({});
    form.resetFields();
    form.setFieldsValue({
      provider: "git",
      enabled: true,
      repo_type: "vul",
      repository_branch: "main",
    });
  }

  function openEdit(row: RowData) {
    setEditing(row);
    form.setFieldsValue(row);
  }

  async function save() {
    const values = await form.validateFields();
    try {
      if (editing?.id) {
        await adminApi.update(
          `/api-admin/resource-repositories/${editing.id}`,
          values,
        );
        message.success("资源仓库已更新");
      } else {
        await adminApi.create("/api-admin/resource-repositories", values);
        message.success("资源仓库已创建");
      }
      setEditing(null);
      await load();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "保存资源仓库失败",
      );
    }
  }

  async function sync(row: RowData) {
    const id = String(row.id ?? "");
    setSyncingId(id);
    try {
      const result = await adminApi.action<{
        message?: string;
        deleted_count?: number;
      }>(`/api-admin/resource-repositories/${row.id}/sync`, "POST", {
        force: true,
        prune_missing: true,
      });
      const deletedText = result.deleted_count
        ? `，清理 ${result.deleted_count} 个旧模板`
        : "";
      message.success(result.message ?? `同步完成${deletedText}`);
      await load();
      await onSynced?.();
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "同步资源仓库失败",
      );
    } finally {
      setSyncingId(null);
    }
  }

  async function remove(row: RowData) {
    await adminApi.remove(`/api-admin/resource-repositories/${row.id}`);
    message.success("资源仓库已删除");
    await load();
  }

  return (
    <>
      <Modal
        title="资源仓库"
        open={open}
        onCancel={onClose}
        footer={null}
        width={1100}
        destroyOnClose
      >
        <div className="resource-toolbar" style={{ marginBottom: 16 }}>
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建仓库
          </Button>
        </div>
        <Table
          rowKey={(row) => String(row.id)}
          loading={loading}
          dataSource={rows}
          scroll={{ x: 900 }}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "名称", dataIndex: "name", width: 140 },
            {
              title: "类型",
              dataIndex: "repo_type",
              width: 80,
              render: (value) => (
                <Tag
                  color={
                    value === "ctf"
                      ? "blue"
                      : value === "vul"
                        ? "volcano"
                        : undefined
                  }
                  className="template-meta-tag"
                >
                  {value === "ctf"
                    ? "CTF"
                    : value === "vul"
                      ? "漏洞"
                      : String(value ?? "-")}
                </Tag>
              ),
            },
            {
              title: "来源",
              dataIndex: "provider",
              width: 70,
              render: (value) => String(value ?? "-"),
            },
            {
              title: "仓库地址",
              dataIndex: "repository_url",
              ellipsis: true,
              minWidth: 160,
            },
            {
              title: "分支",
              dataIndex: "repository_branch",
              width: 90,
              render: (value) => value || "-",
            },
            {
              title: "模板目录",
              dataIndex: "templates_path",
              width: 100,
              render: (value) => value || "/",
            },
            {
              title: "状态",
              dataIndex: "enabled",
              width: 80,
              render: (value) => (
                <Tag color={value ? "green" : "default"}>
                  {value ? "启用" : "停用"}
                </Tag>
              ),
            },
            {
              title: "上次同步",
              dataIndex: "last_run_at",
              width: 110,
              render: (value) => renderValue(value),
            },
            {
              title: "操作",
              key: "actions",
              width: 150,
              render: (_, row) => (
                <Space size={4}>
                  <Button
                    size="small"
                    icon={<EditOutlined />}
                    aria-label="编辑资源仓库"
                    title="编辑资源仓库"
                    onClick={() => openEdit(row)}
                  />
                  <Button
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => setDetail(row)}
                  />
                  <Button
                    size="small"
                    loading={syncingId === String(row.id)}
                    icon={<SyncOutlined />}
                    aria-label="重同步资源仓库"
                    title="重同步资源仓库"
                    onClick={() => void sync(row)}
                  />
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => void remove(row)}
                  />
                </Space>
              ),
            },
          ]}
        />
      </Modal>

      <Modal
        title={editing?.id ? "编辑资源仓库" : "新建资源仓库"}
        open={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={() => void save()}
        width={720}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="repo_type" label="类型" rules={[{ required: true }]}>
            <Select
              options={[
                { label: "CTF", value: "ctf" },
                { label: "漏洞", value: "vul" },
              ]}
            />
          </Form.Item>
          <Form.Item name="provider" label="来源" rules={[{ required: true }]}>
            <Select options={[{ label: "Git", value: "git" }]} />
          </Form.Item>
          <Form.Item
            name="repository_url"
            label="仓库地址"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="repository_branch"
            label="分支"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="templates_path" label="模板目录">
            <Input placeholder="留空表示仓库根目录" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <DetailModal
        title="资源仓库详情"
        record={detail}
        onClose={() => setDetail(null)}
      />
    </>
  );
}

export function SummaryCard({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: unknown;
  tone?: "default" | "success" | "muted" | "danger";
}) {
  return (
    <div className={`virtualization-summary virtualization-summary-${tone}`}>
      <strong>{String(value ?? "-")}</strong>
      <span>{label}</span>
    </div>
  );
}
