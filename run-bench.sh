#!/usr/bin/env bash
#
# run-bench.sh — run the gopolars cross-language filter+sum comparison benchmark.
#
# Compares the three gopolars execution paths against Python Polars on identical
# float64 datasets across two selectivity profiles (empty / half) and a range of
# sizes (1K..10M):
#
#   - eager         df.Filter(pred) materializes a filtered frame, then Sum()
#   - lazy          df.Lazy().Filter(pred).Sum().Collect() (fused reduce)
#   - eager_direct  df.FilterAggregateDirect(pred, "sum", ...) (eager fused,
#                   single masked pass, no materialization, bitmap predicate)
#
# Results print as an ASCII table and are written to
# bench/cross/filter_sum_summary.json.
#
# Usage:
#   ./run-bench.sh                 # Go-only (fast, deterministic)
#   ./run-bench.sh --python        # also run the Python Polars baseline
#   ./run-bench.sh --python -count=5 --benchtime=3s   # extra flags -> go test
#
# Any argument other than --python/--no-python is forwarded verbatim to
# `go test`, so you can pass -count, -benchtime, -run, -cpuprofile, etc.
#
# Environment:
#   GOPOLARS_BENCH_PYTHON=1   same as --python
#   BENCH_PYTHON_VENV=path    venv whose bin/ is prepended to PATH for the
#                             Python harness (default: .venv_polars_313 if present)

set -euo pipefail

# Resolve the repo root as the directory of this script, so it works from anywhere.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

WITH_PYTHON="${GOPOLARS_BENCH_PYTHON:+1}"
GO_ARGS=()

for arg in "$@"; do
	case "$arg" in
		--python)    WITH_PYTHON=1 ;;
		--no-python) WITH_PYTHON="" ;;
		*)           GO_ARGS+=("$arg") ;;
	esac
done

# Default go test flags when the caller passes none of their own benchmark knobs.
# The ${GO_ARGS[@]+...} guard keeps this safe under `set -u` on Bash 3.2 (macOS),
# where expanding an empty array otherwise errors.
have_count=""
have_benchtime=""
have_bench=""
for a in ${GO_ARGS[@]+"${GO_ARGS[@]}"}; do
	[[ "$a" == -count* ]] && have_count=1
	[[ "$a" == -benchtime* ]] && have_benchtime=1
	[[ "$a" == -bench=* || "$a" == -bench ]] && have_bench=1
done
[[ -z "$have_count" ]] && GO_ARGS+=("-count=1")
[[ -z "$have_benchtime" ]] && GO_ARGS+=("-benchtime=2s")
# Default to the whole cross benchmark (all sizes). A caller-supplied -bench
# overrides it, e.g. -bench='BenchmarkCrossFilterSum/.*size_1M$' to run only the
# 1,000,000-element comparison.
[[ -z "$have_bench" ]] && GO_ARGS+=('-bench=BenchmarkCrossFilterSum$')

echo "==> gopolars filter+sum cross benchmark"
echo "    repo:    $ROOT"
echo "    go args: ${GO_ARGS[*]}"

ENV_PREFIX=()
if [[ -n "$WITH_PYTHON" ]]; then
	# Pick a Python with Polars: an explicit venv, the README's default venv, or
	# whatever python3 on PATH already imports polars.
	VENV="${BENCH_PYTHON_VENV:-}"
	if [[ -z "$VENV" && -d "$ROOT/.venv_polars_313" ]]; then
		VENV="$ROOT/.venv_polars_313"
	fi
	if [[ -n "$VENV" ]]; then
		export PATH="$VENV/bin:$PATH"
		echo "    python:  $VENV/bin (prepended to PATH)"
	fi

	if ! python3 -c "import polars" >/dev/null 2>&1; then
		echo "ERROR: --python requested but 'import polars' failed for $(command -v python3)." >&2
		echo "       Install it (pip install -r bench/cross/requirements-bench.txt)" >&2
		echo "       or point BENCH_PYTHON_VENV at a venv that has polars." >&2
		exit 1
	fi
	PY_VERSION="$(python3 -c 'import polars,sys; print("polars", polars.__version__, "/ python", sys.version.split()[0])')"
	echo "    polars:  $PY_VERSION (Polars baseline enabled)"
	ENV_PREFIX=(env GOPOLARS_BENCH_PYTHON=1)
else
	echo "    polars:  disabled (Go-only run; pass --python to compare vs Polars)"
fi

echo
set -x
${ENV_PREFIX[@]+"${ENV_PREFIX[@]}"} go test ./bench/cross \
	-benchmem \
	-timeout=900s \
	${GO_ARGS[@]+"${GO_ARGS[@]}"}
set +x

echo
echo "==> summary JSON: bench/cross/filter_sum_summary.json"
