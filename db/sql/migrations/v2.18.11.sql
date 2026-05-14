{{if .Sqlite}}
alter table `project__device` add column `rdp_user` varchar(255) not null default '';
alter table `project__device` add column `rdp_password` text not null default '';
{{else if .Postgresql}}
alter table `project__device` add column `rdp_user` varchar(255) not null default '';
alter table `project__device` add column `rdp_password` text not null default '';
{{else}}
alter table `project__device` add column `rdp_user` varchar(255) not null default '';
alter table `project__device` add column `rdp_password` longtext not null;
{{end}}
