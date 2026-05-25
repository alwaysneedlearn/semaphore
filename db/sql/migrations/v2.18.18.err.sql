{{if .Sqlite}}
drop table if exists `project__device_discovery_host`;
{{else if .Postgresql}}
drop table if exists `project__device_discovery_host`;
{{else}}
drop table if exists `project__device_discovery_host`;
{{end}}
