ALTER TABLE `schedule` ADD `updatedAtMs` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `schedule` ADD `version` integer DEFAULT 1 NOT NULL;
--> statement-breakpoint
ALTER TABLE `schedule` ADD `deletedAt` integer;
--> statement-breakpoint
ALTER TABLE `schedule` ADD `lastWriterId` text;
--> statement-breakpoint
ALTER TABLE `schedule` ADD `ownerId` text;
--> statement-breakpoint
UPDATE `schedule` SET `updatedAtMs` = `updatedAt` * 1000;
--> statement-breakpoint
ALTER TABLE `tag` ADD `createdAt` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `tag` ADD `updatedAt` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `tag` ADD `updatedAtMs` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `tag` ADD `version` integer DEFAULT 1 NOT NULL;
--> statement-breakpoint
ALTER TABLE `tag` ADD `deletedAt` integer;
--> statement-breakpoint
ALTER TABLE `tag` ADD `lastWriterId` text;
--> statement-breakpoint
ALTER TABLE `tag` ADD `ownerId` text;
--> statement-breakpoint
UPDATE `tag` SET `createdAt` = strftime('%s', 'now'), `updatedAt` = strftime('%s', 'now'), `updatedAtMs` = strftime('%s', 'now') * 1000;
--> statement-breakpoint
DROP INDEX IF EXISTS `tag_name_unique`;
--> statement-breakpoint
ALTER TABLE `schedule_reminder` ADD `updatedAtMs` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `schedule_reminder` ADD `version` integer DEFAULT 1 NOT NULL;
--> statement-breakpoint
ALTER TABLE `schedule_reminder` ADD `deletedAt` integer;
--> statement-breakpoint
ALTER TABLE `schedule_reminder` ADD `lastWriterId` text;
--> statement-breakpoint
ALTER TABLE `schedule_reminder` ADD `ownerId` text;
--> statement-breakpoint
UPDATE `schedule_reminder` SET `updatedAtMs` = strftime('%s', 'now') * 1000;
--> statement-breakpoint
CREATE TABLE `__new_schedule_tags` (
	`id` text PRIMARY KEY NOT NULL,
	`scheduleId` text NOT NULL,
	`tagId` text NOT NULL,
	`updatedAtMs` integer DEFAULT 0 NOT NULL,
	`version` integer DEFAULT 1 NOT NULL,
	`deletedAt` integer,
	`lastWriterId` text,
	`ownerId` text,
	FOREIGN KEY (`scheduleId`) REFERENCES `schedule`(`id`) ON UPDATE no action ON DELETE cascade,
	FOREIGN KEY (`tagId`) REFERENCES `tag`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
INSERT INTO `__new_schedule_tags` (`id`, `scheduleId`, `tagId`, `updatedAtMs`, `version`, `deletedAt`, `lastWriterId`, `ownerId`) SELECT lower(hex(randomblob(16))), `scheduleId`, `tagId`, strftime('%s', 'now') * 1000, 1, NULL, NULL, NULL FROM `schedule_tags`;
--> statement-breakpoint
DROP TABLE `schedule_tags`;
--> statement-breakpoint
ALTER TABLE `__new_schedule_tags` RENAME TO `schedule_tags`;
--> statement-breakpoint
CREATE UNIQUE INDEX `schedule_tags_scheduleId_tagId_unique` ON `schedule_tags` (`scheduleId`,`tagId`);
--> statement-breakpoint
ALTER TABLE `extension_info` ADD `installedAt` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
ALTER TABLE `extension_info` ADD `updatedAt` integer DEFAULT 0 NOT NULL;
--> statement-breakpoint
UPDATE `extension_info` SET `installedAt` = strftime('%s', 'now'), `updatedAt` = strftime('%s', 'now');
