import type { ReactNode } from "react";

export type RowData = Record<string, any>;
export type PageState = { current: number; pageSize: number };
export type HostDetailMode = "overview" | "images" | "containers";
export type VirtualizationSection =
  | "hosts"
  | "containers"
  | "groups"
  | "templates"
  | "host-detail"
  | "host-images"
  | "host-containers";
export type Field = {
  name: string;
  label: string;
  type?: "text" | "number" | "textarea" | "switch";
  required?: boolean;
};
export type Operation = {
  label: string;
  path: string;
  danger?: boolean;
  body?: (row: RowData) => RowData;
};
export type SortOrder = "ascend" | "descend" | null;

export type SectionMeta = Record<
  VirtualizationSection,
  { title: string; breadcrumb: string }
>;

export type SummaryCardProps = {
  title: string;
  value: ReactNode;
  description?: ReactNode;
};
