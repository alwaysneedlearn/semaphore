{{if .Sqlite}}
create table `project__device_profile` (
  `id` integer primary key autoincrement,
  `project_id` integer not null,
  `profile_key` varchar(64) not null,
  `name` varchar(255) not null,
  `enabled` integer not null default 1,
  `created` datetime not null,
  unique (`project_id`, `profile_key`)
);
create index `project__device_profile_project_id_idx` on `project__device_profile` (`project_id`);

create table `project__device_profile_settings` (
  `project_id` integer not null,
  `profile_id` integer not null,
  `discover_template_id` integer null,
  `start_template_id` integer null,
  `stop_template_id` integer null,
  `restart_template_id` integer null,
  `status_template_id` integer null,
  `config_template_id` integer null,
  `default_inventory_id` integer null,
  `default_ansible_user` varchar(255) not null default '',
  `default_ansible_password` text not null default '',
  `default_ansible_connection` varchar(64) not null default 'winrm',
  `default_ansible_winrm_transport` varchar(64) not null default 'ntlm',
  `default_ansible_winrm_scheme` varchar(16) not null default 'http',
  `default_ansible_port` integer not null default 5985,
  `default_ansible_winrm_server_cert_validation` varchar(32) not null default 'ignore',
  `default_config_json` text not null default '',
  `status_refresh_interval_min` integer not null default 0,
  `last_status_refresh_at` datetime null,
  `tdengine_status_table` varchar(128) not null default 'status',
  primary key (`project_id`, `profile_id`)
);

alter table `project__device` add column `device_profile_id` integer not null default 0;
{{else if .Postgresql}}
create table `project__device_profile` (
  `id` serial primary key,
  `project_id` int not null,
  `profile_key` varchar(64) not null,
  `name` varchar(255) not null,
  `enabled` boolean not null default true,
  `created` timestamptz not null,
  unique (`project_id`, `profile_key`)
);
create index `project__device_profile_project_id_idx` on `project__device_profile` (`project_id`);

create table `project__device_profile_settings` (
  `project_id` int not null,
  `profile_id` int not null,
  `discover_template_id` int null,
  `start_template_id` int null,
  `stop_template_id` int null,
  `restart_template_id` int null,
  `status_template_id` int null,
  `config_template_id` int null,
  `default_inventory_id` int null,
  `default_ansible_user` varchar(255) not null default '',
  `default_ansible_password` text not null default '',
  `default_ansible_connection` varchar(64) not null default 'winrm',
  `default_ansible_winrm_transport` varchar(64) not null default 'ntlm',
  `default_ansible_winrm_scheme` varchar(16) not null default 'http',
  `default_ansible_port` int not null default 5985,
  `default_ansible_winrm_server_cert_validation` varchar(32) not null default 'ignore',
  `default_config_json` text not null default '',
  `status_refresh_interval_min` int not null default 0,
  `last_status_refresh_at` timestamptz null,
  `tdengine_status_table` varchar(128) not null default 'status',
  primary key (`project_id`, `profile_id`)
);

alter table `project__device` add column `device_profile_id` int not null default 0;
{{else}}
create table `project__device_profile` (
  `id` int auto_increment primary key,
  `project_id` int not null,
  `profile_key` varchar(64) not null,
  `name` varchar(255) not null,
  `enabled` tinyint(1) not null default 1,
  `created` datetime not null,
  unique key `project_profile_key` (`project_id`, `profile_key`)
);

create table `project__device_profile_settings` (
  `project_id` int not null,
  `profile_id` int not null,
  `discover_template_id` int null,
  `start_template_id` int null,
  `stop_template_id` int null,
  `restart_template_id` int null,
  `status_template_id` int null,
  `config_template_id` int null,
  `default_inventory_id` int null,
  `default_ansible_user` varchar(255) not null default '',
  `default_ansible_password` longtext not null,
  `default_ansible_connection` varchar(64) not null default 'winrm',
  `default_ansible_winrm_transport` varchar(64) not null default 'ntlm',
  `default_ansible_winrm_scheme` varchar(16) not null default 'http',
  `default_ansible_port` int not null default 5985,
  `default_ansible_winrm_server_cert_validation` varchar(32) not null default 'ignore',
  `default_config_json` longtext not null,
  `status_refresh_interval_min` int not null default 0,
  `last_status_refresh_at` datetime null,
  `tdengine_status_table` varchar(128) not null default 'status',
  primary key (`project_id`, `profile_id`)
);

alter table `project__device` add column `device_profile_id` int not null default 0;
{{end}}
