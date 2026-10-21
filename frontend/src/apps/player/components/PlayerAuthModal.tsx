import { LockOutlined, MailOutlined, UserOutlined } from "@ant-design/icons";
import { Alert, Button, Form, Input, Modal, Tabs, message } from "antd";
import { useEffect, useState } from "react";
import {
  playerApi,
  type PlayerRegisterPayload,
  type PublicSystemSettings,
} from "@shared/api/player";

type AuthTab = "login" | "register";

type PlayerAuthModalProps = {
  open: boolean;
  initialTab?: AuthTab;
  onClose: () => void;
};

function persistAuth(
  accessToken: string,
  refreshToken: string,
  user?: unknown,
) {
  localStorage.setItem("cloudctf_token", accessToken);
  localStorage.setItem("cloudctf_refresh_token", refreshToken);
  if (user) {
    localStorage.setItem("cloudctf_user", JSON.stringify(user));
  }
}

export function PlayerAuthModal({
  open,
  initialTab = "login",
  onClose,
}: PlayerAuthModalProps) {
  const [activeTab, setActiveTab] = useState<AuthTab>(initialTab);
  const [loading, setLoading] = useState(false);
  const [registerLoading, setRegisterLoading] = useState(false);
  const [loginError, setLoginError] = useState("");
  const [registerError, setRegisterError] = useState("");
  const [publicSettings, setPublicSettings] = useState<PublicSystemSettings>(
    {},
  );

  useEffect(() => {
    if (open) {
      setActiveTab(initialTab);
      setLoginError("");
      setRegisterError("");
    }
  }, [initialTab, open]);

  useEffect(() => {
    void playerApi
      .publicSettings()
      .then((res) => setPublicSettings(res.data ?? {}))
      .catch(() => setPublicSettings({}));
  }, []);

  const handleLogin = async (values: {
    username: string;
    password: string;
  }) => {
    setLoading(true);
    setLoginError("");
    try {
      const result = await playerApi.login(values);
      persistAuth(result.access_token, result.refresh_token, result.user);
      window.dispatchEvent(
        new CustomEvent("player-auth-changed", {
          detail: { loggedIn: true, user: result.user },
        }),
      );
      message.success("登录成功");
      onClose();
    } catch (error) {
      const text = error instanceof Error ? error.message : "登录失败";
      setLoginError(text);
      message.error(text);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (
    values: PlayerRegisterPayload & { confirmPassword: string },
  ) => {
    setRegisterLoading(true);
    setRegisterError("");
    try {
      const { confirmPassword: _confirmPassword, ...payload } = values;
      const result = await playerApi.register(payload);
      persistAuth(result.access_token, result.refresh_token, result.data);
      window.dispatchEvent(
        new CustomEvent("player-auth-changed", {
          detail: { loggedIn: true, user: result.data },
        }),
      );
      message.success("注册成功，已自动登录");
      onClose();
    } catch (error) {
      const text = error instanceof Error ? error.message : "注册失败";
      setRegisterError(text);
      message.error(text);
    } finally {
      setRegisterLoading(false);
    }
  };

  const brandName =
    publicSettings.site_name?.trim() ||
    publicSettings.site_title?.trim() ||
    "CloudCTF";
  const brandSubtitle =
    publicSettings.site_description?.trim() || "CTF 竞赛平台";
  const registrationEnabled = publicSettings.allow_registration !== false;

  return (
    <Modal
      open={open}
      footer={null}
      width={520}
      centered
      destroyOnHidden
      onCancel={onClose}
      className="player-auth-modal"
    >
      <div className="player-auth-modal-header">
        <div className="player-auth-modal-title">{brandName}</div>
        <div className="player-auth-modal-subtitle">{brandSubtitle}</div>
      </div>
      <Tabs
        centered
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as AuthTab)}
        items={[
          {
            key: "login",
            label: "登录",
            children: (
              <Form
                layout="vertical"
                className="player-auth-form"
                onFinish={handleLogin}
              >
                {loginError ? (
                  <Alert
                    className="player-auth-alert"
                    type="error"
                    showIcon
                    message={loginError}
                  />
                ) : null}
                <Form.Item
                  name="username"
                  rules={[{ required: true, message: "请输入用户名" }]}
                >
                  <Input
                    size="large"
                    prefix={<UserOutlined />}
                    placeholder="用户名"
                    autoComplete="username"
                  />
                </Form.Item>
                <Form.Item
                  name="password"
                  rules={[{ required: true, message: "请输入密码" }]}
                >
                  <Input.Password
                    size="large"
                    prefix={<LockOutlined />}
                    placeholder="密码"
                    autoComplete="current-password"
                  />
                </Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  size="large"
                  block
                  loading={loading}
                >
                  登录
                </Button>
              </Form>
            ),
          },
          {
            key: "register",
            label: "注册",
            children: registrationEnabled ? (
              <Form
                layout="vertical"
                className="player-auth-form"
                onFinish={handleRegister}
              >
                {registerError ? (
                  <Alert
                    className="player-auth-alert"
                    type="error"
                    showIcon
                    message={registerError}
                  />
                ) : null}
                <Form.Item
                  name="username"
                  rules={[{ required: true, message: "请输入用户名" }]}
                >
                  <Input
                    size="large"
                    prefix={<UserOutlined />}
                    placeholder="用户名"
                    autoComplete="username"
                  />
                </Form.Item>
                <Form.Item
                  name="email"
                  rules={[
                    { required: true, message: "请输入邮箱" },
                    { type: "email", message: "请输入有效的邮箱地址" },
                  ]}
                >
                  <Input
                    size="large"
                    prefix={<MailOutlined />}
                    placeholder="邮箱"
                    autoComplete="email"
                  />
                </Form.Item>
                <Form.Item name="full_name">
                  <Input
                    size="large"
                    prefix={<UserOutlined />}
                    placeholder="昵称或真实姓名（可选）"
                    autoComplete="name"
                  />
                </Form.Item>
                <Form.Item
                  name="password"
                  rules={[
                    { required: true, message: "请输入密码" },
                    { min: 6, message: "密码至少 6 位" },
                  ]}
                >
                  <Input.Password
                    size="large"
                    prefix={<LockOutlined />}
                    placeholder="密码"
                    autoComplete="new-password"
                  />
                </Form.Item>
                <Form.Item
                  name="confirmPassword"
                  dependencies={["password"]}
                  rules={[
                    { required: true, message: "请再次输入密码" },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue("password") === value) {
                          return Promise.resolve();
                        }
                        return Promise.reject(
                          new Error("两次输入的密码不一致"),
                        );
                      },
                    }),
                  ]}
                >
                  <Input.Password
                    size="large"
                    prefix={<LockOutlined />}
                    placeholder="确认密码"
                    autoComplete="new-password"
                  />
                </Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  size="large"
                  block
                  loading={registerLoading}
                >
                  注册并进入平台
                </Button>
              </Form>
            ) : (
              <Alert
                type="info"
                showIcon
                message="当前平台未开放注册，请联系管理员开通账号。"
              />
            ),
          },
        ]}
      />
    </Modal>
  );
}
