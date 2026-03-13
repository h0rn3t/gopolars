#!/usr/bin/env bash
set -euo pipefail

BASELINE_FILE="${1:-docs/performance/v0_6_baseline.txt}"
OUTPUT_FILE="${2:-docs/performance/v0_6_regression_report.md}"

go test ./bench/micro -run TestNope -bench . -benchmem > /tmp/gopolars_v06_micro_current.txt
go test ./bench/macro -run TestNope -bench . -benchmem > /tmp/gopolars_v06_macro_current.txt

{
  echo "## v0.6 Performance Regression Report"
  echo
  echo "### Baseline"
  if [ -f "$BASELINE_FILE" ]; then
    cat "$BASELINE_FILE"
  else
    echo "No baseline file found at $BASELINE_FILE"
  fi
  echo
  echo "### Current micro benchmarks"
  cat /tmp/gopolars_v06_micro_current.txt
  echo
  echo "### Current macro benchmarks"
  cat /tmp/gopolars_v06_macro_current.txt
} > "$OUTPUT_FILE"
