{{if .Sqlite}}
-- no-op rollback for v2.18.32
{{else if .Postgresql}}
alter table `project__device_settings`
  add column if not exists `restart_template_id` int null,
  add column if not exists `status_template_id` int null;
{{else}}
alter table `project__device_settings`
  add column `restart_template_id` int null,
  add column `status_template_id` int null;
{{end}}
