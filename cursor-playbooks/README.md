# cursor-playbooks

Local playbooks for device discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Variable Groups (environment variables)

Play-level defaults use **`lookup('env', 'VAR')`**: if the Variable Group / template environment sets **`VAR`** to a **non-empty** value (after trim), that value wins; otherwise the **hardcoded default** in the playbook is used.

- **`device_discovery.yml`**: `NETWORK_SUBNET` (then Semaphore extra-vars `subnet`, `network_cidr` if env empty), `WINRM_PORT`, `RDP_PORT`, **`API_PORT`** (TCP scan for application API; default 9002), `WINRM_USER`, `WINRM_PASSWORD`, `SCAN_TIMEOUT_SECONDS`, `SCAN_WORKERS`.
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`check_restart_redeploy.yml`**: each resolves the HTTP **`api_port`** as **device inventory/extra-var `api_port` first** (Semaphore injects it from the device record when set), then Variable Group **`API_PORT`**, then default **9002**. The play **`vars`** use an explicit **`{% set hp %}…{% if (hp|int)>0 %}…{% else %}…{% endif %}`** one-liner (not `| ternary`) so **`api_port` cannot accidentally become a boolean** across Jinja/Ansible versions (which would print URLs like `http://host:False` and write **`ServerPort=0`**). Shared vars include `EXE_NAME`, `EXE_DIR`, `ZIP_NAME`, `API_STATUS_CALL_TYPE`, `API_TIMEOUT_SECONDS`, `API_EXPECTED_RESPONSE_CODE`, `API_EXPECTED_EXEC_SUCCESS_CODE`, `EXPORT_STARTED`. **`device_status.yml`** and **`device_start.yml`** (状态判断 / 日志状态检查) also honor **`LOG_HEALTH_RECENT_MINUTES`** (default **8**): only log lines whose timestamp is within that many minutes are scanned for the upload-success keyword.
- **`device_start.yml`**: the shared vars above plus `ZIP_PATH`, `EXE_ARGS`, `CONFIG_FILE_NAME`, `API_START_CALL_TYPE`, `EXPORT_NOT_STARTED`, `EXPORT_STARTING`, `POLL_RETRIES`, `POLL_DELAY`, `HIS_DATA_FROM_TIME`, `RESTART_DELAY`, `LOG_SUCCESS_KEYWORD`, `LOG_POLL_RETRIES`, `LOG_POLL_DELAY`, **`LOG_POLL_DEBUG_MAX_LINES`** (default `100`, `0` = no cap on debug line dump per poll attempt), **`API_START_TIMEOUT_SECONDS`** (per-request timeout for the start API call; defaults to `API_TIMEOUT_SECONDS`), **`API_START_RETRIES`** / **`API_START_RETRY_DELAY`** (`until` retries on the “启动验证：调用启动API” `uri` task; defaults `5` / `3`). **Start-verify log polling** (`tasks/log_poll_*.yml`): before **`重配执行：启动程序`**, the play resolves **`log_path`** on the **Windows host** (`Get-Date -Format yyyyMMdd`, not the Ansible controller clock), counts existing lines, and writes that **baseline** to `C:\Windows\Temp\sem_logpoll_*.state`. Each `until` attempt only matches **`LOG_SUCCESS_KEYWORD` on lines after** `max(baseline, previous cursor)`; if the log shrinks (rotate/truncate), it rescans from **baseline**, never from line 0. **Trade-off:** each attempt still reads the file sequentially to count lines. **Debug:** `[DEBUG-LOG]` after exe-start baseline; `[DEBUG-LOG-POLL]` lists each poll attempt with `LOG_POLL_ATTEMPT|…` metadata and `LOG_POLL_LINE|…` for new lines scanned that attempt (capped by **`LOG_POLL_DEBUG_MAX_LINES`**).
- **`device_restart.yml`**: same pattern as start where applicable (`EXPORT_STARTED`, `HIS_DATA_FROM_TIME`, `RESTART_DELAY`, log-related env vars, **`API_START_TIMEOUT_SECONDS`**, **`API_START_RETRIES`**, **`API_START_RETRY_DELAY`**, etc.), including **ZIP copy + `Expand-Archive`** when `{{ exe_path }}` is missing (`ZIP_PATH`, `ZIP_NAME`), then start when the EXE exists or extraction succeeded. **Start-verify log polling** uses the same **`tasks/log_poll_*.yml`** flow as **`device_start.yml`**.
- **`check_restart_redeploy.yml`**: extends **`device_restart.yml`** with a **pre-gate**: when the process is already running, it calls the status API and scans **only the tail** of the latest `NWReport_DBWB*.log` within a **recent time window** before deciding **`need_reconfigure`**. If the process is **not** running, it sets **`need_reconfigure: true`** immediately. The reconfigure + start-verify blocks match **`device_restart.yml`** (username resolution, `ReportApiSettings` merge, `api_status` callback, optional **zip copy + Expand-Archive** when the EXE is missing). **Start-verify log polling** uses the same **`tasks/log_poll_*.yml`** flow. Extra env vars: **`LOG_HEALTH_TAIL_LINES`** (default `8000`), **`LOG_HEALTH_RECENT_MINUTES`** (default `8`), plus **`EXPORT_NOT_STARTED`** / **`EXPORT_STARTING`** for parity with **`device_start.yml`**.
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
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`device_stop.yml`**, **`check_restart_redeploy.yml`** — Second play on `localhost` runs `tasks/semaphore_bulk_put_from_hostvars.yml` using per-host `semaphore_callback_row`.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.

### `semaphore_callback_row` (WinRM / API vs RDP)

- **`rdp_status`** is **not** written by patrol/start/restart/stop templates — use **`POST …/devices/{id}/probe`** (or discovery import) for **RDP TCP** in the UI. **`PUT …/devices/status/bulk`** skips empty fields, so omitting **`rdp_status`** keeps the last probe/import value.
- **`winrm_status`**: **`online`** when this template run successfully used WinRM through the callback path (e.g. post_tasks after the first `win_shell` was reachable; patrol sets **`offline`** if the **log-check** `win_shell` hit **`unreachable`**). **`offline`** on the first-task unreachable rows.
- **`api_status`** (playbook bulk callback): **`device_status.yml` (Patrol)**: **`NORMAL`** requires the status `POST` to return **HTTP `200`** plus **(API business OK and export started) OR log keyword** — log alone cannot make **`NORMAL`** if HTTP ≠ `200`. Callback: **`online`** when **`need_reconfigure` is false** or status `POST` is **HTTP `200`** while still abnormal; else **`offline`**. **`device_start` / `device_restart` / `check_restart_redeploy`**: **`online`** if **`api_check_after` or `api_start_result`** has status `200`; **`offline`** if at least one `uri` ran and neither is `200`; **empty** if no `uri` ran. **`device_stop.yml`** forces **`api_status: offline`** after stop.
- **`device_status`** (start/restart / `check_restart_redeploy` callback field): follow the same **dual verification** outcome — **`healthy`** when **`final_start_ok`** is true (even if the post-start status **`POST`** did not yet show **`ExecResultData == export_started`**). Do **not** mark **`unhealthy`** solely because **`api_final_ok`** is false when **`final_start_ok`** is true, or the UI stays unhealthy despite a successful log line match.
- **Patrol / `device_status.yml` — log task unreachable:** **`检查日志上报状态`** uses **`ignore_unreachable: true`**. Without it, a WinRM disconnect on that task **fatal**s the host and **skips** setting **`semaphore_callback_row`**, so the localhost bulk play **omits** that host (`semaphore_bulk_put_from_hostvars.yml` only loops hosts with **`semaphore_callback_row` defined**) and the UI can stay **`checking`**. When unreachable, **`need_reconfigure`** is forced so the callback writes **unhealthy** / **`winrm_status: offline`** and a dedicated **`abnormal_reason`**.
- **Ansible boolean gotcha:** `set_fact: x: "{{ false }}"` stores the **string** `"False"`, which is **truthy** in Jinja `when:` tests — use two tasks with literal YAML `true` / `false` (as in those playbooks) for flags that gate `fail` / callbacks.
- `PUT …/devices/status/bulk` **persists** playbook fields as given: **`device_status` is not auto-downgraded** when **`api_status`** is **`offline`** (e.g. Patrol: log keyword OK but HTTP status `POST` ≠ 200 → script **NORMAL** matches UI **healthy** while **API** column can stay **offline**). **`device_stop.yml`** still sends **`api_status: offline`** with **`device_status: unhealthy`** as intended.

### Bulk vs single-device extra-vars

- **Bulk** actions pass `devices: [{ id, hostname, ip }, ...]` plus **`configs_by_host`**: the same per-device config map is keyed by **both** `ip` and `hostname` so playbooks can resolve it when `inventory_hostname` is the WinRM target IP.
- **Single-device** actions pass **`device: { id, hostname, ip }`** (no `devices` list). Playbooks build **`_semaphore_device_rows`** from either `devices` or `[device]`. Each **`semaphore_callback_row`** includes **`hostname`** and **`ip`** (device IP, defaulting to `inventory_hostname`). **`PUT …/devices/status/bulk`** matches the project device **first by `ip` when present**, then by **`hostname`**, then treats **`hostname` as an IP** if it equals a device’s **`ip_address`** — so inventory keyed by IP still updates the correct row when DB hostnames differ.

**Stop** actions intentionally report **`device_status: unhealthy`** when the process is stopped (service not running), with **`abnormal_reason`** describing unreachable vs stop result vs already-not-running. The bulk row sets **`api_status: offline`** unconditionally after stop (insurance); when WinRM was reachable, a **`POST`** to the app status URL runs first for **`[DEBUG-STOP-API]`** logs only. **`rdp_status`** is still omitted (use **Probe** for RDP TCP).

## Debug logging (always printed on success)

Playbooks emit **`ansible.builtin.debug`** and **`win_shell` stdout** lines so Semaphore logs stay useful even when tasks succeed:

| Prefix / marker | Meaning |
|-----------------|--------|
| `[DEBUG-重配-用户]` / `[DEBUG-重配-路径]` | Resolved profile user and INI paths after username detection (`device_start` / `device_restart`) |
| `RECONFIG_LOG_*` / `RECONFIG_MODIFY_*` | Same info from the Windows `win_shell` task stdout (visible without expanding `debug`) |
| `RECONFIG_REPORT_API_DEFAULTS` | Effective `ServerIpAddrStr` / `ServerPort` / `EnableReportApiCall` defaults computed from the device |
| `[RECONFIG] …` | Ansible **`debug`** lines: effective **`api_port`**, **`exe_path`**, and post-probe **start/skip** diagnostics (`device_start` / `device_restart` / `check_restart_redeploy`) |
| `RECONFIG_START_EXE_PATH=…\|exists=…` | First line of **`重配执行：启动程序`** `win_shell`: literal **`exe_path`** and **`Test-Path -LiteralPath`** before scheduled-task start |
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

### Start API retry (`device_start.yml` / `device_restart.yml`)

**「启动验证：调用启动API」** uses **`until` / `retries` / `delay`** until **`uri` returns HTTP 200** (e.g. service still starting; **`http_status: -1`** usually means connect/timeout from the controller to `http://device:port`).

| Env (Variable Group) | Play default | Meaning |
|----------------------|--------------|--------|
| `API_START_RETRIES` | `8` | Ansible **`retries`** (number of *re*tries after the first attempt; **total tries = retries + 1**) |
| `API_START_RETRY_DELAY` | `4` | Seconds between attempts |

The DEBUG line **`attempts=`** shows how many tries ran; logs will show **`FAILED - RETRYING`** between attempts when status is not 200 yet.
