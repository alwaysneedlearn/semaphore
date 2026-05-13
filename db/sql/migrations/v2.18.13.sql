update project__device set device_status='unhealthy' where lower(coalesce(device_status,''))='unknown';
update project__device set rdp_status='offline' where lower(coalesce(rdp_status,''))='unknown';
update project__device set winrm_status='offline' where lower(coalesce(winrm_status,''))='unknown';
