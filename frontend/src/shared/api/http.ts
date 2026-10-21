export type RequestOptions = {
  method?: string;
  headers?: HeadersInit;
  body?: unknown;
  rawBody?: BodyInit;
  query?: Record<string, string | number | boolean | undefined | null>;
};

export const API_ERROR_EVENT = "cloudctf-api-error";

type ErrorPayload = {
  success?: boolean;
  detail?: unknown;
  message?: unknown;
  error?: unknown;
  errors?: unknown;
  code?: unknown;
};

export class ApiError extends Error {
  status?: number;
  payload?: ErrorPayload;
  code?: string;
  handled?: boolean;

  constructor(
    message: string,
    options?: { status?: number; payload?: ErrorPayload; code?: string },
  ) {
    super(message);
    this.name = "ApiError";
    this.status = options?.status;
    this.payload = options?.payload;
    this.code = options?.code;
  }
}

export async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const url = withQuery(path, options.query);
  const headers = new Headers(options.headers);
  const token = localStorage.getItem("cloudctf_token");

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  let body: BodyInit | undefined = options.rawBody;
  if (options.rawBody) {
    // Headers are supplied by the caller for raw bodies.
  } else if (options.body instanceof FormData) {
    body = options.body;
  } else if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.body);
  }

  const response = await fetch(url, {
    method: options.method ?? "GET",
    headers,
    body,
  });

  if (!response.ok) {
    if (response.status === 401 && isPlayerApiRequest(path)) {
      clearPlayerAuthState();
      window.dispatchEvent(
        new CustomEvent("player-auth-changed", { detail: { loggedIn: false } }),
      );
      window.dispatchEvent(
        new CustomEvent("show-player-auth", { detail: "login" }),
      );
    }
    throw scheduleGlobalErrorNotification(await buildApiError(response));
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = (await response.json()) as T & ErrorPayload;
  if (isLogicalFailure(payload)) {
    throw scheduleGlobalErrorNotification(
      new ApiError(resolveErrorMessage(payload, response.statusText), {
        status: response.status,
        payload,
        code: normalizeCode(payload.code),
      }),
    );
  }
  return payload as T;
}

function withQuery(path: string, query?: RequestOptions["query"]) {
  if (!query) {
    return path;
  }
  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      params.set(key, String(value));
    }
  });
  const suffix = params.toString();
  return suffix ? `${path}?${suffix}` : path;
}

export function getErrorMessage(error: unknown, fallback = "请求失败") {
  if (error instanceof ApiError) {
    error.handled = true;
    return error.message || fallback;
  }
  if (error instanceof Error) {
    return error.message || fallback;
  }
  if (typeof error === "string" && error.trim()) {
    return error.trim();
  }
  return fallback;
}

function scheduleGlobalErrorNotification(error: ApiError) {
  window.setTimeout(() => {
    if (error.handled) {
      return;
    }
    window.dispatchEvent(
      new CustomEvent(API_ERROR_EVENT, {
        detail: { message: error.message || "请求失败", status: error.status },
      }),
    );
  }, 0);
  return error;
}

async function buildApiError(response: Response) {
  try {
    const payload = (await response.json()) as ErrorPayload;
    return new ApiError(resolveErrorMessage(payload, response.statusText), {
      status: response.status,
      payload,
      code: normalizeCode(payload.code),
    });
  } catch {
    return new ApiError(response.statusText || "请求失败", {
      status: response.status,
    });
  }
}

function isLogicalFailure(payload: ErrorPayload | undefined) {
  if (!payload) return false;
  // 显式标记 success: false
  if (payload.success === false) return true;
  // FastAPI 风格错误响应：仅含 detail 且未显式标记 success 时视作业务失败
  if (payload.detail !== undefined && payload.success === undefined) return true;
  return false;
}

function resolveErrorMessage(
  payload: ErrorPayload | undefined,
  fallback: string,
) {
  if (!payload) {
    return fallback || "请求失败";
  }
  const candidate =
    payload.detail ??
    payload.message ??
    payload.error ??
    flattenErrors(payload.errors);
  const text = normalizeText(candidate);
  return text || fallback || "请求失败";
}

function flattenErrors(value: unknown) {
  if (Array.isArray(value)) {
    return value.map(normalizeText).filter(Boolean).join("; ");
  }
  if (value && typeof value === "object") {
    return Object.values(value as Record<string, unknown>)
      .map(normalizeText)
      .filter(Boolean)
      .join("; ");
  }
  return value;
}

function normalizeText(value: unknown) {
  if (typeof value === "string") {
    return value.trim();
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function normalizeCode(value: unknown) {
  const text = normalizeText(value);
  return text || undefined;
}

function isPlayerApiRequest(path: string) {
  return path.startsWith("/api/") && !path.startsWith("/api-admin/");
}

function clearPlayerAuthState() {
  localStorage.removeItem("cloudctf_token");
  localStorage.removeItem("cloudctf_refresh_token");
  localStorage.removeItem("cloudctf_user");
}
