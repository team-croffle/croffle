import type { ScheduleException } from './models/schedule';
import { listOccurrenceStarts } from './occurrences';

export { listOccurrenceStarts } from './occurrences';

export type ReminderScheduleInput = {
  id: string;
  title: string;
  startDate: Date;
  endDate: Date;
  isAllDay: boolean;
  recurrenceRule?: string | null;
  /** true (or missing) = follow the app default; false = use `reminders`. */
  useDefaultReminder?: boolean;
  /** Offsets in minutes before the occurrence starts. */
  reminders?: number[];
  /** Per-occurrence overrides (recurring schedules only). */
  exceptions?: Pick<ScheduleException, 'occurrenceStart' | 'startDate' | 'cancelled'>[];
  /** 'todo': start = deadline, so the reminder counts from the deadline (1.3.3). Missing = event. */
  kind?: 'event' | 'todo';
  /** A done To-Do does not remind. */
  completedAt?: Date | null;
};

export type ReminderCandidate = {
  scheduleId: string;
  title: string;
  occurrenceStart: Date;
  remindAt: Date;
  reminderMinutes: number;
  isAllDay: boolean;
  /** A To-Do: the occurrence start is its deadline. */
  isTodo: boolean;
};

export const MAX_REMINDER_MINUTES = 10_080;

/**
 * Offsets (minutes before start) that should fire for a schedule.
 * App default when `useDefaultReminder` is not false; otherwise the schedule's own list,
 * filtered to integers in 1..MAX_REMINDER_MINUTES, de-duplicated and sorted ascending.
 * An empty result means "no reminder".
 */
export function resolveReminderMinutesList(
  schedule: Pick<ReminderScheduleInput, 'useDefaultReminder' | 'reminders'>,
  defaultMinutes: number,
): number[] {
  if (schedule.useDefaultReminder !== false) {
    return defaultMinutes > 0 ? [defaultMinutes] : [];
  }
  const valid = (schedule.reminders ?? []).filter(
    (m) => Number.isInteger(m) && m > 0 && m <= MAX_REMINDER_MINUTES,
  );
  return [...new Set(valid)].toSorted((a, b) => a - b);
}

/** Local midnight of the occurrence's calendar day. */
export function startOfLocalDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0);
}

export function toRemindAt(occurrenceStart: Date, minutes: number, isAllDay: boolean): Date {
  const base = isAllDay ? startOfLocalDay(occurrenceStart) : occurrenceStart;
  return new Date(base.getTime() - minutes * 60_000);
}

/**
 * Build reminder candidates for schedules in a time horizon.
 * Occurrence window is expanded backward by max reminder minutes so early remindAt still maps to an occurrence.
 */
export function buildReminderCandidates(
  schedules: ReminderScheduleInput[],
  options: {
    now: Date;
    horizonEnd: Date;
    defaultReminderMinutes: number;
  },
): ReminderCandidate[] {
  const { now, horizonEnd, defaultReminderMinutes } = options;
  const candidates: ReminderCandidate[] = [];

  for (const schedule of schedules) {
    if (schedule.kind === 'todo' && schedule.completedAt) {
      continue;
    }
    const minutesList = resolveReminderMinutesList(schedule, defaultReminderMinutes);
    if (minutesList.length === 0) {
      continue;
    }

    // Expand occurrences once, rewound by the largest offset so early remindAt still maps to an occurrence.
    const maxMinutes = Math.max(...minutesList);
    const occurrenceFrom = new Date(now.getTime() - maxMinutes * 60_000 - 60_000);
    const starts = listOccurrenceStarts(schedule, occurrenceFrom, horizonEnd);

    for (const occurrenceStart of starts) {
      for (const minutes of minutesList) {
        const remindAt = toRemindAt(occurrenceStart, minutes, schedule.isAllDay);
        if (remindAt > horizonEnd) {
          continue;
        }
        candidates.push({
          scheduleId: schedule.id,
          title: schedule.title,
          occurrenceStart,
          remindAt,
          reminderMinutes: minutes,
          isAllDay: schedule.isAllDay,
          isTodo: schedule.kind === 'todo',
        });
      }
    }
  }

  candidates.sort((a, b) => a.remindAt.getTime() - b.remindAt.getTime());
  return candidates;
}

/**
 * Dedup key for one (schedule, occurrence, offset). The occurrence timestamp stays in the
 * second segment because `pruneFired` parses it from there.
 */
export function reminderDedupKey(
  scheduleId: string,
  occurrenceStart: Date,
  minutes: number,
): string {
  return `${scheduleId}|${occurrenceStart.getTime()}|${minutes}`;
}

/** Pre-1.2.1 key (no offset segment); still honoured so already-sent reminders don't repeat after update. */
export function legacyReminderDedupKey(scheduleId: string, occurrenceStart: Date): string {
  return `${scheduleId}|${occurrenceStart.getTime()}`;
}

/** Catalog key and values of a reminder's body; `formatReminderBody` (reminder-format.ts) fills it. */
export function reminderBody(candidate: ReminderCandidate): {
  key:
    | 'reminder.bodyAllDay'
    | 'reminder.bodyTimed'
    | 'reminder.bodyTodoAllDay'
    | 'reminder.bodyTodoTimed';
  values: Record<string, string | number>;
} {
  if (candidate.isAllDay) {
    return {
      key: candidate.isTodo ? 'reminder.bodyTodoAllDay' : 'reminder.bodyAllDay',
      values: { minutes: candidate.reminderMinutes },
    };
  }
  const start = candidate.occurrenceStart;
  const hh = String(start.getHours()).padStart(2, '0');
  const mm = String(start.getMinutes()).padStart(2, '0');
  return {
    key: candidate.isTodo ? 'reminder.bodyTodoTimed' : 'reminder.bodyTimed',
    values: { minutes: candidate.reminderMinutes, time: `${hh}:${mm}` },
  };
}
