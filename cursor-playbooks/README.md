# cursor-playbooks

Ansible playbooks for **Semaphore UI** device actions (discovery, patrol, start/stop/restart). Layout: **shared** common layer + **one directory per device type**.

## Layout

| Path | Purpose |
|------|---------|
| **`shared/`** | WinRM, bulk/discovery callbacks, generic helper scripts — see [`shared/README.md`](shared/README.md) |
| **`neware/`** | NEWARE playbooks (`device_*.yml`), NEWARE-only `tasks/` + `files/` |
| **`land/`** | LAND playbooks and LAND-only logic |
| **`sinexcel/`** | SINEXCEL (LAND lifecycle + HTTP API config; no INI) |
| **`nbt/`** | NBT (same pattern as sinexcel) |
| **`device_discovery.yml`** | Network scan + API callback (**device-type agnostic**, repo root) |

**Semaphore templates** examples:

- `cursor-playbooks/neware/device_status.yml`
- `cursor-playbooks/land/device_status.yml`
- `cursor-playbooks/sinexcel/device_status.yml`
- `cursor-playbooks/nbt/device_status.yml`
- `cursor-playbooks/device_discovery.yml`

Each type playbook sets `sem_tasks_dir` / `sem_files_dir` to `../shared/…`. NEWARE also sets `sem_profile_tasks_dir` / `sem_profile_files_dir` for type-specific tasks (e.g. TDengine).

**Task logs:** search for `[DEBUG-LAND]`, `[DEBUG-SINEXCEL]`, `[DEBUG-NBT]`, `[DEBUG-NEWARE]`, or `[DEBUG-API]` (bulk callback). See [`shared/README.md`](shared/README.md#debug-task-output-debug-).

## Documentation

- **Variable Groups (all device types):** [`VARIABLE_GROUPS.md`](VARIABLE_GROUPS.md)
- **NEWARE (detailed):** [`neware/README.md`](neware/README.md)
- **LAND:** [`land/README.md`](land/README.md)
- **SINEXCEL:** [`sinexcel/README.md`](sinexcel/README.md)
- **NBT:** [`nbt/README.md`](nbt/README.md)
- **Shared tasks:** [`shared/README.md`](shared/README.md)
- **Cloud agent:** `AGENTS.md` (Devices section)

## Adding a new device type

1. Create `cursor-playbooks/<profile_key>/` with: `device_status.yml`, `device_start.yml`, `device_stop.yml`, `device_restart.yml`, `device_redeploy.yml`, `device_check_restart.yml`.
2. In each play `vars`, set `sem_tasks_dir` and `sem_files_dir` (see `shared/README.md`).
3. Put profile-specific tasks under `<profile_key>/tasks/`, scripts under `<profile_key>/files/`.
4. Copy `shared/group_vars/windows_hosts.yml` to `<profile_key>/group_vars/` (Ansible loads vars next to the playbook).
5. Bind templates in Semaphore **Device types** to `cursor-playbooks/<profile_key>/device_*.yml`.
6. Discovery stays at `cursor-playbooks/device_discovery.yml`.
