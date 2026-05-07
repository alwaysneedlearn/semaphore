alter table `project__device` add column `device_status` varchar(20) not null default 'unknown';

update `project__device`
set `device_status` = case
  when `rdp_status` = 'online' and `winrm_status` = 'online' then 'healthy'
  when `rdp_status` = 'offline' and `winrm_status` = 'offline' then 'unhealthy'
  else 'unknown'
end;

update `project__device`
set `name` = `hostname`
where (`name` = '' or `name` is null) and `hostname` <> '';

create unique index `project__device_project_hostname_idx` on `project__device` (`project_id`, `hostname`);
