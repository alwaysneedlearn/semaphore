{{if .Sqlite}}
create table `project__device_discovery_host` (
  `id` integer primary key autoincrement,
  `project_id` integer not null,
  `ip_address` varchar(64) not null,
  `hostname` varchar(255) not null default '',
  `device_status` varchar(20) not null default 'unknown',
  `rdp_status` varchar(20) not null default 'offline',
  `winrm_status` varchar(20) not null default 'offline',
  `api_status` varchar(20) not null default 'offline',
  `api_port` integer not null default 0,
  `abnormal_reason` text null,
  `last_task_id` integer not null default 0,
  `updated` datetime not null,
  unique (`project_id`, `ip_address`)
);
create index `project__device_discovery_host_project_id_idx` on `project__device_discovery_host` (`project_id`);
create index `project__device_discovery_host_last_task_id_idx` on `project__device_discovery_host` (`project_id`, `last_task_id`);
{{else if .Postgresql}}
create table `project__device_discovery_host` (
  `id` serial primary key,
  `project_id` int not null,
  `ip_address` varchar(64) not null,
  `hostname` varchar(255) not null default '',
  `device_status` varchar(20) not null default 'unknown',
  `rdp_status` varchar(20) not null default 'offline',
  `winrm_status` varchar(20) not null default 'offline',
  `api_status` varchar(20) not null default 'offline',
  `api_port` int not null default 0,
  `abnormal_reason` text null,
  `last_task_id` int not null default 0,
  `updated` timestamptz not null,
  unique (`project_id`, `ip_address`)
);
create index `project__device_discovery_host_project_id_idx` on `project__device_discovery_host` (`project_id`);
create index `project__device_discovery_host_last_task_id_idx` on `project__device_discovery_host` (`project_id`, `last_task_id`);
{{else}}
create table `project__device_discovery_host` (
  `id` int auto_increment primary key,
  `project_id` int not null,
  `ip_address` varchar(64) not null,
  `hostname` varchar(255) not null default '',
  `device_status` varchar(20) not null default 'unknown',
  `rdp_status` varchar(20) not null default 'offline',
  `winrm_status` varchar(20) not null default 'offline',
  `api_status` varchar(20) not null default 'offline',
  `api_port` int not null default 0,
  `abnormal_reason` text null,
  `last_task_id` int not null default 0,
  `updated` datetime not null,
  unique key `project_discovery_ip` (`project_id`, `ip_address`),
  key `project__device_discovery_host_last_task_id_idx` (`project_id`, `last_task_id`)
);
{{end}}
