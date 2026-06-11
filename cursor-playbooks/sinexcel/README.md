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

## Agent HTTP API（默认端口 **9002**，变量组 `SINEXCEL_API_PORT`）

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

## Variable Group（环境变量一览）

完整示例见 [`../VARIABLE_GROUPS.md`](../VARIABLE_GROUPS.md)。规则：**变量组非空优先**，否则用下表默认值（playbook 内 `lookup('env', …)`）。

**`EXE_NAME` 推荐带 `.exe`**（如 `sinexcel_agent.exe`）；未设 `PROCESS_NAME` 时从 `EXE_NAME` 自动推导进程名。

### 全模板通用

| Variable | Default | 用于 |
|----------|---------|------|
| `SEMAPHORE_API_TOKEN` | — | bulk 回调（Status 等必填） |
| `SEMAPHORE_URL` | `http://127.0.0.1:3000` | 回调基址（控制器非本机时） |
| `EXE_NAME` | `sinexcel_agent.exe` | 扫描 / 路径 / 进程名 |
| `PROCESS_NAME` | 从 `EXE_NAME` 推导 | `Get-Process`、优雅停止 |
| `EXE_DIR` | `C:\Program Files\SINEXCEL` | 首选安装根 |
| `EXE_DIR_FALLBACK_DRIVES` | `D,E,C` | 盘符回退顺序 |
| `EXE_SCAN_LATEST` | `true` | 各盘浅层扫描最新 exe |
| `EXE_SCAN_MAX_DEPTH` | `2` | 扫描深度 |
| `APP_DIR` | `sinexcel`（同 `ZIP_NAME`） | 未开扫描时的子目录 |
| `ZIP_NAME` | `sinexcel` | 子目录名；Redeploy 的 zip 基名 |

### Status（`device_status.yml`）

| Variable | Default | 用于 |
|----------|---------|------|
| `SINEXCEL_API_PORT` | — | Agent HTTP API 端口（优先于 `API_PORT`） |
| `API_PORT` / 设备 `api_port` | `9002` | API 端口回退 |

### Start / Restart / Check-restart（写 INI + API）

| Variable | Default | 用于 |
|----------|---------|------|
| `CONFIG_FILE_NAME` | `sinexcel.iconf` | 目标机上已有 INI 文件名（仅合并 patch，不从控制器复制） |
| `RECONFIG_CLIENT_REL_PATH` | `Documents\NEWARE\BTSClient`（任务内默认） | **建议在 VG 设为** `Documents\SINEXCEL\BTSClient` |
| `SINEXCEL_API_PORT` / `API_PORT` | `9002` | SetConfig / QueryConfig 等 HTTP API |
| `SINEXCEL_START_CHECK_API` | `true` | 启动后要求 `EnableFlowInfoExtendedSqlite=true` |
| `EXE_ARGS` | 空 | 启动参数 |
| `RESTART_DELAY` | `30` | 启动前等待（秒） |
| `PROCESS_VERIFY_POLL_SECONDS` | `5` | 进程确认轮询 |
| `HIS_DATA_FROM_TIME` | 空 | INI 合并可选字段 |
| `START_POPUP_KEYWORD` | `提示` | 交互启动后自动点弹窗 |
| `START_POPUP_WAIT_SECONDS` | `3` | 等弹窗出现 |
| `STOP_POPUP_KEYWORD` | `警告` | 优雅停止确认 |
| `STOP_POPUP_MATCH_MODE` | `title_or_content` | 无标题弹窗用正文匹配 |
| `STOP_POPUP_WAIT_SECONDS` | `2` | 停前等弹窗 |
| `STOP_FORCE_AFTER_GRACEFUL` | `true` | 优雅停失败后强杀 |

### Redeploy（`device_redeploy.yml`）

| Variable | Default | 用于 |
|----------|---------|------|
| `ZIP_PATH` | `/root/sinexcel/pkg` | **仅 Redeploy**：控制器上 `{{ ZIP_NAME }}.zip` 所在目录 |

控制器上须有：`/root/sinexcel/pkg/sinexcel.zip`（或自定义 `ZIP_PATH` / `ZIP_NAME`）。

### Stop（`device_stop.yml`）

| Variable | Default | 用于 |
|----------|---------|------|
| `SINEXCEL_API_PORT` | `9002`（同 API 端口链） | 停后可选 API 探测（默认路径 `/SyncLims/QueryStatus`） |
| `SINEXCEL_API_SCHEME` | `http` | 同上 |
| `SINEXCEL_API_STATUS_PATH` | `/SyncLims/QueryStatus` | 同上 |
| `SINEXCEL_API_TOKEN` | `sinexcelapi` | 同上 |
| `SINEXCEL_API_TIMEOUT` | `8` | 同上 |
| `SINEXCEL_API_VERIFY_TLS` | `true` | 同上 |
| `SINEXCEL_STOP_CHECK_API` | `false` | 停后是否调 API |
| `STOP_GRACEFUL_PROCESS_NAME` | 同 `process_name` | 勿填错类型进程名 |

### API 路径（一般不用改）

见 `group_vars/windows_hosts.yml`：`/kafka/QueryConfig`、`SetConfig`、`IsEnable`、`QueryHistory`、`Retransmit`。

## Extra-vars

Bulk: `devices`, `configs_by_host`, `default_config`  
Single: `device`, `config`
