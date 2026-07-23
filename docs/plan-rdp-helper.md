# 方案：Semaphore RDP Helper（原生远程桌面）

> 状态：**已实现 MVP**（API + 设备按钮 + `cmd/rdp-helper`；见 §10）  

> 目标：操作员经 SSH 跳板访问 Semaphore 所在网络时，用 **Windows 本地 Helper + mstsc** 打开设备远程桌面；**不做**浏览器内嵌 RDP。

---

## 1. 背景与网络假设

| 情况 | 说明 |
|------|------|
| Semaphore ↔ 设备 | **可直连**（含 RDP 3389） |
| 操作员 ↔ Semaphore / 设备 | 常需 **SSH 跳板**（0 / 1 / 2 跳）；本地打开 Semaphore 网页往往先做端口映射 |
| 操作员系统 | **Windows**（mstsc + OpenSSH 客户端） |
| 操作员权限 | **无管理员 / 非提权账号**（见 §8） |

浏览器内嵌 RDP（Guacamole / Devolutions / IronRDP）**本方案不采用**。

---

## 2. 方案一句话

**Helper 为入口**：本机放置并配置多环境跳板 → 连接环境（一条 SSH ControlMaster）→ 打开 Semaphore 网页 → 在设备列表点「远程桌面」→ 自定义协议拉起 Helper → 复用隧道做 `ssh -L` → 启动 `mstsc` → 结束后只清理本次转发。

Helper 由运维自行分发；Semaphore **只提示**协议拉起失败，**不提供下载链接**，**不新增**项目配置项，**不新增**该功能的权限与审计。

---

## 3. 功能边界

### 3.1 做（In Scope）

#### A. Windows Helper（`semaphore-rdp-helper.exe`）

| 能力 | MVP | 说明 |
|------|-----|------|
| 多环境配置 | ✅ | 按「厂区/项目」保存：名称、跳板 0/1/2、Semaphore 地址、本地 UI 端口等（不必与 Semaphore 项目自动同步） |
| 连接 / 断开环境 | ✅ | `ssh` ControlMaster；需要时 `-L` 映射 Semaphore UI |
| 打开网页 | ✅ | 连接成功后打开本机映射后的 Semaphore URL |
| 自定义协议 | ✅ | **当前用户**注册并处理 `semaphore-rdp://connect?token=...`（无需管理员，见 §8） |
| Token 换参 | ✅ | HTTPS 调 Semaphore API，用一次性 token 取连接参数 |
| 拉起 RDP | ✅ | 复用 master 增加 LocalForward → `mstsc`；退出后 `-O cancel` 该转发 |
| 直连探测 | ✅ | 本机可达设备 3389 则跳过 SSH，直接 mstsc |
| 本地日志 | ✅ | SSH / 协议 / API 错误可查 |
| 托盘 / 精简 UI | ✅ | 选环境、连接状态、打开网页、查看日志 |

#### B. Semaphore 服务端 / Web

| 能力 | MVP | 说明 |
|------|-----|------|
| 设备「远程桌面」按钮 | ✅ | 设备列表行操作（与 Probe / WinRM 并列） |
| 签发一次性 token | ✅ | 绑定当前登录用户 + 设备 + 短 TTL；URL **只带 token** |
| Launch API | ✅ | Helper 凭 token 获取目标 IP/端口、RDP 用户名；密码见下「凭据策略」 |
| 协议失败提示 | ✅ | **仅提示** Helper 未响应/拉起失败（文案引导自行安装配置 Helper） |
| 复用现有权限 | ✅ | 沿用现有项目资源权限即可打开设备页并点按钮；**不**为 RDP 单独加权限点 |
| 审计 | ❌ | **不**为该功能新增审计日志 |
| 下载链接 | ❌ | **不**在产品内提供 Helper 下载 |
| 项目配置项 | ❌ | **不**增加 Gateway URL / 开关 / 下载地址等项目设置 |

#### C. 跳板与复用

| 原则 | 说明 |
|------|------|
| 跳板只在 **Helper 本机** | Semaphore **不存**操作员私钥与跳板机密 |
| 一条 SSH 多用 | ControlMaster：UI 映射与多台 RDP LocalForward 共用 |
| 断开策略 | 关 RDP **不断**环境连接；断开环境才拆 master |
| 设备信息在 Semaphore | `ip` / `rdp_user` / `rdp_password` / `rdp_port` |

#### D. RDP 凭据策略（已确认）

| 情况 | 行为 |
|------|------|
| 设备在 Semaphore 中 **已配置** `rdp_password`（非空） | Launch API **短时下发**密码；Helper **仅内存使用**，不写进可分享 URL，尽量不落盘（或用完即删临时 `.rdp`） |
| 设备 **未配置** `rdp_password`（空） | API 只下发主机 / 端口 / 用户名；Helper 或 `mstsc` **本机输入**密码 |

### 3.2 不做（Out of Scope）

| 排除项 | 说明 |
|--------|------|
| 浏览器内嵌 RDP | Guacamole / IronRDP / Devolutions |
| Helper 内设备管理 | 不列表、不改配置、不跑任务 |
| Mac / Linux Helper | 仅 Windows |
| 服务端代持 SSH 私钥 | — |
| 打进主 `semaphore.exe` | 独立 artifact |
| 产品内下载 / 更新 / 签名 / MSI | 分发与升级走运维渠道 |
| 本功能专用权限与审计 | — |
| 本功能项目配置项 | — |
| 自动更新、开机自启等 | 二期可选 |

### 3.3 二期（可选）

- 托盘增强、断线重连、环境导入导出  
- 代码签名、MSI（若日后有管理员部署场景）  
- SOCKS 作为连网替代  

---

## 4. 端到端操作流程

### 4.1 一次性（操作员 PC，无管理员）

1. 运维下发绿色/`AppData` 目录下的 `semaphore-rdp-helper.exe`（**非** Program Files）  
2. 首次运行：Helper **写 HKCU** 注册 `semaphore-rdp://`（无需提权）  
3. 配置多环境跳板；确认用户级 OpenSSH 与密钥可用  

### 4.2 日常

```text
Helper：选环境 → 连接 → 打开网页 → 登录 Semaphore
Semaphore：设备列表 →「远程桌面」→ semaphore-rdp://?token=...
Helper：换参 →（复用 master / 直连）→ mstsc → 结束仅 cancel 本次 -L
```

### 4.3 异常

| 情况 | 行为 |
|------|------|
| 未装 / 协议未注册 | Semaphore **仅提示** Helper 拉起失败（无下载链接） |
| 未连环境 | Helper 提示先连接环境 |
| Token 过期 | 提示回网页重点 |
| SSH 失败 | Helper 本地错误 + 日志 |

---

## 5. 配置职责

| 位置 | 内容 |
|------|------|
| Helper | 环境、hops、UI 本地端口、映射后 Semaphore URL |
| Semaphore | 现有设备 `rdp_*`、现有登录态发 token（无新配置项） |
| 本机 SSH | 密钥 / agent / 可选 `~/.ssh/config` |

---

## 6. 协议与 API（草案）

```text
semaphore-rdp://connect?token=<one-time-token>

POST /api/project/{id}/devices/{device_id}/rdp/launch
  → { "token", "expires_in", "helper_url" }

GET  /api/rdp/launch-params?token=...
  → {
      host, rdp_port, rdp_user,
      rdp_password: string | null,   # 库中有则短时下发；空则 null → Helper/本机输入
      password_provided: bool,
      suggested_env?: string
    }
```

- URL 不带密码与主机机密  
- 权限：能访问该项目设备即可（现有中间件），无新 RBAC  
- `rdp_password` 仅经 **HTTPS + 一次性 token** 下发；Helper 不写日志明文  

---

## 7. 技术选型与发布

| 项 | 选择 |
|----|------|
| 语言 | Go，单文件 exe |
| SSH / RDP | 系统 `ssh` + `mstsc` |
| UI | 托盘 + 简单窗口 |
| CI | GitHub Actions 打 `windows/amd64` exe |
| 分发 | **运维自行分发**；产品内不挂下载链 |

---

## 8. 无管理员权限是否有问题？

**按本方案设计：一般没有问题**，前提是遵守下列约定：

| 动作 | 是否需要管理员 | 做法 |
|------|----------------|------|
| 注册 `semaphore-rdp://` | **否** | 写入 **HKCU**（`HKEY_CURRENT_USER\Software\Classes`），不要写 HKLM |
| 安装位置 | **否** | 放用户目录 / 绿色版；**不要**装到 `Program Files`（那要管理员） |
| `ssh -L` 高位本地端口 | **否** | 绑 `127.0.0.1` 随机/高端口即可 |
| 启动 `mstsc` | **否** | 普通用户即可 |
| 读用户 `~/.ssh`、ssh-agent | **否** | 当前用户权限即可 |

**可能踩坑（与是否管理员无关或弱相关）：**

- 公司策略禁止用户注册自定义协议 → 需 IT 预置 HKCU 或给策略例外  
- 未安装 **OpenSSH Client** 可选功能 → 需用户或 IT 启用（部分环境要管理员才能「添加可选功能」；若已装好则 Helper 无感）  
- SmartScreen 拦截未签名 exe → 用户可「仍要运行」，或运维侧签名/白名单（签名非 MVP）  
- 组策略禁用 mstsc → 非 Helper 能解决  

**结论：** Helper **按「当前用户、免提权」设计即可**；不要依赖安装服务、写 HKLM、装 Program Files。若 OpenSSH 都未装且用户无法添加可选功能，需运维预装 OpenSSH，这是环境前提，不是 Helper 必须提权。

**本机验证脚本（普通用户 PowerShell）：**

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test-rdp-helper-windows-preflight.ps1
```

脚本会测：AppData 可写、HKCU 协议注册、HKLM/Program Files（预期失败）、OpenSSH、本机高端口监听、`~/.ssh`、mstsc、自定义协议拉起。见仓库 `scripts/test-rdp-helper-windows-preflight.ps1`。

### 8.1 预检实机结果（参考）

在非管理员账号上跑通核心项后可认定环境满足 Helper 前提，例如：

- Elevation / AppData / HKCU 协议 / LocalPortBind / `.ssh` / OpenSSH / mstsc：**PASS**
- Program Files / HKLM 写入失败（脚本记为 PASS）：符合「只用用户目录 + HKCU」
- `ProtocolURLLaunch` 偶发 WARN：不阻断；以 HKCU 注册 PASS 为准，网页点击再验

---

## 9. 边界确认记录

| # | 项 | 结论 |
|---|-----|------|
| 1 | 仅 Windows Helper + mstsc，无浏览器内嵌 RDP | **是** |
| 2 | 跳板只配 Helper，服务端不持私钥 | **是** |
| 3 | Helper 为入口 | **是** |
| 4 | RDP 密码 | **有则短时下发；无则本机输入** |
| 5 | Helper 多环境即可，不必与项目自动同步 | **是** |
| 6 | 直连探测进 MVP | **是** |
| 7 | 独立 exe + Actions 构建 | **是**（分发不靠产品内下载链） |
| 8 | 自启/签名/MSI/自动更新放二期外 | **是** |
| — | Semaphore 仅提示 Helper 失败，**无下载链接** | **是** |
| — | **不**为该功能增加权限与审计 | **是** |
| — | **不**增加项目配置项 | **是** |
| — | 运行账号无管理员 | **可接受**（§8） |

---

## 10. 实施切片

| 切片 | 内容 | 状态 |
|------|------|------|
| S1 | Semaphore：launch token API + 设备按钮 + 失败提示（无下载链） | **已做** |
| S2 | Helper：环境 / 连接 / 开网页 / HKCU 协议 / mstsc / 日志 | **已做**（`cmd/rdp-helper`） |
| S3 | ControlMaster 复用 + 直连探测 + README | **已做** |
| S4 | Actions 出 exe（`.github/workflows/rdp_helper.yml`） | **已做** |

---

## 11. 非目标

不是堡垒机 / PAM，不替代 WinRM / Discovery / Patrol；只是 Windows 操作员在「SSH 进网 + Semaphore 管设备」下的 **原生 RDP 快捷路径**。
