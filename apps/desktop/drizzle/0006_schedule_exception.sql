CREATE TABLE `schedule_exception` (
	`id` text PRIMARY KEY NOT NULL,
	`scheduleId` text NOT NULL,
	`occurrenceStart` integer NOT NULL,
	`startDate` integer,
	`endDate` integer,
	`cancelled` integer DEFAULT false NOT NULL,
	`updatedAtMs` integer DEFAULT 0 NOT NULL,
	`version` integer DEFAULT 1 NOT NULL,
	`deletedAt` integer,
	`lastWriterId` text,
	`ownerId` text,
	FOREIGN KEY (`scheduleId`) REFERENCES `schedule`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `schedule_exception_scheduleId_occurrenceStart_unique` ON `schedule_exception` (`scheduleId`,`occurrenceStart`);
