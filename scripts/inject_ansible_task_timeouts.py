#!/usr/bin/env python3
"""Inject Ansible task-keyword timeout: on win_*/uri module tasks missing one.

Keeps module-level uri timeout and retries/delay. Quotes Jinja values.
Inserts AFTER the full multi-line value of the last same-indent task key.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "cursor-playbooks"

MOD_RE = re.compile(
    r"^(\s+)((?:ansible\.windows\.|ansible\.builtin\.win_|community\.windows\.)[\w.]+"
    r"|(?:ansible\.builtin\.)?uri):\s*(.*)$"
)

EXPR = {
    "ping": '"{{ winrm_task_timeout_sec | default(90) | int }}"',
    "quick": '"{{ winrm_shell_quick_timeout_sec | default(120) | int }}"',
    "service": '"{{ winrm_shell_quick_timeout_sec | default(120) | int }}"',
    "copy_small": '"{{ winrm_copy_task_timeout_sec | default(600) | int }}"',
    "copy_large": '"{{ winrm_copy_large_timeout_sec | default(7200) | int }}"',
    "long": '"{{ winrm_shell_long_timeout_sec | default(900) | int }}"',
    "long_poll": '"{{ winrm_shell_long_timeout_sec | default(900) | int }}"',
    "uri": '"{{ uri_task_timeout_sec | default(60) | int }}"',
    "uri_poll": '"{{ uri_poll_task_timeout_sec | default(300) | int }}"',
}


def classify(path: Path, mod: str, has_retries: bool, block_text: str) -> str:
    p = str(path).replace("\\", "/").lower()
    if "uri" in mod:
        return "uri_poll" if has_retries else "uri"
    if "win_ping" in mod:
        return "ping"
    # Explicit long WinRM plugin timeouts in-task → use large ceiling
    if re.search(r"ansible_winrm_(?:operation|read)_timeout_sec:\s*[23]\d{3}", block_text):
        return "copy_large"
    if "win_copy" in mod:
        if any(x in p for x in ("redeploy", "zip", "package", "installer")):
            return "copy_large"
        return "copy_small"
    if "win_stat" in mod or "win_file" in mod:
        return "quick"
    if "win_service" in mod:
        return "service"
    if has_retries:
        return "long_poll"
    if any(
        x in p
        for x in (
            "redeploy",
            "gui_install",
            "backup_extract",
            "close_main",
            "graceful",
            "start_program",
            "start_verify",
            "reconfig_start",
            "ensure_running",
            "config_stop",
            "stop_program",
            "force_fallback",
        )
    ):
        return "long"
    return "quick"


def insert_index_after_last_task_key(block: list[str], indent_len: int, task_keys: dict[str, int]) -> int:
    """Return index in block at which to insert timeout (after last key's full value)."""
    if not task_keys:
        return len(block)
    idx = max(task_keys.values()) + 1
    while idx < len(block):
        raw = block[idx].rstrip("\n")
        if not raw.strip() or raw.strip().startswith("#"):
            idx += 1
            continue
        lead = len(raw) - len(raw.lstrip(" "))
        if lead > indent_len:
            idx += 1
            continue
        break
    return idx


def process_file(path: Path) -> int:
    lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
    out: list[str] = []
    i = 0
    added = 0
    while i < len(lines):
        line = lines[i]
        m = MOD_RE.match(line.rstrip("\n"))
        if not m:
            out.append(line)
            i += 1
            continue

        indent = m.group(1)
        indent_len = len(indent)
        mod = m.group(2)
        block = [line]
        j = i + 1
        task_keys: dict[str, int] = {}
        while j < len(lines):
            ln = lines[j]
            raw = ln.rstrip("\n")
            if not raw.strip() or raw.strip().startswith("#"):
                block.append(ln)
                j += 1
                continue
            lead = len(raw) - len(raw.lstrip(" "))
            if raw.lstrip().startswith("- ") and lead <= indent_len:
                break
            if lead < indent_len:
                break
            block.append(ln)
            if lead == indent_len:
                km = re.match(r"\s+([\w]+):", raw)
                if km:
                    task_keys[km.group(1)] = len(block) - 1
            j += 1

        if "timeout" in task_keys:
            out.extend(block)
            i = j
            continue

        has_retries = "retries" in task_keys or "until" in task_keys
        block_text = "".join(block)
        tier = classify(path, mod, has_retries, block_text)
        expr = EXPR[tier]
        insert_at = insert_index_after_last_task_key(block, indent_len, task_keys)
        timeout_line = f"{indent}timeout: {expr}\n"
        block.insert(insert_at, timeout_line)
        added += 1
        out.extend(block)
        i = j

    if added:
        path.write_text("".join(out), encoding="utf-8")
    return added


def main() -> int:
    total = 0
    files = 0
    for path in sorted(ROOT.rglob("*.yml")):
        if "group_vars" in path.parts:
            continue
        n = process_file(path)
        if n:
            files += 1
            total += n
            print(f"+{n:3d}  {path.relative_to(ROOT.parent)}")
    print(f"injected {total} timeouts across {files} files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
