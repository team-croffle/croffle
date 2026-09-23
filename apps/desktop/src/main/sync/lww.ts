import { isNull, sql, type SQL } from 'drizzle-orm';
import type { SQLiteColumn } from 'drizzle-orm/sqlite-core';

import { deviceWriterId } from './client-id';

type SyncTable = {
  version: SQLiteColumn;
  deletedAt: SQLiteColumn;
};

/** Values every local write stamps on a synchronised row. */
export function touch(now: Date = new Date()) {
  return {
    updatedAtMs: now.getTime(),
    lastWriterId: deviceWriterId(),
  };
}

/** `version + 1` for an UPDATE / upsert `set`. */
export function bumpVersion(table: SyncTable): SQL<number> {
  return sql<number>`${table.version} + 1`;
}

/** Soft delete: the row stays, hidden by `alive()`. */
export function tombstone(table: SyncTable, now: Date = new Date()) {
  return {
    deletedAt: now,
    version: bumpVersion(table),
    ...touch(now),
  };
}

/** Bring a tombstoned row back (same identity, new version). */
export function revive(table: SyncTable, now: Date = new Date()) {
  return {
    deletedAt: null,
    version: bumpVersion(table),
    ...touch(now),
  };
}

/** `deletedAt IS NULL` — every read of a synchronised table goes through this. */
export function alive(table: SyncTable): SQL {
  return isNull(table.deletedAt);
}
