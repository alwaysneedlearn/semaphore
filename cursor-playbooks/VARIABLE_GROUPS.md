# Variable Groups（变量组）配置指南

Semaphore **变量组**通过模板绑定的 **Environment** 注入 Ansible `lookup('env', 'VAR')`。规则：**环境变量非空（trim 后）优先**，否则用 playbook 内默认值。

## 基本原则

| 原则 | 说明 |
|------|------|
| **一类型一组** | LAND / SINEXCEL / NBT / JHAI / NEWARE / DAHUA / LANH 各建 **独立变量组**，绑定到该类型的模板。不要混用跨类型默认值。 |
| **全项目默认放 VG** | 如 `EXE_NAME`、`ZIP_NAME`、`API_PORT`、弹窗关键字、盘符回退、浅层扫描（SINEXCEL）。 |
| **SINEXCEL 安装路径** | **变量组即可**：`EXE_DIR` + `EXE_DIR_FALLBACK_DRIVES` + `EXE_SCAN_LATEST` + `EXE_NAME` 在各盘符下自动找最新 exe，**不必**逐台配 `Install.ExeDir` / `ExePath`。 |
| **Install（可选）** | 仅当某台机无法被盘符扫描覆盖时，才在设备/类型配置里写 **`Install`** / **`Paths`** 覆盖。见 [`sinexcel/README.md`](sinexcel/README.md)。 |
| **回调 Token** | `SEMAPHORE_API_TOKEN`（必填才能 bulk 回调）；可选 `SEMAPHORE_URL`（默认 `http://127.0.0.1:3000`）。 |

## `EXE_NAME` 要不要加 `.exe`？

**推荐：变量组里写带 `.exe` 的完整文件名**（与磁盘上的 exe 一致）。

| 变量 | 格式 | 示例 |
|------|------|------|
| **`EXE_NAME`** | **带 `.exe`**（推荐） | `LHBTS.exe`、`sinexcel_agent.exe`、`nbt_agent.exe`、`uu.exe` |
| **`PROCESS_NAME`** | **不带 `.exe`**（可选） | `LHBTS`、`sinexcel_agent`、`uu` |

Playbook 会统一规范化：

1. 去掉已有 `.exe` 后缀得到 base name  
2. **`exe_name`** = `base + '.exe'`（拼路径、扫描、存在性检查）  
3. **`process_name`** = base（`Get-Process`、优雅停止）

因此 **旧配置仍兼容**：

- NEWARE 曾用 `EXE_NAME=uu`（无后缀）→ 仍解析为 `uu.exe` / 进程 `uu`  
- 若写成 `EXE_NAME=uu.exe` → 同样正确，**不会**变成 `uu.exe.exe`

**不要**在 VG 里同时改 `EXE_NAME` 和错误的 `PROCESS_NAME`（如 SINEXCEL 写成 `LHBTS`）；未设置 `PROCESS_NAME` 时自动从 `EXE_NAME` 推导。

## 各设备类型变量组示例

### LAND

```env
EXE_NAME=LHBTS.exe
APP_DIR=LHBTS
EXE_DIR=C:\Program Files\LAND
EXE_DIR_FALLBACK_DRIVES=D,F,C
ZIP_NAME=LHBTS5.1.4.4
ZIP_PATH=/root/land/packages
LAND_API_TOKEN=landapi
STOP_POPUP_KEYWORD=警告
SEMAPHORE_API_TOKEN=<token>
```

路径布局：`{{ EXE_DIR }}\{{ APP_DIR }}\{{ EXE_NAME }}`。详见 [`land/README.md`](land/README.md)。

### SINEXCEL

```env
# 路径与进程
EXE_NAME=sinexcel_agent.exe
APP_DIR=sinexcel
EXE_DIR=C:\Program Files\SINEXCEL
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
EXE_SCAN_MAX_DEPTH=2
# Redeploy：控制器上的安装包目录（仅 device_redeploy 复制 zip）
ZIP_PATH=/root/sinexcel/pkg
ZIP_NAME=sinexcel
# Agent HTTP API（巡检 / 启动 / 重传等）
SINEXCEL_API_PORT=9002
API_PORT=9002
SINEXCEL_API_TIMEOUT=30
SINEXCEL_START_CHECK_API=true
# 弹窗（优雅停 / 交互启动）
START_POPUP_KEYWORD=提示
START_POPUP_WAIT_SECONDS=3
STOP_POPUP_KEYWORD=警告
STOP_POPUP_MATCH_MODE=title_or_content
STOP_POPUP_WAIT_SECONDS=2
STOP_FORCE_AFTER_GRACEFUL=true
# 回调
SEMAPHORE_API_TOKEN=<token>
```

**`ZIP_PATH` 仅 Redeploy**：Start / Restart **不写 INI**，配置经 HTTP API；Redeploy 需要控制器上 `{{ ZIP_PATH }}/{{ ZIP_NAME }}.zip`（默认 `/root/sinexcel/pkg/sinexcel.zip`）。

**API 端口**：设备 `api_port` → **`SINEXCEL_API_PORT`** → 旧名 `SINEXCEL_KAFKA_API_PORT` → `API_PORT` → **9002**（用于 `POST …/kafka/QueryConfig` 等 agent HTTP API）。

**API 超时**：统一环境变量 **`SINEXCEL_API_TIMEOUT`**（默认 **30** 秒），覆盖全部 `/kafka/*` HTTP `uri`（含 Retransmit），无单独 Retransmit 超时。

**不必**为每台机单独配 `Install`：playbook 在 `D:\`、`E:\`… 上按 `EXE_DIR_FALLBACK_DRIVES` 浅层扫描 `EXE_NAME`（`EXE_SCAN_LATEST=true`），取 mtime 最新路径。仅极个别例外机台才用可选 **`Install.ExePath`** 覆盖。详见 [`sinexcel/README.md`](sinexcel/README.md)。

### NBT

NBT 为 **Windows 服务**启停，**不写 INI**。变量分工：

- **`SERVICE_NAME`**：exe 文件名（如 `NBTMESService` 或 `NBTMESService.exe`）
- **`NBT_SERVICE_NAME`**：Windows 服务名（`Get-Service -Name`，如 `NBT.MES.Service`）
- **`SERVICE_PATH`**：父目录，在其下查找 exe

```env
SERVICE_NAME=NBT.MES.Service
NBT_SERVICE_NAME=NBTMESService
SERVICE_PATH=D:\MES
ZIP_NAME=nbt.zip
ZIP_PATH=/root/nbt/pkg
NBT_API_PORT=8885
NBT_SERVICE_SCAN_MAX_DEPTH=3
SEMAPHORE_API_TOKEN=<token>
# TDengine（与 NEWARE 共用 shared 写入任务；NBT 建议单独 TAG）
TDENGINE_URL=http://tdengine:6041
TDENGINE_TIMEZONE=Asia/Shanghai
TDENGINE_TAG_SUPPLIER=nbt
TDENGINE_STATUS_TABLE=nbt_remote_computer_status
```

Redeploy：控制器 `{{ ZIP_PATH }}/{{ ZIP_NAME }}` → 目标 `D:\MES\数据上传\nbt.zip`，解压到 `service_path` 的父目录。

TDengine：任务日志搜 **`[DEBUG-TDENGINE]`**；未配置 `TDENGINE_URL` 时会打印 `TDengine publish skipped`。

详见 [`nbt/README.md`](nbt/README.md)。

### JHAI

JHAI 为 **Windows 服务** `UploaderServiceDaemon` + **BTS 上传 HTTP API**（默认 **9002**）。**Kafka 修改 / 数据重传仅调 API，不 restart 服务**。

```env
SERVICE_NAME=BTS.Uploader.Service
JHAI_SERVICE_NAME=UploaderServiceDaemon
SERVICE_PATH=D:\BTS
ZIP_NAME=jhai.zip
ZIP_PATH=/root/jhai/pkg
JHAI_API_PORT=9002
API_PORT=9002
JHAI_START_CHECK_API=true
SEMAPHORE_API_TOKEN=<token>
TDENGINE_TAG_SUPPLIER=jhai
```

设备配置分类：`ModifyKafka.kafkaConnectionInfo`、`ResendData.ResendDataAll=true`（全量重传，忽略时间段）、`ResendData`/`ResendDataPart` 的 `testStartTime` 与 `testEndTime`（**均有值**时分段重传）。详见 [`jhai/README.md`](jhai/README.md)。

### NEWARE

```env
EXE_NAME=uu.exe
ZIP_NAME=test
EXE_DIR=D:\Program Files\NEWARE
ZIP_PATH=/root/neware/dbwb
CONFIG_FILE_NAME=NWReport_DBWB.iconf
API_PORT=9002
API_STATUS_CALL_TYPE=1
SEMAPHORE_API_TOKEN=<token>
# check_restart 通道新鲜度（批量一次 SELECT；未配置则跳过）
TDENGINE_URL=http://tdengine:6041
TDENGINE_CHANNEL_STATUS_TABLE=lab_sync.dwd_channel_status
TDENGINE_TAG_SUPPLIER=newarerm
TDENGINE_CHANNEL_STALE_HOURS=6
TDENGINE_TIMEZONE=Asia/Shanghai
```

路径：`{{ EXE_DIR }}\{{ ZIP_NAME }}\{{ EXE_NAME }}`（与 NBT 相同，`EXE_NAME` 已含 `.exe`）。详见 [`neware/README.md`](neware/README.md)。`TDENGINE_CHANNEL_*` 仅 **check_restart** 读取通道 `LAST(insert_time)`；状态写入仍用 `TDENGINE_STATUS_TABLE`。

### DAHUA

DAHUA **仅支持重发数据**：在目标机运行 **`lims-hist.exe`**（CTSPro → LIMS Kafka）。不配置 Status / Restart / Redeploy / check_restart 模板。

```env
LIMS_HIST_EXE=C:\Apps\lims-hist\lims-hist.exe
LIMS_HIST_PKG_DIR=/root/software/dahua
LIMS_HIST_PKG_NAME=lims-hist.exe
# LIMS_HIST_SRC=/root/software/dahua/lims-hist.exe
# LIMS_HIST_KAFKA_CFG=C:\Program Files (x86)\PNE CTSPro\KafkaCfg.ini
# LIMS_HIST_ROOT=D:\Data
# LIMS_HIST_STATE_DIR=C:\Apps\lims-hist\state
# LIMS_HIST_NO_SCHEDULE=true
# LIMS_HIST_KILL_STALE=true
# LIMS_HIST_DRY_RUN=false
SEMAPHORE_API_TOKEN=<token>
```

目标机无 `LIMS_HIST_EXE` 时，从控制器复制：`LIMS_HIST_PKG_DIR`/`LIMS_HIST_PKG_NAME` → `/root/software/dahua/lims-hist.exe`（或 `LIMS_HIST_SRC` 完整路径）。重发默认 `-state-dir C:\Apps\lims-hist\state`，**后台拉起**（Semaphore 不等待）；进程跑完自己退出。

详见 [`dahua/README.md`](dahua/README.md)。

### LANH

LANH 提供 **status / resend**（设备类型绑定这两项即可）；**check_restart** 仅通过 **Schedule** 定时执行（设备列表无按钮，Device types UI 已去掉 Check-restart 绑定）。确保 **`ccsmon.exe`** 在跑（未运行则启动，**不停止**、**无确认弹窗**）；重发走 SyncLims **Redeliver**（同 LAND）。不配置 Restart / Redeploy。

```env
EXE_NAME=ccsmon.exe
EXE_DIR=C:\Program Files\LANH
APP_DIR=.
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
LANH_API_PORT=8080
LANH_API_TOKEN=landapi
# TDENGINE_URL=http://tdengine:6041
# TDENGINE_CHANNEL_STATUS_TABLE=lab_sync.dwd_channel_status
# TDENGINE_TAG_SUPPLIER=lanh
# TDENGINE_CHANNEL_STALE_HOURS=6
SEMAPHORE_API_TOKEN=<token>
```

详见 [`lanh/README.md`](lanh/README.md)。

## Stop 模板注意事项

| 类型 | 说明 |
|------|------|
| LAND / SINEXCEL | `STOP_GRACEFUL_PROCESS_NAME` 默认 = 从 `EXE_NAME` 推导的 `process_name`，**不要**填主程序别名（如 `LHBTS` 当实际进程是 `sinexcel_agent`） |
| SINEXCEL | `STOP_POPUP_MATCH_MODE=title_or_content` 用于无标题弹窗 |
| NEWARE | 强制结束进程；`EXE_NAME` 规范化后传给辅助脚本时会去掉 `.exe` 再查 `Get-Process` |

## 不要放进变量组的内容

- `semaphore_project_id`（Semaphore 任务 extra-vars 自动注入）  
- 每台不同的 `api_port`（用设备字段 `api_port` 或 `API_PORT` 默认）  
- 单台例外路径（可选 **Install**，一般 SINEXCEL 用不到）

## Playbook 内规范化片段

各 `device_*.yml` 使用同一 Jinja 块（仅 `_exe_name_default` 按类型不同）：

```yaml
_exe_name_default: sinexcel_agent.exe
_exe_name_env: "{{ (lookup('env', 'EXE_NAME') | default('', true) | trim) }}"
_exe_name_base: "{{ (_exe_name_env if (_exe_name_env | length > 0) else _exe_name_default) | regex_replace('(?i)\\.exe$', '') }}"
exe_name: "{{ _exe_name_base ~ '.exe' }}"
process_name: "{{ ((lookup('env', 'PROCESS_NAME') | default('', true) | trim) | regex_replace('(?i)\\.exe$', '')) | default(_exe_name_base, true) }}"
```

（YAML 双引号内 Jinja 正则须写成 `'(?i)\\.exe$'`，不能写 `'(?i)\.exe$'`，否则 Ansible 报 `unknown escape character`。）

参考：[`shared/vars/exe_name_from_env.yml`](shared/vars/exe_name_from_env.yml)。
