# TDengine 状态写入（模板 / Variable Group）

Semaphore **不再**在服务端自动写 TDengine。设备操作 playbook（`cursor-playbooks/neware/device_*.yml`、`check_restart_redeploy.yml`）在 **`PUT /devices/status/bulk`** 之后，由 **`tasks/semaphore_tdengine_publish_from_bulk.yml`** 将**本任务**的 bulk 行写入 TDengine（单台与批量共用同一任务链）。

## Variable Group（ENV）

在绑定设备模板的 **Variable Group → Environment** 中配置：

| 变量 | 必填 | 说明 |
|------|------|------|
| `TDENGINE_URL` | 是 | REST 基址，如 `http://tdengine:6041`（不要带 `/rest/sql`，playbook 会自动追加） |
| `TDENGINE_USER` | 否 | Basic 认证用户名 |
| `TDENGINE_PASSWORD` | 否 | Basic 认证密码 |
| `TDENGINE_DATABASE` | 否 | 库名，默认 `semaphore_devices` |
| `TDENGINE_STATUS_TABLE` | 否 | 表名，默认 `status` |

未设置 `TDENGINE_URL` 时跳过 TDengine，仅写 Semaphore DB。

## 表结构示例

```sql
CREATE DATABASE IF NOT EXISTS semaphore_devices;
CREATE STABLE IF NOT EXISTS semaphore_devices.status (
  ts TIMESTAMP,
  project_id INT,
  device_id INT,
  hostname NCHAR(128),
  ip NCHAR(64),
  status NCHAR(16),
  device_status_raw NCHAR(32),
  winrm_status NCHAR(16),
  api_status NCHAR(16)
) TAGS (dummy INT);
```

（若团队使用子表/其它建模，请按 DBA 规范调整；playbook 当前使用 `INSERT INTO \`db\`.\`table\` (...) VALUES ...`。）

## 映射规则

- `device_status` **`healthy`** → TDengine 列 **`status`** = `online`，否则 `offline`
- 同时写入 `device_status_raw`、`winrm_status`、`api_status`（来自 playbook 回调行）
- `device_id` / `project_id` 来自任务 extra-vars 中的 `devices` / `device` 与 `semaphore_project_id`

## 覆盖的操作

凡包含 **`semaphore_bulk_put_from_hostvars.yml`** 的模板均会尝试写 TDengine（在配置了 `TDENGINE_URL` 时）：

- Patrol / `device_status.yml`
- `device_start.yml` / `device_stop.yml` / `device_restart.yml`
- `check_restart_redeploy.yml`

**不包含**：`device_discovery.yml`（发现不写设备状态 bulk）。

TCP 定时探针、RDP Probe **不**写 TDengine。

## 调试

任务日志中搜索 **`[DEBUG-TDENGINE]`**。
