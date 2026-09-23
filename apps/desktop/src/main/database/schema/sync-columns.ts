import { integer, text } from 'drizzle-orm/sqlite-core';

/**
 * LWW column set for synchronised tables. Must stay in sync with `SyncColumns`
 * in `@croffledev/common` (enforced per table by `assertSchemaMatch`).
 */
export function syncColumns() {
  return {
    updatedAtMs: integer('updatedAtMs').notNull().default(0),
    version: integer('version').notNull().default(1),
    deletedAt: integer('deletedAt', { mode: 'timestamp_ms' }),
    lastWriterId: text('lastWriterId'),
    ownerId: text('ownerId'),
  };
}
