import type { Schedule } from './models/schedule';
import { listOccurrenceStarts } from './occurrences';

/** How far ahead the right sidebar looks, and how many it shows. */
export const UPCOMING_DAYS = 7;
export const UPCOMING_LIMIT = 5;

const DAY_MS = 24 * 60 * 60 * 1000;

/** One occurrence in the "upcoming" list. */
export type UpcomingOccurrence = {
  schedule: Schedule;
  start: Date;
  end: Date;
  /** Original start of a recurring occurrence (the exception key); null for one-off schedules. */
  occurrenceStart: Date | null;
  /** true when the occurrence was moved on its own (a schedule exception). */
  isOverride: boolean;
};

const toDate = (value: Date | string): Date => (value instanceof Date ? value : new Date(value));

/**
 * Occurrences that start from `now` up to `days` days later, earliest first, at most `limit`.
 * An occurrence that already started (even if it has not ended) is not upcoming.
 * Recurring schedules count each occurrence; cancelled ones are skipped and moved ones are
 * placed at their new start.
 */
export function selectUpcoming(
  schedules: Schedule[],
  now: Date,
  { days = UPCOMING_DAYS, limit = UPCOMING_LIMIT }: { days?: number; limit?: number } = {},
): UpcomingOccurrence[] {
  const to = new Date(now.getTime() + days * DAY_MS);
  const result: UpcomingOccurrence[] = [];

  for (const schedule of schedules) {
    const seriesStart = toDate(schedule.startDate);
    const duration = Math.max(toDate(schedule.endDate).getTime() - seriesStart.getTime(), 0);
    const recurring = Boolean(schedule.recurrenceRule?.trim());
    const moved = new Map(
      (schedule.exceptions ?? [])
        .filter((e) => !e.cancelled && e.startDate)
        .map((e) => [toDate(e.startDate!).getTime(), e]),
    );

    for (const start of listOccurrenceStarts(schedule, now, to)) {
      if (start.getTime() < now.getTime()) {
        continue;
      }
      const exception = recurring ? moved.get(start.getTime()) : undefined;
      result.push({
        schedule,
        start,
        end: exception?.endDate ? toDate(exception.endDate) : new Date(start.getTime() + duration),
        occurrenceStart: !recurring ? null : exception ? toDate(exception.occurrenceStart) : start,
        isOverride: exception !== undefined,
      });
    }
  }

  return result
    .toSorted(
      (a, b) =>
        a.start.getTime() - b.start.getTime() || a.schedule.title.localeCompare(b.schedule.title),
    )
    .slice(0, limit);
}
