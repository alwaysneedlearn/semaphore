# LAND playbooks (`cursor-playbooks/land/`)

LAND device type uses API-first status/config with optional Windows process control.

## Playbooks

- `device_status.yml` — check exe existence + process running + API status, then bulk callback
- `device_start.yml` — start process if not running, optional API check, then bulk callback
- `device_stop.yml` — force stop process, optional API check, then bulk callback
- `device_restart.yml` — force restart process, optional API check, then bulk callback
- `device_restart_redeploy.yml` — check process/API health, then restart/redeploy only when unhealthy, then bulk callback

All playbooks reuse shared callback/TDengine tasks from `../neware/tasks/semaphore_bulk_put_from_hostvars.yml`.

## Variable Group ENV

| Variable | Default | Description |
|---|---|---|
| `EXE_DIR` | `C:\Program Files\LAND` | LAND executable directory |
| `EXE_NAME` | `land_agent.exe` | LAND executable file name |
| `PROCESS_NAME` | (`EXE_NAME` without `.exe`) | Process name for `Get-Process` |
| `LAND_API_SCHEME` | `http` | API scheme (`http`/`https`) |
| `LAND_API_PORT` | `9002` | API port |
| `LAND_API_STATUS_PATH` | `/api/status` | Status endpoint |
| `LAND_API_CONFIG_PATH` | `/api/config` | Config endpoint |
| `LAND_API_TIMEOUT` | `8` | API timeout (seconds) |
| `LAND_API_TOKEN` | empty | Optional bearer token |
| `LAND_API_VERIFY_TLS` | `true` | Whether to verify TLS cert |
| `LAND_START_CHECK_API` | `true` | After start/restart, verify API |
| `LAND_STOP_CHECK_API` | `false` | After stop, verify API |

## Extra-vars expected

- Bulk: `devices`, optional `configs_by_host`, `configs_by_hostname`
- Single: `device`, optional `config`

Each host writes `semaphore_callback_row`; localhost bulk phase sends `/devices/status/bulk` and TDengine row publish if configured.
