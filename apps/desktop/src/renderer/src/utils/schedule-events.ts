import type { Schedule, ScheduleException } from '@croffledev/common';
import { toFullCalendarRRule, toLocalDateTimeString } from '@croffledev/common';
import type { EventInput } from '@fullcalendar/core';
import dayjs from 'dayjs';

/** Separator between schedule id and occurrence time in an override event id. */
export const OVERRIDE_ID_SEPARATOR = '::';

/** FullCalendar treats all-day `end` as exclusive; the store keeps the inclusive last day. */
function toDisplayEnd(endDate: Date, isAllDay: boolean): Date {
  return isAllDay ? dayjs(endDate).add(1, 'day').toDate() : endDate;
}

function toEventDuration(schedule: Schedule): EventInput['duration'] {
  const start = dayjs(schedule.startDate);
  const end = dayjs(schedule.endDate);

  if (schedule.isAllDay) {
    const startDay = start.startOf('day');
    const endExclusive = end.startOf('day').add(1, 'day');
    const days = Math.max(endExclusive.diff(startDay, 'day'), 1);
    return { days };
  }

  const ms = Math.max(end.diff(start), 5 * 60 * 1000);
  return { milliseconds: ms };
}

function toSeriesEvent(schedule: Schedule): EventInput {
  const base: EventInput = {
    id: schedule.id,
    title: schedule.title,
    allDay: schedule.isAllDay,
    backgroundColor: schedule.colorLabel,
    borderColor: schedule.colorLabel,
    textColor: '#FFFFFF',
    display: schedule.isAllDay ? 'auto' : 'block',
    extendedProps: {
      scheduleId: schedule.id,
      description: schedule.description,
      location: schedule.location,
      tags: schedule.tags,
      recurrenceRule: schedule.recurrenceRule,
      priority: schedule.priority,
    },
  };

  if (schedule.recurrenceRule?.trim()) {
    const rrule = toFullCalendarRRule(schedule.recurrenceRule, schedule.startDate, {
      allDay: schedule.isAllDay,
    });
    if (rrule) {
      const exceptions = schedule.exceptions ?? [];
      return {
        ...base,
        rrule,
        duration: toEventDuration(schedule),
        // Moved or cancelled occurrences are hidden here; moved ones are drawn by toOverrideEvent.
        ...(exceptions.length > 0
          ? {
              exdate: exceptions.map((e) =>
                toLocalDateTimeString(e.occurrenceStart, schedule.isAllDay),
              ),
            }
          : {}),
      };
    }
  }

  return {
    ...base,
    start: schedule.startDate,
    end: toDisplayEnd(schedule.endDate, schedule.isAllDay),
  };
}

/** One moved occurrence, drawn as a standalone event that still opens its series. */
function toOverrideEvent(schedule: Schedule, exception: ScheduleException): EventInput | null {
  if (exception.cancelled || !exception.startDate || !exception.endDate) {
    return null;
  }
  const occurrenceStart = new Date(exception.occurrenceStart);
  const series = toSeriesEvent({ ...schedule, recurrenceRule: undefined, exceptions: [] });
  return {
    ...series,
    id: `${schedule.id}${OVERRIDE_ID_SEPARATOR}${occurrenceStart.getTime()}`,
    start: exception.startDate,
    end: toDisplayEnd(new Date(exception.endDate), schedule.isAllDay),
    extendedProps: {
      ...series.extendedProps,
      recurrenceRule: schedule.recurrenceRule,
      isOverride: true,
      occurrenceStart,
    },
  };
}

/** Calendar events for one schedule: the series (or single event) plus moved occurrences. */
export function toCalendarEvents(schedule: Schedule): EventInput[] {
  const events = [toSeriesEvent(schedule)];
  if (schedule.recurrenceRule?.trim()) {
    for (const exception of schedule.exceptions ?? []) {
      const event = toOverrideEvent(schedule, exception);
      if (event) {
        events.push(event);
      }
    }
  }
  return events;
}
