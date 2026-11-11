import { allOutboxTriggerSql, OUTBOX_TRIGGER_COUNT } from '@croffledev/common';
import type Database from 'better-sqlite3';

import { logger } from '../logger';

/** Re-creates any outbox trigger a table-rebuilding migration dropped. Runs after migrations. */
export function ensureOutboxTriggers(sqlite: Database.Database): void {
  const present = (
    sqlite
      .prepare(
        `SELECT COUNT(*) AS c FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'sync_outbox_%'`,
      )
      .get() as { c: number }
  ).c;
  if (present === OUTBOX_TRIGGER_COUNT) {
    return;
  }

  for (const statement of allOutboxTriggerSql(true)) {
    sqlite.exec(statement);
  }
  logger.warn(
    'DB',
    `Restored outbox triggers (${present} of ${OUTBOX_TRIGGER_COUNT} were present).`,
  );
}
