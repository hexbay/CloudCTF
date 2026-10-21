import { Card, Space, Typography } from "antd";
import type { ReactNode } from "react";

type AdminDataTableCardProps = {
  title: ReactNode;
  search?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function AdminDataTableCard({
  title,
  search,
  actions,
  children,
  className,
}: AdminDataTableCardProps) {
  return (
    <Card className={`admin-table-card admin-data-table-card${className ? ` ${className}` : ""}`} bordered={false}>
      {search ? <div className="admin-table-search">{search}</div> : null}
      <div className="resource-toolbar">
        <Typography.Title level={5}>{title}</Typography.Title>
        {actions ? <Space wrap>{actions}</Space> : null}
      </div>
      {children}
    </Card>
  );
}
