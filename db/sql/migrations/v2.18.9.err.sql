alter table `project__inventory` drop column `is_device_default_auto`;

alter table `project__device_settings` drop column `default_inventory_id`;
alter table `project__device_settings` drop column `default_ansible_user`;
alter table `project__device_settings` drop column `default_ansible_password`;
alter table `project__device_settings` drop column `default_ansible_connection`;
alter table `project__device_settings` drop column `default_ansible_winrm_transport`;
alter table `project__device_settings` drop column `default_ansible_winrm_scheme`;
alter table `project__device_settings` drop column `default_ansible_port`;
alter table `project__device_settings` drop column `default_ansible_winrm_server_cert_validation`;

alter table `project__device` drop column `ansible_user`;
alter table `project__device` drop column `ansible_password`;
alter table `project__device` drop column `ansible_connection`;
alter table `project__device` drop column `ansible_winrm_transport`;
alter table `project__device` drop column `ansible_winrm_scheme`;
alter table `project__device` drop column `ansible_port`;
alter table `project__device` drop column `ansible_winrm_server_cert_validation`;
