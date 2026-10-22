import {
  BuildOutlined,
  CodeOutlined,
  DeleteOutlined,
  EditOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Form,
  Input,
  List,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Key,
  type ReactNode,
} from "react";
import { adminApi } from "@shared/api/admin";
import {
  repoTypeOptions,
  statusMeta,
  templateStatusOptions,
} from "./constants";
import { DetailModal, ResourceRepositoriesModal } from "./SharedPanels";
import type { PageState, RowData } from "./types";
import {
  columnLabel,
  enumLabels,
  formatDateTime,
  formatBytes,
  normalizePage,
  normalizePayload,
  normalizeRows,
  renderHostIcon,
  renderValue,
} from "./utils";
export function TemplatePanel() {
  const [repoModalOpen, setRepoModalOpen] = useState(false);
  const [rows, setRows] = useState<RowData[]>([]);
  const [hosts, setHosts] = useState<RowData[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState<PageState>({ current: 1, pageSize: 10 });
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [hostId, setHostId] = useState<number>();
  const [status, setStatus] = useState<string>();
  const [templateType, setTemplateType] = useState<string>();
  const [editing, setEditing] = useState<RowData | null>(null);
  const [detail, setDetail] = useState<RowData | null>(null);
  const [saving, setSaving] = useState(false);
  const [startingId, setStartingId] = useState<string | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [form] = Form.useForm();
  const [modal, contextHolder] = Modal.useModal();
  const [drawerRow, setDrawerRow] = useState<RowData | null>(null);
  const [buildModalOpen, setBuildModalOpen] = useState(false);
  const [buildingRow, setBuildingRow] = useState<RowData | null>(null);
  const [buildLogs, setBuildLogs] = useState<string[]>([]);
  const [buildStatus, setBuildStatus] = useState<
    "building" | "completed" | "failed" | null
  >(null);
  const buildEventSourceRef = useRef<EventSource | null>(null);
  const buildLogContainerRef = useRef<HTMLDivElement | null>(null);

  const handleRowSelect = useCallback((keys: Key[]) => {
    setSelectedRowKeys(keys);
  }, []);

  const openDrawer = useCallback((row: RowData) => {
    setDrawerRow(row);
  }, []);

  const closeDrawer = useCallback(() => {
    setDrawerRow(null);
  }, []);

  async function load(nextPage = page) {
    setLoading(true);
    try {
      const [templateResult, hostResult] = await Promise.all([
        adminApi.list<RowData>("/api-admin/virtualization/templates", {
          current: nextPage.current,
          pageSize: nextPage.pageSize,
          keyword: keyword || undefined,
          host_id: hostId,
          status,
          template_type: templateType || undefined,
        }),
        adminApi.list<RowData>("/api-admin/virtualization/hosts", {
          current: 1,
          pageSize: 200,
        }),
      ]);
      const normalized = normalizePage(templateResult);
      setRows(normalized.rows);
      setTotal(normalized.total);
      setHosts(normalizeRows(hostResult));
      setSelectedRowKeys((keys) =>
        keys.filter((key) =>
          normalized.rows.some((row) => String(row.id) === String(key)),
        ),
      );
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载模板失败");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const hostMap = useMemo(() => {
    return new Map(hosts.map((host) => [String(host.id), host]));
  }, [hosts]);

  function openCreate() {
    setEditing({});
    form.resetFields();
    form.setFieldsValue({
      template_type: "vul",
      status: "pending",
      compose_file: "services:\n  app:\n    image: nginx:latest\n",
    });
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
          `/api-admin/virtualization/templates/${editing.id}`,
          values,
        );
        message.success("模板已更新");
      } else {
        await adminApi.create("/api-admin/virtualization/templates", values);
        message.success("模板已创建");
      }
      setEditing(null);
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存模板失败");
    } finally {
      setSaving(false);
    }
  }

  async function runTemplateAction(row: RowData, action: "build" | "sync") {
    if (action === "build") {
      try {
        const result = await adminApi.action<{ task?: RowData }>(
          `/api-admin/virtualization/templates/${row.id}/build`,
          "POST",
        );
        await openBuildModal(row);
        if (result.task) {
          message.success("后台构建任务已创建");
        }
      } catch (error) {
        message.error(
          error instanceof Error ? error.message : "提交构建任务失败",
        );
      }
    } else {
      try {
        await adminApi.action(
          `/api-admin/virtualization/templates/${row.id}/sync`,
          "POST",
        );
        message.success("同步已触发");
        await load();
      } catch (error) {
        message.error(error instanceof Error ? error.message : "操作失败");
      }
    }
  }

  async function startTemplate(row: RowData) {
    const id = String(row.id ?? "");
    setStartingId(id);
    try {
      const result = await adminApi.action<{ group?: RowData }>(
        `/api-admin/virtualization/templates/${row.id}/start`,
        "POST",
      );
      message.success(
        result.group?.id
          ? `模板已启动，容器组 #${result.group.id} 已创建`
          : "模板已启动",
      );
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "启动模板失败");
    } finally {
      setStartingId(null);
    }
  }

  function isStartingTemplate(row: RowData) {
    return startingId === String(row.id ?? "");
  }

  async function openBuildModal(row: RowData) {
    setBuildingRow(row);
    setBuildLogs([]);
    setBuildStatus("building");
    setBuildModalOpen(true);

    if (buildEventSourceRef.current) {
      buildEventSourceRef.current.close();
    }

    const path = `/api-admin/virtualization/templates/${row.id}/build/stream`;
    const ticketResult = await adminApi.action<{ ticket: string }>(
      "/api-admin/auth/stream-ticket",
      "POST",
      { path },
    );
    const streamUrl = new URL(
      path,
      window.location.origin,
    );
    streamUrl.searchParams.set("stream_ticket", ticketResult.ticket);
    const eventSource = new EventSource(streamUrl.toString());
    buildEventSourceRef.current = eventSource;

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.log) {
          setBuildLogs((prev) => [...prev, data.log]);
        }
        if (data.status === "completed") {
          setBuildStatus("completed");
          message.open({
            type: "success",
            content: "模板构建完成",
            duration: 2.5,
            key: `template-build-${row.id}`,
          });
          eventSource.close();
          void load();
        } else if (data.status === "failed" || data.status === "error") {
          setBuildStatus("failed");
          message.open({
            type: "error",
            content: "模板构建失败: " + (data.log || ""),
            duration: 3,
            key: `template-build-${row.id}`,
          });
          eventSource.close();
          void load();
        }
      } catch {
        setBuildLogs((prev) => [...prev, event.data]);
      }
    };

    eventSource.onerror = () => {
      if (eventSource.readyState === EventSource.CLOSED) {
        return;
      }
      setBuildLogs((prev) => [
        ...prev,
        "日志流连接异常，构建可能已中断或后端提前关闭连接。",
      ]);
      setBuildStatus("failed");
      message.error("构建连接出错");
      eventSource.close();
    };
  }

  function closeBuildModal() {
    setBuildModalOpen(false);
  }

  // 清理 EventSource
  useEffect(() => {
    return () => {
      if (buildEventSourceRef.current) {
        buildEventSourceRef.current.close();
      }
    };
  }, []);

  useEffect(() => {
    const container = buildLogContainerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  }, [buildLogs, buildModalOpen]);

  function remove(row: RowData) {
    modal.confirm({
      title: `删除模板 ${row.name ?? row.id}`,
      content: "删除后不可恢复，关联题目将无法继续使用该模板。",
      okButtonProps: { danger: true },
      onOk: async () => {
        await adminApi.remove(`/api-admin/virtualization/templates/${row.id}`);
        message.success("模板已删除");
        await load();
      },
    });
  }

  function batchRemove() {
    if (!selectedRowKeys.length) {
      message.warning("请选择要删除的模板");
      return;
    }
    modal.confirm({
      title: `批量删除 ${selectedRowKeys.length} 个模板`,
      content:
        "删除后不可恢复。用于清理旧同步数据时，请确认相关题目没有继续引用这些模板。",
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          const result = await adminApi.action<{ deleted?: number }>(
            "/api-admin/virtualization/templates/batch/delete",
            "POST",
            { ids: selectedRowKeys.map((key) => String(key)) },
          );
          message.success(
            `已删除 ${result.deleted ?? selectedRowKeys.length} 个模板`,
          );
          setSelectedRowKeys([]);
          await load({ ...page, current: 1 });
        } catch (error) {
          message.error(
            error instanceof Error ? error.message : "批量删除失败",
          );
        }
      },
    });
  }

  function batchBuild() {
    if (!selectedRowKeys.length) {
      message.warning("请选择要构建的模板");
      return;
    }
    modal.confirm({
      title: `批量构建 ${selectedRowKeys.length} 个模板`,
      content: "构建任务会在后台继续执行，可以关闭日志弹窗后稍后回来查看结果。",
      onOk: async () => {
        try {
          const result = await adminApi.action<{ tasks?: RowData[]; count?: number }>(
            "/api-admin/virtualization/templates/batch/build",
            "POST",
            { ids: selectedRowKeys.map((key) => String(key)) },
          );
          message.success(`已提交 ${result.count ?? result.tasks?.length ?? 0} 个后台构建任务`);
          const first = rows.find((row) => String(row.id) === String(selectedRowKeys[0]));
          if (first) {
            openBuildModal(first);
          }
          await load();
        } catch (error) {
          message.error(
            error instanceof Error ? error.message : "批量构建失败",
          );
        }
      },
    });
  }

  const tableColumns = useMemo<ColumnsType<RowData>>(
    () => [
      {
        title: "模板名称",
        dataIndex: "name",
        width: 190,
        ellipsis: true,
        render: (value, row) => (
          <Space direction="vertical" size={2}>
            <Typography.Link strong onClick={() => openDrawer(row)}>
              {String(value ?? "-")}
            </Typography.Link>
            <Typography.Text type="secondary">
              {String(row.repository_path ?? row.compose_path ?? "-")}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: "类型",
        dataIndex: "template_type",
        width: 100,
        render: (value) => renderValue(value, "template_type"),
      },
      {
        title: "主机",
        dataIndex: "host_id",
        width: 150,
        render: (value) => renderTemplateHost(value, hostMap),
      },
      {
        title: "服务/镜像",
        key: "services",
        width: 200,
        render: (_, row) => renderTemplateServicesSummary(row),
      },
      {
        title: "端口",
        key: "ports",
        width: 150,
        render: (_, row) => renderTemplatePorts(row),
      },
      {
        title: "状态",
        dataIndex: "status",
        width: 100,
        render: (value) => renderValue(value, "status"),
      },
      {
        title: "标签",
        dataIndex: "tags",
        width: 120,
        render: (value) => renderValue(value, "tags"),
      },
      {
        title: "更新时间",
        dataIndex: "updated_at",
        width: 150,
        render: formatDateTime,
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
              aria-label="编辑模板"
              title="编辑模板"
              onClick={(e) => {
                e.stopPropagation();
                openEdit(row);
              }}
            />
            <Button
              size="small"
              loading={isStartingTemplate(row)}
              icon={<PlayCircleOutlined />}
              aria-label="启动模板"
              title="启动模板"
              onClick={(e) => {
                e.stopPropagation();
                void startTemplate(row);
              }}
              disabled={
                String(row.status ?? "") !== "completed" &&
                String(row.status ?? "") !== "ready"
              }
            />
            <Button
              size="small"
              icon={<BuildOutlined />}
              aria-label="构建模板"
              title="构建模板"
              onClick={(e) => {
                e.stopPropagation();
                void runTemplateAction(row, "build");
              }}
            />
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              aria-label="删除模板"
              title="删除模板"
              onClick={(e) => {
                e.stopPropagation();
                remove(row);
              }}
            />
          </Space>
        ),
      },
    ],
    [hostMap, openDrawer, startingId],
  );

  const rowSelection = useMemo(
    () => ({
      selectedRowKeys,
      onChange: handleRowSelect,
    }),
    [selectedRowKeys, handleRowSelect],
  );

  return (
    <>
      {contextHolder}
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Space wrap>
            <Input.Search
              allowClear
              placeholder="搜索模板名称、镜像、仓库"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              onSearch={() => {
                const next = { ...page, current: 1 };
                setPage(next);
                void load(next);
              }}
              style={{ width: 280 }}
            />
            <Select
              allowClear
              placeholder="类型"
              value={templateType}
              onChange={setTemplateType}
              style={{ width: 140 }}
              options={repoTypeOptions}
            />
            <Select
              allowClear
              placeholder="主机"
              value={hostId}
              onChange={setHostId}
              style={{ width: 180 }}
              options={hosts.map((host) => ({
                label: host.name,
                value: Number(host.id),
              }))}
            />
            <Select
              allowClear
              placeholder="状态"
              value={status}
              onChange={setStatus}
              style={{ width: 140 }}
              options={templateStatusOptions}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                const next = { ...page, current: 1 };
                setPage(next);
                void load(next);
              }}
            >
              查询
            </Button>
          </Space>
          <Space wrap>
            <Button onClick={() => setRepoModalOpen(true)}>资源仓库</Button>
            <Button
              disabled={!selectedRowKeys.length}
              onClick={() => batchBuild()}
            >
              批量构建
            </Button>
            <Button
              danger
              disabled={!selectedRowKeys.length}
              icon={<DeleteOutlined />}
              onClick={() => batchRemove()}
            >
              批量删除
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
          </Space>
        </div>
        <Table
          rowKey={(row) => String(row.id)}
          columns={tableColumns}
          dataSource={rows}
          loading={loading}
          scroll={{ x: "max-content" }}
          rowSelection={rowSelection}
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
          title={editing?.id ? "编辑模板" : "新建模板"}
          open={Boolean(editing)}
          onCancel={() => setEditing(null)}
          onOk={() => void save()}
          confirmLoading={saving}
          width={820}
          destroyOnClose
        >
          <Form form={form} layout="vertical">
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="name"
                  label="模板名称"
                  rules={[{ required: true }]}
                >
                  <Input />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="host_id"
                  label="关联主机"
                  rules={[{ required: true }]}
                >
                  <Select
                    placeholder="选择模板部署主机"
                    options={hosts.map((host) => ({
                      label: `${host.name} (${enumLabels.host_type[String(host.host_type)] ?? host.host_type})`,
                      value: Number(host.id),
                    }))}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="template_type" label="模板类型">
                  <Select
                    options={[
                      { label: "漏洞", value: "vul" },
                      { label: "CTF", value: "ctf" },
                      { label: "Docker Compose", value: "compose" },
                      { label: "Docker", value: "docker" },
                      { label: "Proxmox VE", value: "pve" },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="status" label="状态">
                  <Select
                    options={templateStatusOptions}
                  />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="image" label="默认镜像">
                  <Input />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="repository_url" label="仓库地址">
                  <Input />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item name="repository_branch" label="分支">
                  <Input />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item name="repository_path" label="模板路径">
                  <Input />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="compose_file" label="Compose 内容">
                  <Input.TextArea autoSize={{ minRows: 6, maxRows: 14 }} />
                </Form.Item>
              </Col>
              <Col span={24}>
                <Form.Item name="description" label="描述">
                  <Input.TextArea autoSize={{ minRows: 2, maxRows: 5 }} />
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </Modal>
        <DetailModal
          title="模板详情"
          record={detail}
          onClose={() => setDetail(null)}
        />
      </Card>
      <ResourceRepositoriesModal
        open={repoModalOpen}
        onClose={() => setRepoModalOpen(false)}
        onSynced={() => load()}
      />
      <Drawer
        className="template-detail-drawer"
        title={drawerRow ? <TemplateDrawerTitle row={drawerRow} /> : "模板详情"}
        placement="right"
        width="min(560px, calc(100vw - 48px))"
        open={Boolean(drawerRow)}
        onClose={closeDrawer}
      >
        {drawerRow && (
          <Space
            direction="vertical"
            size={16}
            className="template-detail-body"
          >
            <TemplateSummary row={drawerRow} hostMap={hostMap} />
            <TemplateServicesTable row={drawerRow} />
          </Space>
        )}
      </Drawer>
      <Modal
        title={
          <Space>
            <span>模板构建日志</span>
            {buildStatus === "building" && <Tag color="blue">构建中</Tag>}
            {buildStatus === "completed" && <Tag color="green">完成</Tag>}
            {buildStatus === "failed" && <Tag color="red">失败</Tag>}
          </Space>
        }
        open={buildModalOpen}
        onCancel={closeBuildModal}
        footer={null}
        width={760}
        destroyOnClose
      >
        <div className="build-log-container">
          <Alert
            showIcon
            type={
              buildStatus === "completed"
                ? "success"
                : buildStatus === "failed"
                  ? "error"
                  : "info"
            }
            message={
              buildStatus === "completed"
                ? "构建完成"
                : buildStatus === "failed"
                  ? "构建失败"
                  : "构建中"
            }
          />
          <div
            ref={buildLogContainerRef}
            className="build-log-list"
            style={{
              maxHeight: "380px",
              overflow: "auto",
              background:
                "linear-gradient(180deg, rgb(21, 24, 31) 0%, rgb(14, 16, 21) 100%)",
              color: "#d4d4d4",
              padding: "10px 12px",
              borderRadius: "6px",
              marginTop: "8px",
              fontFamily: "monospace",
              fontSize: "13px",
              lineHeight: "1.6",
              border: "1px solid rgba(255,255,255,0.06)",
              boxShadow: "inset 0 1px 0 rgba(255,255,255,0.04)",
            }}
          >
            {buildLogs.length === 0 ? (
              <Typography.Text type="secondary" style={{ color: "#888" }}>
                暂未收到日志...
              </Typography.Text>
            ) : (
              <List
                size="small"
                dataSource={buildLogs}
                renderItem={(log) => (
                  <List.Item style={{ padding: "2px 0", border: "none" }}>
                    <Typography.Text
                      style={{
                        color: "#d4d4d4",
                        fontFamily: "monospace",
                        whiteSpace: "pre-wrap",
                        wordBreak: "break-word",
                      }}
                    >
                      {log}
                    </Typography.Text>
                  </List.Item>
                )}
              />
            )}
          </div>
        </div>
      </Modal>
    </>
  );
}

function TemplateDrawerTitle({ row }: { row: RowData }) {
  return (
    <Space direction="vertical" size={4} className="template-drawer-title">
      <Typography.Text strong>{String(row.name ?? "模板详情")}</Typography.Text>
      <Space size={6} wrap>
        {renderValue(row.template_type, "template_type")}
        {renderValue(row.status, "status")}
      </Space>
    </Space>
  );
}

function TemplateSummary({
  row,
  hostMap,
}: {
  row: RowData;
  hostMap: Map<string, RowData>;
}) {
  const summaryItems: Array<{
    label: string;
    value: unknown;
    valueKey?: string;
  }> = [
    { label: "仓库", value: row.repository_url },
    { label: "路径", value: row.repository_path ?? row.compose_path },
    { label: "分支", value: row.repository_branch },
    { label: "镜像", value: row.image },
    {
      label: "模板类型",
      value: row.template_type,
      valueKey: "template_type",
    },
    { label: "更新时间", value: formatDateTime(row.updated_at) },
  ];
  const isComposeTemplate = isComposeTemplateRow(row);
  const portEntries = templatePortEntries(row);

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <Card
        size="small"
        style={{
          borderRadius: 14,
          borderColor: "#e8edf5",
          boxShadow: "0 10px 24px rgba(15, 23, 42, 0.04)",
        }}
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <div>
            <Typography.Text type="secondary">关联主机</Typography.Text>
            <div style={{ marginTop: 6 }}>
              {renderTemplateHost(row.host_id, hostMap)}
            </div>
          </div>
          <Descriptions
            column={2}
            size="small"
            colon={false}
            labelStyle={{ color: "#6b7280" }}
          >
            {summaryItems.map((item) => (
              <Descriptions.Item key={item.label} label={item.label}>
                {item.value ? renderValue(item.value, item.valueKey) : "-"}
              </Descriptions.Item>
            ))}
          </Descriptions>
          {row.description ? (
            <div
              style={{
                padding: "12px 14px",
                borderRadius: 10,
                background: "#f8fafc",
                border: "1px solid #eef2f7",
              }}
            >
              <Typography.Paragraph style={{ marginBottom: 0 }}>
                {String(row.description)}
              </Typography.Paragraph>
            </div>
          ) : null}
        </Space>
      </Card>
      {isComposeTemplate ? (
        <Card
          size="small"
          title="端口映射"
          style={{
            borderRadius: 14,
            borderColor: "#e8edf5",
            boxShadow: "0 10px 24px rgba(15, 23, 42, 0.04)",
          }}
        >
          <Space wrap size={[8, 8]}>
            {portEntries.length ? (
              portEntries.map((entry) => (
                <Tag key={entry.key} color="blue">
                  {entry.label}
                </Tag>
              ))
            ) : (
              <Typography.Text type="secondary">暂无端口映射</Typography.Text>
            )}
          </Space>
        </Card>
      ) : (
        <Card
          size="small"
          title="镜像信息"
          style={{
            borderRadius: 14,
            borderColor: "#e8edf5",
            boxShadow: "0 10px 24px rgba(15, 23, 42, 0.04)",
          }}
        >
          <Space direction="vertical" size={8}>
            <Typography.Text>{String(row.image ?? "-")}</Typography.Text>
            <Typography.Text type="secondary">
              该模板按单镜像方式处理，不展示 Compose 服务拆分。
            </Typography.Text>
          </Space>
        </Card>
      )}
    </Space>
  );
}

function TemplateServicesTable({ row }: { row: RowData }) {
  if (!isComposeTemplateRow(row)) {
    return (
      <Card
        size="small"
        title="运行方式"
        style={{
          borderRadius: 14,
          borderColor: "#e8edf5",
          boxShadow: "0 10px 24px rgba(15, 23, 42, 0.04)",
        }}
      >
        <Typography.Text type="secondary">
          当前模板按单镜像方式处理。服务、端口和参数不会按 Compose
          多服务结构展开。
        </Typography.Text>
      </Card>
    );
  }
  const services = useMemo(() => templateServices(row), [row]);
  if (!services.length) {
    return (
      <div className="template-detail-empty">
        <Typography.Text strong>暂无服务明细</Typography.Text>
        <Typography.Text type="secondary">
          当前模板未提供 Compose 服务配置，可查看上方镜像、仓库和路径信息。
        </Typography.Text>
      </div>
    );
  }
  return (
    <Space direction="vertical" size={12} style={{ width: "100%" }}>
      <Typography.Text strong>服务明细</Typography.Text>
      {services.map((service) => (
        <Card
          key={String(service.name ?? service.image)}
          size="small"
          style={{
            borderRadius: 14,
            borderColor: "#e8edf5",
            boxShadow: "0 10px 24px rgba(15, 23, 42, 0.04)",
          }}
        >
          <Space direction="vertical" size={12} style={{ width: "100%" }}>
            <Space wrap size={[8, 8]}>
              <Typography.Text strong>
                {String(service.name ?? "-")}
              </Typography.Text>
              {service.image ? <Tag>{String(service.image)}</Tag> : null}
              {normalizePortMappings(service.ports).map((port) => (
                <Tag key={`${service.name}-${port}`} color="blue">
                  {port}
                </Tag>
              ))}
            </Space>
            <Descriptions
              column={1}
              size="small"
              colon={false}
              labelStyle={{ color: "#6b7280", width: 88 }}
            >
              <Descriptions.Item label="依赖">
                {renderValue(toStringArray(service.depends_on), "tags")}
              </Descriptions.Item>
              <Descriptions.Item label="资源">
                {renderServiceResources(service)}
              </Descriptions.Item>
              <Descriptions.Item label="参数">
                {renderServiceParameters(service)}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        </Card>
      ))}
    </Space>
  );
}

function renderTemplateHost(
  value: unknown,
  hostMap: Map<string, RowData>,
): ReactNode {
  const host = hostMap.get(String(value ?? ""));
  if (!host) {
    return (
      <Typography.Text type="secondary">
        {value ? `#${value}` : "未关联"}
      </Typography.Text>
    );
  }
  return (
    <Space size={6}>
      {renderHostIcon(host)}
      <span>{String(host.name ?? host.id)}</span>
      {renderValue(host.host_type, "host_type")}
    </Space>
  );
}

function renderTemplateServicesSummary(row: RowData) {
  const services = templateServices(row);
  if (!services.length) {
    return renderValue(row.image, "image");
  }
  return (
    <Space direction="vertical" size={2}>
      {services.slice(0, 2).map((service) => (
        <Typography.Text key={String(service.name ?? service.image)}>
          {String(service.name ?? "service")}：
          <Typography.Text type="secondary">
            {String(service.image ?? row.image ?? "-")}
          </Typography.Text>
        </Typography.Text>
      ))}
    </Space>
  );
}

function renderTemplatePorts(row: RowData) {
  const entries = templatePortEntries(row);
  if (!entries.length) {
    return renderValue(row.ports, "ports");
  }
  return renderExposedPorts(entries.map((entry) => entry.label));
}

function renderExposedPorts(value: unknown) {
  const ports = toStringArray(value);
  if (!ports.length) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  return (
    <Space direction="vertical" size={2}>
      {ports.slice(0, 3).map((port) => (
        <Typography.Text key={port}>{port}</Typography.Text>
      ))}
      {ports.length > 3 ? (
        <Typography.Text type="secondary">+{ports.length - 3}</Typography.Text>
      ) : null}
    </Space>
  );
}

function renderServiceResources(service: RowData) {
  const items = [
    service.cpu_limit ? `CPU ${service.cpu_limit}` : "",
    service.memory_limit ? `内存 ${service.memory_limit}` : "",
    service.replicas ? `${service.replicas} 副本` : "",
  ].filter(Boolean);
  if (!items.length) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  return (
    <Space wrap size={4}>
      {items.map((item) => (
        <Tag key={item}>{item}</Tag>
      ))}
    </Space>
  );
}

function renderServiceParameters(service: RowData) {
  const items = [
    ...toStringArray(service.command).map((value) => `command: ${value}`),
    ...toStringArray(service.environment).map((value) => `env: ${value}`),
    ...toStringArray(service.volumes).map((value) => `volume: ${value}`),
    ...toStringArray(service.depends_on).map((value) => `depends_on: ${value}`),
    ...toStringArray(service.expose).map((value) => `expose: ${value}`),
    ...toStringArray(service.networks).map((value) => `network: ${value}`),
  ].filter(Boolean);
  if (!items.length) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  return (
    <Space wrap size={[4, 4]}>
      {items.map((item) => (
        <Tag key={item}>{item}</Tag>
      ))}
    </Space>
  );
}

function templatePortEntries(row: RowData) {
  const portMap = normalizeTemplatePortMap(row);
  const serviceNames = Object.keys(portMap);
  if (!serviceNames.length) {
    return [];
  }
  const isMultiService = serviceNames.length > 1;
  return serviceNames.flatMap((serviceName) =>
    portMap[serviceName].map((port) => ({
      key: `${serviceName}-${port}`,
      label: isMultiService ? `${serviceName} / ${port}` : port,
    })),
  );
}

function templateServices(row: RowData): RowData[] {
  if (!isComposeTemplateRow(row)) {
    return [];
  }
  const parsed = parseComposeServices(String(row.compose_file ?? ""));
  if (!Array.isArray(row.services) || row.services.length === 0) {
    return parsed;
  }
  const existing = row.services as RowData[];
  const parsedByName = new Map(
    parsed.map((service) => [String(service.name ?? ""), service]),
  );
  const merged = existing.map((service) => {
    const key = String(service.name ?? "");
    const fallback = parsedByName.get(key) ?? {};
    parsedByName.delete(key);
    return { ...fallback, ...service };
  });
  return [...merged, ...parsedByName.values()];
}

function parseComposeServices(content: string): RowData[] {
  const lines = content.split(/\r?\n/);
  const services: RowData[] = [];
  let inServices = false;
  let current: RowData | null = null;
  let listKey: string | null = null;

  for (const raw of lines) {
    const indent = raw.search(/\S|$/);
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    if (!inServices && line === "services:") {
      inServices = true;
      current = null;
      listKey = null;
      continue;
    }
    if (inServices && indent === 0 && line !== "services:") {
      break;
    }
    if (!inServices) continue;
    if (indent <= 1 && line.endsWith(":") && line !== "services:") {
      current = { name: line.slice(0, -1).trim() };
      services.push(current);
      listKey = null;
      continue;
    }
    if (indent === 0) {
      current = null;
      listKey = null;
      continue;
    }
    if (!current) continue;
    if (indent >= 3 && line.endsWith(":")) {
      listKey = line.slice(0, -1).trim();
      current[listKey] = [];
      continue;
    }
    if (indent >= 3 && line.includes(":")) {
      const [key, ...rest] = line.split(":");
      current[key.trim()] = rest
        .join(":")
        .trim()
        .replace(/^["']|["']$/g, "");
      listKey = null;
      continue;
    }
    if (indent >= 6 && listKey && line.startsWith("- ")) {
      const list = Array.isArray(current[listKey]) ? current[listKey] : [];
      list.push(
        line
          .slice(2)
          .trim()
          .replace(/^["']|["']$/g, ""),
      );
      current[listKey] = list;
    }
  }
  return services;
}

function normalizePortMappings(value: unknown): string[] {
  if (!value) {
    return [];
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => normalizePortMappings(item));
  }
  if (typeof value === "string" || typeof value === "number") {
    const text = String(value).trim().replace(/^["']|["']$/g, "");
    return text ? [text] : [];
  }
  if (typeof value !== "object") {
    return [];
  }
  return Object.entries(value as Record<string, unknown>).flatMap(
    ([containerPort, published]) => {
      if (Array.isArray(published)) {
        return published.flatMap((binding) => {
          if (!binding || typeof binding !== "object") {
            return [];
          }
          const row = binding as RowData;
          const hostPort = String(
            row.HostPort ?? row.host_port ?? row.hostPort ?? "",
          ).trim();
          return hostPort ? [`${hostPort}:${containerPort}`] : [containerPort];
        });
      }
      if (published && typeof published === "object") {
        const row = published as RowData;
        const hostPort = String(
          row.HostPort ?? row.host_port ?? row.hostPort ?? row.published ?? "",
        ).trim();
        return hostPort ? [`${hostPort}:${containerPort}`] : [containerPort];
      }
      const hostPort = String(published ?? "").trim();
      return hostPort ? [`${hostPort}:${containerPort}`] : [containerPort];
    },
  );
}

function normalizeExposedPorts(value: unknown): string[] {
  return toStringArray(value)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const clean = item.replace(/^["']|["']$/g, "");
      if (!clean.includes(":")) {
        return clean;
      }
      const segments = clean.split(":").map((segment) => segment.trim());
      return segments[segments.length - 1] || clean;
    });
}

function normalizeTemplatePortMap(row: RowData): Record<string, string[]> {
  const rawPorts = row.ports;
  if (rawPorts && typeof rawPorts === "object" && !Array.isArray(rawPorts)) {
    const portMap = Object.fromEntries(
      Object.entries(rawPorts as Record<string, unknown>)
        .map(([serviceName, ports]) => [
          serviceName,
          normalizeExposedPorts(ports),
        ])
        .filter(([, ports]) => Array.isArray(ports) && ports.length > 0),
    );
    if (Object.keys(portMap).length > 0) {
      return portMap;
    }
  }

  const services = templateServices(row);
  if (services.length > 0) {
    const servicePortMap = Object.fromEntries(
      services
        .map((service) => [
          String(service.name ?? "service"),
          normalizeExposedPorts(service.ports),
        ])
        .filter(([, ports]) => Array.isArray(ports) && ports.length > 0),
    );
    if (Object.keys(servicePortMap).length > 0) {
      return servicePortMap;
    }
  }

  const fallbackPorts = normalizeExposedPorts(rawPorts);
  if (!fallbackPorts.length) {
    return {};
  }
  return {
    [String(row.name ?? "app")]: fallbackPorts,
  };
}

function isComposeTemplateRow(row: RowData) {
  if (Array.isArray(row.services) && row.services.length > 0) {
    return true;
  }
  const compose = String(row.compose_file ?? "").trim();
  if (!compose) {
    return false;
  }
  return compose.includes("\nservices:") || compose.startsWith("services:");
}

function toStringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => String(item)).filter(Boolean);
  }
  if (value && typeof value === "object") {
    return Object.entries(value as Record<string, unknown>).map(([key, val]) =>
      val === true ? key : `${key}:${String(val)}`,
    );
  }
  if (typeof value === "string" && value.trim()) {
    return value.includes(",")
      ? value
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean)
      : [value.trim()];
  }
  return [];
}
