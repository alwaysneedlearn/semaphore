alter table `project__device` add column `abnormal_reason` longtext null;

create table `project__device_status_callback` (
  `id` integer primary key autoincrement,
  `project_id` int not null,
  `device_id` int null,
  `hostname` varchar(255) not null default '',
  `status` varchar(20) not null default 'unknown',
  `rdp_status` varchar(20) not null default 'unknown',
  `winrm_status` varchar(20) not null default 'unknown',
  `abnormal_reason` longtext null,
  `payload` longtext not null,
  `created` datetime not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade,
  foreign key (`device_id`) references `project__device`(`id`) on delete set null
);

create index `project__device_status_callback_project_created_idx` on `project__device_status_callback` (`project_id`, `created`);
create index `project__device_status_callback_device_created_idx` on `project__device_status_callback` (`device_id`, `created`);
