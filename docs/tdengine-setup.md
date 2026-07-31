# TDengine 状态写入（模板 / Variable Group）

Semaphore **不再**在服务端自动写 TDengine。凡包含 **`semaphore_bulk_put_from_hostvars.yml`** 的设备 playbook（**NEWARE / NBT / LAND / SINEXCEL** 的 status/start/stop/restart/redeploy 等）在 **`PUT /devices/status/bulk`** 之后，由 **`shared/tasks/semaphore_tdengine_publish_from_bulk.yml`** 将**本任务**的 bulk 行写入 TDengine（单台与批量共用同一任务链）。**不要求** `sem_profile_tasks_dir`。

## Variable Group（ENV）

在绑定设备模板的 **Variable Group → Environment** 中配置：

| 变量 | 必填 | 说明 |
|------|------|------|
| `TDENGINE_URL` | 是 | REST 基址，如 `http://tdengine:6041`（不要带 `/rest/sql`，playbook 会自动追加） |
| `TDENGINE_USER` | 否 | Basic 认证用户名 |
| `TDENGINE_PASSWORD` | 否 | Basic 认证密码 |
| `TDENGINE_DATABASE` | 否 | 库名，默认 `lab` |
| `TDENGINE_STATUS_TABLE` | 否 | 子表名，默认 `neware_remote_computer_status`（SQL 中为 `lab.neware_remote_computer_status`） |
| `TDENGINE_SUPER_TABLE` | 否 | 超级表名，默认 `dws_computer_status`（SQL 中为 `lab.dws_computer_status`） |
| `TDENGINE_TAG_SUPPLIER` | 否 | 超级表 **TAG** `supplier` 的值，默认 `newarerm`（**不是**数据列） |
| `TDENGINE_TIMEZONE` | 否 | `updated_time` / `check_time` 的 IANA 时区，默认 **`Asia/Shanghai`**（北京时间 UTC+8） |

未设置 `TDENGINE_URL` 时跳过 TDengine，仅写 Semaphore DB。

## 表结构（NEWARE）

```
describe `lab`.`neware_remote_computer_status`
field          type       length
ts             TIMESTAMP  8        -- 固定值 2262-01-01T08:00:00.000000000+08:00
computer_name  VARCHAR    200      -- 设备 hostname
ip_addr        VARCHAR    200      -- 设备 IP（无 IP 时回退 hostname）
status         VARCHAR    200      -- online / offline
updated_time   TIMESTAMP  8        -- 写入时间
check_time     TIMESTAMP  8        -- 写入时间（同 updated_time）
abnormal_reason VARCHAR   255      -- 异常原因（offline 时）

TAG: supplier (NCHAR) — 写入时通过 TAGS('newarerm')，不是普通列
```

## 映射规则

- `device_status` **`healthy`** → `status` = **`online`**；其余 → **`offline`**
- `computer_name` = bulk 回调行的 **`hostname`**
- `ip_addr` = bulk 回调行的 **`ip`**（为空时回退 `hostname`）
- `updated_time` = `check_time` = playbook 写入瞬间的本地时间（**`TDENGINE_TIMEZONE`**，默认 **`Asia/Shanghai`**），格式 `YYYY-MM-DD HH:MM:SS`（秒级，无毫秒/微秒）。字符串不含时区后缀；需 UTC 时在 Variable Group 设 `TDENGINE_TIMEZONE=UTC`。
- `abnormal_reason`（写入 TDengine 时）：`status=online` → 空；`offline` 时按 `api_status` / `winrm_status` / 回调原文映射为中文：
  - **API异常，服务运行中** — `api_status=offline` 且 `winrm_status=online`（或 WinRM 可达、进程在跑）
  - **API异常，服务未运行** — 进程/服务未运行（原文含 Process not running 等），或仅 API 不可达且无法确认 WinRM
  - **网络连接异常（API、WinRM均不可达）** — `api_status` 与 `winrm_status` 均为 `offline`
  - 其它未归类 offline → 保留 playbook 英文 `abnormal_reason`；仍为空则 **设备异常**
- Semaphore DB / UI 仍保存 playbook 原始 `abnormal_reason`（英文）；仅 TDengine 列使用上述中文
- **`supplier`** 为超级表 **TAG**，固定 **`newarerm`**（Variable Group：`TDENGINE_TAG_SUPPLIER`）

## 每台设备只保留一行（ts 固定策略）

`ts` 固定写入：

```
2262-01-01T08:00:00.000000000+08:00
```

每次任务都用同一个 `ts` 写入；`updated_time` / `check_time` 仍写入本次真实时间，反映最后更新时刻。

## 覆盖的操作

凡包含 **`semaphore_bulk_put_from_hostvars.yml`** 的模板均会尝试写 TDengine（在配置了 `TDENGINE_URL` 时），含 **NBT**（`cursor-playbooks/nbt/device_*.yml`）：

- Patrol / `device_status.yml`
- `device_stop.yml` / `device_restart.yml`
- `device_redeploy.yml` / `device_check_restart.yml`

**不包含**：`device_discovery.yml`（发现不写设备状态 bulk）。

TCP 定时探针、RDP Probe **不**写 TDengine。

## check_restart 通道新鲜度（只读）

`device_check_restart.yml`（**NEWARE / SINEXCEL / LAND / NBT / JHAI**）判定顺序：

1. **批量一次**查询通道 `LAST(insert_time)`（按 hostname 匹配）
2. **新鲜（≤ `TDENGINE_CHANNEL_STALE_HOURS`）** → `healthy`，**不**跑 API / WinRM
3. **不新鲜 / 无行 / 查询失败 / 未配置** → 再跑各类型 **API** → **WinRM** → 必要时 restart

Variable Group 仍用 `TDENGINE_CHANNEL_STATUS_TABLE` + `TDENGINE_TAG_SUPPLIER`（各类型 supplier 不同，如 `sinexcel` / `newarerm` / `nbt` / `jhai`）。

```sql
SELECT LAST(`insert_time`), `computer_name`
FROM `lab_sync`.`dwd_channel_status`
WHERE supplier='…'
PARTITION BY computer_name;
```

| 变量 | 说明 |
|------|------|
| `TDENGINE_CHANNEL_STATUS_TABLE` | 必填才启用；如 `lab_sync.dwd_channel_status` |
| `TDENGINE_TAG_SUPPLIER` | `WHERE supplier=…`（与状态写入 TAG 共用） |
| `TDENGINE_CHANNEL_STALE_HOURS` | 默认 **6**；超过则不健康 / 继续 API+WinRM |
| `TDENGINE_URL` / 认证 / `TDENGINE_TIMEZONE` | 与写入相同 |

任务日志搜 **`[DEBUG-TDENGINE-CHANNEL]`**。未配置表名时跳过查询并走 API/WinRM。

## 调试

任务日志中搜索 **`[DEBUG-TDENGINE]`**。成功时 REST 仍可能 **HTTP 200**，请查看：

- `raw_body` / `json.code`（`0` 为成功）
- `json.desc`（失败时的错误说明）
- `json.affected_rows`（写入行数；为 `0` 时表内无变化）

请求 URL 为 `{TDENGINE_URL}/rest/sql/{TDENGINE_DATABASE}`（无状态连接，库名在 URL 中）。INSERT 语法（子表 + 超级表 + TAG）：

```sql
INSERT INTO `lab`.`neware_remote_computer_status`
USING `lab`.`dws_computer_status` TAGS('newarerm')
(ts, computer_name, ip_addr, status, updated_time, check_time, abnormal_reason)
VALUES ('2262-01-01 08:00:00.000', 'JSC1306XHXW018', '10.0.0.18', 'offline', '2026-05-27 10:07:51.000', '2026-05-27 10:07:51.000', 'WinRM ping failed');
```

变量可只写表名（自动加 `TDENGINE_DATABASE` 前缀）；也可写全名如 `lab.dws_computer_status`。

**注意**：若表主键仅 `ts` 且所有设备共用同一固定 `ts`，TDengine 中最终可能只保留 **一行**（后写入覆盖先写入）；需多行时请确认表结构是否以 `computer_name` 等区分主键/标签。
