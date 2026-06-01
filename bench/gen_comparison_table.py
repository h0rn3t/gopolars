#!/usr/bin/env python3
"""Generate gopolars vs Python Polars comparative benchmark table for README.

Sources:
  - bench/top30/top30_summary.json      timing Go vs Polars (top-30 ops)
  - bench/cross/filter_sum_summary.json timing + memory for filter+sum pipelines
  - Optional --benchmem FILE            `go test -benchmem` output for Go memory

Usage:
  python3 bench/gen_comparison_table.py
  python3 bench/gen_comparison_table.py --benchmem bench/top30_benchmem.txt
"""

import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).parent.parent

# ──────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────

def fmt_dur(ns: float) -> str:
    if ns <= 0:
        return "n/a"
    if ns >= 1_000_000_000:
        return f"{ns/1e9:.2f} s"
    if ns >= 1_000_000:
        return f"{ns/1e6:.2f} ms"
    if ns >= 1_000:
        return f"{ns/1e3:.1f} µs"
    return f"{int(ns)} ns"

def fmt_py(sec: float) -> str:
    ns = sec * 1e9
    return fmt_dur(ns)

def fmt_bytes(b: int) -> str:
    if b <= 0:
        return "—"
    if b >= 1 << 20:
        return f"{b/(1<<20):.1f} MB"
    if b >= 1 << 10:
        return f"{b/(1<<10):.1f} KB"
    return f"{b} B"

def fmt_speedup(ratio: float) -> str:
    """ratio = python_sec / go_sec.  >1 means Go is faster."""
    if ratio <= 0:
        return "—"
    if ratio >= 1:
        return f"**Go ×{ratio:.1f}**"
    return f"Py ×{1/ratio:.1f}"


# ──────────────────────────────────────────────
# Parse go test -benchmem stdout
# ──────────────────────────────────────────────
# BenchmarkTop30/DataFrame/filter/size_1K-12   139476   15213 ns/op  28912 B/op  25 allocs/op
_BENCH_RE = re.compile(
    r"BenchmarkTop30/(\w+)/(\w+)/size_(\w+)-\d+\s+"
    r"\d+\s+([\d.]+)\s+ns/op\s+([\d.]+)\s+B/op\s+([\d.]+)\s+allocs/op"
)

def parse_benchmem(text: str) -> dict:
    """Return {(object, op, size): (ns_per_op, bytes_per_op, allocs_per_op)}."""
    result = {}
    for m in _BENCH_RE.finditer(text):
        obj, op, size = m.group(1), m.group(2), m.group(3)
        # size like "1K", "1M"
        result[(obj, op, size)] = (
            float(m.group(4)),
            int(float(m.group(5))),
            int(float(m.group(6))),
        )
    return result


# ──────────────────────────────────────────────
# Deduplicate top30 JSON (multiple runs → keep min ns/op per key)
# ──────────────────────────────────────────────

def load_top30(path: Path) -> dict:
    """Return {(object, op, size): {'go_ns', 'py_sec', 'ratio'}}."""
    data = json.loads(path.read_text())
    best: dict = {}
    for row in data:
        key = (row["object"], row["op"], row["size"])
        if key not in best or row["go_ns_per_op"] < best[key]["go_ns"]:
            best[key] = {
                "go_ns":  row["go_ns_per_op"],
                "py_sec": row["python_sec_per_op"],
                "ratio":  row["python_sec_per_op"] / (row["go_ns_per_op"] / 1e9)
                          if row["go_ns_per_op"] > 0 else 0,
            }
    return best


# ──────────────────────────────────────────────
# Load filter+sum cross data
# ──────────────────────────────────────────────

def load_filter_sum(path: Path) -> dict:
    """Return {(profile, engine, size): row}."""
    data = json.loads(path.read_text())
    result = {}
    for row in data:
        # op like "filter_sum/empty/eager"
        parts = row["op"].split("/")
        if len(parts) == 3:
            _, profile, engine = parts
        else:
            continue
        result[(profile, engine, row["size"])] = row
    return result


# ──────────────────────────────────────────────
# Render tables
# ──────────────────────────────────────────────

def render_top30_section(top30: dict, benchmem: dict, sizes: list[str]) -> str:
    objects = ["DataFrame", "Expr", "Series", "LazyFrame", "SQLContext"]
    ops_by_obj = {
        "DataFrame":  ["filter","select","with_columns","sort","group_by","join","head","tail","unique","fill_null","drop_nulls"],
        "Expr":       ["cum_sum","rank","over","fill_null","fill_nan","rolling_mean","rolling_sum","rolling_min","rolling_max"],
        "Series":     ["null_count","drop_nans","to_list","is_null","is_not_null","fill_nan"],
        "LazyFrame":  ["collect","sql","inspect"],
        "SQLContext": ["execute","register","tables"],
    }

    has_mem = bool(benchmem)
    out = []
    out.append("## Top-30 Operations — gopolars vs Python Polars\n")
    out.append(f"> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars {_polars_version()}")
    out.append("> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds\n")

    for obj in objects:
        out.append(f"### {obj}\n")

        # header
        if has_mem:
            header = "| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |"
            divider = "|-----------|------|---------|---------|---------|---------|--------------|"
        else:
            header = "| operation | size | Go time | Py time | speedup |"
            divider = "|-----------|------|---------|---------|---------|"
        out.append(header)
        out.append(divider)

        for op in ops_by_obj[obj]:
            for size in sizes:
                key = (obj, op, size)
                if key not in top30:
                    continue
                row = top30[key]
                go_time = fmt_dur(row["go_ns"])
                py_time = fmt_py(row["py_sec"])
                speedup = fmt_speedup(row["ratio"])

                if has_mem and key in benchmem:
                    ns, b_per_op, allocs = benchmem[key]
                    if has_mem:
                        out.append(f"| `{op}` | {size} | {go_time} | {py_time} | {speedup} | {fmt_bytes(b_per_op)} | {allocs} |")
                    else:
                        out.append(f"| `{op}` | {size} | {go_time} | {py_time} | {speedup} |")
                else:
                    if has_mem:
                        out.append(f"| `{op}` | {size} | {go_time} | {py_time} | {speedup} | — | — |")
                    else:
                        out.append(f"| `{op}` | {size} | {go_time} | {py_time} | {speedup} |")

        out.append("")

    return "\n".join(out)


def render_filter_sum_section(fs: dict) -> str:
    sizes = ["1K", "10K", "100K", "1M", "10M"]
    engines = [
        ("eager",        "Eager (filter then sum)"),
        ("lazy",         "Lazy fused (filter+sum single pass)"),
        ("eager_direct", "Eager-direct (fused, no plan)"),
    ]
    profiles = [
        ("empty", "0% selectivity (threshold=50, no rows pass)"),
        ("half",  "50% selectivity (threshold=0, half rows pass)"),
    ]

    out = []
    out.append("## Filter + Sum Pipeline — Detailed Comparison\n")
    out.append("> Three execution paths vs Python Polars eager and lazy across two selectivity profiles.\n")
    out.append("> **Go B/op** = heap bytes allocated per operation (Go runtime).  ")
    out.append("> **Py peak RSS** = peak resident-set-size growth Polars drove for the operation.\n")

    for profile_key, profile_label in profiles:
        out.append(f"### {profile_label}\n")
        out.append("| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |")
        out.append("|--------|------|---------|---------|---------|---------|-----------|-------------|")

        for size in sizes:
            for eng_key, eng_label in engines:
                key = (profile_key, eng_key, size)
                if key not in fs:
                    continue
                row = fs[key]
                go_ns  = row["go_ns_per_op"]
                py_sec = row.get("python_sec_per_op", 0)
                b_op   = row.get("go_bytes_per_op", 0)
                allocs = row.get("go_allocs_per_op", 0)
                py_rss = row.get("python_peak_rss_bytes", 0)

                ratio = py_sec / (go_ns / 1e9) if go_ns > 0 and py_sec > 0 else 0
                out.append(
                    f"| {eng_label} | {size} | {fmt_dur(go_ns)} | {fmt_py(py_sec)} | "
                    f"{fmt_speedup(ratio)} | {fmt_bytes(b_op)} | {allocs} | {fmt_bytes(py_rss)} |"
                )
        out.append("")

    return "\n".join(out)


def _polars_version() -> str:
    try:
        import polars
        return polars.__version__
    except ImportError:
        return "?"


# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--benchmem", help="go test -benchmem stdout file")
    parser.add_argument("--sizes", default="1K,1M", help="comma-separated sizes to show")
    parser.add_argument("--output", default="-", help="output file (- = stdout)")
    args = parser.parse_args()

    sizes = args.sizes.split(",")

    top30_path = ROOT / "bench/top30/top30_summary.json"
    fs_path    = ROOT / "bench/cross/filter_sum_summary.json"

    if not top30_path.exists():
        sys.exit(f"ERROR: {top30_path} not found — run the top30 benchmark first")
    if not fs_path.exists():
        sys.exit(f"ERROR: {fs_path} not found — run ./run-bench.sh --python first")

    top30  = load_top30(top30_path)
    fs     = load_filter_sum(fs_path)

    benchmem: dict = {}
    if args.benchmem:
        benchmem = parse_benchmem(Path(args.benchmem).read_text())

    sections = []
    sections.append(render_top30_section(top30, benchmem, sizes))
    sections.append(render_filter_sum_section(fs))

    content = "\n\n".join(sections)

    if args.output == "-":
        print(content)
    else:
        Path(args.output).write_text(content)
        print(f"Written to {args.output}", file=sys.stderr)


if __name__ == "__main__":
    main()
