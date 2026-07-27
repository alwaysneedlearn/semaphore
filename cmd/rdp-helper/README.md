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

## 在目标机「显示器锁屏」上显示提示

桌面置顶横幅（`show-remote-banner.ps1`）只出现在 **RDP 会话桌面**，本机显示器若在锁屏上 **看不到**。

要用 **本机锁屏画面** 显示自定义文案，用锁屏壁纸方案（管理员）：

```powershell
# 拷到被控机后，管理员 PowerShell：
powershell -NoProfile -ExecutionPolicy Bypass -File .\show-lockscreen-banner.ps1 `
  -Title "正在被远程" `
  -Text "操作员远程桌面连接中，请勿本地操作。" `
  -LockConsole
```

作用：
1. **默认保留原锁屏图**：在原壁纸副本顶部叠半透明条 + 文字（不是整屏换成纯色图）
2. 临时强制锁屏策略前会备份原策略；`-Clear` 时恢复
3. `-LockConsole` 尝试锁屏，便于本机显示器立刻看到

若只要纯色底（旧行为）：加 `-SolidBackground`。

**说明：** Windows 锁屏只能显示「一张图」。要在锁屏上出字，必须临时把策略指到这张合成图；`-Clear` 后恢复备份，原锁屏策略/图回来。完全不改锁屏图就只能用 LegalNotice 弹窗（不是壁纸上的字）。

清除（恢复原锁屏策略）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\show-lockscreen-banner.ps1 -Clear
```

若本机已锁屏仍是旧图：在本机按一下键 / 再锁一次，或重登后再看。部分版本策略刷新较慢。

### 其它脚本（对比）

| 脚本 | 显示位置 | 要锁屏吗 |
|------|----------|----------|
| `show-lockscreen-banner.ps1` | **本机锁屏背景** | 要（或已在锁屏） |
| `show-remote-banner.ps1` | 当前会话桌面顶部 | 不要 |
| `set-lock-notice.ps1` | 解锁前 LegalNotice 弹窗 | 要解锁才看到弹窗 |

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o semaphore-rdp-helper.exe ./cmd/rdp-helper
```
