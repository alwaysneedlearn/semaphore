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

## Variable Group (examples)

| Variable | Default | Notes |
|----------|---------|--------|
| `API_PORT` / device `api_port` | `9002` | Kafka API port |
| `SINEXCEL_KAFKA_API_PORT` | — | Override env port |
| `SINEXCEL_START_CHECK_API` | `true` | Require Kafka enabled after start |
| `SINEXCEL_KAFKA_*_PATH` | `/kafka/...` | See `group_vars/windows_hosts.yml` |
| `EXE_DIR`, `ZIP_NAME`, … | See playbooks | Same as before |
| `START_POPUP_KEYWORD` | `提示` | After interactive start, auto-confirm desktop popup (title contains keyword; Enter). Requires RDP/desktop session for profile user. |
| `START_POPUP_WAIT_SECONDS` | `3` | Seconds to wait before scanning for start popup |
| `STOP_POPUP_KEYWORD` | `警告` | Graceful stop confirmation popup (see shared README) |

## Extra-vars

Bulk: `devices`, `configs_by_host`, `default_config`  
Single: `device`, `config`
