{{if .Sqlite}}
create table `project__device_rdp_launch_log` (
  `id` integer primary key autoincrement,
  `project_id` integer not null,
  `device_id` integer not null,
  `user_id` integer not null,
  `username` varchar(255) not null default '',
  `phase` varchar(32) not null default 'requested',
  `host` varchar(255) not null default '',
  `rdp_port` integer not null default 3389,
  `rdp_user` varchar(255) not null default '',
  `client_ip` varchar(64) not null default '',
  `created` datetime not null,
  `helper_fetched_at` datetime null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  foreign key (`user_id`) references `user`(`id`) on delete cascade
);
create index `project__device_rdp_launch_log_device_created_idx` on `project__device_rdp_launch_log` (`project_id`, `device_id`, `created`);
{{else if .Postgresql}}
create table `project__device_rdp_launch_log` (
  `id` serial primary key,
  `project_id` int not null,
  `device_id` int not null,
  `user_id` int not null,
  `username` varchar(255) not null default '',
  `phase` varchar(32) not null default 'requested',
  `host` varchar(255) not null default '',
  `rdp_port` int not null default 3389,
  `rdp_user` varchar(255) not null default '',
  `client_ip` varchar(64) not null default '',
  `created` timestamp not null,
  `helper_fetched_at` timestamp null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  foreign key (`user_id`) references `user`(`id`) on delete cascade
);
create index `project__device_rdp_launch_log_device_created_idx` on `project__device_rdp_launch_log` (`project_id`, `device_id`, `created`);
{{else}}
create table `project__device_rdp_launch_log` (
  `id` int auto_increment primary key,
  `project_id` int not null,
  `device_id` int not null,
  `user_id` int not null,
  `username` varchar(255) not null default '',
  `phase` varchar(32) not null default 'requested',
  `host` varchar(255) not null default '',
  `rdp_port` int not null default 3389,
  `rdp_user` varchar(255) not null default '',
  `client_ip` varchar(64) not null default '',
  `created` datetime not null,
  `helper_fetched_at` datetime null,

  key `project__device_rdp_launch_log_device_created_idx` (`project_id`, `device_id`, `created`),
  constraint `project__device_rdp_launch_log_project_fk` foreign key (`project_id`) references `project`(`id`) on delete cascade,
  constraint `project__device_rdp_launch_log_device_fk` foreign key (`device_id`) references `project__device`(`id`) on delete cascade,
  constraint `project__device_rdp_launch_log_user_fk` foreign key (`user_id`) references `user`(`id`) on delete cascade
);
{{end}}
