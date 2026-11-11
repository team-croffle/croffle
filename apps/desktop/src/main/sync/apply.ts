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
 * Natural keys besides `id`. Two devices can create the same link offline under different ids;
 * the server keeps both, and every device applies them in the same order, so dropping the local
 * row that holds the key converges everywhere on the last one applied.
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

function dropNaturalKeyHolder(
  tx: AppDatabase,
  table: SyncTableName,
  local: Record<string, unknown>,
): void {
  const keys = NATURAL_KEYS[table];
  if (!keys) {
    return;
  }
  const target = SYNCED_TABLES[table];
  const conditions: SQL[] = keys.map((column) => eq(column, local[column.name as string]));
  const holders = tx
    .select({ id: target.id })
    .from(target)
    .where(and(ne(target.id, local.id as string), ...conditions))
    .all() as { id: string }[];
  for (const holder of holders) {
    tx.delete(target).where(eq(target.id, holder.id)).run();
    tx.delete(syncOutbox)
      .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, holder.id)))
      .run();
  }
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
  sqlite.pragma('foreign_keys = OFF');
  try {
    db.transaction((tx) => {
      for (const { table, row } of rows) {
        const target = SYNCED_TABLES[table];
        const local: Record<string, unknown> = {
          ...decodeRow(table, row),
          lastWriterId: SERVER_WRITER_ID,
        };
        dropNaturalKeyHolder(tx, table, local);
        const { id: _id, ...rest } = local;
        tx.insert(target)
          .values(local as never)
          .onConflictDoUpdate({ target: target.id, set: rest as never })
          .run();
        tx.delete(syncOutbox)
          .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, local.id as string)))
          .run();
      }
    });
  } finally {
    sqlite.pragma('foreign_keys = ON');
  }
  return rows.length;
}
