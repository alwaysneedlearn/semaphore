# Semaphore RDP Helper

Windows helper for native Remote Desktop via `mstsc`, with optional SSH ControlMaster
tunnels and a **local web panel**. See [`docs/plan-rdp-helper.md`](../docs/plan-rdp-helper.md).

## Install (no admin)

1. Copy `semaphore-rdp-helper.exe` to a user folder (e.g. `%LOCALAPPDATA%\SemaphoreRdpHelper\`).
2. Double-click the exe (or run with no args) → opens **http://127.0.0.1:17300** panel.
3. In the panel: edit JSON environments → **保存配置** → select env → **连接** → **打开网页**.
4. Optional: **注册协议** (also auto-attempted on UI start).
5. In Semaphore device list, click **远程桌面**.

CLI still works: `install` / `connect` / `open` / `disconnect` / `status` / `ui`.

## Local web panel

| 操作 | 说明 |
|------|------|
| 环境列表 | 选择 active 环境 |
| 连接 / 断开 | SSH ControlMaster（+ 可选 UI 端口映射） |
| 打开网页 | 打开该环境的 `semaphore_url` |
| 注册协议 | HKCU `semaphore-rdp://` |
| 配置 JSON | 编辑并保存 `config.json` |
| 日志 | 查看 `helper.log` 尾部 |

- Listen: `127.0.0.1:17300` only (loopback). Override: env `SEMAPHORE_RDP_HELPER_UI=127.0.0.1:17301`.
- Second launch: if port busy, opens existing panel in browser and exits.

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

GitHub Actions：**Dev** workflow 打包 `semaphore-dev.zip`（Linux `semaphore` + Windows `semaphore-rdp-helper.exe` + README）。
