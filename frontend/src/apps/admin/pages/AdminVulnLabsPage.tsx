import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { App, Button, Form, Input, Modal, Select, Space, Switch, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";
import { AdminTemplateSelect } from "../components/AdminTemplateSelect";
import { formatTime, normalizeFilters, normalizeRows } from "./adminPageUtils";

type VulnLabRow = {
  id: number;
  name: string;
  cve_id?: string;
  severity?: string;
  summary?: string;
  template_id?: number;
  template_name?: string;
  is_public?: boolean;
  is_active?: boolean;
  created_at?: string;
};

type VulnLabFormValues = {
  name: string;
  cve_id?: string;
  severity?: string;
  summary?: string;
  template_id?: number;
  is_public?: boolean;
  is_active?: boolean;
};

type TemplateOption = {
  id: number | string;
  name: string;
  host_name?: string;
};

const severityColorMap: Record<string, string> = {
  CRITICAL: "red",
  HIGH: "orange",
  MEDIUM: "gold",
  LOW: "green",
  UNKNOWN: "default",
};

const severityOptions = [
  { label: "严重", value: "CRITICAL" },
  { label: "高危", value: "HIGH" },
  { label: "中危", value: "MEDIUM" },
  { label: "低危", value: "LOW" },
  { label: "未知", value: "UNKNOWN" },
];

export function AdminVulnLabsPage() {
  const { message } = App.useApp();
  const [rows, setRows] = useState<VulnLabRow[]>([]);
  const [templateOptions, setTemplateOptions] = useState<TemplateOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<VulnLabRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [form] = Form.useForm<VulnLabFormValues>();

  useEffect(() => {
    void load();
    void loadTemplateOptions();
  }, []);

  async function load(nextFilters = filters) {
    setLoading(true);
    try {
      const result = await adminApi.list<VulnLabRow>("/api-admin/vuln-labs", {
        current: 1,
        pageSize: 200,
        ...normalizeFilters(nextFilters),
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载漏洞复现列表失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadTemplateOptions() {
    try {
      const result = await adminApi.list<TemplateOption>("/api-admin/virtualization/templates/options", {
        current: 1,
        pageSize: 500,
      });
      setTemplateOptions(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载模板列表失败");
    }
  }

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      severity: "MEDIUM",
      is_public: true,
      is_active: true,
    });
    setModalOpen(true);
  }

  function openEdit(record: VulnLabRow) {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    try {
      setSaving(true);
      if (editing) {
        await adminApi.update(`/api-admin/vuln-labs/${editing.id}`, values);
        message.success("漏洞复现已更新");
      } else {
        await adminApi.create("/api-admin/vuln-labs", values);
        message.success("漏洞复现已创建");
      }
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存漏洞复现失败");
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: VulnLabRow) {
    try {
      setDeletingId(record.id);
      await adminApi.remove(`/api-admin/vuln-labs/${record.id}`);
      message.success("漏洞复现已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除漏洞复现失败");
    } finally {
      setDeletingId(null);
    }
  }

  const columns: ColumnsType<VulnLabRow> = [
    { title: "标题", dataIndex: "name", width: 200 },
    {
      title: "CVE",
      dataIndex: "cve_id",
      width: 200,
    },
    {
      title: "等级",
      dataIndex: "severity",
      width: 120,
      render: (value) => {
        const key = String(value ?? "UNKNOWN");
        return <Tag color={severityColorMap[key] ?? "default"}>{severityLabel(key)}</Tag>;
      },
    },
    {
      title: "模板名称",
      dataIndex: "template_name",
      width: 220,
      render: (_, record) => {
        const template = templateOptions.find((option) => Number(option.id) === record.template_id);
        if (template) {
          return formatTemplateLabel(template);
        }
        return record.template_name ?? "-";
      },
    },
    {
      title: "公开",
      dataIndex: "is_public",
      width: 90,
      render: (value) => <Tag color={value ? "green" : "default"}>{value ? "公开" : "私有"}</Tag>,
    },
    {
      title: "启用",
      dataIndex: "is_active",
      width: 90,
      render: (value) => <Tag color={value ? "green" : "default"}>{value ? "启用" : "禁用"}</Tag>,
    },
    {
      title: "摘要",
      dataIndex: "summary",
      ellipsis: true,
      width: 200,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 140,
      render: formatTime,
    },
    {
      title: "操作",
      key: "actions",
      width: 150,
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
      <Typography.Title level={4} className="admin-page-title">
        漏洞复现
      </Typography.Title>

      <AdminDataTableCard
        title="漏洞复现列表"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>关键词：</label>
              <Input
                allowClear
                value={filters.keyword ?? ""}
                onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
                onPressEnter={() => void load()}
              />
            </div>
            <div className="admin-search-field">
              <label>等级：</label>
              <Select
                allowClear
                options={severityOptions}
                value={filters.severity || undefined}
                onChange={(value) => setFilters((current) => ({ ...current, severity: value ? String(value) : "" }))}
              />
            </div>
            <Space>
              <Button
                onClick={() => {
                  setFilters({});
                  void load({});
                }}
              >
                重置
              </Button>
              <Button type="primary" onClick={() => void load()}>
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
        <Table rowKey="id" loading={loading} columns={columns} dataSource={rows} pagination={{ pageSize: 10 }} />
      </AdminDataTableCard>

      <Modal title={editing ? "编辑漏洞复现" : "新建漏洞复现"} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={760} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="标题" rules={[{ required: true, message: "请输入标题" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="cve_id" label="CVE 编号">
            <Input />
          </Form.Item>
          <Form.Item name="severity" label="等级">
            <Select options={severityOptions} />
          </Form.Item>
          <Form.Item name="template_id" label="模板名称" rules={[{ required: true, message: "请选择模板" }]}>
            <AdminTemplateSelect
              onChange={(templateId) => {
                form.setFieldValue("template_id", templateId);
              }}
            />
          </Form.Item>
          <Form.Item name="summary" label="摘要">
            <Input.TextArea rows={5} />
          </Form.Item>
          <Form.Item name="is_public" label="公开" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="is_active" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function severityLabel(value: string) {
  return severityOptions.find((item) => item.value === value)?.label ?? value;
}

function formatTemplateLabel(template: TemplateOption) {
  if (template.host_name && template.name) {
    return `${template.host_name} / ${template.name}`;
  }
  return template.host_name || template.name || String(template.id);
}
