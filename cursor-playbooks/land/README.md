# LAND playbooks (`cursor-playbooks/land/`)

LAND device type uses API-first status/config with optional Windows process control.

## Playbooks

- `device_status.yml` — check exe existence + process running + API status, then bulk callback
- `device_stop.yml` — graceful stop (CloseMainWindow + popup confirm), optional API check, then bulk callback
- `device_restart.yml` — graceful stop then ModifyConfig + interactive start (same stop path as `device_stop.yml`)
**Redeploy:** ModifyConfig (if running) → **copy installer to Desktop if missing** → graceful stop → GUI wizard → interactive start → API verify.
- `device_check_restart_redeploy.yml` — check process/API health; when unhealthy, graceful stop + ModifyConfig + start, then bulk callback

Start/restart flow now follows the NEWARE-style skeleton: pre-check health first, run reconfigure only when needed, then start+verify. LAND substitutes API-based steps for NEWARE file/log steps (health via `QueryStatus`, config via `ModifyConfig`).

**ModifyConfig timing:** LAND only accepts config changes while the app is running. Playbooks call `POST …/ModifyConfig` **while the process is still running (before graceful stop on restart/redeploy)**, then after start/restart **retry once** if that attempt failed (`http_status` not 2xx, e.g. `-1` when the API was down) **and** process + `QueryStatus` are healthy. They do **not** POST ModifyConfig between stop and start (that always yields connection errors).

All playbooks reuse shared WinRM/callback tasks from `../shared/tasks/` and helper scripts from `../shared/files/`. Each LAND play sets:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
sem_files_dir: "{{ playbook_dir }}/../shared/files"
```

LAND-specific logic (SyncLims API, registry, zip) stays in this directory only.

**Install layout:** `exe_path` = `{{ EXE_DIR }}\{{ APP_DIR }}\{{ EXE_NAME }}` (default **`APP_DIR=LHBTS`**). **`EXE_DIR`** use `D:\` or `F:\` (not `D:` alone — or rely on script mapping `D:` → `D:\`). **`EXE_DIR_FALLBACK_DRIVES=D,F`**: probes `LHBTS\LHBTS.exe` on each drive before picking the first existing disk.

**Redeploy (GUI installer):** copy installer exe from controller to **current interactive user's Desktop** by default (`C:\Users\{user}\Desktop\{installer}.exe`). Override with **`LAND_INSTALLER_DEST`** or set **`LAND_INSTALLER_USE_DESKTOP=false`** to use `{{ EXE_DIR }}\{installer}.exe`. Wizard clicks: `LHBTS 安装` / **升级** → `提示` / **确定** → poll until `LHBTS 安装` / **确定** (no fixed 5s wait; each step uses **`LAND_INSTALL_STEP_TIMEOUT_SECONDS`** poll). No zip extract, no `HKCU\Software\LH` registry.

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

变量组总览见 [`../VARIABLE_GROUPS.md`](../VARIABLE_GROUPS.md)。**`EXE_NAME` 推荐带 `.exe`**。

| Variable | Default | Description |
|---|---|---|
| `EXE_DIR` | `C:\Program Files\LAND` | Preferred root (e.g. `D:\` or `F:\`); drive letter resolved first |
| `EXE_DIR_FALLBACK_DRIVES` | empty (`preferred,E,C`) | Drive try order when preferred disk missing, e.g. **`D,F,E,C`** for D or F installs |
| `APP_DIR` | `LHBTS` | Install folder under `exe_dir` (unzip creates `{{ exe_dir }}\LHBTS\`) |
| `EXE_NAME` | `LHBTS.exe` | LAND executable file name (GUI + API on typical installs) |
| `ZIP_NAME` | `land` | Legacy name; redeploy prefers **`INSTALLER_EXE_NAME`** (installer `.exe` base name) |
| `INSTALLER_EXE_NAME` | (`ZIP_NAME` or `LHBTS.exe`) | Installer exe file name on controller |
| `ZIP_PATH` | `/root/neware/dbwb` | Controller directory **or** full path to installer `.exe` |
| `LAND_INSTALLER_DEST` | (Desktop) | Explicit full path on target; overrides desktop default |
| `LAND_INSTALLER_USE_DESKTOP` | `true` | `false` → copy to `{{ EXE_DIR }}\{installer}.exe` instead |
| `LAND_INSTALL_STEP_TIMEOUT_SECONDS` | `90` | Max wait per dialog (poll until window/button appears) |
| `LAND_INSTALL_CLICK_SETTLE_MS` | `500` | Pause after each successful click |
| `LAND_INSTALL_COORD_MOVE_DELAY_MS` | `400` | Pause after moving mouse before coord click |
| `LAND_INSTALL_POLL_DELAY_MS` | `400` | Poll interval between dialog/button retries |
| `LAND_INSTALL_TASK_TIMEOUT_SECONDS` | `180` | Interactive scheduled-task timeout |
| `LAND_INSTALL_DLG1_TITLE` / `_BUTTON` | `LHBTS 安装` / `升级` | Step 1 — custom UI; coord fallback **32%, 93%** |
| `LAND_INSTALL_DLG2_TITLE` / `_BUTTON` | `提示` / `确定` | Step 2 — popup after 升级; Win32 text match |
| `LAND_INSTALL_DLG3_TITLE` / `_BUTTON` | `LHBTS 安装` / `确定` | Step 3 — single button; coord fallback **83%, 93%** |
| `LAND_INSTALL_DLG1_COORD_X_PCT` / `_Y_PCT` | `32` / `93` | Client-area % for step 1 coord fallback |
| `LAND_INSTALL_DLG2_COORD_X_PCT` / `_Y_PCT` | `0` / `0` | Step 2 coord disabled |
| `LAND_INSTALL_DLG3_COORD_X_PCT` / `_Y_PCT` | `83` / `93` | Client-area % for step 3 coord fallback |
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
| `LAND_QUERY_STATUS_REQUIRE_REAL_STATUS` | `true` | QueryStatus healthy only when JSON `data.real_status` is `true` (HTTP 2xx alone is not enough) |
| `LAND_API_STATUS_START_POLL_RETRIES` | `6` | After start/restart, poll `QueryStatus` until `api_ok` (patrol uses single POST) |
| `LAND_API_STATUS_START_POLL_DELAY` | `10` | Seconds between `QueryStatus` poll attempts |
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

`device_restart.yml` / `device_check_restart_redeploy.yml` use the same startup strategy as NEWARE:
- check exe first (`{{ exe_path }}`); if missing, check/copy `{{ exe_dir }}\{{ zip_filename }}` then `Expand-Archive`
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

3) Modify config (category **SystemConfig** only)
- `POST http://<ip>:<port>{{ LAND_API_MODIFY_CONFIG_PATH }}`
- Body: **flat JSON** at the root — `{"token":"landapi", "FieldA": "...", "FieldB": "..."}` (keys from merged `SystemConfig` category; device overrides type default).

4) Redeliver (category **Redeliver** only; **`device_restart.yml` only**)
- After restart: process `RUNNING`, `QueryStatus` HTTP 2xx, valid `startTime` / `endTime` in category `Redeliver`.
- **Time format** (playbook regex): `yyyy-M-d HH:mm:ss` — year 4 digits; month/day/hour **1–2** digits (no zero-padding required); **minute and second must be two digits** (`10:10:10` OK, `10:1:10` invalid). Examples: `2026-5-24 10:10:10`, `2026-6-4 10:10:10`. Keys in config: `startTime`, `endTime` under category `Redeliver`.
- `POST http://<ip>:<port>{{ LAND_API_REDELIVER_PATH }}` with `token` from `LAND_API_TOKEN`, times from config.
- Success response example: `{"code":200,"message":"success","seqNo":8,"data":true}`.

**Config categories in Semaphore UI**

| Category | Used by |
|----------|---------|
| `SystemConfig` | ModifyConfig (start / restart / redeploy while app running) |
| `Redeliver` | Redeliver API (`device_restart.yml` after healthy restart only) |

Each row supports an optional **remark** (description) in device list and device type default config.
