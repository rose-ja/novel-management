export type ApiErrorBody = {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
    request_id?: string;
  };
};

/** ApiError 保留稳定错误码与后端中文文案，供页面按 code 决定展示与重试。 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly requestId?: string;
  readonly details?: unknown;

  constructor(options: {
    code: string;
    message: string;
    status: number;
    requestId?: string;
    details?: unknown;
  }) {
    super(options.message);
    this.name = "ApiError";
    this.code = options.code;
    this.status = options.status;
    this.requestId = options.requestId;
    this.details = options.details;
  }
}

/** 网络层失败（断网、DNS、服务未启动）统一映射为 NETWORK_ERROR。 */
export class NetworkError extends Error {
  constructor() {
    super("网络异常，请检查连接");
    this.name = "NetworkError";
  }
}

const basePath = "";

export type ApiMethod = "GET" | "POST" | "PATCH" | "DELETE";

export type ApiRequestOptions = {
  method?: ApiMethod;
  /** 自动 JSON 序列化；传 undefined 表示无请求体。 */
  body?: unknown;
  /** 乐观锁版本，写入 `If-Match: "<version>"`。 */
  ifMatch?: string | number;
  signal?: AbortSignal;
};

async function perform<T>(path: string, init: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${basePath}${path}`, {
      ...init,
      cache: "no-store",
    });
  } catch {
    throw new NetworkError();
  }

  if (!response.ok) {
    let body: ApiErrorBody | null = null;
    try {
      body = (await response.json()) as ApiErrorBody;
    } catch {
      body = null;
    }
    throw new ApiError({
      code: body?.error?.code ?? `HTTP_${response.status}`,
      message: body?.error?.message ?? "请求失败，请稍后重试",
      status: response.status,
      requestId: body?.error?.request_id,
      details: body?.error?.details,
    });
  }

  // 204/205 与空响应体没有 JSON 可解析，返回 undefined。
  if (response.status === 204 || response.status === 205) {
    return undefined as T;
  }
  const text = await response.text();
  if (!text) {
    return undefined as T;
  }
  try {
    return JSON.parse(text) as T;
  } catch {
    return undefined as T;
  }
}

function buildInit(options: ApiRequestOptions): RequestInit {
  const method = options.method ?? "GET";
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  if (method !== "GET") {
    // 本机服务用 X-Requested-With 识别同源脚本请求，降低跨站表单提交风险。
    headers["X-Requested-With"] = "XMLHttpRequest";
  }
  let body: string | undefined;
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  if (options.ifMatch !== undefined) {
    headers["If-Match"] = `"${options.ifMatch}"`;
  }
  return { method, headers, body, signal: options.signal };
}

/** 保留简化签名：只读请求。 */
export function fetchJson<T>(path: string, init?: RequestInit): Promise<T> {
  return perform<T>(path, {
    ...init,
    headers: {
      Accept: "application/json",
      ...init?.headers,
    },
  });
}

/** 写请求统一入口：JSON、`X-Requested-With`、按需 `If-Match`。 */
export function requestJson<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  return perform<T>(path, buildInit(options));
}

/** 无响应体（204）的写请求。 */
export async function requestVoid(path: string, options: ApiRequestOptions = {}): Promise<void> {
  await perform<unknown>(path, buildInit(options));
}
