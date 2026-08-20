# shared — cross–device-type playbooks

Tasks, helper scripts, and group vars used by **all** device profiles (`neware/`, `land/`, …).

## Layout

| Path | Purpose |
|------|---------|
| `tasks/` | WinRM connect/retry, Semaphore bulk/discovery callbacks, interactive EXE start, deploy helper scripts |
| `files/` | Generic `sem_*.ps1` (process alive, scheduled-task start, start/stop popup confirm) |
| `group_vars/windows_hosts.yml` | WinRM timeouts (copy or mirror under each profile dir for Ansible) |

## Ansible `set_fact` gotcha

In a **single** `set_fact` task (without `loop`), one key **must not** reference another key defined in the **same** task — Ansible evaluates all values before writing facts (`_app_from_cfg is undefined`). Safe patterns:

- Split into two tasks, or use a prior **snapshot** task (see `prepare_device_exe_paths.yml` → `_play_*` facts).
- `loop` + `var: "{{ var | default([]) + [item] }}"` is fine (each iteration sees the previous value).

CI helper: `python3 cursor-playbooks/scripts/check_set_fact_self_refs.py`

## Required play vars

Each device-type playbook should set:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
```

**`sem_files_dir`** — source directory copied to `C:\Windows\Temp\` by `deploy_sem_windows_helper_scripts.yml`:

| When | Set `sem_files_dir` to |
|------|-------------------------|
| Profile has `<type>/files/` with type-specific `sem_*.ps1` (NEWARE, LAND, SINEXCEL, NBT, JHAI) | `{{ playbook_dir }}/files` |
| Profile has **no** local `files/` and only needs shared helpers (LANH, DAHUA) | `{{ playbook_dir }}/../shared/files` |

If **`sem_files_dir` is omitted**, deploy defaults to **`{{ playbook_dir }}/files`** (relative to the playbook’s directory). Do **not** point at `shared/files` when the play runs tasks that call scripts that exist **only** under the type’s `files/` (e.g. NEWARE `sem_collect_process_status_windows.ps1` — hosts that never ran Patrol then fail with `-File … does not exist`).

Types with extra tasks also set:

```yaml
sem_profile_tasks_dir: "{{ playbook_dir }}/tasks"
```

SyncLims-style types (LAND/SINEXCEL/NBT) use `tasks/prepare_device_exe_paths.yml` + `tasks/resolve_exe_dir_windows.yml` + `files/sem_resolve_exe_dir_windows.ps1`. **SINEXCEL**：变量组 `EXE_DIR` + `EXE_DIR_FALLBACK_DRIVES` + `EXE_SCAN_LATEST` + `EXE_NAME` 即可在各盘符浅层扫描最新 exe，**不必**逐台 `Install.ExeDir`（`Install` 仅作可选覆盖，见 `prepare_device_exe_paths.yml`）。**LAND** / 固定布局类型：脚本在各盘符探测 `app_dir\EXE_NAME`；`EXE_DIR=D:\` / `F:\` 为盘根，`EXE_DIR=D:` → `D:\`。LAND zip：`tasks/land_prepare_zip_vars.yml`（`ZIP_NAME` 无 `.zip` 后缀）。

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
| `debug_sync_api_patrol_snapshot.yml` | Status patrol — single consolidated `[DEBUG-*]` task (call once after callback row) |
| `debug_sync_api_stop_snapshot.yml` | Stop (graceful script lines, process, optional API) |
| `debug_sync_api_start_snapshot.yml` | Start / restart / redeploy |
| `debug_sync_api_redeploy_gate_snapshot.yml` | `device_check_restart_redeploy` need_reconfigure gate |

NEWARE uses `neware/tasks/debug_patrol_snapshot.yml` and `debug_action_snapshot.yml` (`[DEBUG-NEWARE]`). Semaphore bulk PUT still logs `[DEBUG-API]` in `semaphore_bulk_put_from_hostvars.yml`.

## Windows helper scripts (`deploy_sem_windows_helper_scripts.yml`)

Copies **`{{ sem_files_dir }}/`** (whole directory) to `C:\Windows\Temp\` on the target. Play var **`sem_files_dir`** must list every `sem_*.ps1` the play will invoke via `-File C:\Windows\Temp\…`. If unset, deploy defaults to **`{{ playbook_dir }}/files`**. `resolve_exe_dir_windows.yml` also includes deploy when `sem_tasks_dir` is set.

## Graceful stop (`stop_program_close_main_window_confirm.yml`)

Used by the interactive graceful-stop path (including LAND/SINEXCEL restart and redeploy via `stop_program_graceful_before_reconfig.yml`). Task log should show **`[DEBUG-STOP] STOP_GRACEFUL_CONFIRM_REV=4`** — if missing, the runner playbook copy is stale (sync `develop` / refresh template repo).

Variable Group / play vars:

| Var / ENV | Default | Meaning |
|-----------|---------|---------|
| `STOP_GRACEFUL_PROCESS_NAME` | `LHBTS` | `Get-Process` + `CloseMainWindow` |
| `STOP_VERIFY_PROCESS_NAME` | same as `PROCESS_NAME` | Final running check |
| `STOP_POPUP_WAIT_SECONDS` | `2` | Sleep before scanning/confirming popup |
| `STOP_EXIT_WAIT_SECONDS` | `30` | After popup confirm, poll until process exits (BTS may flush cache) |
| `STOP_POPUP_KEYWORD` | `警告` | Match in window **title** and/or **dialog content** (child control text) |
| `STOP_POPUP_MATCH_MODE` | `title_or_content` | `title` \| `content` \| `title_or_content` — use `content` when popup has no title (SINEXCEL) |
| `STOP_FORCE_AFTER_GRACEFUL` | `true` | Force kill if still running |
| `STOP_TASK_TIMEOUT_SECONDS` | `90` | Interactive scheduled-stop wait timeout (must cover popup + exit wait) |

The graceful stop task runs in the logged-in desktop user's interactive session (scheduled task), not the WinRM service session.

### Start popup confirm (SINEXCEL default)

After the interactive scheduled task launches the EXE, `sem_reconfig_start_program_windows.ps1` can dismiss a **start** dialog before process verify:

| Variable | Default (SINEXCEL) | Notes |
|----------|-------------------|--------|
| `START_POPUP_KEYWORD` | `提示` | Window title substring; sends Enter in desktop user's session |
| `START_POPUP_WAIT_SECONDS` | `3` | Wait before first popup scan |

Set `START_POPUP_KEYWORD` empty to disable (other device types). Log lines: `START_POPUP_TASK`, `POPUP_CONFIRMED`, `POPUP_NOT_FOUND`.

**Restart / redeploy:** `land/device_restart.yml`, `land/device_check_restart.yml` (and SINEXCEL equivalents) include `stop_program_graceful_before_reconfig.yml` before ModifyConfig + start when they enter the restart path — same graceful-stop behavior as the historical stop-only flow, not `Stop-Process -Force`.

**LAND config:** `land_prepare_merged_cfg.yml` splits `SystemConfig` (ModifyConfig) vs `Redeliver` (restart-only redeliver). ModifyConfig two-phase while `RUNNING`; `land_api_redeliver.yml` only from `land/device_restart.yml` after API OK with valid `startTime`/`endTime`.

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

Increase wait: Variable Group `STOP_EXIT_WAIT_SECONDS=60`, `STOP_TASK_TIMEOUT_SECONDS=120`, `STOP_POPUP_WAIT_SECONDS=5`.

**No compatibility shims** under `neware/tasks/` for shared files — include `{{ sem_tasks_dir }}/…` directly from each play. Missing file → add under `shared/` or the device profile; wrong path → fix the play.
