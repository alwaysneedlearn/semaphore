{{if .Sqlite}}
-- no-op rollback for v2.18.31
{{else if .Postgresql}}
alter table `project__device_profile_settings`
  add column `discover_template_id` int null,
  add column `check_restart_template_id` int null,
  add column `config_template_id` int null,
  add column `status_refresh_interval_min` int not null default 0,
  add column `last_status_refresh_at` timestamp null;

alter table `project__device_settings`
  add column `config_template_id` int null,
  add column `status_refresh_interval_min` int not null default 0,
  add column `last_status_refresh_at` timestamp null;
{{else}}
alter table `project__device_profile_settings`
  add column `discover_template_id` int null,
  add column `check_restart_template_id` int null,
  add column `config_template_id` int null,
  add column `status_refresh_interval_min` int not null default 0,
  add column `last_status_refresh_at` datetime null;

alter table `project__device_settings`
  add column `config_template_id` int null,
  add column `status_refresh_interval_min` int not null default 0,
  add column `last_status_refresh_at` datetime null,
  add constraint `project__device_settings_config_template_id_fkey`
    foreign key (`config_template_id`) references `project__template`(`id`) on delete set null;
{{end}}
