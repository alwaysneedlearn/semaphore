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
- Critical deploy sequence for UI changes: after any frontend change, always run `task build:fe`, then rebuild backend (`task build:be` or `go build ...`) so new assets are embedded, then restart the running `./bin/semaphore ...` process. If UI still looks stale, hard refresh (`Ctrl+Shift+R`) to bust browser cache.
- **Device templates**: “Allow inventory override” appears directly under the template **Inventory** field when editing an Ansible template. Tasks started from the Devices UI create short-lived inventories named like `windows_hosts batch <timestamp>`; those rows are removed automatically when the task finishes (skipped if the task is re-queued waiting for a runner).
