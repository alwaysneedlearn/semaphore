drop index `project__device_status_callback_device_created_idx`;
drop index `project__device_status_callback_project_created_idx`;
drop table `project__device_status_callback`;
alter table `project__device` drop column `abnormal_reason`;
