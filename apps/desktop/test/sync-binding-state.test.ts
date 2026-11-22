import assert from 'node:assert/strict';
import type { DatabaseSync } from 'node:sqlite';
import { describe, test } from 'node:test';

import { NEW_SERVER_STATE, SIGNED_OUT_STATE } from '../src/main/sync/binding-state.ts';
import { migratedDatabase } from './support/migrated-db.ts';

type Row = Record<string, unknown>;

/** Writes `state` to the sync_server row the way drizzle's `.set()` does (booleans as 0/1). */
function apply(db: DatabaseSync, state: Record<string, unknown>): void {
  const keys = Object.keys(state);
  const values = keys.map((k) => {
    const v = state[k];
    return typeof v === 'boolean' ? Number(v) : (v as string | number | null);
  });
  db.prepare(
    `UPDATE sync_server SET ${keys.map((k) => `${k} = ?`).join(', ')} WHERE id = 'default'`,
  ).run(...values);
}

/** A signed-in binding with every cursor and error field set. */
function boundRow(enabled: boolean): DatabaseSync {
  const db = migratedDatabase();
  db.prepare(
    `INSERT INTO sync_server (id, url, accountId, teamId, allowInsecureTls, enabled, lastPulledSeq,
       bootstrapCursor, bootstrappedAt, backfilledAt, lastSyncedAt, lastErrorKind, lastError)
     VALUES ('default', 'https://sync.example', 'acct-1', 'team-1', 1, ?, 42, 'cur', 1, 2, 3, 'net', 'down')`,
  ).run(Number(enabled));
  return db;
}

const read = (db: DatabaseSync) =>
  db.prepare(`SELECT * FROM sync_server WHERE id = 'default'`).get() as Row;

describe('sync binding state', () => {
  test('sign-out state clears the account and cursors and leaves enabled alone', () => {
    assert.equal('enabled' in SIGNED_OUT_STATE, false);
    for (const enabled of [true, false]) {
      const db = boundRow(enabled);
      apply(db, SIGNED_OUT_STATE);
      const row = read(db);
      assert.equal(row.accountId, null);
      assert.equal(row.teamId, null);
      assert.equal(row.lastPulledSeq, 0);
      for (const key of [
        'bootstrapCursor',
        'bootstrappedAt',
        'backfilledAt',
        'lastSyncedAt',
        'lastErrorKind',
        'lastError',
      ]) {
        assert.equal(row[key], null, key);
      }
      assert.equal(row.url, 'https://sync.example');
      assert.equal(row.allowInsecureTls, 1);
      assert.equal(row.enabled, Number(enabled));
    }
  });

  test('sign-in after sign-out keeps the enabled value', () => {
    for (const enabled of [true, false]) {
      const db = boundRow(enabled);
      apply(db, SIGNED_OUT_STATE);
      // What a sign-in writes (auth.ts bind → updateServerState).
      apply(db, { accountId: 'acct-1', teamId: 'team-1', lastErrorKind: null, lastError: null });
      assert.equal(read(db).enabled, Number(enabled));
    }
  });

  test('a new server URL turns enabled off', () => {
    assert.equal(NEW_SERVER_STATE.enabled, false);
    const db = boundRow(true);
    apply(db, { url: 'https://other.example', ...NEW_SERVER_STATE });
    const row = read(db);
    assert.equal(row.enabled, 0);
    assert.equal(row.accountId, null);
  });
});
