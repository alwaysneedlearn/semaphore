# TDengine 状态写入（模板 / Variable Group）

Semaphore **不再**在服务端自动写 TDengine。设备操作 playbook（`cursor-playbooks/neware/device_*.yml`、`check_restart_redeploy.yml`）在 **`PUT /devices/status/bulk`** 之后，由 **`tasks/semaphore_tdengine_publish_from_bulk.yml`** 将**本任务**的 bulk 行写入 TDengine（单台与批量共用同一任务链）。

## Variable Group（ENV）

在绑定设备模板的 **Variable Group → Environment** 中配置：

| 变量 | 必填 | 说明 |
|------|------|------|
| `TDENGINE_URL` | 是 | REST 基址，如 `http://tdengine:6041`（不要带 `/rest/sql`，playbook 会自动追加） |
| `TDENGINE_USER` | 否 | Basic 认证用户名 |
| `TDENGINE_PASSWORD` | 否 | Basic 认证密码 |
| `TDENGINE_DATABASE` | 否 | 库名，默认 `lab` |
| `TDENGINE_STATUS_TABLE` | 否 | 表名，默认 `neware_remote_computer_status` |

未设置 `TDENGINE_URL` 时跳过 TDengine，仅写 Semaphore DB。

## 表结构（NEWARE）

```
describe `lab`.`neware_remote_computer_status`
field          type       length
ts             TIMESTAMP  8        -- TDengine 时序主键（自动递增）
computer_name  VARCHAR    200      -- 设备 hostname
status         VARCHAR    200      -- online / offline
updated_time   TIMESTAMP  8        -- 写入时间
check_time     TIMESTAMP  8        -- 写入时间（同 updated_time）
supplier       VARCHAR    100      -- 固定 newarerm
```

## 映射规则

- `device_status` **`healthy`** → `status` = **`online`**；其余 → **`offline`**
- `computer_name` = bulk 回调行的 **`hostname`**
- `updated_time` = `check_time` = playbook 执行时的实际 UTC 时间戳
- `supplier` 固定为 **`newarerm`**

## 每台设备只保留一行（ts 固定策略）

`ts` 由 `computer_name` 的 MD5 哈希前 8 位（十六进制 → 整数毫秒）加上基准时间 `2000-01-01 00:00:00 UTC` 派生：

```
ts_ms = 946684800000 + int(md5(computer_name)[0:8], 16)
```

相同 hostname 每次算出相同 `ts`，TDengine INSERT 相同主键时覆盖该行，从而实现**每台设备只保留最新一行**。`updated_time` / `check_time` 仍写入本次真实时间，反映最后更新时刻。

## 覆盖的操作

凡包含 **`semaphore_bulk_put_from_hostvars.yml`** 的模板均会尝试写 TDengine（在配置了 `TDENGINE_URL` 时）：

- Patrol / `device_status.yml`
- `device_start.yml` / `device_stop.yml` / `device_restart.yml`
- `check_restart_redeploy.yml`

**不包含**：`device_discovery.yml`（发现不写设备状态 bulk）。

TCP 定时探针、RDP Probe **不**写 TDengine。

## 调试

任务日志中搜索 **`[DEBUG-TDENGINE]`**。
