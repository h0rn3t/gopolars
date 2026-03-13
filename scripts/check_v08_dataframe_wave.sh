#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import json
import pathlib
import re
import sys

root = pathlib.Path(".")
cfg = json.loads((root / "docs/parity/v0_8_dataframe_wave.json").read_text())
matrix = (root / "docs/parity/python_polars_full_matrix.md").read_text()

obj = cfg["object"]
required = int(cfg["required_methods_implemented"])
methods = cfg["methods"]

section = re.search(rf"## {obj} \(\d+ методов\)\n([\s\S]*?)(?:\n## |\Z)", matrix)
if not section:
    print(f"missing section for {obj}")
    sys.exit(1)
block = section.group(1)

implemented = 0
for method in methods:
    line = re.search(rf"\| `{re.escape(method)}` \| .*? \| реализовано \|", block)
    if line:
        implemented += 1
    else:
        print(f"missing implementation: {method}")

print(f"v0_8_dataframe_wave={implemented}/{len(methods)} required={required}")
if implemented < required:
    sys.exit(1)
PY
