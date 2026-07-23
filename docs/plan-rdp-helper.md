# 方案：Semaphore RDP Helper（原生远程桌面）

> 状态：方案待确认（功能边界）  
> 目标：操作员经 SSH 跳板访问 Semaphore 所在网络时，用 **Windows 本地 Helper + mstsc** 打开设备远程桌面；**不做**浏览器内嵌 RDP。

---

## 1. 背景与网络假设

| 情况 | 说明 |
|------|------|
| Semaphore ↔ 设备 | **可直连**（含 RDP 3389） |
| 操作员 ↔ Semaphore / 设备 | 常需 **SSH 跳板**（0 / 1 / 2 跳）；本地打开 Semaphore 网页往往先做端口映射 |
| 操作员系统 | **Windows**（mstsc + OpenSSH 客户端） |

浏览器无法做可靠的多层 SSH + 原生 RDP；Guacamole / Devolutions / 自研 IronRDP 均为备选，**本方案不采用**。

---

## 2. 方案一句话

**Helper 为入口**：本机安装并配置多环境跳板 → 连接环境（一条 SSH ControlMaster）→ 打开 Semaphore 网页 → 在设备列表点「远程桌面」→ 自定义协议拉起 Helper → 复用隧道做 `ssh -L` → 启动 `mstsc` → 结束后只清理本次转发。

---

## 3. 功能边界（确认用）

### 3.1 做（In Scope）

#### A. Windows Helper（独立程序 `semaphore-rdp-helper.exe`）

| 能力 | MVP | 说明 |
|------|-----|------|
| 多环境配置 | ✅ | 按「厂区/项目」保存：名称、跳板 0/1/2、Semaphore 地址、本地 UI 端口等 |
| 连接 / 断开环境 | ✅ | `ssh` ControlMaster；需要时 `-L` 映射 Semaphore UI |
| 打开网页 | ✅ | 连接成功后打开本机映射后的 Semaphore URL |
| 自定义协议 | ✅ | 注册并处理 `semaphore-rdp://connect?token=...` |
| Token 换参 | ✅ | HTTPS 调 Semaphore API，用一次性 token 取目标与凭据策略 |
| 拉起 RDP | ✅ | 复用 master 增加 LocalForward → `mstsc`；退出后 `-O cancel` 该转发 |
| 直连探测 | ✅ | 本机可达设备 3389 则跳过 SSH，直接 mstsc |
| 本地日志 | ✅ | SSH / 协议 / API 错误可查 |
| 托盘 / 精简 UI | ✅ | 选环境、连接状态、打开网页、查看日志入口 |
| 未安装引导 | ✅（配合网页） | 网页点击后无法拉起协议时提示下载安装 |

#### B. Semaphore 服务端 / Web

| 能力 | MVP | 说明 |
|------|-----|------|
| 设备「远程桌面」按钮 | ✅ | 设备列表行操作（与 Probe / WinRM 并列） |
| 签发一次性 token | ✅ | 绑定用户 + 设备 + 短 TTL；URL **只带 token** |
| Launch API | ✅ | Helper 凭 token 获取：目标 IP/端口、RDP 用户名、密码策略、建议环境 ID（可选） |
| 权限与审计 | ✅ | 项目管理权限；记录谁在何时对哪台发起 launch |
| 未装 Helper 提示 | ✅ | 协议拉起失败时的安装说明 + 下载链接 |
| 项目配置项 | ✅ | Helper 下载 URL、是否启用远程桌面按钮（可选开关） |

#### C. 跳板与复用（设计原则，属 MVP）

| 原则 | 说明 |
|------|------|
| 跳板配置在 **Helper 本机** | 因人而异（家宽双跳 / 公司直通）；Semaphore **不存**操作员私钥 |
| 一条 SSH 多用 | ControlMaster：UI 端口映射与多台 RDP LocalForward **共用**，避免每次完整 `-J` |
| 断开策略 | 关 RDP **不断**整条「环境连接」；断开环境才拆 master |
| 设备信息在 Semaphore | IP、`rdp_user` / `rdp_password` / `rdp_port` 仍以设备字段为准 |

### 3.2 不做（Out of Scope）— 本方案明确排除

| 排除项 | 说明 |
|--------|------|
| 浏览器内嵌 RDP | 不做 Guacamole / IronRDP / Devolutions 集成（可作为**另一条**后续方案） |
| Helper 内设备管理 | 不列表、不改配置、不跑 Patrol / WinRM |
| Helper 内完整 SSH 实现替代系统客户端 | MVP 调用系统 `ssh` / `mstsc`（不强制自研 SSH 栈） |
| Mac / Linux Helper | MVP 仅 Windows |
| 服务端 SSH 跳板代持私钥 | 私钥与 agent 留在操作员本机 |
| 把 Helper 打进 `semaphore.exe` | 独立 artifact、独立版本号 |
| 自动更新 / 代码签名 | 可二期；MVP 可不签 |
| RDP 会话录像、剪贴板策略引擎 | 不做（原生 mstsc 行为） |
| 用 Ansible DeviceAction 开桌面 | 独立 API + 协议，不走模板任务 |

### 3.3 二期（可选，不挡 MVP）

- 托盘常驻、开机自启、SSH 断线重连  
- 环境配置导入/导出、图形化跳板编辑  
- 多 RDP 会话面板、强制取消某条 forward  
- 代码签名、MSI 安装包、应用内检查更新  
- 网页检测「Helper 是否已安装」的更可靠机制  
- 系统代理 / SOCKS 模式作为 ControlMaster 的替代连网方式  

---

## 4. 端到端操作流程

### 4.1 一次性（每台操作员 PC）

1. 下载并安装 `semaphore-rdp-helper.exe`（注册 `semaphore-rdp://`）  
2. 在 Helper 中新增环境（可多个），填写跳板与 Semaphore 访问方式  
3. 确认本机 OpenSSH、密钥/`ssh-agent` 可用  

### 4.2 日常

```text
Helper：选择环境 →「连接」
    → SSH ControlMaster（0/1/2 跳）
    → 如需：-L 本地端口 → 内网 Semaphore
    →「打开网页」→ 浏览器登录 Semaphore

Semaphore：进入项目 → 设备列表 →「远程桌面」
    → 签发 token → 打开 semaphore-rdp://connect?token=...

Helper：处理协议
    → API 换连接参数
    → 已连接？复用 master 加 -L ；否则提示先连接环境 / 或按默认环境自动连
    → 直连可达则跳过隧道
    → mstsc
    → mstsc 退出 → cancel 本次 -L（保留环境连接）
```

### 4.3 异常引导

| 情况 | 行为 |
|------|------|
| 未安装 Helper | 网页提示安装 + 下载链接 |
| 已装未连环境 | Helper 提示先连接对应环境（或选环境） |
| Token 过期 | 提示回网页重新点击 |
| SSH / 密钥失败 | Helper 本地错误 + 日志 |
| 单台 RDP 认证失败 | mstsc / 系统报错；与 Discovery WinRM 凭据问题同类，非 Helper 协议错误 |

---

## 5. 配置职责划分

| 位置 | 内容 |
|------|------|
| **Helper（本机）** | 环境名、hops[]、land/跳板用户、ControlPath、本地 UI 端口、Semaphore 基址（映射后 URL） |
| **Semaphore（服务端）** | 设备 `ip` / `rdp_*`、用户权限、token、审计、Helper 下载地址 |
| **本机 SSH** | `~/.ssh/config`、私钥、agent（可与 Helper 环境对齐 Host 别名） |

**不推荐**把完整跳板链强行只配在 Semaphore：外出与内网操作员拓扑不同，且易诱使服务端管私钥。

可选：Launch API 返回 `suggested_env_id`，与 Helper 环境名约定一致，减少连错网。

---

## 6. 协议与 API（草案）

### 6.1 自定义协议

```text
semaphore-rdp://connect?token=<one-time-token>
```

- 禁止在 URL 中传密码、主机、跳板机密  
- Token：短 TTL（建议 60–120s）、一次性、服务端校验  

### 6.2 API（示意）

```text
POST /api/project/{id}/devices/{device_id}/rdp/launch
  → { "token": "...", "expires_in": 90, "helper_url": "semaphore-rdp://connect?token=..." }

GET  /api/rdp/launch-params?token=...
  → {
      "project_id", "device_id",
      "host", "rdp_port", "rdp_user",
      "rdp_password": null | string,   # 策略：可仅用户名，密码由用户输入；或短时下发后仅内存使用
      "suggested_env": "plant-a"
    }
```

密码是否下发：边界确认项（见 §9）。

---

## 7. 技术选型与发布

| 项 | 选择 |
|----|------|
| Helper 语言 | **Go**（与 Semaphore 同栈，单文件 exe） |
| SSH / RDP | 调用系统 **OpenSSH** + **mstsc** |
| UI | MVP：托盘 + 简单窗口（非 Electron） |
| 仓库 | 建议同仓 `cmd/rdp-helper/`（或独立仓）；**独立版本** |
| CI | GitHub Actions：`GOOS=windows GOARCH=amd64` 产出 `semaphore-rdp-helper.exe` |
| 与主程序关系 | **不**打进 `semaphore` 主二进制；Release 单独资产 + 网页下载链 |

---

## 8. 与其它方案关系

| 方案 | 本阶段 |
|------|--------|
| Native Helper + mstsc（本文） | **主推** |
| Apache Guacamole | 不做；无 Helper / 非 Windows 时另议 |
| Devolutions Gateway | 不做；另议 |
| IronRDP + 自研代理 | 不做；成本高，另议 |

---

## 9. 待你确认的边界问题

请逐条确认或改口：

1. **MVP 仅 Windows Helper + mstsc**，不做浏览器内嵌 RDP——是否同意？  
2. **跳板只配在 Helper**，Semaphore 不管私钥——是否同意？  
3. **Helper 为入口**（先连环境再开网页）；网页也可点远程，但未连环境时由 Helper 提示——是否同意？  
4. **RDP 密码**：A) API 短时下发仅内存使用；B) 只下发用户名，密码本机输入/保存——选哪个？  
5. **多项目**：Helper 多「环境」配置是否足够（不必与 Semaphore 项目一一自动同步）？  
6. **直连探测**：可达则跳过 SSH——是否要做进 MVP？  
7. **独立 exe + Actions 发布**、网页提供下载——是否同意？  
8. **二期**（自启、签名、MSI、自动更新）是否全部排除在首版外？  

---

## 10. 建议实施切片（确认边界后再动代码）

| 切片 | 内容 |
|------|------|
| S1 | Semaphore：launch token API + 设备按钮 + 未安装提示文案 |
| S2 | Helper MVP：环境配置、连接/断开、打开网页、协议、mstsc、日志 |
| S3 | ControlMaster 复用 + 直连探测 + 文档（操作手册） |
| S4 | Actions 出 exe + 下载链接接入 |

---

## 11. 非目标回顾（避免范围膨胀）

本方案 **不是**：

- 通用堡垒机 / PAM  
- 替代 WinRM 控制台  
- 替代设备 Discovery / Patrol  
- 跨平台远程桌面客户端  

本方案 **是**：

- 在「SSH 进网 + Semaphore 管设备」前提下，给 Windows 操作员一条 **可复用跳板的原生 RDP 快捷路径**。
