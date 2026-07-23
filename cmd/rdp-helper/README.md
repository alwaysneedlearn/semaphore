# Semaphore RDP Helper

Windows helper for native Remote Desktop via `mstsc`, with optional SSH ControlMaster
tunnels. See [`docs/plan-rdp-helper.md`](../docs/plan-rdp-helper.md).

## Install (no admin)

1. Copy `semaphore-rdp-helper.exe` to a user folder (e.g. `%LOCALAPPDATA%\SemaphoreRdpHelper\`).
2. Run:

```text
semaphore-rdp-helper.exe install
```

Registers `semaphore-rdp://` under **HKCU** and creates `config.json`.

3. Edit `%LOCALAPPDATA%\SemaphoreRdpHelper\config.json` (environments / hops).
4. `semaphore-rdp-helper.exe connect <env-id>`
5. `semaphore-rdp-helper.exe open` → log into Semaphore → device **Remote desktop**.

## config.json example

```json
{
  "active_env": "plant-a",
  "environments": [
    {
      "id": "plant-a",
      "name": "Plant A",
      "semaphore_url": "http://127.0.0.1:3000",
      "forward_ui": true,
      "ui_local_port": 3000,
      "ui_remote_host": "10.0.0.5",
      "ui_remote_port": 3000,
      "land_user": "ops",
      "hops": [
        { "host": "jump1.example", "port": 22, "user": "ops" },
        { "host": "jump2.internal", "port": 22, "user": "ops" }
      ]
    }
  ]
}
```

- **0 hops / empty land**: direct (no SSH).
- **hops**: last hop is land unless `land_host` is set; earlier hops become `ProxyJump`.
- Passwords: if the device has `rdp_password` in Semaphore, Helper receives it briefly and uses `cmdkey`+`mstsc`; otherwise mstsc prompts.

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o semaphore-rdp-helper.exe ./cmd/rdp-helper
```

GitHub Actions：**Dev** workflow（`.github/workflows/dev.yml`）构建后打包
`semaphore-dev.zip`（含 Linux `semaphore` + Windows `semaphore-rdp-helper.exe` + README）。
