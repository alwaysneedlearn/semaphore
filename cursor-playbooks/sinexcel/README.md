# SINEXCEL playbooks (`cursor-playbooks/sinexcel/`)

Device type **SINEXCEL**: LAND-style lifecycle (Kafka HTTP API + optional retransmit) with **NEWARE-style INI config** on start/restart.

## Runner 同步（报错 `_app_from_cfg is undefined` 时必做）

日志里若仍是 **`_app_from_cfg`** 或 **`prepare_device_exe_paths.yml` 第 16 行** 为「应用每设备 EXE 目录」，说明 **Ansible 控制器上的文件不是当前 `develop`**（与 Semaphore UI 无关）。

在 **跑任务的 runner** 上：

```bash
cd /root/playbook   # 或你的 cursor-playbooks 根目录
git fetch origin develop
git checkout develop
git pull origin develop
grep -n '_app_from_cfg' shared/tasks/prepare_device_exe_paths.yml
# 应无输出；且应有 PREPARE_DEVICE_EXE_PATHS_REV=3
grep PREPARE_DEVICE_EXE_PATHS_REV shared/tasks/prepare_device_exe_paths.yml
```

重跑巡检后，任务日志里应出现 **`[DEBUG-INSTALL] PREPARE_DEVICE_EXE_PATHS_REV=3`**。若看不到，模板 Playbook 路径是否指向该目录下的 `sinexcel/device_status.yml`。

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
| **`Install`** (or **`Paths`**) | **可选**：仅极个别机台扫描仍找不到 exe 时覆盖路径（日常用变量组即可） |

### EXE 路径：变量组为主（推荐）

全项目统一在 **变量组** 配置即可，**不需要**逐台维护 `Install.ExeDir` / `Install.ExePath`：

| Variable Group | 作用 |
|----------------|------|
| `EXE_NAME` | 要查找的 exe 文件名（如 `sinexcel_agent.exe`） |
| `EXE_DIR` | 首选安装根（盘符探测起点） |
| `EXE_DIR_FALLBACK_DRIVES` | 盘符尝试顺序，如 `D,E,C` |
| `EXE_SCAN_LATEST` | `true`（默认）：各盘符下浅层扫描，取 **mtime 最新** 的 `EXE_NAME` |
| `EXE_SCAN_MAX_DEPTH` | 扫描深度，默认 `2` |
| `APP_DIR` | 可选；未开扫描时用于 `AppDir\ExeName` 固定路径探测 |

解析流程（`resolve_exe_dir_windows` + `sem_resolve_exe_dir_windows.ps1`）：

1. **`EXE_SCAN_LATEST=true`**：在每个候选盘符下最多 **N 层**目录找 `EXE_NAME`，取最新文件 → `EXE_PATH_RESOLVED`  
2. 否则按 `AppDir\ExeName` 在各盘符探测（LAND 方式）  
3. 盘符回退；仍无则巡检报 `Executable not found`

### `Install` / `Paths`（可选覆盖）

仅当变量组 + 扫描仍无法定位某台机时使用。配置入口：**设备 → 配置** 或 **设备类型 → 默认配置**。

| Key | Example | 说明 |
|-----|---------|------|
| `ExePath` | `F:\Apps\sinexcel\sinexcel_agent.exe` | 完整路径，跳过扫描 |
| `ExeDir` / `AppDir` / `ExeDirFallbackDrives` / `ExeScanLatest` / `ExeScanMaxDepth` | 同变量组语义 | 覆盖该主机或类型的 VG 默认 |

## Variable Group (examples)

完整说明见 [`../VARIABLE_GROUPS.md`](../VARIABLE_GROUPS.md)。**`EXE_NAME` 推荐带 `.exe`**（如 `sinexcel_agent.exe`）；写 `sinexcel_agent` 也会自动规范化。

| Variable | Default | Notes |
|----------|---------|--------|
| `SINEXCEL_KAFKA_API_PORT` | — | **Kafka** API 端口（优先于 `API_PORT`） |
| `API_PORT` / device `api_port` | `9002` | Kafka 端口回退（设备 `api_port` 最高） |
| `SINEXCEL_API_PORT` | `8080` | **仅 Stop 模板**可选 SyncLims `QueryStatus` 探测，**不是** Kafka 巡检端口 |
| `SINEXCEL_START_CHECK_API` | `true` | Require Kafka enabled after start |
| `SINEXCEL_KAFKA_*_PATH` | `/kafka/...` | See `group_vars/windows_hosts.yml` |
| `EXE_DIR` | `C:\Program Files\SINEXCEL` | 首选根目录（配合盘符回退 + 扫描） |
| `EXE_DIR_FALLBACK_DRIVES` | `D,E,C` | 盘符尝试顺序 |
| `EXE_SCAN_LATEST` | `true` | 按盘符浅层扫描最新 exe（见上） |
| `EXE_SCAN_MAX_DEPTH` | `2` | 扫描最大目录深度 |
| `APP_DIR` | — | 子目录，默认 `sinexcel`（`ZIP_NAME`） |
| `EXE_NAME` | `sinexcel_agent.exe` | 磁盘文件名（带 `.exe`）；`PROCESS_NAME` 可选覆盖进程名（无后缀） |
| `ZIP_NAME`, … | See playbooks | Same as before |
| `START_POPUP_KEYWORD` | `提示` | After interactive start, auto-confirm desktop popup (title contains keyword; Enter). Requires RDP/desktop session for profile user. |
| `START_POPUP_WAIT_SECONDS` | `3` | Seconds to wait before scanning for start popup |
| `STOP_POPUP_KEYWORD` | `警告` | Stop confirm: match dialog **content** when title is empty (`title_or_content` default) |
| `STOP_POPUP_MATCH_MODE` | `title_or_content` | Set `content` if only body text contains the keyword |
| `STOP_GRACEFUL_PROCESS_NAME` | 从 `EXE_NAME` 推导 | 实际 agent 进程名（如 `sinexcel_agent`），不要填 `LHBTS` |

## Extra-vars

Bulk: `devices`, `configs_by_host`, `default_config`  
Single: `device`, `config`
