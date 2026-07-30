#!/usr/bin/env python3
"""Parse TDengine REST SELECT LAST(insert_time) response into freshness map.

Usage:
  python3 sem_parse_tdengine_channel_last.py <response.json> <stale_hours> <timezone> <http_status>

Stdout: JSON {ok, error, row_count, stale_hours, by_computer}
"""
from __future__ import annotations

import json
import re
import sys
from datetime import datetime, timedelta, timezone


def _now(tz_name: str) -> datetime:
    try:
        from zoneinfo import ZoneInfo

        return datetime.now(ZoneInfo(tz_name))
    except Exception:
        if tz_name in ("Asia/Shanghai", "Asia/Chongqing", "PRC", "UTC+8"):
            return datetime.now(timezone(timedelta(hours=8)))
        return datetime.now(timezone.utc)


def parse_ts(v, fallback_tz):
    if v is None:
        return None
    if isinstance(v, (int, float)):
        n = float(v)
        if n > 1e10:
            n = n / 1000.0
        return datetime.fromtimestamp(n, tz=timezone.utc).astimezone(fallback_tz)

    s = str(v).strip()
    if not s:
        return None

    # Detect explicit UTC / offset; otherwise treat as fallback_tz local wall time
    explicit_utc = s.endswith("Z") or s.endswith("z")
    explicit_offset = bool(re.search(r"[+-]\d{2}:\d{2}$", s))

    s2 = s.replace("T", " ").replace("t", " ")
    s2 = re.sub(r"[Zz]$", "", s2)
    s2 = re.sub(r"[+-]\d{2}:\d{2}$", "", s2).strip()

    if "." in s2:
        head, frac = s2.split(".", 1)
        digits = "".join(c for c in frac if c.isdigit())[:6].ljust(6, "0")
        s2 = "%s.%s" % (head, digits)
        fmt = "%Y-%m-%d %H:%M:%S.%f"
    else:
        fmt = "%Y-%m-%d %H:%M:%S"
        s2 = s2[:19]

    try:
        dt = datetime.strptime(s2[:26] if "." in s2 else s2[:19], fmt)
    except Exception:
        return None

    if explicit_utc:
        return dt.replace(tzinfo=timezone.utc).astimezone(fallback_tz)
    if explicit_offset:
        # offset was stripped; treat remaining as UTC-equivalent wall if unknown — prefer UTC
        return dt.replace(tzinfo=timezone.utc).astimezone(fallback_tz)
    return dt.replace(tzinfo=fallback_tz)


def main() -> int:
    if len(sys.argv) < 5:
        print(json.dumps({
            "ok": False,
            "error": "usage: script <file> <stale_hours> <timezone> <http_status>",
            "by_computer": {},
        }))
        return 0

    path, stale_s, tz_name, http_status = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
    try:
        stale_h = float(stale_s or "6")
    except Exception:
        stale_h = 6.0

    http_status = str(http_status or "").strip()
    if http_status == "-1":
        print(json.dumps({
            "ok": False,
            "error": "tdengine_http_failed status=-1",
            "by_computer": {},
        }))
        return 0

    try:
        with open(path, "r", encoding="utf-8") as f:
            raw = f.read()
    except Exception as e:
        print(json.dumps({
            "ok": False,
            "error": "read_file: %s" % e,
            "by_computer": {},
        }))
        return 0

    if not raw.strip():
        print(json.dumps({
            "ok": False,
            "error": "tdengine_http_failed status=%s empty_body" % (http_status or "n/a"),
            "by_computer": {},
        }))
        return 0

    try:
        body = json.loads(raw)
    except Exception as e:
        print(json.dumps({
            "ok": False,
            "error": "json_parse: %s" % e,
            "by_computer": {},
        }))
        return 0

    code = body.get("code", -1)
    if code not in (0, "0", None):
        print(json.dumps({
            "ok": False,
            "error": "td_code=%s desc=%s" % (code, body.get("desc", "")),
            "by_computer": {},
        }))
        return 0

    now = _now(tz_name.strip() or "Asia/Shanghai")

    cols = []
    for c in body.get("column_meta") or []:
        if isinstance(c, (list, tuple)) and c:
            cols.append(str(c[0]).lower())
        else:
            cols.append(str(c).lower())
    ts_idx, name_idx = 0, 1
    for i, c in enumerate(cols):
        if "computer_name" in c:
            name_idx = i
        if "insert_time" in c or c.startswith("last(") or c == "last":
            ts_idx = i

    by_computer = {}
    for row in body.get("data") or []:
        if not isinstance(row, (list, tuple)) or len(row) <= max(ts_idx, name_idx):
            continue
        name = str(row[name_idx] or "").strip()
        if not name:
            continue
        dt = parse_ts(row[ts_idx], now.tzinfo)
        if dt is None:
            continue
        age_h = max(0.0, (now - dt).total_seconds() / 3600.0)
        fresh = age_h <= stale_h
        by_computer[name.lower()] = {
            "computer_name": name,
            "insert_time": dt.strftime("%Y-%m-%d %H:%M:%S"),
            "age_hours": round(age_h, 3),
            "fresh": fresh,
            "stale": not fresh,
        }

    print(json.dumps({
        "ok": True,
        "error": "",
        "row_count": len(by_computer),
        "stale_hours": stale_h,
        "by_computer": by_computer,
    }))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
