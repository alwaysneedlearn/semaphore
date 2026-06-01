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
| `STOP_TASK_TIMEOUT_SECONDS` | `45` | Interactive scheduled-stop wait timeout |

The graceful stop task runs in the logged-in desktop user's interactive session (scheduled task), not the WinRM service session.

**Restart / redeploy:** `land/device_restart.yml`, `land/device_check_restart_redeploy.yml` (and SINEXCEL equivalents) include `stop_program_graceful_before_reconfig.yml` before ModifyConfig + start — same behavior as `device_stop.yml`, not `Stop-Process -Force`.

**LAND ModifyConfig:** flat JSON via `land_flatten_merged_cfg.yml`; two-phase `land_api_modify_config_before_stop.yml` (while `RUNNING`) + `land_api_modify_config_after_start_retry.yml` (after start if before failed). Do not call API while process is stopped.

### Troubleshooting graceful stop (LAND / SINEXCEL)

| Log line | Meaning | What to check on the target |
|----------|---------|------------------------------|
| `INTERACTIVE_STOP_TASK\|user=pc\|explorer_pid=…` | Scheduled task started as desktop user `pc` | User must be logged in (explorer running for that user) |
| `STOP_TASK_NO_OUTPUT\|…\|log=C:\Windows\Temp\sem_stop_out_….log` | No log file within timeout | On host: Task Scheduler → task history; open the `log=` path; confirm `C:\Windows\Temp\sem_stop_close_main_window_confirm.ps1` exists (play must run `deploy_sem_windows_helper_scripts` first) |
| `STOP_TASK_LOG_INCOMPLETE\|…` | Log read before helper wrote final line (older wrappers broke when log file was merely created) | Fixed in current `sem_stop_close_main_window_confirm_interactive.ps1` — poll until `NOT_RUNNING`/`STILL_RUNNING`/`FORCE_STOPPED`/`ALREADY_STOPPED` in log |
| `STOP_TASK_LAST_RESULT\|code=…` | Windows task exit code (`0` = success) | Non-zero → helper failed before writing log |
| `STOP_HELPER_MISSING` | Helper not deployed to `C:\Windows\Temp` | Re-run stop play from start (includes deploy task) |
| `POPUP_NOT_FOUND` | No window title contained `STOP_POPUP_KEYWORD` (default `警告`) | Confirm dialog title; restore/minimize: script now restores main window and retries hidden/minimized popups |
| `CLOSE_MAIN_WINDOW_SENT\|count=0` | `CloseMainWindow` did not apply | Main window minimized/no handle: script restores before close; try foreground desktop |
| `[DEBUG-*] after_stop=RUNNING` | Process still alive after stop | If `STOP_FORCE_AFTER_GRACEFUL=true`, helper should `FORCE_STOPPED`; else WinRM fallback task runs |

**Does minimizing the main window matter?** Yes, sometimes. `CloseMainWindow()` is most reliable when the main window is restored; a minimized confirmation dialog may not match the first popup scan (only visible titles). The helper restores the process main window before close and, on the second pass, matches popups even when minimized/hidden.

**Manual checks on `10.33.37.157` (example):**

```powershell
Get-Process LHBTS | Select Id,MainWindowTitle,Responding
Get-ScheduledTask | Where TaskName -like 'StopGraceful*'
Get-ChildItem C:\Windows\Temp\sem_stop_out_*.log | Sort LastWriteTime -Desc | Select -First 1 | Get-Content
```

Increase wait: Variable Group `STOP_TASK_TIMEOUT_SECONDS=90`, `STOP_POPUP_WAIT_SECONDS=5`.

**No compatibility shims** under `neware/tasks/` for shared files — include `{{ sem_tasks_dir }}/…` directly from each play. Missing file → add under `shared/` or the device profile; wrong path → fix the play.
