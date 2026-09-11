import { Frequency, RRule, rrulestr, type Options } from 'rrule';

import {
  WEEKDAYS_DEFAULT,
  asSingleRRule,
  extractRuleBody,
  formatUntilDate,
  jsWeekdayToCode,
  optionsToWeekdayCodes,
  sameWeekdays,
  toRRuleWeekdays,
  type WeekdayCode,
} from './recurrence-internal';

export type { WeekdayCode } from './recurrence-internal';
export {
  toFullCalendarRRule,
  toLocalDateTimeString,
  type FullCalendarRRuleInput,
  type FullCalendarRRuleOptions,
} from './recurrence-fullcalendar';

export type RecurrencePreset =
  | 'none'
  | 'daily'
  | 'weekdays'
  | 'weekly'
  | 'monthly'
  | 'every-n-days'
  | 'every-n-weeks'
  | 'custom';

export type RecurrenceEndMode = 'never' | 'until' | 'count';

export type RecurrenceFormState = {
  preset: RecurrencePreset;
  interval: number;
  byWeekday: WeekdayCode[];
  endMode: RecurrenceEndMode;
  until: string | null;
  count: number;
  /** Preset 매칭 실패 시 원본 RRULE 유지 */
  rawRule: string;
};

export const RECURRENCE_PRESET_OPTIONS: RecurrencePreset[] = [
  'none',
  'daily',
  'weekdays',
  'weekly',
  'monthly',
  'every-n-days',
  'every-n-weeks',
  'custom',
];

export const WEEKDAY_OPTIONS: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];

export function createDefaultRecurrenceFormState(): RecurrenceFormState {
  return {
    preset: 'none',
    interval: 1,
    byWeekday: [],
    endMode: 'never',
    until: null,
    count: 10,
    rawRule: '',
  };
}

/** RRULE 본문만 반환. 반복 없으면 undefined */
export function buildRRule(state: RecurrenceFormState, startDate: Date): string | undefined {
  if (state.preset === 'none') {
    return undefined;
  }

  if (state.preset === 'custom') {
    const raw = state.rawRule.trim();
    return raw || undefined;
  }

  const interval = Math.max(1, Math.floor(state.interval || 1));
  const options: Partial<Options> = {
    dtstart: startDate,
  };

  switch (state.preset) {
    case 'daily':
      options.freq = Frequency.DAILY;
      options.interval = 1;
      break;
    case 'weekdays':
      options.freq = Frequency.WEEKLY;
      options.interval = 1;
      options.byweekday = toRRuleWeekdays(WEEKDAYS_DEFAULT);
      break;
    case 'weekly': {
      options.freq = Frequency.WEEKLY;
      options.interval = 1;
      const days =
        state.byWeekday.length > 0 ? state.byWeekday : [jsWeekdayToCode(startDate.getDay())];
      options.byweekday = toRRuleWeekdays(days);
      break;
    }
    case 'monthly':
      options.freq = Frequency.MONTHLY;
      options.interval = 1;
      options.bymonthday = [startDate.getDate()];
      break;
    case 'every-n-days':
      options.freq = Frequency.DAILY;
      options.interval = interval;
      break;
    case 'every-n-weeks': {
      options.freq = Frequency.WEEKLY;
      options.interval = interval;
      const days =
        state.byWeekday.length > 0 ? state.byWeekday : [jsWeekdayToCode(startDate.getDay())];
      options.byweekday = toRRuleWeekdays(days);
      break;
    }
    default:
      return undefined;
  }

  if (state.endMode === 'until' && state.until) {
    const until = new Date(state.until);
    if (!Number.isNaN(until.getTime())) {
      until.setHours(23, 59, 59, 999);
      options.until = until;
    }
  } else if (state.endMode === 'count') {
    options.count = Math.max(1, Math.floor(state.count || 1));
  }

  const built = new RRule(options as Options);
  return built
    .toString()
    .replace(/^DTSTART:[^\n]*\n?/i, '')
    .replace(/^RRULE:/i, '');
}

function matchPreset(options: Partial<Options>): Omit<RecurrenceFormState, 'rawRule'> | null {
  const freq = options.freq;
  const interval = options.interval ?? 1;
  const byWeekday = optionsToWeekdayCodes(options.byweekday);
  const endMode = options.until ? 'until' : options.count ? 'count' : 'never';

  const base = {
    interval,
    byWeekday,
    endMode: endMode as RecurrenceFormState['endMode'],
    until: formatUntilDate(options.until ?? null),
    count: options.count ?? 10,
  };

  if (freq === Frequency.DAILY && interval === 1 && byWeekday.length === 0) {
    return { ...base, preset: 'daily' };
  }
  if (freq === Frequency.DAILY && interval > 1 && byWeekday.length === 0) {
    return { ...base, preset: 'every-n-days' };
  }
  if (freq === Frequency.WEEKLY && interval === 1 && sameWeekdays(byWeekday, WEEKDAYS_DEFAULT)) {
    return { ...base, preset: 'weekdays', byWeekday: [...WEEKDAYS_DEFAULT] };
  }
  if (freq === Frequency.WEEKLY && interval === 1) {
    return { ...base, preset: 'weekly' };
  }
  if (freq === Frequency.WEEKLY && interval > 1) {
    return { ...base, preset: 'every-n-weeks' };
  }
  if (freq === Frequency.MONTHLY && interval === 1) {
    return { ...base, preset: 'monthly' };
  }

  return null;
}

export function parseRRule(rule: string | null | undefined): RecurrenceFormState {
  const empty = createDefaultRecurrenceFormState();
  if (!rule?.trim()) {
    return empty;
  }

  const body = extractRuleBody(rule);
  if (!body) {
    return empty;
  }

  try {
    const parsed = rrulestr(`RRULE:${body}`);
    const single = asSingleRRule(parsed);
    const options = single?.origOptions;
    if (!options) {
      return { ...empty, preset: 'custom', rawRule: body };
    }

    const matched = matchPreset(options);
    if (!matched) {
      return { ...empty, preset: 'custom', rawRule: body };
    }

    return { ...matched, rawRule: body };
  } catch {
    return { ...empty, preset: 'custom', rawRule: body };
  }
}

export function isValidRecurrenceRule(rule: string | null | undefined): boolean {
  if (!rule?.trim()) {
    return true;
  }
  try {
    rrulestr(`RRULE:${extractRuleBody(rule)}`);
    return true;
  } catch {
    return false;
  }
}

/** 반복 일정이 조회 period와 겹칠 수 있는지 (until 기준, count는 보수적으로 포함) */
export function recurrenceMayOverlapPeriod(
  rule: string | null | undefined,
  startDate: Date,
  periodStart: Date,
  periodEnd: Date,
): boolean {
  if (!rule?.trim()) {
    return false;
  }
  if (startDate > periodEnd) {
    return false;
  }

  try {
    const parsed = rrulestr(`RRULE:${extractRuleBody(rule)}`, { dtstart: startDate });
    const single = asSingleRRule(parsed);
    const options = single?.options;
    if (options?.until && options.until < periodStart) {
      return false;
    }
    return true;
  } catch {
    return startDate <= periodEnd;
  }
}

export function weekdayCodeFromDate(date: Date): WeekdayCode {
  return jsWeekdayToCode(date.getDay());
}
