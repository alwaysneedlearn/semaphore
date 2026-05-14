# cursor-playbooks

Local playbooks for device discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Semaphore API callback (optional)

These playbooks call `PUT /api/project/{id}/devices/status/bulk` when **`SEMAPHORE_API_TOKEN`** is set (template Environment / Variable Group, or controller env).

**Project id** is injected automatically as **`semaphore_project_id`** in the task’s extra-vars for Patrol all, scheduled status runs, and other flows that use `runDeviceTemplate`. You only need **`SEMAPHORE_PROJECT_ID`** in the environment if you run the playbook outside Semaphore without that extra-var.

| Variable | Description |
|----------|-------------|
| `semaphore_project_id` | Injected by Semaphore (extra-vars); used by `tasks/semaphore_bulk_put_from_hostvars.yml` |
| `SEMAPHORE_PROJECT_ID` | Optional fallback (numeric id) if not in extra-vars |
| `SEMAPHORE_API_TOKEN` | User API token (`Authorization: Bearer …`) — **required** for the callback to run |
| `SEMAPHORE_URL` | Optional; default `http://127.0.0.1:3000` (must reach Semaphore from the Ansible controller) |

- **`device_discovery.yml`** — **No** bulk callback: only prints a JSON array for the UI to parse; persistence is **Import selected** → API `discovery/import`.
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`device_stop.yml`** — Second play on `localhost` runs `tasks/semaphore_bulk_put_from_hostvars.yml` using per-host `semaphore_callback_row`.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.

### `semaphore_callback_row` and API status

- **`device_start.yml` / `device_restart.yml`** set **`api_status`** (`online` / `offline`) from the same app-API check used for start verification, so a failed restart does not leave **`device_status: healthy`** while the API is down.
- **Ansible boolean gotcha:** `set_fact: x: "{{ false }}"` stores the **string** `"False"`, which is **truthy** in Jinja `when:` tests — use two tasks with literal YAML `true` / `false` (as in those playbooks) for flags that gate `fail` / callbacks.
- The Semaphore API also **rejects inconsistent rows**: `PUT …/devices/status/bulk` coerces **`device_status` away from `healthy` when `api_status` is `offline`** (after merging optional `api_status` from the payload with the stored device).

### Bulk vs single-device extra-vars

- **Bulk** actions pass `devices: [{ id, hostname, ip }, ...]` plus **`configs_by_host`**: the same per-device config map is keyed by **both** `ip` and `hostname` so playbooks can resolve it when `inventory_hostname` is the WinRM target IP.
- **Single-device** actions pass **`device: { id, hostname, ip }`** (no `devices` list). Playbooks build **`_semaphore_device_rows`** from either `devices` or `[device]` so **`hostname` in the bulk status API** matches the DB and callbacks are not dropped.

**Stop** actions intentionally report **`device_status: unhealthy`** when the process is stopped (service not running), with **`abnormal_reason`** describing unreachable vs stop result vs already-not-running.

## Debug logging (always printed on success)

Playbooks emit **`ansible.builtin.debug`** and **`win_shell` stdout** lines so Semaphore logs stay useful even when tasks succeed:

| Prefix / marker | Meaning |
|-----------------|--------|
| `[DEBUG-重配-用户]` / `[DEBUG-重配-路径]` | Resolved profile user and INI paths after username detection (`device_start` / `device_restart`) |
| `RECONFIG_LOG_*` / `RECONFIG_MODIFY_*` | Same info from the Windows `win_shell` task stdout (visible without expanding `debug`) |
| `RECONFIG_CFG_DEBUG|…` | Path exists, JSON section count, file line count before write |
| `RECONFIG_CFG_ADJUST|…` | Each upsert: **NEW_SECTION** / **REPLACE** (old line → new) / **INSERT_NEW_LINE** |
| `RECONFIG_CFG_RUN_SUMMARY` | `user_ok` / `public_ok` booleans after both files processed |
| `[DEBUG-API]` | HTTP status + **raw response body** for device HTTP API POSTs and for Semaphore `PUT …/devices/status/bulk` |

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

`device_start.yml` / `device_restart.yml` populate `SystemConfig.ReportApiSettings.ServerIpAddrStr` (from `merged_system_config.ServerIpAddrStr` or the device IP `api_host`), `SystemConfig.ReportApiSettings.ServerPort` (from `api_port`, default `9002`), and **`EnableReportApiCall: true`** before running the merge — so on every (re)start the device's app callback is pointed at the current endpoint **and** the callback is turned on. Old configs that pinned `SystemConfig.ServerIpAddrStr` / `SystemConfig.ServerPort` / `SystemConfig.EnableReportApiCall` at the **top level** still work (they are folded into `ReportApiSettings` and the flat keys are removed). Override priorities for each field: **device-config in DB → project default config → playbook defaults (device IP / `api_port` / `true`).** Explicit `false` from device or project config is respected.

### Start API retry (`device_start.yml` / `device_restart.yml`)

**「启动验证：调用启动API」** uses **`until` / `retries` / `delay`** until **`uri` returns HTTP 200** (e.g. service still starting; **`http_status: -1`** usually means connect/timeout from the controller to `http://device:port`).

| Env (Variable Group) | Play default | Meaning |
|----------------------|--------------|--------|
| `API_START_RETRIES` | `8` | Ansible **`retries`** (number of *re*tries after the first attempt; **total tries = retries + 1**) |
| `API_START_RETRY_DELAY` | `4` | Seconds between attempts |

The DEBUG line **`attempts=`** shows how many tries ran; logs will show **`FAILED - RETRYING`** between attempts when status is not 200 yet.
