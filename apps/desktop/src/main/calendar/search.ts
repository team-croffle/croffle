import type { SearchQuery } from '@croffledev/common';
import { and, asc, gte, inArray, like, lte, or, type SQL } from 'drizzle-orm';

import { databaseManager } from '../database';
import { schedules, scheduleTags, type ScheduleWithTags } from '../database/schema';

function mapScheduleWithTags(
  row: Awaited<ReturnType<typeof querySchedules>>[number],
): ScheduleWithTags {
  const { scheduleTags: links, scheduleReminders: reminderRows, ...schedule } = row;
  return {
    ...schedule,
    tags: links.map((link) => link.tag),
    reminders: reminderRows.map((reminder) => reminder.minutes).toSorted((a, b) => a - b),
  };
}

async function querySchedules(where?: SQL) {
  const db = databaseManager.getDb();
  return db.query.schedules.findMany({
    where,
    orderBy: [asc(schedules.startDate)],
    with: {
      scheduleTags: {
        with: { tag: true },
      },
      scheduleReminders: true,
    },
  });
}

export async function searchSchedules(query: SearchQuery): Promise<ScheduleWithTags[]> {
  const conditions: SQL[] = [];

  if (query.text?.trim()) {
    const keyword = `%${query.text.trim()}%`;
    const textCondition = or(like(schedules.title, keyword), like(schedules.description, keyword));
    if (textCondition) {
      conditions.push(textCondition);
    }
  }

  const start = query.dateRange?.start ? new Date(query.dateRange.start) : null;
  const end = query.dateRange?.end ? new Date(query.dateRange.end) : null;

  if (start && end) {
    conditions.push(gte(schedules.endDate, start), lte(schedules.startDate, end));
  } else if (start) {
    conditions.push(gte(schedules.endDate, start));
  } else if (end) {
    conditions.push(lte(schedules.startDate, end));
  }

  if (query.tags?.length) {
    const tagIds = query.tags.map((tag) => tag?.id).filter((id): id is string => Boolean(id));
    if (tagIds.length > 0) {
      conditions.push(
        inArray(
          schedules.id,
          databaseManager
            .getDb()
            .select({ id: scheduleTags.scheduleId })
            .from(scheduleTags)
            .where(inArray(scheduleTags.tagId, tagIds)),
        ),
      );
    }
  }

  const rows = await querySchedules(conditions.length > 0 ? and(...conditions) : undefined);
  return rows.map(mapScheduleWithTags);
}
