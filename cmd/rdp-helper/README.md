# Semaphore RDP Helper

Windows helper：本机 Web 面板 + `mstsc` 远程桌面。见 [`docs/plan-rdp-helper.md`](../docs/plan-rdp-helper.md)。

## 下载

Dev Actions 两个 Artifact（不要再下成「zip 里套 zip」的合并包）：

| Artifact | 内容 |
|----------|------|
| `semaphore` | Linux `semaphore` 服务端二进制 |
| `semaphore-rdp-helper` | `semaphore-rdp-helper.exe` + 本 README |

## 用法（面板）

1. 双击 `semaphore-rdp-helper.exe` → 打开 **http://127.0.0.1:17300**
2. **新建项目**（或改默认「我的项目」）
3. 选访问方式：
   - **本机直连**：Semaphore 和设备 RDP 本机都能直接访问
   - **经 SSH 跳板**：
     - Semaphore 也要映射：勾选「映射 Semaphore」，填内网主机
     - **仅 RDP 要跳板、Semaphore 已直连**：**取消勾选**「映射 Semaphore」，「打开网页地址」填直连 URL（如 `http://10.x.x.x:3000`），仍填跳板
4. **保存当前项目** → **连接** → **打开网页**
5. 在 Semaphore 设备列表点 **远程桌面**

经 SSH 时：建议从 **cmd/PowerShell** 启动 Helper（保留原控制台）。点 **连接** 时在**同一控制台**输入跳板/落地机密码。

说明：Windows 自带 OpenSSH **不支持** `ControlMaster`（会报 `getsockname failed: Bad file descriptor`）。Helper 改为 `ssh -N` + 本机 SOCKS（`-D`），远程桌面经 SOCKS 转发，不依赖 ControlMaster。

首次可点 **注册协议**（无管理员，写 HKCU）。

## 配置说明（按项目）

每个「项目环境」一套入口，互不影响：

| 字段 | 含义 |
|------|------|
| 项目 ID / 名称 | 本机区分用，不必等于 Semaphore 项目数字 ID |
| 直连 URL | 浏览器能直接打开的 Semaphore 地址 |
| 跳板 1/2 | `ssh -J` 顺序；最后一跳默认为落地机（**端口填在该跳板的「端口」**，不要依赖默认 22） |
| 落地机（高级） | 仅当落地机与最后一跳不同时填写；此时用「落地机 SSH 端口」 |
| 本机端口 + 内网主机 | `ssh -L 本机端口:内网主机:3000` |
| 打开网页地址 | 通常 `http://127.0.0.1:本机端口` |

配置文件：`%LOCALAPPDATA%\SemaphoreRdpHelper\config.json`（面板会写）。

## CLI（可选）

```text
semaphore-rdp-helper.exe          # 打开面板
semaphore-rdp-helper.exe connect <项目ID>
semaphore-rdp-helper.exe open
semaphore-rdp-helper.exe disconnect
```

## Build

```bash
GOOS=windows GOARCH=amd64 go build -o semaphore-rdp-helper.exe ./cmd/rdp-helper
```
