import { randomUUID } from 'node:crypto';

import type { SyncTableName } from '@croffledev/common';
import { and, asc, count, eq, gte, lt, ne } from 'drizzle-orm';

import { databaseManager } from '../database';
import { syncOutbox, type SyncOutboxRow } from '../database/schema';

/** What a push sent for one entry: enough to tell whether the row moved on meanwhile. */
export type SentEntry = Pick<SyncOutboxRow, 'opId' | 'rowVersion'>;

/** After this many failed attempts an entry waits until the row is edited again (resets to 0). */
export const MAX_ATTEMPTS = 5;

/** The oldest queued changes that are still worth sending, in the order they were first queued. */
export function peekBatch(limit: number): SyncOutboxRow[] {
  return databaseManager
    .getDb()
    .select()
    .from(syncOutbox)
    .where(lt(syncOutbox.attempts, MAX_ATTEMPTS))
    .orderBy(asc(syncOutbox.seq))
    .limit(limit)
    .all();
}

/** Entries that stopped being retried; the settings screen shows them as failed. */
export function countFailed(): number {
  const [row] = databaseManager
    .getDb()
    .select({ n: count() })
    .from(syncOutbox)
    .where(gte(syncOutbox.attempts, MAX_ATTEMPTS))
    .all();
  return row?.n ?? 0;
}

export function countPending(): number {
  const [row] = databaseManager.getDb().select({ n: count() }).from(syncOutbox).all();
  return row?.n ?? 0;
}

/**
 * Settles entries the server accepted. An entry whose row was edited again while the push was
 * in flight stays queued: the server now holds the version that was sent, so that becomes its
 * `baseVersion`, and it gets a fresh `opId` — reusing the old one would make the server replay
 * the earlier result and drop the newer edit.
 */
export function ack(sent: SentEntry[]): void {
  const db = databaseManager.getDb();
  db.transaction((tx) => {
    for (const entry of sent) {
      tx.delete(syncOutbox)
        .where(and(eq(syncOutbox.opId, entry.opId), eq(syncOutbox.rowVersion, entry.rowVersion)))
        .run();
      tx.update(syncOutbox)
        .set({ opId: randomUUID(), baseVersion: entry.rowVersion, attempts: 0, lastError: null })
        .where(and(eq(syncOutbox.opId, entry.opId), ne(syncOutbox.rowVersion, entry.rowVersion)))
        .run();
    }
  });
}

/** Records a failed attempt; the entry stays queued. */
export function fail(opId: string, message: string): void {
  const db = databaseManager.getDb();
  const [row] = db.select().from(syncOutbox).where(eq(syncOutbox.opId, opId)).all();
  if (!row) {
    return;
  }
  db.update(syncOutbox)
    .set({ attempts: row.attempts + 1, lastError: message })
    .where(eq(syncOutbox.opId, opId))
    .run();
}

/** Drops the entry of one row (the server's version won, or the row needs no push). */
export function remove(tableName: SyncTableName, rowId: string): void {
  databaseManager
    .getDb()
    .delete(syncOutbox)
    .where(and(eq(syncOutbox.tableName, tableName), eq(syncOutbox.rowId, rowId)))
    .run();
}

/** The queued entry of one row, if any. */
export function entryFor(tableName: SyncTableName, rowId: string): SyncOutboxRow | undefined {
  const [row] = databaseManager
    .getDb()
    .select()
    .from(syncOutbox)
    .where(and(eq(syncOutbox.tableName, tableName), eq(syncOutbox.rowId, rowId)))
    .all();
  return row;
}
