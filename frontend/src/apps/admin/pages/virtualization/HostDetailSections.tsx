import { CodeOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, EyeOutlined, FileZipOutlined, InfoCircleOutlined, PauseCircleOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, StopOutlined, SyncOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Descriptions, Drawer, Dropdown, Form, Input, InputNumber, Modal, Row, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import type { RowData, SortOrder } from "./types";
import { CreateContainerModal, ImageBuildModal, ImagePullModal, ImageUploadModal } from "./HostModals";
import { statusColor } from "./constants";
import { formatBytes, formatDateTime, hostErrorText, imageDisplayName, imageShortId, normalizeRows, renderPortList, renderPortSummary, splitImageName, normalizeTimestamp } from "./utils";
export function HostOverview({
  host,
  onOpenImages,
  onOpenContainers,
}: {
  host: RowData;
  onOpenImages: () => void;
  onOpenContainers: () => void;
}) {
  const resources = host.resources ?? {};
  const stats = [
    {
      label: "CPU 核心",
      value: resources.cpu_count ?? "-",
      helper: resources.os || resources.os_type || "-",
    },
    {
      label: "内存",
      value: formatBytes(resources.memory_total),
      helper: resources.docker_version || "-",
    },
    {
      label: "容器",
      value: resources.containers_total ?? 0,
      helper: `运行中 ${resources.containers_running ?? 0}`,
      onClick: onOpenContainers,
    },
    {
      label: "镜像",
      value: resources.images_count ?? 0,
      helper: "进入镜像管理",
      onClick: onOpenImages,
    },
  ];

  return (
    <>
      {host.status !== "online" ? (
        <Alert
          type="warning"
          showIcon
          message="当前主机未在线"
          description={hostErrorText(host)}
        />
      ) : null}

      <Row gutter={[12, 12]}>
        {stats.map((stat) => (
          <Col xs={24} md={12} xl={6} key={stat.label}>
            <button
              type="button"
              className={`virtualization-detail-stat ${stat.onClick ? "interactive" : ""}`}
              onClick={stat.onClick}
            >
              <span>{stat.label}</span>
              <strong>{String(stat.value ?? "-")}</strong>
              <small>{stat.helper}</small>
            </button>
          </Col>
        ))}
      </Row>

      <Card className="admin-table-card" bordered={false}>
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="主机名称">
            {host.name ?? "-"}
          </Descriptions.Item>
          <Descriptions.Item label="主机类型">
            {host.host_type === "pve" ? "Proxmox VE" : "Docker"}
          </Descriptions.Item>
          <Descriptions.Item label="连接地址" span={2}>
            <Typography.Text copyable>
              {String(host.endpoint ?? "-")}
            </Typography.Text>
          </Descriptions.Item>
          <Descriptions.Item label="出口 IP">
            {host.out_ip || "-"}
          </Descriptions.Item>
          <Descriptions.Item label="最后检查">
            {host.last_check
              ? new Date(String(host.last_check)).toLocaleString()
              : "-"}
          </Descriptions.Item>
          <Descriptions.Item label="架构">
            {host.resources?.architecture || "-"}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {host.created_at
              ? new Date(String(host.created_at)).toLocaleString()
              : "-"}
          </Descriptions.Item>
          <Descriptions.Item label="描述" span={2}>
            {host.description || "-"}
          </Descriptions.Item>
          {host.resources?.error ? (
            <Descriptions.Item label="错误信息" span={2}>
              <Typography.Text type="danger">
                {String(host.resources.error)}
              </Typography.Text>
            </Descriptions.Item>
          ) : null}
        </Descriptions>
      </Card>
    </>
  );
}

export function ImageManagementSection({
  host,
  images,
  loading,
  onRefresh,
  onCreateContainer,
}: {
  host: RowData;
  images: RowData[];
  loading: boolean;
  onRefresh: () => void;
  onCreateContainer: (imageName: string) => void;
}) {
  const [keyword, setKeyword] = useState("");
  const [pullOpen, setPullOpen] = useState(false);
  const [buildOpen, setBuildOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [detail, setDetail] = useState<RowData | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });
  const [sortKey, setSortKey] = useState<"image" | "size" | "created">(
    "created",
  );
  const [sortOrder, setSortOrder] = useState<SortOrder>("descend");
  const filtered = images.filter((image) => {
    const fullName = imageDisplayName(image).toLowerCase();
    const shortId = imageShortId(image).toLowerCase();
    const q = keyword.trim().toLowerCase();
    if (!q) return true;
    return (
      fullName.includes(q) ||
      shortId.includes(q) ||
      String(image.id ?? "")
        .toLowerCase()
        .includes(q)
    );
  });
  const sorted = [...filtered].sort((left, right) => {
    let result = 0;
    if (sortKey === "image") {
      result = imageDisplayName(left).localeCompare(imageDisplayName(right));
    } else if (sortKey === "size") {
      result = Number(left.size ?? 0) - Number(right.size ?? 0);
    } else {
      result =
        normalizeTimestamp(left.created) - normalizeTimestamp(right.created);
    }
    return sortOrder === "ascend" ? result : -result;
  });

  return (
    <>
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Space wrap>
            <Input.Search
              allowClear
              placeholder="搜索镜像名称、标签或 ID"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              style={{ width: 280 }}
            />
            <Button icon={<ReloadOutlined />} onClick={onRefresh}>
              刷新
            </Button>
          </Space>
          <Space wrap>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => setPullOpen(true)}
            >
              拉取镜像
            </Button>
            <Button icon={<CodeOutlined />} onClick={() => setBuildOpen(true)}>
              构建镜像
            </Button>
            <Button
              type="primary"
              icon={<FileZipOutlined />}
              onClick={() => setUploadOpen(true)}
            >
              上传镜像
            </Button>
          </Space>
        </div>
        <Table
          className="virtualization-managed-table"
          rowKey={(row) =>
            String(row.id ?? row.image_id ?? imageDisplayName(row))
          }
          loading={loading}
          dataSource={sorted}
          onRow={(record) => ({
            onClick: () => setDetail(record),
            style: { cursor: "pointer" },
          })}
          columns={[
            {
              title: "IMAGE ID",
              key: "image_id",
              width: 140,
              sorter: (left, right) =>
                imageShortId(left).localeCompare(imageShortId(right)),
              render: (_, row) => (
                <Typography.Text code copyable={{ text: String(row.id ?? "") }}>
                  {imageShortId(row) || "-"}
                </Typography.Text>
              ),
            },
            {
              title: "镜像",
              key: "image",
              sorter: (left, right) =>
                imageDisplayName(left).localeCompare(imageDisplayName(right)),
              sortOrder: sortKey === "image" ? sortOrder : null,
              render: (_, row) => {
                const parts = splitImageName(imageDisplayName(row));
                return (
                  <Space size={6} wrap>
                    <Typography.Text>{parts.repository}</Typography.Text>
                    <Tag color="blue">{parts.tag}</Tag>
                  </Space>
                );
              },
            },
            {
              title: "大小",
              dataIndex: "size",
              key: "size",
              width: 120,
              sorter: (left, right) =>
                Number(left.size ?? 0) - Number(right.size ?? 0),
              sortOrder: sortKey === "size" ? sortOrder : null,
              render: (value) => formatBytes(value),
            },
            {
              title: "创建时间",
              dataIndex: "created",
              key: "created",
              width: 180,
              sorter: (left, right) =>
                normalizeTimestamp(left.created) -
                normalizeTimestamp(right.created),
              sortOrder: sortKey === "created" ? sortOrder : null,
              render: (value) => {
                const text = formatDateTime(value);
                return text === "-" ? (
                  <Typography.Text type="secondary">-</Typography.Text>
                ) : (
                  <Typography.Text>{text}</Typography.Text>
                );
              },
            },
            {
              title: "操作",
              key: "actions",
              width: 148,
              render: (_, row) => (
                <Space>
                  <Tooltip title="查看详情">
                    <Button
                      size="small"
                      icon={<InfoCircleOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        setDetail(row);
                      }}
                    />
                  </Tooltip>
                  <Tooltip title="从镜像创建容器">
                    <Button
                      size="small"
                      type="primary"
                      icon={<PlayCircleOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        onCreateContainer(imageDisplayName(row));
                      }}
                    />
                  </Tooltip>
                </Space>
              ),
            },
          ]}
          onChange={(_, __, sorter) => {
            const nextSorter = Array.isArray(sorter) ? sorter[0] : sorter;
            if (!nextSorter?.field && !nextSorter?.columnKey) {
              return;
            }
            const key = String(nextSorter.columnKey ?? nextSorter.field);
            if (key === "size" || key === "created") {
              setSortKey(key);
            } else {
              setSortKey("image");
            }
            setSortOrder((nextSorter.order as SortOrder) ?? "ascend");
          }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: sorted.length,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
            showTotal: (total) => `共 ${total} 项`,
            onChange: (current, pageSize) => {
              setPagination({ current, pageSize });
            },
          }}
          scroll={{ y: "calc(100vh - 360px)" }}
        />
      </Card>

      <ImagePullModal
        open={pullOpen}
        host={host}
        onCancel={() => setPullOpen(false)}
        onSuccess={async () => {
          setPullOpen(false);
          onRefresh();
        }}
      />
      <ImageBuildModal
        open={buildOpen}
        host={host}
        onCancel={() => setBuildOpen(false)}
        onSuccess={async () => {
          setBuildOpen(false);
          onRefresh();
        }}
      />
      <ImageUploadModal
        open={uploadOpen}
        host={host}
        onCancel={() => setUploadOpen(false)}
        onSuccess={async () => {
          setUploadOpen(false);
          onRefresh();
        }}
      />

      <Drawer
        title={detail ? imageDisplayName(detail) : "镜像详情"}
        open={Boolean(detail)}
        onClose={() => setDetail(null)}
        width={540}
      >
        {detail ? (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="镜像">
              {imageDisplayName(detail)}
            </Descriptions.Item>
            <Descriptions.Item label="IMAGE ID">
              <Typography.Text copyable>
                {String(detail.id ?? "-")}
              </Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="简短 ID">
              {imageShortId(detail) || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="大小">
              {formatBytes(detail.size)}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDateTime(detail.created)}
            </Descriptions.Item>
          </Descriptions>
        ) : null}
      </Drawer>
    </>
  );
}

export function ContainerManagementSection({
  host,
  containers,
  loading,
  onRefresh,
  onCreate,
}: {
  host: RowData;
  containers: RowData[];
  loading: boolean;
  onRefresh: () => void;
  onCreate: () => void;
}) {
  const [modal, contextHolder] = Modal.useModal();
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState<string>();
  const [detail, setDetail] = useState<RowData | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const filtered = containers.filter((container) => {
    const q = keyword.trim().toLowerCase();
    const matchesKeyword =
      !q ||
      String(container.name ?? "")
        .toLowerCase()
        .includes(q) ||
      String(container.image ?? "")
        .toLowerCase()
        .includes(q) ||
      String(container.id ?? container.container_id ?? "")
        .toLowerCase()
        .includes(q);
    const matchesStatus =
      !status || String(container.status ?? "").toLowerCase() === status;
    return matchesKeyword && matchesStatus;
  });

  async function runAction(
    action: "start" | "stop" | "restart" | "pause" | "unpause" | "delete",
    row: RowData,
  ) {
    const runtimeId = String(row.container_id ?? row.id ?? row.name ?? "");
    if (!runtimeId) {
      message.error("缺少容器标识");
      return;
    }
    const labelMap = {
      start: "启动",
      stop: "停止",
      restart: "重启",
      pause: "暂停",
      unpause: "恢复",
      delete: "删除",
    };
    const danger = action === "delete";
    modal.confirm({
      title: `${labelMap[action]}容器`,
      content: `确认${labelMap[action]}容器 ${row.name ?? runtimeId}？`,
      okButtonProps: danger ? { danger: true } : undefined,
      onOk: async () => {
        if (action === "delete") {
          await adminApi.remove(
            `/api-admin/virtualization/containers/${runtimeId}`,
          );
        } else {
          await adminApi.action(
            "/api-admin/virtualization/containers/action",
            "POST",
            {
              host_id: host.id,
              container_id: runtimeId,
              action,
            },
          );
        }
        message.success(`${labelMap[action]}成功`);
        await onRefresh();
      },
    });
  }

  async function runBatchAction(
    action: "start" | "stop" | "restart" | "pause" | "unpause" | "delete",
  ) {
    if (!selectedRowKeys.length) {
      message.warning("请先选择容器");
      return;
    }
    const labelMap = {
      start: "启动",
      stop: "停止",
      restart: "重启",
      pause: "暂停",
      unpause: "恢复",
      delete: "删除",
    };
    modal.confirm({
      title: `批量${labelMap[action]}容器`,
      content: `确认${labelMap[action]} ${selectedRowKeys.length} 个容器？`,
      okButtonProps: action === "delete" ? { danger: true } : undefined,
      onOk: async () => {
        await adminApi.action(
          "/api-admin/virtualization/containers/batch-action",
          "POST",
          {
            host_id: host.id,
            container_ids: selectedRowKeys,
            action,
          },
        );
        message.success(`批量${labelMap[action]}成功`);
        setSelectedRowKeys([]);
        await onRefresh();
      },
    });
  }

  return (
    <>
      {contextHolder}
      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Space wrap>
            <Input.Search
              allowClear
              placeholder="搜索容器名称、镜像或 ID"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              style={{ width: 280 }}
            />
            <Select
              allowClear
              placeholder="状态"
              value={status}
              onChange={setStatus}
              options={[
                "running",
                "stopped",
                "paused",
                "exited",
                "creating",
              ].map((value) => ({
                value,
                label: value,
              }))}
              style={{ width: 140 }}
            />
            <Button icon={<ReloadOutlined />} onClick={onRefresh}>
              刷新
            </Button>
          </Space>
          <Space wrap>
            {selectedRowKeys.length ? (
              <Dropdown
                menu={{
                  items: [
                    { key: "start", label: "批量启动" },
                    { key: "stop", label: "批量停止" },
                    { key: "restart", label: "批量重启" },
                    { key: "pause", label: "批量暂停" },
                    { key: "unpause", label: "批量恢复" },
                    { type: "divider" as const },
                    { key: "delete", danger: true, label: "批量删除" },
                  ],
                  onClick: ({ key }) => {
                    if (
                      key === "start" ||
                      key === "stop" ||
                      key === "restart" ||
                      key === "pause" ||
                      key === "unpause" ||
                      key === "delete"
                    ) {
                      void runBatchAction(key);
                    }
                  },
                }}
              >
                <Button>批量操作 ({selectedRowKeys.length})</Button>
              </Dropdown>
            ) : null}
            <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
              创建容器
            </Button>
          </Space>
        </div>
        <Table
          className="virtualization-managed-table"
          rowKey={(row) => String(row.container_id ?? row.id ?? row.name)}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys),
          }}
          loading={loading}
          dataSource={filtered}
          onRow={(record) => ({
            onClick: () => setDetail(record),
            style: { cursor: "pointer" },
          })}
          columns={[
            {
              title: "容器",
              render: (_, row) => (
                <Space direction="vertical" size={0}>
                  <Typography.Text strong>
                    {String(row.name ?? "-")}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    {String(row.id ?? row.container_id ?? "-")}
                  </Typography.Text>
                </Space>
              ),
            },
            {
              title: "镜像",
              dataIndex: "image",
              render: (value) => (
                <Typography.Text>{String(value ?? "-")}</Typography.Text>
              ),
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 120,
              render: (value, row) => (
                <Tooltip title={row.error ? String(row.error) : undefined}>
                  <Tag color={statusColor[String(value)]}>
                    {String(value ?? "-")}
                  </Tag>
                </Tooltip>
              ),
            },
            {
              title: "端口",
              dataIndex: "ports",
              render: (value) => renderPortSummary(value),
            },
            {
              title: "创建时间",
              dataIndex: "created",
              width: 180,
              render: (value) => {
                const text = formatDateTime(value);
                return text === "-" ? (
                  <Typography.Text type="secondary">-</Typography.Text>
                ) : (
                  <Typography.Text>{text}</Typography.Text>
                );
              },
            },
            {
              title: "操作",
              key: "actions",
              width: 188,
              render: (_, row) => (
                <Space>
                  <Tooltip title="查看详情">
                    <Button
                      size="small"
                      icon={<EyeOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        setDetail(row);
                      }}
                    />
                  </Tooltip>
                  {String(row.status ?? "").toLowerCase() === "running" ? (
                    <>
                      <Tooltip title="暂停">
                        <Button
                          size="small"
                          icon={<PauseCircleOutlined />}
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction("pause", row);
                          }}
                        />
                      </Tooltip>
                      <Tooltip title="停止">
                        <Button
                          size="small"
                          danger
                          icon={<StopOutlined />}
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction("stop", row);
                          }}
                        />
                      </Tooltip>
                      <Tooltip title="重启">
                        <Button
                          size="small"
                          icon={<SyncOutlined />}
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction("restart", row);
                          }}
                        />
                      </Tooltip>
                    </>
                  ) : String(row.status ?? "").toLowerCase() === "paused" ? (
                    <>
                      <Tooltip title="恢复">
                        <Button
                          size="small"
                          type="primary"
                          icon={<PlayCircleOutlined />}
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction("unpause", row);
                          }}
                        />
                      </Tooltip>
                      <Tooltip title="停止">
                        <Button
                          size="small"
                          danger
                          icon={<StopOutlined />}
                          onClick={(event) => {
                            event.stopPropagation();
                            void runAction("stop", row);
                          }}
                        />
                      </Tooltip>
                    </>
                  ) : (
                    <Tooltip title="启动">
                      <Button
                        size="small"
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        onClick={(event) => {
                          event.stopPropagation();
                          void runAction("start", row);
                        }}
                      />
                    </Tooltip>
                  )}
                  <Tooltip title="删除">
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(event) => {
                        event.stopPropagation();
                        void runAction("delete", row);
                      }}
                    />
                  </Tooltip>
                </Space>
              ),
            },
          ]}
          scroll={{ y: "calc(100vh - 360px)" }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: filtered.length,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
            showTotal: (total) => `共 ${total} 项`,
            onChange: (current, pageSize) => {
              setPagination({ current, pageSize });
            },
          }}
        />
      </Card>

      <Drawer
        title={detail?.name ? `容器详情: ${detail.name}` : "容器详情"}
        open={Boolean(detail)}
        onClose={() => setDetail(null)}
        width={560}
      >
        {detail ? (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="容器名称">
              {String(detail.name ?? "-")}
            </Descriptions.Item>
            <Descriptions.Item label="容器 ID">
              <Typography.Text copyable>
                {String(detail.id ?? detail.container_id ?? "-")}
              </Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="镜像">
              {String(detail.image ?? "-")}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColor[String(detail.status)]}>
                {String(detail.status ?? "-")}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="端口">
              {renderPortList(detail.ports)}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDateTime(detail.created)}
            </Descriptions.Item>
            {detail.error ? (
              <Descriptions.Item label="错误">
                <Typography.Text type="danger">
                  {String(detail.error)}
                </Typography.Text>
              </Descriptions.Item>
            ) : null}
          </Descriptions>
        ) : null}
      </Drawer>
    </>
  );
}



