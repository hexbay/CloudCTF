import {
  BellOutlined,
  BugOutlined,
  ControlOutlined,
  DownOutlined,
  FlagOutlined,
  HomeOutlined,
  LogoutOutlined,
  TeamOutlined,
  TrophyOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Dropdown, Layout, Menu, Space } from "antd";
import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  clearPlayerAuth,
  getStoredPlayerUser,
  isPlayerLoggedIn,
  type PlayerUser,
} from "@shared/auth";
import { playerApi, type PublicSystemSettings } from "@shared/api/player";
import type { MenuItem } from "@shared/types/menu";

const { Header, Content } = Layout;

const menuItems: MenuItem[] = [
  { key: "/dashboard", icon: <HomeOutlined />, label: "首页" },
  { key: "/competition", icon: <TrophyOutlined />, label: "赛事" },
  { key: "/challenges", icon: <FlagOutlined />, label: "练习" },
  { key: "/vuln-labs", icon: <BugOutlined />, label: "漏洞复现" },
  { key: "/team", icon: <TeamOutlined />, label: "队伍" },
  { key: "/scoreboard", icon: <TrophyOutlined />, label: "排行" },
  { key: "/notifications", icon: <BellOutlined />, label: "通知" },
];

export function PlayerLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [loggedIn, setLoggedIn] = useState(isPlayerLoggedIn());
  const [user, setUser] = useState<PlayerUser | null>(getStoredPlayerUser());
  const [publicSettings, setPublicSettings] = useState<PublicSystemSettings>(
    {},
  );

  useEffect(() => {
    const syncAuth = () => {
      setLoggedIn(isPlayerLoggedIn());
      setUser(getStoredPlayerUser());
    };
    window.addEventListener("player-auth-changed", syncAuth);
    return () => window.removeEventListener("player-auth-changed", syncAuth);
  }, []);

  useEffect(() => {
    void playerApi
      .publicSettings()
      .then((res) => setPublicSettings(res.data ?? {}))
      .catch(() => setPublicSettings({}));
  }, []);

  useEffect(() => {
    if (!loggedIn) {
      return;
    }

    void playerApi
      .me()
      .then((currentUser) => {
        setUser(currentUser);
        localStorage.setItem("cloudctf_user", JSON.stringify(currentUser));
      })
      .catch(() => {
        // Keep cached user info if refreshing the profile fails.
      });
  }, [loggedIn]);

  const handleLogout = () => {
    clearPlayerAuth();
    setLoggedIn(false);
    setUser(null);
    window.dispatchEvent(
      new CustomEvent("player-auth-changed", { detail: { loggedIn: false } }),
    );
    navigate("/dashboard");
  };

  const brandName =
    publicSettings.site_name?.trim() ||
    publicSettings.site_title?.trim() ||
    "CloudCTF";
  const displayName =
    user?.username?.trim() || user?.full_name?.trim() || "用户";
  const canAccessAdmin = hasManagementAccess(user);

  return (
    <Layout className="player-shell">
      <Header className="player-header">
        <div className="player-brand">
          <img className="player-brand-logo" src="/logo.svg" alt={brandName} />
          <span>{brandName}</span>
        </div>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[getSelectedKey(location.pathname)]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
        {loggedIn ? (
          <Dropdown
            menu={{
              items: [
                { key: "profile", label: "个人中心", icon: <UserOutlined /> },
                ...(canAccessAdmin
                  ? [
                      {
                        key: "admin",
                        label: "进入管理后台",
                        icon: <ControlOutlined />,
                      },
                    ]
                  : []),
                { key: "logout", label: "退出登录", icon: <LogoutOutlined /> },
              ],
              onClick: ({ key }) => {
                if (key === "profile") navigate("/account/profile");
                if (key === "admin") navigate("/admin/dashboard");
                if (key === "logout") handleLogout();
              },
            }}
            trigger={["click"]}
          >
            <button className="player-user-trigger" type="button">
              <Avatar size={30} icon={<UserOutlined />} />
              <Space size={6}>
                <span>{displayName}</span>
                <DownOutlined />
              </Space>
            </button>
          </Dropdown>
        ) : (
          <div className="player-auth-actions">
            <Button
              type="primary"
              size="small"
              onClick={() =>
                window.dispatchEvent(
                  new CustomEvent("show-player-auth", { detail: "login" }),
                )
              }
            >
              登录
            </Button>
            <Button
              size="small"
              onClick={() =>
                window.dispatchEvent(
                  new CustomEvent("show-player-auth", { detail: "register" }),
                )
              }
            >
              注册
            </Button>
          </div>
        )}
      </Header>
      <Content className="player-content">
        <Outlet />
      </Content>
    </Layout>
  );
}

function getSelectedKey(pathname: string) {
  return (
    menuItems
      .find((item) => pathname.startsWith(String(item?.key)))
      ?.key?.toString() ?? "/dashboard"
  );
}

function hasManagementAccess(user: PlayerUser | null) {
  if (!user) return false;
  if (user.is_admin || user.is_staff) return true;

  const roles = [
    user.role,
    ...(Array.isArray(user.roles) ? user.roles : []),
    ...(Array.isArray(user.permissions) ? user.permissions : []),
  ]
    .filter(Boolean)
    .map((value) => String(value).toLowerCase());

  return roles.some((value) =>
    ["admin", "superadmin", "staff", "manager", "manage"].some((keyword) =>
      value.includes(keyword),
    ),
  );
}
