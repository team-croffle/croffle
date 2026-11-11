// Self-contained (no imports): the Node test runner and the migration generator load this file
// directly, without the bundler's path aliases.

/** `lastWriterId` of rows applied from the sync server. Outbox triggers skip them (no ping-pong). */
export const SERVER_WRITER_ID = 'server';

/** Tables exchanged with the sync server, parents before children (FK order). */
export const SYNC_TABLES = [
  'tag',
  'schedule',
  'schedule_tags',
  'schedule_reminder',
  'schedule_exception',
] as const;

export type SyncTableName = (typeof SYNC_TABLES)[number];

type TriggerEvent = 'INSERT' | 'UPDATE';

export function outboxTriggerName(table: SyncTableName, event: TriggerEvent): string {
  return `sync_outbox_${table}_${event === 'INSERT' ? 'ai' : 'au'}`;
}

/**
 * One outbox trigger. Later edits of a queued row fold into its entry and keep `opId` and
 * `baseVersion`: the server checks conflicts against the version it held before the first
 * unsynced change. Rows written by the sync engine carry `lastWriterId = 'server'` and are
 * skipped, so applied changes are not pushed back.
 *
 * Migration 0008 holds exactly this text (a test compares them); `ensureOutboxTriggers`
 * restores it after a migration that rebuilds a table, since SQLite drops a table's triggers.
 */
export function outboxTriggerSql(
  table: SyncTableName,
  event: TriggerEvent,
  ifNotExists = false,
): string {
  const baseVersion = event === 'INSERT' ? '0' : 'OLD.version';
  return [
    `CREATE TRIGGER ${ifNotExists ? 'IF NOT EXISTS ' : ''}\`${outboxTriggerName(table, event)}\``,
    `AFTER ${event} ON \`${table}\``,
    `WHEN NEW.lastWriterId IS NOT '${SERVER_WRITER_ID}'`,
    'BEGIN',
    '  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)',
    `  VALUES (lower(hex(randomblob(16))), '${table}', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, ${baseVersion}, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)`,
    '  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET',
    '    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;',
    'END;',
  ].join('\n');
}

/** Every outbox trigger, in table order (INSERT then UPDATE). */
export function allOutboxTriggerSql(ifNotExists = false): string[] {
  return SYNC_TABLES.flatMap((table) => [
    outboxTriggerSql(table, 'INSERT', ifNotExists),
    outboxTriggerSql(table, 'UPDATE', ifNotExists),
  ]);
}

export const OUTBOX_TRIGGER_COUNT = SYNC_TABLES.length * 2;

/** The body of migration 0008, statements separated the way drizzle's migrator splits them. */
export function outboxTriggerMigrationSql(): string {
  return `${allOutboxTriggerSql().join('\n--> statement-breakpoint\n')}\n`;
}
