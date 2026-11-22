import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  alignTodoDates,
  kindForCreate,
  mergeKindFields,
  validateKindFields,
} from '../src/main/calendar/schedule-kind.ts';
import type { ScheduleWithTags } from '../src/main/database/schema/index.ts';
import { scheduleMapper } from '../src/main/mapper/schedule-mapper.ts';
import { migratedDatabase, migrationSql, migrationTags } from './support/migrated-db.ts';

const due = new Date('2026-11-25T09:00:00Z');
const done = new Date('2026-11-24T18:00:00Z');

describe('To-Do (schedule kind)', () => {
  test('migration 0009 adds kind default event and nullable completedAt without backfill UPDATE', () => {
    const tag = migrationTags().find((t) => t.startsWith('0009_'));
    assert.ok(tag, 'migration 0009 exists');
    assert.doesNotMatch(migrationSql(tag), /\bUPDATE\b/i);

    const db = migratedDatabase();
    const columns = db.prepare('PRAGMA table_info(schedule)').all() as {
      name: string;
      notnull: number;
      dflt_value: string | null;
    }[];
    const kind = columns.find((c) => c.name === 'kind');
    const completedAt = columns.find((c) => c.name === 'completedAt');
    assert.deepEqual(
      { notnull: kind?.notnull, dflt: kind?.dflt_value },
      { notnull: 1, dflt: "'event'" },
    );
    assert.deepEqual(
      { notnull: completedAt?.notnull, dflt: completedAt?.dflt_value },
      { notnull: 0, dflt: null },
    );

    db.prepare(
      `INSERT INTO schedule (id, title, startDate, endDate, createdAt, updatedAt)
       VALUES ('s1', 'old row', 1, 2, 1, 1)`,
    ).run();
    assert.deepEqual(
      { ...(db.prepare(`SELECT kind, completedAt FROM schedule WHERE id = 's1'`).get() as object) },
      { kind: 'event', completedAt: null },
    );
  });

  test('rejects an invalid kind value', () => {
    assert.throws(() => validateKindFields({ kind: 'task' as never }), /kind/);
    assert.doesNotThrow(() => validateKindFields({ kind: 'event' }));
    assert.doesNotThrow(() => validateKindFields({ kind: 'todo' }));
    assert.doesNotThrow(() => validateKindFields({}));
  });

  test('aligns todo startDate to endDate and rejects recurrence on a todo', () => {
    const start = new Date('2026-11-25T08:00:00Z');
    assert.deepEqual(
      alignTodoDates({ kind: 'todo', startDate: start, endDate: due }).startDate,
      due,
    );
    assert.deepEqual(
      alignTodoDates({ kind: 'event', startDate: start, endDate: due }).startDate,
      start,
    );
    assert.throws(
      () => validateKindFields({ kind: 'todo', recurrenceRule: 'FREQ=DAILY' }),
      /cannot repeat/,
    );
    assert.doesNotThrow(() => validateKindFields({ kind: 'todo', recurrenceRule: '' }));
    assert.doesNotThrow(() => validateKindFields({ kind: 'event', recurrenceRule: 'FREQ=DAILY' }));
  });

  test('rejects completedAt on an event and clears completedAt when switching todo to event', () => {
    assert.throws(() => validateKindFields({ kind: 'event', completedAt: done }), /To-Do/);
    assert.throws(() => validateKindFields({ completedAt: done }), /To-Do/);
    assert.doesNotThrow(() => validateKindFields({ kind: 'event', completedAt: null }));
    assert.deepEqual(mergeKindFields({ kind: 'todo', completedAt: done }, { kind: 'event' }), {
      kind: 'event',
      completedAt: null,
    });
  });

  test('update merge preserves omitted kind and completedAt', () => {
    const existing = { kind: 'todo' as const, completedAt: done };
    assert.deepEqual(mergeKindFields(existing, {}), existing);
    assert.deepEqual(mergeKindFields(existing, { completedAt: null }), {
      kind: 'todo',
      completedAt: null,
    });
    assert.deepEqual(mergeKindFields({ kind: 'event', completedAt: null }, {}), {
      kind: 'event',
      completedAt: null,
    });
  });

  test('scheduleMapper maps kind and completedAt and defaults missing kind on import to event', () => {
    const entity = scheduleMapper.toEntity({ kind: 'todo', completedAt: done, endDate: due });
    assert.equal(entity.kind, 'todo');
    assert.deepEqual(entity.completedAt, done);
    assert.equal(scheduleMapper.toEntity({ completedAt: null }).completedAt, null);

    const row = {
      id: 's1',
      title: 'file taxes',
      description: null,
      location: null,
      startDate: due,
      endDate: due,
      isAllDay: false,
      recurrenceRule: null,
      colorLabel: '#E1E1E1',
      priority: 'medium',
      useDefaultReminder: true,
      kind: 'todo',
      completedAt: done,
      createdAt: due,
      updatedAt: due,
      tags: [],
      reminders: [],
      exceptions: [],
    } as unknown as ScheduleWithTags;
    const dto = scheduleMapper.toInterface(row);
    assert.equal(dto.kind, 'todo');
    assert.deepEqual(dto.completedAt, done);

    // An imported schedule without kind (exported by 1.3.2 or earlier) is created as an event.
    const imported = scheduleMapper.toEntity({ title: 'old', startDate: due, endDate: due });
    assert.equal(imported.kind, undefined);
    assert.equal(kindForCreate(imported), 'event');
  });
});
