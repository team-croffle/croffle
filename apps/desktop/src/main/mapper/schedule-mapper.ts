import type { Schedule, ScheduleException, Tag } from '@croffledev/common';

import type { ScheduleWithTags, TagRow } from '../database/schema';

/** Ascending, de-duplicated. Range checks live in validateScheduleData. */
function normalizeReminders(minutes: number[]): number[] {
  return [...new Set(minutes)].toSorted((a, b) => a - b);
}

/** Exception write input (dates already as Date). */
export type ScheduleExceptionInput = {
  occurrenceStart: Date;
  startDate: Date | null;
  endDate: Date | null;
  cancelled: boolean;
};

/** Drizzle write input (dates already as Date). */
export type ScheduleEntityInput = {
  id?: string;
  title?: string;
  description?: string | null;
  location?: string | null;
  startDate?: Date;
  endDate?: Date;
  isAllDay?: boolean;
  recurrenceRule?: string | null;
  colorLabel?: string;
  priority?: 'low' | 'medium' | 'high';
  useDefaultReminder?: boolean;
  reminders?: number[];
  createdAt?: Date;
  updatedAt?: Date;
  /** Only `id` is persisted (link rows); name/color ride along for callers that echo the input. */
  tags?: Tag[];
  /** Full replacement of the live exception set (import / full-object update). */
  exceptions?: ScheduleExceptionInput[];
};

function toTag(tag: Tag): Tag {
  return {
    id: tag.id,
    name: tag.name,
    color: tag.color,
  };
}

function toTagDto(tag: TagRow): Tag {
  return {
    id: tag.id,
    name: tag.name,
    color: tag.color,
  };
}

function toDate(value: Date): Date {
  return value instanceof Date ? value : new Date(value);
}

function toExceptionDto(row: ScheduleWithTags['exceptions'][number]): ScheduleException {
  return {
    occurrenceStart: row.occurrenceStart,
    startDate: row.startDate,
    endDate: row.endDate,
    cancelled: row.cancelled,
  };
}

function toExceptionInput(exception: ScheduleException): ScheduleExceptionInput | null {
  const occurrenceStart = toDate(exception.occurrenceStart);
  if (Number.isNaN(occurrenceStart.getTime())) {
    return null;
  }
  const cancelled = exception.cancelled === true;
  const startDate = !cancelled && exception.startDate ? toDate(exception.startDate) : null;
  const endDate = !cancelled && exception.endDate ? toDate(exception.endDate) : null;
  if (!cancelled && (!startDate || !endDate)) {
    return null;
  }
  return { occurrenceStart, startDate, endDate, cancelled };
}

export const scheduleMapper = {
  toInterface(entity: ScheduleWithTags): Schedule {
    const reminders = entity.useDefaultReminder ? null : normalizeReminders(entity.reminders);

    return {
      id: entity.id,
      title: entity.title,
      description: entity.description ?? '',
      location: entity.location ?? '',
      startDate: entity.startDate,
      endDate: entity.endDate,
      isAllDay: entity.isAllDay,
      recurrenceRule: entity.recurrenceRule ?? undefined,
      colorLabel: entity.colorLabel,
      priority: entity.priority,
      reminders,
      // Deprecated mirror for extensions written against 1.2.0 (removed in 1.3).
      reminderMinutes: reminders === null ? null : (reminders[0] ?? 0),
      createdAt: entity.createdAt,
      updatedAt: entity.updatedAt,
      tags: entity.tags.map(toTagDto),
      exceptions: entity.recurrenceRule?.trim() ? entity.exceptions.map(toExceptionDto) : [],
    };
  },

  toEntity(data: Partial<Schedule>): ScheduleEntityInput {
    const entity: ScheduleEntityInput = {};

    if (data.id !== undefined) {
      entity.id = data.id;
    }
    if (data.title !== undefined) {
      entity.title = data.title;
    }
    if (data.description !== undefined) {
      entity.description = data.description;
    }
    if (data.location !== undefined) {
      entity.location = data.location;
    }
    if (data.startDate !== undefined) {
      entity.startDate = toDate(data.startDate);
    }
    if (data.endDate !== undefined) {
      entity.endDate = toDate(data.endDate);
    }
    if (data.isAllDay !== undefined) {
      entity.isAllDay = data.isAllDay;
    }
    if (data.recurrenceRule !== undefined) {
      entity.recurrenceRule = data.recurrenceRule;
    }
    if (data.colorLabel !== undefined) {
      entity.colorLabel = data.colorLabel;
    }
    if (data.priority !== undefined) {
      entity.priority = data.priority;
    }
    // `reminders` wins; `reminderMinutes` is the deprecated single-value path.
    if (data.reminders !== undefined) {
      if (data.reminders === null) {
        entity.useDefaultReminder = true;
        entity.reminders = [];
      } else {
        entity.useDefaultReminder = false;
        entity.reminders = normalizeReminders(data.reminders);
      }
    } else if (data.reminderMinutes !== undefined) {
      if (data.reminderMinutes === null) {
        entity.useDefaultReminder = true;
        entity.reminders = [];
      } else {
        entity.useDefaultReminder = false;
        entity.reminders = data.reminderMinutes > 0 ? [data.reminderMinutes] : [];
      }
    }
    if (data.createdAt !== undefined) {
      entity.createdAt = toDate(data.createdAt);
    }
    if (data.updatedAt !== undefined) {
      entity.updatedAt = toDate(data.updatedAt);
    }
    if (data.tags !== undefined) {
      entity.tags = data.tags.map(toTag);
    }
    if (Array.isArray(data.exceptions)) {
      entity.exceptions = data.exceptions
        .map(toExceptionInput)
        .filter((item): item is ScheduleExceptionInput => item !== null);
    }

    return entity;
  },
};
