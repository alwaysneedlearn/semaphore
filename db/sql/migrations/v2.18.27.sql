{{if .Sqlite}}
alter table `project__device_profile_settings` add column `resend_data_template_id` integer null references `project__template`(`id`) on delete set null;
{{else if .Postgresql}}
alter table `project__device_profile_settings` add column `resend_data_template_id` int null references `project__template`(`id`) on delete set null;
{{else}}
alter table `project__device_profile_settings`
  add column `resend_data_template_id` int null,
  add constraint `project__device_profile_settings_resend_tpl_fk`
    foreign key (`resend_data_template_id`) references `project__template`(`id`) on delete set null;
{{end}}
