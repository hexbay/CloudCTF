import {
  DeleteOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  Button,
  Input,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import { AdminDataTableCard } from "../components/AdminDataTableCard";

type SubmissionRow = {
  id: number;
  challenge_name: string;
  username: string;
  team_name?: string;
  is_correct: boolean;
  points: number;
  ip_address?: string;
  submitted_at: string;
};

export function AdminPracticeSubmissionsPage() {
  const [rows, setRows] = useState<SubmissionRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [keyword, setKeyword] = useState("");
  const [deletingId, setDeletingId] = useState<number | null>(null);

  useEffect(() => {
    void load();
  }, [current, pageSize]);

  async function load(nextKeyword = keyword) {
    setLoading(true);
    try {
      const result = await adminApi.list<SubmissionRow>(
        "/api-admin/submissions",
        {
          current,
          pageSize,
          competition_id: "null",
          keyword: nextKeyword || undefined,
        },
      );
      const data =
        (result as { data?: SubmissionRow[]; total?: number }).data ?? [];
      setRows(data);
      setTotal((result as { total?: number }).total ?? data.length);
    } catch (error) {
      message.error(
        error instanceof Error ? error.message : "加载提交记录失败",
      );
    } finally {
      setLoading(false);
    }
  }

  async function remove(id: number) {
    try {
      setDeletingId(id);
      await adminApi.remove(`/api-admin/submissions/${id}`);
      message.success("提交记录已删除");
      await load();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setDeletingId(null);
    }
  }

  const columns: ColumnsType<SubmissionRow> = [
    { title: "用户", dataIndex: "username", width: 140, ellipsis: true },
    {
      title: "题目",
      dataIndex: "challenge_name",
      width: 220,
      ellipsis: true,
    },
    {
      title: "结果",
      dataIndex: "is_correct",
      width: 90,
      render: (value) => (
        <Tag color={value ? "success" : "error"}>{value ? "正确" : "错误"}</Tag>
      ),
    },
    { title: "得分", dataIndex: "points", width: 80 },
    {
      title: "队伍",
      dataIndex: "team_name",
      width: 140,
      ellipsis: true,
      render: (value) => value ?? "-",
    },
    { title: "IP", dataIndex: "ip_address", width: 140, ellipsis: true },
    {
      title: "提交时间",
      dataIndex: "submitted_at",
      width: 180,
      render: (value) => (value ? new Date(value).toLocaleString() : "-"),
    },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, row) => (
        <Space size={4} wrap className="admin-action-group">
          <Popconfirm
            title="删除此提交记录"
            description="删除后不可恢复。"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => remove(row.id)}
          >
            <Button
              type="text"
              size="small"
              danger
              loading={deletingId === row.id}
              className="admin-action-button admin-action-button-danger"
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="admin-page">
      <AdminDataTableCard
        title="提交记录"
        search={
          <div className="admin-search-form">
            <div className="admin-search-field">
              <label>关键词：</label>
              <Input
                allowClear
                placeholder="搜索用户名、题目、队伍"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onPressEnter={() => {
                  setCurrent(1);
                  void load();
                }}
              />
            </div>
            <Space>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={() => {
                  setCurrent(1);
                  void load();
                }}
              >
                查询
              </Button>
            </Space>
          </div>
        }
        actions={
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void load()}
          >
            刷新
          </Button>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: true }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, s) => {
              setCurrent(p);
              setPageSize(s);
            },
          }}
        />
      </AdminDataTableCard>
    </div>
  );
}
