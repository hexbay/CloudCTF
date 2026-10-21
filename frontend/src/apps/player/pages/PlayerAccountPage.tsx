import { UserOutlined } from '@ant-design/icons';
import { Avatar, Button, Card, Col, Form, Input, Row, message } from 'antd';
import { useState } from 'react';
import { PlayerPage } from './PlayerPage';

export function PlayerAccountPage() {
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    try {
      message.success('资料保存已响应，当前页面暂未接入后端保存接口');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存资料失败');
    } finally {
      setSaving(false);
    }
  }

  return (
    <PlayerPage icon={<UserOutlined />} title="个人中心" subtitle="管理个人资料和账户信息">
      <Row gutter={[18, 18]} className="player-account-page">
        <Col xs={24} md={8}>
          <Card className="player-panel-card account-profile-card">
            <Avatar size={88} icon={<UserOutlined />} />
            <h3>CTFer</h3>
            <p>持续训练，保持敏锐。</p>
          </Card>
        </Col>
        <Col xs={24} md={16}>
          <Card className="player-panel-card account-edit-card">
            <Form layout="vertical" className="player-account-form">
              <Form.Item label="用户名"><Input defaultValue="CTFer" /></Form.Item>
              <Form.Item label="邮箱"><Input defaultValue="ctfer@example.com" /></Form.Item>
              <Form.Item label="个人简介"><Input.TextArea rows={5} defaultValue="专注 Web、Crypto 与漏洞复现。" /></Form.Item>
              <Button type="primary" loading={saving} onClick={() => void handleSave()}>保存资料</Button>
            </Form>
          </Card>
        </Col>
      </Row>
    </PlayerPage>
  );
}
