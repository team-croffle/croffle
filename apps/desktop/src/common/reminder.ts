import { rrulestr } from 'rrule';

import { t } from './i18n';
import { asSingleRRule, extractRuleBody } from './recurrence-internal';

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
};

export type ReminderCandidate = {
  scheduleId: string;
  title: string;
  occurrenceStart: Date;
  remindAt: Date;
  reminderMinutes: number;
  isAllDay: boolean;
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
 * List occurrence start times that fall in [from, to] (inclusive).
 * Non-recurring: include startDate when it intersects the window.
 * Recurring: expand RRULE with between().
 */
export function listOccurrenceStarts(
  schedule: Pick<ReminderScheduleInput, 'startDate' | 'endDate' | 'recurrenceRule'>,
  from: Date,
  to: Date,
): Date[] {
  const rule = schedule.recurrenceRule?.trim();
  if (!rule) {
    const start =
      schedule.startDate instanceof Date ? schedule.startDate : new Date(schedule.startDate);
    if (start >= from && start <= to) {
      return [start];
    }
    return [];
  }

  try {
    const parsed = rrulestr(`RRULE:${extractRuleBody(rule)}`, {
      dtstart:
        schedule.startDate instanceof Date ? schedule.startDate : new Date(schedule.startDate),
    });
    const single = asSingleRRule(parsed);
    if (!single) {
      return [];
    }
    return single.between(from, to, true);
  } catch {
    return [];
  }
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

export function formatReminderBody(candidate: ReminderCandidate, locale?: string | null): string {
  if (candidate.isAllDay) {
    return t('reminder.bodyAllDay', locale, { minutes: candidate.reminderMinutes });
  }

  const start = candidate.occurrenceStart;
  const hh = String(start.getHours()).padStart(2, '0');
  const mm = String(start.getMinutes()).padStart(2, '0');
  return t('reminder.bodyTimed', locale, {
    minutes: candidate.reminderMinutes,
    time: `${hh}:${mm}`,
  });
}
