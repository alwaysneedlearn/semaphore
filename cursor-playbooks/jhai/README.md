# JHAI playbooks (`cursor-playbooks/jhai/`)

设备类型 **JHAI**：Windows 服务 **`UploaderServiceDaemon`** 管理上传程序 **`BTS.Uploader.Service.exe`**。

- **Kafka 配置 / 数据重传**：仅通过 HTTP API（**不重启服务**）
- **服务 restart**：仅在 Patrol/check_restart 判定上传心跳不健康，或手动 Restart/Redeploy 时执行

任务日志：搜 **`[DEBUG-JHAI]`**。

## Semaphore templates

| Playbook | 说明 |
|----------|------|
| `device_status.yml` | 巡检：`POST /api/get_upload_status` 优先 + 服务 Running 参考 |
| `device_stop.yml` | 停止 `UploaderServiceDaemon` |
| `device_restart.yml` | 停 → 启 → 心跳轮询（**不含** Kafka/重传 HTTP API） |
| `device_redeploy.yml` | 下发 zip + 服务重启（不含 Kafka/重传 API） |
| `device_check_restart.yml` | TDengine 通道新鲜度优先；不健康时仅服务 restart（不含 Kafka/重传 API） |
| `device_resend_data.yml` | UI **重传数据**：`resend_params` → `POST /api/resend_data_part` 或 `/api/resend_data_all`（**不重启服务**） |

## HTTP API（默认端口 9002）

| 接口 | 方法 | Body | 成功条件 |
|------|------|------|-----------|
| `/api/get_upload_status` | POST | `{}` | HTTP **200** 且 JSON **`code==200`** |
| `/api/modify_kafka_configuration` | POST | `{"kafkaConnectionInfo":"host:9092,..."}` | HTTP **200** 且 JSON **`code==200`** |
| `/api/resend_data_part` | POST | `{"testStartTime":"...","testEndTime":"..."}` | HTTP **200** 且 JSON **`code==200`** |
| `/api/resend_data_all` | POST | `{}` | HTTP **200** 且 JSON **`code==200`** |

设备 `api_port` 优先于变量组 `JHAI_API_PORT` / `API_PORT`。

## Variable Group ENV

```env
SERVICE_NAME=BTS.Uploader.Service
JHAI_SERVICE_NAME=UploaderServiceDaemon
SERVICE_PATH=D:\BTS
ZIP_NAME=jhai.zip
ZIP_PATH=/root/jhai/pkg
JHAI_API_PORT=9002
JHAI_START_CHECK_API=true
SEMAPHORE_API_TOKEN=<token>
TDENGINE_TAG_SUPPLIER=jhai
```

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | `BTS.Uploader.Service` | exe 文件名（自动补 `.exe`） |
| `JHAI_SERVICE_NAME` | `UploaderServiceDaemon` | Windows 服务名 |
| `SERVICE_PATH` / `JHAI_SERVICE_PATH` | `D:\BTS` | 安装搜索根目录 |
| `JHAI_API_PORT` / `API_PORT` | `9002` | 上传程序 HTTP API |
| `JHAI_API_TIMEOUT` | `15` | 单次 API 超时（秒） |
| `JHAI_UPLOAD_STATUS_START_POLL_RETRIES` | `6` | 启动后心跳轮询次数 |
| `JHAI_UPLOAD_STATUS_START_POLL_DELAY` | `10` | 轮询间隔（秒） |

## 设备配置（Semaphore 分类）

| Category | 字段 | 何时生效 |
|----------|------|----------|
| `ModifyKafka` 或 `KafkaConfig` | `kafkaConnectionInfo` | 仅当 playbook 设 **`jhai_include_config_apis: true`** 时调 `POST /api/modify_kafka_configuration`（当前 restart/redeploy/check_restart 均为 **false**） |
| `ResendData` / `ResendDataPart` | `ResendDataAll` 或 `testStartTime`+`testEndTime` | **仅** `device_resend_data.yml`（UI 弹窗 `resend_params`）；restart **不**读取设备配置里的重传分类（`jhai_include_resend_data: false`） |

`device_resend_data` 调用重传 API **不会** restart 服务。若将来在 restart 上启用 `jhai_include_config_apis: true`，也只会下发 Kafka，**不会**顺带重传，除非单独设 `jhai_include_resend_data: true`。

## 路径解析

与 NBT 相同逻辑：`sem_resolve_jhai_service_install.ps1` 在 `SERVICE_PATH` 下查找 `BTS.Uploader.Service.exe`（含服务名子目录与递归扫描）。
