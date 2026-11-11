import { SERVER_WRITER_ID, SYNC_TABLES, type SyncTableName } from '@croffledev/common';

import { databaseManager } from '../database';
import { deviceWriterId } from './client-id';

/** Tables whose rows carry a second-resolution `updatedAt` to date legacy writes from. */
const HAS_UPDATED_AT: ReadonlySet<SyncTableName> = new Set(['tag', 'schedule']);

/**
 * Queues every live row written before sync was set up, once per binding, so the first push
 * takes this device's existing calendar to the server. Runs after the first snapshot, so rows
 * that came from the server (`lastWriterId = 'server'`) are left out.
 *
 * - Rows from before 1.2.3 have `updatedAtMs = 0` and no writer; the server rejects that, so
 *   they are dated from `updatedAt` (or now) and attributed to this device first. That update
 *   fires the triggers, which is fine: the entry ends up the same.
 * - Entries that existed before the backfill are left alone (their `baseVersion` is real); every
 *   entry the backfill creates is for a row the server has never seen, so its `baseVersion` is 0.
 * - Tombstones are skipped: a row deleted before sync never existed anywhere else.
 */
export function backfillOutbox(): number {
  const sqlite = databaseManager.getSqlite();
  const nowMs = Date.now();
  const writer = deviceWriterId();

  const run = sqlite.transaction(() => {
    sqlite.exec('CREATE TEMP TABLE IF NOT EXISTS backfill_existing (tableName TEXT, rowId TEXT)');
    sqlite.exec('DELETE FROM backfill_existing');
    sqlite.exec('INSERT INTO backfill_existing SELECT tableName, rowId FROM sync_outbox');

    for (const table of SYNC_TABLES) {
      const datedFrom = HAS_UPDATED_AT.has(table) ? '`updatedAt` * 1000' : String(nowMs);
      sqlite
        .prepare(
          `UPDATE \`${table}\`
             SET updatedAtMs = CASE WHEN updatedAtMs > 0 THEN updatedAtMs ELSE ${datedFrom} END,
                 lastWriterId = coalesce(lastWriterId, ?)
           WHERE deletedAt IS NULL AND (updatedAtMs = 0 OR lastWriterId IS NULL)`,
        )
        .run(writer);

      // `WHERE true` keeps SQLite from reading ON CONFLICT as part of the SELECT's join.
      sqlite
        .prepare(
          `INSERT INTO sync_outbox
             (opId, tableName, rowId, op, baseVersion, rowVersion, updatedAtMs, enqueuedAt, attempts)
           SELECT lower(hex(randomblob(16))), ?, id, 'upsert', 0, version, updatedAtMs, ?, 0
             FROM \`${table}\`
            WHERE deletedAt IS NULL AND lastWriterId IS NOT ? AND true
           ON CONFLICT (tableName, rowId) DO NOTHING`,
        )
        .run(table, nowMs, SERVER_WRITER_ID);
    }

    const { changes } = sqlite
      .prepare(
        `UPDATE sync_outbox SET baseVersion = 0
          WHERE NOT EXISTS (
            SELECT 1 FROM backfill_existing e
             WHERE e.tableName = sync_outbox.tableName AND e.rowId = sync_outbox.rowId)`,
      )
      .run();
    sqlite.exec('DELETE FROM backfill_existing');
    return changes;
  });

  return run() as number;
}
