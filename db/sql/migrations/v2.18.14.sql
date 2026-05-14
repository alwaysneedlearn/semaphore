{{if .Sqlite}}
alter table `project__device` add column `api_port` integer not null default 9002;
alter table `project__device` add column `api_status` varchar(20) not null default 'offline';
alter table `project__device_status_callback` add column `api_status` varchar(20) not null default 'offline';
{{else if .Postgresql}}
alter table `project__device` add column `api_port` integer not null default 9002;
alter table `project__device` add column `api_status` varchar(20) not null default 'offline';
alter table `project__device_status_callback` add column `api_status` varchar(20) not null default 'offline';
{{else}}
alter table `project__device` add column `api_port` int not null default 9002;
alter table `project__device` add column `api_status` varchar(20) not null default 'offline';
alter table `project__device_status_callback` add column `api_status` varchar(20) not null default 'offline';
{{end}}
