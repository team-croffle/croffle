// All paths are relative, so the dashboard works wherever the server is
// mounted (CROFFLE_BASE_PATH, or a proxy that forwards a subpath).

export type Me = {
  accountId: string;
  username: string;
  teams: string[];
  otpEnabled: boolean;
  /** Shows the operator screens; the server checks it again on every call. */
  operator: boolean;
};

export type AdminAccount = {
  id: string;
  username: string;
  createdAt: number;
  disabledAt?: number;
  operator: boolean;
  otp: 'off' | 'pending' | 'enabled';
  deviceCount: number;
};

export type AdminDevice = Device & { accountId: string; username: string };

export type AdminTeam = { id: string; name: string; createdAt: number; memberCount: number };

export type TeamMember = { accountId: string; username: string; role: string; createdAt: number };

export type IssuedToken = {
  deviceId: string;
  accountId: string;
  /** Shown once: the server keeps only its hash. */
  token: string;
  tokenExpiresAt?: number;
};

export type ServerInfo = {
  schemaVersion: number;
  path: string;
  sizeBytes: number;
  accounts: number;
  devices: number;
  teams: number;
  rows: number;
  commits: number;
  oldestSeq: number;
  newestSeq: number;
  purgedThroughSeq: number;
  secretKey: 'set' | 'missing';
  otpSecretsPlain: number;
  otpSecretsEncrypted: number;
};

export type AdminCommit = Commit & { ownerId: string };

export type OtpEnrolment = {
  secret: string;
  otpauthUri: string;
  /** data: URL of the QR code, rendered by the server. */
  qrPng: string;
};

export type Device = {
  id: string;
  name: string;
  createdAt: number;
  lastSeenAt?: number;
  /** Absent for a device token that never expires. */
  tokenExpiresAt?: number;
  state: 'active' | 'pending' | 'expired' | 'revoked';
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
  features?: string[];
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
  login: (username: string, password: string, otp = '') =>
    request<Me>('dash/login', {
      method: 'POST',
      body: JSON.stringify({ username, password, otp: otp || undefined }),
    }),
  logout: () => request<void>('dash/logout', { method: 'POST' }),
  devices: () => request<{ devices: Device[] }>('dash/devices'),
  revokeDevice: (id: string) =>
    request<void>(`dash/devices/${encodeURIComponent(id)}/revoke`, { method: 'POST' }),
  teams: () => request<{ teams: Team[] }>('dash/teams'),
  otpEnroll: () => request<OtpEnrolment>('dash/otp/enroll', { method: 'POST', body: '{}' }),
  otpConfirm: (code: string) =>
    request<{ recoveryCodes: string[] }>('dash/otp/confirm', {
      method: 'POST',
      body: JSON.stringify({ code }),
    }),
  otpDisable: (password: string, code: string) =>
    request<void>('dash/otp/disable', { method: 'POST', body: JSON.stringify({ password, code }) }),
  /** Changes the signed-in account's own password; other browsers are signed out. */
  changePassword: (currentPassword: string, newPassword: string, code = '') =>
    request<void>('dash/me/password', {
      method: 'POST',
      body: JSON.stringify({ currentPassword, newPassword, code: code || undefined }),
    }),
  rows: (afterTable = '', afterRowId = '', limit = 50) =>
    request<{ rows: Row[]; hasMore: boolean; serverSeq: number }>(
      `dash/rows?limit=${limit}&afterTable=${encodeURIComponent(afterTable)}&afterRowId=${encodeURIComponent(afterRowId)}`,
    ),
  commits: (sinceSeq = 0, limit = 50) =>
    request<{ commits: Commit[]; hasMore: boolean; serverSeq: number }>(
      `dash/commits?sinceSeq=${sinceSeq}&limit=${limit}`,
    ),
};

const post = (body: unknown = {}): RequestInit => ({ method: 'POST', body: JSON.stringify(body) });
const id = (value: string) => encodeURIComponent(value);

/** The operator API (/dash/admin/); every call answers 403 for a regular account. */
export const admin = {
  accounts: () => request<{ accounts: AdminAccount[] }>('dash/admin/accounts'),
  createAccount: (username: string, password: string, operator: boolean) =>
    request<{ account: AdminAccount }>(
      'dash/admin/accounts',
      post({ username, password, operator }),
    ),
  setDisabled: (accountId: string, disabled: boolean) =>
    request<void>(
      `dash/admin/accounts/${id(accountId)}/${disabled ? 'disable' : 'enable'}`,
      post(),
    ),
  setPassword: (accountId: string, password: string) =>
    request<void>(`dash/admin/accounts/${id(accountId)}/password`, post({ password })),
  disableOtp: (accountId: string) =>
    request<void>(`dash/admin/accounts/${id(accountId)}/otp/disable`, post()),
  setOperator: (accountId: string, operator: boolean) =>
    request<void>(`dash/admin/accounts/${id(accountId)}/operator`, post({ operator })),
  devices: (account = '') =>
    request<{ devices: AdminDevice[] }>(
      `dash/admin/devices${account ? `?account=${id(account)}` : ''}`,
    ),
  issueToken: (username: string, name: string, noExpiry: boolean, expiresIn: string) =>
    request<IssuedToken>(
      'dash/admin/devices',
      post({ username, name, noExpiry, expiresIn: noExpiry ? '' : expiresIn }),
    ),
  revokeDevice: (deviceId: string) =>
    request<void>(`dash/admin/devices/${id(deviceId)}/revoke`, post()),
  teams: () => request<{ teams: AdminTeam[] }>('dash/admin/teams'),
  createTeam: (name: string) => request<{ team: AdminTeam }>('dash/admin/teams', post({ name })),
  renameTeam: (teamId: string, name: string) =>
    request<void>(`dash/admin/teams/${id(teamId)}`, post({ name })),
  members: (teamId: string) =>
    request<{ members: TeamMember[] }>(`dash/admin/teams/${id(teamId)}/members`),
  addMember: (teamId: string, username: string) =>
    request<void>(`dash/admin/teams/${id(teamId)}/members`, post({ username })),
  removeMember: (teamId: string, accountId: string) =>
    request<void>(`dash/admin/teams/${id(teamId)}/members/${id(accountId)}`, {
      method: 'DELETE',
    }),
  info: () => request<ServerInfo>('dash/admin/info'),
  vacuum: () => request<{ sizeBefore: number; sizeAfter: number }>('dash/admin/vacuum', post()),
  commits: (sinceSeq = 0, limit = 50) =>
    request<{ commits: AdminCommit[]; hasMore: boolean; serverSeq: number }>(
      `dash/admin/commits?sinceSeq=${sinceSeq}&limit=${limit}`,
    ),
};

/** Formats a byte count for the server tab. */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KiB`;
  }
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
}

/** Formats a unix-millisecond stamp; the tables show nothing for a missing one. */
export function formatTime(ms?: number): string {
  if (!ms) {
    return '';
  }
  return new Date(ms).toISOString().replace('T', ' ').slice(0, 19);
}
