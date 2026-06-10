{{if .Sqlite}}
create table `project__device_winrm_exec_log` (
  `id` integer primary key autoincrement,
  `project_id` integer not null,
  `device_id` integer not null,
  `user_id` integer not null,
  `username` varchar(255) not null default '',
  `credential_mode` varchar(20) not null default 'winrm',
  `shell` varchar(20) not null default 'powershell',
  `command` text not null,
  `ok` integer not null default 0,
  `exit_code` integer null,
  `error_code` varchar(64) null,
  `error_message` text null,
  `stdout` text not null default '',
  `stderr` text not null default '',
  `output_truncated` integer not null default 0,
  `duration_ms` integer not null default 0,
  `resolved_host` varchar(255) not null default '',
  `resolved_port` integer not null default 0,
  `resolved_user` varchar(255) not null default '',
  `created` datetime not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  foreign key (`user_id`) references `user`(`id`) on delete cascade
);
create index `project__device_winrm_exec_log_device_created_idx` on `project__device_winrm_exec_log` (`project_id`, `device_id`, `created`);
{{else if .Postgresql}}
create table `project__device_winrm_exec_log` (
  `id` serial primary key,
  `project_id` int not null,
  `device_id` int not null,
  `user_id` int not null,
  `username` varchar(255) not null default '',
  `credential_mode` varchar(20) not null default 'winrm',
  `shell` varchar(20) not null default 'powershell',
  `command` text not null,
  `ok` boolean not null default false,
  `exit_code` int null,
  `error_code` varchar(64) null,
  `error_message` text null,
  `stdout` text not null default '',
  `stderr` text not null default '',
  `output_truncated` boolean not null default false,
  `duration_ms` int not null default 0,
  `resolved_host` varchar(255) not null default '',
  `resolved_port` int not null default 0,
  `resolved_user` varchar(255) not null default '',
  `created` timestamptz not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  foreign key (`user_id`) references `user`(`id`) on delete cascade
);
create index `project__device_winrm_exec_log_device_created_idx` on `project__device_winrm_exec_log` (`project_id`, `device_id`, `created`);
{{else}}
create table `project__device_winrm_exec_log` (
  `id` int auto_increment primary key,
  `project_id` int not null,
  `device_id` int not null,
  `user_id` int not null,
  `username` varchar(255) not null default '',
  `credential_mode` varchar(20) not null default 'winrm',
  `shell` varchar(20) not null default 'powershell',
  `command` text not null,
  `ok` tinyint(1) not null default 0,
  `exit_code` int null,
  `error_code` varchar(64) null,
  `error_message` text null,
  `stdout` text not null,
  `stderr` text not null,
  `output_truncated` tinyint(1) not null default 0,
  `duration_ms` int not null default 0,
  `resolved_host` varchar(255) not null default '',
  `resolved_port` int not null default 0,
  `resolved_user` varchar(255) not null default '',
  `created` datetime not null,

  key `project__device_winrm_exec_log_device_created_idx` (`project_id`, `device_id`, `created`),
  constraint `project__device_winrm_exec_log_project_fk` foreign key (`project_id`) references `project`(`id`) on delete cascade,
  constraint `project__device_winrm_exec_log_device_fk` foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  constraint `project__device_winrm_exec_log_user_fk` foreign key (`user_id`) references `user`(`id`) on delete cascade
);
{{end}}
