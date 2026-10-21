import { LockOutlined, UserOutlined } from "@ant-design/icons";
import { Alert, Button, Checkbox, Form, Input, Typography } from "antd";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "@shared/api/admin";

type LoginFormValues = {
  username: string;
  password: string;
  remember?: boolean;
};

export function AdminLoginPage() {
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleFinish = async (values: LoginFormValues) => {
    setSubmitting(true);
    setErrorMessage(null);

    try {
      await adminApi.login(values);
      navigate("/admin/dashboard");
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "登录失败，请检查账户信息后重试",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="admin-login-page">
      <section className="admin-login-hero">
        <div className="admin-login-backdrop" />
        <div className="admin-login-container">
          <div className="admin-login-panel-shell">
            <div className="admin-login-brandmark">
              <span className="admin-login-brandicon">CF</span>
              <div>
                <Typography.Text className="admin-login-brandname">
                  CloudCTF Admin
                </Typography.Text>
                <Typography.Text className="admin-login-branddesc">
                  管理控制台
                </Typography.Text>
              </div>
            </div>

            <div className="admin-login-panel-head">
              <Typography.Title level={2} className="admin-login-panel-title">
                管理员登录
              </Typography.Title>
              <Typography.Paragraph className="admin-login-panel-subtitle">
                使用管理员账户进入后台。
              </Typography.Paragraph>
            </div>

            <Form<LoginFormValues>
              layout="vertical"
              size="large"
              initialValues={{ remember: true }}
              onFinish={handleFinish}
              className="admin-login-form"
            >
              {errorMessage ? (
                <Alert
                  type="error"
                  showIcon
                  message={errorMessage}
                  className="admin-login-alert"
                />
              ) : null}

              <Form.Item
                label="用户名"
                name="username"
                rules={[{ required: true, message: "请输入管理员用户名" }]}
              >
                <Input
                  autoComplete="username"
                  prefix={<UserOutlined />}
                  placeholder="请输入管理员用户名"
                />
              </Form.Item>

              <Form.Item
                label="密码"
                name="password"
                rules={[{ required: true, message: "请输入登录密码" }]}
              >
                <Input.Password
                  autoComplete="current-password"
                  prefix={<LockOutlined />}
                  placeholder="请输入登录密码"
                />
              </Form.Item>

              <div className="admin-login-form-meta">
                <Form.Item name="remember" valuePropName="checked" noStyle>
                  <Checkbox>记住当前设备</Checkbox>
                </Form.Item>
              </div>

              <Button
                type="primary"
                htmlType="submit"
                block
                loading={submitting}
              >
                登录
              </Button>
            </Form>
          </div>
        </div>
      </section>
    </main>
  );
}
