# Python Polars Full Method Matrix
Источник: официальная документация Python Polars (stable), выгрузка на 2026-03-13.
Статус рассчитан по публичному API gopolars. Приоритет присваивается только нереализованным методам.

Покрытие по этой автоматической матрице: 130/680.

## DataFrame (141 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `__array__` | — | не реализовано | low |
| `__arrow_c_stream__` | — | не реализовано | low |
| `__dataframe__` | — | не реализовано | low |
| `__getitem__` | — | не реализовано | low |
| `__setitem__` | — | не реализовано | low |
| `approx_n_unique` | DataFrame.approx_n_unique | реализовано | — |
| `bottom_k` | DataFrame.bottom_k | реализовано | — |
| `cast` | DataFrame.cast | реализовано | — |
| `clear` | DataFrame.clear | реализовано | — |
| `clone` | DataFrame.clone | реализовано | — |
| `collect_schema` | DataFrame.collect_schema | реализовано | — |
| `columns` | DataFrame.columns | реализовано | — |
| `corr` | DataFrame.corr | реализовано | — |
| `count` | DataFrame.count | реализовано | — |
| `describe` | DataFrame.describe | реализовано | — |
| `deserialize` | DataFrame.deserialize | реализовано | — |
| `drop` | DataFrame.drop | реализовано | — |
| `drop_in_place` | DataFrame.drop_in_place | реализовано | — |
| `drop_nans` | DataFrame.drop_na_ns | реализовано | — |
| `drop_nulls` | DataFrame.drop_nulls | реализовано | — |
| `dtypes` | DataFrame.dtypes | реализовано | — |
| `equals` | DataFrame.equals | реализовано | — |
| `estimated_size` | DataFrame.estimated_size | реализовано | — |
| `explode` | DataFrame.explode | реализовано | — |
| `extend` | DataFrame.extend | реализовано | — |
| `fill_nan` | DataFrame.fill_na_n | реализовано | — |
| `fill_null` | DataFrame.fill_null | реализовано | — |
| `filter` | DataFrame.filter | реализовано | — |
| `flags` | DataFrame.flags | реализовано | — |
| `fold` | DataFrame.fold | реализовано | — |
| `gather_every` | DataFrame.gather_every | реализовано | — |
| `get_column` | DataFrame.get_column | реализовано | — |
| `get_column_index` | DataFrame.get_column_index | реализовано | — |
| `get_columns` | DataFrame.get_columns | реализовано | — |
| `glimpse` | DataFrame.glimpse | реализовано | — |
| `group_by` | DataFrame.group_by | реализовано | — |
| `group_by_dynamic` | DataFrame.group_by_dynamic | реализовано | — |
| `hash_rows` | DataFrame.hash_rows | реализовано | — |
| `head` | DataFrame.head | реализовано | — |
| `height` | DataFrame.height | реализовано | — |
| `hstack` | DataFrame.hstack | реализовано | — |
| `insert_column` | DataFrame.insert_column | реализовано | — |
| `interpolate` | DataFrame.interpolate | реализовано | — |
| `is_duplicated` | — | не реализовано | low |
| `is_empty` | DataFrame.is_empty | реализовано | — |
| `is_unique` | — | не реализовано | low |
| `item` | — | не реализовано | low |
| `iter_columns` | — | не реализовано | low |
| `iter_rows` | — | не реализовано | low |
| `iter_slices` | — | не реализовано | low |
| `join` | DataFrame.join | реализовано | — |
| `join_asof` | — | не реализовано | low |
| `join_where` | — | не реализовано | low |
| `lazy` | DataFrame.lazy | реализовано | — |
| `limit` | DataFrame.limit | реализовано | — |
| `map_columns` | — | не реализовано | low |
| `map_rows` | — | не реализовано | low |
| `match_to_schema` | — | не реализовано | low |
| `max` | — | не реализовано | low |
| `max_horizontal` | — | не реализовано | low |
| `mean` | — | не реализовано | low |
| `mean_horizontal` | — | не реализовано | low |
| `median` | — | не реализовано | low |
| `melt` | DataFrame.melt | реализовано | — |
| `merge_sorted` | — | не реализовано | low |
| `min` | — | не реализовано | low |
| `min_horizontal` | — | не реализовано | low |
| `n_chunks` | — | не реализовано | low |
| `n_unique` | DataFrame.n_unique | реализовано | — |
| `null_count` | DataFrame.null_count | реализовано | — |
| `partition_by` | — | не реализовано | medium |
| `pipe` | — | не реализовано | low |
| `pivot` | DataFrame.pivot | реализовано | — |
| `plot` | — | не реализовано | low |
| `product` | — | не реализовано | low |
| `quantile` | — | не реализовано | low |
| `rechunk` | — | не реализовано | medium |
| `remove` | — | не реализовано | low |
| `rename` | DataFrame.rename | реализовано | — |
| `replace_column` | — | не реализовано | low |
| `reverse` | — | не реализовано | low |
| `rolling` | — | не реализовано | low |
| `row` | — | не реализовано | low |
| `rows` | — | не реализовано | low |
| `rows_by_key` | — | не реализовано | low |
| `sample` | DataFrame.sample | реализовано | — |
| `schema` | DataFrame.schema | реализовано | — |
| `select` | DataFrame.select | реализовано | — |
| `select_seq` | — | не реализовано | low |
| `serialize` | — | не реализовано | low |
| `set_sorted` | — | не реализовано | low |
| `shape` | — | не реализовано | low |
| `shift` | — | не реализовано | low |
| `show` | — | не реализовано | low |
| `shrink_to_fit` | — | не реализовано | low |
| `slice` | DataFrame.slice | реализовано | — |
| `sort` | DataFrame.sort | реализовано | — |
| `sql` | — | не реализовано | low |
| `std` | — | не реализовано | low |
| `style` | — | не реализовано | low |
| `sum` | — | не реализовано | low |
| `sum_horizontal` | — | не реализовано | low |
| `tail` | DataFrame.tail | реализовано | — |
| `to_arrow` | DataFrame.to_arrow | реализовано | — |
| `to_dict` | — | не реализовано | low |
| `to_dicts` | DataFrame.to_dicts | реализовано | — |
| `to_dummies` | — | не реализовано | low |
| `to_init_repr` | — | не реализовано | low |
| `to_jax` | — | не реализовано | low |
| `to_numpy` | — | не реализовано | medium |
| `to_pandas` | — | не реализовано | medium |
| `to_series` | — | не реализовано | low |
| `to_struct` | — | не реализовано | low |
| `to_torch` | — | не реализовано | low |
| `top_k` | — | не реализовано | low |
| `transpose` | — | не реализовано | low |
| `unique` | DataFrame.unique | реализовано | — |
| `unnest` | — | не реализовано | low |
| `unpivot` | — | не реализовано | low |
| `unstack` | — | не реализовано | low |
| `update` | — | не реализовано | low |
| `upsample` | — | не реализовано | medium |
| `var` | — | не реализовано | low |
| `vstack` | — | не реализовано | low |
| `width` | DataFrame.width | реализовано | — |
| `with_columns` | DataFrame.with_columns | реализовано | — |
| `with_columns_seq` | — | не реализовано | low |
| `with_row_count` | DataFrame.with_row_count | реализовано | — |
| `with_row_index` | DataFrame.with_row_index | реализовано | — |
| `write_avro` | — | не реализовано | low |
| `write_clipboard` | — | не реализовано | low |
| `write_csv` | — | не реализовано | low |
| `write_database` | — | не реализовано | low |
| `write_delta` | — | не реализовано | low |
| `write_excel` | — | не реализовано | low |
| `write_iceberg` | — | не реализовано | low |
| `write_ipc` | — | не реализовано | low |
| `write_ipc_stream` | — | не реализовано | low |
| `write_json` | — | не реализовано | low |
| `write_ndjson` | — | не реализовано | low |
| `write_parquet` | DataFrame.write_parquet | реализовано | — |

Итог для DataFrame: реализовано 61 из 141.

## LazyFrame (89 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `__getitem__` | — | не реализовано | low |
| `approx_n_unique` | — | не реализовано | low |
| `bottom_k` | — | не реализовано | low |
| `cache` | — | не реализовано | medium |
| `cast` | — | не реализовано | low |
| `clear` | — | не реализовано | low |
| `clone` | — | не реализовано | low |
| `collect` | LazyFrame.collect | реализовано | — |
| `collect_async` | LazyFrame.collect_async | реализовано | — |
| `collect_batches` | LazyFrame.collect_batches | реализовано | — |
| `collect_schema` | — | не реализовано | low |
| `columns` | — | не реализовано | low |
| `count` | — | не реализовано | low |
| `describe` | — | не реализовано | low |
| `deserialize` | — | не реализовано | medium |
| `drop` | — | не реализовано | low |
| `drop_nans` | — | не реализовано | low |
| `drop_nulls` | LazyFrame.drop_nulls | реализовано | — |
| `dtypes` | — | не реализовано | low |
| `explain` | LazyFrame.explain | реализовано | — |
| `explode` | LazyFrame.explode | реализовано | — |
| `fill_nan` | — | не реализовано | low |
| `fill_null` | LazyFrame.fill_null | реализовано | — |
| `filter` | LazyFrame.filter | реализовано | — |
| `first` | — | не реализовано | low |
| `gather_every` | — | не реализовано | low |
| `group_by` | LazyFrame.group_by | реализовано | — |
| `group_by_dynamic` | LazyFrame.group_by_dynamic | реализовано | — |
| `head` | — | не реализовано | low |
| `inspect` | LazyFrame.inspect | реализовано | — |
| `interpolate` | — | не реализовано | low |
| `join` | LazyFrame.join | реализовано | — |
| `join_asof` | — | не реализовано | low |
| `join_where` | LazyFrame.join_where | реализовано | — |
| `last` | — | не реализовано | low |
| `lazy` | — | не реализовано | low |
| `limit` | LazyFrame.limit | реализовано | — |
| `map_batches` | — | не реализовано | low |
| `match_to_schema` | — | не реализовано | low |
| `max` | — | не реализовано | low |
| `mean` | — | не реализовано | low |
| `median` | — | не реализовано | low |
| `melt` | LazyFrame.melt | реализовано | — |
| `merge_sorted` | — | не реализовано | low |
| `min` | — | не реализовано | low |
| `null_count` | — | не реализовано | low |
| `pipe` | — | не реализовано | low |
| `pipe_with_schema` | — | не реализовано | low |
| `pivot` | LazyFrame.pivot | реализовано | — |
| `profile` | LazyFrame.profile | реализовано | — |
| `quantile` | — | не реализовано | low |
| `remote` | — | не реализовано | medium |
| `remove` | — | не реализовано | low |
| `rename` | — | не реализовано | low |
| `reverse` | — | не реализовано | low |
| `rolling` | — | не реализовано | low |
| `schema` | — | не реализовано | low |
| `select` | LazyFrame.select | реализовано | — |
| `select_seq` | — | не реализовано | low |
| `serialize` | — | не реализовано | medium |
| `set_sorted` | — | не реализовано | low |
| `shift` | — | не реализовано | low |
| `show` | — | не реализовано | low |
| `show_graph` | — | не реализовано | medium |
| `sink_batches` | — | не реализовано | medium |
| `sink_csv` | — | не реализовано | low |
| `sink_delta` | — | не реализовано | low |
| `sink_iceberg` | — | не реализовано | low |
| `sink_ipc` | — | не реализовано | low |
| `sink_ndjson` | LazyFrame.sink_n_d_j_s_o_n | реализовано | — |
| `sink_parquet` | LazyFrame.sink_parquet | реализовано | — |
| `slice` | LazyFrame.slice | реализовано | — |
| `sort` | LazyFrame.sort | реализовано | — |
| `sql` | LazyFrame.s_q_l | реализовано | — |
| `std` | — | не реализовано | low |
| `sum` | — | не реализовано | low |
| `tail` | — | не реализовано | low |
| `top_k` | — | не реализовано | low |
| `unique` | LazyFrame.unique | реализовано | — |
| `unnest` | — | не реализовано | low |
| `unpivot` | — | не реализовано | low |
| `update` | — | не реализовано | low |
| `var` | — | не реализовано | low |
| `width` | — | не реализовано | low |
| `with_columns` | LazyFrame.with_columns | реализовано | — |
| `with_columns_seq` | — | не реализовано | low |
| `with_context` | — | не реализовано | low |
| `with_row_count` | — | не реализовано | low |
| `with_row_index` | — | не реализовано | low |

Итог для LazyFrame: реализовано 25 из 89.

## Expr (217 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `abs` | — | не реализовано | low |
| `add` | Expr.add | реализовано | — |
| `agg_groups` | — | не реализовано | low |
| `alias` | Expr.alias | реализовано | — |
| `all` | — | не реализовано | low |
| `and_` | — | не реализовано | low |
| `any` | — | не реализовано | low |
| `append` | — | не реализовано | low |
| `approx_n_unique` | — | не реализовано | low |
| `arccos` | — | не реализовано | low |
| `arccosh` | — | не реализовано | low |
| `arcsin` | — | не реализовано | low |
| `arcsinh` | — | не реализовано | low |
| `arctan` | — | не реализовано | low |
| `arctanh` | — | не реализовано | low |
| `arg_max` | — | не реализовано | low |
| `arg_min` | — | не реализовано | low |
| `arg_sort` | — | не реализовано | low |
| `arg_true` | — | не реализовано | low |
| `arg_unique` | — | не реализовано | low |
| `arr` | — | не реализовано | low |
| `backward_fill` | — | не реализовано | low |
| `bin` | — | не реализовано | low |
| `bitwise_and` | — | не реализовано | low |
| `bitwise_count_ones` | — | не реализовано | low |
| `bitwise_count_zeros` | — | не реализовано | low |
| `bitwise_leading_ones` | — | не реализовано | low |
| `bitwise_leading_zeros` | — | не реализовано | low |
| `bitwise_or` | — | не реализовано | low |
| `bitwise_trailing_ones` | — | не реализовано | low |
| `bitwise_trailing_zeros` | — | не реализовано | low |
| `bitwise_xor` | — | не реализовано | low |
| `bottom_k` | — | не реализовано | low |
| `bottom_k_by` | — | не реализовано | low |
| `cast` | Expr.cast | реализовано | — |
| `cat` | — | не реализовано | low |
| `cbrt` | — | не реализовано | low |
| `ceil` | — | не реализовано | low |
| `clip` | — | не реализовано | medium |
| `cos` | — | не реализовано | low |
| `cosh` | — | не реализовано | low |
| `cot` | — | не реализовано | low |
| `count` | — | не реализовано | low |
| `cum_count` | Expr.cum_count | реализовано | — |
| `cum_max` | — | не реализовано | low |
| `cum_min` | — | не реализовано | low |
| `cum_prod` | — | не реализовано | low |
| `cum_sum` | Expr.cum_sum | реализовано | — |
| `cumulative_eval` | — | не реализовано | low |
| `cut` | — | не реализовано | low |
| `degrees` | — | не реализовано | low |
| `deserialize` | — | не реализовано | low |
| `diff` | — | не реализовано | low |
| `dot` | — | не реализовано | low |
| `drop_nans` | — | не реализовано | low |
| `drop_nulls` | — | не реализовано | low |
| `dt` | — | не реализовано | medium |
| `entropy` | — | не реализовано | low |
| `eq` | Expr.eq | реализовано | — |
| `eq_missing` | — | не реализовано | low |
| `ewm_mean` | — | не реализовано | low |
| `ewm_mean_by` | — | не реализовано | low |
| `ewm_std` | — | не реализовано | low |
| `ewm_var` | — | не реализовано | low |
| `exclude` | — | не реализовано | low |
| `exp` | — | не реализовано | low |
| `explode` | — | не реализовано | low |
| `ext` | — | не реализовано | low |
| `extend_constant` | — | не реализовано | low |
| `fill_nan` | Expr.fill_na_n | реализовано | — |
| `fill_null` | Expr.fill_null | реализовано | — |
| `filter` | — | не реализовано | low |
| `first` | — | не реализовано | low |
| `flatten` | — | не реализовано | low |
| `floor` | — | не реализовано | low |
| `floordiv` | — | не реализовано | low |
| `forward_fill` | — | не реализовано | low |
| `from_json` | — | не реализовано | low |
| `gather` | — | не реализовано | low |
| `gather_every` | — | не реализовано | low |
| `ge` | Expr.ge | реализовано | — |
| `get` | — | не реализовано | low |
| `gt` | Expr.gt | реализовано | — |
| `has_nulls` | — | не реализовано | low |
| `hash` | — | не реализовано | low |
| `head` | — | не реализовано | low |
| `hist` | — | не реализовано | low |
| `implode` | — | не реализовано | low |
| `index_of` | — | не реализовано | low |
| `inspect` | — | не реализовано | low |
| `interpolate` | — | не реализовано | low |
| `interpolate_by` | — | не реализовано | low |
| `is_between` | — | не реализовано | low |
| `is_close` | — | не реализовано | low |
| `is_duplicated` | — | не реализовано | low |
| `is_finite` | — | не реализовано | low |
| `is_first_distinct` | — | не реализовано | low |
| `is_in` | — | не реализовано | low |
| `is_infinite` | — | не реализовано | low |
| `is_last_distinct` | — | не реализовано | low |
| `is_nan` | — | не реализовано | low |
| `is_not_nan` | — | не реализовано | low |
| `is_not_null` | Expr.is_not_null | реализовано | — |
| `is_null` | Expr.is_null | реализовано | — |
| `is_unique` | — | не реализовано | low |
| `item` | — | не реализовано | low |
| `kurtosis` | — | не реализовано | low |
| `last` | — | не реализовано | low |
| `le` | Expr.le | реализовано | — |
| `len` | Expr.list_len | реализовано | — |
| `limit` | — | не реализовано | low |
| `list` | — | не реализовано | medium |
| `log` | — | не реализовано | medium |
| `log10` | — | не реализовано | low |
| `log1p` | — | не реализовано | low |
| `lower_bound` | — | не реализовано | low |
| `lt` | Expr.lt | реализовано | — |
| `map_batches` | — | не реализовано | low |
| `map_elements` | — | не реализовано | low |
| `max` | — | не реализовано | low |
| `max_by` | — | не реализовано | low |
| `mean` | — | не реализовано | low |
| `median` | — | не реализовано | low |
| `meta` | — | не реализовано | low |
| `min` | — | не реализовано | low |
| `min_by` | — | не реализовано | low |
| `mod` | — | не реализовано | low |
| `mode` | — | не реализовано | low |
| `mul` | Expr.mul | реализовано | — |
| `n_unique` | — | не реализовано | low |
| `name` | Expr.name | реализовано | — |
| `nan_max` | — | не реализовано | low |
| `nan_min` | — | не реализовано | low |
| `ne` | Expr.ne | реализовано | — |
| `ne_missing` | — | не реализовано | low |
| `neg` | — | не реализовано | low |
| `not_` | — | не реализовано | low |
| `null_count` | — | не реализовано | low |
| `or_` | — | не реализовано | low |
| `over` | Expr.over | реализовано | — |
| `pct_change` | — | не реализовано | low |
| `peak_max` | — | не реализовано | low |
| `peak_min` | — | не реализовано | low |
| `pipe` | — | не реализовано | low |
| `pow` | — | не реализовано | medium |
| `product` | — | не реализовано | low |
| `qcut` | — | не реализовано | low |
| `quantile` | — | не реализовано | low |
| `radians` | — | не реализовано | low |
| `rank` | Expr.rank | реализовано | — |
| `rechunk` | — | не реализовано | low |
| `reinterpret` | — | не реализовано | low |
| `repeat_by` | — | не реализовано | low |
| `replace` | Expr.replace | реализовано | — |
| `replace_strict` | — | не реализовано | low |
| `reshape` | — | не реализовано | low |
| `reverse` | — | не реализовано | low |
| `rle` | — | не реализовано | low |
| `rle_id` | — | не реализовано | low |
| `rolling` | — | не реализовано | low |
| `rolling_kurtosis` | — | не реализовано | low |
| `rolling_map` | — | не реализовано | low |
| `rolling_max` | Expr.rolling_max | реализовано | — |
| `rolling_max_by` | — | не реализовано | low |
| `rolling_mean` | Expr.rolling_mean | реализовано | — |
| `rolling_mean_by` | — | не реализовано | low |
| `rolling_median` | — | не реализовано | low |
| `rolling_median_by` | — | не реализовано | low |
| `rolling_min` | Expr.rolling_min | реализовано | — |
| `rolling_min_by` | — | не реализовано | low |
| `rolling_quantile` | — | не реализовано | low |
| `rolling_quantile_by` | — | не реализовано | low |
| `rolling_rank` | — | не реализовано | low |
| `rolling_rank_by` | — | не реализовано | low |
| `rolling_skew` | — | не реализовано | low |
| `rolling_std` | Expr.rolling_std | реализовано | — |
| `rolling_std_by` | — | не реализовано | low |
| `rolling_sum` | Expr.rolling_sum | реализовано | — |
| `rolling_sum_by` | — | не реализовано | low |
| `rolling_var` | — | не реализовано | low |
| `rolling_var_by` | — | не реализовано | low |
| `round` | — | не реализовано | medium |
| `round_sig_figs` | — | не реализовано | low |
| `sample` | — | не реализовано | low |
| `search_sorted` | — | не реализовано | low |
| `set_sorted` | — | не реализовано | low |
| `shift` | — | не реализовано | low |
| `shrink_dtype` | — | не реализовано | low |
| `shuffle` | — | не реализовано | low |
| `sign` | — | не реализовано | low |
| `sin` | — | не реализовано | low |
| `sinh` | — | не реализовано | low |
| `skew` | — | не реализовано | low |
| `slice` | — | не реализовано | low |
| `sort` | — | не реализовано | low |
| `sort_by` | — | не реализовано | low |
| `sqrt` | — | не реализовано | medium |
| `std` | — | не реализовано | low |
| `str` | — | не реализовано | medium |
| `struct` | — | не реализовано | medium |
| `sub` | Expr.sub | реализовано | — |
| `sum` | — | не реализовано | low |
| `tail` | — | не реализовано | low |
| `tan` | — | не реализовано | low |
| `tanh` | — | не реализовано | low |
| `to_physical` | — | не реализовано | low |
| `top_k` | — | не реализовано | low |
| `top_k_by` | — | не реализовано | low |
| `truediv` | — | не реализовано | low |
| `truncate` | — | не реализовано | low |
| `unique` | — | не реализовано | low |
| `unique_counts` | — | не реализовано | low |
| `upper_bound` | — | не реализовано | low |
| `value_counts` | — | не реализовано | low |
| `var` | — | не реализовано | low |
| `where` | — | не реализовано | low |
| `xor` | — | не реализовано | low |

Итог для Expr: реализовано 27 из 217.

## Series (226 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `__array__` | — | не реализовано | low |
| `__arrow_c_stream__` | — | не реализовано | low |
| `__getitem__` | — | не реализовано | low |
| `abs` | — | не реализовано | low |
| `alias` | — | не реализовано | low |
| `all` | — | не реализовано | low |
| `any` | — | не реализовано | low |
| `append` | — | не реализовано | low |
| `approx_n_unique` | — | не реализовано | low |
| `arccos` | — | не реализовано | low |
| `arccosh` | — | не реализовано | low |
| `arcsin` | — | не реализовано | low |
| `arcsinh` | — | не реализовано | low |
| `arctan` | — | не реализовано | low |
| `arctanh` | — | не реализовано | low |
| `arg_max` | — | не реализовано | low |
| `arg_min` | — | не реализовано | low |
| `arg_sort` | — | не реализовано | low |
| `arg_true` | — | не реализовано | low |
| `arg_unique` | — | не реализовано | low |
| `arr` | — | не реализовано | low |
| `backward_fill` | — | не реализовано | low |
| `bin` | — | не реализовано | low |
| `bitwise_and` | — | не реализовано | low |
| `bitwise_count_ones` | — | не реализовано | low |
| `bitwise_count_zeros` | — | не реализовано | low |
| `bitwise_leading_ones` | — | не реализовано | low |
| `bitwise_leading_zeros` | — | не реализовано | low |
| `bitwise_or` | — | не реализовано | low |
| `bitwise_trailing_ones` | — | не реализовано | low |
| `bitwise_trailing_zeros` | — | не реализовано | low |
| `bitwise_xor` | — | не реализовано | low |
| `bottom_k` | — | не реализовано | low |
| `bottom_k_by` | — | не реализовано | low |
| `cast` | Series.cast | реализовано | — |
| `cat` | — | не реализовано | low |
| `cbrt` | — | не реализовано | low |
| `ceil` | — | не реализовано | low |
| `chunk_lengths` | — | не реализовано | low |
| `clear` | — | не реализовано | low |
| `clip` | — | не реализовано | low |
| `clone` | — | не реализовано | low |
| `cos` | — | не реализовано | low |
| `cosh` | — | не реализовано | low |
| `cot` | — | не реализовано | low |
| `count` | — | не реализовано | low |
| `cum_count` | — | не реализовано | low |
| `cum_max` | — | не реализовано | low |
| `cum_min` | — | не реализовано | low |
| `cum_prod` | — | не реализовано | low |
| `cum_sum` | — | не реализовано | low |
| `cumulative_eval` | — | не реализовано | low |
| `cut` | — | не реализовано | low |
| `describe` | — | не реализовано | medium |
| `diff` | — | не реализовано | low |
| `dot` | — | не реализовано | low |
| `drop_nans` | — | не реализовано | high |
| `drop_nulls` | Series.drop_nulls | реализовано | — |
| `dt` | — | не реализовано | low |
| `dtype` | — | не реализовано | low |
| `entropy` | — | не реализовано | low |
| `eq` | Series.eq | реализовано | — |
| `eq_missing` | — | не реализовано | low |
| `equals` | — | не реализовано | low |
| `estimated_size` | — | не реализовано | low |
| `ewm_mean` | — | не реализовано | low |
| `ewm_mean_by` | — | не реализовано | low |
| `ewm_std` | — | не реализовано | low |
| `ewm_var` | — | не реализовано | low |
| `exp` | — | не реализовано | low |
| `explode` | — | не реализовано | low |
| `ext` | — | не реализовано | low |
| `extend` | — | не реализовано | low |
| `extend_constant` | — | не реализовано | low |
| `fill_nan` | — | не реализовано | high |
| `fill_null` | Series.fill_null | реализовано | — |
| `filter` | — | не реализовано | low |
| `first` | — | не реализовано | low |
| `flags` | — | не реализовано | low |
| `floor` | — | не реализовано | low |
| `forward_fill` | — | не реализовано | low |
| `gather` | — | не реализовано | low |
| `gather_every` | — | не реализовано | low |
| `ge` | Series.ge | реализовано | — |
| `get_chunks` | — | не реализовано | low |
| `gt` | Series.gt | реализовано | — |
| `has_nulls` | — | не реализовано | low |
| `has_validity` | — | не реализовано | low |
| `hash` | — | не реализовано | low |
| `head` | — | не реализовано | low |
| `hist` | — | не реализовано | medium |
| `implode` | — | не реализовано | low |
| `index_of` | — | не реализовано | low |
| `interpolate` | — | не реализовано | medium |
| `interpolate_by` | — | не реализовано | low |
| `is_between` | — | не реализовано | low |
| `is_close` | — | не реализовано | low |
| `is_duplicated` | — | не реализовано | low |
| `is_empty` | — | не реализовано | low |
| `is_finite` | — | не реализовано | low |
| `is_first_distinct` | — | не реализовано | low |
| `is_in` | — | не реализовано | low |
| `is_infinite` | — | не реализовано | low |
| `is_last_distinct` | — | не реализовано | low |
| `is_nan` | — | не реализовано | low |
| `is_not_nan` | — | не реализовано | low |
| `is_not_null` | Series.is_not_null | реализовано | — |
| `is_null` | Series.is_null | реализовано | — |
| `is_sorted` | — | не реализовано | low |
| `is_unique` | — | не реализовано | low |
| `item` | — | не реализовано | low |
| `kurtosis` | — | не реализовано | low |
| `last` | — | не реализовано | low |
| `le` | Series.le | реализовано | — |
| `len` | Series.len | реализовано | — |
| `limit` | — | не реализовано | low |
| `list` | — | не реализовано | low |
| `log` | — | не реализовано | low |
| `log10` | — | не реализовано | low |
| `log1p` | — | не реализовано | low |
| `lower_bound` | — | не реализовано | low |
| `lt` | Series.lt | реализовано | — |
| `map_elements` | — | не реализовано | low |
| `max` | — | не реализовано | low |
| `max_by` | — | не реализовано | low |
| `mean` | — | не реализовано | low |
| `median` | — | не реализовано | low |
| `min` | — | не реализовано | low |
| `min_by` | — | не реализовано | low |
| `mode` | — | не реализовано | low |
| `n_chunks` | — | не реализовано | low |
| `n_unique` | — | не реализовано | low |
| `name` | Series.name | реализовано | — |
| `nan_max` | — | не реализовано | low |
| `nan_min` | — | не реализовано | low |
| `ne` | Series.ne | реализовано | — |
| `ne_missing` | — | не реализовано | low |
| `new_from_index` | — | не реализовано | low |
| `not_` | — | не реализовано | low |
| `null_count` | — | не реализовано | high |
| `pct_change` | — | не реализовано | low |
| `peak_max` | — | не реализовано | low |
| `peak_min` | — | не реализовано | low |
| `plot` | — | не реализовано | low |
| `pow` | — | не реализовано | low |
| `product` | — | не реализовано | low |
| `qcut` | — | не реализовано | low |
| `quantile` | — | не реализовано | low |
| `rank` | — | не реализовано | low |
| `rechunk` | — | не реализовано | low |
| `reinterpret` | — | не реализовано | low |
| `rename` | — | не реализовано | low |
| `repeat_by` | — | не реализовано | low |
| `replace` | — | не реализовано | low |
| `replace_strict` | — | не реализовано | low |
| `reshape` | — | не реализовано | low |
| `reverse` | — | не реализовано | low |
| `rle` | — | не реализовано | low |
| `rle_id` | — | не реализовано | low |
| `rolling_kurtosis` | — | не реализовано | low |
| `rolling_map` | — | не реализовано | low |
| `rolling_max` | — | не реализовано | high |
| `rolling_max_by` | — | не реализовано | low |
| `rolling_mean` | — | не реализовано | high |
| `rolling_mean_by` | — | не реализовано | low |
| `rolling_median` | — | не реализовано | low |
| `rolling_median_by` | — | не реализовано | low |
| `rolling_min` | — | не реализовано | high |
| `rolling_min_by` | — | не реализовано | low |
| `rolling_quantile` | — | не реализовано | low |
| `rolling_quantile_by` | — | не реализовано | low |
| `rolling_rank` | — | не реализовано | low |
| `rolling_rank_by` | — | не реализовано | low |
| `rolling_skew` | — | не реализовано | low |
| `rolling_std` | — | не реализовано | low |
| `rolling_std_by` | — | не реализовано | low |
| `rolling_sum` | — | не реализовано | high |
| `rolling_sum_by` | — | не реализовано | low |
| `rolling_var` | — | не реализовано | low |
| `rolling_var_by` | — | не реализовано | low |
| `round` | — | не реализовано | low |
| `round_sig_figs` | — | не реализовано | low |
| `sample` | — | не реализовано | low |
| `scatter` | — | не реализовано | low |
| `search_sorted` | — | не реализовано | low |
| `set` | — | не реализовано | low |
| `set_sorted` | — | не реализовано | low |
| `shape` | — | не реализовано | low |
| `shift` | — | не реализовано | low |
| `shrink_dtype` | — | не реализовано | low |
| `shrink_to_fit` | — | не реализовано | low |
| `shuffle` | — | не реализовано | low |
| `sign` | — | не реализовано | low |
| `sin` | — | не реализовано | low |
| `sinh` | — | не реализовано | low |
| `skew` | — | не реализовано | low |
| `slice` | — | не реализовано | low |
| `sort` | — | не реализовано | low |
| `sql` | — | не реализовано | low |
| `sqrt` | — | не реализовано | low |
| `std` | — | не реализовано | low |
| `str` | — | не реализовано | low |
| `struct` | — | не реализовано | low |
| `sum` | — | не реализовано | low |
| `tail` | — | не реализовано | low |
| `tan` | — | не реализовано | low |
| `tanh` | — | не реализовано | low |
| `to_arrow` | — | не реализовано | low |
| `to_dummies` | — | не реализовано | low |
| `to_frame` | — | не реализовано | low |
| `to_init_repr` | — | не реализовано | low |
| `to_jax` | — | не реализовано | low |
| `to_list` | — | не реализовано | high |
| `to_numpy` | — | не реализовано | medium |
| `to_pandas` | — | не реализовано | medium |
| `to_physical` | — | не реализовано | low |
| `to_torch` | — | не реализовано | low |
| `top_k` | — | не реализовано | low |
| `top_k_by` | — | не реализовано | low |
| `truncate` | — | не реализовано | low |
| `unique` | — | не реализовано | low |
| `unique_counts` | — | не реализовано | low |
| `upper_bound` | — | не реализовано | low |
| `value_counts` | — | не реализовано | low |
| `var` | — | не реализовано | low |
| `zip_with` | — | не реализовано | low |

Итог для Series: реализовано 13 из 226.

## SQLContext (7 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `execute` | polars.SQLFromDataFrame / ParseSQL / SQLContext.execute | реализовано | — |
| `execute_global` | — | не реализовано | medium |
| `register` | SQLContext.register | реализовано | — |
| `register_globals` | — | не реализовано | medium |
| `register_many` | — | не реализовано | high |
| `tables` | SQLContext.tables | реализовано | — |
| `unregister` | SQLContext.unregister | реализовано | — |

Итог для SQLContext: реализовано 4 из 7.

## Top-30 функций для v0.7 (по приоритету реализации)
| # | Объект | Python Polars method | Приоритет |
| --- | --- | --- | --- |
| 1 | SQLContext | `register_many` | high |
| 2 | Series | `drop_nans` | high |
| 3 | Series | `fill_nan` | high |
| 4 | Series | `null_count` | high |
| 5 | Series | `rolling_max` | high |
| 6 | Series | `rolling_mean` | high |
| 7 | Series | `rolling_min` | high |
| 8 | Series | `rolling_sum` | high |
| 9 | Series | `to_list` | high |
| 10 | DataFrame | `partition_by` | medium |
| 11 | DataFrame | `rechunk` | medium |
| 12 | DataFrame | `to_numpy` | medium |
| 13 | DataFrame | `to_pandas` | medium |
| 14 | DataFrame | `upsample` | medium |
| 15 | Expr | `clip` | medium |
| 16 | Expr | `dt` | medium |
| 17 | Expr | `list` | medium |
| 18 | Expr | `log` | medium |
| 19 | Expr | `pow` | medium |
| 20 | Expr | `round` | medium |
| 21 | Expr | `sqrt` | medium |
| 22 | Expr | `str` | medium |
| 23 | Expr | `struct` | medium |
| 24 | LazyFrame | `cache` | medium |
| 25 | LazyFrame | `deserialize` | medium |
| 26 | LazyFrame | `remote` | medium |
| 27 | LazyFrame | `serialize` | medium |
| 28 | LazyFrame | `show_graph` | medium |
| 29 | LazyFrame | `sink_batches` | medium |
| 30 | SQLContext | `execute_global` | medium |
