#!/usr/bin/env python3
"""Polars benchmark harness for top30 cross-language comparison."""

import argparse
import json
import sys
import time

import polars as pl

_DATAFRAME_OPS = {
    "filter": lambda df: df.filter(pl.col("v") > 0),
    "select": lambda df: df.select(["g", "v"]),
    "with_columns": lambda df: df.with_columns(pl.col("v").alias("v2")),
    "sort": lambda df: df.sort("v"),
    "group_by": lambda df: df.group_by("g").agg(pl.col("v").sum()),
    "join": lambda df: df.join(df.unique(subset=["i"]), on="i", how="inner"),
    "head": lambda df: df.head(100),
    "tail": lambda df: df.tail(100),
    "unique": lambda df: df.select("g").unique(),
    "fill_null": lambda df: df.fill_null(0.0),
    "drop_nulls": lambda df: df.drop_nulls("n"),
}

_EXPR_OPS = {
    "cum_sum": lambda df: df.select(pl.col("v").cum_sum()),
    "rank": lambda df: df.select(pl.col("v").rank()),
    "over": lambda df: df.select(pl.col("v").cum_sum().over("g")),
    "fill_null": lambda df: df.select(pl.col("n").fill_null(0.0)),
    "fill_nan": lambda df: df.select(pl.col("n").fill_nan(0.0)),
    "rolling_mean": lambda df: df.select(pl.col("v").rolling_mean(100)),
    "rolling_sum": lambda df: df.select(pl.col("v").rolling_sum(100)),
    "rolling_min": lambda df: df.select(pl.col("v").rolling_min(100)),
    "rolling_max": lambda df: df.select(pl.col("v").rolling_max(100)),
}

_SERIES_OPS = {
    "null_count": lambda s: s.null_count(),
    "drop_nans": lambda s: s.drop_nans(),
    "to_list": lambda s: s.to_list(),
    "is_null": lambda s: s.is_null(),
    "is_not_null": lambda s: s.is_not_null(),
    "fill_nan": lambda s: s.fill_nan(0.0),
}

_LAZYFRAME_OPS = {
    "collect": lambda lf: lf.collect(),
    "sql": lambda lf: lf.sql("SELECT g, v FROM self WHERE v > 0"),
    "inspect": lambda lf: lf.inspect(lambda df: df),  # noqa: ARG005
}

_SQLCONTEXT_OPS = {
    "execute": lambda ctx, df: ctx.execute("SELECT * FROM t"),
    "register": lambda ctx, df: ctx.register("t", df) or ctx,
    "tables": lambda ctx, df: ctx.tables(),
}


def _run_dataframe_op(df, op):
    fn = _DATAFRAME_OPS.get(op)
    if fn is None:
        raise ValueError(f"unsupported DataFrame op: {op}")
    return fn(df)


def _run_expr_op(df, op):
    fn = _EXPR_OPS.get(op)
    if fn is None:
        raise ValueError(f"unsupported Expr op: {op}")
    return fn(df)


def _run_series_op(df, op):
    s = df["v"]
    fn = _SERIES_OPS.get(op)
    if fn is None:
        raise ValueError(f"unsupported Series op: {op}")
    return fn(s)


def _run_lazyframe_op(df, op):
    lf = df.lazy()
    fn = _LAZYFRAME_OPS.get(op)
    if fn is None:
        raise ValueError(f"unsupported LazyFrame op: {op}")
    return fn(lf)


def _run_sqlcontext_op(df, op):
    ctx = pl.SQLContext()
    ctx.register("t", df)
    fn = _SQLCONTEXT_OPS.get(op)
    if fn is None:
        raise ValueError(f"unsupported SQLContext op: {op}")
    return fn(ctx, df)


def main():
    parser = argparse.ArgumentParser(description="Run a Polars top30 benchmark and emit JSON results.")
    parser.add_argument("--op", required=True, help="Operation to benchmark")
    parser.add_argument("--object", required=True, choices=["DataFrame", "Expr", "Series", "LazyFrame", "SQLContext"], help="Object type")
    parser.add_argument("--input", required=True, help="Path to Arrow IPC input file")
    parser.add_argument("--iters", type=int, required=True, help="Number of timed iterations")
    args = parser.parse_args()

    try:
        df = pl.read_ipc(args.input)
    except Exception as e:
        print(f"Error reading Arrow IPC file: {e}", file=sys.stderr)
        sys.exit(1)

    runners = {
        "DataFrame": _run_dataframe_op,
        "Expr": _run_expr_op,
        "Series": _run_series_op,
        "LazyFrame": _run_lazyframe_op,
        "SQLContext": _run_sqlcontext_op,
    }

    runner = runners.get(args.object)
    if runner is None:
        print(f"Unknown object type: {args.object}", file=sys.stderr)
        sys.exit(1)

    # Warm-up loop (3 iterations) before timed loop
    for _ in range(3):
        try:
            runner(df, args.op)
        except Exception as e:
            print(f"Error during warm-up: {e}", file=sys.stderr)
            sys.exit(1)

    start = time.perf_counter()
    for _ in range(args.iters):
        runner(df, args.op)
    elapsed = time.perf_counter() - start

    result = {
        "op": args.op,
        "object": args.object,
        "iters": args.iters,
        "elapsed_sec": elapsed,
        "elements": len(df),
    }
    print(json.dumps(result))


if __name__ == "__main__":
    main()
