import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, test } from 'node:test';

import './support/register.ts';
import type { ReminderCandidate, ReminderScheduleInput } from '../src/common/reminder.ts';

const { buildReminderCandidates, reminderBody, startOfLocalDay } =
  await import('../src/common/reminder.ts');

const LOCALES = join(import.meta.dirname, '../src/common/i18n/locales');
const catalogs = Object.fromEntries(
  ['ko', 'en'].map((lang) => [
    lang,
    JSON.parse(readFileSync(join(LOCALES, `${lang}.json`), 'utf8')),
  ]),
);

/** The body the app shows (formatReminderBody = t(reminderBody)), from the catalog files. */
function formatReminderBody(candidate: ReminderCandidate, lang: 'ko' | 'en'): string {
  const { key, values } = reminderBody(candidate);
  const text = key.split('.').reduce((tree, part) => tree[part], catalogs[lang]) as string;
  return text.replace(/\{(\w+)\}/g, (_, name: string) => String(values[name]));
}

const now = new Date(2026, 10, 24, 8, 0);
const horizonEnd = new Date(2026, 10, 27, 0, 0);
const due = new Date(2026, 10, 25, 9, 0);

const todo = (overrides: Partial<ReminderScheduleInput> = {}): ReminderScheduleInput => ({
  id: 'todo-1',
  title: 'file taxes',
  startDate: due,
  endDate: due,
  isAllDay: false,
  useDefaultReminder: false,
  reminders: [30],
  kind: 'todo',
  completedAt: null,
  ...overrides,
});

const build = (schedules: ReminderScheduleInput[]) =>
  buildReminderCandidates(schedules, { now, horizonEnd, defaultReminderMinutes: 10 });

describe('reminders for To-Dos', () => {
  test('excludes completed todos from reminder candidates', () => {
    const open = todo();
    const doneTodo = todo({ id: 'todo-2', completedAt: new Date(2026, 10, 24, 7, 0) });
    const ids = build([open, doneTodo]).map((c) => c.scheduleId);
    assert.deepEqual(ids, ['todo-1']);
  });

  test('schedules timed todo reminder relative to due time', () => {
    const [candidate] = build([todo()]);
    assert.equal(candidate.remindAt.getTime(), due.getTime() - 30 * 60_000);
    assert.equal(candidate.isTodo, true);
  });

  test('schedules all-day todo reminder relative to local midnight of the due date', () => {
    const [candidate] = build([todo({ isAllDay: true, reminders: [60] })]);
    assert.equal(candidate.remindAt.getTime(), startOfLocalDay(due).getTime() - 60 * 60_000);
  });

  test('formats todo reminder body with due copy in ko and en and keeps event reminder behaviour', () => {
    const [timedTodo] = build([todo()]);
    assert.equal(formatReminderBody(timedTodo, 'ko'), '30분 후 마감 · 09:00');
    assert.equal(formatReminderBody(timedTodo, 'en'), 'Due in 30 min · 09:00');
    const [allDayTodo] = build([todo({ isAllDay: true })]);
    assert.equal(formatReminderBody(allDayTodo, 'ko'), '30분 전 · 종일 할 일 마감');
    assert.equal(formatReminderBody(allDayTodo, 'en'), '30 minutes before · All-day To-Do due');

    // An event (kind missing, as before 1.3.3) keeps the start-based time and the old copy.
    const eventStart = new Date(2026, 10, 25, 14, 0);
    const [event] = build([
      todo({ id: 'event-1', kind: undefined, startDate: eventStart, endDate: due }),
    ]);
    assert.equal(event.isTodo, false);
    assert.equal(event.remindAt.getTime(), eventStart.getTime() - 30 * 60_000);
    assert.equal(formatReminderBody(event, 'ko'), '30분 후 시작 · 14:00');
    assert.equal(formatReminderBody(event, 'en'), 'Starts in 30 min · 14:00');
    const [allDayEvent] = build([
      todo({ id: 'event-2', kind: 'event', isAllDay: true, startDate: eventStart }),
    ]);
    assert.equal(formatReminderBody(allDayEvent, 'ko'), '30분 전 · 종일 일정');
  });
});
