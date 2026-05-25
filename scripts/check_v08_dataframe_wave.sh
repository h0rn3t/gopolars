#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(".")
cfg = json.loads((root / "docs/parity/v0_8_dataframe_wave.json").read_text())
registry = json.loads((root / "docs/parity/python_polars_method_registry.json").read_text(encoding="utf-8"))["rows"]
reg_index = {(r["object"], r["method"]): r["gopolars_status"] for r in registry}

obj = cfg["object"]
required = int(cfg["required_methods_implemented"])
methods = cfg["methods"]

implemented = 0
for method in methods:
    st = reg_index.get((obj, method))
    if st == "реализовано":
        implemented += 1
    else:
        print(f"missing implementation: {method}")

print(f"v0_8_dataframe_wave={implemented}/{len(methods)} required={required}")
if implemented < required:
    sys.exit(1)
PY
