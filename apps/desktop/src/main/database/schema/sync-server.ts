import { integer, sqliteTable, text } from 'drizzle-orm/sqlite-core';

/**
 * The sync server this device is bound to (one row, `id = 'default'`). Device-local:
 * no sync columns, never pushed. The token lives in `safeStorage`, not here.
 */
export const syncServer = sqliteTable('sync_server', {
  id: text('id').primaryKey(),
  url: text('url').notNull(),
  accountId: text('accountId'),
  teamId: text('teamId'),
  allowInsecureTls: integer('allowInsecureTls', { mode: 'boolean' }).notNull().default(false),
  enabled: integer('enabled', { mode: 'boolean' }).notNull().default(false),
  /** Pull cursor: the last `serverSeq` applied. */
  lastPulledSeq: integer('lastPulledSeq').notNull().default(0),
  /** Opaque cursor of a snapshot that is still being fetched; null when none is in progress. */
  bootstrapCursor: text('bootstrapCursor'),
  /** Millisecond time the last snapshot finished. Null until the first bootstrap completes. */
  bootstrappedAt: integer('bootstrappedAt'),
  /** Millisecond time pre-existing rows were queued for the first push. */
  backfilledAt: integer('backfilledAt'),
  lastSyncedAt: integer('lastSyncedAt'),
  lastErrorKind: text('lastErrorKind'),
  lastError: text('lastError'),
});

export type SyncServerRow = typeof syncServer.$inferSelect;
export type NewSyncServer = typeof syncServer.$inferInsert;
