import { Frequency, RRule, rrulestr, type Options } from 'rrule';

import type { ScheduleException } from './models/schedule';
import {
  WEEKDAY_ORDER,
  asSingleRRule,
  extractRuleBody,
  optionsToWeekdayCodes,
  toRRuleWeekdays,
} from './recurrence-internal';

type ExceptionLike = Pick<ScheduleException, 'occurrenceStart' | 'startDate' | 'cancelled'>;

const toTime = (value: Date | string): number =>
  (value instanceof Date ? value : new Date(value)).getTime();

/**
 * Apply per-occurrence overrides to expanded occurrence starts.
 * Drops every overridden or cancelled original start, then adds moved starts that fall
 * in [from, to] (inclusive). Result is sorted ascending and de-duplicated.
 */
export function applyExceptions(
  starts: Date[],
  exceptions: ExceptionLike[],
  from: Date,
  to: Date,
): Date[] {
  if (exceptions.length === 0) {
    return starts;
  }

  const overridden = new Set(exceptions.map((e) => toTime(e.occurrenceStart)));
  const kept = starts.filter((start) => !overridden.has(start.getTime()));

  const fromMs = from.getTime();
  const toMs = to.getTime();
  const seen = new Set(kept.map((d) => d.getTime()));
  for (const exception of exceptions) {
    if (exception.cancelled || !exception.startDate) {
      continue;
    }
    const ms = toTime(exception.startDate);
    if (ms >= fromMs && ms <= toMs && !seen.has(ms)) {
      kept.push(new Date(ms));
      seen.add(ms);
    }
  }

  return kept.toSorted((a, b) => a.getTime() - b.getTime());
}

/** Options that make a rule impossible to rotate by whole days. */
const UNSHIFTABLE: (keyof Options)[] = [
  'bysetpos',
  'byyearday',
  'byweekno',
  'bymonth',
  'bynweekday',
  'byeaster',
];

/** Largest BYMONTHDAY we accept after shifting: every month has it. */
const MAX_SAFE_MONTHDAY = 28;

/**
 * Rotate a recurrence rule so its occurrences move by `days` whole days together with
 * the series start. Returns the rule body (no DTSTART, no `RRULE:` prefix, same format as
 * `buildRRule`) or `null` when the rule cannot be shifted faithfully.
 *
 * - No BY* parts (e.g. plain DAILY): unchanged, the start date carries the shift.
 * - WEEKLY/DAILY with BYDAY: weekdays rotate by `days` (mod 7).
 * - MONTHLY with BYMONTHDAY: every day must stay within 1..28 after the shift.
 * - Anything else (BYSETPOS, BYMONTH, nth weekday, …): null.
 */
export function shiftRecurrenceRule(rule: string, days: number): string | null {
  const body = extractRuleBody(rule);
  if (!body || !Number.isInteger(days)) {
    return null;
  }
  if (days === 0) {
    return body;
  }

  let options: Partial<Options>;
  try {
    const single = asSingleRRule(rrulestr(`RRULE:${body}`));
    if (!single) {
      return null;
    }
    options = { ...single.origOptions };
  } catch {
    return null;
  }

  for (const key of UNSHIFTABLE) {
    const value = options[key];
    if (value !== undefined && value !== null && !(Array.isArray(value) && value.length === 0)) {
      return null;
    }
  }

  const hasWeekdays = options.byweekday !== undefined && options.byweekday !== null;
  const monthDays =
    options.bymonthday === undefined || options.bymonthday === null
      ? []
      : ([] as number[]).concat(options.bymonthday);

  if (hasWeekdays) {
    // "MO,+1FR"-style nth weekdays keep their `n` only in the raw rule; reject them.
    if (/BYDAY=[^;]*[+-]?\d/i.test(body) || monthDays.length > 0) {
      return null;
    }
    const codes = optionsToWeekdayCodes(options.byweekday);
    if (codes.length === 0) {
      return null;
    }
    const shift = ((days % 7) + 7) % 7;
    const rotated = codes.map((code) => WEEKDAY_ORDER[(WEEKDAY_ORDER.indexOf(code) + shift) % 7]!);
    options.byweekday = toRRuleWeekdays(rotated);
  } else if (monthDays.length > 0) {
    if (options.freq !== Frequency.MONTHLY) {
      return null;
    }
    const shifted = monthDays.map((d) => d + days);
    if (shifted.some((d) => d < 1 || d > MAX_SAFE_MONTHDAY)) {
      return null;
    }
    options.bymonthday = shifted;
  } else {
    return body;
  }

  delete options.dtstart;
  return new RRule(options as Options)
    .toString()
    .replace(/^DTSTART:[^\n]*\n?/i, '')
    .replace(/^RRULE:/i, '');
}
