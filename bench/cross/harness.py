#!/usr/bin/env python3
"""Polars benchmark harness for cross-language SIMD comparison."""

import argparse
import json
import resource
import sys
import time

import polars as pl


def _max_rss_bytes():
    """Peak resident set size of this process so far, in bytes.

    ru_maxrss is bytes on macOS/Darwin but kilobytes on Linux — normalize to
    bytes. This is process-wide and monotonic (it only ever grows), so we read
    it before and after the timed loop and report the delta as an approximate
    working-set cost. Polars allocates in native Rust, so a Python-level
    tracemalloc would miss almost everything; RSS captures the real footprint.
    """
    rss = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    if sys.platform == "darwin":
        return rss
    return rss * 1024

_OPS = {
    "sum": lambda a, b: a.sum(),
    "min": lambda a, b: a.min(),
    "max": lambda a, b: a.max(),
    "mean": lambda a, b: a.mean(),
    "minmax": lambda a, b: (a.min(), a.max()),
    "add": lambda a, b: a + b,
    "mul": lambda a, b: a * b,
    # Full DataFrame filter+sum: filters rows where col("a") > threshold, sums
    # the result. The "b" argument is unused; filter_sum is dispatched specially
    # in main() so the --threshold flag can be threaded through.
}


def _filter_sum_op(series_a, threshold):
    """Eager: reconstruct a one-column DataFrame, filter, then sum.

    Mirrors gopolars' eager df.Filter(pred).Series("a").Sum() — both materialize
    the filtered rows before reducing.
    """
    import polars as pl
    df = pl.DataFrame({"a": series_a})
    return df.filter(pl.col("a") > threshold)["a"].sum()


def _filter_sum_lazy_op(series_a, threshold):
    """Lazy: let Polars' optimizer push the predicate down and fuse where it can.

    Apples-to-apples with gopolars' lazy/fused path
    (df.Lazy().Filter(pred).Sum().Collect()).
    """
    import polars as pl
    lf = pl.DataFrame({"a": series_a}).lazy()
    return lf.filter(pl.col("a") > threshold).select(pl.col("a").sum()).collect()


def _filter_sum_stream_op(series_a, threshold):
    """Lazy filter+sum on Polars' streaming engine — the closest analogue to a
    fused, non-materializing filter->reduce. Falls back across API spellings so
    it works on both newer (engine=) and older (streaming=) Polars."""
    import polars as pl
    lf = pl.DataFrame({"a": series_a}).lazy()
    q = lf.filter(pl.col("a") > threshold).select(pl.col("a").sum())
    for kwargs in ({"engine": "streaming"}, {"streaming": True}):
        try:
            return q.collect(**kwargs)
        except TypeError:
            continue
    return q.collect()


def main():
    parser = argparse.ArgumentParser(description="Run a Polars operation benchmark and emit JSON results.")
    parser.add_argument("--op", required=True, choices=list(_OPS.keys()) + ["filter_sum", "filter_sum_lazy", "filter_sum_stream"], help="Operation to benchmark")
    parser.add_argument("--input", required=True, help="Path to Arrow IPC input file")
    parser.add_argument("--iters", type=int, required=True, help="Number of timed iterations")
    parser.add_argument("--threshold", type=float, default=50.0, help="filter_sum predicate threshold (col('a') > threshold)")
    args = parser.parse_args()

    try:
        df = pl.read_ipc(args.input)
    except Exception as e:
        print(f"Error reading Arrow IPC file: {e}", file=sys.stderr)
        sys.exit(1)

    columns = df.columns
    if len(columns) == 0:
        print("No columns found in Arrow IPC file", file=sys.stderr)
        sys.exit(1)

    series_a = df[columns[0]]
    series_b = None
    if len(columns) >= 2:
        series_b = df[columns[1]]
    elif args.op in ("add", "mul"):
        print(f"Operation '{args.op}' requires two columns, but only one found", file=sys.stderr)
        sys.exit(1)

    if args.op == "filter_sum":
        op_fn = lambda a, b: _filter_sum_op(a, args.threshold)
    elif args.op == "filter_sum_lazy":
        op_fn = lambda a, b: _filter_sum_lazy_op(a, args.threshold)
    elif args.op == "filter_sum_stream":
        op_fn = lambda a, b: _filter_sum_stream_op(a, args.threshold)
    else:
        op_fn = _OPS[args.op]

    # Baseline RSS with the input loaded but the op never run yet. Taken before
    # warm-up so the high-water mark hasn't been bumped by the op itself.
    rss_before = _max_rss_bytes()

    # Warm-up to avoid amortizing Python import / JIT overhead
    try:
        op_fn(series_a, series_b)
    except Exception as e:
        print(f"Error during warm-up: {e}", file=sys.stderr)
        sys.exit(1)

    start = time.perf_counter()
    for _ in range(args.iters):
        op_fn(series_a, series_b)
    elapsed = time.perf_counter() - start
    # Peak RSS growth the op drove over the loaded-but-idle baseline. ru_maxrss
    # is a high-water mark, so this is the op's peak *working set* (one op's
    # footprint — for filter_sum each iteration frees its own intermediates),
    # NOT a cumulative per-iteration allocation count. It is reported as-is and
    # must not be divided by iters. Different in kind from Go's B/op (allocation
    # throughput), but both answer "how much memory did this cost?".
    peak_rss_bytes = max(0, _max_rss_bytes() - rss_before)

    result = {
        "op": args.op,
        "iters": args.iters,
        "elapsed_sec": elapsed,
        "elements": len(series_a),
        "peak_rss_bytes": peak_rss_bytes,
    }
    print(json.dumps(result))


if __name__ == "__main__":
    main()
