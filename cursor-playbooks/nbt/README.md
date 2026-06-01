# NBT playbooks (`cursor-playbooks/nbt/`)

Device type **NBT**: same layout as [sinexcel/README.md](../sinexcel/README.md) with NBT defaults.

Task logs: search **`[DEBUG-NBT]`** (same shared debug tasks as LAND/SINEXCEL; `sem_debug_tag: NBT` in each play).

## Semaphore templates

- `cursor-playbooks/nbt/device_status.yml`
- `cursor-playbooks/nbt/device_start.yml`
- `cursor-playbooks/nbt/device_stop.yml`
- `cursor-playbooks/nbt/device_restart.yml`
- `cursor-playbooks/nbt/device_check_restart_redeploy.yml`

## Variable Group (examples)

| Variable | Default |
|----------|---------|
| `EXE_DIR` | `C:\Program Files\NBT` (preferred drive; fallback `preferred -> E: -> C:`) |
| `EXE_NAME` | `nbt_agent.exe` |
| `ZIP_NAME` | `nbt` |
| `ZIP_PATH` | `/root/nbt/pkg` |
| `CONFIG_FILE_NAME` | `nbt.iconf` |
| `RECONFIG_CLIENT_REL_PATH` | `Documents\NBT\BTSClient` |
| `RECONFIG_CONFIG_FALLBACK_USERS` | `NBT,Administrator` |
| `NBT_API_*` | HTTP API (LAND-style env names with `NBT_` prefix) |
