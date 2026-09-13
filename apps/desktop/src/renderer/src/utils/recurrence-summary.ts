import { parseRRule, type WeekdayCode } from '@croffledev/common';
import { rrulestr } from 'rrule';

type Translate = (key: string, params?: Record<string, unknown>) => string;

/** 2024-01-01 = 월요일. 요일 이름을 로케일로 뽑기 위한 고정 기준일. */
const MONDAY_BASE = Date.UTC(2024, 0, 1);
const WEEKDAY_ORDER: WeekdayCode[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];

const intlLocale = (locale: string): string => (locale === 'ko' ? 'ko-KR' : 'en-US');

function weekdayNames(codes: WeekdayCode[], locale: string): string {
  const fmt = new Intl.DateTimeFormat(intlLocale(locale), { weekday: 'short', timeZone: 'UTC' });
  return WEEKDAY_ORDER.filter((code) => codes.includes(code))
    .map((code) => fmt.format(new Date(MONDAY_BASE + WEEKDAY_ORDER.indexOf(code) * 86_400_000)))
    .join(', ');
}

function formatUntil(until: string, locale: string): string {
  // 'YYYY-MM-DD'는 로컬 날짜로 해석 (new Date(str)는 UTC 자정)
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(until);
  const date = match
    ? new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
    : new Date(until);
  return new Intl.DateTimeFormat(intlLocale(locale), { month: 'short', day: 'numeric' }).format(
    date,
  );
}

function monthDay(rule: string, fallback: Date | undefined): number | null {
  try {
    const parsed = rrulestr(`RRULE:${rule.replace(/^RRULE:/i, '')}`);
    const days = (parsed as { origOptions?: { bymonthday?: number | number[] } }).origOptions
      ?.bymonthday;
    if (Array.isArray(days) && days.length > 0) {
      return days[0]!;
    }
    if (typeof days === 'number') {
      return days;
    }
  } catch {
    // fall through
  }
  return fallback ? fallback.getDate() : null;
}

/**
 * RRULE을 사람이 읽는 한 줄로 만든다. 예: "매주 월, 수 · 10회", "Every 3 days · until Sep 30".
 * 반복이 없으면 null. 프리셋에 맞지 않는 규칙은 `recurrenceSummary.custom` + 원문.
 */
export function describeRecurrence(
  rule: string | null | undefined,
  t: Translate,
  locale: string,
  startDate?: Date,
): string | null {
  const state = parseRRule(rule);
  if (state.preset === 'none') {
    return null;
  }

  let main: string;
  switch (state.preset) {
    case 'daily':
      main = t('recurrenceSummary.daily');
      break;
    case 'weekdays':
      main = t('recurrenceSummary.weekdays');
      break;
    case 'weekly':
      main = t('recurrenceSummary.weekly', { days: weekdayNames(state.byWeekday, locale) });
      break;
    case 'monthly': {
      const day = monthDay(state.rawRule, startDate);
      main =
        day === null
          ? t('recurrenceSummary.monthlyNoDay')
          : t('recurrenceSummary.monthly', { day });
      break;
    }
    case 'every-n-days':
      main = t('recurrenceSummary.everyNDays', { n: state.interval });
      break;
    case 'every-n-weeks':
      main = t('recurrenceSummary.everyNWeeks', {
        n: state.interval,
        days: weekdayNames(state.byWeekday, locale),
      });
      break;
    default:
      return `${t('recurrenceSummary.custom')} · ${state.rawRule}`;
  }

  if (state.endMode === 'until' && state.until) {
    return `${main} · ${t('recurrenceSummary.untilSuffix', { date: formatUntil(state.until, locale) })}`;
  }
  if (state.endMode === 'count') {
    return `${main} · ${t('recurrenceSummary.countSuffix', { n: state.count })}`;
  }
  return main;
}
