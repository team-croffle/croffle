ALTER TABLE `schedule` ADD `kind` text DEFAULT 'event' NOT NULL;--> statement-breakpoint
ALTER TABLE `schedule` ADD `completedAt` integer;