PRAGMA foreign_keys=OFF;--> statement-breakpoint
CREATE TABLE `config_sets` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`app_id` integer NOT NULL,
	`name` text NOT NULL,
	`created_at` text DEFAULT (datetime('now')) NOT NULL,
	`updated_at` text DEFAULT (datetime('now')) NOT NULL,
	FOREIGN KEY (`app_id`) REFERENCES `apps`(`id`) ON UPDATE no action ON DELETE cascade
);--> statement-breakpoint
CREATE INDEX `idx_config_sets_app_id` ON `config_sets` (`app_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `config_sets_app_id_name_unique` ON `config_sets` (`app_id`,`name`);--> statement-breakpoint
ALTER TABLE `apps` ADD `active_config_set_id` integer;--> statement-breakpoint
INSERT INTO `config_sets` (`app_id`, `name`)
SELECT `id`, 'Default' FROM `apps`;--> statement-breakpoint
UPDATE `apps`
SET `active_config_set_id` = (
	SELECT `id` FROM `config_sets` WHERE `config_sets`.`app_id` = `apps`.`id` LIMIT 1
);--> statement-breakpoint
CREATE TABLE `__new_env_vars` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`config_set_id` integer NOT NULL,
	`key` text NOT NULL,
	`value` text DEFAULT '' NOT NULL,
	`created_at` text DEFAULT (datetime('now')) NOT NULL,
	`updated_at` text DEFAULT (datetime('now')) NOT NULL,
	FOREIGN KEY (`config_set_id`) REFERENCES `config_sets`(`id`) ON UPDATE no action ON DELETE cascade
);--> statement-breakpoint
INSERT INTO `__new_env_vars` (`id`, `config_set_id`, `key`, `value`, `created_at`, `updated_at`)
SELECT `e`.`id`, `cs`.`id`, `e`.`key`, `e`.`value`, `e`.`created_at`, `e`.`updated_at`
FROM `env_vars` AS `e`
INNER JOIN `config_sets` AS `cs` ON `cs`.`app_id` = `e`.`app_id`;--> statement-breakpoint
DROP TABLE `env_vars`;--> statement-breakpoint
ALTER TABLE `__new_env_vars` RENAME TO `env_vars`;--> statement-breakpoint
CREATE INDEX `idx_env_vars_config_set_id` ON `env_vars` (`config_set_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `env_vars_config_set_id_key_unique` ON `env_vars` (`config_set_id`,`key`);--> statement-breakpoint
CREATE TABLE `__new_templates` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`config_set_id` integer NOT NULL,
	`file_path` text NOT NULL,
	`content` text DEFAULT '' NOT NULL,
	`created_at` text DEFAULT (datetime('now')) NOT NULL,
	`updated_at` text DEFAULT (datetime('now')) NOT NULL,
	FOREIGN KEY (`config_set_id`) REFERENCES `config_sets`(`id`) ON UPDATE no action ON DELETE cascade
);--> statement-breakpoint
INSERT INTO `__new_templates` (`id`, `config_set_id`, `file_path`, `content`, `created_at`, `updated_at`)
SELECT `t`.`id`, `cs`.`id`, `t`.`file_path`, `t`.`content`, `t`.`created_at`, `t`.`updated_at`
FROM `templates` AS `t`
INNER JOIN `config_sets` AS `cs` ON `cs`.`app_id` = `t`.`app_id`;--> statement-breakpoint
DROP TABLE `templates`;--> statement-breakpoint
ALTER TABLE `__new_templates` RENAME TO `templates`;--> statement-breakpoint
CREATE INDEX `idx_templates_config_set_id` ON `templates` (`config_set_id`);--> statement-breakpoint
CREATE UNIQUE INDEX `templates_config_set_id_file_path_unique` ON `templates` (`config_set_id`,`file_path`);--> statement-breakpoint
CREATE TABLE `__new_run_configs` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`config_set_id` integer NOT NULL,
	`mode` text DEFAULT 'parallel' NOT NULL,
	`created_at` text DEFAULT (datetime('now')) NOT NULL,
	`updated_at` text DEFAULT (datetime('now')) NOT NULL,
	FOREIGN KEY (`config_set_id`) REFERENCES `config_sets`(`id`) ON UPDATE no action ON DELETE cascade
);--> statement-breakpoint
INSERT INTO `__new_run_configs` (`id`, `config_set_id`, `mode`, `created_at`, `updated_at`)
SELECT `r`.`id`, `cs`.`id`, `r`.`mode`, `r`.`created_at`, `r`.`updated_at`
FROM `run_configs` AS `r`
INNER JOIN `config_sets` AS `cs` ON `cs`.`app_id` = `r`.`app_id`;--> statement-breakpoint
DROP TABLE `run_configs`;--> statement-breakpoint
ALTER TABLE `__new_run_configs` RENAME TO `run_configs`;--> statement-breakpoint
CREATE UNIQUE INDEX `run_configs_config_set_id_unique` ON `run_configs` (`config_set_id`);--> statement-breakpoint
PRAGMA foreign_keys=ON;
