{{if .Sqlite}}
-- SQLite handled in migration_2_18_32.PreApply (table rebuild).
{{else if .Postgresql}}
alter table `project__device_settings`
  drop column if exists `restart_template_id`,
  drop column if exists `status_template_id`;
{{else}}
alter table `project__device_settings`
  drop column `restart_template_id`,
  drop column `status_template_id`;
{{end}}
