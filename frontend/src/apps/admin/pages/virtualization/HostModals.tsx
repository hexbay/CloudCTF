import { CloudServerOutlined, DeleteOutlined, FileZipOutlined, PlusOutlined } from "@ant-design/icons";
import { Button, Col, Divider, Form, Input, InputNumber, Modal, Row, Select, Space, Typography, Upload, message } from "antd";
import type { UploadFile } from "antd/es/upload/interface";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";
import type { RowData } from "./types";
import { imageDisplayName, normalizeRows } from "./utils";
export function CreateContainerModal({
  open,
  host,
  images,
  initialImage,
  onCancel,
  onSuccess,
}: {
  open: boolean;
  host: RowData | null;
  images: RowData[];
  initialImage?: string;
  onCancel: () => void;
  onSuccess: () => void | Promise<void>;
}) {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      host_id: host?.id,
      image: initialImage,
      ports: [{ host: "", container: "" }],
      environment: [{ key: "", value: "" }],
    });
  }, [open, host?.id, initialImage]);

  async function submit() {
    const values = await form.validateFields();
    const ports: Record<string, string> = {};
    const environment: Record<string, string> = {};
    for (const item of values.ports ?? []) {
      if (item?.host && item?.container) {
        ports[String(item.host)] = String(item.container);
      }
    }
    for (const item of values.environment ?? []) {
      if (item?.key) {
        environment[String(item.key)] = String(item.value ?? "");
      }
    }
    setSaving(true);
    try {
      await adminApi.create("/api-admin/virtualization/containers", {
        name: values.name,
        description: values.description,
        image: values.image,
        host_id: host?.id,
        ports,
        environment,
      });
      message.success("容器创建成功");
      form.resetFields();
      await onSuccess();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "创建容器失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      title="创建容器"
      open={open}
      onCancel={onCancel}
      onOk={() => void submit()}
      confirmLoading={saving}
      width={760}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Row gutter={12}>
          <Col span={12}>
            <Form.Item
              name="name"
              label="容器名称"
              rules={[{ required: true }]}
            >
              <Input placeholder="例如：web-01" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="image" label="镜像" rules={[{ required: true }]}>
              <Select
                showSearch
                placeholder="选择镜像"
                options={images.map((image) => ({
                  label: imageDisplayName(image),
                  value: imageDisplayName(image),
                }))}
              />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="description" label="描述">
          <Input placeholder="补充容器用途" />
        </Form.Item>
        <Divider orientation="left">端口映射</Divider>
        <Form.List name="ports">
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...rest }) => (
                <Space
                  key={key}
                  style={{ display: "flex", marginBottom: 8 }}
                  align="baseline"
                >
                  <Form.Item
                    {...rest}
                    name={[name, "host"]}
                    rules={[{ required: true }]}
                  >
                    <Input placeholder="主机端口" style={{ width: 160 }} />
                  </Form.Item>
                  <Typography.Text>:</Typography.Text>
                  <Form.Item
                    {...rest}
                    name={[name, "container"]}
                    rules={[{ required: true }]}
                  >
                    <Input placeholder="容器端口" style={{ width: 160 }} />
                  </Form.Item>
                  <Button
                    danger
                    type="text"
                    icon={<DeleteOutlined />}
                    onClick={() => remove(name)}
                  />
                </Space>
              ))}
              <Button
                type="dashed"
                icon={<PlusOutlined />}
                onClick={() => add()}
              >
                添加端口
              </Button>
            </>
          )}
        </Form.List>
        <Divider orientation="left">环境变量</Divider>
        <Form.List name="environment">
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...rest }) => (
                <Space
                  key={key}
                  style={{ display: "flex", marginBottom: 8 }}
                  align="baseline"
                >
                  <Form.Item {...rest} name={[name, "key"]}>
                    <Input placeholder="KEY" style={{ width: 180 }} />
                  </Form.Item>
                  <Typography.Text>=</Typography.Text>
                  <Form.Item {...rest} name={[name, "value"]}>
                    <Input placeholder="VALUE" style={{ width: 220 }} />
                  </Form.Item>
                  <Button
                    danger
                    type="text"
                    icon={<DeleteOutlined />}
                    onClick={() => remove(name)}
                  />
                </Space>
              ))}
              <Button
                type="dashed"
                icon={<PlusOutlined />}
                onClick={() => add()}
              >
                添加环境变量
              </Button>
            </>
          )}
        </Form.List>
      </Form>
    </Modal>
  );
}

export function ImagePullModal({
  open,
  host,
  onCancel,
  onSuccess,
}: {
  open: boolean;
  host: RowData;
  onCancel: () => void;
  onSuccess: () => void | Promise<void>;
}) {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await adminApi.action(
        `/api-admin/virtualization/hosts/${host.id}/images/pull`,
        "POST",
        values,
      );
      message.success("镜像拉取请求已提交");
      form.resetFields();
      await onSuccess();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "拉取镜像失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      title="拉取远程镜像"
      open={open}
      onCancel={onCancel}
      onOk={() => void submit()}
      confirmLoading={saving}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="imageName"
          label="镜像名称"
          rules={[{ required: true, message: "请输入镜像名称" }]}
          extra="例如 nginx:latest 或 redis:7"
        >
          <Input placeholder="输入镜像名称" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export function ImageBuildModal({
  open,
  host,
  onCancel,
  onSuccess,
}: {
  open: boolean;
  host: RowData;
  onCancel: () => void;
  onSuccess: () => void | Promise<void>;
}) {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await adminApi.action(
        `/api-admin/virtualization/hosts/${host.id}/images/build`,
        "POST",
        {
          tag: values.tag,
          dockerfile: values.dockerfile,
          buildArgs: values.buildArgs
            ? JSON.parse(values.buildArgs)
            : undefined,
        },
      );
      message.success("镜像构建请求已提交");
      form.resetFields();
      await onSuccess();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "构建镜像失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      title="构建镜像"
      open={open}
      onCancel={onCancel}
      onOk={() => void submit()}
      confirmLoading={saving}
      width={720}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item name="tag" label="镜像标签" rules={[{ required: true }]}>
          <Input placeholder="例如：myapp:1.0.0" />
        </Form.Item>
        <Form.Item
          name="dockerfile"
          label="Dockerfile"
          rules={[{ required: true }]}
        >
          <Input.TextArea
            rows={10}
            placeholder={`FROM alpine:latest\nRUN echo "hello"\nCMD ["sh"]`}
          />
        </Form.Item>
        <Form.Item name="buildArgs" label="构建参数(JSON)">
          <Input.TextArea rows={4} placeholder={`{"VERSION":"1.0.0"}`} />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export function ImageUploadModal({
  open,
  host,
  onCancel,
  onSuccess,
}: {
  open: boolean;
  host: RowData;
  onCancel: () => void;
  onSuccess: () => void | Promise<void>;
}) {
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [file, setFile] = useState<UploadFile | null>(null);

  async function submit() {
    const values = await form.validateFields();
    if (!file?.originFileObj) {
      message.error("请选择镜像文件");
      return;
    }
    const formData = new FormData();
    formData.append("file", file.originFileObj);
    if (values.tag) {
      formData.append("tag", String(values.tag));
    }
    setSaving(true);
    try {
      await adminApi.create(
        `/api-admin/virtualization/hosts/${host.id}/images/load`,
        formData,
      );
      message.success("镜像上传请求已提交");
      form.resetFields();
      setFile(null);
      await onSuccess();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "上传镜像失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      title="上传镜像文件"
      open={open}
      onCancel={onCancel}
      onOk={() => void submit()}
      confirmLoading={saving}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="upload"
          label="镜像文件"
          rules={[{ required: true, message: "请选择镜像文件" }]}
          extra="支持 .tar / .tar.gz"
        >
          <Upload
            beforeUpload={(next) => {
              if (
                !next.name.endsWith(".tar") &&
                !next.name.endsWith(".tar.gz")
              ) {
                message.error("仅支持 .tar 或 .tar.gz");
                return Upload.LIST_IGNORE;
              }
              setFile(next);
              form.setFieldValue("upload", next.name);
              return false;
            }}
            onRemove={() => {
              setFile(null);
              form.setFieldValue("upload", undefined);
            }}
            maxCount={1}
            fileList={file ? [file] : []}
          >
            <Button icon={<FileZipOutlined />}>选择镜像文件</Button>
          </Upload>
        </Form.Item>
        <Form.Item name="tag" label="镜像标签">
          <Input placeholder="例如：myapp:backup" />
        </Form.Item>
      </Form>
    </Modal>
  );
}



