import {
  assertSchemaMatch,
  type AssertSchema,
  type ScheduleEntity,
  type ScheduleTagEntity,
} from '@croffledev/common';
import { relations } from 'drizzle-orm';
import { integer, sqliteTable, text, uniqueIndex } from 'drizzle-orm/sqlite-core';

import { scheduleExternalLinks } from './schedule-external-link';
import { scheduleReminders, type ScheduleReminderRow } from './schedule-reminder';
import { syncColumns } from './sync-columns';
import { tags, type TagRow } from './tag';

export const schedules = sqliteTable('schedule', {
  id: text('id').primaryKey(),
  title: text('title').notNull(),
  description: text('description'),
  location: text('location'),
  startDate: integer('startDate', { mode: 'timestamp' }).notNull(),
  endDate: integer('endDate', { mode: 'timestamp' }).notNull(),
  isAllDay: integer('isAllDay', { mode: 'boolean' }).notNull().default(false),
  recurrenceRule: text('recurringRule'),
  colorLabel: text('colorLabel').notNull().default('#E1E1E1'),
  priority: text('priority', { enum: ['low', 'medium', 'high'] })
    .notNull()
    .default('medium'),
  /** true = follow notifications.defaultReminderMinutes; false = use schedule_reminder rows */
  useDefaultReminder: integer('useDefaultReminder', { mode: 'boolean' }).notNull().default(true),
  createdAt: integer('createdAt', { mode: 'timestamp' }).notNull(),
  updatedAt: integer('updatedAt', { mode: 'timestamp' }).notNull(),
  ...syncColumns(),
});

/**
 * Schedule ↔ tag link. Own `id` + tombstone so removals sync as rows instead of
 * disappearing through cascade. Cascade FKs stay only as a safety net for physical purges.
 */
export const scheduleTags = sqliteTable(
  'schedule_tags',
  {
    id: text('id').primaryKey(),
    scheduleId: text('scheduleId')
      .notNull()
      .references(() => schedules.id, { onDelete: 'cascade' }),
    tagId: text('tagId')
      .notNull()
      .references(() => tags.id, { onDelete: 'cascade' }),
    ...syncColumns(),
  },
  (t) => [uniqueIndex('schedule_tags_scheduleId_tagId_unique').on(t.scheduleId, t.tagId)],
);

export const schedulesRelations = relations(schedules, ({ many }) => ({
  scheduleTags: many(scheduleTags),
  scheduleReminders: many(scheduleReminders),
  scheduleExternalLinks: many(scheduleExternalLinks),
}));

export const scheduleTagsRelations = relations(scheduleTags, ({ one }) => ({
  schedule: one(schedules, {
    fields: [scheduleTags.scheduleId],
    references: [schedules.id],
  }),
  tag: one(tags, {
    fields: [scheduleTags.tagId],
    references: [tags.id],
  }),
}));

export const tagsRelations = relations(tags, ({ many }) => ({
  scheduleTags: many(scheduleTags),
}));

export type ScheduleRow = typeof schedules.$inferSelect;
export type NewSchedule = typeof schedules.$inferInsert;
export type ScheduleTagRow = typeof scheduleTags.$inferSelect;
export type NewScheduleTag = typeof scheduleTags.$inferInsert;
/** A schedule row joined with its tags and its reminder offsets (minutes, ascending). */
export type ScheduleWithTags = ScheduleRow & { tags: TagRow[]; reminders: number[] };

export type { ScheduleReminderRow };

assertSchemaMatch<AssertSchema<ScheduleRow, ScheduleEntity>>();
assertSchemaMatch<AssertSchema<ScheduleTagRow, ScheduleTagEntity>>();
