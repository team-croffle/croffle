import {
  decodeRow,
  SERVER_WRITER_ID,
  wins,
  type SyncTableName,
  type WireRow,
} from '@croffledev/common';
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
 * differently), so all devices converge on the same one. An unsent local edit that is newer than
 * the row taking its place is carried over onto it rather than lost.
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

type Claim = {
  /** False when a row with a smaller id holds the key: the incoming row loses and is not stored. */
  apply: boolean;
  /** A displaced local row whose unsent edit is newer: carried over onto the incoming row. */
  carry?: Record<string, unknown>;
};

function dropRow(tx: AppDatabase, table: SyncTableName, rowId: string): void {
  const target = SYNCED_TABLES[table];
  tx.delete(target).where(eq(target.id, rowId)).run();
  tx.delete(syncOutbox)
    .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, rowId)))
    .run();
}

function claimNaturalKey(
  tx: AppDatabase,
  table: SyncTableName,
  incoming: Record<string, unknown>,
): Claim {
  const keys = NATURAL_KEYS[table];
  if (!keys) {
    return { apply: true };
  }
  const target = SYNCED_TABLES[table];
  const incomingId = incoming.id as string;
  const conditions: SQL[] = keys.map((column) => eq(column, incoming[column.name as string]));
  const holders = tx
    .select()
    .from(target)
    .where(and(ne(target.id, incomingId), ...conditions))
    .all() as Record<string, unknown>[];

  if (holders.some((holder) => (holder.id as string) < incomingId)) {
    // A local copy of the losing row (under an older key) would leave this device with a row a
    // freshly bootstrapped device does not have.
    dropRow(tx, table, incomingId);
    return { apply: false };
  }

  let carry: Record<string, unknown> | undefined;
  for (const holder of holders) {
    const pending = tx
      .select({ seq: syncOutbox.seq })
      .from(syncOutbox)
      .where(and(eq(syncOutbox.tableName, table), eq(syncOutbox.rowId, holder.id as string)))
      .all();
    const holderKey = {
      updatedAtMs: holder.updatedAtMs as number,
      lastWriterId: holder.lastWriterId as string | null,
    };
    const incomingKey = {
      updatedAtMs: incoming.updatedAtMs as number,
      lastWriterId: incoming.lastWriterId as string | null,
    };
    if (pending.length > 0 && wins(holderKey, incomingKey) === 'a') {
      carry = holder;
    }
    dropRow(tx, table, holder.id as string);
  }
  return { apply: true, carry };
}

/**
 * Re-applies a displaced row's unsent edit to the row that now holds the key, as a local write:
 * the update trigger queues it with the incoming version as its base, so it reaches the server.
 */
function carryOver(
  tx: AppDatabase,
  table: SyncTableName,
  incoming: Record<string, unknown>,
  carry: Record<string, unknown>,
): void {
  const target = SYNCED_TABLES[table];
  const keyNames = new Set((NATURAL_KEYS[table] ?? []).map((column) => column.name as string));
  const fields: Record<string, unknown> = {};
  for (const [name, value] of Object.entries(carry)) {
    if (name === 'id' || keyNames.has(name) || name === 'version' || name === 'ownerId') {
      continue;
    }
    fields[name] = value;
  }
  fields.version = (incoming.version as number) + 1;
  tx.update(target)
    .set(fields as never)
    .where(eq(target.id, incoming.id as string))
    .run();
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
        const claim = claimNaturalKey(tx, table, local);
        if (!claim.apply) {
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
        if (claim.carry) {
          carryOver(tx, table, local, claim.carry);
        }
        applied++;
      }
    });
  } finally {
    sqlite.pragma('foreign_keys = ON');
  }
  return applied;
}
