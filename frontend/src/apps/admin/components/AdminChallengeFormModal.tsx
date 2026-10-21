import { UploadOutlined } from "@ant-design/icons";
import {
  Button,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Switch,
  Typography,
  Upload,
} from "antd";
import type { FormInstance } from "antd/es/form";
import type { UploadFile } from "antd/es/upload/interface";
import { AdminTemplateSelect } from "./AdminTemplateSelect";

export type ChallengeFile = {
  id?: string | number;
  filename?: string;
  file_path?: string;
  url?: string;
  size?: number;
};

type AdminChallengeFormModalProps = {
  open: boolean;
  title: string;
  saving?: boolean;
  width?: number;
  form: FormInstance;
  isDynamic: boolean;
  fileList: UploadFile[];
  existingFiles?: ChallengeFile[];
  onFileListChange: (files: UploadFile[]) => void;
  onRemoveExistingFile?: (file: ChallengeFile) => void;
  onCancel: () => void;
  onOk: () => void;
};

const categoryOptions = [
  { label: "Web", value: "web" },
  { label: "密码学", value: "crypto" },
  { label: "逆向", value: "reverse" },
  { label: "Pwn", value: "pwn" },
  { label: "杂项", value: "misc" },
  { label: "取证", value: "forensics" },
  { label: "移动安全", value: "mobile" },
  { label: "区块链", value: "blockchain" },
  { label: "其他", value: "other" },
];

const difficultyOptions = [
  { label: "简单", value: 1 },
  { label: "较简单", value: 2 },
  { label: "中等", value: 3 },
  { label: "较难", value: 4 },
  { label: "困难", value: 5 },
];

const timeoutOptions = [
  { label: "1 小时", value: 3600 },
  { label: "2 小时", value: 7200 },
  { label: "4 小时", value: 14400 },
  { label: "8 小时", value: 28800 },
  { label: "24 小时", value: 86400 },
];

export function AdminChallengeFormModal({
  open,
  title,
  saving = false,
  width = 820,
  form,
  isDynamic,
  fileList,
  existingFiles = [],
  onFileListChange,
  onRemoveExistingFile,
  onCancel,
  onOk,
}: AdminChallengeFormModalProps) {
  return (
    <Modal
      title={title}
      open={open}
      width={width}
      className="admin-form-modal challenge-form-modal"
      onCancel={onCancel}
      onOk={onOk}
      confirmLoading={saving}
      destroyOnClose
    >
      <Form
        form={form}
        layout="vertical"
        className="admin-compact-form challenge-config-form"
      >
        <div className="challenge-form-grid challenge-form-grid-single">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: "请输入名称" }]}
          >
            <Input placeholder="请输入题目名称" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: "请输入描述" }]}
          >
            <Input.TextArea rows={4} placeholder="请输入题目描述" />
          </Form.Item>
        </div>

        <div className="challenge-form-grid">
          <Form.Item
            name="category"
            label="分类"
            rules={[{ required: true, message: "请选择分类" }]}
          >
            <Select placeholder="请选择分类" options={categoryOptions} />
          </Form.Item>
          <Form.Item
            name="points"
            label="分值"
            rules={[{ required: true, message: "请输入分值" }]}
          >
            <InputNumber min={0} max={10000} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            name="difficulty"
            label="难度"
            rules={[{ required: true, message: "请选择难度" }]}
          >
            <Select placeholder="请选择难度" options={difficultyOptions} />
          </Form.Item>
        </div>

        <div className="challenge-form-grid challenge-form-grid-switches">
          <Form.Item
            name="is_dynamic"
            label="Flag 类型"
            valuePropName="checked"
          >
            <Switch
              checkedChildren="动态"
              unCheckedChildren="静态"
              onChange={(checked) => {
                if (!checked) {
                  form.setFieldsValue({
                    host_id: undefined,
                    template_id: undefined,
                  });
                }
              }}
            />
          </Form.Item>
          <Form.Item name="is_active" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="is_public" label="公开" valuePropName="checked">
            <Switch />
          </Form.Item>
        </div>

        {!isDynamic ? (
          <Form.Item
            name="flag"
            label="Flag"
            rules={[{ required: true, message: "请输入 Flag" }]}
          >
            <Input placeholder="例如：flag{this_is_a_flag}" />
          </Form.Item>
        ) : (
          <div className="challenge-form-grid">
            <Form.Item
              name="template_id"
              label="环境模板"
              rules={[{ required: true, message: "请选择环境模板" }]}
            >
              <AdminTemplateSelect
                onChange={(templateId) => {
                  form.setFieldValue("template_id", templateId);
                }}
                onHostChange={(hostId) => {
                  form.setFieldValue("host_id", hostId);
                }}
              />
            </Form.Item>
            <Form.Item name="container_timeout" label="实例生命周期">
              <Select placeholder="请选择时长" options={timeoutOptions} />
            </Form.Item>
          </div>
        )}

        <Form.Item label="附件上传">
          {existingFiles.length > 0 ? (
            <>
              <Typography.Text strong>已有附件</Typography.Text>
              <div style={{ marginTop: 12, marginBottom: 12 }}>
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {existingFiles.map((file) => (
                    <li
                      key={String(
                        file.id ?? file.url ?? file.file_path ?? file.filename,
                      )}
                      style={{ marginBottom: 8 }}
                    >
                      <span
                        style={{
                          display: "inline-flex",
                          gap: 8,
                          alignItems: "center",
                          flexWrap: "wrap",
                        }}
                      >
                        <a
                          href={file.url ?? file.file_path ?? "#"}
                          target="_blank"
                          rel="noreferrer"
                        >
                          {file.filename ||
                            file.url ||
                            file.file_path ||
                            `附件 ${file.id ?? ""}`}
                        </a>
                        {onRemoveExistingFile ? (
                          <Popconfirm
                            title="删除这个附件？"
                            okText="删除"
                            cancelText="取消"
                            onConfirm={() => onRemoveExistingFile(file)}
                          >
                            <Button danger size="small" type="text">
                              删除
                            </Button>
                          </Popconfirm>
                        ) : null}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
              <Divider style={{ margin: "12px 0" }} />
            </>
          ) : null}

          <Upload.Dragger
            multiple
            fileList={fileList}
            beforeUpload={(file) => {
              onFileListChange([...fileList, file]);
              return false;
            }}
            onRemove={(file) => {
              onFileListChange(
                fileList.filter((item) => item.uid !== file.uid),
              );
            }}
          >
            <p className="ant-upload-drag-icon">
              <UploadOutlined />
            </p>
            <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
            <p className="ant-upload-hint">题目保存成功后自动上传附件。</p>
          </Upload.Dragger>
        </Form.Item>
      </Form>
    </Modal>
  );
}
