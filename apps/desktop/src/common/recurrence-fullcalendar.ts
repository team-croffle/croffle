import { type RRule, type RRuleSet, rrulestr, type ByWeekday } from 'rrule';

import {
  FREQ_TO_FC,
  WEEKDAY_TO_FC,
  asSingleRRule,
  extractRuleBody,
  optionsToWeekdayCodes,
} from './recurrence-internal';

export type FullCalendarRRuleInput = {
  freq: string;
  interval?: number;
  byweekday?: string[];
  bymonthday?: number[];
  /** 오프셋 없는 로컬 ISO 문자열 (`YYYY-MM-DDTHH:mm:ss`, 종일은 `YYYY-MM-DD`) */
  dtstart: string;
  /** dtstart와 같은 형식 */
  until?: string;
  count?: number;
};

export type FullCalendarRRuleOptions = {
  /** 종일 일정이면 날짜만 넘겨 FullCalendar가 시각 없는 반복으로 전개하게 한다 */
  allDay?: boolean;
};

const pad2 = (n: number): string => String(n).padStart(2, '0');

/**
 * 로컬 벽시계 기준 ISO 문자열. 시간대 오프셋을 붙이지 않는다.
 *
 * WHY: @fullcalendar/rrule은 dtstart/until이 Date면 그대로 rrule에 넘기고
 * "시간대 미지정"으로 취급해 UTC 필드를 로컬 시각으로 재해석한다. 실제 인스턴트를
 * Date로 넘기면 KST에서 9시간 앞으로 밀려 전날/이틀짜리로 그려진다. 오프셋 없는
 * 문자열은 플러그인이 `parseMarker`로 읽어 캘린더 시간대(local)로 정확히 전개한다.
 */
export function toLocalDateTimeString(value: Date | string, dateOnly = false): string {
  const d = value instanceof Date ? value : new Date(value);
  const date = `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
  if (dateOnly) {
    return date;
  }
  return `${date}T${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`;
}

/** FullCalendar @fullcalendar/rrule 용 객체 */
export function toFullCalendarRRule(
  rule: string,
  dtstart: Date | string,
  options: FullCalendarRRuleOptions = {},
): FullCalendarRRuleInput | null {
  try {
    const start = dtstart instanceof Date ? dtstart : new Date(dtstart);
    const parsed = rrulestr(`RRULE:${extractRuleBody(rule)}`, { dtstart: start });
    const single = asSingleRRule(parsed);
    if (!single) {
      return null;
    }

    const { options: ruleOptions } = single;
    const freq = FREQ_TO_FC[ruleOptions.freq];
    if (!freq) {
      return null;
    }

    const dateOnly = options.allDay === true;
    const result: FullCalendarRRuleInput = {
      freq,
      interval: ruleOptions.interval || 1,
      dtstart: toLocalDateTimeString(start, dateOnly),
    };

    const weekdays = optionsToWeekdayCodes(ruleOptions.byweekday as ByWeekday | ByWeekday[] | null);
    if (weekdays.length > 0) {
      result.byweekday = weekdays.map((code) => WEEKDAY_TO_FC[code]);
    }

    if (ruleOptions.bymonthday?.length) {
      result.bymonthday = [...ruleOptions.bymonthday];
    }
    if (ruleOptions.until) {
      // UNTIL은 UTC 인스턴트 → 로컬 벽시계로 변환해야 마지막 날이 빠지지 않는다
      result.until = toLocalDateTimeString(ruleOptions.until, dateOnly);
    }
    if (ruleOptions.count) {
      result.count = ruleOptions.count;
    }

    return result;
  } catch {
    return null;
  }
}

// re-export types used only for narrowing RRuleSet in consumers if needed
export type { RRule, RRuleSet };
