{{if .Sqlite}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NEWARE', 'NEWARE', 1, datetime('now')
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NEWARE'
)
and (
  exists (select 1 from `project__device_settings` ds where ds.`project_id` = p.`id`)
  or exists (select 1 from `project__device` d where d.`project_id` = p.`id`)
);

insert into `project__device_profile_settings` (
  `project_id`, `profile_id`,
  `discover_template_id`, `start_template_id`, `stop_template_id`,
  `restart_template_id`, `status_template_id`, `config_template_id`,
  `default_inventory_id`, `default_ansible_user`, `default_ansible_password`,
  `default_ansible_connection`, `default_ansible_winrm_transport`, `default_ansible_winrm_scheme`,
  `default_ansible_port`, `default_ansible_winrm_server_cert_validation`,
  `default_config_json`, `status_refresh_interval_min`, `tdengine_status_table`
)
select
  ds.`project_id`, dp.`id`,
  ds.`discover_template_id`, ds.`start_template_id`, ds.`stop_template_id`,
  ds.`restart_template_id`, ds.`status_template_id`, ds.`config_template_id`,
  ds.`default_inventory_id`, ds.`default_ansible_user`, ds.`default_ansible_password`,
  ds.`default_ansible_connection`, ds.`default_ansible_winrm_transport`, ds.`default_ansible_winrm_scheme`,
  ds.`default_ansible_port`, ds.`default_ansible_winrm_server_cert_validation`,
  ds.`default_config_json`, ds.`status_refresh_interval_min`, 'status'
from `project__device_settings` ds
inner join `project__device_profile` dp on dp.`project_id` = ds.`project_id` and dp.`profile_key` = 'NEWARE'
where not exists (
  select 1 from `project__device_profile_settings` ps
  where ps.`project_id` = ds.`project_id` and ps.`profile_id` = dp.`id`
);

update `project__device_profile_settings` set
  `discover_template_id` = (select ds.`discover_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `start_template_id` = (select ds.`start_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `stop_template_id` = (select ds.`stop_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `restart_template_id` = (select ds.`restart_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `status_template_id` = (select ds.`status_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `config_template_id` = (select ds.`config_template_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `default_inventory_id` = (select ds.`default_inventory_id` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`),
  `default_ansible_user` = coalesce(nullif(`default_ansible_user`, ''), (select ds.`default_ansible_user` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_ansible_password` = coalesce(nullif(`default_ansible_password`, ''), (select ds.`default_ansible_password` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_ansible_connection` = coalesce(nullif(`default_ansible_connection`, ''), (select ds.`default_ansible_connection` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_ansible_winrm_transport` = coalesce(nullif(`default_ansible_winrm_transport`, ''), (select ds.`default_ansible_winrm_transport` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_ansible_winrm_scheme` = coalesce(nullif(`default_ansible_winrm_scheme`, ''), (select ds.`default_ansible_winrm_scheme` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_ansible_port` = case when `default_ansible_port` = 0 then (select ds.`default_ansible_port` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`) else `default_ansible_port` end,
  `default_ansible_winrm_server_cert_validation` = coalesce(nullif(`default_ansible_winrm_server_cert_validation`, ''), (select ds.`default_ansible_winrm_server_cert_validation` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `default_config_json` = coalesce(nullif(`default_config_json`, ''), (select ds.`default_config_json` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`)),
  `status_refresh_interval_min` = case when `status_refresh_interval_min` = 0 then (select ds.`status_refresh_interval_min` from `project__device_settings` ds where ds.`project_id` = `project__device_profile_settings`.`project_id`) else `status_refresh_interval_min` end
where exists (
  select 1 from `project__device_profile` dp
  inner join `project__device_settings` ds on ds.`project_id` = dp.`project_id`
  where dp.`id` = `project__device_profile_settings`.`profile_id`
    and dp.`profile_key` = 'NEWARE'
    and dp.`project_id` = `project__device_profile_settings`.`project_id`
);

update `project__device` set `device_profile_id` = (
  select dp.`id` from `project__device_profile` dp
  where dp.`project_id` = `project__device`.`project_id` and dp.`profile_key` = 'NEWARE' limit 1
)
where `device_profile_id` = 0 or `device_profile_id` is null;
{{else if .Postgresql}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NEWARE', 'NEWARE', true, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NEWARE'
)
and (
  exists (select 1 from `project__device_settings` ds where ds.`project_id` = p.`id`)
  or exists (select 1 from `project__device` d where d.`project_id` = p.`id`)
);

insert into `project__device_profile_settings` (
  `project_id`, `profile_id`,
  `discover_template_id`, `start_template_id`, `stop_template_id`,
  `restart_template_id`, `status_template_id`, `config_template_id`,
  `default_inventory_id`, `default_ansible_user`, `default_ansible_password`,
  `default_ansible_connection`, `default_ansible_winrm_transport`, `default_ansible_winrm_scheme`,
  `default_ansible_port`, `default_ansible_winrm_server_cert_validation`,
  `default_config_json`, `status_refresh_interval_min`, `tdengine_status_table`
)
select
  ds.`project_id`, dp.`id`,
  ds.`discover_template_id`, ds.`start_template_id`, ds.`stop_template_id`,
  ds.`restart_template_id`, ds.`status_template_id`, ds.`config_template_id`,
  ds.`default_inventory_id`, ds.`default_ansible_user`, ds.`default_ansible_password`,
  ds.`default_ansible_connection`, ds.`default_ansible_winrm_transport`, ds.`default_ansible_winrm_scheme`,
  ds.`default_ansible_port`, ds.`default_ansible_winrm_server_cert_validation`,
  ds.`default_config_json`, ds.`status_refresh_interval_min`, 'status'
from `project__device_settings` ds
inner join `project__device_profile` dp on dp.`project_id` = ds.`project_id` and dp.`profile_key` = 'NEWARE'
where not exists (
  select 1 from `project__device_profile_settings` ps
  where ps.`project_id` = ds.`project_id` and ps.`profile_id` = dp.`id`
);

update `project__device` d set `device_profile_id` = dp.`id`
from `project__device_profile` dp
where dp.`project_id` = d.`project_id` and dp.`profile_key` = 'NEWARE'
  and (d.`device_profile_id` = 0 or d.`device_profile_id` is null);
{{else}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NEWARE', 'NEWARE', 1, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NEWARE'
)
and (
  exists (select 1 from `project__device_settings` ds where ds.`project_id` = p.`id`)
  or exists (select 1 from `project__device` d where d.`project_id` = p.`id`)
);

insert into `project__device_profile_settings` (
  `project_id`, `profile_id`,
  `discover_template_id`, `start_template_id`, `stop_template_id`,
  `restart_template_id`, `status_template_id`, `config_template_id`,
  `default_inventory_id`, `default_ansible_user`, `default_ansible_password`,
  `default_ansible_connection`, `default_ansible_winrm_transport`, `default_ansible_winrm_scheme`,
  `default_ansible_port`, `default_ansible_winrm_server_cert_validation`,
  `default_config_json`, `status_refresh_interval_min`, `tdengine_status_table`
)
select
  ds.`project_id`, dp.`id`,
  ds.`discover_template_id`, ds.`start_template_id`, ds.`stop_template_id`,
  ds.`restart_template_id`, ds.`status_template_id`, ds.`config_template_id`,
  ds.`default_inventory_id`, ds.`default_ansible_user`, ds.`default_ansible_password`,
  ds.`default_ansible_connection`, ds.`default_ansible_winrm_transport`, ds.`default_ansible_winrm_scheme`,
  ds.`default_ansible_port`, ds.`default_ansible_winrm_server_cert_validation`,
  ds.`default_config_json`, ds.`status_refresh_interval_min`, 'status'
from `project__device_settings` ds
inner join `project__device_profile` dp on dp.`project_id` = ds.`project_id` and dp.`profile_key` = 'NEWARE'
where not exists (
  select 1 from `project__device_profile_settings` ps
  where ps.`project_id` = ds.`project_id` and ps.`profile_id` = dp.`id`
);

update `project__device` d
inner join `project__device_profile` dp on dp.`project_id` = d.`project_id` and dp.`profile_key` = 'NEWARE'
set d.`device_profile_id` = dp.`id`
where d.`device_profile_id` = 0 or d.`device_profile_id` is null;
{{end}}
