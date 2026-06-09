# NBT playbooks (`cursor-playbooks/nbt/`)

Device type **NBT**: API-first health via **heartbeat** (`GET /SendStatus`) and optional **data reset** (`GET /ResetData`) after restart — same layout pattern as LAND/SINEXCEL with NBT-specific HTTP APIs.

Task logs: search **`[DEBUG-NBT]`** (`sem_debug_tag: NBT` in each play).

## Semaphore templates

- `cursor-playbooks/nbt/device_status.yml` — patrol: exe + process + heartbeat staleness
- `cursor-playbooks/nbt/device_start.yml` — start + optional heartbeat verify
- `cursor-playbooks/nbt/device_stop.yml` — force stop; `api_status` forced offline
- `cursor-playbooks/nbt/device_restart.yml` — stop + start + heartbeat + **ResetData** (when configured)
- `cursor-playbooks/nbt/device_check_restart_redeploy.yml` — pre-gate heartbeat; redeploy only when unhealthy

## NBT HTTP API (from agent on device)

### 1) Heartbeat — last data send time

- `GET http://<ip>:<port>/SendStatus`
- **Response** (plain text): `2026-06-09 13:35:19`
- **Healthy** when HTTP **200** and `(now - last_send) <= NBT_HEARTBEAT_MAX_AGE_MINUTES` (default **90** minutes)
- Used for patrol, start verify, restart verify, redeploy pre-gate

### 2) Reset data (redeliver)

- `GET http://<ip>:<port>/ResetData?StartDate=2026-6-1&EndDate=2026-7-1`
- **Response** (JSON): `{"IsSuccess": true, "Message": "", "Data": null}`
- Called on **`device_restart.yml`** / **`device_check_restart_redeploy.yml`** after process **RUNNING** + heartbeat OK
- Dates from Semaphore config category **`ResetData`**: keys **`StartDate`**, **`EndDate`** (format `yyyy-M-d`)

Shared tasks: `../shared/tasks/nbt_api_heartbeat_check.yml`, `nbt_api_reset_data.yml`, `nbt_prepare_merged_cfg.yml`.

## Variable Group ENV

| Variable | Default | Description |
|----------|---------|-------------|
| `EXE_DIR` | `C:\Program Files\NBT` | Install root (drive fallback via shared `resolve_exe_dir_windows`) |
| `EXE_NAME` | `nbt_agent.exe` | Executable |
| `ZIP_NAME` / `ZIP_PATH` | `nbt` / `/root/nbt/pkg` | Install zip |
| `CONFIG_FILE_NAME` | `nbt.iconf` | NEWARE-style INI config |
| `NBT_API_PORT` | **8885** | Agent API port (device `api_port` overrides) |
| `NBT_API_HEARTBEAT_PATH` | `/SendStatus` | Heartbeat endpoint |
| `NBT_API_RESET_DATA_PATH` | `/ResetData` | Reset data endpoint |
| `NBT_HEARTBEAT_MAX_AGE_MINUTES` | **90** | Max minutes since last send |
| `NBT_API_TIMEOUT` | `8` | HTTP timeout (seconds) |
| `NBT_START_CHECK_API` | `true` | After start, verify heartbeat |
| `NBT_STOP_CHECK_API` | `false` | After stop, GET heartbeat for DEBUG only |
| `RESTART_DELAY` / `PROCESS_VERIFY_POLL_SECONDS` | `30` / `5` | Interactive task start poll |

## Config categories (Semaphore UI)

| Category | Used by |
|----------|---------|
| `SystemConfig` | INI merge via `apply_neware_style_device_config_files.yml` |
| `ResetData` | `ResetData` API on restart / redeploy (`StartDate`, `EndDate`) |
