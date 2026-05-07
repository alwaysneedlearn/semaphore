create table `project__device` (
  `id` integer primary key autoincrement,

  `project_id`     int          not null,
  `name`           varchar(150) not null,
  `ip_address`     varchar(64)  not null default '',
  `hostname`       varchar(255) not null default '',
  `rdp_status`     varchar(20)  not null default 'unknown',
  `winrm_status`   varchar(20)  not null default 'unknown',
  `last_updated`   datetime null,
  `created`        datetime not null,

  foreign key (`project_id`) references `project`(`id`) on delete cascade
);

create index `project__device_project_id_idx` on `project__device` (`project_id`);

create table `project__device_config_item` (
  `id` integer primary key autoincrement,

  `device_id`  int          not null,
  `category`   varchar(150) not null default '',
  `key`        varchar(150) not null,
  `value`      longtext     not null default '',

  foreign key (`device_id`) references `project__device`(`id`) on delete cascade
);

create unique index `project__device_config_item_idx` on `project__device_config_item` (`device_id`, `category`, `key`);

create table `project__device_settings` (
  `project_id`                   int primary key,
  `discover_template_id`         int null,
  `start_template_id`            int null,
  `stop_template_id`             int null,
  `restart_template_id`          int null,
  `status_template_id`           int null,
  `config_template_id`           int null,
  `status_refresh_interval_min`  int not null default 0,
  `last_status_refresh_at`       datetime null,

  foreign key (`project_id`)            references `project`(`id`)          on delete cascade,
  foreign key (`discover_template_id`)  references `project__template`(`id`) on delete set null,
  foreign key (`start_template_id`)     references `project__template`(`id`) on delete set null,
  foreign key (`stop_template_id`)      references `project__template`(`id`) on delete set null,
  foreign key (`restart_template_id`)   references `project__template`(`id`) on delete set null,
  foreign key (`status_template_id`)    references `project__template`(`id`) on delete set null,
  foreign key (`config_template_id`)    references `project__template`(`id`) on delete set null
);
