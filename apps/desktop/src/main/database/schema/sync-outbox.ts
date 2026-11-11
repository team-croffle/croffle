import { integer, sqliteTable, text, uniqueIndex } from 'drizzle-orm/sqlite-core';

/**
 * Local changes waiting to be pushed. Filled by triggers on the synchronised tables, so a
 * row change and its outbox entry share one statement. One entry per row: later edits fold
 * into it and keep its `opId` and `baseVersion`.
 */
export const syncOutbox = sqliteTable(
  'sync_outbox',
  {
    seq: integer('seq').primaryKey({ autoIncrement: true }),
    opId: text('opId').notNull(),
    tableName: text('tableName').notNull(),
    rowId: text('rowId').notNull(),
    op: text('op', { enum: ['upsert', 'delete'] }).notNull(),
    /** Version the server is believed to hold; 0 for a row it has never seen. */
    baseVersion: integer('baseVersion').notNull(),
    rowVersion: integer('rowVersion').notNull(),
    updatedAtMs: integer('updatedAtMs').notNull(),
    enqueuedAt: integer('enqueuedAt').notNull(),
    attempts: integer('attempts').notNull().default(0),
    lastError: text('lastError'),
  },
  (t) => [
    uniqueIndex('sync_outbox_tableName_rowId_unique').on(t.tableName, t.rowId),
    uniqueIndex('sync_outbox_opId_unique').on(t.opId),
  ],
);

export type SyncOutboxRow = typeof syncOutbox.$inferSelect;
export type NewSyncOutbox = typeof syncOutbox.$inferInsert;
