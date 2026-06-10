{{if .Sqlite}}
drop table if exists `project__device_winrm_exec_log`;
{{else if .Postgresql}}
drop table if exists `project__device_winrm_exec_log`;
{{else}}
drop table if exists `project__device_winrm_exec_log`;
{{end}}
