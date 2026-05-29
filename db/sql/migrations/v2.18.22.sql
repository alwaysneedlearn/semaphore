{{if .Sqlite}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'SINEXCEL', 'SINEXCEL', 1, datetime('now')
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'SINEXCEL'
);

insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NBT', 'NBT', 1, datetime('now')
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NBT'
);

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_sinexcel'
from `project__device_profile` dp
where dp.`profile_key` = 'SINEXCEL'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_nbt'
from `project__device_profile` dp
where dp.`profile_key` = 'NBT'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );
{{else if .Postgresql}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'SINEXCEL', 'SINEXCEL', true, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'SINEXCEL'
);

insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NBT', 'NBT', true, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NBT'
);

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_sinexcel'
from `project__device_profile` dp
where dp.`profile_key` = 'SINEXCEL'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_nbt'
from `project__device_profile` dp
where dp.`profile_key` = 'NBT'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );
{{else}}
insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'SINEXCEL', 'SINEXCEL', 1, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'SINEXCEL'
);

insert into `project__device_profile` (`project_id`, `profile_key`, `name`, `enabled`, `created`)
select p.`id`, 'NBT', 'NBT', 1, now()
from `project` p
where not exists (
  select 1 from `project__device_profile` dp
  where dp.`project_id` = p.`id` and dp.`profile_key` = 'NBT'
);

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_sinexcel'
from `project__device_profile` dp
where dp.`profile_key` = 'SINEXCEL'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );

insert into `project__device_profile_settings` (`project_id`, `profile_id`, `tdengine_status_table`)
select dp.`project_id`, dp.`id`, 'status_nbt'
from `project__device_profile` dp
where dp.`profile_key` = 'NBT'
  and not exists (
    select 1 from `project__device_profile_settings` ps
    where ps.`project_id` = dp.`project_id` and ps.`profile_id` = dp.`id`
  );
{{end}}
