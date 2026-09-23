import { assertSchemaMatch, type AssertSchema, type ExtensionInfoEntity } from '@croffledev/common';
import { integer, sqliteTable, text } from 'drizzle-orm/sqlite-core';

/**
 * Extension install registry. Manifest fields (`engines`, `contributes`, …)
 * are read from `croffle-manifest.json` on disk, not duplicated here.
 * Device-local: audit timestamps only, no sync columns.
 */
export const extensionInfo = sqliteTable('extension_info', {
  id: text('id').primaryKey(),
  name: text('name').notNull(),
  version: text('version').notNull(),
  author: text('author').notNull(),
  description: text('description'),
  enabled: integer('enabled', { mode: 'boolean' }).notNull().default(true),
  main: text('main'),
  installedAt: integer('installedAt', { mode: 'timestamp' }).notNull(),
  updatedAt: integer('updatedAt', { mode: 'timestamp' }).notNull(),
});

export type ExtensionInfoRow = typeof extensionInfo.$inferSelect;
export type NewExtensionInfo = typeof extensionInfo.$inferInsert;

assertSchemaMatch<AssertSchema<ExtensionInfoRow, ExtensionInfoEntity>>();
