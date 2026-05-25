#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import json
import re
import sys
import pathlib

root = pathlib.Path(".")
backlog = (root / "docs/parity/v0_7_top30_functions.md").read_text()
registry = json.loads((root / "docs/parity/python_polars_method_registry.json").read_text(encoding="utf-8"))["rows"]
reg_index = {(r["object"], r["method"]): r["gopolars_status"] for r in registry}
coverage_doc = (root / "docs/parity/v0_7_top30_coverage.json").read_text()

required = 30
match = re.search(r'"required_top30_implemented"\s*:\s*(\d+)', coverage_doc)
if match:
    required = int(match.group(1))

rows = re.findall(r'\|\s*\d+\s*\|\s*([A-Za-z]+)\s*\|\s*`([^`]+)`\s*\|\s*(high|medium|low)\s*\|', backlog)
if len(rows) != 30:
    print(f"unexpected top30 row count: {len(rows)}")
    sys.exit(1)

implemented = 0
for obj, method, _ in rows:
    st = reg_index.get((obj, method))
    if st is None:
        print(f"missing registry entry for {obj}.{method}")
        sys.exit(1)
    if st == "реализовано":
        implemented += 1

print(f"top30_implemented={implemented}/30 required={required}")
if implemented < required:
    sys.exit(1)
PY
