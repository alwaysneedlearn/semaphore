# cursor-playbooks

Ansible playbooks for **Semaphore UI** device actions (discovery, patrol, stop/restart/redeploy). Layout: **shared** common layer + **one directory per device type**.

## Layout

| Path | Purpose |
|------|---------|
| **`shared/`** | WinRM, bulk/discovery callbacks, generic helper scripts — see [`shared/README.md`](shared/README.md) |
| **`neware/`** | NEWARE playbooks (`device_*.yml`), NEWARE-only `tasks/` + `files/` |
| **`land/`** | LAND playbooks and LAND-only logic |
| **`sinexcel/`** | SINEXCEL (LAND lifecycle + HTTP API config; no INI) |
| **`nbt/`** | NBT (Windows service + heartbeat API) |
| **`jhai/`** | JHAI (`UploaderServiceDaemon` + BTS upload HTTP API) |
| **`dahua/`** | DAHUA（`device_status.yml`：CTSMonPro 进程；`device_resend_data.yml`：lims-hist） |
| **`lanh/`** | LANH：`status` + `resend`；`check_restart` 仅 Schedule（只启动不停止） |
| **`landv7/`** | LANDV7：与 LANH 功能一致（`status` + `resend`；Schedule `check_restart`） |
| **`device_discovery.yml`** | Network scan + API callback (**device-type agnostic**, repo root) |

**Semaphore templates** examples:

- `cursor-playbooks/neware/device_status.yml`
- `cursor-playbooks/land/device_status.yml`
- `cursor-playbooks/sinexcel/device_status.yml`
- `cursor-playbooks/nbt/device_status.yml`
- `cursor-playbooks/jhai/device_status.yml`
- `cursor-playbooks/device_discovery.yml`

Every type playbook sets **`sem_tasks_dir`** to shared tasks. **`sem_files_dir`** depends on whether the profile has its own helper scripts under `<type>/files/`:

| Profile | `sem_files_dir` | Notes |
|---------|-----------------|-------|
| NEWARE, LAND, SINEXCEL, NBT, JHAI | `{{ playbook_dir }}/files` | Type dir includes profile-only `sem_*.ps1` (and usually copies of shared helpers). |
| LANH, LANDV7, DAHUA | `{{ playbook_dir }}/../shared/files` | No local `files/`; only shared start/resolve scripts are needed. |

Also set **`sem_profile_tasks_dir: "{{ playbook_dir }}/tasks"`** when the play includes type-specific tasks. See [`shared/README.md`](shared/README.md#required-play-vars).

**Batch selected devices:** all `hosts: windows_hosts` plays use **`strategy: free`** so one slow/hung host does not block others on each task (Ansible default `linear` is lockstep). Concurrent hosts are capped by **`forks`** (200 in [`ansible.cfg`](ansible.cfg); override with env **`ANSIBLE_FORKS`** or template CLI `--forks`). The localhost bulk-PUT play still waits until every host finishes play 1. Do not add `run_once` on the Windows play.

**Task logs:** search for `[DEBUG-LAND]`, `[DEBUG-SINEXCEL]`, `[DEBUG-NBT]`, `[DEBUG-JHAI]`, `[DEBUG-NEWARE]`, `[DEBUG-DAHUA]`, `[DEBUG-LANH]`, `[DEBUG-LANDV7]`, or `[DEBUG-API]` (bulk callback). See [`shared/README.md`](shared/README.md#debug-task-output-debug-).

## Documentation

- **Variable Groups (all device types):** [`VARIABLE_GROUPS.md`](VARIABLE_GROUPS.md)
- **NEWARE (detailed):** [`neware/README.md`](neware/README.md)
- **LAND:** [`land/README.md`](land/README.md)
- **SINEXCEL:** [`sinexcel/README.md`](sinexcel/README.md)
- **NBT:** [`nbt/README.md`](nbt/README.md)
- **JHAI:** [`jhai/README.md`](jhai/README.md)
- **DAHUA:** [`dahua/README.md`](dahua/README.md)
- **LANH:** [`lanh/README.md`](lanh/README.md)
- **LANDV7:** [`landv7/README.md`](landv7/README.md)
- **Shared tasks:** [`shared/README.md`](shared/README.md)
- **Cloud agent:** `AGENTS.md` (Devices section)

## Adding a new device type

1. Create `cursor-playbooks/<profile_key>/` with: `device_status.yml`, `device_restart.yml`, `device_redeploy.yml`, `device_check_restart.yml`.
2. In each play `vars`, set `sem_tasks_dir` and `sem_files_dir` per the table above (see [`shared/README.md`](shared/README.md#required-play-vars)).
3. Put profile-specific tasks under `<profile_key>/tasks/`, scripts under `<profile_key>/files/`.
4. Copy `shared/group_vars/windows_hosts.yml` to `<profile_key>/group_vars/` (Ansible loads vars next to the playbook).
5. Bind templates in Semaphore **Device types** to `cursor-playbooks/<profile_key>/device_*.yml`.
6. Discovery stays at `cursor-playbooks/device_discovery.yml`.
