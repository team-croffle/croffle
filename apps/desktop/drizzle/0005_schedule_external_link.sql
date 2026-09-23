CREATE TABLE `schedule_external_link` (
	`id` text PRIMARY KEY NOT NULL,
	`scheduleId` text NOT NULL,
	`provider` text NOT NULL,
	`accountId` text NOT NULL,
	`externalId` text NOT NULL,
	`externalEtag` text,
	`externalUpdatedAtMs` integer,
	`updatedAtMs` integer DEFAULT 0 NOT NULL,
	`version` integer DEFAULT 1 NOT NULL,
	`deletedAt` integer,
	`lastWriterId` text,
	`ownerId` text,
	FOREIGN KEY (`scheduleId`) REFERENCES `schedule`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `schedule_external_link_provider_account_external_unique` ON `schedule_external_link` (`provider`,`accountId`,`externalId`);
--> statement-breakpoint
CREATE INDEX `schedule_external_link_scheduleId_idx` ON `schedule_external_link` (`scheduleId`);
