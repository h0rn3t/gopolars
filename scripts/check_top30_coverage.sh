#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import re, sys, pathlib

root = pathlib.Path(".")
backlog = (root / "docs/parity/v0_7_top30_functions.md").read_text()
matrix = (root / "docs/parity/python_polars_full_matrix.md").read_text()
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
    pattern = rf'\| `{re.escape(method)}` \| .*? \| реализовано \|'
    section = rf'## {obj} \(.*?\)\n([\s\S]*?)(?:\n## |\Z)'
    sec_match = re.search(section, matrix)
    if not sec_match:
        print(f"missing matrix section for {obj}")
        sys.exit(1)
    if re.search(pattern, sec_match.group(1)):
        implemented += 1

print(f"top30_implemented={implemented}/30 required={required}")
if implemented < required:
    sys.exit(1)
PY
