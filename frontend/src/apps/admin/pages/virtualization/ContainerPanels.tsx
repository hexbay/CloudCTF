import {
  DeleteOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  StopOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { ContainerGroupStatus, statusColor } from "./constants";
import type { PageState, RowData } from "./types";
import {
  formatDateTime,
  normalizePage,
  portEntries,
  renderContainerStatus,
  renderPortList,
  renderPortSummary,
  renderValue,
} from "./utils";

export function ContainerPanel() {
  const { message: messageApi } = App.useApp();
  const [detail, setDetail] = useState<RowData | null>(null);
  const [rows, setRows] = useState<RowData[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState<PageState>({ current: 1, pageSize: 10 });
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [editing, setEditing] = useState<RowData | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [modal, contextHolder] = Modal.useModal();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [actingKey, setActingKey] = useState<string | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load(nextPage = page) {
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        "/api-admin/virtualization/containers",
        {
          current: nextPage.current,
          pageSize: nextPage.pageSize,
          keyword: keyword || undefined,
        },
      );
      const normalized = normalizePage(result);
      setRows(normalized.rows);
      setTotal(normalized.total);
    } catch (error) {
      messageApi.error(
        error instanceof Error ? error.message : "加载容器失败",
      );
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setEditing({});
    form.resetFields();
  }

  function openEdit(row: RowData) {
    setEditing(row);
    form.setFieldsValue(row);
  }

  async function save() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing?.id) {
        await adminApi.update(
          `/api-admin/virtualization/containers/${editing.id}`,
          values,
        );
        messageApi.success("容器已更新");
      } else {
        await adminApi.create("/api-admin/virtualization/containers", values);
        messageApi.success("容器已创建");
      }
      setEditing(null);
      await load();
    } catch (error) {
      messageApi.error(
        error instanceof Error ? error.message : "保存容器失败",
      );
    } finally {
      setSaving(false);
    }
  }

  async function confirmRemove(row: RowData) {
    setDeletingId(String(row.id));
    try {
      await adminApi.remove(`/api-admin/virtualization/containers/${row.id}`);
      messageApi.success("容器已删除");
      await load();
    } catch (error) {
      messageApi.error(
        error instanceof Error ? error.message : "删除容器失败",
      );
      throw error;
    } finally {
      setDeletingId(null);
    }
  }

  function remove(row: RowData) {
    modal.confirm({
      title: `删除容器 ${row.name ?? row.id}`,
      content: "删除后不可恢复。",
      okButtonProps: { danger: true, loading: deletingId === String(row.id) },
      onOk: () => confirmRemove(row),
    });
  }

  async function runAction(row: RowData, action: "start" | "stop" | "restart") {
    const key = `${String(row.id)}:${action}`;
    setActingKey(key);
    try {
      await adminApi.action(
        "/api-admin/virtualization/containers/action",
        "POST",
        {
          host_id: row.host_id,
          container_id: row.container_id ?? row.id,
          action,
        },
      );
      messageApi.success(
        action === "start"
          ? "容器已启动"
          : action === "stop"
            ? "容器已停止"
            : "容器已重启",
      );
      await load();
    } catch (error) {
      messageApi.error(
        error instanceof Error ? error.message : "容器操作失败",
      );
    } finally {
      setActingKey(null);
    }
  }

  const columns: ColumnsType<RowData> = [
    {
      title: "名称",
      dataIndex: "name",
      width: 220,
      ellipsis: true,
      render: (value, row) => (
        <Space direction="vertical" size={2}>
          <Typography.Text strong>{String(value ?? "-")}</Typography.Text>
          <Typography.Text type="secondary">
            {String(row.container_id ?? row.id ?? "-")}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: "镜像",
      dataIndex: "image",
      ellipsis: true,
      render: (value) => String(value ?? "-"),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 120,
      render: (value) => renderContainerStatus(value),
    },
    {
      title: "主机",
      key: "host",
      width: 180,
      render: (_, row) => (
        <Space direction="vertical" size={2}>
          <Typography.Text>
            {String(row.host_name ?? row.host_id ?? "-")}
          </Typography.Text>
          <Typography.Text type="secondary">
            出口 IP: {String(row.out_ip || "-")}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: "端口映射",
      dataIndex: "ports",
      width: 220,
      render: (value) => renderPortSummary(value),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 180,
      render: (value, row) => formatDateTime(value ?? row.created),
    },
    {
      title: "操作",
      key: "actions",
      width: 380,
      fixed: "right",
      render: (_, row) => (
        <Space wrap>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setDetail(row)}
          >
            详情
          </Button>
          <Button
            size="small"
            icon={<PlayCircleOutlined />}
            loading={actingKey === `${String(row.id)}:start`}
            disabled={
              String(row.status).toLowerCase() === "running" ||
              (actingKey !== null &&
                actingKey !== `${String(row.id)}:start`)
            }
            onClick={() => void runAction(row, "start")}
          >
            启动
          </Button>
          <Button
            size="small"
            danger
            icon={<StopOutlined />}
            loading={actingKey === `${String(row.id)}:stop`}
            disabled={
              String(row.status).toLowerCase() === "stopped" ||
              (actingKey !== null &&
                actingKey !== `${String(row.id)}:stop`)
            }
            onClick={() => void runAction(row, "stop")}
          >
            停止
          </Button>
          <Button
            size="small"
            icon={<PauseCircleOutlined />}
            loading={actingKey === `${String(row.id)}:restart`}
            disabled={
              actingKey !== null && actingKey !== `${String(row.id)}:restart`
            }
            onClick={() => void runAction(row, "restart")}
          >
            重启
          </Button>
          <Button
            size="small"
            danger
            icon={<DeleteOutlined />}
            loading={deletingId === String(row.id)}
            onClick={() => remove(row)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card className="admin-table-card" bordered={false}>
      {contextHolder}
      <div className="resource-toolbar">
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="搜索容器"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            onSearch={() => {
              const next = { ...page, current: 1 };
              setPage(next);
              void load(next);
            }}
            style={{ width: 260 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
        <Space wrap>
          <Button type="primary" onClick={openCreate}>
            新建
          </Button>
        </Space>
      </div>
      <Table
        rowKey={(row) => String(row.id)}
        columns={columns}
        dataSource={rows}
        loading={loading}
        scroll={{ x: true }}
        pagination={{
          current: page.current,
          pageSize: page.pageSize,
          total,
          showSizeChanger: true,
          onChange: (current, pageSize) => {
            const next = { current, pageSize };
            setPage(next);
            void load(next);
          },
        }}
      />
      <Modal
        title={editing?.id ? "编辑容器" : "新建容器"}
        open={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={() => void save()}
        confirmLoading={saving}
        width={720}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="容器名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="image" label="镜像" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="host_id"
            label="主机 ID"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="ports" label="端口映射 JSON">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="environment" label="环境变量 JSON">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="expires_at" label="过期时间">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
      <ContainerDetailModal
        record={detail}
        onClose={() => setDetail(null)}
      />
    </Card>
  );
}

function ContainerDetailModal({
  record,
  onClose,
}: {
  record: RowData | null;
  onClose: () => void;
}) {
  return (
    <Modal
      title="容器详情"
      open={Boolean(record)}
      footer={null}
      onCancel={onClose}
      width={760}
    >
      {record ? (
        <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label="容器名称">
            {String(record.name ?? "-")}
          </Descriptions.Item>
          <Descriptions.Item label="容器 ID">
            <Typography.Text copyable>
              {String(record.container_id ?? record.id ?? "-")}
            </Typography.Text>
          </Descriptions.Item>
          <Descriptions.Item label="主机">
            {String(record.host_name ?? record.host_id ?? "-")}
          </Descriptions.Item>
          <Descriptions.Item label="出口 IP">
            {String(record.out_ip || "-")}
          </Descriptions.Item>
          <Descriptions.Item label="镜像">
            {String(record.image ?? "-")}
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            {renderContainerStatus(record.status ?? record.state)}
          </Descriptions.Item>
          <Descriptions.Item label="端口映射">
            {renderPortList(record.ports)}
          </Descriptions.Item>
          <Descriptions.Item label="启动命令">
            {record.command ? (
              <Typography.Text code>{String(record.command)}</Typography.Text>
            ) : (
              "-"
            )}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {formatDateTime(record.created_at ?? record.created)}
          </Descriptions.Item>
        </Descriptions>
      ) : null}
    </Modal>
  );
}

export function ContainerGroupPanel() {
  const [modal, contextHolder] = Modal.useModal();
  const [rows, setRows] = useState<RowData[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState<PageState>({ current: 1, pageSize: 10 });
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<string>();
  const [filters, setFilters] = useState({
    challenge_name: "",
    user_name: "",
    vuln_lab_name: "",
  });
  const [startingId, setStartingId] = useState<string | null>(null);
  const [stoppingId, setStoppingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load(
    nextPage = page,
    nextStatus = status,
    nextFilters = filters,
  ) {
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        "/api-admin/virtualization/container-groups",
        {
          current: nextPage.current,
          pageSize: nextPage.pageSize,
          status: nextStatus || undefined,
          challenge_name: nextFilters.challenge_name || undefined,
          user_name: nextFilters.user_name || undefined,
          vuln_lab_name: nextFilters.vuln_lab_name || undefined,
        },
      );
      const normalized = normalizePage(result);
      setRows(normalized.rows);
      setTotal(normalized.total);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载容器组失败");
    } finally {
      setLoading(false);
    }
  }

  async function startGroup(row: RowData) {
    setStartingId(String(row.id));
    try {
      await adminApi.action(
        `/api-admin/virtualization/container-groups/${row.id}/start`,
        "POST",
      );
      message.success("容器组已启动");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "启动容器组失败");
    } finally {
      setStartingId(null);
    }
  }

  async function stopGroup(row: RowData) {
    setStoppingId(String(row.id));
    try {
      await adminApi.action(
        `/api-admin/virtualization/container-groups/${row.id}/stop`,
        "POST",
      );
      message.success("容器组已停止");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "停止容器组失败");
    } finally {
      setStoppingId(null);
    }
  }

  async function confirmRemoveGroup(row: RowData) {
    setDeletingId(String(row.id));
    try {
      await adminApi.remove(
        `/api-admin/virtualization/container-groups/${row.id}`,
      );
      message.success("容器组已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除容器组失败");
      throw error;
    } finally {
      setDeletingId(null);
    }
  }

  function remove(row: RowData) {
    modal.confirm({
      title: `删除容器组 ${row.name ?? row.id}`,
      content: "删除后会连同组内容器一起清理，且不可恢复。",
      okButtonProps: { danger: true, loading: deletingId === String(row.id) },
      onOk: () => confirmRemoveGroup(row),
    });
  }

  const columns: ColumnsType<RowData> = [
    {
      title: "关联",
      key: "binding",
      width: 220,
      render: (_, row) => {
        const challenge = row.challenge as RowData | undefined;
        const vulnLab = row.vul_lab as RowData | undefined;
        const challengeName = displayText(challenge?.name, row.challenge_name);
        const vulnLabName = displayText(
          vulnLab?.name,
          row.vuln_lab_name,
          row.vul_lab_name,
        );
        const templateName = displayText(row.template_name);
        if (challengeName) return <Tag color="blue">题目: {challengeName}</Tag>;
        if (vulnLabName) return <Tag color="orange">漏洞: {vulnLabName}</Tag>;
        if (templateName)
          return <Tag color="geekblue">模板: {templateName}</Tag>;
        return <Typography.Text type="secondary">-</Typography.Text>;
      },
    },
    {
      title: "名称",
      dataIndex: "name",
      width: 180,
      ellipsis: true,
      render: (value) => String(value ?? "-"),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 100,
      render: (value) => {
        const meta = ContainerGroupStatus.find((item) => item.value === value);
        return (
          <Tag color={meta?.color ?? statusColor[String(value)]}>
            {meta?.label ?? String(value ?? "-")}
          </Tag>
        );
      },
    },
    {
      title: "主机",
      key: "host",
      width: 140,
      render: (_, row) =>
        String(
          (row.host as RowData | undefined)?.name ??
            row.host_name ??
            row.host_id ??
            "-",
        ),
    },
    {
      title: "用户",
      key: "user",
      width: 140,
      render: (_, row) =>
        String(
          (row.user as RowData | undefined)?.username ??
            row.user_name ??
            row.user_id ??
            "-",
        ),
    },
    {
      title: "Flag",
      dataIndex: "flag",
      width: 220,
      ellipsis: true,
      render: (value, row) =>
        isVulnLabGroup(row) ? (
          <Typography.Text type="secondary">-</Typography.Text>
        ) : value ? (
          <Typography.Text code copyable>
            {String(value)}
          </Typography.Text>
        ) : (
          <Typography.Text type="secondary">-</Typography.Text>
        ),
    },
    {
      title: "端口映射",
      key: "ports",
      width: 320,
      render: (_, row) => renderContainerGroupPorts(row),
    },
    {
      title: "过期时间",
      dataIndex: "expires_at",
      width: 180,
      render: (value) => renderValue(value),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 180,
      render: (value) => renderValue(value),
    },
    {
      title: "操作",
      key: "actions",
      width: 180,
      fixed: "right",
      render: (_, row) => (
        <Space wrap>
          <Button
            size="small"
            loading={startingId === String(row.id)}
            onClick={() => void startGroup(row)}
            disabled={
              String(row.status) === "running" || stoppingId === String(row.id)
            }
          >
            启动
          </Button>
          <Button
            size="small"
            danger
            loading={stoppingId === String(row.id)}
            onClick={() => void stopGroup(row)}
            disabled={
              String(row.status) === "stopped" || startingId === String(row.id)
            }
          >
            停止
          </Button>
          <Button
            size="small"
            danger
            loading={deletingId === String(row.id)}
            icon={<DeleteOutlined />}
            onClick={() => remove(row)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card className="admin-table-card" bordered={false}>
      {contextHolder}
      <div className="resource-toolbar">
        <Space wrap>
          <Select
            allowClear
            placeholder="状态"
            value={status}
            options={ContainerGroupStatus.map(({ label, value }) => ({
              label,
              value,
            }))}
            onChange={(value) => {
              const nextStatus = value ? String(value) : undefined;
              setStatus(nextStatus);
              const nextPage = { ...page, current: 1 };
              setPage(nextPage);
              void load(nextPage, nextStatus, filters);
            }}
            style={{ width: 180 }}
          />
          <Input.Search
            allowClear
            placeholder="题目名称"
            value={filters.challenge_name}
            onChange={(event) =>
              setFilters((prev) => ({
                ...prev,
                challenge_name: event.target.value,
              }))
            }
            onSearch={() => {
              const nextPage = { ...page, current: 1 };
              setPage(nextPage);
              void load(nextPage);
            }}
            style={{ width: 220 }}
          />
          <Input.Search
            allowClear
            placeholder="用户名"
            value={filters.user_name}
            onChange={(event) =>
              setFilters((prev) => ({ ...prev, user_name: event.target.value }))
            }
            onSearch={() => {
              const nextPage = { ...page, current: 1 };
              setPage(nextPage);
              void load(nextPage);
            }}
            style={{ width: 200 }}
          />
          <Input.Search
            allowClear
            placeholder="漏洞名称"
            value={filters.vuln_lab_name}
            onChange={(event) =>
              setFilters((prev) => ({
                ...prev,
                vuln_lab_name: event.target.value,
              }))
            }
            onSearch={() => {
              const nextPage = { ...page, current: 1 };
              setPage(nextPage);
              void load(nextPage);
            }}
            style={{ width: 220 }}
          />
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void load()}
          >
            刷新
          </Button>
        </Space>
      </div>
      <Table
        rowKey={(row) => String(row.id)}
        columns={columns}
        dataSource={rows}
        loading={loading}
        scroll={{ x: true }}
        expandable={{
          expandedRowRender: (row) => {
            const containers = Array.isArray(row.containers)
              ? (row.containers as RowData[])
              : [];
            if (!containers.length) {
              return (
                <Typography.Text type="secondary">暂无容器</Typography.Text>
              );
            }

            const showServiceName = containers.length > 1;

            return (
              <Table
                rowKey={(container) =>
                  String(
                    container.id ?? container.container_id ?? container.name,
                  )
                }
                pagination={false}
                size="small"
                dataSource={containers}
                columns={[
                  {
                    title: showServiceName ? "服务 / 容器" : "容器名称",
                    key: "service_name",
                    width: 240,
                    ellipsis: true,
                    render: (_, container) => {
                      const serviceName = displayText(container.service_name);
                      const containerName = displayText(
                        container.name,
                        container.container_id,
                        container.id,
                      );
                      if (!showServiceName) {
                        return containerName || "-";
                      }
                      return (
                        <Space direction="vertical" size={2}>
                          <Typography.Text strong>
                            {serviceName || "service"}
                          </Typography.Text>
                          <Typography.Text type="secondary">
                            {containerName || "-"}
                          </Typography.Text>
                        </Space>
                      );
                    },
                  },
                  {
                    title: "状态",
                    dataIndex: "status",
                    width: 100,
                    render: (value) => {
                      const meta = ContainerGroupStatus.find(
                        (item) => item.value === value,
                      );
                      return (
                        <Tag color={meta?.color ?? statusColor[String(value)]}>
                          {meta?.label ?? String(value ?? "-")}
                        </Tag>
                      );
                    },
                  },
                  { title: "镜像", dataIndex: "image", ellipsis: true },
                  {
                    title: showServiceName ? "开放端口" : "端口映射",
                    dataIndex: "ports",
                    width: 280,
                    render: (value) => renderValue(value, "ports"),
                  },
                ]}
              />
            );
          },
        }}
        pagination={{
          current: page.current,
          pageSize: page.pageSize,
          total,
          showSizeChanger: true,
          onChange: (current, pageSize) => {
            const next = { current, pageSize };
            setPage(next);
            void load(next);
          },
        }}
      />
    </Card>
  );
}

function displayText(...values: unknown[]) {
  for (const value of values) {
    const text = String(value ?? "").trim();
    if (text && text !== "<nil>" && text !== "undefined" && text !== "null") {
      return text;
    }
  }
  return "";
}

function isVulnLabGroup(row: RowData) {
  return Boolean(
    displayText(
      row.vuln_lab_name,
      row.vul_lab_name,
      (row.vul_lab as RowData | undefined)?.name,
    ) || row.vuln_lab_id,
  );
}

function renderContainerGroupPorts(row: RowData) {
  const containers = Array.isArray(row.containers)
    ? (row.containers as RowData[])
    : [];
  if (!containers.length) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }

  const showServiceName = containers.length > 1;
  const items = containers
    .map((container) => {
      const ports = normalizePortMappings(container.ports);
      if (!ports.length) {
        return null;
      }
      return {
        serviceName: displayText(container.service_name),
        ports,
      };
    })
    .filter(Boolean) as Array<{ serviceName: string; ports: string[] }>;

  if (!items.length) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }

  return (
    <Space direction="vertical" size={4}>
      {items.map((item, index) => (
        <Typography.Text key={`${item.serviceName}-${index}`}>
          {showServiceName && item.serviceName ? `${item.serviceName}: ` : ""}
          {item.ports.join(", ")}
        </Typography.Text>
      ))}
    </Space>
  );
}

function normalizePortMappings(value: unknown): string[] {
  return portEntries(value).map(
    ([hostPort, containerPort]) => `${containerPort} → ${hostPort}`,
  );
}
