{{if .Sqlite}}
create table `project__device_operation_log` (
  `id` integer primary key autoincrement,
  `project_id` integer not null,
  `device_id` integer not null,
  `task_id` integer null,
  `operation` varchar(32) not null,
  `result` varchar(32) not null,
  `summary` text not null default '',
  `steps_json` text not null default '[]',
  `created` datetime not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade
);
create index `project__device_operation_log_device_created_idx` on `project__device_operation_log` (`project_id`, `device_id`, `created`);
{{else if .Postgresql}}
create table `project__device_operation_log` (
  `id` serial primary key,
  `project_id` int not null,
  `device_id` int not null,
  `task_id` int null,
  `operation` varchar(32) not null,
  `result` varchar(32) not null,
  `summary` text not null default '',
  `steps_json` text not null default '[]',
  `created` timestamptz not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade
);
create index `project__device_operation_log_device_created_idx` on `project__device_operation_log` (`project_id`, `device_id`, `created`);
{{else}}
create table `project__device_operation_log` (
  `id` int auto_increment primary key,
  `project_id` int not null,
  `device_id` int not null,
  `task_id` int null,
  `operation` varchar(32) not null,
  `result` varchar(32) not null,
  `summary` text not null,
  `steps_json` text not null,
  `created` datetime not null,

  key `project__device_operation_log_device_created_idx` (`project_id`, `device_id`, `created`),
  constraint `project__device_operation_log_project_fk` foreign key (`project_id`) references `project`(`id`) on delete cascade,
  constraint `project__device_operation_log_device_fk` foreign key (`device_id`) references `project__device`(`id`) on delete cascade
);
{{end}}
