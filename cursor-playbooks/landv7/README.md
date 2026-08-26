# LANDV7 playbooks (`cursor-playbooks/landv7/`)

设备类型 **LANDV7**（与 **LANH** 功能一致：参照 LAND 路径/WinRM/API，行为更简）：

- **提供**：`device_status.yml`、`device_resend_data.yml`；另有 `device_check_restart.yml` 供 **Schedule** 定时跑
- **不提供**：restart、redeploy、stop（设备列表无对应入口）
- **Patrol / check_restart**：HTTP **QueryStatus**（同 LAND `POST …/SyncLims/QueryStatus`，`real_status=true` 为健康）
- **check_restart 修复**：API 不健康时 **ensure 启动** `ccsmon.exe`（只启动、不停止、无弹窗），启动后轮询 QueryStatus
- 重发：HTTP **Redeliver**（同 LAND SyncLims）

任务日志：搜 **`[DEBUG-LANDV7]`**（QueryStatus 解析仍可能带 `[DEBUG-LAND]`，因复用 `land/tasks/land_api_query_status*.yml`）。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | **QueryStatus API 优先** → 不健康时 WinRM/进程参考 → bulk 回调（设备 **检查状态** / Patrol） |
| `device_resend_data.yml` | UI **重发数据**：`resend_params` → `POST …/SyncLims/Redeliver`（不启停进程） |
| `device_check_restart.yml` | **仅 Schedule**：TDengine 通道新鲜度 → QueryStatus → 不健康则 ensure 启动 + API 轮询 |

在 **Device types** 中绑定 **Status / Patrol** 与 **Resend** 即可；Restart / Redeploy 留空。定时巡检：在 Semaphore **Schedules** 里直接调度指向 `device_check_restart.yml` 的模板。

## Variable Group ENV

```env
EXE_NAME=ccsmon.exe
EXE_DIR=C:\Program Files\LANDV7
APP_DIR=.
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
EXE_SCAN_MAX_DEPTH=3
# QueryStatus / Redeliver（可与 LAND 共用 LAND_API_*）
LANDV7_API_PORT=8080
LANDV7_API_TOKEN=landapi
LANDV7_API_STATUS_PATH=/SyncLims/QueryStatus
# LANDV7_QUERY_STATUS_REQUIRE_REAL_STATUS=true
# LANDV7_START_CHECK_API=true
# check_restart 通道新鲜度（可选）
# TDENGINE_URL=http://tdengine:6041
# TDENGINE_CHANNEL_STATUS_TABLE=lab_sync.dwd_channel_status
# TDENGINE_TAG_SUPPLIER=landv7
# TDENGINE_CHANNEL_STALE_HOURS=6
SEMAPHORE_API_TOKEN=<token>
```

| Variable | Default | Description |
|----------|---------|-------------|
| `EXE_NAME` | `ccsmon.exe` | 目标程序文件名 |
| `PROCESS_NAME` | 由 `EXE_NAME` 推导（`ccsmon`） | `Get-Process -Name` |
| `EXE_DIR` | `C:\Program Files\LANDV7` | 首选安装目录 |
| `APP_DIR` | `.` | `.` / 空 = `EXE_DIR\EXE_NAME` |
| `LANDV7_API_*` / `LAND_API_*` | 同 LAND SyncLims 默认 | QueryStatus / Redeliver |
| `LANDV7_QUERY_STATUS_REQUIRE_REAL_STATUS` / `LAND_*` | `true` | Patrol 健康需 `data.real_status=true` |
| `TDENGINE_CHANNEL_*` | — | 仅 check_restart 新鲜度门禁 |

## 行为摘要

```text
status (Patrol):
  POST QueryStatus（同 LAND）
    real_status=true → healthy 快路径
    HTTP 不可达/非 2xx → unhealthy 快失败
    2xx 但未健康 → WinRM 进程参考（不自动启动）

check_restart(通道不新鲜):
  POST QueryStatus
    健康 → 结束
    不健康 → WinRM ensure 启动 ccsmon → 轮询 QueryStatus

resend:
  POST Redeliver（不启停 ccsmon）
```
