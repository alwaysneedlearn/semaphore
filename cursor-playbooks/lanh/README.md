# LANH playbooks (`cursor-playbooks/lanh/`)

设备类型 **LANH**（参照 LAND 路径/WinRM/交互启动，行为更简）：

- **只提供** `device_status.yml`（巡检 / Patrol）
- **不提供**：restart、redeploy、check_restart、stop、resend
- 判断 **`ccsmon.exe`** 是否运行；**未运行则启动**
- **不停止**已有进程
- 启动 **无需** 确认弹窗（`START_POPUP_KEYWORD` 强制为空）

任务日志：搜 **`[DEBUG-LANH]`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 建连 → 解析 exe → 进程检查 → 必要时启动 → 回调 |

在 **Device types** 中只绑定 **Status / Patrol** 模板即可。

## Variable Group ENV

```env
EXE_NAME=ccsmon.exe
EXE_DIR=C:\Program Files\LANH
APP_DIR=.
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
EXE_SCAN_MAX_DEPTH=3
# EXE_ARGS=
# RESTART_DELAY=15
SEMAPHORE_API_TOKEN=<token>
```

| Variable | Default | Description |
|----------|---------|-------------|
| `EXE_NAME` | `ccsmon.exe` | 目标程序文件名 |
| `PROCESS_NAME` | 由 `EXE_NAME` 推导（`ccsmon`） | `Get-Process -Name` |
| `EXE_DIR` | `C:\Program Files\LANH` | 首选安装目录 |
| `APP_DIR` | `.` | `.` / 空 = `EXE_DIR\EXE_NAME`；否则 `EXE_DIR\APP_DIR\EXE_NAME` |
| `EXE_DIR_FALLBACK_DRIVES` | `D,E,C` | 盘符回退 |
| `EXE_SCAN_LATEST` | `true` | 浅层扫描最新 `ccsmon.exe` |
| `RESTART_DELAY` | `15` | 启动后等待/校验（秒，传给启动脚本） |

路径解析与 LAND 相同：`resolve_exe_dir_windows` + 共享 `sem_*.ps1`（`sem_files_dir` → `../shared/files`）。

## 行为摘要

```text
WinRM OK
  → resolve exe_path
  → Get-Process ccsmon
       RUNNING  → healthy（不重启）
       NOT_RUNNING → 计划任务交互启动（无弹窗确认）→ 再查进程
  → bulk 回调
```
