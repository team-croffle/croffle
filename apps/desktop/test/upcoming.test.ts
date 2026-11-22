import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import './support/register.ts';
import type { Schedule, ScheduleException } from '../src/common/models/schedule.ts';

const { selectUpcoming } = await import('../src/common/upcoming.ts');

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;
/** A fixed "now": Thursday 2026-11-19 10:00 local time. */
const NOW = new Date(2026, 10, 19, 10, 0, 0);

let seq = 0;
function schedule(title: string, start: Date, end: Date, extra: Partial<Schedule> = {}): Schedule {
  seq += 1;
  return {
    id: `s${seq}`,
    title,
    description: '',
    location: '',
    startDate: start,
    endDate: end,
    isAllDay: false,
    colorLabel: '#3c6ea5',
    priority: 'medium',
    reminders: null,
    tags: [],
    createdAt: NOW,
    updatedAt: NOW,
    ...extra,
  };
}

const at = (offsetMs: number) => new Date(NOW.getTime() + offsetMs);
const titles = (list: { schedule: Schedule }[]) => list.map((o) => o.schedule.title);

describe('upcoming schedules', () => {
  test('excludes todos even when due within 7 days', () => {
    const event = schedule('event', at(DAY), at(DAY + HOUR));
    const todo = schedule('todo', at(2 * DAY), at(2 * DAY), { kind: 'todo', completedAt: null });
    assert.deepEqual(titles(selectUpcoming([event, todo], NOW)), ['event']);
  });

  test('keeps schedules starting within 7 days', () => {
    const inside = schedule('inside', at(7 * DAY - HOUR), at(7 * DAY));
    const outside = schedule('outside', at(7 * DAY + 60_000), at(7 * DAY + HOUR));
    assert.deepEqual(titles(selectUpcoming([inside, outside], NOW)), ['inside']);
  });

  test('sorts by start and keeps at most 5', () => {
    const list = [6, 2, 4, 1, 5, 3].map((d) => schedule(`d${d}`, at(d * DAY), at(d * DAY + HOUR)));
    assert.deepEqual(titles(selectUpcoming(list, NOW)), ['d1', 'd2', 'd3', 'd4', 'd5']);
  });

  test('excludes ongoing and past schedules', () => {
    const ongoing = schedule('ongoing', at(-HOUR), at(HOUR));
    const past = schedule('past', at(-3 * HOUR), at(-2 * HOUR));
    const later = schedule('later', at(HOUR), at(2 * HOUR));
    assert.deepEqual(titles(selectUpcoming([ongoing, past, later], NOW)), ['later']);
  });

  test('counts recurring occurrences one by one', () => {
    // Daily at 09:00 from yesterday: today's 09:00 has passed, so the next five days fill the list.
    const start = new Date(2026, 10, 18, 9, 0, 0);
    const daily = schedule('daily', start, new Date(start.getTime() + HOUR), {
      recurrenceRule: 'FREQ=DAILY',
    });
    const result = selectUpcoming([daily], NOW);
    assert.equal(result.length, 5);
    assert.deepEqual(
      result.map((o) => o.start.getTime()),
      [1, 2, 3, 4, 5].map((d) => new Date(2026, 10, 19 + d, 9, 0, 0).getTime()),
    );
    assert.ok(result.every((o) => o.occurrenceStart?.getTime() === o.start.getTime()));
  });

  test('applies exceptions', () => {
    const start = new Date(2026, 10, 18, 9, 0, 0);
    const exceptions: ScheduleException[] = [
      // 11-20 09:00 is skipped
      {
        occurrenceStart: new Date(2026, 10, 20, 9, 0, 0),
        startDate: null,
        endDate: null,
        cancelled: true,
      },
      // 11-21 09:00 is moved to 11-22 08:00, before the 11-22 09:00 occurrence
      {
        occurrenceStart: new Date(2026, 10, 21, 9, 0, 0),
        startDate: new Date(2026, 10, 22, 8, 0, 0),
        endDate: new Date(2026, 10, 22, 8, 30, 0),
        cancelled: false,
      },
    ];
    const daily = schedule('daily', start, new Date(start.getTime() + HOUR), {
      recurrenceRule: 'FREQ=DAILY',
      exceptions,
    });
    const result = selectUpcoming([daily], NOW, { limit: 3 });
    assert.deepEqual(
      result.map((o) => o.start.getTime()),
      [
        new Date(2026, 10, 22, 8, 0, 0),
        new Date(2026, 10, 22, 9, 0, 0),
        new Date(2026, 10, 23, 9, 0, 0),
      ].map((d) => d.getTime()),
    );
    assert.equal(result[0]!.isOverride, true);
    assert.equal(result[0]!.end.getTime(), new Date(2026, 10, 22, 8, 30, 0).getTime());
    assert.equal(result[0]!.occurrenceStart?.getTime(), new Date(2026, 10, 21, 9, 0, 0).getTime());
  });

  test('all-day schedules starting tomorrow are included', () => {
    const tomorrow = new Date(2026, 10, 20);
    const allDay = schedule('all-day', tomorrow, new Date(2026, 10, 20, 23, 59, 59), {
      isAllDay: true,
    });
    const result = selectUpcoming([allDay], NOW);
    assert.deepEqual(titles(result), ['all-day']);
    assert.equal(result[0]!.occurrenceStart, null);
  });
});
