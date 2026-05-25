# cursor-playbooks

Local playbooks for device discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Variable Groups (environment variables)

Play-level defaults use **`lookup('env', 'VAR')`**: if the Variable Group / template environment sets **`VAR`** to a **non-empty** value (after trim), that value wins; otherwise the **hardcoded default** in the playbook is used.

**WinRM 偶发 UNREACHABLE**：所有 `hosts: windows_hosts` 的 play 在其它 `win_*` 任务之前执行 **`tasks/winrm_ensure_reachable.yml`**（`win_ping` 带 **`ignore_unreachable: true`** + `failed_when: false`；**连通判定**用 **`_winrm_ping_ok`**（`ping: pong` 且非 unreachable），**不能**用 `is succeeded`（`ignore_unreachable` 时仍会显示 task ok）。仅此时置 **`_winrm_session_ok`**；其后 **`winrm_gate_play_tasks.yml`** 防止误判后继续 deploy/巡检。成功则停止后续 ping 尝试；第 2 次起先 `reset_connection` 再 ping。**辅助脚本下发**：紧接建连后执行 **`tasks/deploy_sem_windows_helper_scripts.yml`**，将 `files/sem_*.ps1` **整目录一次**复制到 `C:\Windows\Temp\`（每台主机每 play 仅一次；`stop` / `collect` / `resolve` / `reconfig_start` 不再各自 `win_copy`）。长 play 内在重配流程、日志检查/轮询前可 **`include_tasks: tasks/winrm_refresh_midplay.yml`**（`winrm_force_reconnect: true`，重新 ping）。DEBUG 使用 **`_winrm_ping_last_success`**（避免 loop skip 覆盖 `last_ping`）。**`meta` 不能写 `when`**：`clear_host_errors` / `reset_connection` 放在 **block** 或单独 include 里。Variable Group：**`WINRM_CONNECT_RETRIES`**（默认 **4**）、**`WINRM_CONNECT_RETRY_DELAY`**（默认 **5** 秒）。若日志出现 **`ansible_winrm_read_timeout unsupported by pywinrm`**，属旧 pywinrm，可升级 runner 依赖或忽略（不影响 ping）。建连仍失败且 play 定义了 **`_semaphore_device_rows`** 时，先 **`clear_host_errors`**（**必须**：`win_ping` 的 UNREACHABLE 会使 Ansible 跳过该主机后续任务，否则 **`semaphore_callback_winrm_connect_failed`** 不会执行、UI 停在 **checking**），再执行该 task（**`device_status: unhealthy`**、`winrm_status`/`api_status` **offline**）并 **`end_host`**。每次 ping 未成功在 **`winrm_connect_one_attempt.yml`** 内也会 **`clear_host_errors`** 以便重试。中途 **`winrm_refresh_midplay`** 重连失败时同样走该路径。无 bulk 回调变量的 play 仅 **`clear_host_errors`**。

- **`device_discovery.yml`**: `NETWORK_SUBNET` (then Semaphore extra-vars `subnet`, `network_cidr` if env empty), `WINRM_PORT`, `RDP_PORT`, **`API_PORT`** (TCP scan for application API; default 9002), `WINRM_USER`, `WINRM_PASSWORD`, `SCAN_TIMEOUT_SECONDS`, `SCAN_WORKERS`.
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`check_restart_redeploy.yml`**: start/restart playbooks still resolve **`api_port`** for **INI `ReportApiSettings.ServerPort`** (device `api_port` → **`API_PORT`** → **9002**). **`device_status.yml` (Patrol)** does not call the app HTTP API. **`EXE_DIR`** → **`exe_dir_preferred`** (default **`D:\Program Files\NEWARE`**). First tasks: **`winrm_ensure_reachable`**, **`deploy_sem_windows_helper_scripts`**, **`resolve_exe_dir_windows`**. **Health gate** (process running): **`tasks/log_health_check_windows.yml`** + **`tasks/health_gate_need_reconfigure_from_log.yml`** — **`LOG_HEALTH_RECENT_MINUTES`** (default **8**) + **`LOG_SUCCESS_KEYWORD`**; **`check_restart_redeploy`** sets **`log_health_use_tail: true`** and **`LOG_HEALTH_TAIL_LINES`** (default **8000**).
- **`device_start.yml`**: the shared vars above plus `ZIP_PATH`, `EXE_ARGS`, `CONFIG_FILE_NAME`, `EXPORT_NOT_STARTED`, `EXPORT_STARTING`, `POLL_RETRIES` (default **3**), `POLL_DELAY` (default **10**), `HIS_DATA_FROM_TIME`, **`RESTART_DELAY`** (default **30** — total seconds to poll for the process after launch in **`tasks/reconfig_start_program_windows.yml`** and again in **`tasks/start_verify_poll_process_after_start.yml`**), **`PROCESS_VERIFY_POLL_SECONDS`** (default **5** — interval between polls). **`重配执行：启动程序`** only starts the EXE via an **Interactive** scheduled task as **`_reconfig_profile_user`** (desktop user with **explorer.exe**); it does **not** use the WinRM account or `Start-Process` in the automation session. `LOG_SUCCESS_KEYWORD`, `LOG_POLL_RETRIES`, `LOG_POLL_DELAY`. **Start-verify** (`tasks/start_verify_after_reconfig.yml`): no **启动/状态 HTTP API** in the verify block — **`final_start_ok`** requires **`VERIFY_OK`** from the start script, **process running** after **`start_verify_poll_process_after_start`**, and **log keyword** match via **`tasks/log_poll_confirm.yml`**. **Log baseline** (`tasks/log_poll_record_baseline_before_start_api.yml`): after **`log_poll_resolve_windows_path`**, **before `重配执行：启动程序`**, count lines into `C:\Windows\Temp\sem_logpoll_*.state`. Each poll only matches **`LOG_SUCCESS_KEYWORD` on lines after** `max(baseline, cursor)`; if the log shrinks, rescan from **baseline**. **Log read:** **`FileShare.ReadWrite`** with **`LOG_READ_RETRIES`** (default **5**) / **`LOG_READ_RETRY_DELAY`** (default **2** s). **Debug:** `[DEBUG-LOG]` at baseline; `[DEBUG-PROCESS-POLL]` / `[DEBUG-LOG-POLL]` during verify.
- **`device_restart.yml`**: same pattern as start where applicable (`EXPORT_STARTED`, `HIS_DATA_FROM_TIME`, **`RESTART_DELAY`**, **`PROCESS_VERIFY_POLL_SECONDS`**, log-related env vars), including **ZIP copy + `Expand-Archive`** when `{{ exe_path }}` is missing (`ZIP_PATH`, `ZIP_NAME`), then start when the EXE exists or extraction succeeded. **Start-verify** uses **`tasks/start_verify_after_reconfig.yml`** (process poll + log poll; no start/status API in verify).
- **`check_restart_redeploy.yml`**: extends **`device_restart.yml`** with the same **log-only pre-gate** as **`device_start`** (Tail + recent window when process is running). If **not** running → **`need_reconfigure: true`** immediately. Reconfigure + **`start_verify_after_reconfig`** match **`device_restart.yml`**.
- **`device_status.yml` (Patrol)**: process status + **log-only** health (no status `POST`). **`healthy`** when process running and recent log contains **`LOG_SUCCESS_KEYWORD`**; **`api_status: online`** when not **`need_reconfigure`** (same semantics as “upload path OK” proxy).
- **`device_stop.yml`**: `EXE_NAME`.

Callback task env: see table above (`SEMAPHORE_*`). `tasks/semaphore_bulk_put_from_hostvars.yml` uses the same **empty-env → default** rule for `SEMAPHORE_URL`.


These playbooks call `PUT /api/project/{id}/devices/status/bulk` when **`SEMAPHORE_API_TOKEN`** is set (template Environment / Variable Group, or controller env).

**Project id** is injected automatically as **`semaphore_project_id`** in the task’s extra-vars for Patrol all, scheduled status runs, and other flows that use `runDeviceTemplate`. You only need **`SEMAPHORE_PROJECT_ID`** in the environment if you run the playbook outside Semaphore without that extra-var.

| Variable | Description |
|----------|-------------|
| `semaphore_project_id` | Injected by Semaphore (extra-vars); used by `tasks/semaphore_bulk_put_from_hostvars.yml` |
| `SEMAPHORE_PROJECT_ID` | Optional fallback (numeric id) if not in extra-vars |
| `SEMAPHORE_API_TOKEN` | User API token (`Authorization: Bearer …`) — **required** for the callback to run |
| `SEMAPHORE_URL` | Optional; default `http://127.0.0.1:3000` (must reach Semaphore from the Ansible controller) |

- **`device_discovery.yml`** — **No** bulk callback: only prints a JSON array for the UI to parse; persistence is **Import selected** → API `discovery/import`.
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`device_stop.yml`**, **`check_restart_redeploy.yml`** — **`post_tasks`** 上用 **block**（`run_once` + `delegate_to: localhost`）再 **`include_tasks`** `semaphore_bulk_put_from_hostvars.yml`（**不能**把 `delegate_to` 写在 `include_tasks` 上；内含 WinRM ping 失败 **fallback**）。
- **Stop before reconfigure** (`tasks/stop_program_windows.yml`): if the stop script prints **`STOP_FAILED`** / **`STOP_ERROR`**, a follow-up **`Get-Process`** recheck runs; when the process is **not** running, the play **continues** (reconfigure/restart); only **`STILL_RUNNING`** on recheck fails the play.
- **Process “running” vs zombie**: `Get-Process` rows with **`Handles <= PROCESS_ALIVE_MIN_HANDLES`** (default **1**) or **`WorkingSet < PROCESS_ALIVE_MIN_WS_KB`** (default **512** KB) count as **not running** (`NOT_RUNNING` or `NOT_RUNNING|STALE_PID:...`). Used by `collect_process_status_windows.ps1`, stop/verify scripts, and reconfig start verify. Variable Group env: **`PROCESS_ALIVE_MIN_HANDLES`**, **`PROCESS_ALIVE_MIN_WS_KB`**.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.

### `semaphore_callback_row` (WinRM / API vs RDP)

- **`rdp_status`** is **not** written by patrol/start/restart/stop templates — use **`POST …/devices/{id}/probe`** (or discovery import) for **RDP TCP** in the UI. **`PUT …/devices/status/bulk`** skips empty fields, so omitting **`rdp_status`** keeps the last probe/import value.
- **`winrm_status`**: **`online`** when this template run successfully used WinRM through the callback path (e.g. post_tasks after the first `win_shell` was reachable; patrol sets **`offline`** if the **log-check** `win_shell` hit **`unreachable`**). **`offline`** when **WinRM ping 建连失败** (`semaphore_callback_winrm_connect_failed`) or **collect/stop** 等首段 `win_shell` **unreachable**。
- **`api_status`** (bulk callback): **Patrol / pre-gate / start-verify** no longer use HTTP status `POST` for this field. **`online`** when **`need_reconfigure` is false** (log keyword OK in recent window while process running) or **`final_start_ok`** after reconfigure; **`offline`** on failure or process not running. **`device_stop.yml`** still forces **`api_status: offline`**. **`api_port`** in INI is still written on reconfigure (app endpoint config), not probed for **`api_status`**.
- **`device_status` (callback)**: **`healthy`** when process running and **`need_reconfigure` is false**; after reconfigure, when **`final_start_ok`** is true.
- **Patrol / `device_status.yml` — log task unreachable:** **`检查日志上报状态`** uses **`ignore_unreachable: true`**. Without it, a WinRM disconnect on that task **fatal**s the host and **skips** setting **`semaphore_callback_row`**, so the localhost bulk play **omits** that host (`semaphore_bulk_put_from_hostvars.yml` only loops hosts with **`semaphore_callback_row` defined**) and the UI can stay **`checking`**. When unreachable, **`need_reconfigure`** is forced so the callback writes **unhealthy** / **`winrm_status: offline`** and a dedicated **`abnormal_reason`**.
- **Ansible boolean gotcha:** `set_fact: x: "{{ false }}"` stores the **string** `"False"`, which is **truthy** in Jinja `when:` tests — use two tasks with literal YAML `true` / `false` (as in those playbooks) for flags that gate `fail` / callbacks.
- `PUT …/devices/status/bulk` **persists** playbook fields as given. Patrol/start/restart align **`device_status`** and **`api_status`** on **process + log keyword** (no separate HTTP probe for **`api_status`**).

### Bulk vs single-device extra-vars

- **Bulk** actions pass `devices: [{ id, hostname, ip }, ...]` plus **`configs_by_host`**: the same per-device config map is keyed by **both** `ip` and `hostname` so playbooks can resolve it when `inventory_hostname` is the WinRM target IP.
- **Single-device** actions pass **`device: { id, hostname, ip }`** (no `devices` list). Playbooks build **`_semaphore_device_rows`** from either `devices` or `[device]`. Each **`semaphore_callback_row`** includes **`hostname`** and **`ip`** (device IP, defaulting to `inventory_hostname`). **`PUT …/devices/status/bulk`** matches the project device **first by `ip` when present**, then by **`hostname`**, then treats **`hostname` as an IP** if it equals a device’s **`ip_address`** — so inventory keyed by IP still updates the correct row when DB hostnames differ.

**Stop** actions intentionally report **`device_status: unhealthy`** when the process is stopped (service not running), with **`abnormal_reason`** describing unreachable vs stop result vs already-not-running. The bulk row sets **`api_status: offline`** unconditionally after stop (insurance); when WinRM was reachable, a **`POST`** to the app status URL runs first for **`[DEBUG-STOP-API]`** logs only. **`rdp_status`** is still omitted (use **Probe** for RDP TCP).

## Debug logging (always printed on success)

Playbooks emit **`ansible.builtin.debug`** and **`win_shell` stdout** lines so Semaphore logs stay useful even when tasks succeed:

| Prefix / marker | Meaning |
|-----------------|--------|
| `[DEBUG-重配-用户]` / `[DEBUG-重配-路径]` | Resolved profile user and INI paths after username detection (`device_start` / `device_restart`). **`config_user`** may differ from **`profile_user`**: if `C:\Users\<profile_user>` does not exist, config copy/modify uses **`RECONFIG_CONFIG_FALLBACK_USER`** (default **NEWARE**) via `tasks/reconfig_resolve_config_user_paths.yml`; interactive exe start still uses **`_reconfig_profile_user`**. |
| `RECONFIG_LOG_*` / `RECONFIG_MODIFY_*` | Same info from the Windows `win_shell` task stdout (visible without expanding `debug`) |
| `RECONFIG_REPORT_API_DEFAULTS` | Effective `ServerIpAddrStr` / `ServerPort` / `EnableReportApiCall` defaults computed from the device |
| `[DEBUG-WINRM]` | After **`tasks/winrm_ensure_reachable.yml`**: `session_ok`, retry count/delay |
| `[DEBUG-EXE_DIR]` | After **`tasks/resolve_exe_dir_windows.yml`**: Variable Group / preferred path vs drive-resolved **`exe_dir`** (`EXE_DIR_REQUESTED` / `EXE_DIR_CHOSEN` in `win_shell` stdout) |
| `[RECONFIG] …` | Ansible **`debug`** lines: effective **`api_port`**, **`exe_path`**, and post-probe **start/skip** diagnostics (`device_start` / `device_restart` / `check_restart_redeploy`) |
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
| `[DEBUG-STOP-API]` | After **`device_stop.yml`** WinRM succeeds: **`POST`** to app status URL (same **`api_port`** / **`API_STATUS_CALL_TYPE`** as Patrol); bulk still forces **`api_status: offline`** |
| `[DEBUG-API]` | HTTP status + **raw response body** for device HTTP API POSTs and for Semaphore `PUT …/devices/status/bulk` |

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

`device_start.yml` / `device_restart.yml` / **`check_restart_redeploy.yml`** populate `SystemConfig.ReportApiSettings.ServerIpAddrStr` (from `merged_system_config.ServerIpAddrStr` or the device IP `api_host`), **`SystemConfig.ReportApiSettings.ServerPort`**, and **`EnableReportApiCall: true`** before running the merge — so on every (re)start the device's app callback is pointed at the current endpoint **and** the callback is turned on. **`ServerPort` is always overwritten** with the playbook-resolved **`api_port`** (inventory **`api_port`** when \(>0\), else Variable Group **`API_PORT`**, else **9002**), including when the existing INI or merged JSON already had `ServerPort` empty, zero, or invalid — this matches Semaphore device / env defaults and avoids leaving the field unset. Old configs that pinned `SystemConfig.ServerIpAddrStr` / `SystemConfig.ServerPort` / `SystemConfig.EnableReportApiCall` at the **top level** still work (they are folded into `ReportApiSettings` and the flat keys are removed). For **`ServerIpAddrStr`** / **`EnableReportApiCall`**, override priorities remain: **device-config in DB → project default config → playbook defaults.** Explicit `false` from device or project config is respected for **`EnableReportApiCall`**.

### Start-verify (process + log, no HTTP start API)

**`tasks/start_verify_after_reconfig.yml`** runs after **`重配执行：启动程序`**:

1. **`tasks/start_verify_register_exe_start_ok.yml`** — **`_exe_start_script_ok`** from **`VERIFY_OK`** in **`relaunch_result`**.
2. **`tasks/start_verify_poll_process_after_start.yml`** — retries **`collect_process_status_windows`** for up to **`RESTART_DELAY`** seconds every **`PROCESS_VERIFY_POLL_SECONDS`**; sets **`check_result_after_start`** and **`skip_log_poll`** when process not running / unreachable / exe start not **`VERIFY_OK`**.
3. **`tasks/log_poll_assert_baseline_before_poll.yml`** + **`tasks/log_poll_confirm.yml`** when not **`skip_log_poll`**.
4. **`final_start_ok`** — all of: **`_exe_start_script_ok`**, **`process_running_after_start`**, log poll **`rc == 0`**.

Log baseline is recorded in **`tasks/log_poll_record_baseline_before_start_api.yml`** **before** start (not after exe launch).
