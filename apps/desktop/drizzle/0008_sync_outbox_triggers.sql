CREATE TRIGGER `sync_outbox_tag_ai`
AFTER INSERT ON `tag`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'tag', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, 0, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_tag_au`
AFTER UPDATE ON `tag`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'tag', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, OLD.version, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_ai`
AFTER INSERT ON `schedule`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, 0, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_au`
AFTER UPDATE ON `schedule`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, OLD.version, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_tags_ai`
AFTER INSERT ON `schedule_tags`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_tags', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, 0, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_tags_au`
AFTER UPDATE ON `schedule_tags`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_tags', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, OLD.version, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_reminder_ai`
AFTER INSERT ON `schedule_reminder`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_reminder', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, 0, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_reminder_au`
AFTER UPDATE ON `schedule_reminder`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_reminder', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, OLD.version, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_exception_ai`
AFTER INSERT ON `schedule_exception`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_exception', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, 0, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
--> statement-breakpoint
CREATE TRIGGER `sync_outbox_schedule_exception_au`
AFTER UPDATE ON `schedule_exception`
WHEN NEW.lastWriterId IS NOT 'server'
BEGIN
  INSERT INTO `sync_outbox` (`opId`, `tableName`, `rowId`, `op`, `baseVersion`, `rowVersion`, `updatedAtMs`, `enqueuedAt`, `attempts`)
  VALUES (lower(hex(randomblob(16))), 'schedule_exception', NEW.id, CASE WHEN NEW.deletedAt IS NULL THEN 'upsert' ELSE 'delete' END, OLD.version, NEW.version, NEW.updatedAtMs, CAST(unixepoch('subsec') * 1000 AS INTEGER), 0)
  ON CONFLICT (`tableName`, `rowId`) DO UPDATE SET
    `op` = excluded.`op`, `rowVersion` = excluded.`rowVersion`, `updatedAtMs` = excluded.`updatedAtMs`, `attempts` = 0, `lastError` = NULL;
END;
