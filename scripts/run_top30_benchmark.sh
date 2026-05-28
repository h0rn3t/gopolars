#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

LIGHT=""
BENCHTIME="-benchtime=1s"
RESULTS_ONLY=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --light)
      LIGHT="-light"
      shift
      ;;
    --benchtime)
      BENCHTIME="-benchtime=$2"
      shift 2
      ;;
    --results-only)
      RESULTS_ONLY=1
      shift
      ;;
    -h|--help)
      cat <<'HELP'
Usage: run_top30_benchmark.sh [OPTIONS]

Run the top30 cross-language benchmark (Go gopolars vs Python Polars)
and print a comparison table.

Options:
  --light          Run only 1K and 1M sizes (fast CI mode)
  --benchtime T    Set benchtime (default: 1s)
  --results-only   Print existing results without running benchmark
  -h, --help       Show this help

Examples:
  ./scripts/run_top30_benchmark.sh
  ./scripts/run_top30_benchmark.sh --light
  ./scripts/run_top30_benchmark.sh --results-only
HELP
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -n "$RESULTS_ONLY" ]]; then
  if [[ -f "$PROJECT_ROOT/bench/top30/top30_summary.md" ]]; then
    cat "$PROJECT_ROOT/bench/top30/top30_summary.md"
  else
    echo "No results found at bench/top30/top30_summary.md" >&2
    echo "Run benchmark first without --results-only" >&2
    exit 1
  fi
  exit 0
fi

echo "Checking Python Polars..."
if ! python3 -c "import polars" 2>/dev/null; then
  echo "ERROR: Python Polars is not installed." >&2
  echo "Run: pip install -r bench/top30/requirements-bench.txt" >&2
  exit 1
fi
python3 -c "import polars; print(f'Python Polars {polars.__version__} OK')"

echo ""
echo "Running top30 benchmark (Go vs Python Polars)..."
echo "  benchtime: ${BENCHTIME#-benchtime=}  light: ${LIGHT:-no}"
echo ""

cd "$PROJECT_ROOT/bench/top30"
go test -bench=BenchmarkTop30 "$BENCHTIME" $LIGHT .

echo ""
echo "=== Comparison Table ==="
echo ""

if [[ -f "$PROJECT_ROOT/bench/top30/top30_summary.md" ]]; then
  cat "$PROJECT_ROOT/bench/top30/top30_summary.md"
else
  echo "WARNING: no markdown report generated" >&2
fi

echo ""
echo "=== Artifacts ==="
for f in top30_summary.json top30_summary.csv top30_summary.md; do
  if [[ -f "$PROJECT_ROOT/bench/top30/$f" ]]; then
    size=$(stat -f%z "$PROJECT_ROOT/bench/top30/$f" 2>/dev/null || stat -c%s "$PROJECT_ROOT/bench/top30/$f" 2>/dev/null || echo "?")
    echo "  $f (${size} bytes)"
  fi
done

if [[ -f "$PROJECT_ROOT/bench/top30/baselines/top30_baseline.json" ]]; then
  echo ""
  echo "=== Performance Budget Gate ==="
  cd "$PROJECT_ROOT"
  ./scripts/check_top30_performance_budgets.sh || true
else
  echo ""
  echo "No baseline found. To save current results as baseline:"
  echo "  cp bench/top30/top30_summary.json bench/top30/baselines/top30_baseline.json"
fi
