import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, test } from 'node:test';

import {
  decodeRow,
  encodeRow,
  PROTOCOL_VERSION,
  RowCodecError,
  TABLE_COLUMNS,
  wins,
  type LocalRow,
  type LwwKey,
  type WireRow,
} from '../src/common/sync-protocol.ts';
import { SYNC_TABLES } from '../src/common/sync-tables.ts';

// Shared with croffle-server's `go test ./internal/conformance`.
const SCENARIOS_DIR = join(import.meta.dirname, '../../../packages/sync-fixtures/scenarios');

type Scenario = {
  name: string;
  protocol: number;
  rowCodec?: {
    cases: { table: string; row: Record<string, unknown>; wire: WireRow }[];
    lww: { a: LwwKey; b: LwwKey; winner: 'a' | 'b' }[];
  };
};

const scenarioFiles = readdirSync(SCENARIOS_DIR).filter((f) => f.endsWith('.json'));
const scenarios: Scenario[] = scenarioFiles.map((file) =>
  JSON.parse(readFileSync(join(SCENARIOS_DIR, file), 'utf8')),
);

/**
 * Fixture rows hold raw column values (dates as SQLite stores them: seconds or milliseconds);
 * the codec works on drizzle rows (`Date` objects). This is the only translation the reader adds.
 */
function toLocal(table: string, row: Record<string, unknown>): LocalRow {
  return Object.fromEntries(
    TABLE_COLUMNS[table].map((spec) => {
      const value = row[spec.key];
      if (value === null || value === undefined) {
        return [spec.key, null];
      }
      if (spec.kind === 'sec') {
        return [spec.key, new Date((value as number) * 1000)];
      }
      if (spec.kind === 'ms') {
        return [spec.key, new Date(value as number)];
      }
      return [spec.key, value];
    }),
  );
}

describe('shared fixtures', () => {
  test('every scenario is read and speaks this protocol version', () => {
    assert.ok(scenarioFiles.length >= 12, `found ${scenarioFiles.length} scenarios`);
    for (const scenario of scenarios) {
      assert.equal(scenario.protocol, PROTOCOL_VERSION, scenario.name);
    }
  });

  const codecScenarios = scenarios.filter((s) => s.rowCodec);
  test('at least one scenario carries row codec cases', () => {
    assert.ok(codecScenarios.length > 0);
  });

  for (const scenario of codecScenarios) {
    const { cases, lww } = scenario.rowCodec!;
    cases.forEach((c, i) => {
      test(`${scenario.name}: case ${i} (${c.table}) encodes to the wire row`, () => {
        assert.deepEqual(encodeRow(c.table, toLocal(c.table, c.row)), c.wire);
      });
      test(`${scenario.name}: case ${i} (${c.table}) decodes back to the local row`, () => {
        assert.deepEqual(decodeRow(c.table, c.wire), toLocal(c.table, c.row));
      });
    });
    lww.forEach((c, i) => {
      test(`${scenario.name}: lww ${i} picks ${c.winner}`, () => {
        assert.equal(wins(c.a, c.b), c.winner);
      });
    });
  }
});

// The fixtures cover `schedule` only; these pin the other four tables.
describe('row codec, remaining tables', () => {
  const sync = {
    updatedAtMs: 1700000000123,
    version: 3,
    deletedAt: null,
    lastWriterId: 'device:client-1',
    ownerId: 'acct-1',
  };
  const wireSync = { ...sync };

  const cases: { table: string; local: LocalRow; wire: WireRow }[] = [
    {
      table: 'tag',
      local: {
        id: 'tag-1',
        name: 'work',
        color: '#ff0000',
        createdAt: new Date(1699999000_000),
        updatedAt: new Date(1700000000_000),
        ...sync,
      },
      wire: {
        id: 'tag-1',
        name: 'work',
        color: '#ff0000',
        createdAt: 1699999000_000,
        updatedAt: 1700000000_000,
        ...wireSync,
      },
    },
    {
      table: 'schedule_tags',
      local: { id: 'st-1', scheduleId: 'row-1', tagId: 'tag-1', ...sync },
      wire: { id: 'st-1', scheduleId: 'row-1', tagId: 'tag-1', ...wireSync },
    },
    {
      table: 'schedule_reminder',
      local: { id: 'row-1-15', scheduleId: 'row-1', minutes: 15, ...sync },
      wire: { id: 'row-1-15', scheduleId: 'row-1', minutes: 15, ...wireSync },
    },
    {
      table: 'schedule_exception',
      local: {
        id: 'ex-1',
        scheduleId: 'row-1',
        occurrenceStart: new Date(1700000000_250),
        startDate: new Date(1700086400_500),
        endDate: null,
        cancelled: true,
        ...sync,
        deletedAt: new Date(1700000001_999),
      },
      wire: {
        id: 'ex-1',
        scheduleId: 'row-1',
        occurrenceStart: 1700000000_250,
        startDate: 1700086400_500,
        endDate: null,
        cancelled: 1,
        ...wireSync,
        deletedAt: 1700000001_999,
      },
    },
  ];

  test('cases cover every synchronised table except schedule', () => {
    assert.deepEqual(
      cases.map((c) => c.table).sort(),
      SYNC_TABLES.filter((t) => t !== 'schedule').sort(),
    );
  });

  for (const c of cases) {
    test(`${c.table} encodes (millisecond columns keep their milliseconds)`, () => {
      assert.deepEqual(encodeRow(c.table, c.local), c.wire);
    });
    test(`${c.table} decodes back`, () => {
      assert.deepEqual(decodeRow(c.table, c.wire), c.local);
    });
  }

  test('every table has a column spec, and only those', () => {
    assert.deepEqual(Object.keys(TABLE_COLUMNS).sort(), [...SYNC_TABLES].sort());
  });

  test('second-resolution columns drop sub-second parts on the wire', () => {
    const wire = encodeRow('tag', { ...cases[0].local, createdAt: new Date(1699999000_789) });
    assert.equal(wire.createdAt, 1699999000_000);
  });

  test('decode drops fields with no local column (teamId from the server)', () => {
    const row = decodeRow('schedule_tags', { ...cases[1].wire, teamId: 'team-1' });
    assert.equal('teamId' in row, false);
  });

  test('an unknown table is an error, not a silent pass', () => {
    assert.throws(() => encodeRow('schedule_external_link', {}), RowCodecError);
    assert.throws(() => decodeRow('nope', {}), RowCodecError);
  });

  test('a wrongly typed value is an error', () => {
    assert.throws(
      () => encodeRow('tag', { ...cases[0].local, createdAt: 1699999000 }),
      RowCodecError,
    );
    assert.throws(
      () => decodeRow('schedule_reminder', { ...cases[2].wire, minutes: '15' }),
      RowCodecError,
    );
  });
});

describe('last writer wins', () => {
  test('a full tie keeps the first argument', () => {
    const key = { updatedAtMs: 1, lastWriterId: 'device:a' };
    assert.equal(wins(key, { ...key }), 'a');
  });

  test('a missing writer id loses a tie to any writer', () => {
    assert.equal(
      wins({ updatedAtMs: 5, lastWriterId: null }, { updatedAtMs: 5, lastWriterId: 'device:a' }),
      'b',
    );
  });
});
