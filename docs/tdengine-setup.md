# TDengine 状态同步（运维）

Semaphore 在 playbook 通过 `PUT /api/project/{id}/devices/status/bulk` 更新 SQLite 后，可选将**当前 DB 快照**写入 TDengine（`healthy` → `online`，其余 → `offline`）。

## 1. 启用（管理端）

1. 使用 **admin** 账号登录。
2. 打开 **Users** 页工具栏 **TDengine**，或账号菜单 **Platform → TDengine**，或访问 `/admin/tdengine`。
3. 填写 REST URL（例 `http://127.0.0.1:6041`）、用户、密码、数据库名，打开 **Enable**，点 **Test connection** 后 **Save**。

也可在 `config.json` 或环境变量 `SEMAPHORE_TDENGINE_*` 提供默认值；DB 中 Admin 保存的配置优先级更高。

## 2. 建库与超级表（NEWARE 默认表 `status`）

```sql
CREATE DATABASE IF NOT EXISTS semaphore_devices;
CREATE STABLE IF NOT EXISTS semaphore_devices.status (
  ts TIMESTAMP,
  status NCHAR(16),
  device_status_raw NCHAR(16),
  winrm_status NCHAR(16),
  api_status NCHAR(16)
) TAGS (project_id INT, device_id INT, hostname NCHAR(128), ip NCHAR(64));
```

其它设备类型在 **Devices → Device types** 中配置 **TDengine status table**（未填时 NEWARE 用 `status`，其它类型默认 `status_<profile_key小写>`）。

## 3. 写入时机

- 每次 bulk 回调成功更新 DB 后，按 **profile** 对该类型下全部设备做**全量**快照写入对应表。
- TCP 定时探针**不**写 TDengine（仅更新 DB 协议列）。

## 4. 设备类型（Profile）

- 新项目自动创建 **NEWARE** 类型；可在 **Devices → Device types** 新增类型并绑定 6 个模板。
- 设备须绑定 `device_profile_id`；Patrol / 批量启动会按类型拆成多个 Task。

详见 [plan-tdengine-device-profiles.md](./plan-tdengine-device-profiles.md)。
