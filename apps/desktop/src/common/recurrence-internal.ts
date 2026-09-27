import { Frequency, RRule, RRuleSet, type ByWeekday, type Options } from 'rrule';

export type WeekdayCode = 'MO' | 'TU' | 'WE' | 'TH' | 'FR' | 'SA' | 'SU';

export const WEEKDAY_ORDER: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];
export const WEEKDAY_SET = new Set<string>(WEEKDAY_ORDER);
export const WEEKDAYS_DEFAULT: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR'];

export const FREQ_TO_FC: Record<number, string> = {
  [Frequency.YEARLY]: 'yearly',
  [Frequency.MONTHLY]: 'monthly',
  [Frequency.WEEKLY]: 'weekly',
  [Frequency.DAILY]: 'daily',
  [Frequency.HOURLY]: 'hourly',
  [Frequency.MINUTELY]: 'minutely',
  [Frequency.SECONDLY]: 'secondly',
};

export const WEEKDAY_TO_FC: Record<WeekdayCode, string> = {
  MO: 'mo',
  TU: 'tu',
  WE: 'we',
  TH: 'th',
  FR: 'fr',
  SA: 'sa',
  SU: 'su',
};

export function jsWeekdayToCode(day: number): WeekdayCode {
  const map: WeekdayCode[] = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA'];
  return map[day] ?? 'MO';
}

export function normalizeWeekdays(codes: WeekdayCode[]): WeekdayCode[] {
  const unique = [...new Set(codes.filter((c) => WEEKDAY_SET.has(c)))];
  return WEEKDAY_ORDER.filter((d) => unique.includes(d));
}

export function toRRuleWeekdays(codes: WeekdayCode[]) {
  return normalizeWeekdays(codes).map((code) => RRule[code]);
}

export function extractRuleBody(rule: string): string {
  const trimmed = rule.trim();
  if (!trimmed) {
    return '';
  }
  if (trimmed.includes('RRULE:')) {
    const match = /RRULE:([^\n]+)/i.exec(trimmed);
    return (match?.[1] ?? trimmed).trim();
  }
  return trimmed.replace(/^RRULE:/i, '').trim();
}

export function optionsToWeekdayCodes(
  byweekday: ByWeekday | ByWeekday[] | null | undefined,
): WeekdayCode[] {
  if (!byweekday) {
    return [];
  }
  const list = Array.isArray(byweekday) ? byweekday : [byweekday];
  const codes: WeekdayCode[] = [];
  for (const item of list) {
    if (typeof item === 'number') {
      const map: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];
      const code = map[item];
      if (code) {
        codes.push(code);
      }
      continue;
    }
    const weekday = typeof item === 'object' && item && 'weekday' in item ? item.weekday : item;
    const map: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];
    if (typeof weekday === 'number' && map[weekday]) {
      codes.push(map[weekday]!);
    }
  }
  return normalizeWeekdays(codes);
}

export function asSingleRRule(parsed: RRule | RRuleSet): RRule | null {
  if (parsed instanceof RRuleSet) {
    const first = parsed.rrules()[0];
    if (!first) {
      return null;
    }
    return first instanceof RRuleSet ? ((first.rrules()[0] as RRule | undefined) ?? null) : first;
  }
  return parsed;
}

export function formatUntilDate(until: Date | null | undefined): string | null {
  if (!until) {
    return null;
  }
  const y = until.getFullYear();
  const m = String(until.getMonth() + 1).padStart(2, '0');
  const d = String(until.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

export function sameWeekdays(a: WeekdayCode[], b: WeekdayCode[]): boolean {
  return normalizeWeekdays(a).join(',') === normalizeWeekdays(b).join(',');
}

/**
 * rrule.js evaluates BYDAY / BYMONTHDAY on UTC fields. Passing a local Date shifts the
 * weekday for times whose UTC date differs (KST before 09:00). "Floating" dates carry the
 * local wall-clock in their UTC fields so the rule is evaluated in local time.
 */
export function toFloatingUtc(date: Date): Date {
  return new Date(
    Date.UTC(
      date.getFullYear(),
      date.getMonth(),
      date.getDate(),
      date.getHours(),
      date.getMinutes(),
      date.getSeconds(),
      date.getMilliseconds(),
    ),
  );
}

/** Inverse of {@link toFloatingUtc}: UTC fields back to a local Date. */
export function fromFloatingUtc(date: Date): Date {
  return new Date(
    date.getUTCFullYear(),
    date.getUTCMonth(),
    date.getUTCDate(),
    date.getUTCHours(),
    date.getUTCMinutes(),
    date.getUTCSeconds(),
    date.getUTCMilliseconds(),
  );
}

export type { Options };
