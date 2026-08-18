# Agents

## Cursor Cloud specific instructions

### Overview

Semaphore UI is a Go backend + Vue.js 2 frontend served as a single binary. The build system uses [go-task](https://taskfile.dev/) (`Taskfile.yml`), not Make.

**会话交接 / 新 Cloud Agent 上手**：见 [`docs/cloud-agent-handover.md`](docs/cloud-agent-handover.md)（本仓库相对原始 Semaphore 的改造总览、设备类型差异、Runner 注意事项）。

### Prerequisites (already installed by update script)

- **Go 1.24.6** — backend
- **Node.js 22+** — frontend build
- **go-task** — build orchestration (`task` command)

### Key commands

All commands use `task` (go-task). See `Taskfile.yml` for the full list.

| Action | Command |
|--------|---------|
| Device action templates | **Devices → Device types** only (project-level device settings UI removed; use profile settings API) |
| Install all deps | `task deps` |
| Build frontend + backend | `task build` |
| Build frontend only | `task build:fe` |
| Build backend only | `task build:be` |
| Run frontend lint | `task lint:fe` |
| Run Go lint (requires golangci-lint) | `task lint:be` |
| Run backend tests | `task test:be` or `go test ./...` |
| Run frontend tests | `task test:fe` |

### Devices UI routes

- **Device list**: `/project/{id}/devices/list` (sidebar **Devices** redirects here)
- **Discovery** (project-level discover template + import with device type): `/project/{id}/devices/discovery`. Scan results persist in **`project__device_discovery_host`** (upsert by **`project_id` + `ip_address`**). Playbook **`PUT .../devices/discovery/results`**; UI loads **`GET .../discovery/results`** (all hosts) or **`?task_id=`** (hosts from that run). Requires **`SEMAPHORE_API_TOKEN`** and **`semaphore_task_id`** in task extra-vars. Use `cursor-playbooks/device_discovery.yml` (device-type agnostic).
- **Device types**: dialog from list toolbar (**Device types**); templates/inventory per type. **WinRM connection defaults** (windows_hosts inventory): project-only via `GET/PUT /api/project/{id}/devices/settings/connection`.

### Running the server

1. Ensure a `config.json` exists in the workspace root (see `.devcontainer/config.json` for an example). For local dev with SQLite:
   ```json
   {
     "sqlite": { "host": "/workspace/database.sqlite" },
     "dialect": "sqlite",
     "cookie_hash": "5WJjXCLpvf3Cn5t+C/IV9F0asZUQLakOhCT+eSdIwP0=",
     "cookie_encryption": "6x6mmQWGn6YcsHN1rN0HiQjhYA+7HukcbCxUGHuT2CE="
   }
   ```
2. Create an admin user (if DB is fresh):
   ```
   ./bin/semaphore user add --admin --login admin --name Admin --email admin@example.com --password changeme --config ./config.json
   ```
3. Start the server:
   ```
   ./bin/semaphore server --config ./config.json
   ```
   The UI is available at http://localhost:3000 (login: `admin` / `changeme`).

### Gotchas

- `$HOME/go/bin` must be on `PATH` for `task` to work. The update script installs `task` there.
- Frontend lint (`task lint:fe`) has pre-existing errors in the repository; these are not caused by the dev environment setup.
- `task lint:be` requires `golangci-lint` and `swagger` CLI tools which are not installed by default; use `go vet ./...` as a lighter alternative for backend static analysis.
- The `go.mod` has a `replace` directive pointing `pro` module to `./pro`; this is normal for the open-source build.
- SQLite is the simplest DB dialect for local dev — no external database service needed.
- New SQL migrations under `db/sql/migrations/` must also be appended to `commonScripts` in `db/Migration.go`; otherwise `./bin/semaphore migrate` will never apply them.
- After changing Go code, you must rebuild the binary (`task build:be`) before restarting the server. The frontend has a separate build (`task build:fe`).
- **Task log:** Opening a running/finished task (Devices, History, Templates, snackbar) opens **`/project/{id}/history/{taskId}` in a new browser tab**. The current page is not covered by a drawer. Legacy `?t=` URLs redirect to that path.
- **Agents (Cursor Cloud / automation):** After **any Vue/HTML change**, you must run `task build:fe`, then `task build:be` (or equivalent `go build` for `./cli`), then restart `./bin/semaphore ...`, because UI assets are embedded in the binary. If the UI still looks stale in the browser under test, hard refresh (`Ctrl+Shift+R`). This sequence is **operational guidance for agents**, not end-user documentation — do not phrase it in summaries as if instructing the repository owner to “deploy” unless they asked for deployment steps.
- **Device actions**: When a task includes `inventory_id` (e.g. temporary inventory from the Devices UI), the runner uses that inventory first. Short-lived inventories are named like `windows_hosts batch <timestamp>` and are removed automatically when the task finishes (skipped if the task is re-queued waiting for a runner).
- **Auto inventory (per device type)**: Each **device profile** gets its own auto inventory (`windows_hosts (auto: <type name>)`, `is_device_default_auto` + `device_profile_id`). Sync runs on device create/update/delete/import and when creating a device type — **not** before Patrol all. Content is only devices assigned to that type. Project `default_inventory_id` is for **discovery** (e.g. empty inventory) and is **not** overwritten by auto sync. Profile `default_inventory_id` is set to that type’s auto inventory when unset (or still pointing at another auto inv). On server start, `MigrateDeviceProfileAutoInventories` demotes legacy project-wide `windows_hosts (auto)` rows (renamed `… (legacy)`), clears profile defaults that pointed at them, and syncs all type autos (DB migrations `v2.18.19`–`v2.18.20`).
- **Device list API** (`GET /api/project/{id}/devices`): returns JSON `{ "devices": [...], "total": N }` with query params `limit`, `offset`, `sort`, `order`, and optional filters `hostname`, `ip`, `device_status`, `rdp_status`, `winrm_status`, `api_status` (substring match for hostname/ip, exact for status fields).
- **Bulk device actions** enqueue extra-vars `configs_by_host` (config map keyed by **both** device IP and hostname) so Ansible `inventory_hostname` (usually IP) resolves merged config; single-device actions use `device` plus `config` — playbooks under `cursor-playbooks/<device_type>/` (e.g. `neware/`) normalize both via `_semaphore_device_rows`. Actions: `stop`, `restart`, **`redeploy`**, `status` (no separate **start** — use **restart** or **redeploy**). **`hosts: windows_hosts` plays use `strategy: free`** so a hung host does not lockstep the rest of the selection (`forks` default 50 in `cursor-playbooks/ansible.cfg`). Bulk PUT still runs after all hosts finish play 1.
- **Device operation split**: `device_redeploy` checks the target package first and only copies when missing, then applies config+start. **Check-restart is not bound on Device types** (no device-list button); run `device_check_restart.yml` via Semaphore **Schedules**.
- **WinRM ping**：`ignore_unreachable: true`；**`_winrm_ping_ok`**（`ping: pong`）才置 **`_winrm_session_ok`**；勿用 `is succeeded`。建连失败后 **`winrm_gate_play_tasks`** + unhealthy bulk。
- **编写/审查 cursor-playbooks**：遵循 **`.claude/skills/semaphore-cursor-playbooks/SKILL.md`**（Ansible TaskInclude 限制、UNREACHABLE 回调、bulk post_tasks、**`set_fact` 同任务内不可交叉引用键** 等）。速查：**`.claude/skills/semaphore-cursor-playbooks/references/ansible-errors-cheatsheet.md`**。
- **Discovery import**: Each row has its own **device type** in the table; **Import selected** sends `imports: [{ip_address, device_profile_id}]`. Successfully imported IPs are removed from **`project__device_discovery_host`** (discovery list), not from the device list.
- **Device discovery callback**: Playbook `cursor-playbooks/device_discovery.yml` needs **`SEMAPHORE_API_TOKEN`** (template Variable Group / JSON extra-var, or server `config.json` → `env_vars.SEMAPHORE_DEVICE_CALLBACK_API_TOKEN`). Without it, task logs show `discovery_token_set=False` and the discovery table stays empty until log sync (`SEMAPHORE_DISCOVERY_JSON=` in task output) or re-scan with token configured. Imported devices appear under **Devices → List** only after **Import selected** on the discovery tab.
- **Patrol all**: Marks every device `checking`, then runs the configured **Status** template.
- **TDengine**: No server-side sync. After bulk PUT, **`shared/tasks/semaphore_tdengine_publish_from_bulk.yml`** runs for all device types that use **`semaphore_bulk_put_from_hostvars.yml`** (NEWARE/NBT/LAND/SINEXCEL) when the template Variable Group sets **`TDENGINE_URL`** (optional **`TDENGINE_USER`**, **`TDENGINE_PASSWORD`**, **`TDENGINE_DATABASE`**, **`TDENGINE_STATUS_TABLE`**, **`TDENGINE_TAG_SUPPLIER`**, **`TDENGINE_TIMEZONE`** default `Asia/Shanghai`). See `docs/tdengine-setup.md`. Not discovery.
- Playbooks under `cursor-playbooks/neware/` (e.g. `device_status.yml`) bulk-callback when the template has **`SEMAPHORE_API_TOKEN`** (and **`SEMAPHORE_URL`** if the controller is not co-located with Semaphore). **`semaphore_project_id`** is injected into task extra-vars by the API; unreachable hosts must still get a callback row — see `neware/device_status.yml` / `cursor-playbooks/README.md`. Callback payloads include **`ip`** for bulk matching; **`winrm_status`** reflects **this run’s** WinRM outcome; **`device_status.yml`** sets **`api_status: online`** when **`need_reconfigure` is false** (same rule as **`NORMAL`**), not HTTP-only; **`rdp_status`** is omitted so **Probe** stays authoritative for RDP TCP. **`PUT …/devices/status/bulk` saves `device_status` from the playbook** (no server downgrade when **`api_status`** is **`offline`**).
- **Aggregate `device_status`:** `db.DeviceStatusFromChannelProbes` uses **WinRM + API** only. **`rdp_status` is informational** — RDP `offline` does **not** downgrade `device_status` from `healthy` when WinRM and API indicate the host is fine.
- **Device ports**: Each device has **`rdp_port`** (default 3389), **`ansible_port`** (WinRM HTTP default 5985, overridable per device and via project defaults), and **`api_port`** (default **9002** for TCP probes and HTTP calls in playbooks). **`rdp_status`** in the DB is normally from **Probe** or **discovery import** (not from patrol/start/restart callbacks). **`winrm_status`** reflects whether WinRM commands in that template run succeeded; **`api_status`** from those templates is **`online`/`offline` from HTTP status on the `uri` tasks** (see `cursor-playbooks/neware/README.md`; start/restart may **omit** `api_status` when no `uri` ran). The **Probe** action still updates **all three** TCP columns plus `last_updated`. Generated `windows_hosts` inventory includes **`api_port`** when the device has a valid port; playbooks resolve HTTP **`api_port`** as **inventory host var → Variable Group `API_PORT` → 9002**. Task extra-vars for `devices` arrays include `api_port` alongside `rdp_port` and `ansible_port`. Discovery **import** clears port fields to zero before upsert so normalized defaults do not overwrite stored per-device ports; new rows still get defaults in `CreateDevice`.
- **`cursor-playbooks/` defaults**: Playbook literals are overridden by **non-empty** Variable Group / template environment variables (`lookup('env', …)` with trim + `default(…, true)`). Discovery subnet order is **env `NETWORK_SUBNET` → extra-vars `subnet` / `network_cidr` → default CIDR**. The **Discover** API requires **`subnet`** (or `network_cidr`) in the JSON body (CIDR or single IP). See `cursor-playbooks/README.md` for the env name list.
- **Git branches (device work):** Use a **single** long-lived feature branch for ongoing device UI/API/playbook changes. Avoid opening extra `cursor/temp-*` (or parallel scratch) branches unless you must split unrelated PRs; merge related work into that branch instead of starting new temporaries.
- **Playbook sync (Semaphore):** Runner logs reflect whatever Git revision the project’s repository checkout uses (often under `/root/playbook/`). If new task names never appear between existing ones, the project is still on an old commit or a forked copy of the playbooks. Device start/restart print reconfigure paths as **`RECONFIG_LOG_*`** lines in the **stdout** of **重配执行：获取当前用户名** (last stdout line remains the bare profile username for Ansible) and **`RECONFIG_MODIFY_*`** at the start of **重配执行：修改配置文件**; expand the task result in the UI if lines are collapsed.
- **Mid-play UNREACHABLE callback fallback:** `tasks/semaphore_callback_winrm_fallback_missing_rows.yml` backfills any host that still lacks `semaphore_callback_row` during localhost bulk phase (not only ping-stage failures). This covers disconnects during helper-script deploy / early `win_*` tasks so devices do not remain `checking`, and TDengine write still receives rows.
