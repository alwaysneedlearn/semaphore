# cursor-playbooks

Local playbooks for device discovery, status patrol, start/stop/restart. **Not required to be committed** if you use them only on your machine; this repo copy includes Semaphore **status bulk callbacks**.

## Semaphore API callback (optional)

These playbooks call `PUT /api/project/{id}/devices/status/bulk` when **`SEMAPHORE_PROJECT_ID`** and **`SEMAPHORE_API_TOKEN`** are set (template Environment / Variable Group, or controller env).

| Variable | Description |
|----------|-------------|
| `SEMAPHORE_PROJECT_ID` | Numeric project id |
| `SEMAPHORE_API_TOKEN` | User API token (`Authorization: Bearer …`) |
| `SEMAPHORE_URL` | Optional; default `http://127.0.0.1:3000` (must reach Semaphore from the Ansible controller) |

- **`device_discovery.yml`** — **No** bulk callback: only prints a JSON array for the UI to parse; persistence is **Import selected** → API `discovery/import`.
- **`device_status.yml`**, **`device_start.yml`**, **`device_restart.yml`**, **`device_stop.yml`** — Second play on `localhost` runs `tasks/semaphore_bulk_put_from_hostvars.yml` using per-host `semaphore_callback_row`.

Patrol all sets devices to `checking` first; the status template should run a playbook like **`device_status.yml`** so the callback clears `checking` to healthy/unhealthy.
