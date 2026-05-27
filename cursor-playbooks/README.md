# cursor-playbooks

Ansible playbooks for **Semaphore UI** device actions (discovery, patrol, start/stop/restart). Layout is **one directory per device type** so new profiles can add their own tree without touching existing ones.

## Layout

| Path | Purpose |
|------|---------|
| **`neware/`** | NEWARE Windows hosts — current production playbooks (`device_*.yml`, `tasks/`, `files/`, `group_vars/`) |
| *(future)* **`other_type/`** | Additional device types — copy `neware/` as a template |

**Semaphore templates** should point at playbooks under the type folder, for example:

- `cursor-playbooks/neware/device_status.yml`
- `cursor-playbooks/neware/device_start.yml`
- `cursor-playbooks/device_discovery.yml` (network scan + API callback; device-type agnostic; uses `neware/tasks/semaphore_discovery_put_results.yml` for the API PUT)

The **`cursor-playbooks/`** root keeps only **`README.md`** and **`device_discovery.yml`**. All NEWARE device action playbooks live under **`neware/`**.

Semaphore binds **one auto inventory per device type** (`windows_hosts (auto: …)`). Point each type’s templates at that inventory for manual runs; list actions still use ephemeral `windows_hosts batch …` inventories.

## Documentation

- **NEWARE playbooks (detailed):** [`neware/README.md`](neware/README.md) — Variable Groups, callbacks, WinRM, debug markers.
- **Authoring / review:** `.claude/skills/semaphore-cursor-playbooks/SKILL.md`
- **Cloud agent notes:** `AGENTS.md` (Devices section)

## Adding a new device type

1. Create `cursor-playbooks/<profile_key>/` (e.g. `acme/`).
2. Start from `neware/` — copy or symlink playbooks and adjust paths, EXE paths, and callbacks as needed.
3. Bind templates in Semaphore **Device types** to `cursor-playbooks/<profile_key>/device_*.yml`.
4. Project-level **discovery** uses `cursor-playbooks/device_discovery.yml` (device-type agnostic).
