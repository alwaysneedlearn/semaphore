# Semaphore RDP Helper

Windows CLI helper：`ssh` 跳板 + `mstsc` 远程桌面。见 [`docs/plan-rdp-helper.md`](../docs/plan-rdp-helper.md)。

## 下载

Dev Actions 两个 Artifact：

| Artifact | 内容 |
|----------|------|
| `semaphore` | Linux `semaphore` 服务端二进制 |
| `semaphore-rdp-helper` | `semaphore-rdp-helper.exe` + 本 README |

## 布局（绿色 / 便携）

把 `semaphore-rdp-helper.exe` 放到任意可写目录（**不要**放 Program Files）。所有数据与 exe **同目录**：

| 文件 | 说明 |
|------|------|
| `config.json` | 项目 / 跳板配置 |
| `state.json` | 各项目连接状态 |
| `helper.log` | 本地日志 |
| `launch-*.rdp` | 临时 RDP 文件（用完删除） |

可用桌面快捷方式启动：目标指向 exe（或 `cmd.exe /k "...\semaphore-rdp-helper.exe"` 方便看输出）。**起始位置**随意——程序按 exe 所在目录读写配置，不依赖当前工作目录。移动文件夹后请再执行一次 `install`（更新协议注册路径）。

## 用法（命令行）

在 **cmd / PowerShell** 中运行（保留窗口以便输入 SSH 密码）：

```text
semaphore-rdp-helper.exe install
semaphore-rdp-helper.exe envs
semaphore-rdp-helper.exe connect <项目ID>
semaphore-rdp-helper.exe open
semaphore-rdp-helper.exe status
semaphore-rdp-helper.exe disconnect <项目ID>
```

典型流程：

1. `install`（注册 `semaphore-rdp://`，若尚无则生成默认 `config.json`）
2. 编辑 exe 同目录的 `config.json`
3. `connect <项目ID>`（多项目可分别 connect，互不影响）
4. `open` 打开 Semaphore 网页
5. 在设备列表点 **远程桌面**

说明：Windows 自带 OpenSSH **不支持** `ControlMaster`；Helper 使用 `ssh -N` + SOCKS（`-D`）转发 RDP。

## 配置文件

路径：`<exe所在目录>\config.json`

示例（Semaphore 直连 + 仅 RDP 走跳板）：

```json
{
  "active_env": "plant-a",
  "environments": [
    {
      "id": "plant-a",
      "name": "厂区A",
      "semaphore_url": "http://10.20.1.10:3000",
      "forward_ui": false,
      "ui_local_port": 3000,
      "ui_remote_host": "",
      "ui_remote_port": 3000,
      "hops": [
        { "host": "10.20.1.134", "port": 22, "user": "root" }
      ],
      "land_host": "",
      "land_user": "",
      "land_port": 0,
      "ssh_identity": ""
    }
  ]
}
```

| 字段 | 含义 |
|------|------|
| `id` / `name` | 本机项目标识；`connect` / `disconnect` 用 `id` |
| `semaphore_url` | 浏览器打开的 Semaphore 地址 |
| `forward_ui` | `true` 时用 SSH `-L` 映射网页；Semaphore 已直连则设 `false` |
| `hops` | 跳板列表（含非 22 端口）；空 `land_host` 时最后一跳即落地机 |
| `land_*` | 仅当落地机与最后一跳不同时填写 |

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o semaphore-rdp-helper.exe ./cmd/rdp-helper
```
