import { Select, Spin, message } from "antd";
import { useEffect, useState } from "react";
import { adminApi } from "@shared/api/admin";

type TemplateOption = {
  id: number | string;
  name: string;
  host_id?: number | string;
  host_name?: string;
  status?: string;
};

type AdminTemplateSelectProps = {
  disabled?: boolean;
  onHostChange?: (hostId?: number) => void;
  onChange?: (value?: number) => void;
  placeholder?: string;
  value?: number | string;
};

export function AdminTemplateSelect({
  disabled,
  onHostChange,
  onChange,
  placeholder,
  value,
}: AdminTemplateSelectProps) {
  const [options, setOptions] = useState<TemplateOption[]>([]);
  const [loading, setLoading] = useState(false);
  const normalizedValue = normalizeNumber(value);

  useEffect(() => {
    void loadOptions();
  }, []);

  useEffect(() => {
    if (!normalizedValue) {
      return;
    }
    const exists = options.some(
      (option) => normalizeNumber(option.id) === normalizedValue,
    );
    if (!exists) {
      void loadCurrentOption(normalizedValue);
    }
  }, [normalizedValue, options]);

  async function loadOptions() {
    setLoading(true);
    try {
      const result = await adminApi.list<TemplateOption>("/api-admin/virtualization/templates/options", {
        current: 1,
        pageSize: 500,
      });
      setOptions(normalizeRows<TemplateOption>(result));
    } catch (error) {
      message.error(error instanceof Error ? error.message : "加载模板失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadCurrentOption(templateId: number) {
    try {
      const result = await adminApi.list<TemplateOption>(
        `/api-admin/virtualization/templates/${templateId}`,
      );
      setOptions((current) => {
        if (
          current.some(
            (option) => normalizeNumber(option.id) === templateId,
          )
        ) {
          return current;
        }
        return [...current, result as TemplateOption];
      });
    } catch {
      // Ignore detail fallback failure and keep raw value rendering.
    }
  }

  return (
    <Select
      showSearch
      disabled={disabled}
      value={normalizedValue}
      placeholder={placeholder ?? "请选择环境模板"}
      optionFilterProp="label"
      loading={loading}
      notFoundContent={loading ? <Spin size="small" /> : null}
      options={options.map((option) => ({
        value: normalizeNumber(option.id),
        label: formatTemplateLabel(option),
        raw: option,
      }))}
      onChange={(nextValue, option) => {
        onChange?.(normalizeNumber(nextValue));
        const selected = Array.isArray(option) ? option[0] : option;
        const raw = selected?.raw as TemplateOption | undefined;
        onHostChange?.(normalizeNumber(raw?.host_id));
      }}
    />
  );
}

function formatTemplateLabel(option: TemplateOption): string {
  if (option.host_name && option.name) {
    return `${option.host_name} / ${option.name}`;
  }
  return option.host_name || option.name || String(option.id);
}

function normalizeNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return undefined;
    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

function normalizeRows<T>(result: unknown): T[] {
  if (Array.isArray(result)) {
    return result as T[];
  }
  const object = (result ?? {}) as { data?: T[]; items?: T[]; list?: T[] };
  return object.data ?? object.items ?? object.list ?? [];
}
