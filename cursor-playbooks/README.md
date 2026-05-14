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

Note: piping `$lines | Set-Content` used to echo every line into PowerShell’s success stream and could leave Ansible’s captured **`stdout`/`stdout_lines` empty** even when the file was modified (`changed`). Writes now use **`$null = … | Set-Content`** so logs stay intact.
| `[DEBUG-API]` | HTTP status + **raw response body** for device HTTP API POSTs and for Semaphore `PUT …/devices/status/bulk` |
