#!/usr/bin/env bash
set -euo pipefail

BENCH_DIR="bench/top30"
BASELINE_FILE="${1:-$BENCH_DIR/baselines/top30_baseline.json}"
CURRENT_FILE="${2:-$BENCH_DIR/top30_summary.json}"
THRESHOLDS_FILE="$BENCH_DIR/baselines/thresholds.json"

if [ ! -f "$BASELINE_FILE" ]; then
  echo "Baseline file not found: $BASELINE_FILE"
  exit 1
fi

if [ ! -f "$CURRENT_FILE" ]; then
  echo "Current benchmark file not found: $CURRENT_FILE"
  exit 1
fi

if [ ! -f "$THRESHOLDS_FILE" ]; then
  echo "Thresholds file not found: $THRESHOLDS_FILE"
  exit 1
fi

export BASELINE_FILE CURRENT_FILE THRESHOLDS_FILE

python3 - <<'PY'
import json
import sys
import os

baseline_path = os.environ["BASELINE_FILE"]
current_path = os.environ["CURRENT_FILE"]
thresholds_path = os.environ["THRESHOLDS_FILE"]

with open(thresholds_path) as f:
    thresholds = json.load(f)

default_tol = thresholds.get("default_tolerance", 0.10)
overrides = thresholds.get("overrides", {})

with open(baseline_path) as f:
    baseline = json.load(f)

with open(current_path) as f:
    current = json.load(f)

baseline_by_key = {
    (r["object"], r["op"], r["size"]): r
    for r in baseline
}

current_by_key = {
    (r["object"], r["op"], r["size"]): r
    for r in current
}

regressions = []

for key, cur in current_by_key.items():
    base = baseline_by_key.get(key)
    if base is None:
        continue
    obj, op, size = key
    tol = overrides.get(op, default_tol)
    base_ratio = base.get("ratio", 0)
    cur_ratio = cur.get("ratio", 0)
    if base_ratio == 0:
        continue
    delta = abs(cur_ratio - base_ratio) / base_ratio
    if delta > tol:
        regressions.append({
            "object": obj,
            "op": op,
            "size": size,
            "base_ratio": base_ratio,
            "cur_ratio": cur_ratio,
            "delta": delta,
            "tolerance": tol,
        })

if regressions:
    print("FAIL: Performance regressions detected")
    for r in regressions:
        print(f"  {r['object']}.{r['op']} size={r['size']}: "
              f"baseline_ratio={r['base_ratio']:.4f} current_ratio={r['cur_ratio']:.4f} "
              f"delta={r['delta']:.2%} (tolerance={r['tolerance']:.0%})")
    sys.exit(1)
else:
    print("PASS: All ratios within tolerance")
    sys.exit(0)
PY
