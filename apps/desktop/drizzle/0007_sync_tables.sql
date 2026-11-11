CREATE TABLE `sync_outbox` (
	`seq` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`opId` text NOT NULL,
	`tableName` text NOT NULL,
	`rowId` text NOT NULL,
	`op` text NOT NULL,
	`baseVersion` integer NOT NULL,
	`rowVersion` integer NOT NULL,
	`updatedAtMs` integer NOT NULL,
	`enqueuedAt` integer NOT NULL,
	`attempts` integer DEFAULT 0 NOT NULL,
	`lastError` text
);
--> statement-breakpoint
CREATE UNIQUE INDEX `sync_outbox_tableName_rowId_unique` ON `sync_outbox` (`tableName`,`rowId`);--> statement-breakpoint
CREATE UNIQUE INDEX `sync_outbox_opId_unique` ON `sync_outbox` (`opId`);--> statement-breakpoint
CREATE TABLE `sync_server` (
	`id` text PRIMARY KEY NOT NULL,
	`url` text NOT NULL,
	`accountId` text,
	`teamId` text,
	`allowInsecureTls` integer DEFAULT false NOT NULL,
	`enabled` integer DEFAULT false NOT NULL,
	`lastPulledSeq` integer DEFAULT 0 NOT NULL,
	`bootstrapCursor` text,
	`bootstrappedAt` integer,
	`backfilledAt` integer,
	`lastSyncedAt` integer,
	`lastErrorKind` text,
	`lastError` text
);
