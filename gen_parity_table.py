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
def get_go_methods(file_path):
    text = Path(file_path).read_text()
    methods = set()
    for m in re.finditer(r'func \([^)]*\) ([A-Z][A-Za-z0-9_]*)\(', text):
        methods.add(m.group(1))
    for m in re.finditer(r'^func ([A-Z][A-Za-z0-9_]*)\(', text, re.MULTILINE):
        if m.group(1) not in {'Expr', 'DataFrame', 'LazyFrame', 'Series'}:
            methods.add(m.group(1))
    return methods

root = Path("/Volumes/External HD/GolandProjects/gopolars")

df_methods = get_go_methods(root / "pkg/polars/dataframe.go")
lf_methods = get_go_methods(root / "pkg/polars/lazyframe.go")
s_methods = get_go_methods(root / "pkg/polars/series.go") | \
           get_go_methods(root / "pkg/polars/series_low_priority.go") | \
           get_go_methods(root / "pkg/polars/series_namespace.go")
expr_methods = get_go_methods(root / "pkg/expr/expr.go")

# 3. Функція перевірки відповідності
def snake_to_camel(snake):
    """Конвертує snake_case в CamelCase (спрощено)."""
    parts = snake.split('_')
    return ''.join(p.capitalize() for p in parts)

def check_match(py_methods, go_methods):
    result = {}
    for py in py_methods:
        camel = snake_to_camel(py)
        # Деякі спеціальні випадки
        if py == 'sql':
            result[py] = 'SQL' in go_methods or 'Sql' in go_methods
        elif py == 'dt':
            result[py] = 'Dt' in go_methods
        elif py == 'not_':
            result[py] = 'Not_' in go_methods or 'Not' in go_methods
        elif py == 'and_':
            result[py] = 'And_' in go_methods or 'And' in go_methods
        elif py == 'or_':
            result[py] = 'Or_' in go_methods or 'Or' in go_methods
        elif py == 'str':
            result[py] = 'Str' in go_methods
        elif py == 'cat':
            result[py] = 'Cat' in go_methods
        elif py == 'bin':
            result[py] = 'Bin' in go_methods
        elif py == 'list':
            result[py] = 'List' in go_methods or 'Arr' in go_methods
        elif py == 'struct':
            result[py] = 'Struct' in go_methods
        elif py == 'len':
            result[py] = 'Len' in go_methods
        elif py == 'name':
            result[py] = 'Name' in go_methods
        elif py == 'dtype':
            result[py] = 'DataType' in go_methods or 'Dtype' in go_methods
        else:
            result[py] = camel in go_methods
    return result

df_match = check_match(python_methods['DataFrame'], df_methods)
lf_match = check_match(python_methods['LazyFrame'], lf_methods)
s_match = check_match(python_methods['Series'], s_methods)
expr_match = check_match(python_methods['Expr'], expr_methods)

def print_table(name, match):
    print(f"\n## {name}\n")
    print("| Python (snake_case) | Go (CamelCase) | Статус |")
    print("|---|---|---|")
    implemented = 0
    for py in sorted(match.keys()):
        status = "✅" if match[py] else "❌"
        if match[py]:
            implemented += 1
        camel = snake_to_camel(py)
        # Спеціальні випадки для відображення
        if py == 'sql':
            camel = 'SQL / Sql'
        elif py == 'not_':
            camel = 'Not_'
        elif py == 'and_':
            camel = 'And_'
        elif py == 'or_':
            camel = 'Or_'
        elif py == 'dt':
            camel = 'Dt'
        elif py == 'str':
            camel = 'Str'
        elif py == 'cat':
            camel = 'Cat'
        elif py == 'bin':
            camel = 'Bin'
        elif py == 'list':
            camel = 'List / Arr'
        elif py == 'struct':
            camel = 'Struct'
        elif py == 'len':
            camel = 'Len'
        elif py == 'name':
            camel = 'Name'
        elif py == 'dtype':
            camel = 'DataType'
        print(f"| `{py}` | `{camel}` | {status} |")
    print(f"\n**Підсумок {name}:** {implemented}/{len(match)} реалізовано (~{int(100*implemented/len(match))}%).\n")
    return implemented, len(match)

print("# Таблиця відповідності gopolars ↔ Polars Python 1.41.0\n")
print("**Легенда:**")
print("- ✅ Реалізовано — метод присутній у gopolars")
print("- ❌ Не реалізовано — метод відсутній у gopolars")
print("---")

df_impl, df_total = print_table("DataFrame", df_match)
lf_impl, lf_total = print_table("LazyFrame", lf_match)
s_impl, s_total = print_table("Series", s_match)
expr_impl, expr_total = print_table("Expr", expr_match)

print("## Загальний підсумок\n")
print("| Клас | Реалізовано | Загалом | Відсоток |")
print("|---|---|---|---|")
print(f"| DataFrame | {df_impl} | {df_total} | ~{int(100*df_impl/df_total)}% |")
print(f"| LazyFrame | {lf_impl} | {lf_total} | ~{int(100*lf_impl/lf_total)}% |")
print(f"| Series | {s_impl} | {s_total} | ~{int(100*s_impl/s_total)}% |")
print(f"| Expr | {expr_impl} | {expr_total} | ~{int(100*expr_impl/expr_total)}% |")
