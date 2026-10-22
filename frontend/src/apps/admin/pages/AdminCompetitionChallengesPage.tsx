import {
  ArrowLeftOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Form,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { UploadFile } from "antd/es/upload/interface";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import {
  AdminChallengeFormModal,
  type ChallengeFile,
} from "../components/AdminChallengeFormModal";

type ChallengeRow = {
  id: number;
  name: string;
  category?: string;
  points?: number;
  difficulty?: number;
  is_dynamic?: boolean;
  is_active?: boolean;
  is_public?: boolean;
  description?: string;
  flag?: string;
  level_one_solution?: string;
  usage?: string;
  host_id?: number;
  template_id?: number;
  container_image?: string;
  container_port?: number;
  container_timeout?: number;
  competition_id?: number | null;
  files?: ChallengeFile[];
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

export function AdminCompetitionChallengesPage({
  embedded = false,
}: {
  embedded?: boolean;
}) {
  const navigate = useNavigate();
  const { competitionId = "" } = useParams();
  const [boundRows, setBoundRows] = useState<ChallengeRow[]>([]);
  const [practiceRows, setPracticeRows] = useState<ChallengeRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [adding, setAdding] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [creatingOpen, setCreatingOpen] = useState(false);
  const [editingRow, setEditingRow] = useState<ChallengeRow | null>(null);
  const [saving, setSaving] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [existingFiles, setExistingFiles] = useState<ChallengeFile[]>([]);
  const [pickerForm] = Form.useForm<{ challenge_id: number }>();
  const [createForm] = Form.useForm<ChallengeFormValues>();
  const createIsDynamic = Form.useWatch("is_dynamic", createForm) ?? false;

  useEffect(() => {
    void load();
  }, [competitionId]);

  async function load() {
    setLoading(true);
    try {
      const [bound, practice] = await Promise.all([
        adminApi.list<ChallengeRow>(
          `/api-admin/competitions/${competitionId}/challenges`,
        ),
        adminApi.list<ChallengeRow>("/api-admin/challenges", {
          current: 1,
          pageSize: 500,
        }),
      ]);
      setBoundRows(normalizeRows(bound));
      setPracticeRows(
        normalizeRows(practice).filter(
          (row) =>
            !row.competition_id &&
            (!row.usage || row.usage === "PRACTICE" || row.usage === "BOTH"),
        ),
      );
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载赛事题目失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function addChallenge(values: { challenge_id: number }) {
    setAdding(true);
    try {
      await adminApi.create(
        `/api-admin/competitions/${competitionId}/challenges`,
        {
          challenge_id: values.challenge_id,
        },
      );
      pickerForm.resetFields();
      setPickerOpen(false);
      message.success("题目已复制到比赛");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "复制题目失败");
    } finally {
      setAdding(false);
    }
  }

  function openCreate() {
    setEditingRow(null);
    setFileList([]);
    setExistingFiles([]);
    createForm.resetFields();
    createForm.setFieldsValue({
      category: "web",
      points: 100,
      difficulty: 1,
      is_dynamic: false,
      is_active: true,
      is_public: false,
      container_timeout: 3600,
    });
    setCreatingOpen(true);
  }

  function openEdit(record: ChallengeRow) {
    setEditingRow(record);
    setFileList([]);
    setExistingFiles(record.files ?? []);
    createForm.resetFields();
    createForm.setFieldsValue({
      name: record.name,
      description: record.description,
      category: record.category ?? "web",
      points: record.points ?? 100,
      difficulty: record.difficulty ?? 1,
      flag: record.flag,
      level_one_solution: record.level_one_solution,
      is_dynamic: record.is_dynamic ?? false,
      is_active: record.is_active ?? true,
      is_public: record.is_public ?? false,
      host_id: record.host_id,
      template_id: record.template_id,
      container_timeout: record.container_timeout ?? 3600,
    });
    setCreatingOpen(true);
  }

  async function createChallenge() {
    try {
      const values = await createForm.validateFields();
      setSaving(true);
      const payload = {
        name: values.name,
        description: values.description,
        level_one_solution: values.level_one_solution,
        category: values.category,
        points: values.points,
        difficulty: values.difficulty,
        flag: values.is_dynamic ? undefined : values.flag,
        is_dynamic: values.is_dynamic ?? false,
        is_active: values.is_active ?? true,
        is_public: values.is_public ?? false,
        host_id: values.is_dynamic ? values.host_id : undefined,
        template_id: values.is_dynamic ? values.template_id : undefined,
        container_timeout: values.is_dynamic
          ? values.container_timeout
          : undefined,
        competition_id: Number(competitionId),
        usage: "COMPETITION",
      };
      if (editingRow) {
        await adminApi.update(
          `/api-admin/challenges/${editingRow.id}`,
          payload,
        );
      } else {
        await adminApi.create("/api-admin/challenges", payload);
      }

      const challengeId = editingRow?.id;
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

      message.success(editingRow ? "比赛题目已更新" : "比赛题目已创建");
      setCreatingOpen(false);
      setEditingRow(null);
      setFileList([]);
      setExistingFiles([]);
      createForm.resetFields();
      await load();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  }

  async function removeChallenge(record: ChallengeRow) {
    try {
      await adminApi.remove(
        `/api-admin/competitions/${competitionId}/challenges/${record.id}`,
      );
      message.success("题目已移出比赛");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "移出题目失败");
    }
  }

  async function removeExistingFile(file: ChallengeFile) {
    if (!editingRow?.id || file.id == null) {
      return;
    }
    try {
      await adminApi.remove(
        `/api-admin/challenges/${editingRow.id}/files/${file.id}`,
      );
      setExistingFiles((currentFiles) =>
        currentFiles.filter((item) => String(item.id) !== String(file.id)),
      );
      message.success("附件已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除附件失败");
    }
  }

  const columns: ColumnsType<ChallengeRow> = [
    { title: "ID", dataIndex: "id", width: 80 },
    { title: "题目名称", dataIndex: "name" },
    { title: "分类", dataIndex: "category", width: 120 },
    { title: "分值", dataIndex: "points", width: 100 },
    { title: "难度", dataIndex: "difficulty", width: 100 },
    {
      title: "环境",
      dataIndex: "is_dynamic",
      width: 110,
      render: (value) => (
        <Tag color={value ? "blue" : "default"}>{value ? "动态" : "静态"}</Tag>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 180,
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            编辑
          </Button>
          <Button danger size="small" onClick={() => removeChallenge(record)}>
            移出
          </Button>
        </Space>
      ),
    },
  ];

  const content = (
    <>
      {!embedded ? (
        <div className="admin-page-heading">
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate("/admin/competition")}
          >
            返回赛事列表
          </Button>
        </div>
      ) : null}

      <Card className="admin-search-card" bordered={false}>
        <Space wrap>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setPickerOpen(true)}
          >
            从练习场选择题目
          </Button>
          <Button onClick={openCreate}>新建比赛题目</Button>
          <Button icon={<ReloadOutlined />} onClick={load}>
            刷新
          </Button>
        </Space>
      </Card>

      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>比赛题目</Typography.Title>
          <Typography.Text type="secondary">
            当前比赛仅显示已复制到本场比赛的题目
          </Typography.Text>
        </div>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={boundRows}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title="从练习场选择题目"
        open={pickerOpen}
        onCancel={() => {
          setPickerOpen(false);
          pickerForm.resetFields();
        }}
        onOk={() => void pickerForm.submit()}
        confirmLoading={adding}
        destroyOnClose
      >
        <Form form={pickerForm} layout="vertical" onFinish={addChallenge}>
          <Form.Item
            name="challenge_id"
            label="练习场题目"
            rules={[{ required: true, message: "请选择题目" }]}
          >
            <Select
              showSearch
              placeholder="请选择要复制到比赛的练习场题目"
              optionFilterProp="label"
              options={practiceRows.map((row) => ({
                value: row.id,
                label: `${row.name} (#${row.id}) · ${row.category ?? "-"} · ${row.points ?? 0}分`,
              }))}
            />
          </Form.Item>
          <Typography.Text type="secondary">
            选择后会深度复制一份到当前比赛，后续修改互不影响。
          </Typography.Text>
        </Form>
      </Modal>

      <AdminChallengeFormModal
        open={creatingOpen}
        title={editingRow ? "编辑比赛题目" : "新建比赛题目"}
        saving={saving}
        width={760}
        form={createForm}
        isDynamic={createIsDynamic}
        fileList={fileList}
        existingFiles={existingFiles}
        onFileListChange={setFileList}
        onRemoveExistingFile={removeExistingFile}
        onCancel={() => {
          setCreatingOpen(false);
          setEditingRow(null);
          setFileList([]);
          setExistingFiles([]);
          createForm.resetFields();
        }}
        onOk={() => void createChallenge()}
      />
    </>
  );

  if (embedded) {
    return <div className="admin-page">{content}</div>;
  }

  return (
    <div className="admin-page">
      {content}
    </div>
  );
}

function normalizeRows(result: unknown): ChallengeRow[] {
  if (Array.isArray(result)) {
    return result as ChallengeRow[];
  }
  const object = (result ?? {}) as {
    data?: ChallengeRow[];
    items?: ChallengeRow[];
    list?: ChallengeRow[];
  };
  return object.data ?? object.items ?? object.list ?? [];
}
