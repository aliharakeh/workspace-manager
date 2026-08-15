ALTER TABLE `ready_url_patterns` ADD `key` text;--> statement-breakpoint
CREATE UNIQUE INDEX `ready_url_patterns_key_unique` ON `ready_url_patterns` (`key`);
