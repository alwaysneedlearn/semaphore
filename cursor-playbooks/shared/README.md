# shared — cross–device-type playbooks

Tasks, helper scripts, and group vars used by **all** device profiles (`neware/`, `land/`, …).

## Layout

| Path | Purpose |
|------|---------|
| `tasks/` | WinRM connect/retry, Semaphore bulk/discovery callbacks, interactive EXE start, deploy helper scripts |
| `files/` | Generic `sem_*.ps1` (process alive, scheduled-task start) |
| `group_vars/windows_hosts.yml` | WinRM timeouts (copy or mirror under each profile dir for Ansible) |

## Required play vars

Each device-type playbook should set:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
sem_files_dir: "{{ playbook_dir }}/../shared/files"
```

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

Those stay under `neware/` or `land/` respectively.
