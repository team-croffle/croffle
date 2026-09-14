import { randomUUID } from 'node:crypto';

import { isValidRecurrenceRule, recurrenceMayOverlapPeriod } from '@croffledev/common';
import {
  and,
  asc,
  eq,
  gte,
  isNotNull,
  isNull,
  lte,
  ne,
  or,
  type SQL as DrizzleSQL,
} from 'drizzle-orm';

import { databaseManager } from '../database';
import {
  schedules,
  scheduleTags,
  type NewSchedule,
  type ScheduleRow,
  type ScheduleWithTags,
} from '../database/schema';
import type { ScheduleEntityInput } from '../mapper/schedule-mapper';
import { colorValidation } from '../utils/color-validator';
import { stringValidation } from '../utils/string-validator';

/** 0 = no reminder, null = app default. Upper bound: 7 days. */
const MAX_REMINDER_MINUTES = 7 * 24 * 60;

function mapScheduleWithTags(
  row: ScheduleRow & {
    scheduleTags: { tag: ScheduleWithTags['tags'][number] }[];
  },
): ScheduleWithTags {
  const { scheduleTags: links, ...schedule } = row;
  return {
    ...schedule,
    tags: links.map((link) => link.tag),
  };
}

export function validateScheduleData(schedule: ScheduleEntityInput) {
  if (schedule.title !== undefined && !stringValidation(schedule.title, false, 100, 1)) {
    throw new Error('Title must be between 1 and 100 characters');
  }

  if (schedule.colorLabel !== undefined && !colorValidation(schedule.colorLabel)) {
    throw new Error('Invalid color label format');
  }

  if (schedule.description && !stringValidation(schedule.description, true, 2000, 0)) {
    throw new Error('Description must be less than 2000 characters');
  }

  if (schedule.location && !stringValidation(schedule.location, true, 200, 0)) {
    throw new Error('Location must be less than 200 characters');
  }

  if (schedule.startDate && schedule.endDate) {
    if (schedule.startDate > schedule.endDate) {
      throw new Error('Start date cannot be later than end date');
    }
  }

  if (schedule.recurrenceRule !== undefined && schedule.recurrenceRule !== null) {
    if (!isValidRecurrenceRule(schedule.recurrenceRule)) {
      throw new Error('Invalid recurrence rule');
    }
  }

  if (schedule.reminderMinutes !== undefined && schedule.reminderMinutes !== null) {
    const minutes = schedule.reminderMinutes;
    if (!Number.isInteger(minutes) || minutes < 0 || minutes > MAX_REMINDER_MINUTES) {
      throw new Error(`Reminder minutes must be an integer between 0 and ${MAX_REMINDER_MINUTES}`);
    }
  }
}

async function syncScheduleTags(scheduleId: string, tags: { id: string }[] | undefined) {
  if (tags === undefined) {
    return;
  }

  const db = databaseManager.getDb();
  await db.delete(scheduleTags).where(eq(scheduleTags.scheduleId, scheduleId));

  if (tags.length === 0) {
    return;
  }

  await db.insert(scheduleTags).values(
    tags.map((tag) => ({
      scheduleId,
      tagId: tag.id,
    })),
  );
}

async function findScheduleWithTags(id: string): Promise<ScheduleWithTags | null> {
  const db = databaseManager.getDb();
  const row = await db.query.schedules.findFirst({
    where: eq(schedules.id, id),
    with: {
      scheduleTags: {
        with: { tag: true },
      },
    },
  });

  return row ? mapScheduleWithTags(row) : null;
}

export async function getSchedules(period: {
  start: Date;
  end: Date;
}): Promise<ScheduleWithTags[]> {
  const db = databaseManager.getDb();
  const { start, end } = period;

  let where: DrizzleSQL | undefined;

  if (start && end) {
    const nonRecurring = and(
      or(isNull(schedules.recurrenceRule), eq(schedules.recurrenceRule, '')),
      gte(schedules.endDate, start),
      lte(schedules.startDate, end),
    );
    const recurring = and(
      isNotNull(schedules.recurrenceRule),
      ne(schedules.recurrenceRule, ''),
      lte(schedules.startDate, end),
    );
    where = or(nonRecurring, recurring);
  } else if (start) {
    where = gte(schedules.endDate, start);
  } else if (end) {
    where = lte(schedules.startDate, end);
  }

  const rows = await db.query.schedules.findMany({
    where,
    orderBy: [asc(schedules.startDate)],
    with: {
      scheduleTags: {
        with: { tag: true },
      },
    },
  });

  const mapped = rows.map(mapScheduleWithTags);

  if (!start || !end) {
    return mapped;
  }

  return mapped.filter((schedule) => {
    if (!schedule.recurrenceRule?.trim()) {
      return true;
    }
    return recurrenceMayOverlapPeriod(schedule.recurrenceRule, schedule.startDate, start, end);
  });
}

export async function createSchedule(data: ScheduleEntityInput): Promise<ScheduleWithTags> {
  if (!data.title) {
    throw new Error('Title is required');
  }
  if (!data.startDate || !data.endDate) {
    throw new Error('Date range is required');
  }

  validateScheduleData(data);

  const now = new Date();
  const id = data.id ?? randomUUID();
  const { tags: inputTags, ...scheduleFields } = data;
  const startDate = data.startDate;
  const endDate = data.endDate;

  const values: NewSchedule = {
    id,
    title: scheduleFields.title!,
    description: scheduleFields.description ?? null,
    location: scheduleFields.location ?? null,
    startDate,
    endDate,
    isAllDay: scheduleFields.isAllDay ?? false,
    recurrenceRule: scheduleFields.recurrenceRule ?? null,
    colorLabel: scheduleFields.colorLabel ?? '#E1E1E1',
    priority: scheduleFields.priority ?? 'medium',
    reminderMinutes: scheduleFields.reminderMinutes ?? null,
    createdAt: scheduleFields.createdAt ?? now,
    updatedAt: scheduleFields.updatedAt ?? now,
  };

  const db = databaseManager.getDb();
  await db.insert(schedules).values(values);
  await syncScheduleTags(id, inputTags);

  const created = await findScheduleWithTags(id);
  if (!created) {
    throw new Error('Failed to create schedule');
  }
  return created;
}

export async function updateSchedule(
  id: string,
  data: ScheduleEntityInput,
): Promise<ScheduleWithTags> {
  const existing = await findScheduleWithTags(id);
  if (!existing) {
    throw new Error('Schedule not found');
  }

  const { tags: inputTags, ...scheduleFields } = data;
  const merged: ScheduleEntityInput = {
    id: existing.id,
    title: scheduleFields.title ?? existing.title,
    description:
      scheduleFields.description !== undefined ? scheduleFields.description : existing.description,
    location: scheduleFields.location !== undefined ? scheduleFields.location : existing.location,
    startDate: scheduleFields.startDate ?? existing.startDate,
    endDate: scheduleFields.endDate ?? existing.endDate,
    isAllDay: scheduleFields.isAllDay ?? existing.isAllDay,
    recurrenceRule:
      scheduleFields.recurrenceRule !== undefined
        ? scheduleFields.recurrenceRule
        : existing.recurrenceRule,
    colorLabel: scheduleFields.colorLabel ?? existing.colorLabel,
    priority: scheduleFields.priority ?? existing.priority,
    reminderMinutes:
      scheduleFields.reminderMinutes !== undefined
        ? scheduleFields.reminderMinutes
        : existing.reminderMinutes,
    createdAt: scheduleFields.createdAt ?? existing.createdAt,
    updatedAt: scheduleFields.updatedAt ?? existing.updatedAt,
    tags: inputTags ?? existing.tags,
  };

  validateScheduleData(merged);

  const db = databaseManager.getDb();
  await db
    .update(schedules)
    .set({
      title: merged.title,
      description: merged.description ?? null,
      location: merged.location ?? null,
      startDate: merged.startDate,
      endDate: merged.endDate,
      isAllDay: merged.isAllDay,
      recurrenceRule: merged.recurrenceRule ?? null,
      colorLabel: merged.colorLabel,
      priority: merged.priority ?? 'medium',
      reminderMinutes: merged.reminderMinutes ?? null,
      updatedAt: new Date(),
    })
    .where(eq(schedules.id, id));

  await syncScheduleTags(id, inputTags);

  const updated = await findScheduleWithTags(id);
  if (!updated) {
    throw new Error('Schedule not found');
  }
  return updated;
}

export async function deleteSchedule(id: string): Promise<boolean> {
  const db = databaseManager.getDb();
  const existing = await db.query.schedules.findFirst({
    where: eq(schedules.id, id),
  });

  if (!existing) {
    throw new Error('Schedule not found');
  }

  await db.delete(schedules).where(eq(schedules.id, id));
  return true;
}
