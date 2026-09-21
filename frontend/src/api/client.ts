// 后端接口的统一请求封装。
//
// 后端所有响应都是 { code, message, data, timestamp } 信封结构，
// 这里统一拆包并抛出带中文提示的 ApiError，页面只需要处理成功数据。

const BASE_PATH = '/api/v1';

export interface Envelope<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
}

export class ApiError extends Error {
  readonly code: number;
  readonly status: number;

  constructor(message: string, code: number, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

async function unwrap<T>(response: Response): Promise<T> {
  const text = await response.text();
  if (!text) {
    throw new ApiError('后端返回了空响应，请稍后重试', -1, response.status);
  }

  let envelope: Envelope<T>;
  try {
    envelope = JSON.parse(text) as Envelope<T>;
  } catch {
    throw new ApiError(`后端返回的内容无法解析（HTTP ${response.status}）`, -1, response.status);
  }

  if (!response.ok || envelope.code !== 0) {
    const message = envelope.message || `请求失败（HTTP ${response.status}）`;
    throw new ApiError(message, envelope.code, response.status);
  }
  return envelope.data;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${BASE_PATH}${path}`, {
      headers: { 'Content-Type': 'application/json' },
      ...init
    });
  } catch {
    throw new ApiError('无法连接后端服务，请确认后端已启动', -1, 0);
  }
  return unwrap<T>(response);
}

export const http = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' })
};

type QueryValue = string | number | boolean | undefined | null;

/** 把查询条件拼成查询串，自动跳过空值。 */
export function buildQuery(params: Record<string, QueryValue>): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') {
      return;
    }
    search.set(key, String(value));
  });
  const query = search.toString();
  return query ? `?${query}` : '';
}

/** 把任意异常转换成可以直接展示的中文提示。 */
export function toErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return '发生未知错误，请稍后重试';
}
