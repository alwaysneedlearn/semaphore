-- Disable built-in DeviceStatusScheduler intervals (check_restart via Semaphore Schedules).
UPDATE `project__device_profile_settings` SET `status_refresh_interval_min` = 0 WHERE `status_refresh_interval_min` <> 0;
UPDATE `project__device_settings` SET `status_refresh_interval_min` = 0 WHERE `status_refresh_interval_min` <> 0;
