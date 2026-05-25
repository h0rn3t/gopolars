#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import json
import sys
from pathlib import Path

reg = Path("docs/parity/python_polars_method_registry.json")
rows = [(r["method"], r["gopolars_status"], r["priority"]) for r in json.loads(reg.read_text(encoding="utf-8"))["rows"]]
remaining_high = [name for name, status, prio in rows if status == "не реализовано" and prio == "high"]
implemented = sum(1 for _, status, _ in rows if status == "реализовано")

print(f"implemented={implemented} total={len(rows)}")
print(f"remaining_high={len(remaining_high)}")
if remaining_high:
    print("high_remaining_methods:")
    for name in remaining_high:
        print(name)
    sys.exit(1)
PY
