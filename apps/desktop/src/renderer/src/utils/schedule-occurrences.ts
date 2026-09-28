import { listOccurrenceStarts, type Schedule } from '@croffledev/common';

/** One concrete occurrence of a schedule (a one-off schedule has exactly one). */
export type ScheduleOccurrence = {
  schedule: Schedule;
  start: Date;
  end: Date;
  /** Original start of a recurring occurrence (the exception key); null for one-off schedules. */
  occurrenceStart: Date | null;
  /** true when the occurrence was moved on its own (a schedule exception). */
  isOverride: boolean;
};

const toDate = (value: Date | string): Date => (value instanceof Date ? value : new Date(value));

const overlaps = (start: Date, end: Date, from: Date, to: Date) => start <= to && end >= from;

/**
 * Occurrences of `schedule` that touch the local day of `date`.
 * Recurring schedules are expanded in local time (same as reminders), originals replaced by
 * their exceptions: moved ones appear on their new day only, cancelled ones not at all.
 */
export function occurrencesOnDate(schedule: Schedule, date: Date): ScheduleOccurrence[] {
  const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const dayEnd = new Date(dayStart.getFullYear(), dayStart.getMonth(), dayStart.getDate() + 1);
  dayEnd.setMilliseconds(-1);

  const seriesStart = toDate(schedule.startDate);
  const seriesEnd = toDate(schedule.endDate);
  const duration = Math.max(seriesEnd.getTime() - seriesStart.getTime(), 0);

  if (!schedule.recurrenceRule?.trim()) {
    return overlaps(seriesStart, seriesEnd, dayStart, dayEnd)
      ? [{ schedule, start: seriesStart, end: seriesEnd, occurrenceStart: null, isOverride: false }]
      : [];
  }

  const exceptions = schedule.exceptions ?? [];
  const excepted = new Set(exceptions.map((e) => toDate(e.occurrenceStart).getTime()));
  const result: ScheduleOccurrence[] = [];

  // Rewind by the duration so an occurrence that started earlier but runs into this day counts.
  const originals = listOccurrenceStarts(
    { startDate: seriesStart, endDate: seriesEnd, recurrenceRule: schedule.recurrenceRule },
    new Date(dayStart.getTime() - duration),
    dayEnd,
  );
  for (const start of originals) {
    if (excepted.has(start.getTime())) {
      continue;
    }
    const end = new Date(start.getTime() + duration);
    if (overlaps(start, end, dayStart, dayEnd)) {
      result.push({ schedule, start, end, occurrenceStart: start, isOverride: false });
    }
  }

  for (const exception of exceptions) {
    if (exception.cancelled || !exception.startDate || !exception.endDate) {
      continue;
    }
    const start = toDate(exception.startDate);
    const end = toDate(exception.endDate);
    if (overlaps(start, end, dayStart, dayEnd)) {
      result.push({
        schedule,
        start,
        end,
        occurrenceStart: toDate(exception.occurrenceStart),
        isOverride: true,
      });
    }
  }

  return result;
}
