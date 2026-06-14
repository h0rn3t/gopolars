#!/usr/bin/env bash
#
# coverage.sh — run pkg/... tests with coverage, print a per-package summary,
# and gate each package against a minimum statement-coverage threshold.
#
# Usage:
#   scripts/coverage.sh              # uses the default MODE (see below)
#   MODE=report  scripts/coverage.sh # never fails the build, just reports
#   MODE=enforce scripts/coverage.sh # exit non-zero if any package is below its threshold
#   make cover                       # convenience wrapper (see Makefile)
#
# Thresholds:
#   BASELINE  — minimum statement coverage required for every pkg/... package (default 70%)
#   TOTAL_MIN — minimum combined coverage for the whole pkg/... tree (default 75%)
# Per-package overrides live in the `override` function below; document a reason
# for every override so the gate stays honest.
#
# Baseline recorded 2026-06-14 (before this change added tests), MODE=report:
#   cache 100.0  chunk 78.9  dtypes 100.0  exec 26.7  expr 27.2
#   expr/evalbatch 86.2  frame 72.6  io/arrow 82.8  io/csv 96.2
#   io/database 84.5  io/ipc 100.0  io/json 95.4  io/parquet 98.8
#   plan/logical [no test files]  plan/optimizer 65.2  plan/physical 0.0
#   polars 45.9  series 91.4  simd 100.0  sql 63.3
#   TOTAL 58.5
set -uo pipefail

BASELINE=${BASELINE:-70.0}
TOTAL_MIN=${TOTAL_MIN:-75.0}
# Default to enforcing: every pkg/... package now clears its floor, so a drop
# below threshold should fail the build. Override with MODE=report while iterating.
MODE=${MODE:-enforce}
PROFILE=${PROFILE:-coverage.out}

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT" || exit 2

# Per-package threshold overrides. Echo a number to override BASELINE for a
# package path, echo nothing to keep the baseline. Add a comment with the reason.
override() {
  case "$1" in
    # No overrides yet. Example for partly-untestable platform code:
    # */pkg/simd) echo 60.0 ;;  # SIMD asm fallbacks are platform-specific
    *) : ;;
  esac
}

below() { awk "BEGIN{exit !($1 < $2)}"; } # true (rc 0) when $1 < $2

echo ">> go test ./pkg/... -coverprofile=$PROFILE -covermode=atomic"
TEST_OUT="$(go test ./pkg/... -coverprofile="$PROFILE" -covermode=atomic 2>&1)"
TEST_RC=$?
echo "$TEST_OUT"

if [ $TEST_RC -ne 0 ]; then
  echo "!! go test failed (rc=$TEST_RC); coverage gate not evaluated"
  exit $TEST_RC
fi

TOTAL="$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"

echo ""
echo "================= per-package coverage ================="
fail=0
while IFS= read -r line; do
  case "$line" in
    *"coverage: [no statements]"*)
      pkg="$(printf '%s' "$line" | grep -oE 'github\.com/[^[:space:]]+' | head -1)"
      printf "  %-7s %6s   (min %s%%)  %s\n" "NOSTMT" "-" "$BASELINE" "$pkg"
      ;;
    *"coverage:"*)
      pkg="$(printf '%s' "$line" | grep -oE 'github\.com/[^[:space:]]+' | head -1)"
      [ -z "$pkg" ] && continue
      cov="$(printf '%s' "$line" | sed -E 's/.*coverage: ([0-9.]+)%.*/\1/')"
      thr="$BASELINE"
      ovr="$(override "$pkg")"; [ -n "$ovr" ] && thr="$ovr"
      status="OK"
      if below "$cov" "$thr"; then status="LOW"; fail=1; fi
      printf "  %-7s %6s%%  (min %s%%)  %s\n" "$status" "$cov" "$thr" "$pkg"
      ;;
    *"[no test files]"*)
      pkg="$(printf '%s' "$line" | grep -oE 'github\.com/[^[:space:]]+' | head -1)"
      printf "  %-7s %6s   (min %s%%)  %s\n" "NOTEST" "-" "$BASELINE" "$pkg"
      fail=1
      ;;
  esac
done <<< "$TEST_OUT"
echo "-------------------------------------------------------"
printf "  TOTAL   %6s%%  (min %s%%)\n" "$TOTAL" "$TOTAL_MIN"
if below "$TOTAL" "$TOTAL_MIN"; then fail=1; fi
echo "======================================================="

if [ "$fail" -ne 0 ]; then
  if [ "$MODE" = "enforce" ]; then
    echo "FAIL: coverage below threshold (MODE=enforce)"
    exit 1
  fi
  echo "WARN: coverage below threshold (MODE=report; not failing the build)"
fi
exit 0
