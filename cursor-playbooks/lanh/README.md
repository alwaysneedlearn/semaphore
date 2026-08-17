# LANH playbooks (`cursor-playbooks/lanh/`)

设备类型 **LANH**（参照 LAND 路径/WinRM/交互启动，行为更简）：

- **提供**：`device_status.yml`、`device_resend_data.yml`；另有 `device_check_restart.yml` 供 **Schedule** 定时跑
- **不提供**：restart、redeploy、stop（设备列表无对应入口）
- 判断 **`ccsmon.exe`** 是否运行；**未运行则启动**
- **不停止**已有进程
- 启动 **无需** 确认弹窗（`START_POPUP_KEYWORD` 强制为空）
- 重发：HTTP **Redeliver**（同 LAND SyncLims）

任务日志：搜 **`[DEBUG-LANH]`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 建连 → 解析 exe → 进程检查 → 必要时启动 → 回调（设备 **检查状态** / Patrol） |
| `device_resend_data.yml` | UI **重发数据**：`resend_params` → `POST …/SyncLims/Redeliver`（不启停进程） |
| `device_check_restart.yml` | **仅 Schedule**：TDengine 通道新鲜度优先；不新鲜则仅 **ensure 启动**（不停止）。设备列表无 check_restart 按钮 |

在 **Device types** 中绑定 **Status / Patrol** 与 **Resend** 即可；Restart / Redeploy / **Check-restart 留空**。定时巡检：在 Semaphore **Schedules** 里直接调度指向 `device_check_restart.yml` 的模板。

## Variable Group ENV

```env
EXE_NAME=ccsmon.exe
EXE_DIR=C:\Program Files\LANH
APP_DIR=.
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
EXE_SCAN_MAX_DEPTH=3
# Redeliver（可与 LAND 共用 LAND_API_*）
LANH_API_PORT=8080
LANH_API_TOKEN=landapi
# check_restart 通道新鲜度（可选）
# TDENGINE_URL=http://tdengine:6041
# TDENGINE_CHANNEL_STATUS_TABLE=lab_sync.dwd_channel_status
# TDENGINE_TAG_SUPPLIER=lanh
# TDENGINE_CHANNEL_STALE_HOURS=6
SEMAPHORE_API_TOKEN=<token>
```

| Variable | Default | Description |
|----------|---------|-------------|
| `EXE_NAME` | `ccsmon.exe` | 目标程序文件名 |
| `PROCESS_NAME` | 由 `EXE_NAME` 推导（`ccsmon`） | `Get-Process -Name` |
| `EXE_DIR` | `C:\Program Files\LANH` | 首选安装目录 |
| `APP_DIR` | `.` | `.` / 空 = `EXE_DIR\EXE_NAME` |
| `LANH_API_*` / `LAND_API_*` | 同 LAND Redeliver 默认 | 重发 HTTP |
| `TDENGINE_CHANNEL_*` | — | 仅 check_restart 新鲜度门禁 |

## 行为摘要

```text
status / check_restart(非新鲜):
  WinRM → resolve exe → Get-Process ccsmon
    RUNNING → healthy（不重启）
    NOT_RUNNING → 交互启动（无弹窗）→ 再查进程

check_restart(通道新鲜):
  healthy 快路径（不连 WinRM）

resend:
  POST Redeliver（不启停 ccsmon）
```
