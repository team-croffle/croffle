// The sync wire contract shared with croffle-server (apps/server/internal/protocol). Both sides
// are pinned by packages/sync-fixtures; when they disagree, one of them has a bug.
//
// Self-contained (no imports): the Node test runner loads this file without the bundler's aliases.

/** Wire version this client speaks. Checked against `GET /health` `protocol.min..max`. */
export const PROTOCOL_VERSION = 1;

export const HEADER_VERSION = 'X-Croffle-Sync-Version';
export const HEADER_CLIENT_ID = 'X-Croffle-Client';

/** Commits per push request (the server accepts up to 500). */
export const PUSH_BATCH_SIZE = 100;

export type SyncOp = 'upsert' | 'delete';

/** A synchronised row on the wire: column names, dates in milliseconds, booleans as 0/1. */
export type WireRow = Record<string, string | number | null>;

export type Commit = {
  opId: string;
  table: string;
  rowId: string;
  op: SyncOp;
  row?: WireRow;
  baseVersion: number;
  version: number;
  updatedAtMs: number;
  lastWriterId: string;
};

export type PushRequest = { clientId: string; commits: Commit[] };

export type Conflict = {
  opId: string;
  table: string;
  rowId: string;
  serverRow?: WireRow;
  serverVersion: number;
  serverUpdatedAtMs: number;
  serverLastWriterId?: string;
  serverRowDeleted?: boolean;
};

export type PushResponse = { accepted: string[]; conflicts: Conflict[]; serverSeq: number };

export type Change = {
  serverSeq: number;
  table: string;
  rowId: string;
  op: SyncOp;
  row?: WireRow;
  version: number;
  updatedAtMs: number;
  lastWriterId?: string;
};

export type PullRequest = { lastServerSeq: number; limit?: number };
export type PullResponse = { changes: Change[]; serverSeq: number; hasMore: boolean };

export type BootstrapRequest = { cursor?: string; limit?: number };
export type BootstrapResponse = {
  rows: Change[];
  serverSeq: number;
  cursor?: string;
  hasMore: boolean;
};

export type LoginRequest = {
  username: string;
  password: string;
  otp?: string;
  clientId: string;
  deviceName?: string;
};
export type LoginResponse = {
  token: string;
  accountId: string;
  teamId?: string | null;
  expiresAt?: number | null;
};
export type WhoAmIResponse = { accountId: string; teamId?: string | null };

export type HealthResponse = {
  server: string;
  commit?: string;
  protocol: { min: number; max: number };
  db: string;
  retention?: { oldestCommitSeq: number; purgedThroughSeq: number };
  time: number;
};

/** Error body of every non-2xx response. */
export type ErrorBody = { code: string; message?: string };

/** Error codes the client branches on. */
export const ERROR_CODES = {
  unauthorized: 'unauthorized',
  otpRequired: 'otp_required',
  accountDisabled: 'account_disabled',
  tooManyAttempts: 'too_many_attempts',
  invalidCommit: 'invalid_commit',
  foreignTeam: 'foreign_team',
  cursorTooOld: 'cursor_too_old',
  invalidCursor: 'invalid_cursor',
} as const;

// --- row codec --------------------------------------------------------------------------------

/**
 * How a column is stored locally:
 * - `text` / `int` — as is
 * - `sec` — drizzle `mode: 'timestamp'` (a `Date`, second resolution in SQLite)
 * - `ms` — drizzle `mode: 'timestamp_ms'` (a `Date`, millisecond resolution)
 * - `bool` — drizzle `mode: 'boolean'` (stored 0/1)
 * On the wire every date is milliseconds and every boolean is 0/1.
 */
export type ColumnKind = 'text' | 'int' | 'sec' | 'ms' | 'bool';

/** `key` is the drizzle field, `column` the SQLite column, which is also the wire name. */
export type ColumnSpec = { key: string; column: string; kind: ColumnKind };

const col = (key: string, kind: ColumnKind, column = key): ColumnSpec => ({ key, column, kind });

const SYNC_COLUMN_SPECS: ColumnSpec[] = [
  col('updatedAtMs', 'int'),
  col('version', 'int'),
  col('deletedAt', 'ms'),
  col('lastWriterId', 'text'),
  col('ownerId', 'text'),
];

/** Columns of every synchronised table, in schema order. */
export const TABLE_COLUMNS: Record<string, ColumnSpec[]> = {
  tag: [
    col('id', 'text'),
    col('name', 'text'),
    col('color', 'text'),
    col('createdAt', 'sec'),
    col('updatedAt', 'sec'),
    ...SYNC_COLUMN_SPECS,
  ],
  schedule: [
    col('id', 'text'),
    col('title', 'text'),
    col('description', 'text'),
    col('location', 'text'),
    col('startDate', 'sec'),
    col('endDate', 'sec'),
    col('isAllDay', 'bool'),
    col('recurrenceRule', 'text', 'recurringRule'),
    col('colorLabel', 'text'),
    col('priority', 'text'),
    col('useDefaultReminder', 'bool'),
    col('createdAt', 'sec'),
    col('updatedAt', 'sec'),
    ...SYNC_COLUMN_SPECS,
  ],
  schedule_tags: [
    col('id', 'text'),
    col('scheduleId', 'text'),
    col('tagId', 'text'),
    ...SYNC_COLUMN_SPECS,
  ],
  schedule_reminder: [
    col('id', 'text'),
    col('scheduleId', 'text'),
    col('minutes', 'int'),
    ...SYNC_COLUMN_SPECS,
  ],
  schedule_exception: [
    col('id', 'text'),
    col('scheduleId', 'text'),
    col('occurrenceStart', 'ms'),
    col('startDate', 'ms'),
    col('endDate', 'ms'),
    col('cancelled', 'bool'),
    ...SYNC_COLUMN_SPECS,
  ],
};

/** A local row as drizzle reads and writes it. */
export type LocalRow = Record<string, unknown>;

export class RowCodecError extends Error {}

function columnsOf(table: string): ColumnSpec[] {
  const columns = TABLE_COLUMNS[table];
  if (!columns) {
    throw new RowCodecError(`Unknown sync table: ${table}`);
  }
  return columns;
}

function toMs(value: unknown, spec: ColumnSpec): number {
  if (value instanceof Date) {
    return value.getTime();
  }
  throw new RowCodecError(`${spec.key} must be a Date`);
}

function encodeValue(value: unknown, spec: ColumnSpec): string | number | null {
  if (value === null || value === undefined) {
    return null;
  }
  switch (spec.kind) {
    case 'text':
      if (typeof value !== 'string') {
        throw new RowCodecError(`${spec.key} must be a string`);
      }
      return value;
    case 'int':
      if (typeof value !== 'number' || !Number.isInteger(value)) {
        throw new RowCodecError(`${spec.key} must be an integer`);
      }
      return value;
    case 'sec':
      // Second-resolution columns: drop sub-second parts so a round trip is exact.
      return Math.floor(toMs(value, spec) / 1000) * 1000;
    case 'ms':
      return toMs(value, spec);
    case 'bool':
      if (typeof value !== 'boolean') {
        throw new RowCodecError(`${spec.key} must be a boolean`);
      }
      return value ? 1 : 0;
  }
}

function decodeValue(value: unknown, spec: ColumnSpec): unknown {
  if (value === null || value === undefined) {
    return null;
  }
  switch (spec.kind) {
    case 'text':
      if (typeof value !== 'string') {
        throw new RowCodecError(`${spec.column} must be a string`);
      }
      return value;
    case 'int':
    case 'sec':
    case 'ms':
    case 'bool': {
      if (typeof value !== 'number' || !Number.isFinite(value)) {
        throw new RowCodecError(`${spec.column} must be a number`);
      }
      if (spec.kind === 'int') {
        return value;
      }
      if (spec.kind === 'bool') {
        return value !== 0;
      }
      return new Date(value);
    }
  }
}

/** Local row → wire row. Unknown local fields are not sent. */
export function encodeRow(table: string, row: LocalRow): WireRow {
  const wire: WireRow = {};
  for (const spec of columnsOf(table)) {
    wire[spec.column] = encodeValue(row[spec.key], spec);
  }
  return wire;
}

/**
 * Wire row → local row. Fields the server adds that have no local column (such as `teamId`
 * before sharing exists) are dropped.
 */
export function decodeRow(table: string, wire: WireRow): LocalRow {
  const row: LocalRow = {};
  for (const spec of columnsOf(table)) {
    row[spec.key] = decodeValue(wire[spec.column], spec);
  }
  return row;
}

// --- last writer wins -------------------------------------------------------------------------

export type LwwKey = { updatedAtMs: number; lastWriterId: string | null | undefined };

/**
 * The rule both sides use to settle a conflict without talking: the later wall clock wins; on a
 * tie the larger `lastWriterId` wins, so `'server'` beats any `'device:…'`. A full tie keeps `a`.
 */
export function wins(a: LwwKey, b: LwwKey): 'a' | 'b' {
  if (a.updatedAtMs !== b.updatedAtMs) {
    return a.updatedAtMs > b.updatedAtMs ? 'a' : 'b';
  }
  const writerA = a.lastWriterId ?? '';
  const writerB = b.lastWriterId ?? '';
  if (writerA === writerB) {
    return 'a';
  }
  return writerA > writerB ? 'a' : 'b';
}
