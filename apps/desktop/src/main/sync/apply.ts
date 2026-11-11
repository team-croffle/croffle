import { decodeRow, SERVER_WRITER_ID, type SyncTableName, type WireRow } from '@croffledev/common';
import { and, eq, ne, type SQL } from 'drizzle-orm';
import type { SQLiteColumn, SQLiteTable } from 'drizzle-orm/sqlite-core';

import { databaseManager, type AppDatabase } from '../database';
import {
  scheduleExceptions,
  scheduleReminders,
  schedules,
  scheduleTags,
  syncOutbox,
  tags,
} from '../database/schema';

type SyncedTable = SQLiteTable & { id: SQLiteColumn; version: SQLiteColumn };

export const SYNCED_TABLES: Record<SyncTableName, SyncedTable> = {
  tag: tags,
  schedule: schedules,
  schedule_tags: scheduleTags,
  schedule_reminder: scheduleReminders,
  schedule_exception: scheduleExceptions,
};

/**
 * Natural keys besides `id`. Two devices can create the same link offline under different ids
 * and the server keeps both, but locally only one row may hold the key. Every device keeps the
 * row with the smallest id, whatever order the rows arrive in (the log and a snapshot order them
 * differently), so all devices converge on the same one.
 */
const NATURAL_KEYS: Partial<Record<SyncTableName, SQLiteColumn[]>> = {
  schedule_tags: [scheduleTags.scheduleId, scheduleTags.tagId],
  schedule_reminder: [scheduleReminders.scheduleId, scheduleReminders.minutes],
  schedule_exception: [scheduleExceptions.scheduleId, scheduleExceptions.occurrenceStart],
};

export type ServerRow = { table: SyncTableName; row: WireRow };

export function isSyncTable(table: string): table is SyncTableName {
  return table in SYNCED_TABLES;
}

/** The local row of one synchronised table, as drizzle reads it. */
export function readLocalRow(
  db: AppDatabase,
  table: SyncTableName,
  rowId: string,
): Record<string, unknown> | undefined {
  const target = SYNCED_TABLES[table];
  const [row] = db.select().from(target).where(eq(target.id, rowId)).all();
  return row as Record<string, unknown> | undefined;
}

/** Returns false when a row with a smaller id already holds the key: the incoming row loses. */
function claimNaturalKey(
  tx: AppDatabase,
  table: SyncTableName,
  local: Record<string, unknown>,
): boolean {
  const keys = NATURAL_KEYS[table];
  if (!keys) {
    return true;
  }
  const target = SYNCED_TABLES[table];
  const conditions: SQL[] = keys.map((column) => eq(column, local[column.name as string]));
  const holders = tx
    .select({ id: target.id })
    .from(target)
    .where(and(ne(target.id, local.id as string), ...conditions))
    .all() as { id: string }[];
  const incomingId = local.id as string;
  if (holders.some((holder) => holder.id < incomingId)) {
    return false;
  }
  for (const holder of holders) {
    tx.delete(target).where(eq(target.id, holder.id)).run();
    tx.delete(syncOutbox)
      .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, holder.id)))
      .run();
  }
  return true;
}

/**
 * Writes rows received from the server. Each is stored as the server holds it, marked
 * `lastWriterId = 'server'` so the outbox triggers skip it, and any queued local entry for it is
 * dropped. Callers decide beforehand whether a local unsynced edit should win instead.
 *
 * Foreign keys are off for the batch: a page can carry a child before its parent, and the parent
 * arrives later in the same log. Nothing here deletes a parent, so no cascade is missed.
 */
export function applyServerRows(rows: ServerRow[]): number {
  if (rows.length === 0) {
    return 0;
  }
  const sqlite = databaseManager.getSqlite();
  const db = databaseManager.getDb();
  let applied = 0;
  sqlite.pragma('foreign_keys = OFF');
  try {
    db.transaction((tx) => {
      for (const { table, row } of rows) {
        const target = SYNCED_TABLES[table];
        const local: Record<string, unknown> = {
          ...decodeRow(table, row),
          lastWriterId: SERVER_WRITER_ID,
        };
        if (!claimNaturalKey(tx, table, local)) {
          continue;
        }
        const { id: _id, ...rest } = local;
        tx.insert(target)
          .values(local as never)
          .onConflictDoUpdate({ target: target.id, set: rest as never })
          .run();
        tx.delete(syncOutbox)
          .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, local.id as string)))
          .run();
        applied++;
      }
    });
  } finally {
    sqlite.pragma('foreign_keys = ON');
  }
  return applied;
}
