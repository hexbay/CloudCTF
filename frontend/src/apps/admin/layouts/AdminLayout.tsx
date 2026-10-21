import {
  AuditOutlined,
  BellOutlined,
  CloudOutlined,
  CompassOutlined,
  DashboardOutlined,
  FlagOutlined,
  GlobalOutlined,
  ReadOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  IdcardOutlined,
  SafetyOutlined,
  SettingOutlined,
  TeamOutlined,
  TrophyOutlined,
  UserOutlined,
} from "@ant-design/icons";
import {
  Avatar,
  Badge,
  Button,
  Alert,
  Dropdown,
  Form,
  Input,
  Layout,
  Menu,
  Modal,
  Space,
  Tag,
  message,
} from "antd";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import type { MenuItem } from "@shared/types/menu";
import { adminApi } from "@shared/api/admin";
import { publicApi } from "@shared/api/public";
import { clearAdminTokens } from "@shared/auth/token";
import { AdminBreadcrumb } from "../components/AdminBreadcrumb";

const { Header, Sider, Content } = Layout;

type AdminMe = {
  id: number;
  username: string;
  email?: string;
  nickname?: string;
  role?: string;
  must_change_password?: boolean;
  permissions?: string[];
};

const menuItems: MenuItem[] = [
  { key: "/admin/dashboard", icon: <DashboardOutlined />, label: "数据概览" },
  {
    key: "/admin/users-root",
    icon: <TeamOutlined />,
    label: "用户管理",
    children: [
      { key: "/admin/users", label: "用户列表" },
      { key: "/admin/teams", label: "队伍管理" },
    ],
  },
  { key: "/admin/competition", icon: <TrophyOutlined />, label: "比赛列表" },
  {
    key: "/admin/theory-root",
    icon: <ReadOutlined />,
    label: "理论题管理",
    children: [
      { key: "/admin/theory", label: "理论题库" },
      { key: "/admin/theory-exams", label: "理论考试" },
      { key: "/admin/theory-submissions", label: "答题记录" },
    ],
  },
  {
    key: "/admin/practice-root",
    icon: <FlagOutlined />,
    label: "夺旗练习场",
    children: [
      { key: "/admin/challenge", label: "题目管理" },
      { key: "/admin/practice-submissions", label: "提交记录" },
    ],
  },
  { key: "/admin/vuln-lab", icon: <SafetyOutlined />, label: "漏洞列表" },
  {
    key: "/admin/virtualization-root",
    icon: <CloudOutlined />,
    label: "虚拟化管理",
    children: [
      { key: "/admin/virtualization", label: "主机管理" },
      { key: "/admin/virtualization-templates", label: "模板管理" },
      { key: "/admin/virtualization-groups", label: "容器组" },
      { key: "/admin/virtualization-containers", label: "容器管理" },
    ],
  },
  {
    key: "/admin/admin-root",
    icon: <UserOutlined />,
    label: "管理员管理",
    children: [
      { key: "/admin/admins", label: "管理员列表" },
      { key: "/admin/roles", label: "角色管理" },
    ],
  },
  { key: "/admin/settings", icon: <SettingOutlined />, label: "系统设置" },
  { key: "/admin/audit", icon: <AuditOutlined />, label: "审计日志" },
];

export function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [openKeys, setOpenKeys] = useState<string[]>(() =>
    getOpenKeys(location.pathname),
  );
  const [collapsed, setCollapsed] = useState(false);
  const [currentUser, setCurrentUser] = useState<AdminMe | null>(null);
  const [profileOpen, setProfileOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [systemVersion, setSystemVersion] = useState("dev");
  const [passwordForm] = Form.useForm<{ current_password: string; new_password: string; confirm_password: string }>();
  const visibleMenuItems = useMemo(
    () => filterMenuItems(menuItems, currentUser?.permissions),
    [currentUser?.permissions],
  );

  useEffect(() => {
    const routeOpenKeys = getOpenKeys(location.pathname);
    setOpenKeys((current) =>
      Array.from(new Set([...current, ...routeOpenKeys])),
    );
  }, [location.pathname]);

  useEffect(() => {
    if (!adminApi.hasToken()) {
      return;
    }
    void adminApi
      .me()
      .then((user) => {
        const admin = user as AdminMe;
        setCurrentUser(admin);
        if (admin.must_change_password) {
          setPasswordModalOpen(true);
        }
      })
      .catch(() => {
        clearAdminTokens();
        navigate("/admin/login", { replace: true });
      });
  }, [navigate]);

  useEffect(() => {
    void publicApi
      .version()
      .then((payload) => setSystemVersion(payload.version?.trim() || "dev"))
      .catch(() => setSystemVersion("dev"));
  }, []);

  useEffect(() => {
    if (!adminApi.hasToken()) {
      return;
    }
    void loadUnreadCount();
  }, [location.pathname]);

  if (!adminApi.hasToken()) {
    return <Navigate to="/admin/login" replace />;
  }

  const handleLogout = async () => {
    try {
      await adminApi.logout();
    } catch (error) {
      message.warning(
        error instanceof Error
          ? error.message
          : "退出请求失败，已清除本地登录状态",
      );
    }
    clearAdminTokens();
    setCurrentUser(null);
    navigate("/admin/login", { replace: true });
  };

  async function loadUnreadCount() {
    try {
      const result = await adminApi.list<{ count?: number }>(
        "/api-admin/notifications/unread-count",
      );
      setUnreadCount(Number((result as { count?: number }).count ?? 0));
    } catch {
      setUnreadCount(0);
    }
  }

  async function handleChangePassword() {
    try {
      const values = await passwordForm.validateFields();
      setPasswordSubmitting(true);
      await adminApi.changePassword({
        current_password: values.current_password,
        new_password: values.new_password,
      });
      setCurrentUser((prev) => (prev ? { ...prev, must_change_password: false } : prev));
      setPasswordModalOpen(false);
      passwordForm.resetFields();
      message.success("密码已更新");
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setPasswordSubmitting(false);
    }
  }

  return (
    <Layout className="admin-shell">
      <Sider
        width={216}
        collapsedWidth={72}
        collapsed={collapsed}
        onCollapse={setCollapsed}
        breakpoint="lg"
        className="admin-sider"
      >
        <div
          className={
            collapsed ? "admin-brand admin-brand-collapsed" : "admin-brand"
          }
        >
          <img className="admin-brand-mark" src="/logo.svg" alt="logo" />
          {!collapsed ? (
            <span className="admin-brand-text">CloudCTF Admin</span>
          ) : null}
        </div>
        <Menu
          className="admin-menu"
          theme="dark"
          mode="inline"
          inlineCollapsed={collapsed}
          selectedKeys={[getSelectedKey(location.pathname, visibleMenuItems)]}
          openKeys={collapsed ? undefined : openKeys}
          onOpenChange={(keys) => {
            if (!collapsed) {
              setOpenKeys(keys.map(String));
            }
          }}
          items={visibleMenuItems}
          motion={{ motionName: "" }}
          onClick={({ key }) => {
            if (!String(key).endsWith("-root")) navigate(key);
          }}
        />
      </Sider>
      <Layout>
        <Header className="admin-header">
          <Button
            className="admin-collapse-button"
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed((value) => !value)}
          />
          <Space size={10} className="admin-top-actions">
            <Tag className="admin-version-tag" bordered={false}>
              {systemVersion}
            </Tag>
            <Button
              className="admin-lang-button"
              type="text"
              icon={<GlobalOutlined />}
            />
            <Badge count={unreadCount} size="small" offset={[-2, 4]}>
              <Button
                className="admin-icon-button"
                type="text"
                icon={<BellOutlined />}
                onClick={() => navigate("/admin/notifications")}
              />
            </Badge>
            <Dropdown
              trigger={["click"]}
              menu={{
                items: [
                  {
                    key: "player",
                    icon: <CompassOutlined />,
                    label: "前往选手端",
                  },
                  {
                    key: "profile",
                    icon: <IdcardOutlined />,
                    label: "个人中心",
                  },
                  {
                    key: "logout",
                    icon: <LogoutOutlined />,
                    label: "退出登录",
                  },
                ],
                onClick: ({ key }) => {
                  if (key === "player") {
                    navigate("/dashboard");
                    return;
                  }
                  if (key === "profile") {
                    setProfileOpen(true);
                    return;
                  }
                  if (key === "logout") {
                    void handleLogout();
                  }
                },
              }}
            >
              <button type="button" className="admin-user-trigger">
                <Avatar
                  size={26}
                  className="admin-user-avatar"
                  icon={!currentUser?.username ? <UserOutlined /> : undefined}
                >
                  {currentUser?.username
                    ? currentUser.username.slice(0, 1).toUpperCase()
                    : null}
                </Avatar>
              </button>
            </Dropdown>
          </Space>
        </Header>
        <Content className="admin-content">
          <AdminBreadcrumb />
          <Outlet />
        </Content>
      </Layout>
      <Modal
        title="个人中心"
        open={profileOpen}
        onCancel={() => setProfileOpen(false)}
        footer={[
          <Button key="close" onClick={() => setProfileOpen(false)}>
            关闭
          </Button>,
          <Button
            key="password"
            type="primary"
            onClick={() => {
              setProfileOpen(false);
              setPasswordModalOpen(true);
            }}
          >
            修改密码
          </Button>,
          <Button key="logout" danger onClick={() => void handleLogout()}>
            退出登录
          </Button>,
        ]}
      >
        <div className="admin-profile-modal">
          <div className="admin-profile-row">
            <span>用户名</span>
            <strong>{currentUser?.username || "-"}</strong>
          </div>
          <div className="admin-profile-row">
            <span>昵称</span>
            <strong>{currentUser?.nickname || "-"}</strong>
          </div>
          <div className="admin-profile-row">
            <span>邮箱</span>
            <strong>{currentUser?.email || "-"}</strong>
          </div>
          <div className="admin-profile-row">
            <span>角色</span>
            <strong>{currentUser?.role || "-"}</strong>
          </div>
        </div>
      </Modal>
      <Modal
        title={currentUser?.must_change_password ? "请先修改管理员密码" : "修改密码"}
        open={passwordModalOpen}
        closable={!currentUser?.must_change_password}
        maskClosable={!currentUser?.must_change_password}
        keyboard={!currentUser?.must_change_password}
        onCancel={() => {
          if (currentUser?.must_change_password) {
            return;
          }
          setPasswordModalOpen(false);
        }}
        onOk={() => void handleChangePassword()}
        okText="确认修改"
        cancelText="取消"
        confirmLoading={passwordSubmitting}
      >
        {currentUser?.must_change_password ? (
          <Alert
            type="warning"
            showIcon
            message="当前管理员密码仍是初始化密码或被重置后的密码，请修改后继续使用后台。"
            style={{ marginBottom: 16 }}
          />
        ) : null}
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            label="当前密码"
            name="current_password"
            rules={[{ required: true, message: "请输入当前密码" }]}
          >
            <Input.Password autoComplete="current-password" placeholder="请输入当前密码" />
          </Form.Item>
          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 6, message: "密码至少 6 位" },
            ]}
          >
            <Input.Password autoComplete="new-password" placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            label="确认新密码"
            name="confirm_password"
            dependencies={["new_password"]}
            rules={[
              { required: true, message: "请再次输入新密码" },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue("new_password") === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error("两次输入的新密码不一致"));
                },
              }),
            ]}
          >
            <Input.Password autoComplete="new-password" placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
}

function getSelectedKey(pathname: string, items: MenuItem[] = menuItems) {
  const leafKeys = flattenMenuKeys(items);
  const matches = leafKeys.filter((key) => pathname.startsWith(key));
  return matches.sort((a, b) => b.length - a.length)[0] ?? "/admin/dashboard";
}

function filterMenuItems(items: MenuItem[], permissions?: string[]): MenuItem[] {
  if (!permissions) return items;
  return items.flatMap((item) => {
    const anyItem = item as any;
    if (anyItem?.children) {
      const children = filterMenuItems(anyItem.children, permissions);
      return children.length ? [{ ...anyItem, children }] : [];
    }
    const required = menuPermission(String(anyItem?.key ?? ""));
    return !required || hasPermission(permissions, required) ? [item] : [];
  });
}

function menuPermission(key: string) {
  if (key.startsWith("/admin/users")) return "admin:users";
  if (key.startsWith("/admin/teams")) return "admin:teams";
  if (key.startsWith("/admin/admins") || key.startsWith("/admin/roles")) return "admin:roles";
  if (key.startsWith("/admin/competition")) return "admin:competitions";
  if (key.startsWith("/admin/theory-exams")) return "admin:exam";
  if (key.startsWith("/admin/theory")) return "admin:theory";
  if (key.startsWith("/admin/challenge")) return "admin:challenges";
  if (key.startsWith("/admin/practice-submissions")) return "admin:submissions";
  if (key.startsWith("/admin/vuln-lab")) return "admin:vuln_labs";
  if (key.startsWith("/admin/virtualization")) return "admin:virtualization";
  if (key.startsWith("/admin/settings")) return "admin:settings";
  if (key.startsWith("/admin/audit")) return "admin:audit";
  return "";
}

function hasPermission(permissions: string[], required: string) {
  return permissions.some((permission) => {
    const normalized = permission.trim();
    if (normalized === "*" || normalized === "admin:*" || normalized === required) return true;
    return normalized.endsWith("*") && required.startsWith(normalized.slice(0, -1));
  });
}

function getOpenKeys(pathname: string) {
  return menuItems
    .filter(
      (item) =>
        "children" in (item ?? {}) &&
        (item as any).children?.some((child: any) =>
          pathname.startsWith(child.key),
        ),
    )
    .map((item) => String(item?.key));
}

function flattenMenuKeys(items: MenuItem[]) {
  return items.flatMap((item) => {
    const anyItem = item as any;
    if (anyItem?.children) {
      return anyItem.children.map((child: any) => String(child.key));
    }
    return [String(anyItem?.key)];
  });
}
