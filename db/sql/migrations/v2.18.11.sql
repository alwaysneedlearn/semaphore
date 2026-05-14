alter table `project__device` add column `api_port` int not null default 9002;
alter table `project__device` add column `api_status` varchar(20) not null default 'unknown';
