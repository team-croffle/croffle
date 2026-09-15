CREATE TABLE `schedule_reminder` (
	`id` text PRIMARY KEY NOT NULL,
	`scheduleId` text NOT NULL,
	`minutes` integer NOT NULL,
	FOREIGN KEY (`scheduleId`) REFERENCES `schedule`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `schedule_reminder_scheduleId_minutes_unique` ON `schedule_reminder` (`scheduleId`,`minutes`);
--> statement-breakpoint
ALTER TABLE `schedule` ADD `useDefaultReminder` integer DEFAULT true NOT NULL;
--> statement-breakpoint
INSERT INTO `schedule_reminder` (`id`, `scheduleId`, `minutes`) SELECT `id` || '-' || `reminderMinutes`, `id`, `reminderMinutes` FROM `schedule` WHERE `reminderMinutes` IS NOT NULL AND `reminderMinutes` > 0;
--> statement-breakpoint
UPDATE `schedule` SET `useDefaultReminder` = 0 WHERE `reminderMinutes` IS NOT NULL;
--> statement-breakpoint
ALTER TABLE `schedule` DROP COLUMN `reminderMinutes`;
