import {
  assertSchemaMatch,
  type AssertSchema,
  type ScheduleReminderEntity,
} from '@croffledev/common';
import { relations } from 'drizzle-orm';
import { integer, sqliteTable, text, uniqueIndex } from 'drizzle-orm/sqlite-core';

import { schedules } from './schedule';

/**
 * Reminder offsets for one schedule. A schedule with `useDefaultReminder = true`
 * ignores these rows and follows notifications.defaultReminderMinutes instead.
 * No rows + `useDefaultReminder = false` means "no reminder".
 */
export const scheduleReminders = sqliteTable(
  'schedule_reminder',
  {
    id: text('id').primaryKey(),
    scheduleId: text('scheduleId')
      .notNull()
      .references(() => schedules.id, { onDelete: 'cascade' }),
    minutes: integer('minutes').notNull(),
  },
  (t) => [uniqueIndex('schedule_reminder_scheduleId_minutes_unique').on(t.scheduleId, t.minutes)],
);

export const scheduleRemindersRelations = relations(scheduleReminders, ({ one }) => ({
  schedule: one(schedules, {
    fields: [scheduleReminders.scheduleId],
    references: [schedules.id],
  }),
}));

export type ScheduleReminderRow = typeof scheduleReminders.$inferSelect;
export type NewScheduleReminder = typeof scheduleReminders.$inferInsert;

assertSchemaMatch<AssertSchema<ScheduleReminderRow, ScheduleReminderEntity>>();
