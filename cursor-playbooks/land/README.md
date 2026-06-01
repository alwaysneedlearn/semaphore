# LAND playbooks (`cursor-playbooks/land/`)

LAND device type uses API-first status/config with optional Windows process control.

## Playbooks

- `device_status.yml` — check exe existence + process running + API status, then bulk callback
- `device_start.yml` — start process if not running, optional API check, then bulk callback
- `device_stop.yml` — force stop process, optional API check, then bulk callback
- `device_restart.yml` — force restart process, optional API check, then bulk callback
- `device_check_restart_redeploy.yml` — check process/API health, then restart/redeploy only when unhealthy, then bulk callback

Start/restart flow now follows the NEWARE-style skeleton: pre-check health first, run reconfigure only when needed, then start+verify. LAND substitutes API-based steps for NEWARE file/log steps (health via `QueryStatus`, config via `ModifyConfig`).

All playbooks reuse shared WinRM/callback tasks from `../shared/tasks/` and helper scripts from `../shared/files/`. Each LAND play sets:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
sem_files_dir: "{{ playbook_dir }}/../shared/files"
```

LAND-specific logic (SyncLims API, registry, zip) stays in this directory only.

**Install layout:** playbooks expect `{{ EXE_DIR }}\{{ ZIP_NAME }}\{{ EXE_NAME }}` (defaults `C:\Program Files\LAND\land\LHBTS.exe`). If `LHBTS.exe` sits directly under `EXE_DIR`, set `ZIP_NAME=.` or adjust `EXE_DIR` in the Variable Group. Older docs used `land_agent.exe`; typical sites only ship **`LHBTS.exe`** — do not configure a separate `land_agent` process.

## Runner 部署（必做）

日志若出现 `included: .../neware/tasks/winrm_ensure_reachable.yml` 或 `land/tasks/winrm_connect_one_attempt.yml`，说明 **runner 上的文件不是当前 develop**，与仓库代码无关。

在 runner 上执行：

```bash
cd /root/playbook   # 或你的 cursor-playbooks 根目录
git pull

# 必须存在：
test -f shared/tasks/winrm_connect_one_attempt.yml && echo OK shared
test -f land/device_status.yml && head -28 land/device_status.yml | tail -8

# 应看到：include_tasks: "{{ sem_tasks_dir }}/winrm_ensure_reachable.yml"
# 不应出现 include ../neware/tasks/ 或 tasks/playbook_path_assert
# LAND 不需要 land/tasks/ 目录（业务在 device_*.yml 内）

# 删除 runner 上残留的旧文件（若存在）：
rm -f neware/tasks/winrm_ensure_reachable.yml neware/tasks/winrm_gate_play_tasks.yml
```

**Semaphore 模板**：LAND 类型的 Status/Start/… 模板 Playbook 路径填  
`cursor-playbooks/land/device_status.yml`（等），**不要**填 `neware/device_status.yml`。

## Variable Group ENV

| Variable | Default | Description |
|---|---|---|
| `EXE_DIR` | `C:\Program Files\LAND` | Preferred executable directory; drive falls back `preferred -> E: -> C:` |
| `EXE_DIR_FALLBACK_DRIVES` | empty (`preferred,E,C`) | Optional drive order override (comma/space separated), e.g. `D,E,C` |
| `EXE_NAME` | `LHBTS.exe` | LAND executable file name (GUI + API on typical installs) |
| `ZIP_NAME` | `land` | Package folder/zip base name |
| `ZIP_PATH` | `/root/neware/dbwb` | Controller-side zip source directory |
| `EXE_ARGS` | empty | Optional executable arguments |
| `PROCESS_NAME` | (`EXE_NAME` without `.exe`, default `LHBTS`) | Process name for `Get-Process` |
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
| `STOP_GRACEFUL_PROCESS_NAME` | `LHBTS` | Process for `CloseMainWindow` (graceful stop) |
| `STOP_VERIFY_PROCESS_NAME` | (`STOP_GRACEFUL_PROCESS_NAME`) | Process name to verify stopped (default `LHBTS`, same as graceful stop) |
| `STOP_POPUP_WAIT_SECONDS` | `2` | Seconds to wait for confirmation dialog |
| `STOP_POPUP_KEYWORD` | `警告` | Substring match on visible window title; Enter is sent |
| `STOP_FORCE_AFTER_GRACEFUL` | `true` | `Stop-Process -Force` if still running after popup step |
| `RESTART_DELAY` | `30` | Seconds to poll for process after interactive task launch |
| `PROCESS_VERIFY_POLL_SECONDS` | `5` | Poll interval seconds during process verification |

## Extra-vars expected

- Bulk: `devices`, optional `configs_by_host`, `configs_by_hostname`
- Single: `device`, optional `config`

Each host writes `semaphore_callback_row`; localhost bulk phase sends `/devices/status/bulk` and TDengine row publish if configured.

## Task log tags (`[DEBUG-LAND]`)

Implemented via `../shared/tasks/debug_sync_api_*.yml` with play var `sem_debug_tag: LAND`. Search task output for `[DEBUG-LAND]` (exe path, process, SyncLims API, callback row; stop also prints graceful-stop script lines). Bulk PUT uses `[DEBUG-API]` in `semaphore_bulk_put_from_hostvars.yml`.

SINEXCEL and NBT use the same shared tasks with `sem_debug_tag: SINEXCEL` / `NBT`. NEWARE uses `[DEBUG-NEWARE]` in `neware/tasks/debug_*.yml`.

`device_start.yml` / `device_restart.yml` / `device_check_restart_redeploy.yml` now use the same startup strategy as NEWARE:
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
