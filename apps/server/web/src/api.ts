// All paths are relative, so the dashboard works wherever the server is
// mounted (CROFFLE_BASE_PATH, or a proxy that forwards a subpath).

export type Me = {
  accountId: string;
  username: string;
  teams: string[];
};

export type Device = {
  id: string;
  name: string;
  createdAt: number;
  lastSeenAt?: number;
  state: 'active' | 'pending' | 'revoked';
};

export type Team = {
  id: string;
  name: string;
  isMember: boolean;
};

export type Row = {
  table: string;
  rowId: string;
  version: number;
  updatedAtMs: number;
  teamId?: string;
  deleted: boolean;
  serverSeq: number;
  payload: string;
};

export type Commit = {
  serverSeq: number;
  table: string;
  rowId: string;
  op: string;
  version: number;
  updatedAtMs: number;
  lastWriterId?: string;
  teamId?: string;
};

export type Health = {
  server: string;
  protocol: { min: number; max: number };
  db: string;
  retention?: { oldestCommitSeq: number; purgedThroughSeq: number };
  time: number;
};

/** Thrown for any non-2xx response; `status` lets the caller tell 401 apart. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    // The session is an HttpOnly cookie; nothing here ever touches a token.
    credentials: 'same-origin',
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  const body = text ? (JSON.parse(text) as unknown) : {};

  if (!response.ok) {
    const error = body as { code?: string; message?: string };
    throw new ApiError(
      response.status,
      error.code ?? 'error',
      error.message ?? `request failed with ${response.status}`,
    );
  }
  return body as T;
}

export const api = {
  health: () => request<Health>('health'),
  me: () => request<Me>('dash/me'),
  login: (username: string, password: string) =>
    request<Me>('dash/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  logout: () => request<void>('dash/logout', { method: 'POST' }),
  devices: () => request<{ devices: Device[] }>('dash/devices'),
  teams: () => request<{ teams: Team[] }>('dash/teams'),
  rows: (afterTable = '', afterRowId = '', limit = 50) =>
    request<{ rows: Row[]; hasMore: boolean; serverSeq: number }>(
      `dash/rows?limit=${limit}&afterTable=${encodeURIComponent(afterTable)}&afterRowId=${encodeURIComponent(afterRowId)}`,
    ),
  commits: (sinceSeq = 0, limit = 50) =>
    request<{ commits: Commit[]; hasMore: boolean; serverSeq: number }>(
      `dash/commits?sinceSeq=${sinceSeq}&limit=${limit}`,
    ),
};

/** Formats a unix-millisecond stamp; the tables show nothing for a missing one. */
export function formatTime(ms?: number): string {
  if (!ms) {
    return '';
  }
  return new Date(ms).toISOString().replace('T', ' ').slice(0, 19);
}
