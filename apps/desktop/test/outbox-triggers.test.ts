import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { DatabaseSync } from 'node:sqlite';
import { beforeEach, describe, test } from 'node:test';

import {
  allOutboxTriggerSql,
  OUTBOX_TRIGGER_COUNT,
  outboxTriggerMigrationSql,
} from '../src/common/sync-tables.ts';

const DRIZZLE_DIR = join(import.meta.dirname, '../drizzle');

type Journal = { entries: { tag: string }[] };

/** Applies the committed migrations in journal order, split the way drizzle's migrator splits. */
function migratedDatabase(): DatabaseSync {
  const db = new DatabaseSync(':memory:');
  db.exec('PRAGMA foreign_keys = ON');
  const journal = JSON.parse(
    readFileSync(join(DRIZZLE_DIR, 'meta/_journal.json'), 'utf8'),
  ) as Journal;
  for (const { tag } of journal.entries) {
    const sql = readFileSync(join(DRIZZLE_DIR, `${tag}.sql`), 'utf8');
    for (const statement of sql.split('--> statement-breakpoint')) {
      if (statement.trim()) {
        db.exec(statement);
      }
    }
  }
  return db;
}

function outbox(db: DatabaseSync) {
  return db
    .prepare(
      'SELECT tableName, rowId, op, baseVersion, rowVersion, opId FROM sync_outbox ORDER BY seq',
    )
    .all() as {
    tableName: string;
    rowId: string;
    op: string;
    baseVersion: number;
    rowVersion: number;
    opId: string;
  }[];
}

const insertSchedule = (db: DatabaseSync, id: string, writer: string, version = 1) =>
  db.exec(
    `INSERT INTO schedule (id, title, startDate, endDate, createdAt, updatedAt, updatedAtMs, version, lastWriterId)
     VALUES ('${id}', 't', 1, 2, 1, 1, 1000, ${version}, '${writer}')`,
  );

describe('migration 0008', () => {
  test('is exactly the generated trigger text', () => {
    assert.equal(
      readFileSync(join(DRIZZLE_DIR, '0008_sync_outbox_triggers.sql'), 'utf8'),
      outboxTriggerMigrationSql(),
    );
  });
});

describe('outbox triggers', () => {
  let db: DatabaseSync;
  beforeEach(() => {
    db = migratedDatabase();
  });

  test('every synchronised table has an insert and an update trigger', () => {
    const { c } = db
      .prepare(
        "SELECT COUNT(*) AS c FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'sync_outbox_%'",
      )
      .get() as { c: number };
    assert.equal(c, OUTBOX_TRIGGER_COUNT);
    assert.equal(OUTBOX_TRIGGER_COUNT, 10);
  });

  test('a new row is queued as an upsert with baseVersion 0', () => {
    insertSchedule(db, 's1', 'device:a');
    const [entry] = outbox(db);
    assert.deepEqual(
      { ...entry, opId: undefined },
      {
        tableName: 'schedule',
        rowId: 's1',
        op: 'upsert',
        baseVersion: 0,
        rowVersion: 1,
        opId: undefined,
      },
    );
  });

  test('later edits fold into one entry and keep its opId and baseVersion', () => {
    insertSchedule(db, 's1', 'device:a');
    const [first] = outbox(db);
    for (let i = 0; i < 3; i++) {
      db.exec(
        "UPDATE schedule SET version = version + 1, updatedAtMs = updatedAtMs + 1 WHERE id = 's1'",
      );
    }
    const entries = outbox(db);
    assert.equal(entries.length, 1);
    assert.equal(entries[0].opId, first.opId);
    assert.equal(entries[0].baseVersion, 0);
    assert.equal(entries[0].rowVersion, 4);
  });

  test('an edit of a synced row records the version the server held', () => {
    insertSchedule(db, 's1', 'server', 5);
    db.exec("UPDATE schedule SET version = 6, lastWriterId = 'device:a' WHERE id = 's1'");
    const [entry] = outbox(db);
    assert.equal(entry.baseVersion, 5);
    assert.equal(entry.rowVersion, 6);
  });

  test('a tombstone is queued as a delete', () => {
    insertSchedule(db, 's1', 'device:a');
    db.exec("UPDATE schedule SET version = 2, deletedAt = 1700000000000 WHERE id = 's1'");
    assert.equal(outbox(db)[0].op, 'delete');
  });

  test('rows written by the sync engine are not queued (no ping-pong)', () => {
    insertSchedule(db, 's1', 'server');
    db.exec("UPDATE schedule SET version = 2, lastWriterId = 'server' WHERE id = 's1'");
    assert.equal(outbox(db).length, 0);
  });

  test('a table outside sync is not queued', () => {
    insertSchedule(db, 's1', 'server');
    db.exec(
      `INSERT INTO schedule_external_link (id, scheduleId, provider, accountId, externalId, updatedAtMs, lastWriterId)
       VALUES ('l1', 's1', 'google', 'acct', 'x', 1, 'device:a')`,
    );
    assert.equal(outbox(db).length, 0);
  });

  test('the repair statements restore a dropped trigger', () => {
    db.exec('DROP TRIGGER sync_outbox_tag_ai');
    for (const statement of allOutboxTriggerSql(true)) {
      db.exec(statement);
    }
    const { c } = db
      .prepare(
        "SELECT COUNT(*) AS c FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'sync_outbox_%'",
      )
      .get() as { c: number };
    assert.equal(c, OUTBOX_TRIGGER_COUNT);
  });
});
