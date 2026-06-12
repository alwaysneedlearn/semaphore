{{if .Sqlite}}
-- project__device_settings: rebuilt in migration_2_18_25.go (SQLite FK blocks drop column)
alter table `project__device_profile_settings`
  drop column `start_template_id`;
{{else}}
alter table `project__device_settings`
  drop column `start_template_id`;

alter table `project__device_profile_settings`
  drop column `start_template_id`;
{{end}}
