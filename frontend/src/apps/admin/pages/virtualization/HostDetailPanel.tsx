import { ApiOutlined, CloudServerOutlined, CodeOutlined, InfoCircleOutlined, ReloadOutlined, SyncOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Descriptions, Row, Space, Tabs, Tag, Tooltip, Typography, message } from "antd";
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import type { HostDetailMode, RowData } from "./types";
import { HostOverview, ImageManagementSection, ContainerManagementSection } from "./HostDetailSections";
import { statusColor } from "./constants";
import { CreateContainerModal } from "./HostModals";
import { formatBytes, formatDateTime, hostErrorText, normalizeRows, renderArchitecture, renderHostIcon } from "./utils";
export function HostDetailPanel({ mode }: { mode: HostDetailMode }) {
  const navigate = useNavigate();
  const { hostId } = useParams();
  const [host, setHost] = useState<RowData | null>(null);
  const [loading, setLoading] = useState(false);
  const [images, setImages] = useState<RowData[]>([]);
  const [imagesLoading, setImagesLoading] = useState(false);
  const [containers, setContainers] = useState<RowData[]>([]);
  const [containersLoading, setContainersLoading] = useState(false);
  const [showCreateContainer, setShowCreateContainer] = useState(false);
  const [prefillImage, setPrefillImage] = useState<string>();
  const resolvedHostId = Number(hostId ?? 0);

  async function loadHost() {
    if (!resolvedHostId) return;
    setLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        `/api-admin/virtualization/hosts/${resolvedHostId}`,
      );
      setHost(result as RowData);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载主机详情失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function loadImages() {
    if (!resolvedHostId) return;
    setImagesLoading(true);
    try {
      const result = await adminApi.list<RowData[]>(
        `/api-admin/virtualization/hosts/${resolvedHostId}/images`,
      );
      setImages(Array.isArray(result) ? result : normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载镜像失败");
    } finally {
      setImagesLoading(false);
    }
  }

  async function loadContainers() {
    if (!resolvedHostId) return;
    setContainersLoading(true);
    try {
      const result = await adminApi.list<RowData>(
        "/api-admin/virtualization/containers",
        {
          current: 1,
          pageSize: 1000,
          host_id: resolvedHostId,
        },
      );
      const rows = normalizeRows(result).filter((row) => {
        if (row.host_id === undefined || row.host_id === null) return true;
        return Number(row.host_id) === resolvedHostId;
      });
      setContainers(rows);
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载容器失败");
    } finally {
      setContainersLoading(false);
    }
  }

  async function checkHost() {
    if (!resolvedHostId) return;
    try {
      const result = await adminApi.list<RowData>(
        `/api-admin/virtualization/hosts/${resolvedHostId}/check`,
        { force_refresh: true },
      );
      message.success(
        `检查完成：${String((result as RowData).status ?? "unknown")}`,
      );
      await loadHost();
      if (mode !== "overview") {
        await Promise.all([loadImages(), loadContainers()]);
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "检查主机失败");
    }
  }

  useEffect(() => {
    void loadHost();
  }, [resolvedHostId]);

  useEffect(() => {
    if (mode === "images") {
      void loadImages();
    }
    if (mode === "containers") {
      void loadContainers();
    }
  }, [mode, resolvedHostId]);

  const tabs = [
    {
      key: "overview",
      label: "概览",
      to: `/admin/virtualization/${resolvedHostId}`,
    },
    {
      key: "containers",
      label: "容器",
      to: `/admin/virtualization/${resolvedHostId}/containers`,
    },
    {
      key: "images",
      label: "镜像",
      to: `/admin/virtualization/${resolvedHostId}/images`,
    },
  ];

  return (
    <Space className="virtualization-stack" direction="vertical" size={16}>
      <Card
        className="virtualization-detail-card"
        loading={loading}
        bordered={false}
      >
        {host ? (
          <>
            <div className="virtualization-inline-header">
              <div className="virtualization-inline-header-main">
                <Space size={10} wrap>
                  <span className="virtualization-host-icon large">
                    {renderHostIcon(host)}
                  </span>
                  <Typography.Text
                    strong
                    className="virtualization-inline-hostname"
                  >
                    {host.name}
                  </Typography.Text>
                  <Tag color={host.host_type === "pve" ? "purple" : "blue"}>
                    {host.host_type === "pve" ? "Proxmox VE" : "Docker"}
                  </Tag>
                  <Tooltip title={hostErrorText(host)}>
                    <Tag color={statusColor[String(host.status)]}>
                      {String(host.status ?? "unknown")}
                    </Tag>
                  </Tooltip>
                  {renderArchitecture(host)}
                </Space>
                <Typography.Text type="secondary">
                  {String(host.endpoint ?? "-")}
                </Typography.Text>
              </div>
              <Space wrap>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => navigate("/admin/virtualization")}
                >
                  返回列表
                </Button>
                <Button
                  icon={<SyncOutlined />}
                  onClick={() => void checkHost()}
                >
                  检查连接
                </Button>
              </Space>
            </div>

            <div className="virtualization-detail-tabs">
              {tabs.map((tab) => (
                <Link
                  key={tab.key}
                  className={`virtualization-detail-tab ${mode === tab.key ? "active" : ""}`}
                  to={tab.to}
                >
                  {tab.label}
                </Link>
              ))}
            </div>
          </>
        ) : (
          <Alert type="error" message="未找到主机信息" />
        )}
      </Card>

      {host && mode === "overview" ? (
        <HostOverview
          host={host}
          onOpenImages={() =>
            navigate(`/admin/virtualization/${resolvedHostId}/images`)
          }
          onOpenContainers={() =>
            navigate(`/admin/virtualization/${resolvedHostId}/containers`)
          }
        />
      ) : null}

      {host && mode === "images" ? (
        <ImageManagementSection
          host={host}
          images={images}
          loading={imagesLoading}
          onRefresh={() => void loadImages()}
          onCreateContainer={(imageName) => {
            setPrefillImage(imageName);
            setShowCreateContainer(true);
          }}
        />
      ) : null}

      {host && mode === "containers" ? (
        <ContainerManagementSection
          host={host}
          containers={containers}
          loading={containersLoading}
          onRefresh={() => void loadContainers()}
          onCreate={() => {
            setPrefillImage(undefined);
            setShowCreateContainer(true);
          }}
        />
      ) : null}

      <CreateContainerModal
        open={showCreateContainer}
        host={host}
        images={images}
        initialImage={prefillImage}
        onCancel={() => {
          setShowCreateContainer(false);
          setPrefillImage(undefined);
        }}
        onSuccess={async () => {
          setShowCreateContainer(false);
          setPrefillImage(undefined);
          await loadContainers();
        }}
      />
    </Space>
  );
}



