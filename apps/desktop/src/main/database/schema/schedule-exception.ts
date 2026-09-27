import {
  assertSchemaMatch,
  type AssertSchema,
  type ScheduleExceptionEntity,
} from '@croffledev/common';
import { relations } from 'drizzle-orm';
import { integer, sqliteTable, text, uniqueIndex } from 'drizzle-orm/sqlite-core';

import { schedules } from './schedule';
import { syncColumns } from './sync-columns';

/**
 * Per-occurrence override of a recurring schedule, keyed by the occurrence's original
 * start instant. Same shape a future per-user reminder overlay can extend with `userId`.
 */
export const scheduleExceptions = sqliteTable(
  'schedule_exception',
  {
    id: text('id').primaryKey(),
    scheduleId: text('scheduleId')
      .notNull()
      .references(() => schedules.id, { onDelete: 'cascade' }),
    occurrenceStart: integer('occurrenceStart', { mode: 'timestamp_ms' }).notNull(),
    startDate: integer('startDate', { mode: 'timestamp_ms' }),
    endDate: integer('endDate', { mode: 'timestamp_ms' }),
    cancelled: integer('cancelled', { mode: 'boolean' }).notNull().default(false),
    ...syncColumns(),
  },
  (t) => [
    uniqueIndex('schedule_exception_scheduleId_occurrenceStart_unique').on(
      t.scheduleId,
      t.occurrenceStart,
    ),
  ],
);

export const scheduleExceptionsRelations = relations(scheduleExceptions, ({ one }) => ({
  schedule: one(schedules, {
    fields: [scheduleExceptions.scheduleId],
    references: [schedules.id],
  }),
}));

export type ScheduleExceptionRow = typeof scheduleExceptions.$inferSelect;
export type NewScheduleException = typeof scheduleExceptions.$inferInsert;

assertSchemaMatch<AssertSchema<ScheduleExceptionRow, ScheduleExceptionEntity>>();
