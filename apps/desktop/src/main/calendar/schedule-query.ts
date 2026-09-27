import { and, eq } from 'drizzle-orm';

import { databaseManager } from '../database';
import {
  scheduleExceptions,
  scheduleReminders,
  schedules,
  scheduleTags,
  type ScheduleExceptionRow,
  type ScheduleRow,
  type ScheduleWithTags,
} from '../database/schema';
import { alive } from '../sync/lww';

/** Relational `with` shared by every schedule read: live links, reminders and exceptions. */
export const withLiveChildren = {
  scheduleTags: {
    where: alive(scheduleTags),
    with: { tag: true },
  },
  scheduleReminders: {
    where: alive(scheduleReminders),
  },
  scheduleExceptions: {
    where: alive(scheduleExceptions),
  },
} as const;

type ScheduleQueryRow = ScheduleRow & {
  scheduleTags: { tag: ScheduleWithTags['tags'][number] }[];
  scheduleReminders: { minutes: number }[];
  scheduleExceptions: ScheduleExceptionRow[];
};

export function mapScheduleWithTags(row: ScheduleQueryRow): ScheduleWithTags {
  const {
    scheduleTags: links,
    scheduleReminders: reminderRows,
    scheduleExceptions: exceptionRows,
    ...schedule
  } = row;
  return {
    ...schedule,
    // Link rows are filtered by `alive()` in the query; the tag itself may be tombstoned.
    tags: links.map((link) => link.tag).filter((tag) => tag.deletedAt === null),
    reminders: reminderRows.map((reminder) => reminder.minutes).toSorted((a, b) => a - b),
    exceptions: exceptionRows.toSorted(
      (a, b) => a.occurrenceStart.getTime() - b.occurrenceStart.getTime(),
    ),
  };
}

export async function findScheduleWithTags(id: string): Promise<ScheduleWithTags | null> {
  const db = databaseManager.getDb();
  const row = await db.query.schedules.findFirst({
    where: and(eq(schedules.id, id), alive(schedules)),
    with: withLiveChildren,
  });

  return row ? mapScheduleWithTags(row) : null;
}
