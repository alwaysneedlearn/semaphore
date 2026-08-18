# Plan: split_device_redeploy

> **状态：已完成**（`develop`，2026-06）— DB/API/UI/playbook 均已落地；`check_restart_redeploy_template_id` 已删除；四类设备类型均有 `device_redeploy.yml` / `device_check_restart.yml`。
>
> **历史说明**：本文保留了方案设计阶段的对比内容，因此会出现 `device_start.yml`、`device_stop.yml`、`device_check_restart_redeploy.yml`、`check_restart_redeploy_template_id` 等旧名；请以文首状态说明和当前仓库文件为准。

> 设备操作拆分：独立「重新部署」，启动/重启/巡检重启均不做制品下发；**启动每次写配置**；**重新部署先查目标机压缩包，不存在再复制**。
>
> **原则：不做兼容层。** 直接按目标模型改 DB、API、UI、playbook；删除旧字段与旧 playbook 名。

---

## 1. 目标操作模型（6 类）

| 操作 | Playbook | 职责 |
|------|----------|------|
| 巡检 | `device_status.yml` | 仅探测 + 回调；**无** zip、**无** 写配置 |
| 启动 | `device_start.yml` | **每次**应用配置 → 启动 → 验证；**无** zip；无 exe 则 unhealthy，提示先「重新部署」 |
| 重启 | `device_restart.yml` | 停 → **写配置** → 启 → 验证；**无** zip |
| **重新部署** | **`device_redeploy.yml`（新）** | 目标机 zip 检查 → 缺则复制 → 解压（如需）→ 写配置 → 停 → 启 → 验证 |
| 巡检重启 | **`device_check_restart.yml`**（由 `device_check_restart_redeploy.yml` **重命名**） | 健康门禁；不健康仅走 **restart** 路径；**无** zip、**无** redeploy |
| 停止 | `device_stop.yml` | 保持不变 |

### 1.1 与旧行为的差异（刻意打破兼容）

| 项 | 旧 | 新 |
|----|----|-----|
| `check_restart_redeploy_template_id` | 定时巡检用「巡检重启重配」模板 | **删除**；改为 `check_restart_template_id` |
| `device_check_restart_redeploy.yml` | 4 类型均有 | **删除**；仅保留 `device_check_restart.yml` |
| NEWARE `device_start` | 健康则跳过；不健康才 INI+zip | **每次** INI；**从不** zip |
| `device_restart` | 常含 zip（exe 缺失时） | **从不** zip；exe 缺失则失败并提示 redeploy |
| 设备列表动作 | start/stop/restart/status | 增加 **`redeploy`** |
| Scheduler | 优先 `check_restart_redeploy_template_id` | 仅用 **`check_restart_template_id`**（未配则回退 `status_template_id`） |

**不保留：** 转发桩 playbook、`check_restart_redeploy_template_id` 双读、旧模板 ID 自动迁移脚本。

---

## 2. 配置与 zip 规则（核心）

### 2.1 启动（`device_start`）— 每次写配置

所有类型在 **启动路径** 中，只要 `merged_cfg` / 环境变量表明有配置，**无条件**执行配置写入（与进程是否在跑、健康门禁无关）：

| 类型 | 配置方式 | 顺序要点 |
|------|----------|----------|
| **NEWARE** | `reconfig_copy_and_patch_config.yml`（INI） | 若在跑：先 graceful stop → 写 INI → `reconfig_start_program_windows` → `start_verify_after_reconfig` |
| **LAND** | `land_api_modify_config_before_stop` + `after_start_retry` | 与现 restart 类似，但 **去掉** zip/解压块 |
| **SINEXCEL / NBT** | `apply_neware_style_device_config_files.yml` | 写 INI 后再启动 |

**启动不做：**

- `win_copy` 安装包
- `Expand-Archive`
- `need_reconfigure` 门控下的「重配大块」（健康时跳过配置）

**启动前置：**

- `win_stat` exe；不存在 → `device_status: unhealthy`，`abnormal_reason` 含「请先执行重新部署」，**不** 尝试复制 zip。

### 2.2 重启（`device_restart`）— 写配置，无 zip

与启动相同的 **配置任务**，固定顺序：**停 → 配置 → 启 → 验证**。

- NEWARE：去掉整块 zip 检查/复制/解压。
- LAND：保留 Redeliver（`land_api_redeliver.yml`）在 restart 末尾（**不归入 redeploy**）。
- NBT：保留 `ResetData`（restart 健康门禁通过后，**不归入 redeploy**）。

exe 缺失 → 失败/回调 unhealthy，提示 redeploy。

### 2.3 重新部署（`device_redeploy`）— zip 先查目标机

**仅 redeploy** 允许制品操作。统一抽取共享任务：

`cursor-playbooks/shared/tasks/redeploy_ensure_package_windows.yml`

```yaml
# 逻辑（伪代码）
1. win_shell: Test-Path "{{ exe_dir }}\{{ zip_name }}.zip"
2. when ZIP_NOT_FOUND:
     win_copy: controller {{ zip_path }}/{{ zip_name }}.zip → dest
3. when ZIP_FOUND:
     debug: skip copy
4. when exe 不存在且 zip 在目标机:
     Expand-Archive → {{ exe_dir }}\{{ zip_name }}
5. win_stat exe → 仍不存在则 fail
```

要点：

- **先查目标机** `{{ exe_dir }}\{{ zip_name }}.zip`，存在则 **跳过** `win_copy`。
- 复制源仍为控制器 `ZIP_PATH` / `zip_path`（与现 NEWARE 重配块一致）。
- 解压：exe 目录不存在或 exe 缺失时解压；已解压且 exe 在则跳过。
- 之后：**写配置 → stop（若在跑）→ start → verify**（与各类型 restart 后半段共用 task）。

**redeploy 不做** 健康/API 预检跳过整条链路（与 restart 一致：总是走完部署链）。

### 2.4 巡检重启（`device_check_restart`）

1. 复用 `device_status` 探测逻辑（API/日志/进程）。
2. 健康 → 仅回调 healthy，**结束**。
3. 不健康 → `include_tasks: device_restart_core.yml`（或等价的 restart 子流程），**禁止** include redeploy / zip 任务。

---

## 3. Playbook 文件变更

### 3.1 新增（×4 类型 + shared）

```
cursor-playbooks/shared/tasks/
  redeploy_ensure_package_windows.yml   # zip 查目标 → 条件复制 → 条件解压
  restart_core_after_config_windows.yml # 停+启+验证（类型可传 sem_profile_tasks_dir）
  apply_device_config_<vendor>.yml      # 可选：按类型薄封装，避免 start/restart/redeploy 三份粘贴

cursor-playbooks/{neware,land,sinexcel,nbt}/
  device_redeploy.yml                   # 新
  device_check_restart.yml              # 由 check_restart_redeploy 改名+瘦身
```

### 3.2 删除

```
cursor-playbooks/{neware,land,sinexcel,nbt}/device_check_restart_redeploy.yml
```

### 3.3 修改

| 文件 | 变更 |
|------|------|
| `device_start.yml` | 移除 zip、移除 `need_reconfigure` 大块；**始终**配置+启动；无 exe 报错 |
| `device_restart.yml` | 移除 zip；保留类型特有 API（Redeliver/ResetData） |
| `device_status.yml` | 仅探测；`need_reconfigure` 可改为 UI 提示字段，**不触发写配置** |
| `device_check_restart.yml` | 门禁 + restart only |

### 3.4 共享调试任务

- `debug_sync_api_redeploy_gate_snapshot.yml` → 仅 redeploy / check_restart 不健康分支使用；start/restart 改用 `debug_sync_api_start_snapshot.yml`（已有）。

---

## 4. 后端（破坏性迁移）

### 4.1 `db/Device.go`

```go
DeviceActionRedeploy DeviceAction = "redeploy"
```

`TemplateIDForAction` / `DeviceProfile.TemplateIDForAction` 增加 `redeploy` → `RedeployTemplateID`。

### 4.2 `db/DeviceProfile.go` + SQL

**删除列：** `check_restart_redeploy_template_id`

**新增列：**

| 列 | 说明 |
|----|------|
| `redeploy_template_id` | 手动「重新部署」+ API `action=redeploy` |
| `check_restart_template_id` | 定时巡检不健康时跑的 playbook |

迁移文件：`db/sql/migrations/v2.18.23.sql`（示例）

```sql
alter table project__device_profile_settings
  drop column check_restart_redeploy_template_id;

alter table project__device_profile_settings
  add column redeploy_template_id int null,
  add column check_restart_template_id int null;
```

同步 `db/sql/device_profile.go`、`db/Migration.go`。

### 4.3 `api/projects/devices.go`

- Bulk/single action 白名单增加 `redeploy`。
- `redeploy` / `start` / `restart` 注入 `config` / `configs_by_host`（与现 start/restart 相同）。
- `patrol` 仍只用 `status_template_id`。

### 4.4 `services/server/device_status_scheduler.go`

```go
templateID := ps.StatusTemplateID
if ps.CheckRestartTemplateID != nil && *ps.CheckRestartTemplateID > 0 {
    templateID = *ps.CheckRestartTemplateID
}
```

删除对 `CheckRestartRedeployTemplateID` 的引用。

---

## 5. 前端

### 5.1 `web/src/views/project/Devices.vue`

- 行菜单 / 批量菜单增加 **重新部署** → `action: 'redeploy'`。
- i18n：`deviceRedeploy` / `deviceRedeployConfirm`（中英）。

### 5.2 `web/src/components/DeviceProfileSettingsEditor.vue`

**Playbook templates 列表：**

```
start_template_id
stop_template_id
restart_template_id
redeploy_template_id          # 新
status_template_id
```

**定时巡检卡片：**

- 删除 `check_restart_redeploy_template_id` 表单项。
- 新增 `check_restart_template_id`（「巡检重启模板」；未选则回退 status 模板——**仅 UI 文案说明**，后端 scheduler 逻辑见上）。

`TEMPLATE_ID_FIELDS` / normalize / save 同步更新。

---

## 6. 文档与 Semaphore 模板绑定

| 文档 | 内容 |
|------|------|
| `cursor-playbooks/README.md` | 六类操作表、zip 规则、迁移清单 |
| `AGENTS.md` | `redeploy` action、`check_restart_template_id`、启动必写配置 |
| `.claude/skills/...`（如有 device playbook skill） | 同步操作分类 |

**运维迁移（人工，无自动脚本）：**

1. 每个 Device Profile 新建 Semaphore 模板指向 `device_redeploy.yml`、`device_check_restart.yml`。
2. Profile 设置里绑定 `redeploy_template_id`、`check_restart_template_id`。
3. 删除对 `device_check_restart_redeploy.yml` 的模板引用。
4. Runner 更新 playbook 仓库后 `git pull`。

---

## 7. 实施顺序

1. **Shared tasks** — `redeploy_ensure_package_windows.yml`（目标机 zip 检查 + 条件复制）、配置/重启核心块。
2. **4 × `device_redeploy.yml`** — 从现 start/restart 迁出 zip+解压+配置块。
3. **瘦身 `device_start` / `device_restart`** — 启动每次配置；去掉 zip。
4. **重命名 `device_check_restart.yml`** — 删除旧文件；门禁仅 restart。
5. **DB 迁移 + API + Scheduler** — 破坏性列替换。
6. **前端** — redeploy 菜单 + 模板下拉。
7. **文档** + 本地 `task build` / `task test:be` / 冒烟（start、redeploy、check_restart）。

---

## 8. 验收标准

- [x] `device_start`：进程在跑且日志健康时仍执行配置写入并重启验证（或 LAND API 修改）。
- [x] `device_start`：无 exe 时不复制 zip，回调提示 redeploy。
- [x] `device_redeploy`：目标机已有 zip 时不 `win_copy`；无 zip 时复制并解压。
- [x] `device_restart` / `device_check_restart`：日志与任务中无 `win_copy` zip。
- [x] API `POST .../actions/bulk` `action=redeploy` 可调度模板。
- [x] DB 无 `check_restart_redeploy_template_id`；Scheduler 使用 `check_restart_template_id`。
- [x] 四类 NEWARE/LAND/SINEXCEL/NBT playbook 均已对齐。

---

## 9. 不在本计划内

- 自动把旧 `check_restart_redeploy_template_id` 复制到新列（用户手动重绑模板）。
- `device_check_restart_redeploy.yml` 转发桩。
- 修改 NEWARE API/日志健康判定算法本身（仅调整在哪些 playbook 里调用）。
