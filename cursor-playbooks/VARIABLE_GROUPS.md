# Variable Groups（变量组）配置指南

Semaphore **变量组**通过模板绑定的 **Environment** 注入 Ansible `lookup('env', 'VAR')`。规则：**环境变量非空（trim 后）优先**，否则用 playbook 内默认值。

## 基本原则

| 原则 | 说明 |
|------|------|
| **一类型一组** | LAND / SINEXCEL / NBT / NEWARE 各建 **独立变量组**，绑定到该类型的 Status / Start / Stop / Restart / Redeploy 模板。不要混用跨类型默认值。 |
| **全项目默认放 VG** | 如 `EXE_NAME`、`ZIP_NAME`、`API_PORT`、弹窗关键字、盘符回退顺序。 |
| **每台机不同放设备配置** | 安装路径用设备 **配置** 的 **`Install`** / **`Paths`**（`ExeDir`、`ExePath`、`AppDir`…），不要写进 VG。见 [`sinexcel/README.md`](sinexcel/README.md#install--paths-keys-per-device-or-type-default)。 |
| **回调 Token** | `SEMAPHORE_API_TOKEN`（必填才能 bulk 回调）；可选 `SEMAPHORE_URL`（默认 `http://127.0.0.1:3000`）。 |

## `EXE_NAME` 要不要加 `.exe`？

**推荐：变量组里写带 `.exe` 的完整文件名**（与磁盘上的 exe 一致）。

| 变量 | 格式 | 示例 |
|------|------|------|
| **`EXE_NAME`** | **带 `.exe`**（推荐） | `LHBTS.exe`、`sinexcel_agent.exe`、`nbt_agent.exe`、`uu.exe` |
| **`PROCESS_NAME`** | **不带 `.exe`**（可选） | `LHBTS`、`sinexcel_agent`、`uu` |

Playbook 会统一规范化：

1. 去掉已有 `.exe` 后缀得到 base name  
2. **`exe_name`** = `base + '.exe'`（拼路径、扫描、存在性检查）  
3. **`process_name`** = base（`Get-Process`、优雅停止）

因此 **旧配置仍兼容**：

- NEWARE 曾用 `EXE_NAME=uu`（无后缀）→ 仍解析为 `uu.exe` / 进程 `uu`  
- 若写成 `EXE_NAME=uu.exe` → 同样正确，**不会**变成 `uu.exe.exe`

**不要**在 VG 里同时改 `EXE_NAME` 和错误的 `PROCESS_NAME`（如 SINEXCEL 写成 `LHBTS`）；未设置 `PROCESS_NAME` 时自动从 `EXE_NAME` 推导。

## 各设备类型变量组示例

### LAND

```env
EXE_NAME=LHBTS.exe
APP_DIR=LHBTS
EXE_DIR=C:\Program Files\LAND
EXE_DIR_FALLBACK_DRIVES=D,F,C
ZIP_NAME=LHBTS5.1.4.4
ZIP_PATH=/root/land/packages
LAND_API_TOKEN=landapi
STOP_POPUP_KEYWORD=警告
SEMAPHORE_API_TOKEN=<token>
```

路径布局：`{{ EXE_DIR }}\{{ APP_DIR }}\{{ EXE_NAME }}`。详见 [`land/README.md`](land/README.md)。

### SINEXCEL

```env
EXE_NAME=sinexcel_agent.exe
APP_DIR=sinexcel
EXE_DIR=C:\Program Files\SINEXCEL
EXE_DIR_FALLBACK_DRIVES=D,E,C
EXE_SCAN_LATEST=true
EXE_SCAN_MAX_DEPTH=2
ZIP_NAME=sinexcel
API_PORT=9002
START_POPUP_KEYWORD=提示
STOP_POPUP_KEYWORD=警告
STOP_POPUP_MATCH_MODE=title_or_content
SEMAPHORE_API_TOKEN=<token>
```

每台安装目录不同 → 设备配置 **`Install.ExeDir`** / **`Install.ExePath`**，不要写进 VG。详见 [`sinexcel/README.md`](sinexcel/README.md)。

### NBT

```env
EXE_NAME=nbt_agent.exe
ZIP_NAME=nbt
EXE_DIR=C:\Program Files\NBT
EXE_DIR_FALLBACK_DRIVES=D,E,C
CONFIG_FILE_NAME=nbt.iconf
API_PORT=9002
SEMAPHORE_API_TOKEN=<token>
```

路径：`{{ EXE_DIR }}\{{ ZIP_NAME }}\{{ EXE_NAME }}`。详见 [`nbt/README.md`](nbt/README.md)。

### NEWARE

```env
EXE_NAME=uu.exe
ZIP_NAME=test
EXE_DIR=D:\Program Files\NEWARE
ZIP_PATH=/root/neware/dbwb
CONFIG_FILE_NAME=NWReport_DBWB.iconf
API_PORT=9002
API_STATUS_CALL_TYPE=1
SEMAPHORE_API_TOKEN=<token>
```

路径：`{{ EXE_DIR }}\{{ ZIP_NAME }}\{{ EXE_NAME }}`（与 NBT 相同，`EXE_NAME` 已含 `.exe`）。详见 [`neware/README.md`](neware/README.md)。

## Stop 模板注意事项

| 类型 | 说明 |
|------|------|
| LAND / SINEXCEL | `STOP_GRACEFUL_PROCESS_NAME` 默认 = 从 `EXE_NAME` 推导的 `process_name`，**不要**填主程序别名（如 `LHBTS` 当实际进程是 `sinexcel_agent`） |
| SINEXCEL | `STOP_POPUP_MATCH_MODE=title_or_content` 用于无标题弹窗 |
| NEWARE | 强制结束进程；`EXE_NAME` 规范化后传给辅助脚本时会去掉 `.exe` 再查 `Get-Process` |

## 不要放进变量组的内容

- 单台机器的 `ExeDir` / `ExePath`（用设备 **Install** 配置）  
- `semaphore_project_id`（Semaphore 任务 extra-vars 自动注入）  
- 每台不同的 `api_port`（用设备字段 `api_port` 或 `API_PORT` 默认）

## Playbook 内规范化片段

各 `device_*.yml` 使用同一 Jinja 块（仅 `_exe_name_default` 按类型不同）：

```yaml
_exe_name_default: sinexcel_agent.exe
_exe_name_env: "{{ (lookup('env', 'EXE_NAME') | default('', true) | trim) }}"
_exe_name_base: "{{ (_exe_name_env if (_exe_name_env | length > 0) else _exe_name_default) | regex_replace('(?i)\\.exe$', '') }}"
exe_name: "{{ _exe_name_base ~ '.exe' }}"
process_name: "{{ ((lookup('env', 'PROCESS_NAME') | default('', true) | trim) | regex_replace('(?i)\\.exe$', '')) | default(_exe_name_base, true) }}"
```

（YAML 双引号内 Jinja 正则须写成 `'(?i)\\.exe$'`，不能写 `'(?i)\.exe$'`，否则 Ansible 报 `unknown escape character`。）

参考：[`shared/vars/exe_name_from_env.yml`](shared/vars/exe_name_from_env.yml)。
