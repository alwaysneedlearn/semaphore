# cursor-playbooks

Local playbooks for device discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Variable Groups (environment variables)

Play-level defaults use **`lookup('env', 'VAR')`**: if the Variable Group / template environment sets **`VAR`** to a **non-empty** value (after trim), that value wins; otherwise the **hardcoded default** in the playbook is used.

- **`device_discovery.yml`**: `NETWORK_SUBNET` (then Semaphore extra-vars `subnet`, `network_cidr` if env empty), `WINRM_PORT`, `RDP_PORT`, `WINRM_USER`, `WINRM_PASSWORD`, `SCAN_TIMEOUT_SECONDS`, `SCAN_WORKERS`.
- **`device_status.yml`**: `EXE_NAME`, `EXE_DIR`, `ZIP_NAME`, `API_PORT`, `API_STATUS_CALL_TYPE`, `API_TIMEOUT_SECONDS`, `API_EXPECTED_RESPONSE_CODE`, `API_EXPECTED_EXEC_SUCCESS_CODE`, `EXPORT_STARTED`.
- **`device_start.yml`**: the status vars plus `ZIP_PATH`, `EXE_ARGS`, `CONFIG_FILE_NAME`, `API_START_CALL_TYPE`, `EXPORT_NOT_STARTED`, `EXPORT_STARTING`, `POLL_RETRIES`, `POLL_DELAY`, `HIS_DATA_FROM_TIME`, `RESTART_DELAY`, `LOG_SUCCESS_KEYWORD`, `LOG_TAIL_LINES`, `LOG_POLL_RETRIES`, `LOG_POLL_DELAY`, **`API_START_TIMEOUT_SECONDS`** (per-request timeout for the start API call; defaults to `API_TIMEOUT_SECONDS`), **`API_START_RETRIES`** / **`API_START_RETRY_DELAY`** (`until` retries on the “启动验证：调用启动API” `uri` task; defaults `5` / `3`).
- **`device_restart.yml`**: same pattern as start where applicable (`EXPORT_STARTED`, `HIS_DATA_FROM_TIME`, `RESTART_DELAY`, log-related env vars, **`API_START_TIMEOUT_SECONDS`**, **`API_START_RETRIES`**, **`API_START_RETRY_DELAY`**, etc.).
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
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`device_stop.yml`** — Second play on `localhost` runs `tasks/semaphore_bulk_put_from_hostvars.yml` using per-host `semaphore_callback_row`.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.

### Bulk vs single-device extra-vars

- **Bulk** actions pass `devices: [{ id, hostname, ip }, ...]` plus **`configs_by_host`**: the same per-device config map is keyed by **both** `ip` and `hostname` so playbooks can resolve it when `inventory_hostname` is the WinRM target IP.
- **Single-device** actions pass **`device: { id, hostname, ip }`** (no `devices` list). Playbooks build **`_semaphore_device_rows`** from either `devices` or `[device]` so **`hostname` in the bulk status API** matches the DB and callbacks are not dropped.

**Stop** actions intentionally report **`device_status: unhealthy`** when the process is stopped (service not running), with **`abnormal_reason`** describing unreachable vs stop result vs already-not-running.
