{{if .Sqlite}}
-- no-op rollback for v2.18.30
{{else if .Postgresql}}
alter table `project__device_profile_settings`
  add column `stop_template_id` int null;

alter table `project__device_settings`
  add column `stop_template_id` int null;
{{else}}
alter table `project__device_profile_settings`
  add column `stop_template_id` int null;

alter table `project__device_settings`
  add column `stop_template_id` int null,
  add constraint `project__device_settings_stop_template_id_fkey`
    foreign key (`stop_template_id`) references `project__template`(`id`) on delete set null;
{{end}}
