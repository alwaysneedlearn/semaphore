# Semaphore fork 改造总览

> 本文只保留**相对原始 Semaphore** 的改造点，供新 Cloud Agent、维护者和评审快速理解当前 fork。  
> 运行与构建细节以 `AGENTS.md` 为准；设备类型细节以各 `cursor-playbooks/<type>/README.md` 为准。

---

## 1. 这不是原始 Semaphore 的地方

本 fork 在原始 Semaphore 基础上新增/改造了以下能力：

- **Windows 设备管理体系**：设备列表、设备类型、Discovery、Probe、Remote Desktop、批量设备操作。
- **Ansible playbook 驱动的设备运维**：仓库内置 `cursor-playbooks/`，由 Semaphore 模板直接调用。
- **多设备类型支持**：NEWARE、LAND、SINEXCEL、NBT、JHAI、DAHUA、LANH、LANDV7。
- **设备状态回写**：playbook 通过 bulk callback 回写 `device_status / winrm_status / api_status`。
- **TDengine 集成**：bulk callback 后按设备类型写时序状态快照。
- **RDP Helper**：通过本地 Windows helper 承接 Web 端“远程桌面”操作并回传审计事件。
- **任务日志页面改造**：触发任务时仍用弹窗查看日志；弹窗可「新标签页打开」独立页 `/project/{id}/history/{taskId}`（无侧栏，顶栏模板名/任务名不可跳转）。

---

## 2. 当前设备侧 UI / 模板模型

### 设备列表当前动作

当前设备列表 UI 保留：

- `status`
- `resend_data`
- `restart`
- `redeploy`

不再提供：

- `stop` 设备列表入口
- `start` 独立入口
- `check_restart` 设备列表入口（仅通过 **Schedules** 跑 `device_check_restart.yml`）

### 设备类型配置

设备类型（Device types）按 profile 绑定模板，不再使用旧的 project-level device settings UI。  
每个设备类型通常只关心：

- `status_template_id`
- `restart_template_id`
- `redeploy_template_id`
- `resend_data_template_id`

---

## 3. 设备类型差异（与原始 Semaphore 无关、为本 fork 自增）

| 类型 | 核心对象 | 状态检测 | 重启/重部署特点 | 数据重传 |
|------|----------|----------|----------------|----------|
| **NEWARE** | GUI `uu.exe` | 进程 + upload-status API / 日志 | INI 改配 + 启动验证 | 通过配置字段 |
| **LAND** | GUI `LHBTS.exe` | `QueryStatus` | `ModifyConfig` + 优雅停止 + GUI 安装器 redeploy | `Redeliver` |
| **SINEXCEL** | GUI agent | `QueryConfig` | Kafka API 改配；默认优雅停止，不强杀 | `Retransmit` |
| **NBT** | Windows 服务 | 服务 + `SendStatus` | 服务重启 / zip redeploy | `ResetData` |
| **JHAI** | Windows 服务 + HTTP | 上传心跳 API | 服务重启 / zip redeploy | API resend |
| **DAHUA** | `CTSMonPro` | 进程 `Get-Process` | 无 | WinRM 跑 `lims-hist -from/-to` |
| **LANH** | GUI `ccsmon.exe` | `QueryStatus` | check_restart 仅 ensure 启动，不做完整 stop/start | `Redeliver` |
| **LANDV7** | GUI `ccsmon.exe`（同 LANH） | `QueryStatus` | 与 LANH 一致：check_restart 仅 ensure 启动 | `Redeliver` |

共同点：

- `hosts: windows_hosts` 的 playbook 统一走 `strategy: free`
- bulk callback 统一经 `shared/tasks/semaphore_bulk_put_from_hostvars.yml`
- Discovery 统一使用 `cursor-playbooks/device_discovery.yml`

---

## 4. 关键改造约定

### Playbook 布局

```text
cursor-playbooks/
  device_discovery.yml
  shared/
  neware/
  land/
  sinexcel/
  nbt/
  jhai/
  dahua/
  lanh/
  landv7/
```

### Runner 约定

- Runner 上 playbook checkout 常在 `/root/playbook`
- 代码改完但任务日志看不到新步骤，优先检查 runner 是否已 `git pull origin develop`

### 前端 / 二进制构建

本 fork 仍是单二进制，但前端资源嵌入后端二进制，因此：

1. 改 Vue/HTML 后先 `task build:fe`
2. 再 `task build:be`
3. 再重启 `./bin/semaphore`

---

## 5. 运维相关附加能力

### TDengine

- 不是原始 Semaphore 能力
- 由本 fork 在 bulk callback 后写入
- 仅对使用 `semaphore_bulk_put_from_hostvars.yml` 的设备 playbook 生效
- 运维说明见 `docs/tdengine-setup.md`

### RDP Helper

- 不是原始 Semaphore 能力
- 由 `cmd/rdp-helper/` 提供本地 Windows helper
- Web 端通过自定义协议触发本地 `mstsc`
- 支持 ssh 隧道、中转、启动/结束审计

---

## 6. 新 Agent 起手顺序

1. `git checkout develop && git pull origin develop`
2. 读 `AGENTS.md`
3. 读本文
4. 读目标设备类型的 `cursor-playbooks/<type>/README.md`
5. 改 playbook 后先做 `ansible-playbook --syntax-check cursor-playbooks/<type>/device_*.yml`
6. 改 UI 后做 `task build:fe && task build:be`

---

## 7. 关键文件

- `AGENTS.md`
- `cursor-playbooks/README.md`
- `cursor-playbooks/VARIABLE_GROUPS.md`
- `cursor-playbooks/<type>/README.md`
- `docs/tdengine-setup.md`
- `cmd/rdp-helper/README.md`
