import { color } from "echarts";
import type { SectionMeta } from "./types";

export const hostTypeOptions = [
  { label: "Docker", value: "docker" },
  { label: "Proxmox VE", value: "pve" },
];

export const repoTypeOptions = [
  { label: "CTF", value: "ctf" },
  { label: "漏洞", value: "vul" },
];

export const booleanStatusMeta = {
  enabled: {
    true: { label: "启用", color: "green" },
    false: { label: "停用", color: "default" },
  },
} as const;

export const statusMeta: Record<string, { label: string; color: string }> = {
  pending: { label: "待处理", color: "default" },
  online: { label: "在线", color: "green" },
  running: { label: "运行中", color: "green" },
  completed: { label: "已就绪", color: "green" },
  ready: { label: "就绪", color: "green" },
  offline: { label: "离线", color: "default" },
  stopped: { label: "已停止", color: "default" },
  paused: { label: "已暂停", color: "gold" },
  exited: { label: "已退出", color: "default" },
  creating: { label: "创建中", color: "blue" },
  restarting: { label: "重启中", color: "blue" },
  building: { label: "构建中", color: "blue" },
  error: { label: "错误", color: "red" },
  failed: { label: "失败", color: "red" },
};

export const templateStatusOptions = [
  { label: "未就绪", value: "pending", color: "default" },
  { label: "已就绪", value: "completed", color: "green" },
  { label: "编译失败", value: "error", color: "red" },
];
export const ContainerGroupStatus = [
  { label: "运行中", value: "running", color: "green" },
  { label: "已停止", value: "stopped", color: "default" },
  { label: "已暂停", value: "paused", color: "gold" },
];

export const statusColor: Record<string, string> = Object.fromEntries(
  Object.entries(statusMeta).map(([k, v]) => [k, v.color]),
);

export const sectionMeta: SectionMeta = {
  hosts: { title: "主机管理", breadcrumb: "管理 / 虚拟化管理 / 主机管理" },
  containers: {
    title: "容器管理",
    breadcrumb: "管理 / 虚拟化管理 / 容器管理",
  },
  groups: { title: "容器组", breadcrumb: "管理 / 虚拟化管理 / 容器组" },
  templates: { title: "模板管理", breadcrumb: "管理 / 虚拟化管理 / 模板管理" },
  "host-detail": {
    title: "主机详情",
    breadcrumb: "管理 / 虚拟化管理 / 主机详情",
  },
  "host-images": {
    title: "镜像管理",
    breadcrumb: "管理 / 虚拟化管理 / 镜像管理",
  },
  "host-containers": {
    title: "容器管理",
    breadcrumb: "管理 / 虚拟化管理 / 主机容器管理",
  },
};
