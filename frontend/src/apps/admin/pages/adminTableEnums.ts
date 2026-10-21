export const competitionTypeOptions = [
  { label: "CTF", value: "ctf" },
  { label: "AWD", value: "awd" },
  { label: "理论赛", value: "theory" },
  { label: "混合赛", value: "mixed" },
] as const;

export const competitionStatusOptions = [
  { label: "草稿", value: "draft" },
  { label: "未开始", value: "upcoming" },
  { label: "进行中", value: "ongoing" },
  { label: "已结束", value: "finished" },
  { label: "已归档", value: "archived" },
] as const;

export const challengeCategoryOptions = [
  { label: "Web", value: "web" },
  { label: "密码学", value: "crypto" },
  { label: "逆向", value: "reverse" },
  { label: "Pwn", value: "pwn" },
  { label: "杂项", value: "misc" },
  { label: "取证", value: "forensics" },
  { label: "移动安全", value: "mobile" },
  { label: "区块链", value: "blockchain" },
  { label: "其他", value: "other" },
] as const;

export const challengeDifficultyOptions = [
  { label: "简单", value: 1 },
  { label: "较简单", value: 2 },
  { label: "中等", value: 3 },
  { label: "较难", value: 4 },
  { label: "困难", value: 5 },
] as const;

export const vulnSeverityOptions = [
  { label: "严重", value: "CRITICAL" },
  { label: "高危", value: "HIGH" },
  { label: "中危", value: "MEDIUM" },
  { label: "低危", value: "LOW" },
  { label: "未知", value: "UNKNOWN" },
] as const;

export const theoryQuestionTypeOptions = [
  { label: "问答题", value: "text" },
  { label: "判断题", value: "judge" },
  { label: "单选题", value: "single" },
  { label: "多选题", value: "multiple" },
] as const;

export const virtualizationStatusMeta: Record<string, { label: string; color: string }> = {
  online: { label: "在线", color: "green" },
  running: { label: "运行中", color: "green" },
  pending: { label: "未就绪", color: "default" },
  ready: { label: "就绪", color: "green" },
  offline: { label: "离线", color: "default" },
  stopped: { label: "已停止", color: "default" },
  paused: { label: "已暂停", color: "gold" },
  exited: { label: "已退出", color: "default" },
  creating: { label: "创建中", color: "blue" },
  restarting: { label: "重启中", color: "blue" },
  building: { label: "构建中", color: "blue" },
  completed: { label: "已就绪", color: "green" },
  error: { label: "错误", color: "red" },
  failed: { label: "失败", color: "red" },
};

export const hostTypeOptions = [
  { label: "Docker", value: "docker" },
  { label: "Proxmox VE", value: "pve" },
] as const;

export const templateTypeOptions = [
  { label: "CTF", value: "ctf" },
  { label: "漏洞", value: "vul" },
] as const;

export const repositoryTypeOptions = [
  { label: "CTF", value: "ctf" },
  { label: "漏洞", value: "vul" },
] as const;

export const providerOptions = [
  { label: "Git", value: "git" },
  { label: "本地", value: "local" },
] as const;

export function findOptionLabel<T extends string | number>(
  options: ReadonlyArray<{ label: string; value: T }>,
  value: T | string | number | undefined | null,
) {
  return options.find((item) => item.value === value)?.label ?? String(value ?? "");
}
