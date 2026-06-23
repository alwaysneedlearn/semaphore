{{if .Sqlite}}
-- SQLite: column drop not supported in rollback; no-op
{{else if .Postgresql}}
alter table `project__device_profile_settings` drop column if exists `resend_data_template_id`;
{{else}}
alter table `project__device_profile_settings` drop foreign key `project__device_profile_settings_resend_tpl_fk`;
alter table `project__device_profile_settings` drop column `resend_data_template_id`;
{{end}}
