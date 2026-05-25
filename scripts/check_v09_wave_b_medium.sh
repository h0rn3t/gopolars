#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import json
import sys
from pathlib import Path

rows = [(r["method"], r["gopolars_status"], r["priority"]) for r in json.loads(Path("docs/parity/python_polars_method_registry.json").read_text(encoding="utf-8"))["rows"]]
remaining_high = [name for name, status, prio in rows if status == "не реализовано" and prio == "high"]
remaining_medium = [name for name, status, prio in rows if status == "не реализовано" and prio == "medium"]

print(f"remaining_high={len(remaining_high)}")
print(f"remaining_medium={len(remaining_medium)}")
if remaining_high or remaining_medium:
    print("unfinished_methods:")
    for name in remaining_high + remaining_medium:
        print(name)
    sys.exit(1)
PY
