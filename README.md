# gopolars

`gopolars` is a high-performance Go DataFrame library inspired by Polars Python API.

## Current status

The project has completed parity waves through **v0.6** and now covers a broad core for Go-native analytics pipelines, including advanced joins, reshape operations, temporal windows, and performance diagnostics.\
It is production-usable for many DataFrame workloads, but it is **not yet a full drop-in replacement** for Python Polars.

- ✅ Strong DataFrame/LazyFrame core for real analytics workloads
- ✅ Stable IO surface (CSV/JSON/Parquet/IPC + scans + pushdown)
- 🚧 Remaining long-tail parity mainly in low-priority API surface

## Implemented capabilities

### DataFrame, Series and Expressions

- Eager and lazy execution over a columnar in-memory DataFrame engine
- Core DataFrame operations: `select`, `filter`, `with_columns`, `sort`, `limit`, `group_by`, `join`
- Extended DataFrame surface: `slice`, `head`, `tail`, `unique`, `concat`, `fill_null`, `drop_nulls`, `drop`, `rename`
- Join modes: `inner`, `left`, `right`, `full`, `semi`, `anti`, `cross`, `asof`
- Temporal analytics: `group_by_dynamic`, `rolling_mean`
- Reshape support: `melt`, `pivot`
- Public `Series` API with null-aware operations, vector arithmetic and comparisons
- Expression namespaces for string/datetime/list/struct workflows:
  - `list_len`
  - `list_contains`
  - `list_get`
  - `struct_field`
  - `str_lower`, `str_upper`, `str_len`, `str_replace`, `str_trim`, `starts_with`, `contains`
  - `dt_year`, `dt_month`, `dt_day`, `dt_hour`, `dt_weekday`
  - `explode`
  - `flatten` (struct flattening)

### SQL and Query Planning

- SQL parsing and planning for `SELECT` pipelines
- CTE support (`WITH ... AS (...)`)
- Subqueries in `FROM`
- Set operations: `UNION`, `INTERSECT`, `EXCEPT`
- Window expression support with `PARTITION BY` and `ORDER BY`
- Logical optimization passes including pushdown and adaptive planning rules

### IO and Interoperability

- CSV, JSON/NDJSON, Parquet, IPC read/write support
- Source-level lazy scan for CSV/JSON/Parquet/IPC
- Projection and predicate pushdown on scan pipelines
- Partition-aware Parquet dataset scan (multi-file layout)
- Partition pruning by predicate for dataset scans
- Arrow import/export bridge
- Object store URI mapping profile (`s3://`, `gcs://`, `az://`) via environment-configured roots

### Streaming, Diagnostics and Quality

- Streaming collect with bounded-memory path and deterministic fallback
- Explain and diagnostics output with stable schema for automation (`schema_version: v2`)
- Operator-level execution report structure for telemetry integrations (duration, memory, temporal operator markers)
- Unit/conformance tests, benchmarks, `go vet`, race tests, CI quality gates
- Compatibility governance artifacts:
  - versioning policy
  - migration notes
  - breaking-change evidence gate script
  - performance budget and regression evidence scripts

## Capability matrix

| Capability                                                        | Status         |
| ----------------------------------------------------------------- | -------------- |
| DataFrame eager API                                               | ✅ ready        |
| Lazy execution, scans, pushdown                                   | ✅ ready        |
| Series public API                                                 | ✅ ready        |
| Nested transforms (explode/flatten + list/struct expr)            | ✅ ready        |
| SQL base + CTE + window expressions                               | ✅ ready        |
| GroupBy, temporal windows and joins                               | ✅ ready        |
| Streaming collect                                                 | ✅ ready        |
| CSV/JSON/Parquet/IPC IO                                           | ✅ ready        |
| Arrow interoperability                                            | ✅ ready        |
| Cloud-style partitioned dataset scans                             | ✅ ready        |
| Explain/telemetry schema v2 and perf markers                      | ✅ ready        |
| Full Python Polars API parity                                     | 🚧 in progress |
| Full SQL parity with Python Polars SQLContext                     | 🚧 in progress |
| Performance parity on all workloads                               | 🚧 in progress |
| Ecosystem parity (all namespaces, plugins, advanced UDF patterns) | 🚧 in progress |

## Python Polars vs gopolars function matrix

Ниже — **полная сводная таблица** по всем методам из Python Polars (stable).

- ✅ — реализовано
- ❌ — не реализовано

| Python функция                  | Go функция (gopolars)        | Статус |
| ------------------------------- | ---------------------------- | ------ |
| `DataFrame.__array__`           | —                            | ❌      |
| `DataFrame.__arrow_c_stream__`  | —                            | ❌      |
| `DataFrame.__dataframe__`       | —                            | ❌      |
| `DataFrame.__getitem__`         | —                            | ❌      |
| `DataFrame.__setitem__`         | —                            | ❌      |
| `DataFrame.approx_n_unique`     | `DataFrame.ApproxNUnique`    | ✅      |
| `DataFrame.bottom_k`            | `DataFrame.BottomK`          | ✅      |
| `DataFrame.cast`                | `DataFrame.Cast`             | ✅      |
| `DataFrame.clear`               | `DataFrame.Clear`            | ✅      |
| `DataFrame.clone`               | `DataFrame.Clone`            | ✅      |
| `DataFrame.collect_schema`      | `DataFrame.CollectSchema`    | ✅      |
| `DataFrame.columns`             | `DataFrame.Columns`          | ✅      |
| `DataFrame.corr`                | `DataFrame.Corr`             | ✅      |
| `DataFrame.count`               | `DataFrame.Count`            | ✅      |
| `DataFrame.describe`            | `DataFrame.Describe`         | ✅      |
| `DataFrame.deserialize`         | `DataFrame.Deserialize`      | ✅      |
| `DataFrame.drop`                | `DataFrame.Drop`             | ✅      |
| `DataFrame.drop_in_place`       | `DataFrame.DropInPlace`      | ✅      |
| `DataFrame.drop_nans`           | `DataFrame.DropNaNs`         | ✅      |
| `DataFrame.drop_nulls`          | `DataFrame.DropNulls`        | ✅      |
| `DataFrame.dtypes`              | `DataFrame.Dtypes`           | ✅      |
| `DataFrame.equals`              | `DataFrame.Equals`           | ✅      |
| `DataFrame.estimated_size`      | `DataFrame.EstimatedSize`    | ✅      |
| `DataFrame.explode`             | `DataFrame.Explode`          | ✅      |
| `DataFrame.extend`              | `DataFrame.Extend`           | ✅      |
| `DataFrame.fill_nan`            | `DataFrame.FillNaN`          | ✅      |
| `DataFrame.fill_null`           | `DataFrame.FillNull`         | ✅      |
| `DataFrame.filter`              | `DataFrame.Filter`           | ✅      |
| `DataFrame.flags`               | `DataFrame.Flags`            | ✅      |
| `DataFrame.fold`                | `DataFrame.Fold`             | ✅      |
| `DataFrame.gather_every`        | `DataFrame.GatherEvery`      | ✅      |
| `DataFrame.get_column`          | `DataFrame.GetColumn`        | ✅      |
| `DataFrame.get_column_index`    | `DataFrame.GetColumnIndex`   | ✅      |
| `DataFrame.get_columns`         | `DataFrame.GetColumns`       | ✅      |
| `DataFrame.glimpse`             | `DataFrame.Glimpse`          | ✅      |
| `DataFrame.group_by`            | `DataFrame.GroupBy`          | ✅      |
| `DataFrame.group_by_dynamic`    | `DataFrame.GroupByDynamic`   | ✅      |
| `DataFrame.hash_rows`           | `DataFrame.HashRows`         | ✅      |
| `DataFrame.head`                | `DataFrame.Head`             | ✅      |
| `DataFrame.height`              | `DataFrame.Height`           | ✅      |
| `DataFrame.hstack`              | `DataFrame.Hstack`           | ✅      |
| `DataFrame.insert_column`       | `DataFrame.InsertColumn`     | ✅      |
| `DataFrame.interpolate`         | `DataFrame.Interpolate`      | ✅      |
| `DataFrame.is_duplicated`       | `DataFrame.IsDuplicated`     | ✅      |
| `DataFrame.is_empty`            | `DataFrame.IsEmpty`          | ✅      |
| `DataFrame.is_unique`           | `DataFrame.IsUnique`         | ✅      |
| `DataFrame.item`                | `DataFrame.Item`             | ✅      |
| `DataFrame.iter_columns`        | `DataFrame.IterColumns`      | ✅      |
| `DataFrame.iter_rows`           | `DataFrame.IterRows`         | ✅      |
| `DataFrame.iter_slices`         | `DataFrame.IterSlices`       | ✅      |
| `DataFrame.join`                | `DataFrame.Join`             | ✅      |
| `DataFrame.join_asof`           | `DataFrame.JoinAsof`         | ✅      |
| `DataFrame.join_where`          | `DataFrame.JoinWhere`        | ✅      |
| `DataFrame.lazy`                | `DataFrame.Lazy`             | ✅      |
| `DataFrame.limit`               | `DataFrame.Limit`            | ✅      |
| `DataFrame.map_columns`         | `DataFrame.MapColumns`       | ✅      |
| `DataFrame.map_rows`            | `DataFrame.MapRows`          | ✅      |
| `DataFrame.match_to_schema`     | `DataFrame.MatchToSchema`    | ✅      |
| `DataFrame.max`                 | `DataFrame.Max`              | ✅      |
| `DataFrame.max_horizontal`      | `DataFrame.MaxHorizontal`    | ✅      |
| `DataFrame.mean`                | `DataFrame.Mean`             | ✅      |
| `DataFrame.mean_horizontal`     | `DataFrame.MeanHorizontal`   | ✅      |
| `DataFrame.median`              | `DataFrame.Median`           | ✅      |
| `DataFrame.melt`                | `DataFrame.Melt`             | ✅      |
| `DataFrame.merge_sorted`        | `DataFrame.MergeSorted`      | ✅      |
| `DataFrame.min`                 | `DataFrame.Min`              | ✅      |
| `DataFrame.min_horizontal`      | `DataFrame.MinHorizontal`    | ✅      |
| `DataFrame.n_chunks`            | `DataFrame.NChunks`          | ✅      |
| `DataFrame.n_unique`            | `DataFrame.NUnique`          | ✅      |
| `DataFrame.null_count`          | `DataFrame.NullCount`        | ✅      |
| `DataFrame.partition_by`        | `DataFrame.PartitionBy`      | ✅      |
| `DataFrame.pipe`                | `DataFrame.Pipe`             | ✅      |
| `DataFrame.pivot`               | `DataFrame.Pivot`            | ✅      |
| `DataFrame.plot`                | `DataFrame.Plot`             | ✅      |
| `DataFrame.product`             | `DataFrame.Product`          | ✅      |
| `DataFrame.quantile`            | `DataFrame.Quantile`         | ✅      |
| `DataFrame.rechunk`             | `DataFrame.Rechunk`          | ✅      |
| `DataFrame.remove`              | `DataFrame.Remove`           | ✅      |
| `DataFrame.rename`              | `DataFrame.Rename`           | ✅      |
| `DataFrame.replace_column`      | `DataFrame.ReplaceColumn`    | ✅      |
| `DataFrame.reverse`             | `DataFrame.Reverse`          | ✅      |
| `DataFrame.rolling`             | `DataFrame.Rolling`          | ✅      |
| `DataFrame.row`                 | `DataFrame.Row`              | ✅      |
| `DataFrame.rows`                | `DataFrame.Rows`             | ✅      |
| `DataFrame.rows_by_key`         | `DataFrame.RowsByKey`        | ✅      |
| `DataFrame.sample`              | `DataFrame.Sample`           | ✅      |
| `DataFrame.schema`              | `DataFrame.Schema`           | ✅      |
| `DataFrame.select`              | `DataFrame.Select`           | ✅      |
| `DataFrame.select_seq`          | `DataFrame.SelectSeq`        | ✅      |
| `DataFrame.serialize`           | `DataFrame.Serialize`        | ✅      |
| `DataFrame.set_sorted`          | `DataFrame.SetSorted`        | ✅      |
| `DataFrame.shape`               | `DataFrame.Shape`            | ✅      |
| `DataFrame.shift`               | `DataFrame.Shift`            | ✅      |
| `DataFrame.show`                | `DataFrame.Show`             | ✅      |
| `DataFrame.shrink_to_fit`       | `DataFrame.ShrinkToFit`      | ✅      |
| `DataFrame.slice`               | `DataFrame.Slice`            | ✅      |
| `DataFrame.sort`                | `DataFrame.Sort`             | ✅      |
| `DataFrame.sql`                 | `DataFrame.Sql`              | ✅      |
| `DataFrame.std`                 | `DataFrame.Std`              | ✅      |
| `DataFrame.style`               | `DataFrame.Style`            | ✅      |
| `DataFrame.sum`                 | `DataFrame.Sum`              | ✅      |
| `DataFrame.sum_horizontal`      | `DataFrame.SumHorizontal`    | ✅      |
| `DataFrame.tail`                | `DataFrame.Tail`             | ✅      |
| `DataFrame.to_arrow`            | `DataFrame.ToArrow`          | ✅      |
| `DataFrame.to_dict`             | `DataFrame.ToDict`           | ✅      |
| `DataFrame.to_dicts`            | `DataFrame.ToDicts`          | ✅      |
| `DataFrame.to_dummies`          | `DataFrame.ToDummies`        | ✅      |
| `DataFrame.to_init_repr`        | `DataFrame.ToInitRepr`       | ✅      |
| `DataFrame.to_jax`              | `DataFrame.ToJax`            | ✅      |
| `DataFrame.to_numpy`            | `DataFrame.ToNumpy`          | ✅      |
| `DataFrame.to_pandas`           | `DataFrame.ToPandas`         | ✅      |
| `DataFrame.to_series`           | `DataFrame.ToSeries`         | ✅      |
| `DataFrame.to_struct`           | `DataFrame.ToStruct`         | ✅      |
| `DataFrame.to_torch`            | `DataFrame.ToTorch`          | ✅      |
| `DataFrame.top_k`               | `DataFrame.TopK`             | ✅      |
| `DataFrame.transpose`           | `DataFrame.Transpose`        | ✅      |
| `DataFrame.unique`              | `DataFrame.Unique`           | ✅      |
| `DataFrame.unnest`              | `DataFrame.Unnest`           | ✅      |
| `DataFrame.unpivot`             | `DataFrame.Unpivot`          | ✅      |
| `DataFrame.unstack`             | `DataFrame.Unstack`          | ✅      |
| `DataFrame.update`              | `DataFrame.Update`           | ✅      |
| `DataFrame.upsample`            | `DataFrame.Upsample`         | ✅      |
| `DataFrame.var`                 | `DataFrame.Var`              | ✅      |
| `DataFrame.vstack`              | `DataFrame.Vstack`           | ✅      |
| `DataFrame.width`               | `DataFrame.Width`            | ✅      |
| `DataFrame.with_columns`        | `DataFrame.WithColumns`      | ✅      |
| `DataFrame.with_columns_seq`    | `DataFrame.WithColumnsSeq`   | ✅      |
| `DataFrame.with_row_count`      | `DataFrame.WithRowCount`     | ✅      |
| `DataFrame.with_row_index`      | `DataFrame.WithRowIndex`     | ✅      |
| `DataFrame.write_avro`          | `DataFrame.WriteAvro`        | ✅      |
| `DataFrame.write_clipboard`     | `DataFrame.WriteClipboard`   | ✅      |
| `DataFrame.write_csv`           | `DataFrame.WriteCsv`         | ✅      |
| `DataFrame.write_database`      | `DataFrame.WriteDatabase`    | ✅      |
| `DataFrame.write_delta`         | `DataFrame.WriteDelta`       | ✅      |
| `DataFrame.write_excel`         | `DataFrame.WriteExcel`       | ✅      |
| `DataFrame.write_iceberg`       | `DataFrame.WriteIceberg`     | ✅      |
| `DataFrame.write_ipc`           | `DataFrame.WriteIpc`         | ✅      |
| `DataFrame.write_ipc_stream`    | `DataFrame.WriteIpcStream`   | ✅      |
| `DataFrame.write_json`          | `DataFrame.WriteJson`        | ✅      |
| `DataFrame.write_ndjson`        | `DataFrame.WriteNdjson`      | ✅      |
| `DataFrame.write_parquet`       | `DataFrame.WriteParquet`     | ✅      |
| `LazyFrame.__getitem__`         | —                            | ❌      |
| `LazyFrame.approx_n_unique`     | `LazyFrame.ApproxNUnique`    | ✅      |
| `LazyFrame.bottom_k`            | `LazyFrame.BottomK`          | ✅      |
| `LazyFrame.cache`               | `LazyFrame.Cache`            | ✅      |
| `LazyFrame.cast`                | —                            | ❌      |
| `LazyFrame.clear`               | —                            | ❌      |
| `LazyFrame.clone`               | —                            | ❌      |
| `LazyFrame.collect`             | `LazyFrame.Collect`          | ✅      |
| `LazyFrame.collect_async`       | `LazyFrame.CollectAsync`     | ✅      |
| `LazyFrame.collect_batches`     | `LazyFrame.CollectBatches`   | ✅      |
| `LazyFrame.collect_schema`      | —                            | ❌      |
| `LazyFrame.columns`             | —                            | ❌      |
| `LazyFrame.count`               | —                            | ❌      |
| `LazyFrame.describe`            | —                            | ❌      |
| `LazyFrame.deserialize`         | `LazyFrame.Deserialize`      | ✅      |
| `LazyFrame.drop`                | —                            | ❌      |
| `LazyFrame.drop_nans`           | —                            | ❌      |
| `LazyFrame.drop_nulls`          | `LazyFrame.DropNulls`        | ✅      |
| `LazyFrame.dtypes`              | —                            | ❌      |
| `LazyFrame.explain`             | `LazyFrame.Explain`          | ✅      |
| `LazyFrame.explode`             | `LazyFrame.Explode`          | ✅      |
| `LazyFrame.fill_nan`            | —                            | ❌      |
| `LazyFrame.fill_null`           | `LazyFrame.FillNull`         | ✅      |
| `LazyFrame.filter`              | `LazyFrame.Filter`           | ✅      |
| `LazyFrame.first`               | —                            | ❌      |
| `LazyFrame.gather_every`        | —                            | ❌      |
| `LazyFrame.group_by`            | `LazyFrame.GroupBy`          | ✅      |
| `LazyFrame.group_by_dynamic`    | `LazyFrame.GroupByDynamic`   | ✅      |
| `LazyFrame.head`                | —                            | ❌      |
| `LazyFrame.inspect`             | `LazyFrame.Inspect`          | ✅      |
| `LazyFrame.interpolate`         | —                            | ❌      |
| `LazyFrame.join`                | `LazyFrame.Join`             | ✅      |
| `LazyFrame.join_asof`           | —                            | ❌      |
| `LazyFrame.join_where`          | `LazyFrame.JoinWhere`        | ✅      |
| `LazyFrame.last`                | —                            | ❌      |
| `LazyFrame.lazy`                | —                            | ❌      |
| `LazyFrame.limit`               | `LazyFrame.Limit`            | ✅      |
| `LazyFrame.map_batches`         | —                            | ❌      |
| `LazyFrame.match_to_schema`     | —                            | ❌      |
| `LazyFrame.max`                 | —                            | ❌      |
| `LazyFrame.mean`                | —                            | ❌      |
| `LazyFrame.median`              | —                            | ❌      |
| `LazyFrame.melt`                | `LazyFrame.Melt`             | ✅      |
| `LazyFrame.merge_sorted`        | —                            | ❌      |
| `LazyFrame.min`                 | —                            | ❌      |
| `LazyFrame.null_count`          | —                            | ❌      |
| `LazyFrame.pipe`                | —                            | ❌      |
| `LazyFrame.pipe_with_schema`    | —                            | ❌      |
| `LazyFrame.pivot`               | `LazyFrame.Pivot`            | ✅      |
| `LazyFrame.profile`             | `LazyFrame.Profile`          | ✅      |
| `LazyFrame.quantile`            | —                            | ❌      |
| `LazyFrame.remote`              | `LazyFrame.Remote`           | ✅      |
| `LazyFrame.remove`              | —                            | ❌      |
| `LazyFrame.rename`              | —                            | ❌      |
| `LazyFrame.reverse`             | —                            | ❌      |
| `LazyFrame.rolling`             | —                            | ❌      |
| `LazyFrame.schema`              | —                            | ❌      |
| `LazyFrame.select`              | `LazyFrame.Select`           | ✅      |
| `LazyFrame.select_seq`          | —                            | ❌      |
| `LazyFrame.serialize`           | `LazyFrame.Serialize`        | ✅      |
| `LazyFrame.set_sorted`          | —                            | ❌      |
| `LazyFrame.shift`               | —                            | ❌      |
| `LazyFrame.show`                | —                            | ❌      |
| `LazyFrame.show_graph`          | `LazyFrame.ShowGraph`        | ✅      |
| `LazyFrame.sink_batches`        | `LazyFrame.SinkBatches`      | ✅      |
| `LazyFrame.sink_csv`            | —                            | ❌      |
| `LazyFrame.sink_delta`          | —                            | ❌      |
| `LazyFrame.sink_iceberg`        | —                            | ❌      |
| `LazyFrame.sink_ipc`            | —                            | ❌      |
| `LazyFrame.sink_ndjson`         | `LazyFrame.SinkNDJSON`       | ✅      |
| `LazyFrame.sink_parquet`        | `LazyFrame.SinkParquet`      | ✅      |
| `LazyFrame.slice`               | `LazyFrame.Slice`            | ✅      |
| `LazyFrame.sort`                | `LazyFrame.Sort`             | ✅      |
| `LazyFrame.sql`                 | `LazyFrame.SQL`              | ✅      |
| `LazyFrame.std`                 | —                            | ❌      |
| `LazyFrame.sum`                 | —                            | ❌      |
| `LazyFrame.tail`                | —                            | ❌      |
| `LazyFrame.top_k`               | —                            | ❌      |
| `LazyFrame.unique`              | `LazyFrame.Unique`           | ✅      |
| `LazyFrame.unnest`              | —                            | ❌      |
| `LazyFrame.unpivot`             | —                            | ❌      |
| `LazyFrame.update`              | —                            | ❌      |
| `LazyFrame.var`                 | —                            | ❌      |
| `LazyFrame.width`               | —                            | ❌      |
| `LazyFrame.with_columns`        | `LazyFrame.WithColumns`      | ✅      |
| `LazyFrame.with_columns_seq`    | —                            | ❌      |
| `LazyFrame.with_context`        | —                            | ❌      |
| `LazyFrame.with_row_count`      | —                            | ❌      |
| `LazyFrame.with_row_index`      | —                            | ❌      |
| `Expr.abs`                      | `Expr.Abs`                   | ✅      |
| `Expr.add`                      | `Expr.Add`                   | ✅      |
| `Expr.agg_groups`               | `Expr.AggGroups`             | ✅      |
| `Expr.alias`                    | `Expr.Alias`                 | ✅      |
| `Expr.all`                      | `Expr.All`                   | ✅      |
| `Expr.and_`                     | `Expr.And_`                  | ✅      |
| `Expr.any`                      | `Expr.Any`                   | ✅      |
| `Expr.append`                   | `Expr.Append`                | ✅      |
| `Expr.approx_n_unique`          | `Expr.ApproxNUnique`         | ✅      |
| `Expr.arccos`                   | `Expr.Arccos`                | ✅      |
| `Expr.arccosh`                  | `Expr.Arccosh`               | ✅      |
| `Expr.arcsin`                   | `Expr.Arcsin`                | ✅      |
| `Expr.arcsinh`                  | `Expr.Arcsinh`               | ✅      |
| `Expr.arctan`                   | `Expr.Arctan`                | ✅      |
| `Expr.arctanh`                  | `Expr.Arctanh`               | ✅      |
| `Expr.arg_max`                  | `Expr.ArgMax`                | ✅      |
| `Expr.arg_min`                  | `Expr.ArgMin`                | ✅      |
| `Expr.arg_sort`                 | `Expr.ArgSort`               | ✅      |
| `Expr.arg_true`                 | `Expr.ArgTrue`               | ✅      |
| `Expr.arg_unique`               | `Expr.ArgUnique`             | ✅      |
| `Expr.arr`                      | `Expr.Arr`                   | ✅      |
| `Expr.backward_fill`            | `Expr.BackwardFill`          | ✅      |
| `Expr.bin`                      | `Expr.Bin`                   | ✅      |
| `Expr.bitwise_and`              | `Expr.BitwiseAnd`            | ✅      |
| `Expr.bitwise_count_ones`       | `Expr.BitwiseCountOnes`      | ✅      |
| `Expr.bitwise_count_zeros`      | `Expr.BitwiseCountZeros`     | ✅      |
| `Expr.bitwise_leading_ones`     | `Expr.BitwiseLeadingOnes`    | ✅      |
| `Expr.bitwise_leading_zeros`    | `Expr.BitwiseLeadingZeros`   | ✅      |
| `Expr.bitwise_or`               | `Expr.BitwiseOr`             | ✅      |
| `Expr.bitwise_trailing_ones`    | `Expr.BitwiseTrailingOnes`   | ✅      |
| `Expr.bitwise_trailing_zeros`   | `Expr.BitwiseTrailingZeros`  | ✅      |
| `Expr.bitwise_xor`              | `Expr.BitwiseXor`            | ✅      |
| `Expr.bottom_k`                 | `Expr.BottomK`               | ✅      |
| `Expr.bottom_k_by`              | `Expr.BottomKBy`             | ✅      |
| `Expr.cast`                     | `Expr.Cast`                  | ✅      |
| `Expr.cat`                      | `Expr.Cat`                   | ✅      |
| `Expr.cbrt`                     | `Expr.Cbrt`                  | ✅      |
| `Expr.ceil`                     | `Expr.Ceil`                  | ✅      |
| `Expr.clip`                     | `Expr.Clip`                  | ✅      |
| `Expr.cos`                      | `Expr.Cos`                   | ✅      |
| `Expr.cosh`                     | `Expr.Cosh`                  | ✅      |
| `Expr.cot`                      | `Expr.Cot`                   | ✅      |
| `Expr.count`                    | `Expr.Count`                 | ✅      |
| `Expr.cum_count`                | `Expr.CumCount`              | ✅      |
| `Expr.cum_max`                  | `Expr.CumMax`                | ✅      |
| `Expr.cum_min`                  | `Expr.CumMin`                | ✅      |
| `Expr.cum_prod`                 | `Expr.CumProd`               | ✅      |
| `Expr.cum_sum`                  | `Expr.CumSum`                | ✅      |
| `Expr.cumulative_eval`          | `Expr.CumulativeEval`        | ✅      |
| `Expr.cut`                      | `Expr.Cut`                   | ✅      |
| `Expr.degrees`                  | `Expr.Degrees`               | ✅      |
| `Expr.deserialize`              | `Expr.Deserialize`           | ✅      |
| `Expr.diff`                     | `Expr.Diff`                  | ✅      |
| `Expr.dot`                      | `Expr.Dot`                   | ✅      |
| `Expr.drop_nans`                | `Expr.DropNans`              | ✅      |
| `Expr.drop_nulls`               | `Expr.DropNulls`             | ✅      |
| `Expr.dt`                       | `Expr.Dt`                    | ✅      |
| `Expr.entropy`                  | `Expr.Entropy`               | ✅      |
| `Expr.eq`                       | `Expr.Eq`                    | ✅      |
| `Expr.eq_missing`               | `Expr.EqMissing`             | ✅      |
| `Expr.ewm_mean`                 | `Expr.EwmMean`               | ✅      |
| `Expr.ewm_mean_by`              | `Expr.EwmMeanBy`             | ✅      |
| `Expr.ewm_std`                  | `Expr.EwmStd`                | ✅      |
| `Expr.ewm_var`                  | `Expr.EwmVar`                | ✅      |
| `Expr.exclude`                  | `Expr.Exclude`               | ✅      |
| `Expr.exp`                      | `Expr.Exp`                   | ✅      |
| `Expr.explode`                  | `Expr.Explode`               | ✅      |
| `Expr.ext`                      | `Expr.Ext`                   | ✅      |
| `Expr.extend_constant`          | `Expr.ExtendConstant`        | ✅      |
| `Expr.fill_nan`                 | `Expr.FillNaN`               | ✅      |
| `Expr.fill_null`                | `Expr.FillNull`              | ✅      |
| `Expr.filter`                   | `Expr.Filter`                | ✅      |
| `Expr.first`                    | `Expr.First`                 | ✅      |
| `Expr.flatten`                  | `Expr.Flatten`               | ✅      |
| `Expr.floor`                    | `Expr.Floor`                 | ✅      |
| `Expr.floordiv`                 | `Expr.Floordiv`              | ✅      |
| `Expr.forward_fill`             | `Expr.ForwardFill`           | ✅      |
| `Expr.from_json`                | `Expr.FromJson`              | ✅      |
| `Expr.gather`                   | `Expr.Gather`                | ✅      |
| `Expr.gather_every`             | `Expr.GatherEvery`           | ✅      |
| `Expr.ge`                       | `Expr.Ge`                    | ✅      |
| `Expr.get`                      | `Expr.Get`                   | ✅      |
| `Expr.gt`                       | `Expr.Gt`                    | ✅      |
| `Expr.has_nulls`                | `Expr.HasNulls`              | ✅      |
| `Expr.hash`                     | `Expr.Hash`                  | ✅      |
| `Expr.head`                     | `Expr.Head`                  | ✅      |
| `Expr.hist`                     | `Expr.Hist`                  | ✅      |
| `Expr.implode`                  | `Expr.Implode`               | ✅      |
| `Expr.index_of`                 | `Expr.IndexOf`               | ✅      |
| `Expr.inspect`                  | `Expr.Inspect`               | ✅      |
| `Expr.interpolate`              | `Expr.Interpolate`           | ✅      |
| `Expr.interpolate_by`           | `Expr.InterpolateBy`         | ✅      |
| `Expr.is_between`               | `Expr.IsBetween`             | ✅      |
| `Expr.is_close`                 | `Expr.IsClose`               | ✅      |
| `Expr.is_duplicated`            | `Expr.IsDuplicated`          | ✅      |
| `Expr.is_finite`                | `Expr.IsFinite`              | ✅      |
| `Expr.is_first_distinct`        | `Expr.IsFirstDistinct`       | ✅      |
| `Expr.is_in`                    | `Expr.IsIn`                  | ✅      |
| `Expr.is_infinite`              | `Expr.IsInfinite`            | ✅      |
| `Expr.is_last_distinct`         | `Expr.IsLastDistinct`        | ✅      |
| `Expr.is_nan`                   | `Expr.IsNan`                 | ✅      |
| `Expr.is_not_nan`               | `Expr.IsNotNan`              | ✅      |
| `Expr.is_not_null`              | `Expr.IsNotNull`             | ✅      |
| `Expr.is_null`                  | `Expr.IsNull`                | ✅      |
| `Expr.is_unique`                | `Expr.IsUnique`              | ✅      |
| `Expr.item`                     | `Expr.Item`                  | ✅      |
| `Expr.kurtosis`                 | `Expr.Kurtosis`              | ✅      |
| `Expr.last`                     | `Expr.Last`                  | ✅      |
| `Expr.le`                       | `Expr.Le`                    | ✅      |
| `Expr.len`                      | `Expr.ListLen`               | ✅      |
| `Expr.limit`                    | `Expr.Limit`                 | ✅      |
| `Expr.list`                     | `Expr.List`                  | ✅      |
| `Expr.log`                      | `Expr.Log`                   | ✅      |
| `Expr.log10`                    | `Expr.Log10`                 | ✅      |
| `Expr.log1p`                    | `Expr.Log1p`                 | ✅      |
| `Expr.lower_bound`              | `Expr.LowerBound`            | ✅      |
| `Expr.lt`                       | `Expr.Lt`                    | ✅      |
| `Expr.map_batches`              | `Expr.MapBatches`            | ✅      |
| `Expr.map_elements`             | `Expr.MapElements`           | ✅      |
| `Expr.max`                      | `Expr.Max`                   | ✅      |
| `Expr.max_by`                   | `Expr.MaxBy`                 | ✅      |
| `Expr.mean`                     | `Expr.Mean`                  | ✅      |
| `Expr.median`                   | `Expr.Median`                | ✅      |
| `Expr.meta`                     | `Expr.Meta`                  | ✅      |
| `Expr.min`                      | `Expr.Min`                   | ✅      |
| `Expr.min_by`                   | `Expr.MinBy`                 | ✅      |
| `Expr.mod`                      | `Expr.Mod`                   | ✅      |
| `Expr.mode`                     | `Expr.Mode`                  | ✅      |
| `Expr.mul`                      | `Expr.Mul`                   | ✅      |
| `Expr.n_unique`                 | `Expr.NUnique`               | ✅      |
| `Expr.name`                     | `Expr.Name`                  | ✅      |
| `Expr.nan_max`                  | `Expr.NanMax`                | ✅      |
| `Expr.nan_min`                  | `Expr.NanMin`                | ✅      |
| `Expr.ne`                       | `Expr.Ne`                    | ✅      |
| `Expr.ne_missing`               | `Expr.NeMissing`             | ✅      |
| `Expr.neg`                      | `Expr.Neg`                   | ✅      |
| `Expr.not_`                     | `Expr.Not_`                  | ✅      |
| `Expr.null_count`               | `Expr.NullCount`             | ✅      |
| `Expr.or_`                      | `Expr.Or_`                   | ✅      |
| `Expr.over`                     | `Expr.Over`                  | ✅      |
| `Expr.pct_change`               | `Expr.PctChange`             | ✅      |
| `Expr.peak_max`                 | `Expr.PeakMax`               | ✅      |
| `Expr.peak_min`                 | `Expr.PeakMin`               | ✅      |
| `Expr.pipe`                     | `Expr.Pipe`                  | ✅      |
| `Expr.pow`                      | `Expr.Pow`                   | ✅      |
| `Expr.product`                  | `Expr.Product`               | ✅      |
| `Expr.qcut`                     | —                            | ❌      |
| `Expr.quantile`                 | `Expr.Quantile`              | ✅      |
| `Expr.radians`                  | —                            | ❌      |
| `Expr.rank`                     | `Expr.Rank`                  | ✅      |
| `Expr.rechunk`                  | —                            | ❌      |
| `Expr.reinterpret`              | —                            | ❌      |
| `Expr.repeat_by`                | —                            | ❌      |
| `Expr.replace`                  | `Expr.Replace`               | ✅      |
| `Expr.replace_strict`           | —                            | ❌      |
| `Expr.reshape`                  | —                            | ❌      |
| `Expr.reverse`                  | —                            | ❌      |
| `Expr.rle`                      | —                            | ❌      |
| `Expr.rle_id`                   | —                            | ❌      |
| `Expr.rolling`                  | —                            | ❌      |
| `Expr.rolling_kurtosis`         | —                            | ❌      |
| `Expr.rolling_map`              | —                            | ❌      |
| `Expr.rolling_max`              | `Expr.RollingMax`            | ✅      |
| `Expr.rolling_max_by`           | —                            | ❌      |
| `Expr.rolling_mean`             | `Expr.RollingMean`           | ✅      |
| `Expr.rolling_mean_by`          | —                            | ❌      |
| `Expr.rolling_median`           | —                            | ❌      |
| `Expr.rolling_median_by`        | —                            | ❌      |
| `Expr.rolling_min`              | `Expr.RollingMin`            | ✅      |
| `Expr.rolling_min_by`           | —                            | ❌      |
| `Expr.rolling_quantile`         | —                            | ❌      |
| `Expr.rolling_quantile_by`      | —                            | ❌      |
| `Expr.rolling_rank`             | —                            | ❌      |
| `Expr.rolling_rank_by`          | —                            | ❌      |
| `Expr.rolling_skew`             | —                            | ❌      |
| `Expr.rolling_std`              | `Expr.RollingStd`            | ✅      |
| `Expr.rolling_std_by`           | —                            | ❌      |
| `Expr.rolling_sum`              | `Expr.RollingSum`            | ✅      |
| `Expr.rolling_sum_by`           | —                            | ❌      |
| `Expr.rolling_var`              | —                            | ❌      |
| `Expr.rolling_var_by`           | —                            | ❌      |
| `Expr.round`                    | `Expr.Round`                 | ✅      |
| `Expr.round_sig_figs`           | —                            | ❌      |
| `Expr.sample`                   | —                            | ❌      |
| `Expr.search_sorted`            | —                            | ❌      |
| `Expr.set_sorted`               | —                            | ❌      |
| `Expr.shift`                    | —                            | ❌      |
| `Expr.shrink_dtype`             | —                            | ❌      |
| `Expr.shuffle`                  | —                            | ❌      |
| `Expr.sign`                     | —                            | ❌      |
| `Expr.sin`                      | —                            | ❌      |
| `Expr.sinh`                     | —                            | ❌      |
| `Expr.skew`                     | —                            | ❌      |
| `Expr.slice`                    | —                            | ❌      |
| `Expr.sort`                     | —                            | ❌      |
| `Expr.sort_by`                  | —                            | ❌      |
| `Expr.sqrt`                     | `Expr.Sqrt`                  | ✅      |
| `Expr.std`                      | —                            | ❌      |
| `Expr.str`                      | `Expr.Str`                   | ✅      |
| `Expr.struct`                   | `Expr.Struct`                | ✅      |
| `Expr.sub`                      | `Expr.Sub`                   | ✅      |
| `Expr.sum`                      | —                            | ❌      |
| `Expr.tail`                     | —                            | ❌      |
| `Expr.tan`                      | —                            | ❌      |
| `Expr.tanh`                     | —                            | ❌      |
| `Expr.to_physical`              | —                            | ❌      |
| `Expr.top_k`                    | —                            | ❌      |
| `Expr.top_k_by`                 | —                            | ❌      |
| `Expr.truediv`                  | —                            | ❌      |
| `Expr.truncate`                 | —                            | ❌      |
| `Expr.unique`                   | —                            | ❌      |
| `Expr.unique_counts`            | —                            | ❌      |
| `Expr.upper_bound`              | —                            | ❌      |
| `Expr.value_counts`             | —                            | ❌      |
| `Expr.var`                      | —                            | ❌      |
| `Expr.where`                    | —                            | ❌      |
| `Expr.xor`                      | —                            | ❌      |
| `Series.__array__`              | —                            | ❌      |
| `Series.__arrow_c_stream__`     | —                            | ❌      |
| `Series.__getitem__`            | —                            | ❌      |
| `Series.abs`                    | `Series.Abs`                 | ✅      |
| `Series.alias`                  | —                            | ❌      |
| `Series.all`                    | —                            | ❌      |
| `Series.any`                    | —                            | ❌      |
| `Series.append`                 | —                            | ❌      |
| `Series.approx_n_unique`        | —                            | ❌      |
| `Series.arccos`                 | —                            | ❌      |
| `Series.arccosh`                | —                            | ❌      |
| `Series.arcsin`                 | —                            | ❌      |
| `Series.arcsinh`                | —                            | ❌      |
| `Series.arctan`                 | —                            | ❌      |
| `Series.arctanh`                | —                            | ❌      |
| `Series.arg_max`                | —                            | ❌      |
| `Series.arg_min`                | —                            | ❌      |
| `Series.arg_sort`               | —                            | ❌      |
| `Series.arg_true`               | —                            | ❌      |
| `Series.arg_unique`             | —                            | ❌      |
| `Series.arr`                    | —                            | ❌      |
| `Series.backward_fill`          | —                            | ❌      |
| `Series.bin`                    | —                            | ❌      |
| `Series.bitwise_and`            | —                            | ❌      |
| `Series.bitwise_count_ones`     | —                            | ❌      |
| `Series.bitwise_count_zeros`    | —                            | ❌      |
| `Series.bitwise_leading_ones`   | —                            | ❌      |
| `Series.bitwise_leading_zeros`  | —                            | ❌      |
| `Series.bitwise_or`             | —                            | ❌      |
| `Series.bitwise_trailing_ones`  | —                            | ❌      |
| `Series.bitwise_trailing_zeros` | —                            | ❌      |
| `Series.bitwise_xor`            | —                            | ❌      |
| `Series.bottom_k`               | —                            | ❌      |
| `Series.bottom_k_by`            | —                            | ❌      |
| `Series.cast`                   | `Series.Cast`                | ✅      |
| `Series.cat`                    | —                            | ❌      |
| `Series.cbrt`                   | —                            | ❌      |
| `Series.ceil`                   | —                            | ❌      |
| `Series.chunk_lengths`          | —                            | ❌      |
| `Series.clear`                  | —                            | ❌      |
| `Series.clip`                   | —                            | ❌      |
| `Series.clone`                  | —                            | ❌      |
| `Series.cos`                    | —                            | ❌      |
| `Series.cosh`                   | —                            | ❌      |
| `Series.cot`                    | —                            | ❌      |
| `Series.count`                  | —                            | ❌      |
| `Series.cum_count`              | —                            | ❌      |
| `Series.cum_max`                | —                            | ❌      |
| `Series.cum_min`                | —                            | ❌      |
| `Series.cum_prod`               | —                            | ❌      |
| `Series.cum_sum`                | —                            | ❌      |
| `Series.cumulative_eval`        | —                            | ❌      |
| `Series.cut`                    | —                            | ❌      |
| `Series.describe`               | `Series.Describe`            | ✅      |
| `Series.diff`                   | —                            | ❌      |
| `Series.dot`                    | —                            | ❌      |
| `Series.drop_nans`              | `Series.DropNans`            | ✅      |
| `Series.drop_nulls`             | `Series.DropNulls`           | ✅      |
| `Series.dt`                     | —                            | ❌      |
| `Series.dtype`                  | —                            | ❌      |
| `Series.entropy`                | —                            | ❌      |
| `Series.eq`                     | `Series.Eq`                  | ✅      |
| `Series.eq_missing`             | —                            | ❌      |
| `Series.equals`                 | —                            | ❌      |
| `Series.estimated_size`         | —                            | ❌      |
| `Series.ewm_mean`               | —                            | ❌      |
| `Series.ewm_mean_by`            | —                            | ❌      |
| `Series.ewm_std`                | —                            | ❌      |
| `Series.ewm_var`                | —                            | ❌      |
| `Series.exp`                    | `Series.Exp`                 | ✅      |
| `Series.explode`                | —                            | ❌      |
| `Series.ext`                    | —                            | ❌      |
| `Series.extend`                 | —                            | ❌      |
| `Series.extend_constant`        | —                            | ❌      |
| `Series.fill_nan`               | `Series.FillNan`             | ✅      |
| `Series.fill_null`              | `Series.FillNull`            | ✅      |
| `Series.filter`                 | —                            | ❌      |
| `Series.first`                  | —                            | ❌      |
| `Series.flags`                  | —                            | ❌      |
| `Series.floor`                  | —                            | ❌      |
| `Series.forward_fill`           | —                            | ❌      |
| `Series.gather`                 | —                            | ❌      |
| `Series.gather_every`           | —                            | ❌      |
| `Series.ge`                     | `Series.Ge`                  | ✅      |
| `Series.get_chunks`             | —                            | ❌      |
| `Series.gt`                     | `Series.Gt`                  | ✅      |
| `Series.has_nulls`              | —                            | ❌      |
| `Series.has_validity`           | —                            | ❌      |
| `Series.hash`                   | —                            | ❌      |
| `Series.head`                   | —                            | ❌      |
| `Series.hist`                   | `Series.Hist`                | ✅      |
| `Series.implode`                | —                            | ❌      |
| `Series.index_of`               | —                            | ❌      |
| `Series.interpolate`            | `Series.Interpolate`         | ✅      |
| `Series.interpolate_by`         | —                            | ❌      |
| `Series.is_between`             | —                            | ❌      |
| `Series.is_close`               | —                            | ❌      |
| `Series.is_duplicated`          | —                            | ❌      |
| `Series.is_empty`               | —                            | ❌      |
| `Series.is_finite`              | —                            | ❌      |
| `Series.is_first_distinct`      | —                            | ❌      |
| `Series.is_in`                  | —                            | ❌      |
| `Series.is_infinite`            | —                            | ❌      |
| `Series.is_last_distinct`       | —                            | ❌      |
| `Series.is_nan`                 | —                            | ❌      |
| `Series.is_not_nan`             | —                            | ❌      |
| `Series.is_not_null`            | `Series.IsNotNull`           | ✅      |
| `Series.is_null`                | `Series.IsNull`              | ✅      |
| `Series.is_sorted`              | —                            | ❌      |
| `Series.is_unique`              | —                            | ❌      |
| `Series.item`                   | —                            | ❌      |
| `Series.kurtosis`               | —                            | ❌      |
| `Series.last`                   | —                            | ❌      |
| `Series.le`                     | `Series.Le`                  | ✅      |
| `Series.len`                    | `Series.Len`                 | ✅      |
| `Series.limit`                  | —                            | ❌      |
| `Series.list`                   | —                            | ❌      |
| `Series.log`                    | `Series.Log`                 | ✅      |
| `Series.log10`                  | —                            | ❌      |
| `Series.log1p`                  | —                            | ❌      |
| `Series.lower_bound`            | —                            | ❌      |
| `Series.lt`                     | `Series.Lt`                  | ✅      |
| `Series.map_elements`           | —                            | ❌      |
| `Series.max`                    | —                            | ❌      |
| `Series.max_by`                 | —                            | ❌      |
| `Series.mean`                   | —                            | ❌      |
| `Series.median`                 | —                            | ❌      |
| `Series.min`                    | —                            | ❌      |
| `Series.min_by`                 | —                            | ❌      |
| `Series.mode`                   | —                            | ❌      |
| `Series.n_chunks`               | —                            | ❌      |
| `Series.n_unique`               | —                            | ❌      |
| `Series.name`                   | `Series.Name`                | ✅      |
| `Series.nan_max`                | —                            | ❌      |
| `Series.nan_min`                | —                            | ❌      |
| `Series.ne`                     | `Series.Ne`                  | ✅      |
| `Series.ne_missing`             | —                            | ❌      |
| `Series.new_from_index`         | —                            | ❌      |
| `Series.not_`                   | —                            | ❌      |
| `Series.null_count`             | `Series.NullCount`           | ✅      |
| `Series.pct_change`             | —                            | ❌      |
| `Series.peak_max`               | —                            | ❌      |
| `Series.peak_min`               | —                            | ❌      |
| `Series.plot`                   | —                            | ❌      |
| `Series.pow`                    | —                            | ❌      |
| `Series.product`                | —                            | ❌      |
| `Series.qcut`                   | —                            | ❌      |
| `Series.quantile`               | —                            | ❌      |
| `Series.rank`                   | —                            | ❌      |
| `Series.rechunk`                | —                            | ❌      |
| `Series.reinterpret`            | —                            | ❌      |
| `Series.rename`                 | —                            | ❌      |
| `Series.repeat_by`              | —                            | ❌      |
| `Series.replace`                | —                            | ❌      |
| `Series.replace_strict`         | —                            | ❌      |
| `Series.reshape`                | —                            | ❌      |
| `Series.reverse`                | `Series.Reverse`             | ✅      |
| `Series.rle`                    | —                            | ❌      |
| `Series.rle_id`                 | —                            | ❌      |
| `Series.rolling_kurtosis`       | —                            | ❌      |
| `Series.rolling_map`            | —                            | ❌      |
| `Series.rolling_max`            | `Series.RollingMax`          | ✅      |
| `Series.rolling_max_by`         | —                            | ❌      |
| `Series.rolling_mean`           | `Series.RollingMean`         | ✅      |
| `Series.rolling_mean_by`        | —                            | ❌      |
| `Series.rolling_median`         | —                            | ❌      |
| `Series.rolling_median_by`      | —                            | ❌      |
| `Series.rolling_min`            | `Series.RollingMin`          | ✅      |
| `Series.rolling_min_by`         | —                            | ❌      |
| `Series.rolling_quantile`       | —                            | ❌      |
| `Series.rolling_quantile_by`    | —                            | ❌      |
| `Series.rolling_rank`           | —                            | ❌      |
| `Series.rolling_rank_by`        | —                            | ❌      |
| `Series.rolling_skew`           | —                            | ❌      |
| `Series.rolling_std`            | —                            | ❌      |
| `Series.rolling_std_by`         | —                            | ❌      |
| `Series.rolling_sum`            | `Series.RollingSum`          | ✅      |
| `Series.rolling_sum_by`         | —                            | ❌      |
| `Series.rolling_var`            | —                            | ❌      |
| `Series.rolling_var_by`         | —                            | ❌      |
| `Series.round`                  | —                            | ❌      |
| `Series.round_sig_figs`         | —                            | ❌      |
| `Series.sample`                 | —                            | ❌      |
| `Series.scatter`                | —                            | ❌      |
| `Series.search_sorted`          | —                            | ❌      |
| `Series.set`                    | —                            | ❌      |
| `Series.set_sorted`             | —                            | ❌      |
| `Series.shape`                  | —                            | ❌      |
| `Series.shift`                  | `Series.Shift`               | ✅      |
| `Series.shrink_dtype`           | —                            | ❌      |
| `Series.shrink_to_fit`          | —                            | ❌      |
| `Series.shuffle`                | —                            | ❌      |
| `Series.sign`                   | —                            | ❌      |
| `Series.sin`                    | —                            | ❌      |
| `Series.sinh`                   | —                            | ❌      |
| `Series.skew`                   | —                            | ❌      |
| `Series.slice`                  | —                            | ❌      |
| `Series.sort`                   | —                            | ❌      |
| `Series.sql`                    | —                            | ❌      |
| `Series.sqrt`                   | `Series.Sqrt`                | ✅      |
| `Series.std`                    | `Series.Std`                 | ✅      |
| `Series.str`                    | —                            | ❌      |
| `Series.struct`                 | —                            | ❌      |
| `Series.sum`                    | `Series.Sum`                 | ✅      |
| `Series.tail`                   | —                            | ❌      |
| `Series.tan`                    | —                            | ❌      |
| `Series.tanh`                   | —                            | ❌      |
| `Series.to_arrow`               | —                            | ❌      |
| `Series.to_dummies`             | —                            | ❌      |
| `Series.to_frame`               | —                            | ❌      |
| `Series.to_init_repr`           | —                            | ❌      |
| `Series.to_jax`                 | —                            | ❌      |
| `Series.to_list`                | `Series.ToList`              | ✅      |
| `Series.to_numpy`               | `Series.ToNumpy`             | ✅      |
| `Series.to_pandas`              | `Series.ToPandas`            | ✅      |
| `Series.to_physical`            | —                            | ❌      |
| `Series.to_torch`               | —                            | ❌      |
| `Series.top_k`                  | —                            | ❌      |
| `Series.top_k_by`               | —                            | ❌      |
| `Series.truncate`               | —                            | ❌      |
| `Series.unique`                 | —                            | ❌      |
| `Series.unique_counts`          | —                            | ❌      |
| `Series.upper_bound`            | —                            | ❌      |
| `Series.value_counts`           | —                            | ❌      |
| `Series.var`                    | —                            | ❌      |
| `Series.zip_with`               | —                            | ❌      |
| `SQLContext.execute`            | `SQLContext.Execute`         | ✅      |
| `SQLContext.execute_global`     | `SQLContext.ExecuteGlobal`   | ✅      |
| `SQLContext.register`           | `SQLContext.Register`        | ✅      |
| `SQLContext.register_globals`   | `SQLContext.RegisterGlobals` | ✅      |
| `SQLContext.register_many`      | `SQLContext.RegisterMany`    | ✅      |
| `SQLContext.tables`             | `SQLContext.Tables`          | ✅      |
| `SQLContext.unregister`         | `SQLContext.Unregister`      | ✅      |

### Сводка покрытия

| Объект     | Реализовано | Всего в Python Polars |
| ---------- | ----------- | --------------------- |
| DataFrame  | 136         | 141                   |
| LazyFrame  | 33          | 89                    |
| Expr       | 159         | 217                   |
| Series     | 34          | 226                   |
| SQLContext | 7           | 7                     |
| Итого      | 369         | 680                   |

| Приоритет среди нереализованных | Количество |
| ------------------------------- | ---------- |
| high                            | 0          |
| medium                          | 0          |
| low                             | 311        |

- Исходная машинная матрица: [docs/parity/python\_polars\_full\_matrix.md](docs/parity/python_polars_full_matrix.md)
- Shortlist ближайшей волны: [docs/parity/v0\_7\_top30\_functions.md](docs/parity/v0_7_top30_functions.md)

## What is still needed to replace Python Polars

To position `gopolars` as a practical replacement for Python Polars in most teams, the following areas still need expansion:

1. **Broader API parity**
   - Full namespace coverage across string/datetime/list/struct/window semantics
   - Deeper expression parity for edge-case behavior and error contracts
2. **Deeper SQL parity**
   - Wider SQL surface and compatibility with advanced analytical query patterns
   - Stronger parity guarantees across planner and execution semantics
3. **Performance and scale hardening**
   - More optimization rules and workload-adaptive planning
   - Larger benchmark corpus and stricter regression budgets
4. **Cloud and lakehouse robustness**
   - Expanded object-store behavior and dataset semantics at scale
   - More integration coverage for partitioned and heterogeneous datasets
5. **Compatibility and migration experience**
   - Continued stabilization of deprecation/migration workflows
   - Clear release evidence for every potentially breaking alignment change

## Roadmap focus

- **Near term:** expand SQL/catalog surface, close remaining namespace long-tail, and harden performance budgets on larger datasets.
- **Mid term:** improve planner/runtime adaptability for mixed temporal + join + reshape workloads.
- **Final parity push:** close long-tail semantic differences and publish repeatable parity evidence for release readiness.

## Quick start

```go
io := polars.NewIO()
df, _ := io.ReadCSV(polars.ReadCSVInput{
    Path: "data.csv",
    HasHeader: true,
    Separator: ',',
})

out, _ := df.
    Lazy().
    Filter(polars.Col("value").Gt(polars.Lit(int64(10)))).
    GroupBy("city").
    Agg(polars.Sum(polars.Col("value"))).
    Collect(context.Background())

_ = out
```

## Examples

- [Parquet scan](examples/parquet_scan/main.go)
- [Lazy pushdown](examples/lazy_pushdown/main.go)
- [Join variants](examples/join_variants/main.go)
- [Streaming collect](examples/streaming_collect/main.go)
  ```go
  ```

collect
