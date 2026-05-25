{{if .Sqlite}}
drop table if exists `project__device_discovery_run`;
{{else if .Postgresql}}
drop table if exists `project__device_discovery_run`;
{{else}}
drop table if exists `project__device_discovery_run`;
{{end}}
