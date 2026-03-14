# Python Polars Full Method Matrix
Источник: официальная документация Python Polars (stable), выгрузка на 2026-03-13.
Статус рассчитан по публичному API gopolars. Приоритет присваивается только нереализованным методам.

Покрытие по этой автоматической матрице: 369/680.

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
| `is_duplicated` | DataFrame.is_duplicated | реализовано | — |
| `is_empty` | DataFrame.is_empty | реализовано | — |
| `is_unique` | DataFrame.is_unique | реализовано | — |
| `item` | DataFrame.item | реализовано | — |
| `iter_columns` | DataFrame.iter_columns | реализовано | — |
| `iter_rows` | DataFrame.iter_rows | реализовано | — |
| `iter_slices` | DataFrame.iter_slices | реализовано | — |
| `join` | DataFrame.join | реализовано | — |
| `join_asof` | DataFrame.join_asof | реализовано | — |
| `join_where` | DataFrame.join_where | реализовано | — |
| `lazy` | DataFrame.lazy | реализовано | — |
| `limit` | DataFrame.limit | реализовано | — |
| `map_columns` | DataFrame.map_columns | реализовано | — |
| `map_rows` | DataFrame.map_rows | реализовано | — |
| `match_to_schema` | DataFrame.match_to_schema | реализовано | — |
| `max` | DataFrame.max | реализовано | — |
| `max_horizontal` | DataFrame.max_horizontal | реализовано | — |
| `mean` | DataFrame.mean | реализовано | — |
| `mean_horizontal` | DataFrame.mean_horizontal | реализовано | — |
| `median` | DataFrame.median | реализовано | — |
| `melt` | DataFrame.melt | реализовано | — |
| `merge_sorted` | DataFrame.merge_sorted | реализовано | — |
| `min` | DataFrame.min | реализовано | — |
| `min_horizontal` | DataFrame.min_horizontal | реализовано | — |
| `n_chunks` | DataFrame.n_chunks | реализовано | — |
| `n_unique` | DataFrame.n_unique | реализовано | — |
| `null_count` | DataFrame.null_count | реализовано | — |
| `partition_by` | DataFrame.partition_by | реализовано | — |
| `pipe` | DataFrame.pipe | реализовано | — |
| `pivot` | DataFrame.pivot | реализовано | — |
| `plot` | DataFrame.plot | реализовано | — |
| `product` | DataFrame.product | реализовано | — |
| `quantile` | DataFrame.quantile | реализовано | — |
| `rechunk` | DataFrame.rechunk | реализовано | — |
| `remove` | DataFrame.remove | реализовано | — |
| `rename` | DataFrame.rename | реализовано | — |
| `replace_column` | DataFrame.replace_column | реализовано | — |
| `reverse` | DataFrame.reverse | реализовано | — |
| `rolling` | DataFrame.rolling | реализовано | — |
| `row` | DataFrame.row | реализовано | — |
| `rows` | DataFrame.rows | реализовано | — |
| `rows_by_key` | DataFrame.rows_by_key | реализовано | — |
| `sample` | DataFrame.sample | реализовано | — |
| `schema` | DataFrame.schema | реализовано | — |
| `select` | DataFrame.select | реализовано | — |
| `select_seq` | DataFrame.select_seq | реализовано | — |
| `serialize` | DataFrame.serialize | реализовано | — |
| `set_sorted` | DataFrame.set_sorted | реализовано | — |
| `shape` | DataFrame.shape | реализовано | — |
| `shift` | DataFrame.shift | реализовано | — |
| `show` | DataFrame.show | реализовано | — |
| `shrink_to_fit` | DataFrame.shrink_to_fit | реализовано | — |
| `slice` | DataFrame.slice | реализовано | — |
| `sort` | DataFrame.sort | реализовано | — |
| `sql` | DataFrame.sql | реализовано | — |
| `std` | DataFrame.std | реализовано | — |
| `style` | DataFrame.style | реализовано | — |
| `sum` | DataFrame.sum | реализовано | — |
| `sum_horizontal` | DataFrame.sum_horizontal | реализовано | — |
| `tail` | DataFrame.tail | реализовано | — |
| `to_arrow` | DataFrame.to_arrow | реализовано | — |
| `to_dict` | DataFrame.to_dict | реализовано | — |
| `to_dicts` | DataFrame.to_dicts | реализовано | — |
| `to_dummies` | DataFrame.to_dummies | реализовано | — |
| `to_init_repr` | DataFrame.to_init_repr | реализовано | — |
| `to_jax` | DataFrame.to_jax | реализовано | — |
| `to_numpy` | DataFrame.to_numpy | реализовано | — |
| `to_pandas` | DataFrame.to_pandas | реализовано | — |
| `to_series` | DataFrame.to_series | реализовано | — |
| `to_struct` | DataFrame.to_struct | реализовано | — |
| `to_torch` | DataFrame.to_torch | реализовано | — |
| `top_k` | DataFrame.top_k | реализовано | — |
| `transpose` | DataFrame.transpose | реализовано | — |
| `unique` | DataFrame.unique | реализовано | — |
| `unnest` | DataFrame.unnest | реализовано | — |
| `unpivot` | DataFrame.unpivot | реализовано | — |
| `unstack` | DataFrame.unstack | реализовано | — |
| `update` | DataFrame.update | реализовано | — |
| `upsample` | DataFrame.upsample | реализовано | — |
| `var` | DataFrame.var | реализовано | — |
| `vstack` | DataFrame.vstack | реализовано | — |
| `width` | DataFrame.width | реализовано | — |
| `with_columns` | DataFrame.with_columns | реализовано | — |
| `with_columns_seq` | DataFrame.with_columns_seq | реализовано | — |
| `with_row_count` | DataFrame.with_row_count | реализовано | — |
| `with_row_index` | DataFrame.with_row_index | реализовано | — |
| `write_avro` | DataFrame.write_avro | реализовано | — |
| `write_clipboard` | DataFrame.write_clipboard | реализовано | — |
| `write_csv` | DataFrame.write_csv | реализовано | — |
| `write_database` | DataFrame.write_database | реализовано | — |
| `write_delta` | DataFrame.write_delta | реализовано | — |
| `write_excel` | DataFrame.write_excel | реализовано | — |
| `write_iceberg` | DataFrame.write_iceberg | реализовано | — |
| `write_ipc` | DataFrame.write_ipc | реализовано | — |
| `write_ipc_stream` | DataFrame.write_ipc_stream | реализовано | — |
| `write_json` | DataFrame.write_json | реализовано | — |
| `write_ndjson` | DataFrame.write_ndjson | реализовано | — |
| `write_parquet` | DataFrame.write_parquet | реализовано | — |

Итог для DataFrame: реализовано 136 из 141.

## LazyFrame (89 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `__getitem__` | — | не реализовано | low |
| `approx_n_unique` | LazyFrame.approx_n_unique | реализовано | — |
| `bottom_k` | LazyFrame.bottom_k | реализовано | — |
| `cache` | LazyFrame.cache | реализовано | — |
| `cast` | — | не реализовано | low |
| `clear` | `Clear` | реализовано | low |
| `clone` | `Clone` | реализовано | low |
| `collect` | LazyFrame.collect | реализовано | — |
| `collect_async` | LazyFrame.collect_async | реализовано | — |
| `collect_batches` | LazyFrame.collect_batches | реализовано | — |
| `collect_schema` | — | не реализовано | low |
| `columns` | — | не реализовано | low |
| `count` | — | не реализовано | low |
| `describe` | — | не реализовано | low |
| `deserialize` | LazyFrame.deserialize | реализовано | — |
| `drop` | `Drop` | реализовано | low |
| `drop_nans` | `DropNaNs` | реализовано | low |
| `drop_nulls` | LazyFrame.drop_nulls | реализовано | — |
| `dtypes` | — | не реализовано | low |
| `explain` | LazyFrame.explain | реализовано | — |
| `explode` | LazyFrame.explode | реализовано | — |
| `fill_nan` | — | не реализовано | low |
| `fill_null` | LazyFrame.fill_null | реализовано | — |
| `filter` | LazyFrame.filter | реализовано | — |
| `first` | `First` | реализовано | low |
| `gather_every` | `GatherEvery` | реализовано | low |
| `group_by` | LazyFrame.group_by | реализовано | — |
| `group_by_dynamic` | LazyFrame.group_by_dynamic | реализовано | — |
| `head` | `Head` | реализовано | low |
| `inspect` | LazyFrame.inspect | реализовано | — |
| `interpolate` | — | не реализовано | low |
| `join` | LazyFrame.join | реализовано | — |
| `join_asof` | — | не реализовано | low |
| `join_where` | LazyFrame.join_where | реализовано | — |
| `last` | `Last` | реализовано | low |
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
| `remote` | LazyFrame.remote | реализовано | — |
| `remove` | — | не реализовано | low |
| `rename` | `Rename` | реализовано | low |
| `reverse` | `Reverse` | реализовано | low |
| `rolling` | — | не реализовано | low |
| `schema` | — | не реализовано | low |
| `select` | LazyFrame.select | реализовано | — |
| `select_seq` | — | не реализовано | low |
| `serialize` | LazyFrame.serialize | реализовано | — |
| `set_sorted` | — | не реализовано | low |
| `shift` | — | не реализовано | low |
| `show` | — | не реализовано | low |
| `show_graph` | LazyFrame.show_graph | реализовано | — |
| `sink_batches` | LazyFrame.sink_batches | реализовано | — |
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
| `tail` | `Tail` | реализовано | low |
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

Итог для LazyFrame: реализовано 33 из 89.

## Expr (217 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `abs` | Expr.abs | реализовано | — |
| `add` | Expr.add | реализовано | — |
| `agg_groups` | Expr.agg_groups | реализовано | — |
| `alias` | Expr.alias | реализовано | — |
| `all` | Expr.all | реализовано | — |
| `and_` | Expr.and_ | реализовано | — |
| `any` | Expr.any | реализовано | — |
| `append` | Expr.append | реализовано | — |
| `approx_n_unique` | Expr.approx_n_unique | реализовано | — |
| `arccos` | Expr.arccos | реализовано | — |
| `arccosh` | Expr.arccosh | реализовано | — |
| `arcsin` | Expr.arcsin | реализовано | — |
| `arcsinh` | Expr.arcsinh | реализовано | — |
| `arctan` | Expr.arctan | реализовано | — |
| `arctanh` | Expr.arctanh | реализовано | — |
| `arg_max` | Expr.arg_max | реализовано | — |
| `arg_min` | Expr.arg_min | реализовано | — |
| `arg_sort` | Expr.arg_sort | реализовано | — |
| `arg_true` | Expr.arg_true | реализовано | — |
| `arg_unique` | Expr.arg_unique | реализовано | — |
| `arr` | Expr.arr | реализовано | — |
| `backward_fill` | Expr.backward_fill | реализовано | — |
| `bin` | Expr.bin | реализовано | — |
| `bitwise_and` | Expr.bitwise_and | реализовано | — |
| `bitwise_count_ones` | Expr.bitwise_count_ones | реализовано | — |
| `bitwise_count_zeros` | Expr.bitwise_count_zeros | реализовано | — |
| `bitwise_leading_ones` | Expr.bitwise_leading_ones | реализовано | — |
| `bitwise_leading_zeros` | Expr.bitwise_leading_zeros | реализовано | — |
| `bitwise_or` | Expr.bitwise_or | реализовано | — |
| `bitwise_trailing_ones` | Expr.bitwise_trailing_ones | реализовано | — |
| `bitwise_trailing_zeros` | Expr.bitwise_trailing_zeros | реализовано | — |
| `bitwise_xor` | Expr.bitwise_xor | реализовано | — |
| `bottom_k` | Expr.bottom_k | реализовано | — |
| `bottom_k_by` | Expr.bottom_k_by | реализовано | — |
| `cast` | Expr.cast | реализовано | — |
| `cat` | Expr.cat | реализовано | — |
| `cbrt` | Expr.cbrt | реализовано | — |
| `ceil` | Expr.ceil | реализовано | — |
| `clip` | Expr.clip | реализовано | — |
| `cos` | Expr.cos | реализовано | — |
| `cosh` | Expr.cosh | реализовано | — |
| `cot` | Expr.cot | реализовано | — |
| `count` | Expr.count | реализовано | — |
| `cum_count` | Expr.cum_count | реализовано | — |
| `cum_max` | Expr.cum_max | реализовано | — |
| `cum_min` | Expr.cum_min | реализовано | — |
| `cum_prod` | Expr.cum_prod | реализовано | — |
| `cum_sum` | Expr.cum_sum | реализовано | — |
| `cumulative_eval` | Expr.cumulative_eval | реализовано | — |
| `cut` | Expr.cut | реализовано | — |
| `degrees` | Expr.degrees | реализовано | — |
| `deserialize` | Expr.deserialize | реализовано | — |
| `diff` | Expr.diff | реализовано | — |
| `dot` | Expr.dot | реализовано | — |
| `drop_nans` | Expr.drop_nans | реализовано | — |
| `drop_nulls` | Expr.drop_nulls | реализовано | — |
| `dt` | Expr.dt | реализовано | — |
| `entropy` | Expr.entropy | реализовано | — |
| `eq` | Expr.eq | реализовано | — |
| `eq_missing` | Expr.eq_missing | реализовано | — |
| `ewm_mean` | Expr.ewm_mean | реализовано | — |
| `ewm_mean_by` | Expr.ewm_mean_by | реализовано | — |
| `ewm_std` | Expr.ewm_std | реализовано | — |
| `ewm_var` | Expr.ewm_var | реализовано | — |
| `exclude` | Expr.exclude | реализовано | — |
| `exp` | Expr.exp | реализовано | — |
| `explode` | Expr.explode | реализовано | — |
| `ext` | Expr.ext | реализовано | — |
| `extend_constant` | Expr.extend_constant | реализовано | — |
| `fill_nan` | Expr.fill_na_n | реализовано | — |
| `fill_null` | Expr.fill_null | реализовано | — |
| `filter` | Expr.filter | реализовано | — |
| `first` | Expr.first | реализовано | — |
| `flatten` | Expr.flatten | реализовано | — |
| `floor` | Expr.floor | реализовано | — |
| `floordiv` | Expr.floordiv | реализовано | — |
| `forward_fill` | Expr.forward_fill | реализовано | — |
| `from_json` | Expr.from_json | реализовано | — |
| `gather` | Expr.gather | реализовано | — |
| `gather_every` | Expr.gather_every | реализовано | — |
| `ge` | Expr.ge | реализовано | — |
| `get` | Expr.get | реализовано | — |
| `gt` | Expr.gt | реализовано | — |
| `has_nulls` | Expr.has_nulls | реализовано | — |
| `hash` | Expr.hash | реализовано | — |
| `head` | Expr.head | реализовано | — |
| `hist` | Expr.hist | реализовано | — |
| `implode` | Expr.implode | реализовано | — |
| `index_of` | Expr.index_of | реализовано | — |
| `inspect` | Expr.inspect | реализовано | — |
| `interpolate` | Expr.interpolate | реализовано | — |
| `interpolate_by` | Expr.interpolate_by | реализовано | — |
| `is_between` | Expr.is_between | реализовано | — |
| `is_close` | Expr.is_close | реализовано | — |
| `is_duplicated` | Expr.is_duplicated | реализовано | — |
| `is_finite` | Expr.is_finite | реализовано | — |
| `is_first_distinct` | Expr.is_first_distinct | реализовано | — |
| `is_in` | Expr.is_in | реализовано | — |
| `is_infinite` | Expr.is_infinite | реализовано | — |
| `is_last_distinct` | Expr.is_last_distinct | реализовано | — |
| `is_nan` | Expr.is_nan | реализовано | — |
| `is_not_nan` | Expr.is_not_nan | реализовано | — |
| `is_not_null` | Expr.is_not_null | реализовано | — |
| `is_null` | Expr.is_null | реализовано | — |
| `is_unique` | Expr.is_unique | реализовано | — |
| `item` | Expr.item | реализовано | — |
| `kurtosis` | Expr.kurtosis | реализовано | — |
| `last` | Expr.last | реализовано | — |
| `le` | Expr.le | реализовано | — |
| `len` | Expr.list_len | реализовано | — |
| `limit` | Expr.limit | реализовано | — |
| `list` | Expr.list | реализовано | — |
| `log` | Expr.log | реализовано | — |
| `log10` | Expr.log10 | реализовано | — |
| `log1p` | Expr.log1p | реализовано | — |
| `lower_bound` | Expr.lower_bound | реализовано | — |
| `lt` | Expr.lt | реализовано | — |
| `map_batches` | Expr.map_batches | реализовано | — |
| `map_elements` | Expr.map_elements | реализовано | — |
| `max` | Expr.max | реализовано | — |
| `max_by` | Expr.max_by | реализовано | — |
| `mean` | Expr.mean | реализовано | — |
| `median` | Expr.median | реализовано | — |
| `meta` | Expr.meta | реализовано | — |
| `min` | Expr.min | реализовано | — |
| `min_by` | Expr.min_by | реализовано | — |
| `mod` | Expr.mod | реализовано | — |
| `mode` | Expr.mode | реализовано | — |
| `mul` | Expr.mul | реализовано | — |
| `n_unique` | Expr.n_unique | реализовано | — |
| `name` | Expr.name | реализовано | — |
| `nan_max` | Expr.nan_max | реализовано | — |
| `nan_min` | Expr.nan_min | реализовано | — |
| `ne` | Expr.ne | реализовано | — |
| `ne_missing` | Expr.ne_missing | реализовано | — |
| `neg` | Expr.neg | реализовано | — |
| `not_` | Expr.not_ | реализовано | — |
| `null_count` | Expr.null_count | реализовано | — |
| `or_` | Expr.or_ | реализовано | — |
| `over` | Expr.over | реализовано | — |
| `pct_change` | Expr.pct_change | реализовано | — |
| `peak_max` | Expr.peak_max | реализовано | — |
| `peak_min` | Expr.peak_min | реализовано | — |
| `pipe` | Expr.pipe | реализовано | — |
| `pow` | Expr.pow | реализовано | — |
| `product` | Expr.product | реализовано | — |
| `qcut` | — | не реализовано | low |
| `quantile` | Expr.quantile | реализовано | — |
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
| `round` | Expr.round | реализовано | — |
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
| `sqrt` | Expr.sqrt | реализовано | — |
| `std` | — | не реализовано | low |
| `str` | Expr.str | реализовано | — |
| `struct` | Expr.struct | реализовано | — |
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

Итог для Expr: реализовано 159 из 217.

## Series (226 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `__array__` | — | не реализовано | low |
| `__arrow_c_stream__` | — | не реализовано | low |
| `__getitem__` | — | не реализовано | low |
| `abs` | Series.abs | реализовано | — |
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
| `describe` | Series.describe | реализовано | — |
| `diff` | — | не реализовано | low |
| `dot` | — | не реализовано | low |
| `drop_nans` | Series.drop_nans | реализовано | — |
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
| `exp` | Series.exp | реализовано | — |
| `explode` | — | не реализовано | low |
| `ext` | — | не реализовано | low |
| `extend` | — | не реализовано | low |
| `extend_constant` | — | не реализовано | low |
| `fill_nan` | Series.fill_nan | реализовано | — |
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
| `hist` | Series.hist | реализовано | — |
| `implode` | — | не реализовано | low |
| `index_of` | — | не реализовано | low |
| `interpolate` | Series.interpolate | реализовано | — |
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
| `log` | Series.log | реализовано | — |
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
| `null_count` | Series.null_count | реализовано | — |
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
| `reverse` | Series.reverse | реализовано | — |
| `rle` | — | не реализовано | low |
| `rle_id` | — | не реализовано | low |
| `rolling_kurtosis` | — | не реализовано | low |
| `rolling_map` | — | не реализовано | low |
| `rolling_max` | Series.rolling_max | реализовано | — |
| `rolling_max_by` | — | не реализовано | low |
| `rolling_mean` | Series.rolling_mean | реализовано | — |
| `rolling_mean_by` | — | не реализовано | low |
| `rolling_median` | — | не реализовано | low |
| `rolling_median_by` | — | не реализовано | low |
| `rolling_min` | Series.rolling_min | реализовано | — |
| `rolling_min_by` | — | не реализовано | low |
| `rolling_quantile` | — | не реализовано | low |
| `rolling_quantile_by` | — | не реализовано | low |
| `rolling_rank` | — | не реализовано | low |
| `rolling_rank_by` | — | не реализовано | low |
| `rolling_skew` | — | не реализовано | low |
| `rolling_std` | — | не реализовано | low |
| `rolling_std_by` | — | не реализовано | low |
| `rolling_sum` | Series.rolling_sum | реализовано | — |
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
| `shift` | Series.shift | реализовано | — |
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
| `sqrt` | Series.sqrt | реализовано | — |
| `std` | Series.std | реализовано | — |
| `str` | — | не реализовано | low |
| `struct` | — | не реализовано | low |
| `sum` | Series.sum | реализовано | — |
| `tail` | — | не реализовано | low |
| `tan` | — | не реализовано | low |
| `tanh` | — | не реализовано | low |
| `to_arrow` | — | не реализовано | low |
| `to_dummies` | — | не реализовано | low |
| `to_frame` | — | не реализовано | low |
| `to_init_repr` | — | не реализовано | low |
| `to_jax` | — | не реализовано | low |
| `to_list` | Series.to_list | реализовано | — |
| `to_numpy` | Series.to_numpy | реализовано | — |
| `to_pandas` | Series.to_pandas | реализовано | — |
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

Итог для Series: реализовано 34 из 226.

## SQLContext (7 методов)
| Python Polars method | gopolars equivalent | Статус | Приоритет |
| --- | --- | --- | --- |
| `execute` | polars.SQLFromDataFrame / ParseSQL / SQLContext.execute | реализовано | — |
| `execute_global` | SQLContext.execute_global | реализовано | — |
| `register` | SQLContext.register | реализовано | — |
| `register_globals` | SQLContext.register_globals | реализовано | — |
| `register_many` | SQLContext.register_many | реализовано | — |
| `tables` | SQLContext.tables | реализовано | — |
| `unregister` | SQLContext.unregister | реализовано | — |

Итог для SQLContext: реализовано 7 из 7.

## Top-30 функций для v0.7 (по приоритету реализации)
| # | Объект | Python Polars method | Приоритет |
| --- | --- | --- | --- |
| 1 | DataFrame | `__array__` | low |
| 2 | DataFrame | `__arrow_c_stream__` | low |
| 3 | DataFrame | `__dataframe__` | low |
| 4 | DataFrame | `__getitem__` | low |
| 5 | DataFrame | `__setitem__` | low |
| 6 | Expr | `qcut` | low |
| 7 | Expr | `radians` | low |
| 8 | Expr | `rechunk` | low |
| 9 | Expr | `reinterpret` | low |
| 10 | Expr | `repeat_by` | low |
| 11 | Expr | `replace_strict` | low |
| 12 | Expr | `reshape` | low |
| 13 | Expr | `reverse` | low |
| 14 | Expr | `rle` | low |
| 15 | Expr | `rle_id` | low |
| 16 | Expr | `rolling` | low |
| 17 | Expr | `rolling_kurtosis` | low |
| 18 | Expr | `rolling_map` | low |
| 19 | Expr | `rolling_max_by` | low |
| 20 | Expr | `rolling_mean_by` | low |
| 21 | Expr | `rolling_median` | low |
| 22 | Expr | `rolling_median_by` | low |
| 23 | Expr | `rolling_min_by` | low |
| 24 | Expr | `rolling_quantile` | low |
| 25 | Expr | `rolling_quantile_by` | low |
| 26 | Expr | `rolling_rank` | low |
| 27 | Expr | `rolling_rank_by` | low |
| 28 | Expr | `rolling_skew` | low |
| 29 | Expr | `rolling_std_by` | low |
| 30 | Expr | `rolling_sum_by` | low |
