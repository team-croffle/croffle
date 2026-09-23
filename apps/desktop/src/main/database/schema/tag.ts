import { assertSchemaMatch, type AssertSchema, type TagEntity } from '@croffledev/common';
import { integer, sqliteTable, text } from 'drizzle-orm/sqlite-core';

import { syncColumns } from './sync-columns';

/**
 * `name` is intentionally not UNIQUE at the DB level: two devices may create the
 * same tag name before syncing. Uniqueness among live rows is enforced in `calendar/tag.ts`.
 */
export const tags = sqliteTable('tag', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  color: text('color').notNull(),
  createdAt: integer('createdAt', { mode: 'timestamp' }).notNull(),
  updatedAt: integer('updatedAt', { mode: 'timestamp' }).notNull(),
  ...syncColumns(),
});

export type TagRow = typeof tags.$inferSelect;
export type NewTag = typeof tags.$inferInsert;

assertSchemaMatch<AssertSchema<TagRow, TagEntity>>();
