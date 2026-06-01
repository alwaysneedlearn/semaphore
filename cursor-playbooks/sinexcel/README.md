# SINEXCEL playbooks (`cursor-playbooks/sinexcel/`)

Device type **SINEXCEL**: LAND-style lifecycle (process + optional HTTP API) with **NEWARE-style INI config** on start/restart (via `../neware/tasks/apply_neware_style_device_config_files.yml`).

## Playbooks

| File | Purpose |
|------|---------|
| `device_status.yml` | Exe + process + API check, bulk callback |
| `device_start.yml` | Config files (NEWARE) + zip/deploy + interactive task start |
| `device_stop.yml` | Force stop process |
| `device_restart.yml` | Stop + redeploy + config + start |
| `device_check_restart_redeploy.yml` | Restart only when unhealthy (bind as **check_restart_redeploy** template) |

Set in each play:

```yaml
sem_tasks_dir: "{{ playbook_dir }}/../shared/tasks"
sem_files_dir: "{{ playbook_dir }}/../shared/files"
sem_debug_tag: SINEXCEL
```

Task logs: search **`[DEBUG-SINEXCEL]`** (exe, process, SyncLims API, callback; see `shared/README.md`).

## Semaphore templates

Example paths:

- `cursor-playbooks/sinexcel/device_status.yml`
- `cursor-playbooks/sinexcel/device_start.yml`
- …

## Variable Group (examples)

| Variable | Default | Notes |
|----------|---------|--------|
| `EXE_DIR` | `C:\Program Files\SINEXCEL` | Preferred install root; drive fallback `preferred -> E: -> C:` |
| `EXE_DIR_FALLBACK_DRIVES` | empty (`preferred,E,C`) | Optional drive order override (comma/space separated), e.g. `E,C,D` |
| `EXE_NAME` | `sinexcel_agent.exe` | |
| `ZIP_NAME` | `sinexcel` | Subfolder under `EXE_DIR` |
| `ZIP_PATH` | `/root/sinexcel/pkg` | Controller zip directory |
| `CONFIG_FILE_NAME` | `sinexcel.iconf` | Template on controller under `ZIP_PATH` |
| `RECONFIG_CLIENT_REL_PATH` | `Documents\SINEXCEL\BTSClient` | User/Public INI directory |
| `RECONFIG_CONFIG_FALLBACK_USERS` | `SINEXCEL,Administrator` | |
| `SINEXCEL_API_*` | See `device_status.yml` | Same pattern as LAND SyncLims-style API |
| `API_PORT` | `8080` | Written into INI `ReportApiSettings.ServerPort` |
| `STOP_GRACEFUL_PROCESS_NAME` | `LHBTS` | Graceful stop (`CloseMainWindow`) |
| `STOP_VERIFY_PROCESS_NAME` | `sinexcel_agent` | Verify process stopped |
| `STOP_POPUP_WAIT_SECONDS` | `2` | Wait for confirmation dialog |
| `STOP_POPUP_KEYWORD` | `警告` | Dialog title keyword |
| `STOP_FORCE_AFTER_GRACEFUL` | `true` | Force kill if still running |

## Extra-vars

Bulk: `devices`, `configs_by_host`, `default_config`  
Single: `device`, `config`
