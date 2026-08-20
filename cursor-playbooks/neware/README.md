# NEWARE playbooks (`cursor-playbooks/neware/`)

Windows / NEWARE device playbooks: discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Variable Groups (environment variables)

Play-level defaults use **`lookup('env', 'VAR')`**: if the Variable Group / template environment sets **`VAR`** to a **non-empty** value (after trim), that value wins; otherwise the **hardcoded default** in the playbook is used.

**WinRM 偶发 UNREACHABLE**：所有 `hosts: windows_hosts` 的 play 在其它 `win_*` 任务之前执行 **`{{ sem_tasks_dir }}/winrm_ensure_reachable.yml`**（`sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"`）。共享逻辑仅在 **`../shared/`**，本目录 `tasks/` 仅 NEWARE 专用，无转发桩。（`win_ping` 带 **`ignore_unreachable: true`** + `failed_when: false`；**连通判定**用 **`_winrm_ping_ok`**（`ping: pong` 且非 unreachable），**不能**用 `is succeeded`（`ignore_unreachable` 时仍会显示 task ok）。仅此时置 **`_winrm_session_ok`**；其后 **`winrm_gate_play_tasks.yml`** 防止误判后继续 deploy/巡检。成功则停止后续 ping 尝试；第 2 次起先 `reset_connection` 再 ping。**辅助脚本下发**：紧接建连后执行 **`tasks/deploy_sem_windows_helper_scripts.yml`**，将 `files/sem_*.ps1` **整目录一次**复制到 `C:\Windows\Temp\`（每台主机每 play 仅一次；`stop` / `collect` / `resolve` / `reconfig_start` 不再各自 `win_copy`）。长 play 内在重配流程、日志检查/轮询前可 **`winrm_refresh_midplay.yml`**（仅 **`_winrm_session_ok`** 时；play 开头 ping 全失败不会跑第二遍 ensure）。**`WINRM_CONNECT_RETRIES`**（默认 **2**）时日志会出现「第 2/2 次」的 **`重试前重置与等待`** + **`WinRM ping 探测`**，属建连重试而非 deploy/巡检。DEBUG 使用 **`_winrm_ping_last_success`**（避免 loop skip 覆盖 `last_ping`）。**`meta` 不能写 `when`**：`clear_host_errors` / `reset_connection` 放在 **block** 或单独 include 里。可选环境变量（Variable Group **ENV** 或服务器 `config.json` → `env_vars`，**不是** JSON extra-vars）：**`WINRM_CONNECT_RETRIES`**（未设置时 playbook 默认 **2**）、**`WINRM_CONNECT_RETRY_DELAY`**（默认 **5** 秒）。**`device_start` / `device_restart` / `check_restart_redeploy`** 在日志检查或重配前还会跑 **`winrm_refresh_midplay`**（又一次完整 ensure，最多再 **2** 次 ping）。**`device_status`（巡检）** 仅在 play 开头 **`winrm_ensure_reachable`** 一轮，不在日志检查前 refresh。长 play 日志里最多可见 **4** 次「第 N/M 次」ping 时，**不等于** `WINRM_CONNECT_RETRIES=4`。看 **`[DEBUG-WINRM] attempts=`** 与 **`context=play-start|mid-play`** 区分。若日志出现 **`ansible_winrm_read_timeout unsupported by pywinrm`**，属旧 pywinrm，可升级 runner 依赖或忽略（不影响 ping）。建连仍失败且 play 定义了 **`_semaphore_device_rows`** 时，先 **`clear_host_errors`**（**必须**：`win_ping` 的 UNREACHABLE 会使 Ansible 跳过该主机后续任务，否则 **`semaphore_callback_winrm_connect_failed`** 不会执行、UI 停在 **checking**），再执行该 task（**`device_status: unhealthy`**、`winrm_status`/`api_status` **offline**）并 **`end_host`**。每次 ping 未成功在 **`winrm_connect_one_attempt.yml`** 内也会 **`clear_host_errors`** 以便重试。中途 **`winrm_refresh_midplay`** 重连失败时同样走该路径。无 bulk 回调变量的 play 仅 **`clear_host_errors`**。

- **Discovery** (project-level, not in this folder): **`../device_discovery.yml`** — device-type agnostic; API callback uses **`../shared/tasks/semaphore_discovery_put_results.yml`** (`sem_tasks_dir` on root play).
- **`device_status.yml`**, **`device_restart.yml`**, **`device_check_restart.yml`**: **`api_port`** for INI **`ReportApiSettings.ServerPort`** and for **BTSClient 数据上报状态 API** (device `api_port` → **`API_PORT`** → **9002**). **`EXE_DIR`** → **`exe_dir_preferred`** (default **`D:\Program Files\NEWARE`**). **Health gate** (process running): API **`ExecResultData==3`** (`EXPORT_STARTED`) → healthy; else fall back to **Kafka TCP** (`netstat -ano` to ports **`KAFKA_REMOTE_PORTS`** default **9092,9093,9094** — any **Established** ⇒ working; **do not** use unfiltered `Get-NetTCPConnection`, it can hang for hours). Helper: **`files/sem_check_kafka_tcp_windows.ps1`**.
- **`device_restart.yml`** / **`device_check_restart.yml`** / **`device_redeploy.yml`**: all set **`neware_include_his_data_resend: true`** and **write** `HisDataFromTime`/`HisDataToTime` during INI patch: from device/default `SystemConfig` if set, else env `HIS_DATA_FROM_TIME` / empty → **today** / **tomorrow** (`yyyy-MM-dd`). Restart also uses `ZIP_PATH`, `EXE_ARGS`, `CONFIG_FILE_NAME`, export poll vars, **`RESTART_DELAY`** (default **30**), **`PROCESS_VERIFY_POLL_SECONDS`** (default **5**).
- **`device_resend_data.yml`**: UI **Resend data** — same HisData patch, but dates come from UI **`resend_params`** (not today/tomorrow defaults).
- **`device_check_restart.yml`**: **TDengine channel freshness first** (batch `LAST(insert_time)` via `TDENGINE_CHANNEL_STATUS_TABLE` + `TDENGINE_TAG_SUPPLIER`). Age **≤ `TDENGINE_CHANNEL_STALE_HOURS` (6)** → **healthy**, skip API/WinRM/restart. Otherwise → **API** (`ExecResultData==3` → healthy) → **WinRM** / process / Kafka → restart (channel not-fresh still forces restart even if Kafka OK). Unconfigured/failed TDengine query → treat as not-fresh. HisData patched like ordinary restart when restart runs. **`sem_files_dir`** must be **`{{ playbook_dir }}/files`** — `sem_collect_process_status_windows.ps1` is **not** in `shared/files`; pointing at shared leaves `C:\Windows\Temp\` without that script (hosts that never ran Patrol fail with `-File … does not exist`). Collect uses **`failed_when: false`** so a missing script becomes an unhealthy callback instead of play `FAILED`.
- **`device_status.yml` (Patrol)**: process + **API-first upload health**; **`healthy`** / **`api_status: online`** when **`need_reconfigure` is false** (API **`ExecResultData==3`** or Kafka TCP **Established**).
- **停止/重启前停进程路径**: `EXE_NAME`（推荐 `uu.exe`；旧值 `uu` 仍兼容）。见 [`../VARIABLE_GROUPS.md`](../VARIABLE_GROUPS.md)。

Callback task env: see table above (`SEMAPHORE_*`). `tasks/semaphore_bulk_put_from_hostvars.yml` uses the same **empty-env → default** rule for `SEMAPHORE_URL`.


These playbooks call `PUT /api/project/{id}/devices/status/bulk` when **`SEMAPHORE_API_TOKEN`** is set (template Environment / Variable Group, or controller env).

**Project id** is injected automatically as **`semaphore_project_id`** in the task’s extra-vars for Patrol all, scheduled status runs, and other flows that use `runDeviceTemplate`. You only need **`SEMAPHORE_PROJECT_ID`** in the environment if you run the playbook outside Semaphore without that extra-var.

| Variable | Description |
|----------|-------------|
| `semaphore_project_id` | Injected by Semaphore (extra-vars); used by `tasks/semaphore_bulk_put_from_hostvars.yml` |
| `SEMAPHORE_PROJECT_ID` | Optional fallback (numeric id) if not in extra-vars |
| `SEMAPHORE_API_TOKEN` | User API token (`Authorization: Bearer …`) — **required** for bulk PUT. Use Variable Group **ENV** *or* **JSON** extra-var (playbook reads both); server may inject via `env_vars` / `SEMAPHORE_DEVICE_CALLBACK_API_TOKEN` in `config.json` |
| `SEMAPHORE_URL` | Optional; default `http://127.0.0.1:3000` (must reach Semaphore from the Ansible controller) |
| `TDENGINE_URL` | If set (Variable Group ENV), after bulk PUT writes this task’s rows to TDengine REST (`tasks/semaphore_tdengine_publish_from_bulk.yml`) |
| `TDENGINE_USER` / `TDENGINE_PASSWORD` | Optional Basic auth for TDengine REST |
| `TDENGINE_DATABASE` | Default `lab` |
| `TDENGINE_STATUS_TABLE` | Child table; default `neware_remote_computer_status` → `lab.neware_remote_computer_status` in SQL |
| `TDENGINE_SUPER_TABLE` | Super table; default `dws_computer_status` → `USING lab.dws_computer_status` |
| `TDENGINE_TAG_SUPPLIER` | Super-table TAG `supplier` (not a column); default `newarerm`. Also used by **check_restart** channel freshness `WHERE supplier=…` |
| `TDENGINE_TIMEZONE` | IANA tz for `updated_time` / `check_time` / channel age; default `Asia/Shanghai` |
| `TDENGINE_CHANNEL_STATUS_TABLE` | **check_restart** only: channel table for `LAST(insert_time)` (e.g. `lab_sync.dwd_channel_status`). Requires `TDENGINE_URL`. Batch query once per task |
| `TDENGINE_CHANNEL_STALE_HOURS` | **check_restart** (all types): age threshold hours; default **6**. Fresh → healthy skip API/WinRM; older/missing → continue API then WinRM |

- **`../device_discovery.yml`** — **`PUT /api/project/{id}/devices/discovery/results`** (via **`tasks/semaphore_discovery_put_results.yml`**) when **`SEMAPHORE_API_TOKEN`** is set (`task_id` + `devices` array). Rows are **upserted in DB by IP** (`project__device_discovery_host`). UI **`GET .../discovery/results`** (persisted list) or **`?task_id=`** after a scan. Import to inventory is **Import selected** → **`discovery/import`** (with **`device_profile_id`**).
- **`device_status.yml`**, **`device_restart.yml`**, **`device_check_restart.yml`**, **`device_redeploy.yml`**, **`device_resend_data.yml`** — play1 在任务或 **`post_tasks`** 登记每台 **`semaphore_callback_row`**；**bulk PUT 与状态总览仅在独立 `hosts: localhost` 第二 play**（`strategy: free` 下不可在 play1 `run_once`，避免第一台抢跑）。**已移除** **`semaphore_bulk_put_immediate.yml`** 的 include。`windows_hosts` play 使用 **`strategy: free`**，批量选中设备时一台卡住不会堵住其它台的 WinRM 任务。
- **第二 play bulk 主机列表**：`tasks/semaphore_resolve_bulk_credentials.yml` 在 `ansible_play_hosts_all` 仅为 `localhost` 时改用 **`groups['windows_hosts']`** 汇总回调行；否则会跳过 PUT、设备一直 **checking**（重启/启动验证失败时仍会在 post_tasks 登记 `semaphore_callback_row`）。
- **Stop before reconfigure** (`tasks/stop_program_windows.yml`): if the stop script prints **`STOP_FAILED`** / **`STOP_ERROR`**, a follow-up **`Get-Process`** recheck runs; when the process is **not** running, the play **continues** (reconfigure/restart); only **`STILL_RUNNING`** on recheck fails the play.
- **Process “running” vs zombie**: `Get-Process` rows with **`Handles <= PROCESS_ALIVE_MIN_HANDLES`** (default **1**) or **`WorkingSet < PROCESS_ALIVE_MIN_WS_KB`** (default **512** KB) count as **not running** (`NOT_RUNNING` or `NOT_RUNNING|STALE_PID:...`). Used by `collect_process_status_windows.ps1`, stop/verify scripts, and reconfig start verify. Variable Group env: **`PROCESS_ALIVE_MIN_HANDLES`**, **`PROCESS_ALIVE_MIN_WS_KB`**.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.

### `semaphore_callback_row` (WinRM / API vs RDP)

- **`rdp_status`** is **not** written by patrol/start/restart/stop templates — use **`POST …/devices/{id}/probe`** (or discovery import) for **RDP TCP** in the UI. **`PUT …/devices/status/bulk`** skips empty fields, so omitting **`rdp_status`** keeps the last probe/import value.
- **`winrm_status`**: **`online`** when this template run successfully used WinRM through the callback path (e.g. post_tasks after the first `win_shell` was reachable; patrol sets **`offline`** if the **log-check** `win_shell` hit **`unreachable`**). **`offline`** when **WinRM ping 建连失败** (`semaphore_callback_winrm_connect_failed`) or **collect/stop** 等首段 `win_shell` **unreachable**。
- **`api_status`** (bulk callback): **`online`** when **`need_reconfigure` is false** (BTSClient upload-status API **`ExecResultData==3`**, or Kafka TCP **Established** while process running) or **`final_start_ok`** after reconfigure; **`offline`** on failure or process not running. The legacy stop-only path forces **`api_status: offline`** (debug POST still runs). **`api_port`** is also written to INI **`ReportApiSettings.ServerPort`** on reconfigure.
- **`device_status` (callback)**: **`healthy`** when process running and **`need_reconfigure` is false**; after reconfigure, when **`final_start_ok`** is true.
- **Patrol / `device_status.yml` — log task unreachable:** **`检查日志上报状态`** uses **`ignore_unreachable: true`**. Without it, a WinRM disconnect on that task **fatal**s the host and **skips** setting **`semaphore_callback_row`**, so the localhost bulk play **omits** that host (`semaphore_bulk_put_from_hostvars.yml` only loops hosts with **`semaphore_callback_row` defined**) and the UI can stay **`checking`**. When unreachable, **`need_reconfigure`** is forced so the callback writes **unhealthy** / **`winrm_status: offline`** and a dedicated **`abnormal_reason`**.
- **Ansible boolean gotcha:** `set_fact: x: "{{ false }}"` stores the **string** `"False"`, which is **truthy** in Jinja `when:` tests — use two tasks with literal YAML `true` / `false` (as in those playbooks) for flags that gate `fail` / callbacks.
- `PUT …/devices/status/bulk` **persists** playbook fields as given. Patrol/start/restart align **`device_status`** and **`api_status`** on **process + upload-status API (preferred) / Kafka TCP Established fallback**.

### BTSClient 数据上报状态 API（本机 HTTP）

| Item | Value |
|------|--------|
| URL | `POST http://{device_ip}:{api_port}` (no path) |
| Body | `{ "CallType": 1 }` — **`API_STATUS_CALL_TYPE`** (default **1**) |
| Retries | **`API_STATUS_RETRIES`** (default **3**), **`API_STATUS_RETRY_DELAY`** (default **3** s) — patrol/health |
| Start-verify poll | **`API_STATUS_START_POLL_RETRIES`** / **`API_STATUS_START_POLL_DELAY`** (default = **`LOG_POLL_RETRIES`** / **`LOG_POLL_DELAY`**, else **8** / **10** s) — restart/redeploy until **`ExecResultData==3`** |
| Envelope OK | HTTP **200**, **`ResponeResultCode==0`**, **`ExecResultCode==0`** |
| **`ExecResultData`** | **1** = 程序已启动 · **2** = 启动数据上报中 · **3** = 数据上报已启动（健康，跳过 Kafka TCP） |

**Kafka TCP 回退**（API 未达 **3** 且进程在跑）：`netstat -ano`（带 **`KAFKA_TCP_TIMEOUT_SEC`** 默认 **12s** 上限），进程名默认 **`NWReport_DBWB`**（可用 **`KAFKA_PROCESS_NAME`** / 显式 **`EXE_NAME`**），远端端口 **`KAFKA_REMOTE_PORTS`**（默认 **9092,9093,9094**）；任一 **Established** / **已建立** → 健康。不要用无过滤的 **`Get-NetTCPConnection`**（连接多或 CIM 卡住时会挂数小时）。

Tasks: **`tasks/neware_query_upload_status_api.yml`**, **`tasks/kafka_health_check_windows.yml`**, **`tasks/health_gate_upload_status_api_then_log.yml`**. Logs: **`[DEBUG-NEWARE] upload_status_api`**, **`[Kafka检查]`**.

### Bulk vs single-device extra-vars

- **Bulk** actions pass `devices: [{ id, hostname, ip }, ...]` plus **`configs_by_host`**: the same per-device config map is keyed by **both** `ip` and `hostname` so playbooks can resolve it when `inventory_hostname` is the WinRM target IP.
- **Single-device** actions pass **`device: { id, hostname, ip }`** (no `devices` list). Playbooks build **`_semaphore_device_rows`** from either `devices` or `[device]`. Each **`semaphore_callback_row`** includes **`hostname`** and **`ip`** (device IP, defaulting to `inventory_hostname`). **`PUT …/devices/status/bulk`** matches the project device **first by `ip` when present**, then by **`hostname`**, then treats **`hostname` as an IP** if it equals a device’s **`ip_address`** — so inventory keyed by IP still updates the correct row when DB hostnames differ.

**Stop** actions intentionally report **`device_status: unhealthy`** when the process is stopped (service not running), with **`abnormal_reason`** describing unreachable vs stop result vs already-not-running. The bulk row sets **`api_status: offline`** unconditionally after stop (insurance); when WinRM was reachable, a **`POST`** to the app status URL runs first for **`[DEBUG-STOP-API]`** logs only. **`rdp_status`** is still omitted (use **Probe** for RDP TCP).

## Debug logging (always printed on success)

Playbooks emit **`ansible.builtin.debug`** and **`win_shell` stdout** lines so Semaphore logs stay useful even when tasks succeed:

| Prefix / marker | Meaning |
|-----------------|--------|
| `[DEBUG-重配-用户]` / `[DEBUG-重配-路径]` | Resolved in **one** `win_shell` (`tasks/reconfig_prepare_profile_user.yml`): fast `Win32_ComputerSystem.UserName`, else **first** `explorer.exe` only (not every explorer + `GetOwner`). **`config_user`** may differ from **`profile_user`** via **`RECONFIG_CONFIG_FALLBACK_USERS`** (default **`NEWARE,Administrator`**). Interactive start still uses **`_reconfig_profile_user`**. If logs used to stall at「检测 Profile 用户目录是否存在」, update playbooks — that task was removed (merged into prepare). |

**Variable Group — config path fallbacks when profile folder missing**

| ENV | Example | Meaning |
|-----|---------|---------|
| **`RECONFIG_CONFIG_FALLBACK_USERS`** | `NEWARE,Administrator` | Comma/semicolon-separated; try left-to-right, first existing `C:\Users\<name>` wins |
| **`RECONFIG_CONFIG_FALLBACK_USE`** | *(alias)* | Same as **`RECONFIG_CONFIG_FALLBACK_USERS`** (typo-tolerant alias) |
| **`RECONFIG_CONFIG_FALLBACK_USER`** | `NEWARE` | Legacy: used only if **`RECONFIG_CONFIG_FALLBACK_USERS`** / **`USE`** are empty (then defaults to `NEWARE,Administrator`) |

Add more local profile names as needed, e.g. `NEWARE,Administrator,Operator`.
| `RECONFIG_LOG_*` / `RECONFIG_MODIFY_*` | Same info from the Windows `win_shell` task stdout (visible without expanding `debug`) |
| `RECONFIG_REPORT_API_DEFAULTS` | Effective `ServerIpAddrStr` / `ServerPort` / `EnableReportApiCall` defaults computed from the device |
| `[DEBUG-WINRM]` | After **`tasks/winrm_ensure_reachable.yml`**: `session_ok`, retry count/delay |
| `[DEBUG-EXE_DIR]` | After **`tasks/resolve_exe_dir_windows.yml`**: Variable Group / preferred path vs drive-resolved **`exe_dir`** (`EXE_DIR_REQUESTED` / `EXE_DIR_CHOSEN` in `win_shell` stdout) |
| `[RECONFIG] …` | Ansible **`debug`** lines: effective **`api_port`**, **`exe_path`**, and post-probe **start/skip** diagnostics (`device_restart` / `check_restart_redeploy`) |
| `RECONFIG_START_EXE_PATH=…\|exists=…` | **`tasks/reconfig_start_program_windows.yml`**: **`exe_path`**, working directory, process name from the `.exe` file |
| `RECONFIG_TASK_INFO\|…` | Scheduled task **`LastTaskResult`** / **`LastRunTime`** (non-zero often means **Interactive** task could not run — profile user not logged on at the console) |
| `VERIFY_POLL\|…` | Process poll attempts during the **`RESTART_DELAY`** window |
| `RECONFIG_WINRM_RUN_AS=…` | WinRM/Ansible 连接账号（仅下发脚本）；**不会**用该账号 `Start-Process` 启动 EXE |
| `RECONFIG_INTERACTIVE_SESSION\|…` | 检测到 **`_reconfig_profile_user`** 的 **explorer.exe**（已登录桌面） |
| `INTERACTIVE_SESSION_REQUIRED\|…` | 无交互会话，跳过计划任务启动；需 **RDP 登录** profile 用户后再跑模板 |
| `VERIFY_OK\|method=scheduled_task_interactive` | 由 **Interactive** 计划任务在桌面用户下启动成功 |
| `VERIFY_FAILED\|…` | 计划任务未留下进程 — 查 **`scheduled_last_result`**（如 **0x800710E0**）、确认用户在线或增大 **`RESTART_DELAY`** |
| `CFG_CHANGE|<path>|[<section>]|…` | **Only printed when a value really changes.** Flat keys: `key: <old> -> <new>` or `+ key=<new>` for inserts. JSON keys: one line per changed sub-key, e.g. `ReportApiSettings.EnableReportApiCall: false -> true` |
| `CONFIG_MODIFIED` / `CONFIG_NOT_FOUND` / `CONFIG_MODIFY_ERROR` | Per-file outcome |
| `RECONFIG_CFG_RUN_SUMMARY` | `user_ok` / `public_ok` booleans after both files processed |
| `[DEBUG-NEWARE]` | Patrol (`debug_patrol_snapshot.yml`): process, log gate, callback row; start/stop/restart (`debug_action_snapshot.yml`): `final_start_ok`, `need_reconfigure`, stop stdout |
| `[DEBUG-NEWARE] stop_api` | After the legacy stop-only path succeeds over WinRM: **`POST`** to app status URL (same **`api_port`** / **`API_STATUS_CALL_TYPE`**); bulk still forces **`api_status: offline`** |
| `[DEBUG-API]` | HTTP status + **raw response body** for Semaphore `PUT …/devices/status/bulk` |

The `重配执行：配置修改变更明细（仅打印实际变化与摘要）` task also prints **`[RECONFIG]`** lines for **`api_port`** and **`exe_path`**, then `stdout_lines` / `stderr`. After **`重配执行：检查程序文件是否存在`**, **`重配执行：EXE 检查与启动条件（诊断）`** prints the raw exe probe plus booleans for whether **`EXE_FOUND` / `EXE_NOT_FOUND`** appear in a normalized text form (so **`重配执行：启动程序`** skip vs run is obvious in the log). The `win_shell` script no longer emits `PATH_OK` / `JSON_OK` / `PLAN_*` / `FILE_READ` noise, so what you see is exactly: device IP/port defaults, per-file `RECONFIG_MODIFY_*=…` paths, any `CFG_CHANGE|…` lines (only on real value changes), `CONFIG_MODIFIED|<path>`, and the per-file `RECONFIG_CFG_RUN_SUMMARY`. **Do not** rely on an Ansible-side `select('match', ...)` filter here: older Ansibles can swallow every line silently if the `match` test resolves differently.

Note: previously `$lines | Set-Content` could echo every line into PowerShell’s success stream and leave Ansible’s captured **`stdout`/`stdout_lines` empty** even when the file was modified (`changed`). Writes now use **`$null = … | Set-Content`** so logs stay intact.

**Merged config JSON into `win_shell`:** `merged_cfg_json` is injected as a **single-line** PowerShell assignment (`$cfgJson = '…'`) with **`replace("\u0027", "\u0027\u0027")`** so JSON values containing `'` stay valid in PowerShell **and** Ansible never sees unindented lines after Jinja (which breaks YAML `|` blocks). Do **not** use a multiline `@' … '@` here-string for this value: YAML requires every continuation line to stay indented, while PowerShell historically required the closing `'@` at the start of the script line—those rules conflict inside a playbook literal block.

**Windows PowerShell 5.1 compatibility:** the config update task does **not** use `ConvertFrom-Json -AsHashtable` (added in PowerShell 6). It calls a local **`ConvertTo-HT`** helper that recursively converts the `PSCustomObject` returned by `ConvertFrom-Json` into nested `Hashtable`s. When you add new code that walks the parsed config, keep using the helper instead of `-AsHashtable` so Windows 10/Server (which still ships PowerShell 5.1) keeps working.

**`ReportApiSettings` (and other `key=<json-object>` lines):** the BTSClient `.iconf` stores some keys as **one INI line whose value is a JSON object**, e.g.

```ini
[SystemConfig]
ReportApiSettings={"EnableReportApiCall":false,"ServerIpAddrStr":"","ServerPort":9002}
```

When `merged_cfg.SystemConfig.ReportApiSettings` is a **dict** (or any other section/key whose value is an object), the playbook calls **`Merge-JsonKey`** instead of the flat `Upsert-SectionKey`:

1. parse the existing JSON value from the line in the file,
2. **deep-merge** the updates on top (so `EnableReportApiCall` and other unrelated fields stay),
3. write back a **single-line** compact JSON value (`ConvertTo-Json -Compress -Depth 10`).

`device_restart.yml` / **`device_check_restart.yml`** populate `SystemConfig.ReportApiSettings.ServerIpAddrStr` (from `merged_system_config.ServerIpAddrStr` or the device IP `api_host`), **`SystemConfig.ReportApiSettings.ServerPort`**, and **`EnableReportApiCall: true`** before running the merge — so on every (re)start the device's app callback is pointed at the current endpoint **and** the callback is turned on. **`ServerPort` is always overwritten** with the playbook-resolved **`api_port`** (inventory **`api_port`** when \(>0\), else Variable Group **`API_PORT`**, else **9002**), including when the existing INI or merged JSON already had `ServerPort` empty, zero, or invalid — this matches Semaphore device / env defaults and avoids leaving the field unset. Old configs that pinned `SystemConfig.ServerIpAddrStr` / `SystemConfig.ServerPort` / `SystemConfig.EnableReportApiCall` at the **top level** still work (they are folded into `ReportApiSettings` and the flat keys are removed). For **`ServerIpAddrStr`** / **`EnableReportApiCall`**, override priorities remain: **device-config in DB → project default config → playbook defaults.** Explicit `false` from device or project config is respected for **`EnableReportApiCall`**.

### Start-verify (process + upload-status API + log fallback)

**`tasks/start_verify_after_reconfig.yml`** runs after **`重配执行：启动程序`**:

1. **`tasks/start_verify_register_exe_start_ok.yml`** — **`_exe_start_script_ok`** from **`VERIFY_OK`**.
2. **`tasks/start_verify_poll_process_after_start.yml`** — process poll; may set **`skip_log_poll`** on process/exe failure.
3. **`tasks/neware_query_upload_status_api_start_poll.yml`** — upload-status POST **polled** until **`ExecResultData==3`** (defaults align with **`LOG_POLL_RETRIES`** / **`LOG_POLL_DELAY`**; override **`API_STATUS_START_POLL_RETRIES`** / **`API_STATUS_START_POLL_DELAY`**). Patrol/health still uses **`neware_query_upload_status_api.yml`** (short retries).
4. **`tasks/log_poll_confirm.yml`** when API did not return **3**.
5. **`final_start_ok`** — **`_exe_start_script_ok`** + (**`api_upload_started`** via localhost API poll, **independent of WinRM**) or (**`process_running_after_start`** + log poll **`rc == 0`**). **`winrm_status`** reflects post-start WinRM probe (`_start_verify_winrm_ok`), not API health.

Log baseline is recorded in **`tasks/log_poll_record_baseline_before_start_api.yml`** **before** start (not after exe launch).
