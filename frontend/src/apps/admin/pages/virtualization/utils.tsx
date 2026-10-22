import { ApiOutlined, CloudServerOutlined, LinuxOutlined, WindowsOutlined } from "@ant-design/icons";
import { Input, InputNumber, Space, Switch, Tag, Typography } from "antd";
import type { Field, RowData } from "./types";
import { statusMeta, templateStatusOptions } from "./constants";

const templateTypeColors: Record<string, string> = {
  ctf: "blue",
  vul: "volcano",
};

const templateStatusColors: Record<string, string> = {
  pending: "gold",
  completed: "green",
  ready: "green",
  running: "green",
  online: "green",
  building: "blue",
  creating: "blue",
  restarting: "blue",
  paused: "gold",
  error: "red",
  failed: "red",
};

export function renderField(field: Field) {
  if (field.type === "number") return <InputNumber style={{ width: "100%" }} />;
  if (field.type === "textarea")
    return <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} />;
  if (field.type === "switch") return <Switch />;
  return <Input />;
}

export function normalizePayload(values: RowData, fields: Field[]) {
  const payload: RowData = {};
  fields.forEach((field) => {
    const value = values[field.name];
    if (value === undefined || value === "") return;
    if (
      field.type === "textarea" &&
      typeof value === "string" &&
      ["ports", "environment", "volumes", "resources"].includes(field.name)
    ) {
      try {
        payload[field.name] = JSON.parse(value);
      } catch {
        payload[field.name] = value;
      }
      return;
    }
    payload[field.name] = value;
  });
  return payload;
}

export function normalizePage(result: unknown) {
  if (Array.isArray(result))
    return { rows: result as RowData[], total: result.length };
  const object = (result ?? {}) as {
    data?: RowData[];
    items?: RowData[];
    list?: RowData[];
    total?: number;
  };
  const rows = object.data ?? object.items ?? object.list ?? [];
  return { rows, total: object.total ?? rows.length };
}

export function normalizeRows(result: unknown) {
  return normalizePage(result).rows;
}

export const enumLabels: Record<string, Record<string, string>> = {
  status: Object.fromEntries(
    Object.entries(statusMeta).map(([k, v]) => [k, v.label]),
  ),
  template_type: { ctf: "CTF", vul: "漏洞" },
  repo_type: { ctf: "CTF", vul: "漏洞" },
  host_type: { docker: "Docker", pve: "Proxmox VE" },
  provider: { git: "Git", local: "本地" },
};

export function renderValue(value: unknown, key?: string) {
  if (value === null || value === undefined || value === "") {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  if (key === "status") {
    const s = String(value);
    const option = templateStatusOptions.find((item) => item.value === s);
    const meta = statusMeta[s];
    return (
      <Tag
        color={templateStatusColors[s] ?? meta?.color ?? "default"}
        className="template-meta-tag"
      >
        {option?.label ?? meta?.label ?? s}
      </Tag>
    );
  }
  if (key === "ports") {
    return renderPortList(value);
  }
  if (key && enumLabels[key]) {
    const label = enumLabels[key][String(value)];
    const color =
      key === "template_type" || key === "repo_type"
        ? templateTypeColors[String(value)]
        : undefined;
    return (
      <Tag color={color} className={color ? "template-meta-tag" : undefined}>
        {label ?? String(value)}
      </Tag>
    );
  }
  if (typeof value === "boolean") {
    return <Tag color={value ? "green" : "default"}>{value ? "是" : "否"}</Tag>;
  }
  if (Array.isArray(value)) {
    if (value.length === 0)
      return <Typography.Text type="secondary">-</Typography.Text>;
    return (
      <Space size={[6, 6]} wrap className="admin-tag-cluster">
        {value.map((v, i) => (
          <Tag key={i} className="admin-soft-tag">
            {String(v)}
          </Tag>
        ))}
      </Space>
    );
  }
  if (typeof value === "object") {
    return <Typography.Text code>{JSON.stringify(value)}</Typography.Text>;
  }
  const str = String(value);
  const d = new Date(str);
  if (!isNaN(d.getTime()) && /\d{4}[-/]\d{2}[-/]\d{2}/.test(str)) {
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  }
  return str;
}

export function containerStatusLabel(value: unknown) {
  const status = String(value ?? "").trim().toLowerCase();
  return statusMeta[status]?.label ?? (status || "-");
}

export function renderContainerStatus(value: unknown) {
  const status = String(value ?? "").trim().toLowerCase();
  return (
    <Tag color={statusMeta[status]?.color ?? "default"}>
      {containerStatusLabel(status)}
    </Tag>
  );
}

export function renderHostIcon(host: RowData) {
  const resources = host.resources ?? {};
  const osType = String(resources.os_type ?? "").toLowerCase();
  const osText = String(resources.os ?? "").toLowerCase();
  if (osType.includes("windows") || osText.includes("windows")) {
    return <WindowsOutlined />;
  }
  if (
    osType.includes("linux") ||
    osText.includes("linux") ||
    osText.includes("ubuntu") ||
    osText.includes("debian")
  ) {
    return <LinuxOutlined />;
  }
  if (host.host_type === "docker") {
    return <ApiOutlined />;
  }
  return <CloudServerOutlined />;
}

export function renderArchitecture(host: RowData) {
  const architecture = host?.resources?.architecture;
  return architecture ? <Tag>{String(architecture)}</Tag> : null;
}

export function hostErrorText(host?: RowData | null, status?: unknown) {
  const currentStatus = String(status ?? host?.status ?? "");
  if (currentStatus === "online") {
    return "主机在线";
  }
  const endpoint = String(host?.endpoint ?? "");
  const runtimeError = host?.resources?.error;
  if (runtimeError) {
    return String(runtimeError);
  }
  if (endpoint.startsWith("npipe://")) {
    return "该地址仅适用于 Docker Desktop 或 Windows Docker Engine。当前如果使用 WSL + dockerd，请改用 tcp://127.0.0.1:2375 之类的 TCP 监听地址。";
  }
  if (endpoint.startsWith("unix://")) {
    return "该地址仅适用于后端进程运行在 Linux / WSL 内部。当前如果后端运行在 Windows，本地 Unix Socket 无法直接访问。";
  }
  if (endpoint.startsWith("tcp://")) {
    return "请确认 WSL 中的 dockerd 已监听该 TCP 地址，并允许当前 Windows 侧后端访问。";
  }
  return currentStatus === "error"
    ? "连接失败，请检查 Docker daemon 是否可达。"
    : "主机离线。";
}

export function formatBytes(value: unknown) {
  const size = Number(value ?? 0);
  if (!Number.isFinite(size) || size <= 0) return "-";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let current = size;
  let index = 0;
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024;
    index += 1;
  }
  return `${current.toFixed(current >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}

export function formatDateTime(value: unknown) {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  if (typeof value === "number") {
    const date = new Date(value > 1e12 ? value : value * 1000);
    return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString();
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return "-";
    if (/^\d+$/.test(trimmed)) {
      const numeric = Number(trimmed);
      const date = new Date(numeric > 1e12 ? numeric : numeric * 1000);
      return Number.isNaN(date.getTime()) ? trimmed : date.toLocaleString();
    }
    const date = new Date(trimmed);
    return Number.isNaN(date.getTime()) ? trimmed : date.toLocaleString();
  }
  return String(value);
}

export function normalizeTimestamp(value: unknown) {
  if (value === null || value === undefined || value === "") return 0;
  if (typeof value === "number") {
    return value > 1e12 ? value : value * 1000;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return 0;
    if (/^\d+$/.test(trimmed)) {
      const numeric = Number(trimmed);
      return numeric > 1e12 ? numeric : numeric * 1000;
    }
    const parsed = new Date(trimmed).getTime();
    return Number.isNaN(parsed) ? 0 : parsed;
  }
  return 0;
}

export function renderPortSummary(value: unknown) {
  if (!value) return <Typography.Text type="secondary">-</Typography.Text>;
  const pairs = portEntries(value);
  if (!pairs.length)
    return <Typography.Text type="secondary">-</Typography.Text>;
  return (
    <Space size={[4, 4]} wrap>
      {pairs.slice(0, 2).map(([hostPort, containerPort]) => (
        <Tag
          key={`${hostPort}-${containerPort}`}
        >{`${hostPort}:${containerPort}`}</Tag>
      ))}
      {pairs.length > 2 ? (
        <Typography.Text type="secondary">+{pairs.length - 2}</Typography.Text>
      ) : null}
    </Space>
  );
}

export function renderPortList(value: unknown) {
  const pairs = portEntries(value);
  if (!pairs.length)
    return <Typography.Text type="secondary">-</Typography.Text>;
  return (
    <Space direction="vertical" size={4}>
      {pairs.map(([hostPort, containerPort]) => (
        <Typography.Text key={`${hostPort}-${containerPort}`}>
          {hostPort} → {containerPort}
        </Typography.Text>
      ))}
    </Space>
  );
}

export function portEntries(value: unknown): Array<[string, string]> {
  if (!value) return [];
  if (Array.isArray(value)) {
    return value.flatMap((item) => portEntryFromObject(item));
  }
  if (typeof value !== "object") return [];
  return Object.entries(value as Record<string, unknown>).flatMap(
    ([containerPort, published]) => {
      if (Array.isArray(published)) {
        return published.flatMap((binding) =>
          portEntryFromObject(binding, containerPort),
        );
      }
      if (published && typeof published === "object") {
        return portEntryFromObject(published, containerPort);
      }
      const hostPort = scalarText(published);
      return hostPort ? [[hostPort, containerPort]] : [];
    },
  );
}

function portEntryFromObject(
  value: unknown,
  fallbackContainerPort?: string,
): Array<[string, string]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [];
  }
  const row = value as RowData;
  const hostPort = firstText(
    row.host_port,
    row.hostPort,
    row.published,
    row.public_port,
    row.port,
    row.HostPort,
  );
  const containerPort = normalizeContainerPort(
    firstText(
      row.container_port,
      row.containerPort,
      row.target,
      row.private_port,
      row.PrivatePort,
      fallbackContainerPort,
    ),
    firstText(row.protocol, row.type, row.Proto),
  );
  if (!hostPort && !containerPort) return [];
  return [[hostPort || "-", containerPort || "-"]];
}

function normalizeContainerPort(port: string, protocol: string) {
  if (!port) return "";
  if (port.includes("/") || !protocol) return port;
  return `${port}/${protocol}`;
}

function firstText(...values: unknown[]) {
  for (const value of values) {
    const text = scalarText(value);
    if (text) return text;
  }
  return "";
}

function scalarText(value: unknown) {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

export function imageDisplayName(image: RowData) {
  const repository = String(image.repository ?? "");
  const tag = String(image.tag ?? "");
  if (repository) {
    return `${repository}:${tag || "<none>"}`;
  }
  if (Array.isArray(image.tags) && image.tags.length > 0) {
    return String(image.tags[0] ?? "<none>:<none>");
  }
  if (tag && tag.includes(":")) {
    return tag;
  }
  return String(image.tag ?? image.id ?? "<none>");
}

export function imageShortId(image: RowData) {
  const imageId = String(image.image_id ?? image.id ?? "");
  return imageId.replace("sha256:", "").slice(0, 12);
}

export function splitImageName(name: string) {
  const parts = name.split(":");
  if (parts.length < 2) {
    return { repository: name, tag: "<none>" };
  }
  return {
    repository: parts.slice(0, -1).join(":"),
    tag: parts.at(-1) ?? "<none>",
  };
}

export function columnLabel(key: string) {
  const labels: Record<string, string> = {
    id: "ID",
    name: "名称",
    image: "镜像",
    status: "状态",
    host_id: "主机 ID",
    host_type: "主机类型",
    endpoint: "连接地址",
    out_ip: "出口 IP",
    ports: "端口",
    template_type: "模板类型",
    tags: "标签",
    repo_type: "仓库类型",
    provider: "来源",
    repository_url: "仓库地址",
    repository_branch: "分支",
    enabled: "启用",
    competition_id: "赛事 ID",
    challenge_id: "题目 ID",
    user_id: "用户 ID",
    created_at: "创建时间",
    updated_at: "更新时间",
    last_run_at: "上次运行",
  };
  return labels[key] ?? key;
}


