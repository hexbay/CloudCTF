import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Input, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { formatTime, normalizeRows } from "./adminPageUtils";

type AuditRow = {
  id: number;
  username?: string;
  action_type?: string;
  resource_type?: string;
  resource_id?: string | number;
  description?: string;
  ip_address?: string;
  created_at?: string;
};

export function AdminAuditPage() {
  const [rows, setRows] = useState<AuditRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");

  useEffect(() => {
    void load();
  }, []);

  async function load(nextKeyword = keyword) {
    setLoading(true);
    try {
      const result = await adminApi.list<AuditRow>("/api-admin/audit/logs", {
        current: 1,
        pageSize: 200,
        username: nextKeyword || undefined,
      });
      setRows(normalizeRows(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载审计日志失败");
    } finally {
      setLoading(false);
    }
  }

  const columns: ColumnsType<AuditRow> = [
    {
      title: "用户",
      dataIndex: "username",
      width: 140,
      render: (value) => value || "-",
    },
    {
      title: "操作",
      dataIndex: "action_type",
      width: 140,
      render: (value) => <Tag>{String(value ?? "-")}</Tag>,
    },
    {
      title: "资源类型",
      dataIndex: "resource_type",
      width: 140,
      render: (value) => value || "-",
    },
    {
      title: "资源 ID",
      dataIndex: "resource_id",
      width: 120,
      render: (value) => value ?? "-",
    },
    {
      title: "说明",
      dataIndex: "description",
      ellipsis: true,
      render: (value) => value || "-",
    },
    {
      title: "IP",
      dataIndex: "ip_address",
      width: 160,
      render: (value) => value || "-",
    },
    {
      title: "时间",
      dataIndex: "created_at",
      width: 180,
      render: formatTime,
    },
  ];

  return (
    <div className="admin-page">
      <Typography.Title level={4} className="admin-page-title">
        审计日志
      </Typography.Title>

      <Card className="admin-table-card" bordered={false}>
        <div className="resource-toolbar">
          <Space>
            <Input.Search
              allowClear
              placeholder="按用户名筛选搜索"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              onSearch={(value) => void load(value)}
              style={{ width: 260 }}
            />
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
    </div>
  );
}
