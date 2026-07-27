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

可用桌面快捷方式启动：目标直接指向 exe 即可（无参数会进入交互提示符 `rdp>`，窗口不会秒关）。也可 `cmd.exe /k "...\semaphore-rdp-helper.exe"`。**起始位置**随意——程序按 exe 所在目录读写配置。移动文件夹后请再执行一次 `install`（更新协议注册路径）。

## 用法

### 交互模式（推荐快捷方式 / 双击）

```text
rdp> install
rdp> envs
rdp> connect plant-a
rdp> open
rdp> status
rdp> disconnect plant-a
rdp> exit
```

### 一次性命令

```text
semaphore-rdp-helper.exe install
semaphore-rdp-helper.exe connect <项目ID>
semaphore-rdp-helper.exe open
semaphore-rdp-helper.exe status
semaphore-rdp-helper.exe disconnect <项目ID>
```

典型流程：

1. 快捷方式打开 → `install`（注册协议；生成默认 `config.json`）
2. 编辑 exe 同目录的 `config.json`
3. `connect <项目ID>`（多项目可分别 connect，互不影响）
4. `open` 打开 Semaphore 网页
5. 在设备列表点 **远程桌面**
6. 用完 `exit` 关闭窗口（SSH 子进程仍可按需保留；要断隧道用 `disconnect`）

说明：Windows 自带 OpenSSH **不支持** `ControlMaster`；Helper 使用 `ssh -N` + SOCKS（`-D`）转发 RDP。

## 配置文件

路径：`<exe所在目录>\config.json`

示例（Semaphore 直连 + 仅 RDP 走跳板）：

```json
{
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
| `id` / `name` | 本机项目标识；`connect` / `disconnect` / `open` 用 `id`（多项目时必填） |
| `semaphore_url` | 浏览器打开的 Semaphore 地址（远程桌面协议用 `base=` 匹配项目） |
| `forward_ui` | `true` 时用 SSH `-L` 映射网页；Semaphore 已直连则设 `false` |
| `hops` | 跳板列表（含非 22 端口）；空 `land_host` 时最后一跳即落地机 |
| `land_*` | 仅当落地机与最后一跳不同时填写 |

## RDP 选项

### 剪贴板（默认开启）

Helper 生成的临时 `.rdp` 含 **`redirectclipboard:i:1`**（对应 mstsc「本地资源 → 剪贴板」勾选）。无需在连接对话框再勾。

### 锁屏显示「正在被远程」（自定义文案）

**mstsc / `.rdp` 无法**在被控机锁屏上动态写「正在被远程」。需在 **被控 Windows** 上部署策略或脚本：

| 方式 | 效果 | 说明 |
|------|------|------|
| **LegalNotice**（登录/解锁提示） | 锁屏/登录前弹出标题+正文 | 可用 GPO 或本目录 `scripts/set-lock-notice.ps1` |
| **会话感知**（推荐） | 有活跃 `rdp-tcp` 会话时写入提示，断开后清除 | `scripts/watch-rdp-session-notice.ps1`（计划任务每分钟或常驻） |

被控机上（管理员 PowerShell）示例：

```powershell
# 一次性固定文案（每次锁屏/登录都会显示，与是否在远程无关）
.\scripts\set-lock-notice.ps1 -Title "远程协助中" -Text "本机可能被运维远程桌面连接，请确认后再操作。"

# 仅在有人 RDP 连入时显示（计划任务：每 1 分钟 -Once）
.\scripts\watch-rdp-session-notice.ps1 -Title "正在被远程" -Text "操作员远程桌面连接中，请勿本地操作。" -Once
```

注册表（与脚本相同）：

- `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System\LegalNoticeCaption`
- `...\LegalNoticeText`

下次锁屏或登录生效。若域 GPO/Intune 也下发这两项，以策略为准。

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o semaphore-rdp-helper.exe ./cmd/rdp-helper
```
