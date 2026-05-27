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
| `TDENGINE_STATUS_TABLE` | 否 | 超级表或子表名，默认 `neware_remote_computer_status` |
| `TDENGINE_TAG_SUPPLIER` | 否 | 超级表 **TAG** `supplier` 的值，默认 `newarerm`（**不是**数据列） |
| `TDENGINE_SUPER_TABLE` | 否 | 若子表名与超级表名不同，填超级表名；playbook 生成 `INSERT INTO <table> USING <super> TAGS(...)` |

未设置 `TDENGINE_URL` 时跳过 TDengine，仅写 Semaphore DB。

## 表结构（NEWARE）

```
describe `lab`.`neware_remote_computer_status`
field          type       length
ts             TIMESTAMP  8        -- 固定值 2262-01-01T08:00:00.000000000+08:00
computer_name  VARCHAR    200      -- 设备 hostname
status         VARCHAR    200      -- online / offline
updated_time   TIMESTAMP  8        -- 写入时间
check_time     TIMESTAMP  8        -- 写入时间（同 updated_time）

TAG: supplier (NCHAR) — 写入时通过 TAGS('newarerm')，不是普通列
```

## 映射规则

- `device_status` **`healthy`** → `status` = **`online`**；其余 → **`offline`**
- `computer_name` = bulk 回调行的 **`hostname`**
- `updated_time` = `check_time` = playbook 执行时的实际 UTC 时间戳
- **`supplier`** 为超级表 **TAG**，固定 **`newarerm`**（Variable Group：`TDENGINE_TAG_SUPPLIER`）

## 每台设备只保留一行（ts 固定策略）

`ts` 固定写入：

```
2262-01-01T08:00:00.000000000+08:00
```

每次任务都用同一个 `ts` 写入；`updated_time` / `check_time` 仍写入本次真实时间，反映最后更新时刻。

## 覆盖的操作

凡包含 **`semaphore_bulk_put_from_hostvars.yml`** 的模板均会尝试写 TDengine（在配置了 `TDENGINE_URL` 时）：

- Patrol / `device_status.yml`
- `device_start.yml` / `device_stop.yml` / `device_restart.yml`
- `check_restart_redeploy.yml`

**不包含**：`device_discovery.yml`（发现不写设备状态 bulk）。

TCP 定时探针、RDP Probe **不**写 TDengine。

## 调试

任务日志中搜索 **`[DEBUG-TDENGINE]`**。成功时 REST 仍可能 **HTTP 200**，请查看：

- `raw_body` / `json.code`（`0` 为成功）
- `json.desc`（失败时的错误说明）
- `json.affected_rows`（写入行数；为 `0` 时表内无变化）

请求 URL 为 `{TDENGINE_URL}/rest/sql/{TDENGINE_DATABASE}`（无状态连接，库名在 URL 中）。INSERT 示例（表名不带库前缀）：

```sql
INSERT INTO neware_remote_computer_status TAGS('newarerm')
VALUES ('2262-01-01 08:00:00.000','JSC3211CCXW004','online','2026-05-27 09:09:39.','2026-05-27 09:09:39.');
```

若使用子表 + 超级表：`INSERT INTO <child> USING <super_stable> TAGS('newarerm') VALUES (...)`（设置 `TDENGINE_SUPER_TABLE`）。

**注意**：若表主键仅 `ts` 且所有设备共用同一固定 `ts`，TDengine 中最终可能只保留 **一行**（后写入覆盖先写入）；需多行时请确认表结构是否以 `computer_name` 等区分主键/标签。
