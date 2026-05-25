{{if .Sqlite}}
create table `project__device_discovery_run` (
  `task_id` integer primary key,
  `project_id` integer not null,
  `subnet` varchar(128) not null default '',
  `status` varchar(20) not null default 'pending',
  `devices_json` text not null default '[]',
  `updated` datetime not null
);
create index `project__device_discovery_run_project_id_idx` on `project__device_discovery_run` (`project_id`);
{{else if .Postgresql}}
create table `project__device_discovery_run` (
  `task_id` int primary key,
  `project_id` int not null,
  `subnet` varchar(128) not null default '',
  `status` varchar(20) not null default 'pending',
  `devices_json` text not null default '[]',
  `updated` timestamptz not null
);
create index `project__device_discovery_run_project_id_idx` on `project__device_discovery_run` (`project_id`);
{{else}}
create table `project__device_discovery_run` (
  `task_id` int not null,
  `project_id` int not null,
  `subnet` varchar(128) not null default '',
  `status` varchar(20) not null default 'pending',
  `devices_json` longtext not null,
  `updated` datetime not null,
  primary key (`task_id`)
);
create index `project__device_discovery_run_project_id_idx` on `project__device_discovery_run` (`project_id`);
{{end}}
