import re
import json
from pathlib import Path

# 1. Зчитуємо методи Python Polars (вже отримані раніше)
python_methods = {
    "DataFrame": [
        "approx_n_unique", "bottom_k", "cast", "clear", "clone", "collect_schema",
        "columns", "corr", "count", "describe", "deserialize", "drop", "drop_in_place",
        "drop_nans", "drop_nulls", "dtypes", "equals", "estimated_size", "explode",
        "extend", "fill_nan", "fill_null", "filter", "flags", "fold", "gather",
        "gather_every", "get_column", "get_column_index", "get_columns", "glimpse",
        "group_by", "group_by_dynamic", "hash_rows", "head", "height", "hstack",
        "insert_column", "interpolate", "is_duplicated", "is_empty", "is_unique",
        "item", "iter_columns", "iter_rows", "iter_slices", "join", "join_asof",
        "join_where", "lazy", "limit", "map_columns", "map_rows", "match_to_schema",
        "max", "max_horizontal", "mean", "mean_horizontal", "median", "melt",
        "merge_sorted", "min", "min_horizontal", "n_chunks", "n_unique", "null_count",
        "partition_by", "pipe", "pivot", "plot", "product", "quantile", "rechunk",
        "remove", "rename", "replace_column", "reverse", "rolling", "row", "rows",
        "rows_by_key", "sample", "schema", "select", "select_seq", "serialize",
        "set_sorted", "shape", "shift", "show", "shrink_to_fit", "slice", "sort",
        "sql", "std", "style", "sum", "sum_horizontal", "tail", "to_arrow",
        "to_dict", "to_dicts", "to_dummies", "to_init_repr", "to_jax", "to_numpy",
        "to_pandas", "to_series", "to_struct", "to_torch", "top_k", "transpose",
        "unique", "unnest", "unpivot", "unstack", "update", "upsample", "var",
        "vstack", "width", "with_columns", "with_columns_seq", "with_row_count",
        "with_row_index", "write_avro", "write_clipboard", "write_csv", "write_database",
        "write_delta", "write_excel", "write_iceberg", "write_ipc", "write_ipc_stream",
        "write_json", "write_ndjson", "write_parquet"
    ],
    "LazyFrame": [
        "approx_n_unique", "bottom_k", "cache", "cast", "clear", "clone", "collect",
        "collect_async", "collect_batches", "collect_schema", "columns", "count",
        "describe", "deserialize", "drop", "drop_nans", "drop_nulls", "dtypes",
        "execute", "explain", "explode", "fetch", "fill_nan", "fill_null", "filter",
        "first", "gather", "gather_every", "group_by", "group_by_dynamic", "head",
        "inspect", "interpolate", "join", "join_asof", "join_where", "last", "lazy",
        "limit", "map_batches", "match_to_schema", "max", "mean", "median", "melt",
        "merge_sorted", "min", "null_count", "pipe", "pipe_with_schema", "pivot",
        "profile", "quantile", "remote", "remove", "rename", "reverse", "rolling",
        "schema", "select", "select_seq", "serialize", "set_sorted", "shift", "show",
        "show_graph", "sink_batches", "sink_csv", "sink_delta", "sink_iceberg",
        "sink_ipc", "sink_ndjson", "sink_parquet", "slice", "sort", "sql", "std",
        "sum", "tail", "top_k", "unique", "unnest", "unpivot", "update", "var",
        "width", "with_columns", "with_columns_seq", "with_context", "with_row_count",
        "with_row_index"
    ],
    "Series": [
        "abs", "alias", "all", "any", "append", "approx_n_unique", "arccos", "arccosh",
        "arcsin", "arcsinh", "arctan", "arctanh", "arg_max", "arg_min", "arg_sort",
        "arg_true", "arg_unique", "arr", "backward_fill", "bin", "bitwise_and",
        "bitwise_count_ones", "bitwise_count_zeros", "bitwise_leading_ones",
        "bitwise_leading_zeros", "bitwise_or", "bitwise_trailing_ones",
        "bitwise_trailing_zeros", "bitwise_xor", "bottom_k", "bottom_k_by", "cast",
        "cat", "cbrt", "ceil", "chunk_lengths", "clear", "clip", "clone", "cos",
        "cosh", "cot", "count", "cum_count", "cum_max", "cum_min", "cum_prod",
        "cum_sum", "cumulative_eval", "cut", "describe", "diff", "dot", "drop_nans",
        "drop_nulls", "dt", "dtype", "entropy", "eq", "eq_missing", "equals",
        "estimated_size", "ewm_mean", "ewm_mean_by", "ewm_std", "ewm_var", "exp",
        "explode", "ext", "extend", "extend_constant", "fill_nan", "fill_null",
        "filter", "first", "flags", "floor", "forward_fill", "gather", "gather_every",
        "ge", "get_chunks", "gt", "has_nulls", "has_validity", "hash", "head",
        "hist", "implode", "index_of", "interpolate", "interpolate_by", "is_between",
        "is_close", "is_duplicated", "is_empty", "is_finite", "is_first_distinct",
        "is_in", "is_infinite", "is_last_distinct", "is_nan", "is_not_nan",
        "is_not_null", "is_null", "is_sorted", "is_unique", "item", "kurtosis",
        "last", "le", "len", "limit", "list", "log", "log10", "log1p", "lower_bound",
        "lt", "map_elements", "max", "max_by", "mean", "median", "min", "min_by",
        "mode", "n_chunks", "n_unique", "name", "nan_max", "nan_min", "ne",
        "ne_missing", "new_from_index", "not_", "null_count", "pct_change",
        "peak_max", "peak_min", "plot", "pow", "product", "qcut", "quantile",
        "rank", "rechunk", "reinterpret", "rename", "repeat_by", "replace",
        "replace_strict", "reshape", "reverse", "rle", "rle_id", "rolling_kurtosis",
        "rolling_map", "rolling_max", "rolling_max_by", "rolling_mean",
        "rolling_mean_by", "rolling_median", "rolling_median_by", "rolling_min",
        "rolling_min_by", "rolling_quantile", "rolling_quantile_by", "rolling_rank",
        "rolling_rank_by", "rolling_skew", "rolling_std", "rolling_std_by",
        "rolling_sum", "rolling_sum_by", "rolling_var", "rolling_var_by", "round",
        "round_sig_figs", "sample", "scatter", "search_sorted", "set", "set_sorted",
        "shape", "shift", "shrink_dtype", "shrink_to_fit", "shuffle", "sign", "sin",
        "sinh", "skew", "slice", "sort", "sql", "sqrt", "std", "str", "struct",
        "sum", "tail", "tan", "tanh", "to_arrow", "to_dummies", "to_frame",
        "to_init_repr", "to_jax", "to_list", "to_numpy", "to_pandas", "to_physical",
        "to_torch", "top_k", "top_k_by", "truncate", "unique", "unique_counts",
        "upper_bound", "value_counts", "var", "zip_with"
    ],
    "Expr": [
        "abs", "add", "agg_groups", "alias", "all", "and_", "any", "append",
        "approx_n_unique", "arccos", "arccosh", "arcsin", "arcsinh", "arctan",
        "arctanh", "arg_max", "arg_min", "arg_sort", "arg_true", "arg_unique",
        "arr", "backward_fill", "bin", "bitwise_and", "bitwise_count_ones",
        "bitwise_count_zeros", "bitwise_leading_ones", "bitwise_leading_zeros",
        "bitwise_or", "bitwise_trailing_ones", "bitwise_trailing_zeros",
        "bitwise_xor", "bottom_k", "bottom_k_by", "cast", "cat", "cbrt", "ceil",
        "clip", "cos", "cosh", "cot", "count", "cum_count", "cum_max", "cum_min",
        "cum_prod", "cum_sum", "cumulative_eval", "cut", "degrees", "deserialize",
        "diff", "dot", "drop_nans", "drop_nulls", "dt", "entropy", "eq",
        "eq_missing", "ewm_mean", "ewm_mean_by", "ewm_std", "ewm_var", "exclude",
        "exp", "explode", "ext", "extend_constant", "fill_nan", "fill_null",
        "filter", "first", "flatten", "floor", "floordiv", "forward_fill",
        "from_json", "gather", "gather_every", "ge", "get", "gt", "has_nulls",
        "hash", "head", "hist", "implode", "index_of", "inspect", "interpolate",
        "interpolate_by", "is_between", "is_close", "is_duplicated", "is_empty",
        "is_finite", "is_first_distinct", "is_in", "is_infinite", "is_last_distinct",
        "is_nan", "is_not_nan", "is_not_null", "is_null", "is_unique", "item",
        "kurtosis", "last", "le", "len", "limit", "list", "log", "log10", "log1p",
        "lower_bound", "lt", "map_batches", "map_elements", "max", "max_by",
        "mean", "median", "meta", "min", "min_by", "mod", "mode", "mul",
        "n_unique", "name", "nan_max", "nan_min", "ne", "ne_missing", "neg",
        "not_", "null_count", "or_", "over", "pct_change", "peak_max", "peak_min",
        "pipe", "pow", "product", "qcut", "quantile", "radians", "rank",
        "rechunk", "register_plugin", "reinterpret", "repeat_by", "replace",
        "replace_strict", "reshape", "reverse", "rle", "rle_id", "rolling",
        "rolling_kurtosis", "rolling_map", "rolling_max", "rolling_max_by",
        "rolling_mean", "rolling_mean_by", "rolling_median", "rolling_median_by",
        "rolling_min", "rolling_min_by", "rolling_quantile", "rolling_quantile_by",
        "rolling_rank", "rolling_rank_by", "rolling_skew", "rolling_std",
        "rolling_std_by", "rolling_sum", "rolling_sum_by", "rolling_var",
        "rolling_var_by", "round", "round_sig_figs", "sample", "search_sorted",
        "set_sorted", "shift", "shrink_dtype", "shuffle", "sign", "sin", "sinh",
        "skew", "slice", "sort", "sort_by", "sqrt", "std", "str", "struct",
        "sub", "sum", "tail", "tan", "tanh", "to_physical", "top_k", "top_k_by",
        "truediv", "truncate", "unique", "unique_counts", "upper_bound",
        "value_counts", "var", "where", "xor"
    ]
}

# 2. Зчитуємо методи gopolars
#
# Скануємо всі .go-файли пакета: і реалізації (func (recv) Name(...)),
# і декларації методів в інтерфейсах (рядок виду "    Name(...)" у types.go).
def get_go_methods(*globs):
    methods = set()
    for pattern in globs:
        for path in sorted(root.glob(pattern)):
            if path.name.endswith("_test.go"):
                continue
            text = path.read_text()
            # методи з ресивером: func (d DataFrame) Select(...)
            for m in re.finditer(r'func \([^)]*\) ([A-Z][A-Za-z0-9_]*)\(', text):
                methods.add(m.group(1))
            # вільні функції верхнього рівня: func Col(...)
            for m in re.finditer(r'^func ([A-Z][A-Za-z0-9_]*)\(', text, re.MULTILINE):
                methods.add(m.group(1))
            # декларації методів в інтерфейсах: рядок з відступом "Name("
            for m in re.finditer(r'^\s+([A-Z][A-Za-z0-9_]*)\(', text, re.MULTILINE):
                methods.add(m.group(1))
    return methods

# root відносно розташування цього скрипта — не залежить від машини.
root = Path(__file__).resolve().parent

# DataFrame / LazyFrame / Series інтерфейси та реалізації лежать у pkg/polars/*.go;
# Expr — у pkg/expr/*.go. Скануємо весь пакет, бо методи рознесені по файлах.
polars_methods = get_go_methods("pkg/polars/*.go")
expr_methods = get_go_methods("pkg/expr/*.go") | polars_methods
df_methods = polars_methods
lf_methods = polars_methods
s_methods = polars_methods | get_go_methods("pkg/expr/*.go")

# 3. Функція перевірки відповідності
def snake_to_camel(snake):
    """Конвертує snake_case в CamelCase (спрощено)."""
    parts = snake.strip('_').split('_')
    return ''.join(p.capitalize() for p in parts)

# Нормалізація для регістронезалежного співставлення без підкреслень:
# знімає різницю SinkCsv↔SinkCSV, Sql↔SQL, not_↔Not тощо.
def _norm(name):
    return name.replace('_', '').lower()

# Семантичні псевдоніми, де імена справді відрізняються (не лише регістр).
ALIASES = {
    'dtype': {'datatype', 'dtype'},
    'list': {'list', 'arr'},
    'arr': {'arr', 'list'},
}

def check_match(py_methods, go_methods):
    go_norm = {_norm(g) for g in go_methods}
    result = {}
    for py in py_methods:
        candidates = ALIASES.get(py, {_norm(py)})
        result[py] = any(c in go_norm for c in candidates)
    return result

df_match = check_match(python_methods['DataFrame'], df_methods)
lf_match = check_match(python_methods['LazyFrame'], lf_methods)
s_match = check_match(python_methods['Series'], s_methods)
expr_match = check_match(python_methods['Expr'], expr_methods)

OUT = []
def emit(line=""):
    OUT.append(line)

def print_table(name, match):
    emit(f"\n## {name}\n")
    emit("| Python (snake_case) | Go (CamelCase) | Статус |")
    emit("|---|---|---|")
    implemented = 0
    missing = []
    for py in sorted(match.keys()):
        status = "✅" if match[py] else "❌"
        if match[py]:
            implemented += 1
        else:
            missing.append(py)
        emit(f"| `{py}` | `{snake_to_camel(py)}` | {status} |")
    pct = int(round(100 * implemented / len(match))) if match else 100
    emit(f"\n**Підсумок {name}:** {implemented}/{len(match)} реалізовано (~{pct}%).")
    if missing:
        emit(f"\n**Не реалізовано ({len(missing)}):** " + ", ".join(f"`{m}`" for m in missing))
    emit()
    return implemented, len(match), missing

emit("# Таблиця відповідності gopolars ↔ Polars Python")
emit()
emit("> Згенеровано `gen_parity_table.py`. Не редагувати вручну.")
emit()
emit("**Легенда:**")
emit("- ✅ Реалізовано — метод присутній у gopolars")
emit("- ❌ Не реалізовано — метод відсутній у gopolars")
emit("\n---")

df_impl, df_total, df_miss = print_table("DataFrame", df_match)
lf_impl, lf_total, lf_miss = print_table("LazyFrame", lf_match)
s_impl, s_total, s_miss = print_table("Series", s_match)
expr_impl, expr_total, expr_miss = print_table("Expr", expr_match)

tot_impl = df_impl + lf_impl + s_impl + expr_impl
tot_all = df_total + lf_total + s_total + expr_total

emit("## Загальний підсумок\n")
emit("| Клас | Реалізовано | Загалом | Відсоток |")
emit("|---|---|---|---|")
emit(f"| DataFrame | {df_impl} | {df_total} | ~{int(round(100*df_impl/df_total))}% |")
emit(f"| LazyFrame | {lf_impl} | {lf_total} | ~{int(round(100*lf_impl/lf_total))}% |")
emit(f"| Series | {s_impl} | {s_total} | ~{int(round(100*s_impl/s_total))}% |")
emit(f"| Expr | {expr_impl} | {expr_total} | ~{int(round(100*expr_impl/expr_total))}% |")
emit(f"| **Разом** | **{tot_impl}** | **{tot_all}** | **~{round(100*tot_impl/tot_all, 1)}%** |")
emit()

report = "\n".join(OUT)
print(report)

out_path = root / "docs/parity/python_polars_full_matrix.md"
out_path.write_text(report)
print(f"\n[written] {out_path}")
