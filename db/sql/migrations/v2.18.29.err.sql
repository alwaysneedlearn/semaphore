{{if .Sqlite}}
drop table if exists `project__device_rdp_launch_log`;
{{else if .Postgresql}}
drop table if exists `project__device_rdp_launch_log`;
{{else}}
drop table if exists `project__device_rdp_launch_log`;
{{end}}
