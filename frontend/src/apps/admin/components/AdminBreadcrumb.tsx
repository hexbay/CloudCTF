import { Breadcrumb } from "antd";
import { Link, useLocation } from "react-router-dom";

const pathLabel: Record<string, string> = {
  dashboard: "数据概览",
  challenge: "题目管理",
  "vuln-lab": "漏洞列表",
  "practice-submissions": "提交记录",
  competition: "比赛管理",
  users: "用户管理",
  admins: "管理员管理",
  teams: "队伍管理",
  roles: "角色管理",
  theory: "理论题库",
  "theory-exams": "理论考试",
  "theory-submissions": "答题记录",
  announcements: "公告管理",
  notifications: "通知中心",
  audit: "审计日志",
  settings: "系统设置",
  virtualization: "虚拟化管理",
  "virtualization-containers": "容器管理",
  "virtualization-groups": "容器组管理",
  "virtualization-templates": "模板管理",
  "resource-repositories": "资源仓库",
};

// 扁平路径的父子映射，key 为子路径，value 为完整面包屑片段
const pathExpand: Record<string, string[]> = {
  "virtualization-containers": ["virtualization", "virtualization-containers"],
  "virtualization-groups": ["virtualization", "virtualization-groups"],
  "virtualization-templates": ["virtualization", "virtualization-templates"],
  "resource-repositories": ["virtualization", "resource-repositories"],
  "practice-submissions": ["practice-submissions"], // 已在练习场子菜单下，无父级路径段
};

function segmentLabel(segment: string): string {
  if (pathLabel[segment]) return pathLabel[segment];
  if (/^\d+$/.test(segment)) return segment;
  return decodeURIComponent(segment);
}

export function AdminBreadcrumb() {
  const { pathname } = useLocation();
  const raw = pathname.replace(/^\/admin\/?/, "").split("/").filter(Boolean);

  // 展开扁平路径
  const segments: string[] = [];
  for (const seg of raw) {
    const expanded = pathExpand[seg];
    if (expanded) {
      segments.push(...expanded);
    } else {
      segments.push(seg);
    }
  }

  const items = segments.map((seg, i) => {
    const path = `/admin/${segments.slice(0, i + 1).join("/")}`;
    const last = i === segments.length - 1;
    return {
      title: last ? segmentLabel(seg) : <Link to={path}>{segmentLabel(seg)}</Link>,
    };
  });

  return <Breadcrumb items={items} className="admin-breadcrumb" />;
}
