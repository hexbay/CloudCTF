import {
  ClearOutlined,
  CloudDownloadOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  DeploymentUnitOutlined,
  DownloadOutlined,
  FileTextOutlined,
  HddOutlined,
  InfoCircleOutlined,
  ReloadOutlined,
  RocketOutlined,
  RollbackOutlined,
  SaveOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Progress,
  Radio,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { ReactNode } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import { publicApi } from "@shared/api/public";
import { formatTime, normalizeRows } from "./adminPageUtils";

type SettingsForm = {
  site_name: string;
  site_title?: string;
  site_logo?: string;
  site_description?: string;
  allow_registration?: boolean;
  email_verification?: boolean;
  max_team_size?: number;
  ctf_ttl_seconds?: number;
  vuln_ttl_seconds?: number;
  enable_writeup?: boolean;
  enable_notifications?: boolean;
  show_github_link?: boolean;
};

type SystemInfo = {
  cpu_percent?: number;
  memory_percent?: number;
  disk_percent?: number;
  db_size?: string;
  uptime?: string;
  hostname?: string;
  os?: string;
  arch?: string;
  last_updated?: string;
};

type UpdateInfo = {
  current_version: string;
  latest_version?: string;
  update_available: boolean;
  supported: boolean;
  reason?: string;
  release_name?: string;
  release_notes?: string;
  release_url?: string;
  published_at?: string;
  asset_name?: string;
  asset_size?: number;
  repository: string;
};

type AdminIdentity = {
  is_superuser?: boolean;
};

type BackupRow = {
  id: number | string;
  name?: string;
  created_at?: string;
  updated_at?: string;
};

type LogRow = {
  id: number;
  level?: string;
  message?: string;
  action_type?: string;
  resource_type?: string;
  created_at?: string;
};

type DemoScale = "2-10" | "10-100" | "100+";

const settingsTabKeys = ["general", "info", "backup", "maintenance", "logs"] as const;

export function AdminSettingsPage() {
  const navigate = useNavigate();
  const { tabKey } = useParams<{ tabKey?: string }>();
  const [modal, contextHolder] = Modal.useModal();
  const [form] = Form.useForm<SettingsForm>();
  const loadedRef = useRef(false);
  const updateCheckStartedRef = useRef(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [systemInfoLoading, setSystemInfoLoading] = useState(false);
  const [backups, setBackups] = useState<BackupRow[]>([]);
  const [backupLoading, setBackupLoading] = useState(false);
  const [logs, setLogs] = useState<LogRow[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [demoDataLoading, setDemoDataLoading] = useState(false);
  const [demoScale, setDemoScale] = useState<DemoScale>("10-100");
  const [systemVersion, setSystemVersion] = useState("dev");
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updateChecking, setUpdateChecking] = useState(false);
  const [updateApplying, setUpdateApplying] = useState(false);
  const [canInstallUpdates, setCanInstallUpdates] = useState(false);

  const activeTabKey = settingsTabKeys.includes((tabKey || "") as (typeof settingsTabKeys)[number])
    ? (tabKey as (typeof settingsTabKeys)[number])
    : "general";

  useEffect(() => {
    void load();
    void loadBackups();
    void loadLogs();
    void loadAdminIdentity();
  }, []);

  useEffect(() => {
    if (activeTabKey !== "info") {
      return;
    }
    void refreshSystemInfo();
    if (!updateCheckStartedRef.current) {
      updateCheckStartedRef.current = true;
      void checkForUpdates(false);
    }
    const timer = window.setInterval(() => {
      void refreshSystemInfo();
    }, 15000);
    return () => window.clearInterval(timer);
  }, [activeTabKey]);

  async function load() {
    setLoading(true);
    try {
      const settings = await adminApi.list<SettingsForm>("/api-admin/settings/settings");
      if (!loadedRef.current) {
        form.setFieldsValue(settings as SettingsForm);
        loadedRef.current = true;
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载系统设置失败");
    } finally {
      setLoading(false);
    }
  }

  async function submit() {
    try {
      const values = await form.validateFields();
      setSaving(true);
      await adminApi.update("/api-admin/settings/settings", values);
      window.dispatchEvent(new CustomEvent("system-settings-updated", { detail: values }));
      message.success("系统设置已保存");
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  }

  async function loadSystemInfo() {
    setSystemInfoLoading(true);
    try {
      const result = await adminApi.get<SystemInfo>("/api-admin/settings/settings/info");
      setSystemInfo(result);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载系统信息失败");
    } finally {
      setSystemInfoLoading(false);
    }
  }

  async function loadVersion() {
    try {
      const result = await publicApi.version();
      setSystemVersion(result.version?.trim() || "dev");
    } catch {
      setSystemVersion("dev");
    }
  }

  async function refreshSystemInfo() {
    await Promise.allSettled([loadSystemInfo(), loadVersion()]);
  }

  async function checkForUpdates(announce = true) {
    setUpdateChecking(true);
    try {
      const result = await adminApi.get<UpdateInfo>("/api-admin/settings/update/check");
      setUpdateInfo(result);
      if (!announce) {
        return;
      }
      if (!result.supported) {
        message.warning(result.reason || "当前环境不支持在线更新");
      } else if (result.update_available) {
        message.success(`发现新版本 ${result.latest_version}`);
      } else {
        message.success("当前已是最新版本");
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "检查更新失败");
    } finally {
      setUpdateChecking(false);
    }
  }

  async function loadAdminIdentity() {
    try {
      const identity = await adminApi.me() as AdminIdentity;
      setCanInstallUpdates(Boolean(identity.is_superuser));
    } catch {
      setCanInstallUpdates(false);
    }
  }

  function applyUpdate() {
    if (!updateInfo?.latest_version) {
      return;
    }
    const targetVersion = updateInfo.latest_version;
    modal.confirm({
      title: `更新到 ${targetVersion}`,
      content: "系统会下载并校验 GitHub Release，替换程序后自动重启。更新前建议先创建数据备份，重启期间服务会短暂不可用。",
      okText: "确认更新",
      cancelText: "取消",
      okButtonProps: { danger: true },
      onOk: async () => {
        setUpdateApplying(true);
        try {
          await adminApi.create("/api-admin/settings/update/apply", { tag: targetVersion });
          message.loading({ content: `已安装 ${targetVersion}，正在等待服务重启…`, key: "system-update", duration: 0 });
          await waitForUpdatedService(targetVersion);
          message.success({ content: `已更新到 ${targetVersion}`, key: "system-update", duration: 2 });
          window.location.reload();
        } catch (error) {
          message.destroy("system-update");
          message.error(error instanceof Error ? error.message : "在线更新失败");
          setUpdateApplying(false);
        }
      },
    });
  }

  async function loadBackups() {
    setBackupLoading(true);
    try {
      const result = await adminApi.list<BackupRow>("/api-admin/settings/backups");
      setBackups(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载备份列表失败");
    } finally {
      setBackupLoading(false);
    }
  }

  async function createBackup() {
    setBackupLoading(true);
    try {
      await adminApi.create("/api-admin/settings/backup", {});
      message.success("备份已创建");
      await loadBackups();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建备份失败");
    } finally {
      setBackupLoading(false);
    }
  }

  function restoreBackup(record: BackupRow) {
    modal.confirm({
      title: `恢复备份 ${record.name || record.id}`,
      content: "恢复备份会覆盖当前系统设置，请确认已了解影响。",
      okText: "恢复",
      onOk: async () => {
        await adminApi.action(`/api-admin/settings/restore/${record.id}`, "POST");
        message.success("备份已恢复");
        await load();
      },
    });
  }

  function deleteBackup(record: BackupRow) {
    modal.confirm({
      title: `删除备份 ${record.name || record.id}`,
      content: "删除后不可恢复。",
      okText: "删除",
      okButtonProps: { danger: true },
      onOk: async () => {
        await adminApi.remove(`/api-admin/settings/backups/${record.id}`);
        message.success("备份已删除");
        await loadBackups();
      },
    });
  }

  function downloadBackup(record: BackupRow) {
    void (async () => {
      try {
        const token = localStorage.getItem("cloudctf_token");
        const response = await fetch(`/api-admin/settings/backups/${record.id}/download`, {
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
        });
        if (!response.ok) {
          const payload = (await response.json().catch(() => null)) as { detail?: string; message?: string } | null;
          throw new Error(payload?.detail || payload?.message || "下载备份失败");
        }
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = record.name || String(record.id);
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
      } catch (error) {
        message.error(error instanceof Error ? error.message : "下载备份失败");
      }
    })();
  }

  async function uploadBackup(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    setBackupLoading(true);
    try {
      await adminApi.create("/api-admin/settings/backups/upload", formData);
      message.success("备份已上传");
      await loadBackups();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "上传备份失败");
    } finally {
      setBackupLoading(false);
    }
    return false;
  }

  function clearCache() {
    modal.confirm({
      title: "确认清理缓存",
      content: "清理缓存会释放临时数据，不会删除业务数据。",
      onOk: async () => {
        await adminApi.action("/api-admin/settings/cache/clear", "POST");
        message.success("缓存已清理");
      },
    });
  }

  function generateDemoData() {
    modal.confirm({
      title: "确认生成演示数据",
      content: `将按 ${demoScale} 规模生成演示用户、队伍、比赛、题目和成绩，并覆盖个人赛、团队赛、攻防赛、夺旗占点等类型。已存在的同名演示数据会自动跳过，不会清空当前业务数据。`,
      okText: "开始生成",
      onOk: async () => {
        setDemoDataLoading(true);
        try {
          const result = (await adminApi.create("/api-admin/settings/demo-data/generate", {
            scale: demoScale,
          })) as {
            message?: string;
            data?: {
              scale?: string;
              users?: number;
              teams?: number;
              competitions?: number;
              challenges?: number;
              submissions?: number;
            };
          };
          const summary = result.data;
          if (summary) {
            message.success(
              `演示数据已生成：规模 ${summary.scale || demoScale}，用户 ${summary.users || 0}，队伍 ${summary.teams || 0}，比赛 ${summary.competitions || 0}，题目 ${summary.challenges || 0}，成绩 ${summary.submissions || 0}`,
            );
          } else {
            message.success(result.message || "演示数据已生成");
          }
        } catch (error) {
          message.error(error instanceof Error ? error.message : "生成演示数据失败");
        } finally {
          setDemoDataLoading(false);
        }
      },
    });
  }

  async function loadLogs() {
    setLogsLoading(true);
    try {
      const result = await adminApi.list<LogRow>("/api-admin/settings/logs");
      setLogs(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载日志失败");
    } finally {
      setLogsLoading(false);
    }
  }

  function exportLogs() {
    const content = logs
      .map(
        (log) =>
          `[${formatTime(log.created_at)}] ${log.level || "info"} ${log.resource_type || "-"} ${log.action_type || "-"} ${log.message || ""}`,
      )
      .join("\n");
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `system_logs_${Date.now()}.log`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }

  const backupColumns: ColumnsType<BackupRow> = [
    { title: "备份 ID", dataIndex: "id", width: 200 },
    { title: "名称", dataIndex: "name", render: (value) => value || "-", width: 200 },
    { title: "创建时间", dataIndex: "created_at", width: 180, render: formatTime },
    { title: "更新时间", dataIndex: "updated_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "actions",
      width: 220,
      render: (_, row) => (
        <Space size={4} wrap className="admin-action-group">
          <Button type="text" size="small" className="admin-action-button" icon={<DownloadOutlined />} onClick={() => downloadBackup(row)}>
            下载
          </Button>
          <Button type="text" size="small" className="admin-action-button" icon={<RollbackOutlined />} onClick={() => restoreBackup(row)}>
            恢复
          </Button>
          <Button
            type="text"
            size="small"
            danger
            className="admin-action-button admin-action-button-danger"
            icon={<DeleteOutlined />}
            onClick={() => deleteBackup(row)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const logColumns: ColumnsType<LogRow> = [
    { title: "时间", dataIndex: "created_at", width: 180, render: formatTime },
    { title: "级别", dataIndex: "level", width: 90, render: (value) => <Tag>{value || "info"}</Tag> },
    { title: "资源", dataIndex: "resource_type", width: 140, render: (value) => value || "-" },
    { title: "动作", dataIndex: "action_type", width: 120, render: (value) => value || "-" },
    { title: "内容", dataIndex: "message", ellipsis: true, render: (value) => value || "-" },
  ];

  const systemHealth = useMemo(() => resolveSystemHealth(systemInfo), [systemInfo]);
  const systemFacts = useMemo(
    () => [
      { label: "系统版本", value: systemVersion },
      { label: "服务运行时长", value: systemInfo?.uptime || "-" },
      { label: "节点主机", value: systemInfo?.hostname || "-" },
      { label: "运行环境", value: formatRuntimeLabel(systemInfo) },
      { label: "最近刷新", value: formatTime(systemInfo?.last_updated) || "-" },
    ],
    [systemInfo, systemVersion],
  );

  return (
    <div className="admin-page">
      {contextHolder}
      <Typography.Title level={4} className="admin-page-title">
        系统设置
      </Typography.Title>

      <Tabs
        destroyInactiveTabPane={false}
        activeKey={activeTabKey}
        onChange={(nextKey) => navigate(`/admin/settings/${nextKey}`)}
        items={[
          {
            key: "general",
            label: (
              <span>
                <SettingOutlined /> 常规设置
              </span>
            ),
            children: (
              <Card className="admin-table-card" bordered={false} loading={loading}>
                <Form form={form} layout="vertical" className="admin-settings-form">
                  <Form.Item label="站点名称" name="site_name" rules={[{ required: true, message: "请输入站点名称" }]}>
                    <Input placeholder="CloudCTF" />
                  </Form.Item>
                  <Form.Item label="页面标题" name="site_title">
                    <Input placeholder="CloudCTF" />
                  </Form.Item>
                  <Form.Item label="站点 Logo" name="site_logo">
                    <Input placeholder="https://example.com/logo.png" />
                  </Form.Item>
                  <Form.Item label="站点描述" name="site_description">
                    <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} />
                  </Form.Item>
                  <Form.Item label="最大队伍人数" name="max_team_size">
                    <InputNumber min={1} max={99} style={{ width: "100%" }} />
                  </Form.Item>
                  <Form.Item label="CTF环境时长(秒)" name="ctf_ttl_seconds">
                    <InputNumber min={1} max={86400} style={{ width: "100%" }} />
                  </Form.Item>
                  <Form.Item label="漏洞环境时长(秒)" name="vuln_ttl_seconds">
                    <InputNumber min={1} max={86400} style={{ width: "100%" }} />
                  </Form.Item>
                  <Form.Item label="允许注册" name="allow_registration" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item label="邮箱验证" name="email_verification" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item label="启用 Writeup" name="enable_writeup" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item label="启用通知" name="enable_notifications" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item
                    label="显示 GitHub 仓库入口"
                    name="show_github_link"
                    valuePropName="checked"
                    initialValue
                    tooltip="控制选手端和管理后台顶部导航中的 GitHub 图标，默认显示"
                  >
                    <Switch />
                  </Form.Item>
                </Form>
                <Space>
                  <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={submit}>
                    保存设置
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={load}>
                    重新加载
                  </Button>
                </Space>
              </Card>
            ),
          },
          {
            key: "info",
            label: (
              <span>
                <InfoCircleOutlined /> 系统信息
              </span>
            ),
            children: (
              <Card className="admin-table-card" bordered={false} loading={systemInfoLoading}>
                <div className="settings-info-hero">
                  <div className="settings-info-hero-copy">
                    <Tag className="settings-info-kicker" bordered={false}>
                      系统运行视图
                    </Tag>
                    <Typography.Title level={4}>CloudCTF 管理节点</Typography.Title>
                    <Typography.Paragraph>
                      在这里快速查看当前版本、运行状态和实时资源占用。系统信息标签激活时会自动刷新，适合巡检和发布后确认。
                    </Typography.Paragraph>
                    <Space wrap>
                      <Button type="primary" icon={<ReloadOutlined />} loading={systemInfoLoading} onClick={() => void refreshSystemInfo()}>
                        刷新系统信息
                      </Button>
                      <Tag className="settings-version-pill" bordered={false}>
                        版本 {systemVersion}
                      </Tag>
                      <Button icon={<CloudDownloadOutlined />} loading={updateChecking} disabled={updateApplying} onClick={() => void checkForUpdates()}>
                        检查更新
                      </Button>
                    </Space>
                  </div>
                  <div className="settings-info-hero-meta">
                    <InfoSummaryCard icon={<DeploymentUnitOutlined />} label="系统版本" value={systemVersion} />
                    <InfoSummaryCard icon={<RocketOutlined />} label="服务状态" value={systemHealth} />
                    <InfoSummaryCard icon={<DatabaseOutlined />} label="数据库大小" value={systemInfo?.db_size || "-"} />
                  </div>
                </div>

                <div className="settings-runtime-grid">
                  <MetricCard title="CPU 使用率" icon={<ThunderboltOutlined />} value={systemInfo?.cpu_percent ?? 0} tone="cyan" />
                  <MetricCard title="内存使用率" icon={<DatabaseOutlined />} value={systemInfo?.memory_percent ?? 0} tone="green" />
                  <MetricCard title="磁盘使用率" icon={<HddOutlined />} value={systemInfo?.disk_percent ?? 0} tone="amber" />
                </div>

                <div className="settings-info-strip">
                  {systemFacts.map((fact) => (
                    <InfoItem key={fact.label} label={fact.label} value={fact.value} />
                  ))}
                </div>
                {updateInfo && (
                  <Card
                    size="small"
                    title={`在线更新 · ${updateInfo.repository}`}
                    style={{ marginTop: 20 }}
                    extra={updateInfo.release_url ? <Typography.Link href={updateInfo.release_url} target="_blank" rel="noreferrer">查看 GitHub Release</Typography.Link> : null}
                  >
                    {!updateInfo.supported ? (
                      <Alert type="warning" showIcon message="当前环境不支持在线更新" description={updateInfo.reason} />
                    ) : updateInfo.update_available ? (
                      <Space direction="vertical" size={12} style={{ width: "100%" }}>
                        <Alert
                          type="info"
                          showIcon
                          message={`发现新版本 ${updateInfo.latest_version}`}
                          description={`${updateInfo.asset_name || "发布文件"} · ${formatBytes(updateInfo.asset_size)} · 发布于 ${formatTime(updateInfo.published_at) || "-"}`}
                        />
                        {updateInfo.release_notes && (
                          <Typography.Paragraph ellipsis={{ rows: 6, expandable: true, symbol: "展开发布说明" }} style={{ whiteSpace: "pre-wrap", marginBottom: 0 }}>
                            {updateInfo.release_notes}
                          </Typography.Paragraph>
                        )}
                        <Button type="primary" danger icon={<CloudDownloadOutlined />} loading={updateApplying} disabled={!canInstallUpdates} onClick={applyUpdate}>
                          下载并更新到 {updateInfo.latest_version}
                        </Button>
                        {!canInstallUpdates && <Typography.Text type="secondary">只有超级管理员可以执行系统更新。</Typography.Text>}
                      </Space>
                    ) : (
                      <Alert type="success" showIcon message="当前已是最新版本" description={`当前版本：${updateInfo.current_version}`} />
                    )}
                  </Card>
                )}
              </Card>
            ),
          },
          {
            key: "backup",
            label: (
              <span>
                <DatabaseOutlined /> 备份管理
              </span>
            ),
            children: (
              <Card
                className="admin-table-card"
                bordered={false}
                title="备份列表"
                extra={
                  <Space wrap>
                    <Button type="primary" icon={<CloudDownloadOutlined />} loading={backupLoading} onClick={createBackup}>
                      创建备份
                    </Button>
                    <Upload showUploadList={false} accept=".json,.zip" beforeUpload={(file) => uploadBackup(file)}>
                      <Button icon={<UploadOutlined />}>上传备份</Button>
                    </Upload>
                    <Button icon={<ReloadOutlined />} onClick={loadBackups}>
                      刷新
                    </Button>
                  </Space>
                }
              >
                <Table rowKey="id" loading={backupLoading} columns={backupColumns} dataSource={backups} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
              </Card>
            ),
          },
          {
            key: "maintenance",
            label: (
              <span>
                <ClearOutlined /> 维护
              </span>
            ),
            children: (
              <Card className="admin-table-card" bordered={false}>
                <Space direction="vertical" size={16}>
                  <Typography.Title level={5}>缓存管理</Typography.Title>
                  <Typography.Text type="secondary">
                    清理系统缓存可以释放临时资源，适用于配置更新后需要刷新运行状态的场景。
                  </Typography.Text>
                  <Button icon={<ClearOutlined />} onClick={clearCache}>
                    清理缓存
                  </Button>
                </Space>
                <div style={{ height: 24 }} />
                <Space direction="vertical" size={16}>
                  <Typography.Title level={5}>演示数据</Typography.Title>
                  <Typography.Text type="secondary">
                    快速生成用于联调和验收的演示用户、队伍、比赛、题目与成绩数据，覆盖多种比赛类型。
                  </Typography.Text>
                  <Radio.Group value={demoScale} onChange={(event) => setDemoScale(event.target.value as DemoScale)} optionType="button" buttonStyle="solid">
                    <Radio.Button value="2-10">2-10</Radio.Button>
                    <Radio.Button value="10-100">10-100</Radio.Button>
                    <Radio.Button value="100+">100+</Radio.Button>
                  </Radio.Group>
                  <Typography.Text type="secondary">当前档位会影响用户数、队伍数、参赛人数和成绩数量。</Typography.Text>
                  <Button type="primary" icon={<DatabaseOutlined />} loading={demoDataLoading} onClick={generateDemoData}>
                    生成演示数据
                  </Button>
                </Space>
              </Card>
            ),
          },
          {
            key: "logs",
            label: (
              <span>
                <FileTextOutlined /> 日志
              </span>
            ),
            children: (
              <Card
                className="admin-table-card"
                bordered={false}
                title="系统日志"
                extra={
                  <Space wrap>
                    <Button icon={<ReloadOutlined />} loading={logsLoading} onClick={loadLogs}>
                      刷新
                    </Button>
                    <Button icon={<DownloadOutlined />} disabled={!logs.length} onClick={exportLogs}>
                      导出
                    </Button>
                  </Space>
                }
              >
                <Table rowKey="id" loading={logsLoading} columns={logColumns} dataSource={logs} scroll={{ x: true }} pagination={{ pageSize: 10 }} />
              </Card>
            ),
          },
        ]}
      />
    </div>
  );
}

function MetricCard({
  title,
  value,
  icon,
  tone,
}: {
  title: string;
  value: number;
  icon: ReactNode;
  tone: "cyan" | "green" | "amber";
}) {
  const percent = Math.max(0, Math.min(100, Math.round(Number(value) || 0)));
  const strokeColor = metricStrokeColor(tone);
  return (
    <div className={`settings-metric-card settings-metric-card-${tone}`}>
      <div className="settings-metric-head">
        <span className="settings-metric-icon">{icon}</span>
        <div>
          <span>{title}</span>
          <strong>{percent}%</strong>
        </div>
      </div>
      <Progress
        percent={percent}
        strokeColor={strokeColor}
        trailColor="rgba(148, 163, 184, 0.18)"
        showInfo={false}
      />
      <small>实时资源占用</small>
    </div>
  );
}

function InfoSummaryCard({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="settings-summary-card">
      <span className="settings-summary-icon">{icon}</span>
      <small>{label}</small>
      <strong>{value}</strong>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="settings-info-item">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function resolveSystemHealth(systemInfo: SystemInfo | null) {
  if (!systemInfo) {
    return "采集中";
  }
  const peak = Math.max(systemInfo.cpu_percent ?? 0, systemInfo.memory_percent ?? 0, systemInfo.disk_percent ?? 0);
  if (peak >= 90) {
    return "高负载";
  }
  if (peak >= 70) {
    return "注意观察";
  }
  return "运行正常";
}

function formatRuntimeLabel(systemInfo: SystemInfo | null) {
  const os = systemInfo?.os?.trim();
  const arch = systemInfo?.arch?.trim();
  if (os && arch) {
    return `${os} / ${arch}`;
  }
  return os || arch || "-";
}

function metricStrokeColor(tone: "cyan" | "green" | "amber") {
  if (tone === "cyan") {
    return "#0891b2";
  }
  if (tone === "green") {
    return "#0f766e";
  }
  return "#d97706";
}

function formatBytes(value?: number) {
  if (!value || value < 1) {
    return "大小未知";
  }
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size >= 10 || unit === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unit]}`;
}

async function waitForUpdatedService(version: string) {
  await new Promise((resolve) => window.setTimeout(resolve, 1500));
  for (let attempt = 0; attempt < 30; attempt += 1) {
    try {
      const response = await fetch(`/api/public/version?updated=${Date.now()}`, { cache: "no-store" });
      if (response.ok) {
        const payload = (await response.json()) as { version?: string };
        if (payload.version?.trim() === version) {
          return;
        }
      }
    } catch {
      // A connection failure is expected while the service is restarting.
    }
    await new Promise((resolve) => window.setTimeout(resolve, 1000));
  }
  throw new Error("新版本已安装，但服务未在预期时间内恢复，请检查服务日志");
}
