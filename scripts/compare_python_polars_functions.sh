#!/usr/bin/env bash
set -euo pipefail

PARQUET_PATH="${1:-/tmp/polars_compare_sample.parquet}"
CSV_OUT="${2:-/tmp/func_compare.csv}"
MD_OUT="${3:-/tmp/func_compare.md}"
VENV_DIR="${4:-.venv_polars_313}"

if ! command -v python3.13 >/dev/null 2>&1; then
  echo "python3.13 not found"
  exit 1
fi

if [ ! -d "$VENV_DIR" ]; then
  python3.13 -m venv "$VENV_DIR"
fi

"$VENV_DIR/bin/python" -m pip install --upgrade pip >/dev/null
"$VENV_DIR/bin/python" -m pip install --upgrade polars >/dev/null

if [ ! -f "$PARQUET_PATH" ]; then
  "$VENV_DIR/bin/python" - "$PARQUET_PATH" <<'PY'
import pathlib
import polars as pl
import sys

p = pathlib.Path(sys.argv[1]).resolve()
p.parent.mkdir(parents=True, exist_ok=True)
pl.DataFrame(
    {
        "g": ["a", "a", "b", "b"],
        "v": [1.0, 2.0, 3.0, 4.0],
        "n": [1.0, None, float("nan"), 4.0],
    }
).write_parquet(p)
print(f"sample_parquet_created={p}")
PY
fi

"$VENV_DIR/bin/python" - "$PARQUET_PATH" "$CSV_OUT" "$MD_OUT" <<'PY'
import csv
import collections
import os
import pathlib
import time
import re
import subprocess
import sys

parquet_path = pathlib.Path(sys.argv[1]).resolve()
csv_out = pathlib.Path(sys.argv[2]).resolve()
md_out = pathlib.Path(sys.argv[3]).resolve()
matrix_path = pathlib.Path("POLARS_PARITY_TABLE.md").resolve()
registry_path = matrix_path.with_name("python_polars_method_registry.json").resolve()

import json

import polars as pl

ansi_re = re.compile(r"\x1b\[[0-9;]*m")

def strip_ansi(text):
    return ansi_re.sub("", text)

def pad_ansi(text, width):
    raw = strip_ansi(text)
    pad = width - len(raw)
    if pad <= 0:
        return text
    return text + (" " * pad)

def render_ascii_table(headers, rows):
    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(strip_ansi(str(cell))))
    sep = "+" + "+".join("-" * (w + 2) for w in widths) + "+"
    out = [sep]
    out.append("| " + " | ".join(pad_ansi(headers[i], widths[i]) for i in range(len(headers))) + " |")
    out.append(sep)
    for row in rows:
        out.append("| " + " | ".join(pad_ansi(str(row[i]), widths[i]) for i in range(len(headers))) + " |")
    out.append(sep)
    return "\n".join(out)

if registry_path.exists():
    rows = json.loads(registry_path.read_text(encoding="utf-8"))["rows"]
else:
    # Формат, який видає gen_parity_table.py:
    #   ## DataFrame
    #   | `approx_n_unique` | `ApproxNUnique` | ✅ |
    #   | `gather`          | —               | ❌ |
    matrix_text = matrix_path.read_text(encoding="utf-8")
    sections = ["DataFrame", "LazyFrame", "Expr", "Series", "SQLContext"]
    rows = []
    for obj in sections:
        section_match = re.search(rf"^## {obj}$\n([\s\S]*?)(?:\n## |\Z)", matrix_text, re.M)
        if not section_match:
            continue
        block = section_match.group(1)
        for line in block.splitlines():
            m = re.match(r"\| `([^`]+)` \| (?:`([^`]+)`|—) \| (✅|❌) \|", line.strip())
            if not m:
                continue
            method, equivalent, mark = m.groups()
            rows.append(
                {
                    "object": obj,
                    "method": method,
                    "equivalent": equivalent or "—",
                    "gopolars_status": "реализовано" if mark == "✅" else "не реализовано",
                    "priority": "—",
                }
            )
    if not rows:
        raise SystemExit(f"не розібрано жодного рядка з {matrix_path}")

df = pl.read_parquet(str(parquet_path))
lf = df.lazy()
series = df[df.columns[0]] if df.columns else pl.Series("empty", [])
expr_obj = pl.col(df.columns[0]) if df.columns else pl.lit(1)
sql_ctx = pl.SQLContext()
sql_ctx.register("t", df)

python_objects = {
    "DataFrame": pl.DataFrame,
    "LazyFrame": pl.LazyFrame,
    "Expr": pl.Expr,
    "Series": pl.Series,
    "SQLContext": pl.SQLContext,
}

python_instances = {
    "DataFrame": df,
    "LazyFrame": lf,
    "Expr": expr_obj,
    "Series": series,
    "SQLContext": sql_ctx,
}

exec_smoke = {
    ("DataFrame", "height"): lambda x: x.height,
    ("DataFrame", "columns"): lambda x: x.columns,
    ("DataFrame", "is_empty"): lambda x: x.is_empty(),
    ("DataFrame", "estimated_size"): lambda x: x.estimated_size(),
    ("DataFrame", "null_count"): lambda x: x.null_count(),
    ("DataFrame", "n_unique"): lambda x: x.n_unique(),
    ("DataFrame", "to_dicts"): lambda x: x.to_dicts(),
    ("DataFrame", "with_row_count"): lambda x: x.with_row_count("rn"),
    ("DataFrame", "with_row_index"): lambda x: x.with_row_index("ri"),
    ("DataFrame", "sample"): lambda x: x.sample(n=min(2, len(x)), seed=42),
    ("DataFrame", "fill_nan"): lambda x: x.fill_nan(0.0),
    ("DataFrame", "drop_nans"): lambda x: x.drop_nans(),
    ("LazyFrame", "collect"): lambda x: x.collect(),
    ("LazyFrame", "collect_async"): lambda x: x.collect_async(),
    ("LazyFrame", "collect_batches"): lambda x: list(x.collect_batches()),
    ("LazyFrame", "inspect"): lambda x: x.inspect(),
    ("LazyFrame", "profile"): lambda x: x.profile(),
    ("LazyFrame", "sql"): lambda x: x.sql("SELECT * FROM self"),
    ("Expr", "cum_sum"): lambda x: x.cum_sum(),
    ("Expr", "cum_count"): lambda x: x.cum_count(),
    ("Expr", "rank"): lambda x: x.rank(),
    ("Expr", "over"): lambda x: x.over("g"),
    ("Expr", "replace"): lambda x: x.replace(1, 2),
    ("Expr", "fill_null"): lambda x: x.fill_null(0),
    ("Expr", "fill_nan"): lambda x: x.fill_nan(0.0),
    ("Expr", "rolling_min"): lambda x: x.rolling_min(2),
    ("Expr", "rolling_max"): lambda x: x.rolling_max(2),
    ("Expr", "rolling_mean"): lambda x: x.rolling_mean(2),
    ("Expr", "rolling_sum"): lambda x: x.rolling_sum(2),
    ("Expr", "rolling_std"): lambda x: x.rolling_std(2),
    ("Series", "is_null"): lambda x: x.is_null(),
    ("Series", "is_not_null"): lambda x: x.is_not_null(),
    ("Series", "fill_nan"): lambda x: x.fill_nan(0.0),
    ("Series", "drop_nans"): lambda x: x.drop_nans(),
    ("Series", "null_count"): lambda x: x.null_count(),
    ("Series", "rolling_mean"): lambda x: x.rolling_mean(2),
    ("Series", "rolling_sum"): lambda x: x.rolling_sum(2),
    ("Series", "rolling_min"): lambda x: x.rolling_min(2),
    ("Series", "rolling_max"): lambda x: x.rolling_max(2),
    ("Series", "to_list"): lambda x: x.to_list(),
    ("SQLContext", "execute"): lambda x: x.execute("SELECT * FROM t"),
    ("SQLContext", "register"): lambda x: x.register("tx", df),
    ("SQLContext", "tables"): lambda x: x.tables(),
    ("SQLContext", "unregister"): lambda x: x.unregister("tx"),
}

go_suite_ok = False
go_version = "unknown"
go_suite_time_ms = 0.0
try:
    go_version_proc = subprocess.run(
        ["go", "version"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    go_version = go_version_proc.stdout.strip() or "unknown"
    go_started = time.perf_counter()
    p = subprocess.run(
        ["go", "test", "./test/unit", "-run", "TestV07DataFrameTop30Utilities|TestV07ExprTop30Methods|TestV07LazyTop30MethodsAndSQLContext"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    go_suite_time_ms = (time.perf_counter() - go_started) * 1000.0
    go_suite_ok = p.returncode == 0
except Exception:
    go_suite_ok = False

full_console = os.environ.get("FULL_REPORT", "0") == "1"
python_version = sys.version.split()[0]

def benchmark_callable(fn, budget_s=0.02, max_iters=200):
    started = time.perf_counter()
    runs = 0
    while runs < max_iters:
        fn()
        runs += 1
        if time.perf_counter() - started >= budget_s:
            break
    elapsed = time.perf_counter() - started
    avg_ms = (elapsed / runs) * 1000 if runs > 0 else 0.0
    return runs, avg_ms, elapsed * 1000.0

report = []
python_benchmark_total_ms = 0.0
for idx, row in enumerate(rows, start=1):
    obj = row["object"]
    method = row["method"]
    py_cls = python_objects[obj]
    py_inst = python_instances[obj]
    py_has = hasattr(py_cls, method)
    py_exec = "skip"
    py_error = ""
    if (obj, method) in exec_smoke and py_has:
        try:
            exec_smoke[(obj, method)](py_inst)
            py_exec = "ok"
        except Exception as e:
            py_exec = "error"
            py_error = str(e)
    elif py_has:
        py_exec = "requires_args_or_complex_context"
    else:
        py_exec = "missing_method"

    g_status = row["gopolars_status"]
    g_exec = "not_implemented"
    if g_status == "реализовано":
        g_exec = "suite_ok" if go_suite_ok else "suite_unverified"

    bench_mode = "disabled"
    bench_runs = 0
    bench_avg_ms = ""
    bench_error = ""
    if full_console:
        bench_mode = "light"
        try:
            if (obj, method) in exec_smoke and py_has:
                bench_runs, bench_ms, bench_total_ms = benchmark_callable(lambda: exec_smoke[(obj, method)](py_inst))
                bench_avg_ms = f"{bench_ms:.4f}"
            else:
                bench_runs, bench_ms, bench_total_ms = benchmark_callable(lambda: hasattr(py_cls, method), budget_s=0.002, max_iters=50)
                bench_avg_ms = f"{bench_ms:.4f}"
            python_benchmark_total_ms += bench_total_ms
        except Exception as e:
            bench_mode = "error"
            bench_error = str(e)

    report.append(
        {
            "index": idx,
            "object": obj,
            "method": method,
            "python_available": "yes" if py_has else "no",
            "python_exec": py_exec,
            "python_error": py_error,
            "gopolars_equivalent": row["equivalent"],
            "gopolars_status": g_status,
            "gopolars_exec": g_exec,
            "priority": row["priority"],
            "benchmark_mode": bench_mode,
            "benchmark_runs": bench_runs,
            "benchmark_avg_ms": bench_avg_ms,
            "benchmark_error": bench_error,
        }
    )

csv_out.parent.mkdir(parents=True, exist_ok=True)
with csv_out.open("w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(
        f,
        fieldnames=[
            "index",
            "object",
            "method",
            "python_available",
            "python_exec",
            "python_error",
            "gopolars_equivalent",
            "gopolars_status",
            "gopolars_exec",
            "priority",
            "benchmark_mode",
            "benchmark_runs",
            "benchmark_avg_ms",
            "benchmark_error",
        ],
    )
    w.writeheader()
    w.writerows(report)

md_lines = []
md_lines.append("# Python Polars vs gopolars function comparison")
md_lines.append("")
md_lines.append(f"- parquet: `{parquet_path}`")
md_lines.append(f"- rows_compared: `{len(report)}`")
md_lines.append(f"- go_suite_ok: `{go_suite_ok}`")
md_lines.append(f"- go_version: `{go_version}`")
md_lines.append(f"- python_version: `{python_version}`")
md_lines.append(f"- go_suite_time_ms: `{go_suite_time_ms:.2f}`")
md_lines.append(f"- python_benchmark_total_ms: `{python_benchmark_total_ms:.2f}`")
md_lines.append(f"- full_report_mode: `{full_console}`")
go_speed_per_function_ms = go_suite_time_ms / len(report) if len(report) > 0 else 0.0
python_speed_per_function_ms = python_benchmark_total_ms / len(report) if len(report) > 0 else 0.0
implemented_count = sum(1 for r in report if r["gopolars_status"] == "реализовано")
go_speed_per_implemented_ms = go_suite_time_ms / implemented_count if implemented_count > 0 else 0.0
md_lines.append(
    f"- performance: `golang={go_speed_per_function_ms:.4f}ms_per_function/python={python_speed_per_function_ms:.4f}ms_per_function`"
)
md_lines.append(
    f"- golang_benchmark_note: `shared_suite_avg={go_speed_per_implemented_ms:.4f}ms (не per-method benchmark)`"
)
md_lines.append("")
md_lines.append("| # | Object | Method | Python | Python exec | gopolars | gopolars exec | Priority | Benchmark |")
md_lines.append("| --- | --- | --- | --- | --- | --- | --- | --- | --- |")
for r in report:
    if r["gopolars_status"] == "реализовано":
        python_bench = f"{r['benchmark_avg_ms']}ms" if r["benchmark_avg_ms"] else "-"
        benchmark_cell = f"golang=shared_suite_avg({go_speed_per_implemented_ms:.4f}ms)/python={python_bench}"
    else:
        benchmark_cell = f"{r['benchmark_mode']}:{r['benchmark_avg_ms']}ms/{r['benchmark_runs']}"
    md_lines.append(
        f"| {r['index']} | {r['object']} | `{r['method']}` | {r['python_available']} | {r['python_exec']} | {r['gopolars_status']} | {r['gopolars_exec']} | {r['priority']} | {benchmark_cell} |"
    )
md_out.parent.mkdir(parents=True, exist_ok=True)
md_out.write_text("\n".join(md_lines), encoding="utf-8")

python_exec_counts = collections.Counter(r["python_exec"] for r in report)
gopolars_status_counts = collections.Counter(r["gopolars_status"] for r in report)
priority_counts = collections.Counter(r["priority"] for r in report)
benchmark_mode_counts = collections.Counter(r["benchmark_mode"] for r in report)
top_not_implemented = [r for r in report if r["gopolars_status"] == "не реализовано"][:30]
top_implemented = [r for r in report if r["gopolars_status"] == "реализовано"][:30]
ascii_summary = render_ascii_table(
    ["metric", "value"],
    [
        ["parquet", str(parquet_path)],
        ["rows", str(len(report))],
        ["go_suite_ok", str(go_suite_ok)],
        ["go_version", go_version],
        ["python_version", python_version],
        ["go_suite_time_ms", f"{go_suite_time_ms:.2f}"],
        ["python_benchmark_total_ms", f"{python_benchmark_total_ms:.2f}"],
        ["full_report_mode", str(full_console)],
    ],
)
ascii_impl = render_ascii_table(
    ["#", "Object", "Method", "Status", "Bench ms", "Runs"],
    [[str(r["index"]), r["object"], r["method"], r["gopolars_status"], r["benchmark_avg_ms"] or "-", str(r["benchmark_runs"])] for r in top_implemented],
)
ascii_not_impl = render_ascii_table(
    ["#", "Object", "Method", "Status", "Priority", "Bench ms", "Runs"],
    [[str(r["index"]), r["object"], r["method"], r["gopolars_status"], r["priority"], r["benchmark_avg_ms"] or "-", str(r["benchmark_runs"])] for r in top_not_implemented],
)

print("=== comparison_report ===")
print(ascii_summary)
print(f"python_exec_counts={dict(python_exec_counts)}")
print(
    "gopolars_status_counts={"
    + ", ".join(
        [
            f"реализовано: {gopolars_status_counts.get('реализовано', 0)}",
            f"не реализовано: {gopolars_status_counts.get('не реализовано', 0)}",
        ]
    )
    + "}"
)
print(f"priority_counts={dict(priority_counts)}")
print(f"benchmark_mode_counts={dict(benchmark_mode_counts)}")
print("top_implemented_first_30:")
print(ascii_impl)
print("top_not_implemented_first_30:")
print(ascii_not_impl)

if full_console:
    md_lines.append("")
    md_lines.append("## ASCII Summary")
    md_lines.append("")
    md_lines.append("```text")
    md_lines.append(strip_ansi(ascii_summary))
    md_lines.append("```")
    md_lines.append("")
    md_lines.append(
        f"golang={go_speed_per_function_ms:.4f}ms_per_function/python={python_speed_per_function_ms:.4f}ms_per_function"
    )
    md_out.write_text("\n".join(md_lines), encoding="utf-8")
    print("=== markdown_report_begin ===")
    print(md_out.read_text(encoding="utf-8"))
    print("=== markdown_report_end ===")

print(f"comparison_written_csv={csv_out}")
print(f"comparison_written_md={md_out}")
print(f"rows={len(report)}")
PY
