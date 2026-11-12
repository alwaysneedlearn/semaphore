# LANBTS playbooks (`cursor-playbooks/lanbts/`)

设备类型 **LANBTS**：从 **NBT** 复制的 Windows **服务**启停（`win_service`），**不写 INI**。

- **提供**：`device_status.yml`、`device_restart.yml`、`device_resend_data.yml`、`device_check_restart.yml`（Schedule）
- **不提供**：`device_redeploy.yml`（无重新部署 / zip 下发）
- API 与 NBT 相同：`GET /SendStatus`、`GET /ResetData`

- **`SERVICE_NAME`** → exe 文件名（如 `NBT.MES.Service.exe`）
- **`LANBTS_SERVICE_NAME`** → Windows 服务名（`win_service`，如 `NBTMESService`）
- **`SERVICE_PATH` / `LANBTS_SERVICE_PATH`** → 父目录，在其下查找 exe

任务日志：搜 **`[DEBUG-LANBTS]`**；路径解析见 **`tasks/resolve_lanbts_service_install.yml`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 巡检：心跳 API 优先 + 服务 Running 参考 |
| `device_restart.yml` | 停 → 启 → 心跳（不含 ResetData；重发见 `device_resend_data.yml`） |
| `device_resend_data.yml` | UI **重发数据**：`GET /ResetData`（不启停服务） |
| `device_check_restart.yml` | TDengine 通道新鲜度优先；不健康时服务重启 |

在 **Device types** 中绑定 Status / Restart / Resend；**Redeploy 留空**。定时巡检可在 **Schedules** 调度 `device_check_restart.yml`。

## 路径解析

| 概念 | 变量 | 示例 |
|------|------|------|
| **搜索根目录** | `SERVICE_PATH` / `LANBTS_SERVICE_PATH` | `D:\MES` |
| **exe 文件名** | `SERVICE_NAME` → `service_exe_name` | `NBT.MES.Service` → `NBT.MES.Service.exe` |
| **Windows 服务名** | `LANBTS_SERVICE_NAME` → `service_name` | `NBTMESService` |
| 解析到的 exe | `service_exe_path` / `install_path` | `D:\MES\NBTMESService\NBT.MES.Service.exe` |

解析顺序（`sem_resolve_lanbts_service_install.ps1`）：

1. `{SERVICE_PATH}\NBT.MES.Service.exe`
2. `{SERVICE_PATH}\NBT.MES.Service\NBT.MES.Service.exe`
3. `{SERVICE_PATH}\NBTMESService\NBT.MES.Service.exe`（服务名子目录 + exe 文件名）
4. 在 `SERVICE_PATH` 下递归查找（默认深度 **3**）

## Variable Group ENV

```env
SERVICE_NAME=NBT.MES.Service
LANBTS_SERVICE_NAME=NBTMESService
SERVICE_PATH=D:\MES
LANBTS_API_PORT=8885
LANBTS_SERVICE_SCAN_MAX_DEPTH=3
LANBTS_SERVICE_SCAN_LATEST=true
SEMAPHORE_API_TOKEN=<token>
# TDENGINE_URL=...
# TDENGINE_TAG_SUPPLIER=lanbts
```

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | `NBT.MES.Service` | **exe 文件名**（自动补 `.exe`） |
| `LANBTS_SERVICE_NAME` | `NBTMESService` | **Windows 服务名** |
| `SERVICE_PATH` / `LANBTS_SERVICE_PATH` | `D:\MES` | 父目录 |
| `LANBTS_API_PORT` | `8885` | 心跳 API；设备 `api_port` 优先 |
| `LANBTS_API_HEARTBEAT_PATH` | `/SendStatus` | 心跳路径 |
| `LANBTS_API_RESET_DATA_PATH` | `/ResetData` | 重发路径 |
| `LANBTS_HEARTBEAT_MAX_AGE_MINUTES` | `90` | 心跳最大间隔（分钟） |
| `LANBTS_API_TIMEOUT` | `8` | 单次 HTTP 超时（秒） |
| `LANBTS_START_CHECK_API` | `true` | 启动/重启后验证心跳 |
| `SEMAPHORE_API_TOKEN` | — | 回调 Token（必填） |

## 设备配置（非变量组）

| Category | 用途 |
|----------|------|
| `ResetData` | `GET /ResetData`（`StartDate` / `EndDate`，`yyyy-M-d`） |
