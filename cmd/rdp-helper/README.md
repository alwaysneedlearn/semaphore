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
   - **本机直连**：填 Semaphore 网址，如 `http://10.x.x.x:3000`
   - **经 SSH 跳板**：填跳板主机/用户、内网 Semaphore 主机，勾选端口映射
4. **保存当前项目** → **连接** → **打开网页**
5. 在 Semaphore 设备列表点 **远程桌面**

经 SSH 时：建议从 **cmd/PowerShell** 启动 Helper（保留原控制台）。点 **连接**（或点远程桌面自动连接）时在**同一控制台**输入跳板/落地机密码。连接成功后 `state.json` 会标记已连接；若未连接就点远程桌面，Helper 会自动建连而不再只报 `not connected`。

首次可点 **注册协议**（无管理员，写 HKCU）。需要 OpenSSH Client（Windows 可选功能），且支持 `ControlMaster`。

## 配置说明（按项目）

每个「项目环境」一套入口，互不影响：

| 字段 | 含义 |
|------|------|
| 项目 ID / 名称 | 本机区分用，不必等于 Semaphore 项目数字 ID |
| 直连 URL | 浏览器能直接打开的 Semaphore 地址 |
| 跳板 1/2 | `ssh -J` 顺序；最后一跳默认为落地机 |
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
