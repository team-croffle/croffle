import {
  assertSchemaMatch,
  type AssertSchema,
  type ScheduleExternalLinkEntity,
} from '@croffledev/common';
import { relations } from 'drizzle-orm';
import { index, integer, sqliteTable, text, uniqueIndex } from 'drizzle-orm/sqlite-core';

import { schedules } from './schedule';
import { syncColumns } from './sync-columns';

/**
 * Link between a local schedule and an event of an external provider
 * (Google Calendar extension, CalDAV, …). Provider-neutral; no service uses it yet.
 * `(provider, accountId, externalId)` is unique so two devices importing the
 * same external event resolve to one row.
 */
export const scheduleExternalLinks = sqliteTable(
  'schedule_external_link',
  {
    id: text('id').primaryKey(),
    scheduleId: text('scheduleId')
      .notNull()
      .references(() => schedules.id, { onDelete: 'cascade' }),
    provider: text('provider').notNull(),
    accountId: text('accountId').notNull(),
    externalId: text('externalId').notNull(),
    externalEtag: text('externalEtag'),
    externalUpdatedAtMs: integer('externalUpdatedAtMs'),
    ...syncColumns(),
  },
  (t) => [
    uniqueIndex('schedule_external_link_provider_account_external_unique').on(
      t.provider,
      t.accountId,
      t.externalId,
    ),
    index('schedule_external_link_scheduleId_idx').on(t.scheduleId),
  ],
);

export const scheduleExternalLinksRelations = relations(scheduleExternalLinks, ({ one }) => ({
  schedule: one(schedules, {
    fields: [scheduleExternalLinks.scheduleId],
    references: [schedules.id],
  }),
}));

export type ScheduleExternalLinkRow = typeof scheduleExternalLinks.$inferSelect;
export type NewScheduleExternalLink = typeof scheduleExternalLinks.$inferInsert;

assertSchemaMatch<AssertSchema<ScheduleExternalLinkRow, ScheduleExternalLinkEntity>>();
