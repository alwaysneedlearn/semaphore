alter table `project__device_profile_settings`
  drop column `check_restart_redeploy_template_id`;

alter table `project__device_profile_settings`
  add column `redeploy_template_id` int null,
  add column `check_restart_template_id` int null;
