import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  ImportOutlined,
  PlusOutlined,
  ReloadOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  Upload,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { UploadFile } from "antd/es/upload/interface";
import { useEffect, useMemo, useState } from "react";
import { AdminDataTableCard } from "../components/AdminDataTableCard";
import {
  AdminChallengeFormModal,
  type ChallengeFile,
} from "../components/AdminChallengeFormModal";
import { adminApi } from "@shared/api/admin";
import { getErrorMessage } from "@shared/api/http";

type ChallengeRow = {
  id: number;
  name: string;
  description?: string;
  files?: ChallengeFile[];
  category: string;
  points: number;
  difficulty: number;
  usage?: string;
  competition_id?: number | null;
  flag?: string;
  level_one_solution?: string;
  is_dynamic: boolean;
  is_active?: boolean;
  is_public?: boolean;
  host_id?: number | null;
  template_id?: number | null;
  container_timeout?: number | null;
  created_at?: string;
  template?: { id: number; name: string };
  host?: { id: number; name: string; host_type?: string };
};

type ChallengeFormValues = {
  name: string;
  description?: string;
  category: string;
  points: number;
  difficulty: number;
  flag?: string;
  level_one_solution?: string;
  is_dynamic: boolean;
  is_active?: boolean;
  is_public?: boolean;
  host_id?: number;
  template_id?: number;
  container_timeout?: number;
};

const categoryOptions = [
  { label: "Web", value: "web" },
  { label: "密码学", value: "crypto" },
  { label: "逆向", value: "reverse" },
  { label: "Pwn", value: "pwn" },
  { label: "杂项", value: "misc" },
  { label: "取证", value: "forensics" },
  { label: "移动安全", value: "mobile" },
  { label: "区块链", value: "blockchain" },
  { label: "其他", value: "other" },
];

const difficultyOptions = [
  { label: "简单", value: 1 },
  { label: "较简单", value: 2 },
  { label: "中等", value: 3 },
  { label: "较难", value: 4 },
  { label: "困难", value: 5 },
];

const timeoutOptions = [
  { label: "1 小时", value: 3600 },
  { label: "2 小时", value: 7200 },
  { label: "4 小时", value: 14400 },
  { label: "8 小时", value: 28800 },
  { label: "24 小时", value: 86400 },
];

export function AdminChallengePage() {
  const { message } = App.useApp();
  const [rows, setRows] = useState<ChallengeRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<ChallengeRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingIds, setDeletingIds] = useState<number[]>([]);
  const [batchDeleting, setBatchDeleting] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [existingFiles, setExistingFiles] = useState<ChallengeFile[]>([]);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [form] = Form.useForm<ChallengeFormValues>();
  const isDynamic = Form.useWatch("is_dynamic", form) ?? false;

  useEffect(() => {
    void load();
  }, [current, pageSize]);

  const columns = useMemo<ColumnsType<ChallengeRow>>(
    () => [
      { title: "名称", dataIndex: "name", width: 220, ellipsis: true },
      {
        title: "类别",
        dataIndex: "category",
        width: 110,
        render: (value) => <Tag>{categoryLabel(String(value))}</Tag>,
      },
      {
        title: "环境",
        dataIndex: "is_dynamic",
        width: 220,
        render: (_, record) =>
          record.is_dynamic ? (
            <Space size={[6, 6]} wrap>
              <Tag color="orange">动态</Tag>
              {record.host?.name ? (
                <Tag color="blue">{record.host.name}</Tag>
              ) : null}
              {record.template?.name ? (
                <Tag color="cyan">{record.template.name}</Tag>
              ) : null}
            </Space>
          ) : (
            <Tag color="default">静态</Tag>
          ),
      },
      { title: "分值", dataIndex: "points", width: 90 },
      {
        title: "难度",
        dataIndex: "difficulty",
        width: 100,
        render: (value) => (
          <Tag color={difficultyColor(Number(value))}>
            {difficultyLabel(Number(value))}
          </Tag>
        ),
      },
      {
        title: "启用",
        dataIndex: "is_active",
        width: 90,
        render: (value) => (
          <Tag color={value ? "success" : "default"}>
            {value ? "启用" : "禁用"}
          </Tag>
        ),
      },
      {
        title: "公开",
        dataIndex: "is_public",
        width: 90,
        render: (value) => (
          <Tag color={value ? "processing" : "default"}>
            {value ? "公开" : "私有"}
          </Tag>
        ),
      },
      {
        title: "创建时间",
        dataIndex: "created_at",
        width: 180,
        render: (value) => formatTime(value),
      },
      {
        title: "操作",
        key: "actions",
        width: 180,
        render: (_, record) => (
          <Space size={4} wrap className="admin-action-group">
            <Button
              type="text"
              className="admin-action-button"
              icon={<EditOutlined />}
              size="small"
              onClick={() => openEdit(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title={`删除题目 ${record.name}`}
              description="删除后不可恢复。"
              okText="删除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              onConfirm={() => remove(record)}
            >
              <Button
                type="text"
                className="admin-action-button admin-action-button-danger"
                icon={<DeleteOutlined />}
                size="small"
                danger
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deletingIds],
  );

  async function load(nextFilters = filters, nextCurrent = current) {
    setLoading(true);
    try {
      const result = await adminApi.list<ChallengeRow>(
        "/api-admin/challenges",
        {
          current: nextCurrent,
          pageSize,
          competition_id: "null",
          ...normalizeFilters(nextFilters),
        },
      );
      const normalized = normalizePageResult<ChallengeRow>(result);
      setRows(normalized.rows);
      setTotal(normalized.total);
    } catch (error) {
      message.error(getErrorMessage(error, "加载题目失败"));
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing(null);
    setModalOpen(true);
    setFileList([]);
    setExistingFiles([]);
    form.resetFields();
    form.setFieldsValue({
      category: "web",
      points: 100,
      difficulty: 1,
      is_dynamic: false,
      is_active: true,
      is_public: true,
      container_timeout: 3600,
    });
  }

  function openEdit(record: ChallengeRow) {
    setEditing(record);
    setModalOpen(true);
    setFileList([]);
    setExistingFiles(record.files ?? []);
    form.resetFields();
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      category: record.category,
      points: record.points,
      difficulty: record.difficulty,
      flag: record.flag,
      level_one_solution: record.level_one_solution,
      is_dynamic: record.is_dynamic,
      is_active: record.is_active ?? true,
      is_public: record.is_public ?? false,
      host_id: record.host_id ?? undefined,
      template_id: record.template_id ?? undefined,
      container_timeout: record.container_timeout ?? 3600,
    });
  }

  function closeModal() {
    setModalOpen(false);
    setEditing(null);
    setFileList([]);
    setExistingFiles([]);
    form.resetFields();
  }

  async function removeExistingFile(file: ChallengeFile) {
    if (!editing?.id || file.id == null) {
      return;
    }
    try {
      await adminApi.remove(`/api-admin/challenges/${editing.id}/files/${file.id}`);
      setExistingFiles((currentFiles) =>
        currentFiles.filter((item) => String(item.id) !== String(file.id)),
      );
      message.success("附件已删除");
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "删除附件失败"));
    }
  }

  async function submit() {
    try {
      const values = await form.validateFields();
      const payload = buildChallengePayload(values);
      setSaving(true);
      const response = editing
        ? await adminApi.update<ChallengeRow>(
            `/api-admin/challenges/${editing.id}`,
            payload,
          )
        : await adminApi.create<ChallengeRow>("/api-admin/challenges", payload);

      const challengeId = editing?.id ?? response?.id;
      if (challengeId && fileList.length > 0) {
        for (const file of fileList) {
          const uploadFile = file.originFileObj ?? (file as unknown as File);
          if (!(uploadFile instanceof File)) {
            continue;
          }
          const formData = new FormData();
          formData.append("file", uploadFile);
          await adminApi.create(
            `/api-admin/challenges/${challengeId}/files`,
            formData,
          );
        }
      }

      message.success(editing ? "题目已更新" : "题目已创建");
      closeModal();
      await load();
    } catch (error) {
      message.error(
        getErrorMessage(error, editing ? "更新题目失败" : "创建题目失败"),
      );
    } finally {
      setSaving(false);
    }
  }

  async function remove(record: ChallengeRow) {
    setDeletingIds((currentIds) => [...currentIds, record.id]);
    try {
      await adminApi.remove(`/api-admin/challenges/${record.id}`);
      message.success("删除成功");
      setSelectedRowKeys((currentKeys) =>
        currentKeys.filter((key) => Number(key) !== record.id),
      );
      if (rows.length === 1 && current > 1) {
        setCurrent((page) => page - 1);
        return;
      }
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "删除失败"));
    } finally {
      setDeletingIds((currentIds) =>
        currentIds.filter((id) => id !== record.id),
      );
    }
  }

  async function batchRemove() {
    if (!selectedRowKeys.length) {
      return;
    }
    setBatchDeleting(true);
    try {
      await Promise.all(
        selectedRowKeys.map((id) =>
          adminApi.remove(`/api-admin/challenges/${id}`),
        ),
      );
      message.success("批量删除成功");
      const removedCount = selectedRowKeys.length;
      setSelectedRowKeys([]);
      if (rows.length === removedCount && current > 1) {
        setCurrent((page) => page - 1);
        return;
      }
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "批量删除失败"));
    } finally {
      setBatchDeleting(false);
    }
  }

  function handleExport() {
    window.open("/api-admin/challenges/export?format=json", "_blank");
  }

  async function handleImport() {
    if (!importFile) {
      message.error("请选择导入文件");
      return;
    }
    const formData = new FormData();
    formData.append("file", importFile);
    try {
      setImporting(true);
      await adminApi.create("/api-admin/challenges/import", formData);
      message.success("导入成功");
      setImportModalOpen(false);
      setImportFile(null);
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "导入失败"));
    } finally {
      setImporting(false);
    }
  }

  return (
    <div className="admin-page">

      <AdminDataTableCard
        title="题目列表"
        className="challenge-table-card"
        search={
          <div className="admin-search-form challenge-search-form">
            <div className="admin-search-field">
              <label>名称：</label>
              <Input
                allowClear
                placeholder="请输入题目名称"
                value={filters.name ?? ""}
                onChange={(event) =>
                  setFilters((prev) => ({ ...prev, name: event.target.value }))
                }
              />
            </div>
            <div className="admin-search-field">
              <label>分类：</label>
              <Select
                allowClear
                placeholder="请选择分类"
                options={categoryOptions}
                value={filters.category || undefined}
                onChange={(value) =>
                  setFilters((prev) => ({
                    ...prev,
                    category: value ? String(value) : "",
                  }))
                }
              />
            </div>
            <div className="admin-search-field">
              <label>难度：</label>
              <Select
                allowClear
                placeholder="请选择难度"
                options={difficultyOptions.map((item) => ({
                  label: item.label,
                  value: String(item.value),
                }))}
                value={filters.difficulty || undefined}
                onChange={(value) =>
                  setFilters((prev) => ({
                    ...prev,
                    difficulty: value ? String(value) : "",
                  }))
                }
              />
            </div>
            <Space wrap>
              <Button
                onClick={() => {
                  const nextFilters = {};
                  setFilters(nextFilters);
                  setCurrent(1);
                  void load(nextFilters, 1);
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
            <Button icon={<DownloadOutlined />} onClick={handleExport}>
              导出
            </Button>
            <Button
              icon={<ImportOutlined />}
              onClick={() => setImportModalOpen(true)}
            >
              导入
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
            {selectedRowKeys.length ? (
              <Popconfirm
                title={`删除选中的 ${selectedRowKeys.length} 个题目`}
                description="删除后不可恢复。"
                okText="批量删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={() => batchRemove()}
              >
                <Button danger icon={<DeleteOutlined />}>
                  批量删除
                </Button>
              </Popconfirm>
            ) : null}
          </>
        }
      >
        <Table
          rowKey="id"
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: true }}
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

      <AdminChallengeFormModal
        open={modalOpen}
        title={editing ? "编辑 CTF 题目" : "新建 CTF 题目"}
        saving={saving}
        form={form}
        isDynamic={isDynamic}
        fileList={fileList}
        existingFiles={existingFiles}
        onFileListChange={setFileList}
        onRemoveExistingFile={removeExistingFile}
        onCancel={closeModal}
        onOk={() => void submit()}
      />

      <Modal
        title="导入题目"
        open={importModalOpen}
        onCancel={() => {
          setImportModalOpen(false);
          setImportFile(null);
        }}
        onOk={() => void handleImport()}
        confirmLoading={importing}
      >
        <Upload
          accept=".json,.yaml,.yml"
          maxCount={1}
          beforeUpload={(file) => {
            setImportFile(file);
            return false;
          }}
          onRemove={() => setImportFile(null)}
        >
          <Button icon={<UploadOutlined />}>选择文件</Button>
        </Upload>
        <Typography.Paragraph
          type="secondary"
          style={{ marginTop: 16, marginBottom: 0 }}
        >
          支持 JSON 和 YAML 格式的题目文件。
        </Typography.Paragraph>
      </Modal>
    </div>
  );
}

function buildChallengePayload(values: ChallengeFormValues) {
  const payload: Record<string, unknown> = {
    name: values.name,
    description: values.description,
    level_one_solution: values.level_one_solution,
    category: values.category,
    points: values.points,
    difficulty: values.difficulty,
    is_dynamic: values.is_dynamic ?? false,
    is_active: values.is_active ?? true,
    is_public: values.is_public ?? false,
  };

  if (values.is_dynamic) {
    payload.host_id = values.host_id;
    payload.template_id = values.template_id;
    payload.container_timeout = values.container_timeout;
    payload.flag = undefined;
  } else {
    payload.flag = values.flag;
    payload.host_id = undefined;
    payload.template_id = undefined;
    payload.container_timeout = undefined;
  }

  return payload;
}

function normalizePageResult<T>(result: unknown) {
  if (Array.isArray(result)) {
    return { rows: result as T[], total: result.length };
  }
  const object = (result ?? {}) as {
    data?: T[];
    items?: T[];
    list?: T[];
    total?: number;
  };
  const rows = object.data ?? object.items ?? object.list ?? [];
  return { rows, total: object.total ?? rows.length };
}

function normalizeFilters(filters: Record<string, string>) {
  return Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value),
  );
}

function categoryLabel(value: string) {
  return categoryOptions.find((item) => item.value === value)?.label ?? value;
}

function difficultyLabel(value: number) {
  return difficultyOptions.find((item) => item.value === value)?.label ?? value;
}

function difficultyColor(value: number) {
  if (value <= 1) return "success";
  if (value <= 2) return "cyan";
  if (value <= 3) return "processing";
  if (value <= 4) return "warning";
  return "error";
}

function formatTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}
