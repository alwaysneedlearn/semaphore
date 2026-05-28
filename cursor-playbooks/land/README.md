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
| `ZIP_NAME` | `land` | Package folder/zip base name |
| `ZIP_PATH` | `/root/neware/dbwb` | Controller-side zip source directory |
| `EXE_ARGS` | empty | Optional executable arguments |
| `PROCESS_NAME` | (`EXE_NAME` without `.exe`) | Process name for `Get-Process` |
| `LAND_API_SCHEME` | `http` | API scheme (`http`/`https`) |
| `LAND_API_PORT` | `8080` | API port |
| `LAND_API_STATUS_PATH` | `/SyncLims/QueryStatus` | Query status endpoint |
| `LAND_API_REDELIVER_PATH` | `/SyncLims/Redeliver` | Redeliver endpoint (example) |
| `LAND_API_MODIFY_CONFIG_PATH` | `/SyncLims/ModifyConfig` | Modify config endpoint (example) |
| `LAND_API_TIMEOUT` | `8` | API timeout (seconds) |
| `LAND_API_TOKEN` | `landapi` | Token in request JSON body |
| `LAND_API_VERIFY_TLS` | `true` | Whether to verify TLS cert |
| `LAND_START_CHECK_API` | `true` | After start/restart, verify API |
| `LAND_STOP_CHECK_API` | `false` | After stop, verify API |
| `RESTART_DELAY` | `30` | Seconds to poll for process after interactive task launch |
| `PROCESS_VERIFY_POLL_SECONDS` | `5` | Poll interval seconds during process verification |

## Extra-vars expected

- Bulk: `devices`, optional `configs_by_host`, `configs_by_hostname`
- Single: `device`, optional `config`

Each host writes `semaphore_callback_row`; localhost bulk phase sends `/devices/status/bulk` and TDengine row publish if configured.

`device_start.yml` / `device_restart.yml` / `device_restart_redeploy.yml` now use the same startup strategy as NEWARE:
- check exe first (`{{ exe_path }}`); if missing, check/copy `{{ exe_dir }}\{{ zip_name }}.zip` then `Expand-Archive`
- launch via **interactive scheduled task** (desktop logged-in user, RunLevel Highest), not plain `Start-Process` under WinRM session

## LAND API examples (from Postman collection)

1) Query status
- `POST http://<ip>:<port>{{ LAND_API_STATUS_PATH }}`
- Body:
  - `{"token":"landapi"}`

2) Redeliver data
- `POST http://<ip>:<port>{{ LAND_API_REDELIVER_PATH }}`
- Body:
  - `{"token":"landapi","startTime":"2026-5-24 10:10:10","endTime":"2026-5-26 10:10:10"}`

3) Modify config
- `POST http://<ip>:<port>{{ LAND_API_MODIFY_CONFIG_PATH }}`
- Body:
  - `{"token":"landapi", "...": "see your full Postman payload"}`
