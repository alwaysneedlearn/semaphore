#!/usr/bin/env python3
"""Fail if a multi-key set_fact (without loop) references another key from the same task."""

from __future__ import annotations

import re
import sys
from pathlib import Path


def find_issues(text: str, path: str) -> list[str]:
    issues: list[str] = []
    lines = text.splitlines()
    i = 0
    while i < len(lines):
        if not re.match(r"^\s*-\s+name:", lines[i]):
            i += 1
            continue
        task_line = i + 1
        task_name = lines[i].strip()
        j = i + 1
        has_loop = False
        set_fact_idx: int | None = None
        while j < len(lines):
            if re.match(r"^\s*-\s+name:", lines[j]) and j > i:
                break
            if re.match(r"^\s*loop:", lines[j]):
                has_loop = True
            if re.match(r"^\s*ansible\.builtin\.set_fact:", lines[j]):
                set_fact_idx = j
            j += 1
        if set_fact_idx is None or has_loop:
            i = j
            continue
        base_indent = len(lines[set_fact_idx]) - len(lines[set_fact_idx].lstrip())
        key_indent = base_indent + 2
        keys: dict[str, str] = {}
        k = set_fact_idx + 1
        while k < len(lines):
            stripped = lines[k]
            if not stripped.strip():
                k += 1
                continue
            indent = len(stripped) - len(stripped.lstrip())
            if indent <= base_indent:
                break
            m = re.match(r"^(\s+)([a-zA-Z_][a-zA-Z0-9_]*):\s*(.*)$", stripped)
            if m and len(m.group(1)) == key_indent:
                key = m.group(2)
                val_lines = [m.group(3)]
                k += 1
                while k < len(lines):
                    s2 = lines[k]
                    if not s2.strip():
                        val_lines.append(s2)
                        k += 1
                        continue
                    if len(s2) - len(s2.lstrip()) <= key_indent:
                        break
                    val_lines.append(s2)
                    k += 1
                keys[key] = "\n".join(val_lines)
                continue
            k += 1
        if len(keys) < 2:
            i = j
            continue
        for key, val in keys.items():
            for other in keys:
                if other == key:
                    continue
                if re.search(r"\{\{[^}]*\b" + re.escape(other) + r"\b", val) or re.search(
                    r"\{%[^%]*\b" + re.escape(other) + r"\b", val
                ):
                    issues.append(
                        f"{path}:{task_line} {task_name}: key '{key}' references '{other}' in same set_fact"
                    )
        i = j
    return issues


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    all_issues: list[str] = []
    for yml in sorted(root.rglob("*.yml")):
        all_issues.extend(find_issues(yml.read_text(), str(yml.relative_to(root))))
    if all_issues:
        print("set_fact self-reference issues:\n", file=sys.stderr)
        for item in all_issues:
            print(f"  {item}", file=sys.stderr)
        return 1
    print(f"OK: no multi-key set_fact self-refs in {len(list(root.rglob('*.yml')))} YAML files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
