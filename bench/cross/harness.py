#!/usr/bin/env python3
"""Polars benchmark harness for cross-language SIMD comparison."""

import argparse
import json
import sys
import time

import polars as pl

_OPS = {
    "sum": lambda a, b: a.sum(),
    "min": lambda a, b: a.min(),
    "max": lambda a, b: a.max(),
    "mean": lambda a, b: a.mean(),
    "minmax": lambda a, b: (a.min(), a.max()),
    "add": lambda a, b: a + b,
    "mul": lambda a, b: a * b,
}


def main():
    parser = argparse.ArgumentParser(description="Run a Polars operation benchmark and emit JSON results.")
    parser.add_argument("--op", required=True, choices=list(_OPS.keys()), help="Operation to benchmark")
    parser.add_argument("--input", required=True, help="Path to Arrow IPC input file")
    parser.add_argument("--iters", type=int, required=True, help="Number of timed iterations")
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

    op_fn = _OPS[args.op]

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

    result = {
        "op": args.op,
        "iters": args.iters,
        "elapsed_sec": elapsed,
        "elements": len(series_a),
    }
    print(json.dumps(result))


if __name__ == "__main__":
    main()
