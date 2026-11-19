import { RRule, rrulestr } from 'rrule';

import type { ScheduleException } from './models/schedule';
import { applyExceptions } from './recurrence-exceptions';
import {
  asSingleRRule,
  extractRuleBody,
  fromFloatingUtc,
  toFloatingUtc,
} from './recurrence-internal';

/** What occurrence expansion needs from a schedule (no i18n, so tests can load it). */
export type OccurrenceInput = {
  startDate: Date;
  endDate: Date;
  recurrenceRule?: string | null;
  exceptions?: Pick<ScheduleException, 'occurrenceStart' | 'startDate' | 'cancelled'>[];
};

/**
 * List occurrence start times that fall in [from, to] (inclusive).
 * Non-recurring: include startDate when it intersects the window.
 * Recurring: expand RRULE with between().
 */
export function listOccurrenceStarts(schedule: OccurrenceInput, from: Date, to: Date): Date[] {
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
    const start =
      schedule.startDate instanceof Date ? schedule.startDate : new Date(schedule.startDate);
    const single = asSingleRRule(rrulestr(`RRULE:${extractRuleBody(rule)}`));
    if (!single) {
      return [];
    }
    // Expand in local wall-clock time (see toFloatingUtc). UNTIL is an instant; its local
    // wall-clock is what the editor meant (last day 23:59:59), same as the calendar view.
    const options = { ...single.origOptions, dtstart: toFloatingUtc(start) };
    if (options.until) {
      options.until = toFloatingUtc(options.until);
    }
    const starts = new RRule(options)
      .between(toFloatingUtc(from), toFloatingUtc(to), true)
      .map(fromFloatingUtc);
    return applyExceptions(starts, schedule.exceptions ?? [], from, to);
  } catch {
    return [];
  }
}
