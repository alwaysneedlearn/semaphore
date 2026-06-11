# SINEXCEL playbooks (`cursor-playbooks/sinexcel/`)

Device type **SINEXCEL**: LAND-style lifecycle (Kafka HTTP API + optional retransmit) with **NEWARE-style INI config** on start/restart.

## Playbooks

| File | Purpose |
|------|---------|
| `device_status.yml` | Exe + process + **Kafka QueryConfig** (`EnableFlowInfoExtendedSqlite`), bulk callback |
| `device_start.yml` | INI + **SetConfig / IsEnable** + start (no retransmit) |
| `device_restart.yml` | Same + **QueryHistory + batch Retransmit** when `Retransmit` category set |
| `device_redeploy.yml` | Zip deploy + config/start (no retransmit) |
| `device_check_restart.yml` | Unhealthy gate via Kafka QueryConfig; restart only when needed |
| `device_stop.yml` | Force stop process |

Shared core: `../shared/tasks/sinexcel_config_stop_start.yml`

## Kafka API (agent HTTP, default port **9002**)

| Endpoint | Purpose |
|----------|---------|
| `POST /kafka/QueryConfig` | Patrol/health: `Msg.EnableFlowInfoExtendedSqlite == true` |
| `POST /kafka/SetConfig` | Apply **KafkaConfig** category (addrs, topics) |
| `POST /kafka/IsEnable` | `{"EnableFlowInfoExtendedSqlite": true}` |
| `POST /kafka/QueryHistory` | `StartTime` / `EndTime` → history rows with `FlowId` |
| `POST /kafka/Retransmit` | Batch `[{"FlowId":"..."}, ...]` (restart only) |

## Semaphore config categories (like LAND)

| Category | Used by |
|----------|---------|
| `KafkaConfig` | `SetConfig` body (`KafkaAddrs`, `KafkaTopic*`, …). Falls back to `SystemConfig` if empty. |
| `Retransmit` | `StartTime`, `EndTime` — format `yyyy/MM/dd HH:mm:ss.SSS` (e.g. `2026/06/08 00:00:00.000`). Restart queries history then retransmits **UploadExist=false** rows by default. |
| **`Install`** (or **`Paths`**) | **Per-device EXE 目录**（变量组无法统一时用这个） |

### `Install` / `Paths` keys (per device or type default)

| Key | Example | Priority |
|-----|---------|----------|
| `ExePath` | `F:\Apps\sinexcel\sinexcel_agent.exe` | 最高：完整路径，跳过拼接 |
| `ExeDir` | `D:\Program Files\SINEXCEL` 或 `D:` | 安装根目录（配合盘符探测） |
| `AppDir` | `sinexcel` | 子目录（默认 = `ZIP_NAME`） |
| `ExeDirFallbackDrives` | `D,F,C` | 仅该设备盘符探测顺序 |

配置入口：**设备 → 配置**，或 **设备类型 → 默认配置**（`default_config` / `configs_by_host`）。

解析顺序：`设备 Install` → `类型默认 Install` → 变量组 `EXE_DIR` → playbook 默认 → **`resolve_exe_dir_windows`** 按盘符探测已有 `exe` 目录。

## Variable Group (examples)

| Variable | Default | Notes |
|----------|---------|--------|
| `API_PORT` / device `api_port` | `9002` | Kafka API port |
| `SINEXCEL_KAFKA_API_PORT` | — | Override env port |
| `SINEXCEL_START_CHECK_API` | `true` | Require Kafka enabled after start |
| `SINEXCEL_KAFKA_*_PATH` | `/kafka/...` | See `group_vars/windows_hosts.yml` |
| `EXE_DIR` | `C:\Program Files\SINEXCEL` | 全项目默认；**每台机不同请用 Install.ExeDir** |
| `EXE_DIR_FALLBACK_DRIVES` | `D,E,C` | 盘符回退顺序（可被 Install.ExeDirFallbackDrives 覆盖） |
| `APP_DIR` | — | 子目录，默认 `sinexcel`（`ZIP_NAME`） |
| `ZIP_NAME`, `EXE_NAME`, … | See playbooks | Same as before |
| `START_POPUP_KEYWORD` | `提示` | After interactive start, auto-confirm desktop popup (title contains keyword; Enter). Requires RDP/desktop session for profile user. |
| `START_POPUP_WAIT_SECONDS` | `3` | Seconds to wait before scanning for start popup |
| `STOP_POPUP_KEYWORD` | `警告` | Stop confirm: match dialog **content** when title is empty (`title_or_content` default) |
| `STOP_POPUP_MATCH_MODE` | `title_or_content` | Set `content` if only body text contains the keyword |
| `STOP_GRACEFUL_PROCESS_NAME` | same as `PROCESS_NAME` | Must be actual agent process (e.g. `sinexcel_agent`), not `LHBTS` |

## Extra-vars

Bulk: `devices`, `configs_by_host`, `default_config`  
Single: `device`, `config`
