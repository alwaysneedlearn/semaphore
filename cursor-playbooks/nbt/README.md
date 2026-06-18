# NBT playbooks (`cursor-playbooks/nbt/`)

Device type **NBT**: Windows **服务**启停（`win_service`），**不写 INI**。

- **`SERVICE_NAME`** → exe 文件名（如 `NBTMESService.exe`）
- **`NBT_SERVICE_NAME`** → Windows 服务名（`win_service`）
- **`SERVICE_PATH`** → 父目录，在其下查找 exe

任务日志：搜 **`[DEBUG-NBT]`**；路径解析见 **`tasks/resolve_nbt_service_install.yml`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 巡检：解析 exe + 服务 Running + 心跳 |
| `device_stop.yml` | 停止服务 |
| `device_restart.yml` | 停 → 启 → 心跳 → 可选 ResetData |
| `device_redeploy.yml` | 下发 `nbt.zip` 到 `SERVICE_PATH` + 服务重启 |
| `device_check_restart.yml` | 不健康时服务重启 |

## 路径与 Redeploy

| 概念 | 变量 | 示例 |
|------|------|------|
| **搜索根目录** | `SERVICE_PATH` / `service_base_dir` | `D:\MES` |
| **exe 文件名** | `SERVICE_NAME` → `service_exe_name` | `NBTMESService` → `NBTMESService.exe` |
| **Windows 服务名** | `NBT_SERVICE_NAME` → `service_name` | `NBT.MES.Service` |
| 解析到的 exe | `service_exe_path` / `install_path` | `D:\MES\数据上传\NBTMESService\NBTMESService.exe` |
| 安装目录 | `service_path` | exe 所在文件夹 |
| 控制器 zip | `ZIP_PATH` + `ZIP_NAME` | `/root/nbt/pkg/nbt.zip` |
| 目标机 zip | `service_parent_dir\nbt.zip` | `D:\MES\nbt.zip` |
| 解压目录 | `redeploy_extract_dest` | `D:\MES`（= `SERVICE_PATH`） |

解析顺序（`sem_resolve_nbt_service_install.ps1`）：

1. `{SERVICE_PATH}\{SERVICE_NAME}.exe`
2. `{SERVICE_PATH}\{exe主干目录}\{SERVICE_NAME}.exe`（如 `NBTMESService\NBTMESService.exe`）
3. `{SERVICE_PATH}\{NBT_SERVICE_NAME}\{SERVICE_NAME}.exe`
4. 在 `SERVICE_PATH` 下递归查找 exe（默认深度 **3**）

未解析到 exe 时**不会**对空路径执行 `win_stat`（避免 illegal path fatal）。

## Variable Group ENV

```env
SERVICE_NAME=NBTMESService
NBT_SERVICE_NAME=NBT.MES.Service
SERVICE_PATH=D:\MES
ZIP_NAME=nbt.zip
ZIP_PATH=/root/nbt/pkg
NBT_API_PORT=8885
NBT_SERVICE_SCAN_MAX_DEPTH=3
NBT_SERVICE_SCAN_LATEST=true
SEMAPHORE_API_TOKEN=<token>
```

`NBT_SERVICE_PATH` 优先于 `SERVICE_PATH`（若单独配置）。

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | — | **exe 文件名**（必填推荐；自动补 `.exe`） |
| `NBT_SERVICE_NAME` | `NBT.MES.Service` | **Windows 服务名**（`Get-Service` / 启停） |
| `SERVICE_PATH` / `NBT_SERVICE_PATH` | `D:\MES` | 父目录，在其下查找 exe |
| `NBT_SERVICE_SCAN_MAX_DEPTH` | `3` | 递归扫描最大深度 |
| `NBT_SERVICE_SCAN_LATEST` | `true` | 多个 exe 命中时取最新修改时间 |
| `ZIP_NAME` | `nbt.zip` | 安装包文件名（含 `.zip`） |
| `ZIP_PATH` | `/root/nbt/pkg` | 控制器上 zip 目录 |
| `NBT_API_PORT` | `8885` | 心跳 API；设备 `api_port` 优先 |
| `NBT_HEARTBEAT_MAX_AGE_MINUTES` | `90` | 心跳最大间隔（分钟） |
| `NBT_API_TIMEOUT` | `8` | 单次 `GET /SendStatus` 超时（秒） |
| `NBT_HEARTBEAT_START_POLL_RETRIES` | `6` | **重启/启动后** SendStatus 轮询：首次 + 6 次重试，间隔见下行 |
| `NBT_HEARTBEAT_START_POLL_DELAY` | `10` | 轮询间隔（秒） |
| `NBT_START_CHECK_API` | `true` | 启动/重启后验证心跳（关闭则不做 SendStatus） |
| `SEMAPHORE_API_TOKEN` | — | 回调 Token（必填） |
| `TDENGINE_URL` | — | 配置后 bulk 回调后写入 TDengine（见 `docs/tdengine-setup.md`） |
| `TDENGINE_TAG_SUPPLIER` | `newarerm` | NBT 建议设为 `nbt`（超级表 TAG） |
| `TDENGINE_STATUS_TABLE` | `neware_remote_computer_status` | 可按类型改为 NBT 子表名 |

**Bulk 回调 / TDengine**：仅在第二 play `hosts: localhost` 执行一次 `semaphore_bulk_put_from_hostvars.yml`（不在 `post_tasks` 里重复 bulk，避免 TDengine 写入两次）。任务日志应只看到一组 `[DEBUG-API]` bulk PUT 与 `[DEBUG-TDENGINE]`。

## 设备配置（非变量组）

| Category | 用途 |
|----------|------|
| `ResetData` | Restart 后 `GET /ResetData`（`StartDate` / `EndDate`，`yyyy-M-d`） |
