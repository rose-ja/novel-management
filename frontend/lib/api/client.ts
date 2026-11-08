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

export async function fetchJson<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${basePath}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        ...init?.headers,
      },
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

  return (await response.json()) as T;
}
