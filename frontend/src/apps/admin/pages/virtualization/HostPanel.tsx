import { CloudServerOutlined, DeleteOutlined, EditOutlined, EyeOutlined, InfoCircleOutlined, PlusOutlined, ReloadOutlined, SyncOutlined } from "@ant-design/icons";
import { Alert, Button, Checkbox, Col, Collapse, Form, Input, Modal, Row, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import { hostTypeOptions, statusColor, statusMeta } from "./constants";
import { SummaryCard } from "./SharedPanels";
import { AdminDataTableCard } from "../../components/AdminDataTableCard";
import type { HostDetailMode, RowData } from "./types";
import { formatDateTime, hostErrorText, normalizePayload, normalizeRows, renderArchitecture, renderHostIcon } from "./utils";
export function HostPanel() {
  const navigate = useNavigate();
  const [modal, contextHolder] = Modal.useModal();
  const [hosts, setHosts] = useState<RowData[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [hostType, setHostType] = useState<string>();
  const [status, setStatus] = useState<string>();
  const [editing, setEditing] = useState<RowData | null>(null);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const editingHostType = Form.useWatch("host_type", form) ?? "docker";

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        "/api-admin/virtualization/hosts",
        {
          current: 1,
          pageSize: 100,
          keyword: keyword || undefined,
          host_type: hostType,
          status,
        },
      );
      setHosts(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载主机失败");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  function openHost(record?: RowData) {
    setEditing(record ?? {});
    form.setFieldsValue(record ?? { host_type: "docker", is_default: false, is_active: true });
  }

  function openHostDetail(host: RowData, tab: HostDetailMode = "overview") {
    if (tab === "images") {
      navigate(`/admin/virtualization/${host.id}/images`);
      return;
    }
    if (tab === "containers") {
      navigate(`/admin/virtualization/${host.id}/containers`);
      return;
    }
    navigate(`/admin/virtualization/${host.id}`);
  }

  async function saveHost() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      if (editing?.id) {
        await adminApi.update(
          `/api-admin/virtualization/hosts/${editing.id}`,
          values,
        );
        message.success("主机已更新");
      } else {
        await adminApi.create("/api-admin/virtualization/hosts", values);
        message.success("主机已创建");
      }
      setEditing(null);
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "保存主机失败");
    } finally {
      setSaving(false);
    }
  }

  async function checkHost(host: RowData) {
    try {
      const result = await adminApi.list<RowData>(
        `/api-admin/virtualization/hosts/${host.id}/check`,
        {
          force_refresh: true,
        },
      );
      message.success(
        `检查完成：${String((result as RowData).status ?? "unknown")}`,
      );
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "检查主机失败");
    }
  }

  function deleteHost(host: RowData) {
    modal.confirm({
      title: `删除主机 ${host.name ?? host.id}`,
      content: "删除后不可恢复，请确认该主机上没有仍需保留的容器或模板。",
      okButtonProps: { danger: true },
      onOk: async () => {
        await adminApi.remove(`/api-admin/virtualization/hosts/${host.id}`);
        message.success("主机已删除");
        await load();
      },
    });
  }

  return (
    <Space className="virtualization-stack" direction="vertical" size={16}>
      {contextHolder}
      <Row gutter={[12, 12]}>
        <Col xs={12} lg={6}>
          <SummaryCard label="主机总数" value={hosts.length} />
        </Col>
        <Col xs={12} lg={6}>
          <SummaryCard
            label="在线"
            value={hosts.filter((host) => host.status === "online").length}
            tone="success"
          />
        </Col>
        <Col xs={12} lg={6}>
          <SummaryCard
            label="离线"
            value={hosts.filter((host) => host.status === "offline").length}
            tone="muted"
          />
        </Col>
        <Col xs={12} lg={6}>
          <SummaryCard
            label="异常"
            value={hosts.filter((host) => host.status === "error").length}
            tone="danger"
          />
        </Col>
      </Row>

      <AdminDataTableCard
        title="主机列表"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>关键词：</label>
              <Input.Search
                allowClear
                placeholder="搜索主机名称或连接地址"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                onSearch={() => load()}
              />
            </div>
            <div className="admin-search-field">
              <label>类型：</label>
              <Select
                allowClear
                placeholder="主机类型"
                value={hostType}
                onChange={setHostType}
                options={hostTypeOptions}
              />
            </div>
            <div className="admin-search-field">
              <label>状态：</label>
              <Select
                allowClear
                placeholder="状态"
                value={status}
                onChange={setStatus}
                options={["online", "offline", "error"].map((value) => ({
                  label: value,
                  value,
                }))}
              />
            </div>
          </div>
        }
        actions={
          <>
            <Button icon={<ReloadOutlined />} onClick={() => load()}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openHost()}
            >
              新建主机
            </Button>
          </>
        }
      >
        <Table
          rowKey={(host) => String(host.id)}
          loading={loading}
          dataSource={hosts}
          onRow={(host) => ({
            onClick: () => openHostDetail(host, "overview"),
            style: { cursor: "pointer" },
          })}
          columns={[
            {
              title: "主机",
              dataIndex: "name",
              width: 240,
              render: (_, host) => (
                <Space>
                  <span className="virtualization-host-icon compact">
                    {renderHostIcon(host)}
                  </span>
                  <Space direction="vertical" size={0}>
                    <Typography.Text strong>
                      {host.name ?? `主机 ${host.id}`}
                    </Typography.Text>
                    <Space size={6} wrap>
                      <Typography.Text type="secondary">
                        # {host.id}
                      </Typography.Text>
                      {host.is_default ? <Tag color="gold">默认</Tag> : null}
                      {renderArchitecture(host)}
                    </Space>
                  </Space>
                </Space>
              ),
            },
            {
              title: "类型",
              dataIndex: "host_type",
              width: 120,
              render: (value) => (
                <Tag color={value === "pve" ? "purple" : "blue"}>
                  {String(value ?? "docker")}
                </Tag>
              ),
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 110,
              render: (value, host) => (
                <Tooltip title={hostErrorText(host)}>
                  <Tag color={statusColor[String(value)]}>
                    {String(value ?? "unknown")}
                  </Tag>
                </Tooltip>
              ),
            },
            {
              title: "连接地址",
              dataIndex: "endpoint",
              ellipsis: true,
              render: (value) => {
                const endpoint = String(value ?? "-");
                let color = "default";
                let label = "unknown";
                if (endpoint.startsWith("tcp://")) {
                  color = "blue";
                  label = "TCP";
                } else if (endpoint.startsWith("unix://")) {
                  color = "geekblue";
                  label = "Unix Socket";
                } else if (endpoint.startsWith("npipe://")) {
                  color = "purple";
                  label = "Named Pipe";
                }
                return (
                  <Space size={8}>
                    <Tag color={color}>{label}</Tag>
                    <Typography.Text copyable>{endpoint}</Typography.Text>
                  </Space>
                );
              },
            },
            {
              title: "资源",
              width: 280,
              render: (_, host) => {
                const resources = host.resources ?? {};
                const hasCpuUsage =
                  typeof resources.cpu_usage === "number" &&
                  Number(resources.cpu_usage) > 0;
                if (host.status !== "online") {
                  return (
                    <Tooltip title={hostErrorText(host)}>
                      <Space
                        className="host-resource-cell"
                        direction="vertical"
                        size={2}
                      >
                        <Typography.Text type="secondary">
                          {host.status === "error"
                            ? "连接失败，无法获取 Docker 资源"
                            : "主机离线，暂无资源数据"}
                        </Typography.Text>
                      </Space>
                    </Tooltip>
                  );
                }
                return (
                  <Space
                    className="host-resource-cell"
                    direction="vertical"
                    size={6}
                  >
                    {hasCpuUsage ? (
                      <Typography.Text>
                        CPU {Math.round(Number(resources.cpu_usage))}%
                      </Typography.Text>
                    ) : null}
                    <Space size={8} wrap>
                      <button
                        type="button"
                        className="host-metric-card"
                        onClick={(event) => {
                          event.stopPropagation();
                          openHostDetail(host, "containers");
                        }}
                      >
                        <strong>{resources.containers_total ?? 0}</strong>
                        <span>容器</span>
                      </button>
                      <button
                        type="button"
                        className="host-metric-card"
                        onClick={(event) => {
                          event.stopPropagation();
                          openHostDetail(host, "images");
                        }}
                      >
                        <strong>{resources.images_count ?? 0}</strong>
                        <span>镜像</span>
                      </button>
                    </Space>
                  </Space>
                );
              },
            },
            {
              title: "最后检查",
              dataIndex: "last_check",
              width: 180,
              render: (value, host) =>
                value ? (
                  <Space direction="vertical" size={0}>
                    <Typography.Text>
                      {new Date(String(value)).toLocaleString()}
                    </Typography.Text>
                    {host.status === "error" ? (
                      <Typography.Text type="danger">
                        最近检查失败
                      </Typography.Text>
                    ) : null}
                  </Space>
                ) : (
                  <Typography.Text type="secondary">-</Typography.Text>
                ),
            },
            {
              title: "操作",
              key: "actions",
              fixed: "right",
              width: 132,
              render: (_, host) => (
                <Space>
                  <Tooltip title="检查连接">
                    <Button
                      size="small"
                      icon={<SyncOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        void checkHost(host);
                      }}
                    />
                  </Tooltip>
                  <Tooltip title="编辑主机">
                    <Button
                      size="small"
                      icon={<EditOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        openHost(host);
                      }}
                    />
                  </Tooltip>
                  <Tooltip title="删除主机">
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        deleteHost(host);
                      }}
                    />
                  </Tooltip>
                </Space>
              ),
            },
          ]}
          scroll={{ x: 1200 }}
          pagination={{ pageSize: 10, showSizeChanger: true }}
        />
      </AdminDataTableCard>

      <Modal
        title={editing?.id ? "编辑主机" : "新建主机"}
        open={Boolean(editing)}
        onCancel={() => setEditing(null)}
        onOk={saveHost}
        confirmLoading={saving}
        width={760}
        destroyOnClose
        styles={{
          body: {
            maxHeight: "70vh",
            overflowY: "auto",
            overflowX: "hidden",
            paddingBottom: 8,
          },
        }}
      >
        <Form form={form} layout="vertical">
          <Row gutter={[12, 0]} style={{ marginLeft: 0, marginRight: 0 }}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="主机名称"
                rules={[{ required: true }]}
              >
                <Input placeholder="例如：Local Docker" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="host_type"
                label="主机类型"
                rules={[{ required: true }]}
              >
                <Select options={hostTypeOptions} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="endpoint"
            label="连接地址"
            rules={[{ required: true }]}
            extra={
              editingHostType === "docker"
                ? "Windows: npipe:////./pipe/docker_engine，WSL/Linux: unix:///var/run/docker.sock，远程: tcp://host:2375"
                : "PVE: https://pve.example.com:8006"
            }
          >
            <Input
              placeholder={
                editingHostType === "docker"
                  ? "例如：tcp://127.0.0.1:2375"
                  : "例如：https://pve.example.com:8006"
              }
            />
          </Form.Item>

          <Row gutter={[12, 0]} style={{ marginLeft: 0, marginRight: 0 }}>
            <Col span={12}>
              <Form.Item name="out_ip" label="出口 IP">
                <Input placeholder="例如：ctf.example.com" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="description" label="描述">
                <Input placeholder="补充主机用途或备注" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="is_default"
            valuePropName="checked"
            tooltip="设为默认后，模板同步等未显式指定主机的流程会优先使用这台同类型主机。"
          >
            <Checkbox>设为默认主机</Checkbox>
          </Form.Item>

          <Collapse
            ghost
            items={[
              {
                key: "advanced",
                label: "高级连接配置",
                children: (
                  <Row
                    gutter={[12, 0]}
                    style={{ marginLeft: 0, marginRight: 0 }}
                  >
                    <Col span={12}>
                      <Form.Item name="username" label="用户名">
                        <Input
                          placeholder={
                            editingHostType === "pve"
                              ? "例如：root@pam"
                              : "Docker 一般留空"
                          }
                        />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name="password" label="密码">
                        <Input.Password
                          placeholder={
                            editingHostType === "pve"
                              ? "请输入密码"
                              : "Docker 一般留空"
                          }
                        />
                      </Form.Item>
                    </Col>
                    <Col span={24}>
                      <Form.Item
                        name="api_key"
                        label={
                          editingHostType === "pve"
                            ? "API Token / Key"
                            : "API Key / TLS 配置"
                        }
                      >
                        <Input.Password
                          placeholder={
                            editingHostType === "pve"
                              ? "可填写 PVE API Token"
                              : "仅远程 Docker 认证场景填写"
                          }
                        />
                      </Form.Item>
                    </Col>
                  </Row>
                ),
              },
            ]}
          />
        </Form>
      </Modal>
    </Space>
  );
}



