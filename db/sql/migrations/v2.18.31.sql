{{if .Sqlite}}
-- SQLite handled in migration_2_18_31.PreApply (table rebuild).
{{else if .Postgresql}}
alter table `project__device_profile_settings`
  drop column if exists `discover_template_id`,
  drop column if exists `check_restart_template_id`,
  drop column if exists `config_template_id`,
  drop column if exists `status_refresh_interval_min`,
  drop column if exists `last_status_refresh_at`;

alter table `project__device_settings`
  drop column if exists `config_template_id`,
  drop column if exists `status_refresh_interval_min`,
  drop column if exists `last_status_refresh_at`;
{{else}}
alter table `project__device_profile_settings`
  drop column `discover_template_id`,
  drop column `check_restart_template_id`,
  drop column `config_template_id`,
  drop column `status_refresh_interval_min`,
  drop column `last_status_refresh_at`;

alter table `project__device_settings`
  drop column `config_template_id`,
  drop column `status_refresh_interval_min`,
  drop column `last_status_refresh_at`;
{{end}}
