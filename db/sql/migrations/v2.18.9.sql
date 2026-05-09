alter table `project__device` add column `ansible_user` varchar(255) not null default '';
alter table `project__device` add column `ansible_password` longtext not null;
alter table `project__device` add column `ansible_connection` varchar(64) not null default 'winrm';
alter table `project__device` add column `ansible_winrm_transport` varchar(64) not null default 'basic';
alter table `project__device` add column `ansible_winrm_scheme` varchar(64) not null default 'http';
alter table `project__device` add column `ansible_port` int not null default 5985;
alter table `project__device` add column `ansible_winrm_server_cert_validation` varchar(64) not null default 'ignore';

alter table `project__device_settings` add column `default_inventory_id` int null;
alter table `project__device_settings` add column `default_ansible_user` varchar(255) not null default '';
alter table `project__device_settings` add column `default_ansible_password` longtext not null;
alter table `project__device_settings` add column `default_ansible_connection` varchar(64) not null default 'winrm';
alter table `project__device_settings` add column `default_ansible_winrm_transport` varchar(64) not null default 'basic';
alter table `project__device_settings` add column `default_ansible_winrm_scheme` varchar(64) not null default 'http';
alter table `project__device_settings` add column `default_ansible_port` int not null default 5985;
alter table `project__device_settings` add column `default_ansible_winrm_server_cert_validation` varchar(64) not null default 'ignore';

alter table `project__inventory`
  add column `is_device_default_auto` tinyint(1) not null default 0;
