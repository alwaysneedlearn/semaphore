# Cloud Agent 交接文档

> 生成目的：供新建 Cloud Agent 快速接手本仓库（Semaphore UI + `cursor-playbooks`）的上下文。  
> 基准分支：**`develop`**（截至 2026-06 会话末，`30466361` 前后）。  
> Runner 上 playbook 根目录常为 **`/root/playbook`**（与仓库 `cursor-playbooks/` 同步）。

---

## 1. 仓库与运行环境

| 项 | 说明 |
|----|------|
| 产品 | **Semaphore UI**：Go 后端 + Vue 2 前端，单二进制 `./bin/semaphore` |
| Playbook 布局 | `cursor-playbooks/{shared,neware,land,sinexcel,nbt}/` + 根目录 `device_discovery.yml` |
| 构建 | **go-task**（`Taskfile.yml`），非 Make |
| 本地 dev | SQLite：`config.json` + `./bin/semaphore server --config ./config.json`（`admin` / `changeme`） |
| Cloud Agent 必读 | 根目录 **`AGENTS.md`**（WinRM、build 顺序、`task` 命令、设备 API 摘要） |

**VM 启动后依赖刷新**（update script 层）：`task deps`（Go + Node + `go-task` 等）。服务启动不在 update script 内。

---

## 2. 设备类型与操作实现（速查）

| 类型 | 管控对象 | 停止 | 启动 | 改配置 | 状态检测 | 重部署 | 数据重传 |
|------|----------|------|------|--------|----------|--------|----------|
| **NEWARE** | GUI `uu.exe` | WinRM 强杀脚本 | 计划任务启动 + 启动验证 | **INI** 写用户目录 | 进程 + `POST :api_port` `ExecResultData==3` 或日志关键字 | zip + INI + 验证 | 无（INI 字段） |
| **LAND** | GUI `LHBTS.exe` | 优雅停止（关窗+弹窗） | 计划任务 | **HTTP** `ModifyConfig`（进程在跑时） | exe+进程+`QueryStatus`（`real_status==true`） | zip + API 启停 | **`Redeliver`**（仅 restart） |
| **SINEXCEL** | GUI agent | 强杀或优雅 | 计划任务 | **HTTP** Kafka API（无 INI） | `QueryConfig` | zip + API | **`Retransmit`**（restart） |
| **NBT** | Windows **服务** | `win_service` stop | `win_service` start | 无 INI | 服务 Running + `SendStatus` 心跳 | zip + 服务重启 | **`ResetData`**（restart） |
| **DAHUA** | `lims-hist.exe` | — | — | — | —（不支持） | — | **仅** WinRM 跑 lims-hist `-from/-to` |
| **LANH** | GUI `ccsmon.exe` | —（不停止） | 未运行则交互启动（无弹窗） | — | 进程 Running（status/check_restart） | — | **Redeliver**（同 LAND） |

**已无独立 `device_start`**：启动合并在 restart/redeploy 流程。

共享层：`shared/tasks/winrm_*.yml`、`deploy_sem_windows_helper_scripts.yml`、`reconfig_start_program_windows.yml`、`stop_program_close_main_window_confirm.yml`。

详细说明见各目录 `README.md` 与会话中整理的「五类操作对照表」。

---

## 3. 本会话已合并到 `develop` 的修复（按主题）

### 3.1 LAND

| 提交/主题 | 内容 |
|-----------|------|
| `8bd0bba9` | **QueryStatus**：健康需 HTTP 2xx 且 `data.real_status==true`（`LAND_QUERY_STATUS_REQUIRE_REAL_STATUS`，默认 true） |
| `2b261991` | **ModifyConfig（停止前）**：检查 `_land_process_before`，不再误用 `_land_process_before_restart` |
| `a8193b97` | **force-stop**：LAND restart 等补 `sem_files_dir`；部署检查完整 ps1 列表；force fallback 补部署 + `powershell -File` |
| `ed6419e6` | **deploy**：去掉 `rejectattr('skipped')`（旧 Ansible 报 `dict has no attribute skipped`） |
| `9d8c3426` | **force fallback YAML**：`set_fact` `_stop_needs_force_fallback`，避免列表 `when:` 里写 `'NOT_RUNNING' not in` 解析失败 |
| 轮询 | 启动后 `land_api_query_status_start_poll.yml`（`LAND_API_STATUS_START_POLL_*`） |

**排障**：日志应有 `[DEBUG-STOP] STOP_GRACEFUL_CONFIRM_REV=3`、`[DEBUG-LAND]`、`[DEBUG-DEPLOY]`。

### 3.2 NEWARE

| 提交/主题 | 内容 |
|-----------|------|
| `739e34e4` | **`RECONFIG_CONFIG_FALLBACK_USE`** 作为 **`RECONFIG_CONFIG_FALLBACK_USERS`** 别名（用户常配错变量名） |
| `6d8d3fa8` | **启动验证**：`neware_query_upload_status_api_start_poll.yml`，轮询至 `ExecResultData==3`（默认与 `LOG_POLL_*` 对齐） |
| 配置用户 | `SERVICE_NAME`≠服务名；回退列表 `RECONFIG_CONFIG_FALLBACK_USERS`；exe 路径在 `Documents\NEWARE\BTSClient\...` |

**启动验证失败典型原因**：进程已起但 API `-1`（端口/未监听）或日志关键字未出现 → 查 `API_PORT`、增大 `API_STATUS_START_POLL_*` / `LOG_POLL_*`。

### 3.3 TDengine / 巡检

| 提交/主题 | 内容 |
|-----------|------|
| `96ffb69f` | **`TDENGINE_TIMEZONE`** 默认 `Asia/Shanghai`（`updated_time` / `check_time`） |
| `24c58b46` | **`device_status` trim**，避免 YAML `>-` 空白导致误 `offline` |

文档：`docs/tdengine-setup.md`。

### 3.4 NBT（本会话重点）

**变量语义（当前约定）**：

```env
SERVICE_NAME=NBT.MES.Service          # exe 文件名 → NBT.MES.Service.exe
NBT_SERVICE_NAME=NBTMESService        # Windows 服务名（Get-Service / 启停）
SERVICE_PATH=D:\MES                   # 父目录，在其下查找 exe
NBT_SERVICE_SCAN_MAX_DEPTH=3          # 可加大
```

| 提交/主题 | 内容 |
|-----------|------|
| `e8ea14d0` | `SERVICE_PATH` 只填目录；按 exe 名解析路径 |
| `38877139` | 解析脚本改为 **`nbt/files/sem_resolve_nbt_service_install.ps1`** + `powershell -File`（避免 inline `{ }` 触发 Jinja 错误） |
| `a9d05863` | **拆分** `SERVICE_NAME`=exe、`NBT_SERVICE_NAME`=服务名；空 `install_path` 不 `win_stat` |
| `30466361` | 默认与文档：exe `NBT.MES.Service`，服务 `NBTMESService` |

解析顺序：`{PATH}\{exe}` → `{PATH}\{exe主干}\{exe}` → `{PATH}\{NBT_SERVICE_NAME}\{exe}` → 递归扫描。

任务：`nbt/tasks/resolve_nbt_service_install.yml`、`nbt_stat_service_exe.yml`。

### 3.5 SINEXCEL / 其他（会话前期，已在 develop）

- 启动后轮询 `sinexcel_kafka_query_config_start_poll.yml`（勿在 `include_tasks` 上用 `until`）
- 优雅停止 YAML、`FORCE_STOPPED` 误判等（见 `STOP_GRACEFUL_CONFIRM_REV=3`）
- **`device_start` 已删除**（Go + 前端 + migration），仅保留 restart/redeploy 路径

---

## 4. 常见 Runner / Ansible 踩坑

1. **代码未同步**：日志路径仍是 `neware/tasks/winrm_...` 或缺少 `STOP_GRACEFUL_CONFIRM_REV=3` → runner 上 `git pull origin develop`。
2. **旧 Ansible**：  
   - 列表 `when:` 勿写 `'NOT_RUNNING' not in (...)` → 用 `when: >` 或 `set_fact`。  
   - `rejectattr('skipped')` 在 loop 结果上会炸 → 用 `selectattr('stat', 'defined')`。  
   - `until`/`retries` 挂在 **`uri`/`win_shell`** 上，不要挂在 `include_tasks`。
3. **WinRM 辅助脚本**：`deploy_sem_windows_helper_scripts.yml` 检查多文件；play 必须设 **`sem_files_dir`**（LAND restart 曾遗漏）。
4. **Vue/HTML 改动后**：`task build:fe` → `task build:be` → 重启 `./bin/semaphore`（资源嵌入二进制）。
5. **前端 lint**：`task lint:fe` 有历史错误，非环境问题。

---

## 5. 关键文件索引

```
cursor-playbooks/
  shared/tasks/
    deploy_sem_windows_helper_scripts.yml
    stop_program_close_main_window_confirm.yml
    stop_program_force_fallback.yml
    land_config_stop_start.yml
    land_api_query_status_apply.yml
    land_api_query_status_start_poll.yml
    reconfig_start_program_windows.yml
  neware/tasks/
    reconfig_prepare_profile_user.yml
    neware_query_upload_status_api_start_poll.yml
    start_verify_after_reconfig.yml
  nbt/
    tasks/resolve_nbt_service_install.yml
    files/sem_resolve_nbt_service_install.ps1
    vars/service_layout.yml
  VARIABLE_GROUPS.md
  land/README.md  neware/README.md  sinexcel/README.md  nbt/README.md
AGENTS.md
docs/tdengine-setup.md
docs/cloud-agent-handover.md   # 本文档
```

---

## 6. 新建 Cloud Agent 建议起手式

1. `git checkout develop && git pull origin develop`
2. 读 **`AGENTS.md`** + 本文件 + 目标设备类型 `README.md`
3. `task deps`（若未跑过 update script）
4. 改 playbook 后：`ansible-playbook --syntax-check cursor-playbooks/<type>/device_*.yml`
5. 涉及 UI：`task build:fe && task build:be`，重启服务做 hello-world（登录 → Devices → 触发对应 action）
6. 提交约定：分支 `cursor/<desc>-fc24`，合并 **`develop`**，删多余远程分支（用户偏好）

---

## 7. 未决 / 需人工确认项

| 项 | 说明 |
|----|------|
| 仓库改私有 | Git 操作不变，需凭据与协作者权限；Cursor Cloud / Semaphore runner token 需能访问私有库 |
| 部分主机 exe 仍 `not_found` | 核对 `SERVICE_PATH` 深度、`NBT_SERVICE_SCAN_MAX_DEPTH`、目标机实际目录 |
| NEWARE 启动验证 | 确认 `API_PORT` 与 INI `ReportApiSettings.ServerPort` 一致 |

---

## 8. 相关 PR / 分支（会话内）

功能均已 fast-forward 进 **`develop`**；会话中使用的临时分支已删除，例如：

- `cursor/land-force-stop-helper-fc24`
- `cursor/land-deploy-jinja-fix-fc24`
- `cursor/nbt-service-path-resolve-fc24`
- 等

新 Agent **从 `develop` 拉取即可**，无需再找已删分支。

---

*文档由 Cloud Agent 根据 2026-06 会话整理。有冲突时以 `develop` 上代码与 `AGENTS.md` 为准。*
