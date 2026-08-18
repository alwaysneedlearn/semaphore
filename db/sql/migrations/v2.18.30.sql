{{if .Sqlite}}
-- SQLite handled in migration_2_18_30.PreApply (table rebuild).
{{else if .Postgresql}}
alter table `project__device_profile_settings`
  drop column if exists `stop_template_id`;

alter table `project__device_settings`
  drop column if exists `stop_template_id`;
{{else}}
alter table `project__device_profile_settings`
  drop column `stop_template_id`;

alter table `project__device_settings`
  drop column `stop_template_id`;
{{end}}
