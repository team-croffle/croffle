import {
  decodeRow,
  encodeRow,
  ERROR_CODES,
  type BootstrapResponse,
  type Change,
  type PullResponse,
} from '@croffledev/common';

import { databaseManager } from '../database';
import { logger } from '../logger';
import { applyServerRows, isSyncTable, readLocalRow, type ServerRow } from './apply';
import { assertBinding, getServer, updateServerState } from './sync-config';
import { request, SyncError } from './transport';

export type PullContext = { url: string; token: string; generation: number };

const PAGE_LIMIT = 500;

/**
 * Turns a page into rows to apply, failing on anything this version cannot read. Whether an
 * unsent local edit wins over a row is decided when it is applied (`applyServerRows`).
 */
function selectApplicable(changes: Change[]): ServerRow[] {
  const db = databaseManager.getDb();
  const rows: ServerRow[] = [];
  for (const change of changes) {
    if (!isSyncTable(change.table)) {
      logger.warn('Sync', `Skipping a change for unknown table ${change.table}`);
      continue;
    }
    let wire = change.row;
    if (!wire) {
      // A delete may come without its row: tombstone the local copy, if there is one.
      const local = change.op === 'delete' ? readLocalRow(db, change.table, change.rowId) : null;
      if (!local) {
        if (change.op === 'delete') {
          continue;
        }
        throw new SyncError(
          'protocol',
          `The server sent ${change.table}/${change.rowId} without a row`,
        );
      }
      wire = {
        ...encodeRow(change.table, local),
        version: change.version,
        updatedAtMs: change.updatedAtMs,
        deletedAt: change.updatedAtMs,
      };
    }
    try {
      decodeRow(change.table, wire);
    } catch (error) {
      // Fail the page rather than skip the row: the cursor must not move past a change that was
      // never applied, or it is lost for good. A newer Croffle can read it later.
      throw new SyncError(
        'protocol',
        `The server sent a ${change.table} row this version cannot read: ${String(error)}`,
      );
    }
    rows.push({ table: change.table, row: wire });
  }
  return rows;
}

function isCode(error: unknown, code: string): boolean {
  return error instanceof SyncError && error.code === code;
}

/**
 * Current state instead of history: for a new device, and for one whose cursor is older than
 * the server keeps. The cursor is saved after every page, so an interrupted snapshot resumes.
 * The pull cursor moves only after the last page, to the instant the snapshot was taken.
 */
export async function bootstrap(ctx: PullContext): Promise<number> {
  let cursor = getServer()?.bootstrapCursor ?? undefined;
  let applied = 0;
  let restarted = false;
  for (;;) {
    assertBinding(ctx.generation);
    let page: BootstrapResponse;
    try {
      page = await request<BootstrapResponse>(ctx.url, 'POST', '/sync/bootstrap', {
        token: ctx.token,
        body: { ...(cursor ? { cursor } : {}), limit: PAGE_LIMIT },
      });
    } catch (error) {
      // A cursor from before a server reset: start the snapshot over, once.
      if (cursor && !restarted && isCode(error, ERROR_CODES.invalidCursor)) {
        restarted = true;
        cursor = undefined;
        assertBinding(ctx.generation);
        updateServerState({ bootstrapCursor: null });
        continue;
      }
      throw error;
    }

    assertBinding(ctx.generation);
    applied += applyServerRows(selectApplicable(page.rows));
    if (page.hasMore && page.cursor) {
      cursor = page.cursor;
      updateServerState({ bootstrapCursor: cursor });
      continue;
    }
    updateServerState({
      lastPulledSeq: page.serverSeq,
      bootstrapCursor: null,
      bootstrappedAt: Date.now(),
    });
    return applied;
  }
}

/** History since the cursor, page by page. Falls back to a snapshot when the cursor expired. */
export async function pull(ctx: PullContext): Promise<number> {
  let applied = 0;
  for (;;) {
    assertBinding(ctx.generation);
    const lastServerSeq = getServer()?.lastPulledSeq ?? 0;
    let page: PullResponse;
    try {
      page = await request<PullResponse>(ctx.url, 'POST', '/sync/pull', {
        token: ctx.token,
        body: { lastServerSeq, limit: PAGE_LIMIT },
      });
    } catch (error) {
      if (isCode(error, ERROR_CODES.cursorTooOld)) {
        logger.info('Sync', 'Cursor is older than the server keeps; taking a snapshot');
        return applied + (await bootstrap(ctx));
      }
      throw error;
    }

    assertBinding(ctx.generation);
    applied += applyServerRows(selectApplicable(page.changes));
    // Never move past what was returned: a cursor beyond an unreturned commit loses it.
    if (page.serverSeq > lastServerSeq) {
      updateServerState({ lastPulledSeq: page.serverSeq });
    }
    if (!page.hasMore || page.serverSeq <= lastServerSeq) {
      return applied;
    }
  }
}
