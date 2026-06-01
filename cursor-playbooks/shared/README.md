# shared — cross–device-type playbooks

Tasks, helper scripts, and group vars used by **all** device profiles (`neware/`, `land/`, …).

## Layout

| Path | Purpose |
|------|---------|
| `tasks/` | WinRM connect/retry, Semaphore bulk/discovery callbacks, interactive EXE start, deploy helper scripts |
| `files/` | Generic `sem_*.ps1` (process alive, scheduled-task start, graceful stop + popup confirm) |
| `group_vars/windows_hosts.yml` | WinRM timeouts (copy or mirror under each profile dir for Ansible) |

## Required play vars

Each device-type playbook should set:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
sem_files_dir: "{{ playbook_dir }}/../shared/files"
```

SyncLims-style types (LAND/SINEXCEL/NBT) also use `tasks/resolve_exe_dir_windows.yml` + `files/sem_resolve_exe_dir_windows.ps1` to resolve `EXE_DIR` drive. You can configure fallback order from Variable Group with `EXE_DIR_FALLBACK_DRIVES` (examples: `D,E,C` or `E,C,D`). If unset, default is: preferred drive from `EXE_DIR`, then `E:`, then `C:`.

NEWARE (and any type with extra `sem_*.ps1` / TDengine) also sets:

```yaml
sem_profile_files_dir: "{{ playbook_dir }}/files"
sem_profile_tasks_dir: "{{ playbook_dir }}/tasks"
```

Root **`device_discovery.yml`** uses:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/shared/tasks"
```

## Do not put here

- NEWARE-only: log tail, BTSClient iconf, TDengine publish, `resolve_exe_dir_windows`, …
- LAND-only: SyncLims API bodies, LH registry, zip layout, …

## Debug task output (`[DEBUG-*]`)

Playbooks set `sem_debug_tag` (e.g. `LAND`, `SINEXCEL`, `NBT`) and include:

| Task file | Used by |
|-----------|---------|
| `debug_sync_api_patrol_snapshot.yml` | Status patrol (before/after API + callback) |
| `debug_sync_api_stop_snapshot.yml` | Stop (graceful script lines, process, optional API) |
| `debug_sync_api_start_snapshot.yml` | Start / restart / redeploy |
| `debug_sync_api_redeploy_gate_snapshot.yml` | `device_check_restart_redeploy` need_reconfigure gate |

NEWARE uses `neware/tasks/debug_patrol_snapshot.yml` and `debug_action_snapshot.yml` (`[DEBUG-NEWARE]`). Semaphore bulk PUT still logs `[DEBUG-API]` in `semaphore_bulk_put_from_hostvars.yml`.

## Graceful stop (`stop_program_close_main_window_confirm.yml`)

Used by `land/device_stop.yml` and `sinexcel/device_stop.yml`. Variable Group / play vars:

| Var / ENV | Default | Meaning |
|-----------|---------|---------|
| `STOP_GRACEFUL_PROCESS_NAME` | `LHBTS` | `Get-Process` + `CloseMainWindow` |
| `STOP_VERIFY_PROCESS_NAME` | same as `PROCESS_NAME` | Final running check |
| `STOP_POPUP_WAIT_SECONDS` | `2` | Sleep before sending Enter |
| `STOP_POPUP_KEYWORD` | `警告` | Visible window title substring |
| `STOP_FORCE_AFTER_GRACEFUL` | `true` | Force kill if still running |

Those stay under `neware/` or `land/` respectively.

**No compatibility shims** under `neware/tasks/` for shared files — include `{{ sem_tasks_dir }}/…` directly from each play. Missing file → add under `shared/` or the device profile; wrong path → fix the play.
