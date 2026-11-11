import {
  encodeRow,
  ERROR_CODES,
  PUSH_BATCH_SIZE,
  wins,
  type Commit,
  type Conflict,
  type PushResponse,
} from '@croffledev/common';
import { and, eq, sql } from 'drizzle-orm';

import { databaseManager } from '../database';
import { syncOutbox, type SyncOutboxRow } from '../database/schema';
import { logger } from '../logger';
import { applyServerRows, isSyncTable, readLocalRow, SYNCED_TABLES } from './apply';
import { getClientId } from './client-id';
import { ack, fail, peekBatch, remove } from './outbox';
import { request, SyncError } from './transport';

export type PushContext = { url: string; token: string };
export type PushResult = { accepted: number; conflicts: number; failed: number };

type Prepared = { entry: SyncOutboxRow; commit: Commit };

function prepare(entries: SyncOutboxRow[]): Prepared[] {
  const db = databaseManager.getDb();
  const prepared: Prepared[] = [];
  for (const entry of entries) {
    if (!isSyncTable(entry.tableName)) {
      remove(entry.tableName as never, entry.rowId);
      continue;
    }
    const row = readLocalRow(db, entry.tableName, entry.rowId);
    if (!row) {
      // Rows are tombstoned, never deleted; a missing one was dropped as a natural-key duplicate.
      remove(entry.tableName, entry.rowId);
      continue;
    }
    const version = row.version as number;
    if (version <= entry.baseVersion) {
      fail(entry.opId, `version ${version} is not above baseVersion ${entry.baseVersion}`);
      continue;
    }
    prepared.push({
      entry,
      commit: {
        opId: entry.opId,
        table: entry.tableName,
        rowId: entry.rowId,
        op: entry.op,
        row: encodeRow(entry.tableName, row),
        baseVersion: entry.baseVersion,
        version,
        updatedAtMs: row.updatedAtMs as number,
        lastWriterId: (row.lastWriterId as string | null) ?? '',
      },
    });
  }
  return prepared;
}

/**
 * Settles one conflict with the shared last-writer-wins rule. If the server's row wins it
 * replaces the local one; if the local row wins, it is lifted above the server's version and
 * pushed again (the server does not record conflicted opIds, so the same one can be reused).
 */
function settle(conflict: Conflict, item: Prepared): void {
  const { entry, commit } = item;
  const table = entry.tableName as keyof typeof SYNCED_TABLES;
  const serverKey = {
    updatedAtMs: conflict.serverUpdatedAtMs,
    lastWriterId: conflict.serverLastWriterId,
  };
  const localKey = { updatedAtMs: commit.updatedAtMs, lastWriterId: commit.lastWriterId };

  if (conflict.serverRow && wins(localKey, serverKey) === 'b') {
    applyServerRows([{ table, row: conflict.serverRow }]);
    return;
  }

  const db = databaseManager.getDb();
  const target = SYNCED_TABLES[table];
  db.transaction((tx) => {
    // Raising the version fires the update trigger, which folds into this entry.
    tx.update(target)
      .set({ version: sql`max(${target.version}, ${conflict.serverVersion}) + 1` } as never)
      .where(eq(target.id, entry.rowId))
      .run();
    tx.update(syncOutbox)
      .set({ baseVersion: conflict.serverVersion, attempts: 0, lastError: null })
      .where(and(eq(syncOutbox.tableName, entry.tableName), eq(syncOutbox.rowId, entry.rowId)))
      .run();
  });
}

async function send(ctx: PushContext, items: Prepared[]): Promise<PushResponse> {
  return request<PushResponse>(ctx.url, 'POST', '/sync/push', {
    token: ctx.token,
    body: { clientId: getClientId(), commits: items.map((i) => i.commit) },
  });
}

function isRejectedCommit(error: unknown): boolean {
  return (
    error instanceof SyncError &&
    (error.code === ERROR_CODES.invalidCommit || error.code === ERROR_CODES.foreignTeam)
  );
}

/**
 * A rejected batch says nothing about which commit was wrong, so it is retried one commit at a
 * time; only the bad ones are marked failed and the rest go through.
 */
async function sendIsolating(ctx: PushContext, items: Prepared[], result: PushResult) {
  for (const item of items) {
    try {
      handle(await send(ctx, [item]), [item], result);
    } catch (error) {
      if (!isRejectedCommit(error)) {
        throw error;
      }
      fail(item.entry.opId, (error as SyncError).code ?? 'rejected');
      result.failed++;
      logger.warn('Sync', `Rejected ${item.entry.tableName}/${item.entry.rowId}`);
    }
  }
}

function handle(response: PushResponse, items: Prepared[], result: PushResult): void {
  const byOp = new Map(items.map((i) => [i.entry.opId, i]));
  const accepted = response.accepted
    .map((opId) => byOp.get(opId))
    .filter((i): i is Prepared => i !== undefined)
    .map((i) => ({ opId: i.entry.opId, rowVersion: i.entry.rowVersion }));
  ack(accepted);
  result.accepted += accepted.length;

  for (const conflict of response.conflicts) {
    const item = byOp.get(conflict.opId);
    if (item) {
      settle(conflict, item);
      result.conflicts++;
    }
  }
}

/** Sends the outbox until it is empty or only holds entries that keep failing. */
export async function push(ctx: PushContext): Promise<PushResult> {
  const result: PushResult = { accepted: 0, conflicts: 0, failed: 0 };
  // Bounded: a conflict re-queues its entry, and a row that keeps conflicting must not spin.
  for (let round = 0; round < 50; round++) {
    const items = prepare(peekBatch(PUSH_BATCH_SIZE));
    if (items.length === 0) {
      break;
    }
    try {
      handle(await send(ctx, items), items, result);
    } catch (error) {
      if (!isRejectedCommit(error)) {
        throw error;
      }
      await sendIsolating(ctx, items, result);
    }
  }
  return result;
}
