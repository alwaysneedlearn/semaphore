# NBT playbooks (`cursor-playbooks/nbt/`)

Device type **NBT**: Windows **服务**启停（`Start-Service` / `Stop-Service`，Ansible `win_service`），**不写 INI 配置**。健康检查靠 **心跳 API**（`GET /SendStatus`）；重启后可调 **ResetData**（设备配置分类 `ResetData`）。

任务日志：搜 **`[DEBUG-NBT]`**；服务模式标记 **`NBT_SERVICE_CTRL_REV=1`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 巡检：exe 存在 + **服务 Running** + 心跳新鲜度 |
| `device_start.yml` | **启动服务**（无配置修改） |
| `device_stop.yml` | **停止服务** |
| `device_restart.yml` | 停服务 → 启服务 → 心跳 → 可选 ResetData |
| `device_redeploy.yml` | 下发 zip + **服务重启**（无 INI） |
| `device_check_restart.yml` | 不健康时服务重启 |

## NBT HTTP API

### 心跳

- `GET http://<ip>:<port>/SendStatus` → 纯文本上次发送时间
- 健康：`HTTP 200` 且距上次发送 ≤ `NBT_HEARTBEAT_MAX_AGE_MINUTES`（默认 90）

### ResetData（仅 restart / check_restart）

- `GET …/ResetData?StartDate=…&EndDate=…`
- 日期来自设备配置 **`ResetData`**（`yyyy-M-d`），**不是**变量组

## Variable Group ENV

| Variable | Default | Description |
|----------|---------|-------------|
| **`NBT_SERVICE_NAME`** | `nbt_agent` | Windows 服务名（`Get-Service -Name`） |
| `NBT_API_PORT` | **8885** | Agent API；设备 `api_port` 优先 |
| `NBT_API_HEARTBEAT_PATH` | `/SendStatus` | 心跳路径 |
| `NBT_API_RESET_DATA_PATH` | `/ResetData` | 补数据路径 |
| `NBT_HEARTBEAT_MAX_AGE_MINUTES` | `90` | 心跳最大间隔（分钟） |
| `NBT_API_TIMEOUT` | `8` | HTTP 超时（秒） |
| `NBT_START_CHECK_API` | `true` | 启动/重启后验证心跳 |
| `NBT_STOP_CHECK_API` | `false` | 停止后 GET 心跳（仅 DEBUG） |
| `SEMAPHORE_API_TOKEN` | — | 回调 Token（必填） |

**路径相关**（仅 Status / Redeploy 检查安装目录，启停不依赖）：

| Variable | Default |
|----------|---------|
| `EXE_NAME` | `nbt_agent.exe` |
| `ZIP_NAME` | `nbt` |
| `EXE_DIR` | `C:\Program Files\NBT` |
| `EXE_DIR_FALLBACK_DRIVES` | `D,E,C` |
| `ZIP_PATH` | `/root/nbt/pkg`（仅 Redeploy） |

**不再需要**（服务模式下）：`CONFIG_FILE_NAME`、`SystemConfig` 分类、`HIS_DATA_FROM_TIME`、启动弹窗相关变量。

## 最小变量组示例

```env
NBT_SERVICE_NAME=nbt_agent
NBT_API_PORT=8885
SEMAPHORE_API_TOKEN=<token>
```

Redeploy 时再加 `EXE_DIR`、`ZIP_PATH`、`ZIP_NAME`、`EXE_NAME` 等。
