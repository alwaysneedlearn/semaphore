# DAHUA playbooks (`cursor-playbooks/dahua/`)

设备类型 **DAHUA**：**状态巡检**（`CTSMonPro` 进程）+ **数据重发**（`lims-hist.exe`）。在目标 Windows 机通过 WinRM 运行 **`lims-hist.exe`**，按时间窗解析 CTSPro 历史原始数据并上送 LIMS Kafka。

不提供：`device_stop` / `device_restart` / `device_redeploy` / `device_check_restart`。

任务日志：搜 **`[DEBUG-DAHUA]`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | UI **巡检 / Patrol all**：WinRM + **`Get-Process`**；进程在跑 → `healthy` |
| `device_resend_data.yml` | UI **重发数据**：`resend_params.start/end` → `lims-hist -from/-to` |

在 **Device types** 中绑定 **Status** 与 **Resend data** 模板；Restart / Redeploy / check_restart 留空。

## 状态巡检（CTSMonPro）

- 默认进程名 **`CTSMonPro`**（`Get-Process -Name`，不带 `.exe`）
- 变量组 **`PROCESS_NAME`** 可覆盖（不带 `.exe`）；或设 **`EXE_NAME=CTSMonPro.exe`**，未设 `PROCESS_NAME` 时自动推导
- **WinRM 不可达** → `unhealthy` / `winrm_status=offline`
- **WinRM 可达但进程未运行** → `unhealthy`，`abnormal_reason=Process not running: …`
- **进程在跑** → `healthy`（`api_status=online` 表示应用层在跑；DAHUA 无独立 HTTP 巡检 API）

## lims-hist

- 默认路径：`C:\Apps\lims-hist\lims-hist.exe`（变量组 `LIMS_HIST_EXE` 可改）
- **目标机无 exe 时**：从控制器复制（同重新部署 `win_copy`）→ 默认源 `/root/software/dahua/lims-hist.exe`（`LIMS_HIST_SRC` / `LIMS_HIST_PKG_DIR`+`LIMS_HIST_PKG_NAME` / `ZIP_PATH`）
- 源可为 **`.exe`**（直接拷到 `LIMS_HIST_EXE`）或 **`.zip`**（拷到目标目录后 `Expand-Archive`；zip 内需在目标目录下解出 `lims-hist.exe`）
- 时间格式：`yyyy-MM-ddTHH:mm:ss`（本地时区；与 lims-hist CLI 一致）
- Kafka / equipNo：读目标机 PNE `KafkaCfg.ini`（可用 `LIMS_HIST_KAFKA_CFG` 覆盖路径）
- 数据根：默认工具 `auto`（本机本地固定盘）；可用 `LIMS_HIST_ROOT` 指定逗号分隔**本地**路径（勿用 UNC）
- 默认 **`-state-dir C:\Apps\lims-hist\state`**（`LIMS_HIST_STATE_DIR` 可改）
- 默认加 **`-mtime-prefilter`**（`LIMS_HIST_MTIME_PREFILTER=false` 可关）
- **后台启动**：Semaphore 任务只负责拉起进程即结束；`lims-hist` 跑完自己退出（不超时杀进程）
- 启动前默认结束已有 `lims-hist`（`LIMS_HIST_KILL_STALE`）
- 退出码（启动脚本）：`0` 已后台拉起；目标机上工具自身：`0` 成功；`1` 业务错误；`2` 参数错误；`3` 已有实例

## Variable Group ENV

```env
# 状态巡检（device_status.yml）
EXE_NAME=CTSMonPro.exe
# PROCESS_NAME=CTSMonPro
LIMS_HIST_EXE=C:\Apps\lims-hist\lims-hist.exe
# 控制器制品（目标机缺失时复制）
LIMS_HIST_PKG_DIR=/root/software/dahua
LIMS_HIST_PKG_NAME=lims-hist.exe
# 或完整路径：LIMS_HIST_SRC=/root/software/dahua/lims-hist.exe
# 可选运行参数
# LIMS_HIST_KAFKA_CFG=C:\Program Files (x86)\PNE CTSPro\KafkaCfg.ini
# LIMS_HIST_ROOT=D:\Data,E:\CTSPro\Data
# LIMS_HIST_STATE_DIR=C:\Apps\lims-hist\state
# LIMS_HIST_MTIME_PREFILTER=true
# LIMS_HIST_KILL_STALE=true
# LIMS_HIST_DRY_RUN=false
SEMAPHORE_API_TOKEN=<token>
```

| Variable | Default | Description |
|----------|---------|-------------|
| `EXE_NAME` | `CTSMonPro.exe` | 状态巡检默认程序名（推导 `process_name`） |
| `PROCESS_NAME` | （从 `EXE_NAME` 推导） | `Get-Process -Name`，**不带** `.exe` |
| `LIMS_HIST_EXE` | `C:\Apps\lims-hist\lims-hist.exe` | 目标机可执行文件路径 |
| `LIMS_HIST_PKG_DIR` / `ZIP_PATH` | `/root/software/dahua` | 控制器上制品目录 |
| `LIMS_HIST_PKG_NAME` | `lims-hist.exe` | 控制器文件名（`.exe` 或 `.zip`） |
| `LIMS_HIST_SRC` | `{PKG_DIR}/{PKG_NAME}` | 控制器完整路径（优先） |
| `LIMS_HIST_KAFKA_CFG` | （工具默认） | PNE `KafkaCfg.ini` 路径 |
| `LIMS_HIST_ROOT` | （工具 `auto`） | 逗号分隔本地数据根 |
| `LIMS_HIST_STATE_DIR` | `C:\Apps\lims-hist\state` | `-state-dir` |
| `LIMS_HIST_MTIME_PREFILTER` | `true` | 传 `-mtime-prefilter`；`false` 则不加 |
| `LIMS_HIST_KILL_STALE` | `true` | 启动前结束已有 lims-hist 进程 |
| `LIMS_HIST_DRY_RUN` | `false` | 只解析不发 Kafka |
| `LIMS_HIST_RESUME` | `false` | 跳过已成功发送的文件 |
| `SEMAPHORE_API_TOKEN` | — | bulk 回调（写设备状态 / 操作历史） |

## 数据路径约定（目标机）

```text
{root}/{作业名}/M01Ch03[003]/
  ├── {作业名}AUX.tlt
  ├── {作业名}.sch
  └── Restore\
        ├── ch03_SaveData0001.csv
        ├── ch03_SaveData0001_auxT.csv
        └── ch03_SaveData0001_auxV.csv
```

## 常见现象

| 现象 | 说明 |
|------|------|
| `rows=0` / `orig=0` 但仍有 schedule | CSV 行时间不在 `-from/-to` 内 |
| Semaphore 任务很快成功 | 正常：只负责后台拉起；exe 在目标机继续跑，结束后自己退出 |
| exit `3` | 目标机已有 lims-hist 实例；现默认会先杀旧进程。若仍出现，查是否 `LIMS_HIST_KILL_STALE=false` |
| exe not found / copy failed | 把 `lims-hist.exe`（或 zip）放到控制器 `LIMS_HIST_PKG_DIR`；查 `[DEBUG-DAHUA] ensure_copy` |
