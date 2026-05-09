# Agents

## Cursor Cloud specific instructions

### Overview

Semaphore UI is a Go backend + Vue.js 2 frontend served as a single binary. The build system uses [go-task](https://taskfile.dev/) (`Taskfile.yml`), not Make.

### Prerequisites (already installed by update script)

- **Go 1.24.6** — backend
- **Node.js 22+** — frontend build
- **go-task** — build orchestration (`task` command)

### Key commands

All commands use `task` (go-task). See `Taskfile.yml` for the full list.

| Action | Command |
|--------|---------|
| Install all deps | `task deps` |
| Build frontend + backend | `task build` |
| Build frontend only | `task build:fe` |
| Build backend only | `task build:be` |
| Run frontend lint | `task lint:fe` |
| Run Go lint (requires golangci-lint) | `task lint:be` |
| Run backend tests | `task test:be` or `go test ./...` |
| Run frontend tests | `task test:fe` |

### Running the server

1. Ensure a `config.json` exists in the workspace root (see `.devcontainer/config.json` for an example). For local dev with SQLite:
   ```json
   {
     "sqlite": { "host": "/workspace/database.sqlite" },
     "dialect": "sqlite",
     "cookie_hash": "5WJjXCLpvf3Cn5t+C/IV9F0asZUQLakOhCT+eSdIwP0=",
     "cookie_encryption": "6x6mmQWGn6YcsHN1rN0HiQjhYA+7HukcbCxUGHuT2CE="
   }
   ```
2. Create an admin user (if DB is fresh):
   ```
   ./bin/semaphore user add --admin --login admin --name Admin --email admin@example.com --password changeme --config ./config.json
   ```
3. Start the server:
   ```
   ./bin/semaphore server --config ./config.json
   ```
   The UI is available at http://localhost:3000 (login: `admin` / `changeme`).

### Gotchas

- `$HOME/go/bin` must be on `PATH` for `task` to work. The update script installs `task` there.
- Frontend lint (`task lint:fe`) has pre-existing errors in the repository; these are not caused by the dev environment setup.
- `task lint:be` requires `golangci-lint` and `swagger` CLI tools which are not installed by default; use `go vet ./...` as a lighter alternative for backend static analysis.
- The `go.mod` has a `replace` directive pointing `pro` module to `./pro`; this is normal for the open-source build.
- SQLite is the simplest DB dialect for local dev — no external database service needed.
- After changing Go code, you must rebuild the binary (`task build:be`) before restarting the server. The frontend has a separate build (`task build:fe`).
- **Agents (Cursor Cloud / automation):** After **any Vue/HTML change**, you must run `task build:fe`, then `task build:be` (or equivalent `go build` for `./cli`), then restart `./bin/semaphore ...`, because UI assets are embedded in the binary. If the UI still looks stale in the browser under test, hard refresh (`Ctrl+Shift+R`). This sequence is **operational guidance for agents**, not end-user documentation — do not phrase it in summaries as if instructing the repository owner to “deploy” unless they asked for deployment steps.
- **Device actions**: When a task includes `inventory_id` (e.g. temporary inventory from the Devices UI), the runner uses that inventory first. Short-lived inventories are named like `windows_hosts batch <timestamp>` and are removed automatically when the task finishes (skipped if the task is re-queued waiting for a runner).
- **Device list API** (`GET /api/project/{id}/devices`): returns JSON `{ "devices": [...], "total": N }` with query params `limit`, `offset`, `sort`, `order`, and optional filters `hostname`, `ip`, `device_status`, `rdp_status`, `winrm_status` (substring match for hostname/ip, exact for status fields).
- **Patrol all**: Runs TCP probes on each device (same idea as the periodic scheduler), then enqueues the configured status template. Final `healthy` / detailed flags from Ansible require your playbook to call `PUT /api/project/{id}/devices/status/bulk` (or equivalent) if you rely on script-derived status beyond port checks.
