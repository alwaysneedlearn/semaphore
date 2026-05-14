{{if .Sqlite}}
alter table `project__device` add column `rdp_port` integer not null default 3389;
{{else if .Postgresql}}
alter table `project__device` add column `rdp_port` integer not null default 3389;
{{else}}
alter table `project__device` add column `rdp_port` int not null default 3389;
{{end}}
