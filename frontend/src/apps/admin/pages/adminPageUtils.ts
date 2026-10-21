export function normalizeRows<T>(result: unknown): T[] {
  if (Array.isArray(result)) {
    return result as T[];
  }

  const object = (result ?? {}) as { data?: T[]; items?: T[]; list?: T[] };
  return object.data ?? object.items ?? object.list ?? [];
}

export function normalizeFilters(filters: Record<string, string>) {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value));
}

export function formatTime(value?: string) {
  if (!value) {
    return "-";
  }

  return new Date(value).toLocaleString();
}
