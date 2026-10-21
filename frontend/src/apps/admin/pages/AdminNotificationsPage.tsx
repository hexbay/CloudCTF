import { ReloadOutlined } from "@ant-design/icons";
import { CheckOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { formatTime, normalizeRows } from "./adminPageUtils";

type NotificationRow = {
  id: number;
  title: string;
  content?: string;
  message_type?: string;
  audience?: string;
  action_url?: string;
  is_read?: boolean;
  created_at?: string;
};

export function AdminNotificationsPage() {
  const [rows, setRows] = useState<NotificationRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<NotificationRow | null>(null);
  const [markingAll, setMarkingAll] = useState(false);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const result = await adminApi.list<NotificationRow>("/api-admin/notifications", {
        current: 1,
        pageSize: 200,
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载通知列表失败");
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(row: NotificationRow) {
    setDetail(row);
    if (!row.is_read) {
      try {
        await adminApi.action(`/api-admin/notifications/${row.id}/read`, "PUT");
        setRows((current) => current.map((item) => (item.id === row.id ? { ...item, is_read: true } : item)));
      } catch {
        // Reading state is best-effort; keep the detail visible.
      }
    }
  }

  async function markAllRead() {
    try {
      setMarkingAll(true);
      await adminApi.action("/api-admin/notifications/mark-all-read", "PUT");
      setRows((current) => current.map((item) => ({ ...item, is_read: true })));
      setDetail((current) => (current ? { ...current, is_read: true } : current));
      message.success("全部通知已标记为已读");
    } catch (error) {
      message.error(error instanceof Error ? error.message : "标记全部已读失败");
    } finally {
      setMarkingAll(false);
    }
  }

  const columns: ColumnsType<NotificationRow> = [
    { title: "标题", dataIndex: "title", ellipsis: true },
    {
      title: "类型",
      dataIndex: "message_type",
      width: 120,
      render: (value) => <Tag>{typeLabel(String(value ?? "system"))}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "is_read",
      width: 90,
      render: (value) => <Tag color={value ? "default" : "blue"}>{value ? "已读" : "未读"}</Tag>,
    },
    {
      title: "端",
      dataIndex: "audience",
      width: 90,
      render: (value) => <Tag color="purple">{value === "admin" ? "管理员" : "选手"}</Tag>,
    },
    { title: "创建时间", dataIndex: "created_at", width: 180, render: formatTime },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, row) => (
        <Button type="text" size="small" className="admin-action-button" onClick={() => void openDetail(row)}>
          查看
        </Button>
      ),
    },
  ];

  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">
        通知中心
      </Typography.Title>

      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Typography.Title level={5}>管理员通知列表</Typography.Title>
          <Space>
            <Button
              icon={<CheckOutlined />}
              loading={markingAll}
              disabled={!rows.some((item) => !item.is_read)}
              onClick={() => void markAllRead()}
            >
              全部已读
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
          </Space>
        </div>

        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: true }}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Drawer
        title={detail?.title ?? "通知详情"}
        open={Boolean(detail)}
        width={520}
        onClose={() => setDetail(null)}
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <div>
            <Typography.Text type="secondary">类型</Typography.Text>
            <div>
              <Tag>{typeLabel(String(detail?.message_type ?? "system"))}</Tag>
            </div>
          </div>
          <div>
            <Typography.Text type="secondary">状态</Typography.Text>
            <div>
              <Tag color={detail?.is_read ? "default" : "blue"}>{detail?.is_read ? "已读" : "未读"}</Tag>
            </div>
          </div>
          <div>
            <Typography.Text type="secondary">接收端</Typography.Text>
            <div>{detail?.audience === "admin" ? "管理员" : "选手"}</div>
          </div>
          <div>
            <Typography.Text type="secondary">创建时间</Typography.Text>
            <div>{formatTime(detail?.created_at)}</div>
          </div>
          <div>
            <Typography.Text type="secondary">跳转地址</Typography.Text>
            <div>{detail?.action_url || "-"}</div>
          </div>
          <div>
            <Typography.Text type="secondary">内容</Typography.Text>
            <Typography.Paragraph style={{ marginTop: 8, whiteSpace: "pre-wrap" }}>
              {detail?.content || "-"}
            </Typography.Paragraph>
          </div>
        </Space>
      </Drawer>
    </div>
  );
}

function typeLabel(value: string) {
  if (value === "competition") return "赛事";
  if (value === "team") return "队伍";
  if (value === "personal") return "个人";
  return "系统";
}

