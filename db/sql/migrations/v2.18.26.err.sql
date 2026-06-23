{{if .Sqlite}}
drop table if exists `project__device_operation_log`;
{{else if .Postgresql}}
drop table if exists `project__device_operation_log`;
{{else}}
drop table if exists `project__device_operation_log`;
{{end}}
