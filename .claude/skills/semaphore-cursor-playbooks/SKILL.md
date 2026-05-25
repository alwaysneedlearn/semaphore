---
name: semaphore-cursor-playbooks
description: >
  Author, review, or debug Ansible playbooks under cursor-playbooks/ for Semaphore UI Windows device
  management (Patrol, start/stop/restart, bulk status callbacks). Use whenever editing device_status.yml,
  device_start.yml, WinRM tasks, semaphore_callback_row, PUT /devices/status/bulk, win_ping UNREACHABLE,
  include_tasks/delegate_to errors, or runner playbooks at /root/playbook/. Triggers on "cursor-playbooks",
  "playbook 编写", "WinRM ping", "checking 不更新", "TaskInclude delegate_to", "semaphore bulk callback",
  "ignore_unreachable", "clear_host_errors", or Ansible + Semaphore device template work in this repo.
---

# Semaphore UI — cursor-playbooks authoring skill

Ansible playbooks in **`cursor-playbooks/`** drive Windows hosts (`windows_hosts`) and write device state back to Semaphore via **`PUT /api/project/{id}/devices/status/bulk`**. Runners are often **older Ansible** (Semaphore embedded runner). Treat every change as **compatibility + callback correctness** first.

**Canonical docs in repo:** `cursor-playbooks/README.md`, `AGENTS.md` (Cloud gotchas). **This skill** captures **hard limits** and **patterns that broke production** so they are not repeated.

---

## Before you edit

1. Read the target playbook’s **first 80 lines** (vars: `_semaphore_device_rows`, `devices` / `device`) and **how it ends** (bulk `post_tasks`).
2. Prefer **shared tasks** under `cursor-playbooks/tasks/` over duplicating callback Jinja in five playbooks.
3. Assume the runner inventory uses **`inventory_hostname` = device IP**; callback rows must include **`hostname` + `ip`** from `_semaphore_device_rows`.
4. After changing playbooks, remind operators to sync **`/root/playbook/`** (or their runner path) from **`develop`**.

---

## Ansible hard limits (runner-safe)

These are **invalid or broken** on typical Semaphore runners — do **not** use them without verifying Ansible version.

| Do NOT | Do instead |
|--------|------------|
| `register:` on a **`block:`** | Register on a **single module task** inside the block (e.g. `ansible.windows.win_shell`) |
| `until:` / `retries:` / `delay:` on **`include_tasks:`** | Put polling on a **module task** (e.g. `win_shell` running `sem_collect_process_status_windows.ps1`) |
| `delegate_to:` / `run_once:` on **`include_tasks:`** (TaskInclude) | Wrap in a **block**; put `delegate_to` / `run_once` on the **block** (see bulk pattern below) |
| `when:` directly on **`meta:`** (`clear_host_errors`, `end_host`, `reset_connection`) | Put `meta` inside a **block** or **include** that has `when:` on the block/include |
| `when:` on **`include_tasks:`** in a **loop** expecting per-iteration re-evaluation | Put `when:` on an inner **block** inside the included file (see `winrm_connect_one_attempt.yml`) |

**Polling example (valid):**

```yaml
- ansible.windows.win_shell: ...
  register: _poll
  until: "'RUNNING' in (_poll.stdout | default(''))"
  retries: 12
  delay: 5
  failed_when: false
```

**Bulk PUT from post_tasks (valid):**

```yaml
post_tasks:
  - name: 写回 Semaphore 设备状态
    run_once: true
    delegate_to: localhost
    block:
      - ansible.builtin.include_tasks: tasks/semaphore_bulk_put_from_hostvars.yml
```

**Invalid (will ERROR!):**

```yaml
- ansible.builtin.include_tasks: tasks/semaphore_bulk_put_from_hostvars.yml
  delegate_to: localhost   # FAIL: TaskInclude
  run_once: true
```

**Immediate bulk (`tasks/semaphore_bulk_put_immediate.yml`) — same rule**

This file is included from **`semaphore_callback_winrm_connect_failed.yml`** and other paths **before `end_host`**. Any localhost work that uses **`include_tasks`** must use a **block** (credentials resolve) or a **module** (`uri`, `debug`) with `delegate_to` on that task — never on the `include_tasks` line.

```yaml
# Valid (current pattern in repo)
- name: 解析 bulk PUT 凭据（localhost）
  run_once: true
  delegate_to: localhost
  block:
    - ansible.builtin.include_tasks: tasks/semaphore_resolve_bulk_credentials.yml

- name: 立即 PUT 本主机设备状态（单条 bulk）
  ansible.builtin.uri:
    ...
  delegate_to: localhost
```

```yaml
# Invalid — caused production ERROR! (fixed in 629708b5)
- ansible.builtin.include_tasks: tasks/semaphore_resolve_bulk_credentials.yml
  delegate_to: localhost
  run_once: true
```

**Before committing any new/edited `cursor-playbooks/tasks/*.yml`**, run:

```bash
rg -n 'include_tasks:.*\n\s+delegate_to:' cursor-playbooks/ || true
rg 'delegate_to:|run_once:' cursor-playbooks/tasks/*.yml | rg -B2 'include_tasks'
```

If the second command shows `include_tasks` immediately followed by `delegate_to` / `run_once` on the **same task** (not inside a parent `block:`), fix it before push.

---

## WinRM UNREACHABLE (most common production bug)

### What goes wrong

- `ansible.windows.win_ping` (or any `win_*`) returns **`UNREACHABLE`** → Ansible **removes the host from the active play** for remaining tasks.
- **`failed_when: false` does NOT stop this.** The task may show as non-failed; the host is still skipped afterward.
- Tasks **after** ping in the **same `block`** (e.g. `clear_host_errors`, `set_fact` callback) **never run** → no `semaphore_callback_row` → bulk omits host → UI stays **`checking`**.
- Play recap **`unreachable=1`**, exit code **4**; Semaphore may **skip a second `hosts: localhost` play** — do not rely on a separate localhost play only.

### Required pattern (play start)

1. **`tasks/winrm_ensure_reachable.yml`** at the **top** of every `windows_hosts` play (before other `win_*`).
2. **`win_ping`** in `tasks/winrm_connect_one_attempt.yml`:
   - `failed_when: false`
   - **`ignore_unreachable: true`** (mandatory)
   - **Never use `_winrm_ping is succeeded` alone** — with `ignore_unreachable`, unreachable still shows task **ok** and used to set **`_winrm_session_ok`** wrongly. Use **`tasks/winrm_eval_ping_result.yml`**: `_winrm_ping_ok` = `ping == 'pong'` and not `unreachable`. Only then set **`_winrm_session_ok`**.
   - After ensure, include **`tasks/winrm_gate_play_tasks.yml`** so deploy/collect never run on a host without real pong (keep full **`WINRM_CONNECT_RETRIES`** loop).
3. In the same attempt, use **`block` + `always:`** when ping is not succeeded:
   - `meta: clear_host_errors`
   - optional `set_fact: _winrm_connect_failed: true`
4. After all retries, if still not `_winrm_session_ok` and play defines **`_semaphore_device_rows`**:
   - `clear_host_errors` (again if needed)
   - **`include_tasks: tasks/semaphore_callback_winrm_connect_failed.yml`** → `device_status: unhealthy`, `winrm_status`/`api_status: offline`, then **`meta: end_host`**
5. **`deploy_sem_windows_helper_scripts.yml`**: only when `_winrm_session_ok` (avoid fatal `win_copy` on dead sessions).

### Mid-play reconnect

- **`tasks/winrm_refresh_midplay.yml`** re-includes `winrm_ensure_reachable` with `winrm_force_reconnect: true` before long `win_shell` chains (log check, reconfig) on **start/restart/check_restart** — **not** on **`device_status.yml`** (patrol: play-start ensure only).
- Reconnect failure uses the **same** ping-failure callback path.

### Patrol batch

- `device_status.yml`: `max_fail_percentage: 100` on the `windows_hosts` play so one bad host does not abort the whole batch.

---

## Semaphore status callback contract

### Every template that clears `checking`

Playbooks with bulk callback (**not** `device_discovery.yml`) must ensure **each** inventory host ends with **`hostvars[host].semaphore_callback_row` defined**, or **`semaphore_callback_winrm_fallback_missing_rows.yml`** runs on localhost before PUT.

| Field | Rules |
|-------|--------|
| `device_status` | `healthy` / `unhealthy` — **never** use `unknown` for WinRM-down (use **`unhealthy`**) |
| `winrm_status` | `online` if WinRM used for the success path; **`offline`** on ping/collect/log unreachable |
| `api_status` | Patrol/start/restart: **log/process gate**, not HTTP status API; `online` when `need_reconfigure` is false or `final_start_ok`; stop → **`offline`** always |
| `rdp_status` | **Omit** in patrol/start/restart/stop — RDP is **Probe** / discovery only |
| `abnormal_reason` | Human-readable; distinguish **ping failed** vs **collect unreachable** vs **log check unreachable** |
| `hostname`, `ip` | From `_semaphore_device_rows` match on `inventory_hostname` |

### Play vars (required for matching)

```yaml
_devices_from_extra: "{{ devices | default([]) }}"
_device_singleton: "{{ [device] if (device is defined) else [] }}"
_semaphore_device_rows: "{{ _devices_from_extra if (_devices_from_extra | length > 0) else _device_singleton }}"
```

### Bulk PUT execution

- **Bulk PUT** must run in a **`hosts: localhost` second play** after every `windows_hosts` host has run post_tasks **`登记 Semaphore 回调行`**. **Do not** put `run_once` bulk in the same play’s post_tasks — the first host to finish post_tasks triggers bulk while others still lack **`semaphore_callback_row`** (batch restart: 3× `final_start_ok=True` but UI stays unhealthy). **`device_stop`** may keep post_tasks-only bulk (short play).
- When all hosts `end_host` early, play1 may skip post_tasks; use second play + optional **`semaphore_bulk_put_immediate.yml`** before `end_host` on failure paths.
- **`semaphore_bulk_put_immediate.yml`**: resolves credentials in a **localhost block** → single-host **`uri` PUT** → DEBUG. **`semaphore_callback_winrm_fallback_missing_rows.yml`** runs only in **`semaphore_bulk_put_from_hostvars.yml`** (batch post_tasks), not inside the immediate file.
- **Requires** env **`SEMAPHORE_API_TOKEN`**; optional `SEMAPHORE_URL` (default `http://127.0.0.1:3000`); **`semaphore_project_id`** from Semaphore extra-vars.

### Callback before `end_host`

Any path that calls **`meta: end_host`** must **`set_fact: semaphore_callback_row` in an earlier task on that host** (unreachable rows). If you `end_host` first, later tasks on that host will not run.

### `ignore_unreachable` on fragile `win_shell`

Wrap **collect**, **resolve EXE_DIR**, **log health check** in blocks with **`ignore_unreachable: true`**, then branch on `result.unreachable` to set callback + `end_host`. Patrol log task without this **fatal**s and skips the final callback row.

---

## Jinja / boolean gotchas

| Mistake | Fix |
|---------|-----|
| `set_fact: flag: "{{ false }}"` | Stores string **`"False"`** → **truthy** in `when: flag` | Use YAML literals: `flag: false` or two explicit `set_fact` tasks |
| `final_start_ok \| default(true)` | Masks failure → false healthy | Use **`default(false)`** for success flags |
| `need_reconfigure \| default(false)` when var may be string `"False"` | Wrong gate | Set booleans with literal `true`/`false` in `health_gate_need_reconfigure_from_log.yml` pattern |
| `api_port` via broken ternary | Becomes boolean | Use explicit `{% if (hp \| int) > 0 %}` pattern in play vars |

---

## Repository layout & conventions

```
cursor-playbooks/
  device_status.yml      # Patrol — no app HTTP API for api_status
  device_start.yml       # Start + reconfig + start_verify_after_reconfig
  device_restart.yml
  device_stop.yml
  check_restart_redeploy.yml
  device_discovery.yml   # No bulk callback
  files/sem_*.ps1        # Deployed once via deploy_sem_windows_helper_scripts.yml
  tasks/
    winrm_ensure_reachable.yml
    winrm_connect_one_attempt.yml
    semaphore_callback_winrm_connect_failed.yml
    semaphore_callback_winrm_fallback_missing_rows.yml
    semaphore_bulk_put_from_hostvars.yml
    deploy_sem_windows_helper_scripts.yml
    collect_process_status_windows.yml
    log_health_check_windows.yml
    health_gate_need_reconfigure_from_log.yml
    start_verify_after_reconfig.yml
    ...
```

- **Env defaults:** `lookup('env', 'VAR')` with trim; empty env → playbook default (see README table).
- **Scripts:** one **`win_copy`** of `files/` → `C:\Windows\Temp\`, not per-task copy.
- **Health gate:** process running + recent log line matching **`LOG_SUCCESS_KEYWORD`** (`LOG_HEALTH_RECENT_MINUTES`; check_restart may use tail mode).
- **Start verify:** no start/status HTTP API in verify; `final_start_ok` = script `VERIFY_OK` + process poll + log poll; log baseline **before** starting EXE.

---

## Review checklist (use before commit)

- [ ] Ran grep for **`delegate_to`/`run_once` on `include_tasks`** (see command under “Immediate bulk” above).
- [ ] No `delegate_to` / `run_once` / `until` / `register` on illegal targets (see table above).
- [ ] New `win_*` task: is `winrm_ensure_reachable` still first? Mid-play reconnect if long chain?
- [ ] Any new unreachable path: **`semaphore_callback_row`** + bulk/fallback still reached?
- [ ] Ping failure: `ignore_unreachable` + `always` clear + shared failed callback task?
- [ ] Bulk PUT in **`post_tasks`** block on localhost, not bare `include_tasks`?
- [ ] **`semaphore_bulk_put_immediate.yml`** (if touched): credentials **`include_tasks` inside block**; PUT is **`uri`**, not include with `delegate_to`?
- [ ] Booleans and `final_start_ok` defaults sane?
- [ ] README / AGENTS.md updated if behavior or env vars changed?

---

## Mistakes we already made (do not repeat)

1. **Ping failed → only `clear_host_errors`**, no callback → UI **checking**.
2. **`clear_host_errors` after ping in same block** without `ignore_unreachable` → clear **never ran**.
3. **Removed `clear_host_errors`** when adding callback → callback task **skipped** on UNREACHABLE.
4. **`delegate_to: localhost` on `include_tasks`** → `TaskInclude` parse error (including first draft of **`semaphore_bulk_put_immediate.yml`** — skill already forbade it; fix is block-wrap per **`629708b5`**).
5. **Second play localhost only** for bulk → Semaphore **exit 4** skipped PUT; recap had no localhost.
6. **`device_status: unknown`** on WinRM down → inconsistent; use **`unhealthy`**.
7. **`until` on `include_tasks`** for process poll → old Ansible error.
8. **`register` on `block`** for poll → old Ansible error.
9. **Log check without `ignore_unreachable`** → fatal, no final patrol callback.
10. **`final_start_ok | default(true)`** → false **healthy** in API row.
11. **`_winrm_ping is succeeded` after `ignore_unreachable`** → false **已连通** / `winrm_status: online` — use **`_winrm_ping_ok`** (`ping: pong`).
12. **Ping fail then `重试前重置与等待` + ping again** — usually **`WINRM_CONNECT_RETRIES` loop** (label `第 N/M 次`, default **2** per ensure), not business steps. **Four ping lines** on start/restart often means **two** ensures (`play-start` + **`winrm_refresh_midplay`**); **patrol** should only show **play-start** (max 2 pings). Playbook reads **`lookup('env', …)` only** — Variable Group **JSON** extra-vars do **not** set this; use VG **ENV** or server `env_vars`. Check **`[DEBUG-WINRM] attempts=`** and **`WINRM_CONNECT_RETRIES_env=`**.

### Why “skill said no” but it still shipped once

The **Ansible limits table** and mistake **#4** were already in this skill when **`semaphore_bulk_put_immediate.yml`** was added. The failure mode was **process**, not missing documentation: a new file was written by copying the **post_tasks** *idea* but putting `delegate_to` on the **`include_tasks`** line for credentials (invalid), without running the checklist grep or opening the **valid block example** above. **Skills do not run automatically on every edit** — treat the checklist + `rg` as mandatory when adding localhost `include_tasks`.

---

## When editing Semaphore Go code (out of scope but linked)

Device actions inject **`devices`**, **`configs_by_host`**, **`semaphore_project_id`**. Playbooks are copied to the runner separately; **commit playbooks to git** and sync runner — CI may **`paths-ignore` cursor-playbooks/**.

For UI/backend device behavior see **`AGENTS.md`** (device list API, patrol all, bulk `configs_by_host`). This skill is **playbook-only**.
