# NBT playbooks (`cursor-playbooks/nbt/`)

Device type **NBT**: Windows **服务**启停（`win_service`），**不写 INI**。安装目录用 **`service_path`**，服务名用 **`service_name`**（不再使用 `exe_name` / `exe_path`）。

任务日志：搜 **`[DEBUG-NBT]`**；布局标记 **`NBT_SERVICE_CTRL_REV=2`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 巡检：`service_path` 存在 + 服务 Running + 心跳 |
| `device_start.yml` / `device_stop.yml` | 启停服务 |
| `device_restart.yml` | 停 → 启 → 心跳 → 可选 ResetData |
| `device_redeploy.yml` | 下发 `nbt.zip` 到 `service_parent_dir` + 服务重启 |
| `device_check_restart.yml` | 不健康时服务重启 |

## 路径与 Redeploy

| 概念 | 变量 | 示例 |
|------|------|------|
| 服务安装目录 | `service_path` | `D:\MES\数据上传\NBT.MES.Service` |
| Windows 服务名 | `service_name` | `NBT.MES.Service` |
| 控制器 zip | `ZIP_PATH` + `ZIP_NAME` | `/root/nbt/pkg/nbt.zip` |
| 目标机 zip | `service_parent_dir\nbt.zip` | `D:\MES\数据上传\nbt.zip` |
| 解压目录 | `service_parent_dir` | `D:\MES\数据上传` |

`service_parent_dir` = `service_path` 的父目录（playbook 自动推导）。

## Variable Group ENV

```env
SERVICE_NAME=NBT.MES.Service
SERVICE_PATH=D:\MES\数据上传\NBT.MES.Service
ZIP_NAME=nbt.zip
ZIP_PATH=/root/nbt/pkg
NBT_API_PORT=8885
SEMAPHORE_API_TOKEN=<token>
```

也接受 `NBT_SERVICE_NAME` / `NBT_SERVICE_PATH`（优先于 `SERVICE_*`）。

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` / `NBT_SERVICE_NAME` | `NBT.MES.Service` | `Get-Service -Name` |
| `SERVICE_PATH` / `NBT_SERVICE_PATH` | `D:\MES\数据上传\NBT.MES.Service` | 安装目录（巡检 / Redeploy 检查） |
| `ZIP_NAME` | `nbt.zip` | 安装包文件名（含 `.zip`） |
| `ZIP_PATH` | `/root/nbt/pkg` | 控制器上 zip 目录 |
| `NBT_API_PORT` | `8885` | 心跳 API；设备 `api_port` 优先 |
| `NBT_HEARTBEAT_MAX_AGE_MINUTES` | `90` | 心跳最大间隔（分钟） |
| `NBT_START_CHECK_API` | `true` | 启动/重启后验证心跳 |
| `SEMAPHORE_API_TOKEN` | — | 回调 Token（必填） |

## 设备配置（非变量组）

| Category | 用途 |
|----------|------|
| `ResetData` | Restart 后 `GET /ResetData`（`StartDate` / `EndDate`，`yyyy-M-d`） |
