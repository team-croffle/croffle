import { assertSchemaMatch, type AssertSchema, type ScheduleEntity } from '@croffledev/common';
import { relations } from 'drizzle-orm';
import { integer, primaryKey, sqliteTable, text } from 'drizzle-orm/sqlite-core';

import { scheduleReminders, type ScheduleReminderRow } from './schedule-reminder';
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
});

export const scheduleTags = sqliteTable(
  'schedule_tags',
  {
    scheduleId: text('scheduleId')
      .notNull()
      .references(() => schedules.id, { onDelete: 'cascade' }),
    tagId: text('tagId')
      .notNull()
      .references(() => tags.id, { onDelete: 'cascade' }),
  },
  (t) => [primaryKey({ columns: [t.scheduleId, t.tagId] })],
);

export const schedulesRelations = relations(schedules, ({ many }) => ({
  scheduleTags: many(scheduleTags),
  scheduleReminders: many(scheduleReminders),
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
/** A schedule row joined with its tags and its reminder offsets (minutes, ascending). */
export type ScheduleWithTags = ScheduleRow & { tags: TagRow[]; reminders: number[] };

export type { ScheduleReminderRow };

assertSchemaMatch<AssertSchema<ScheduleRow, ScheduleEntity>>();
