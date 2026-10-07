## v0.7.0 Migration Notes

Target release: **v0.7.0** — the next release after the `v0.6.2` tag. This release fixes
correctness defects found by an audit (OpenSpec change `fix-audit-correctness-bugs`). Most fixes
only make wrong results right; the sections below list the ones where code that relied on the old
behavior can observe a difference. No exported signature changed.

### **BREAKING**: `DataFrame.Update` and `LazyFrame.Update` follow Polars

`Update(other)` now overwrites values **by row position** with the **non-null** values of
`other`, for the columns both frames have. The left frame keeps its columns, order and height;
columns that exist only in `other`, and rows of `other` past the left height, are ignored. An
`Int64` column updated from a `Float64` column becomes `Float64`; any other dtype mismatch is an
error.

Before: eager `DataFrame.Update` appended `other`'s rows (it was `VStack`); `LazyFrame.Update`
re-ran `other`'s operations on the *left* frame's data, so it either failed with
"update node missing plan" or returned the left values.

**How to check your code.** Search for `.Update(`. Code that used eager `Update` to append rows
should call `VStack`. Code that relied on a null in `other` clearing a value now keeps the left
value (Polars `include_nulls=False`).

### **BREAKING**: `InsertColumn` rejects invalid positions and duplicate names

`InsertColumn(index, column)` returns an error when `index` is outside `[0, Width()]` (it used to
clamp silently) and when a column with the same name already exists (it used to replace that
column and move it), matching Polars. Inserting with a duplicate name at the last position used
to panic.

**How to check your code.** Callers that relied on clamping pass `0` or `Width()` explicitly;
callers that relied on replacement call `Drop(name)` first, or use `WithColumns`.

### **BREAKING**: Arrow, Parquet and ADBC columns of narrow types are typed

Importing Arrow data (`NewDataFrameFromArrow`, `ReadParquet`, database reads, DuckDB SQL results)
converts narrow signed/unsigned integers to `Int64`, `float16`/`float32` to `Float64`,
string/binary views to `String`/`Binary`, dictionary arrays to their decoded values, durations to
`Duration`, and the Arrow null type to an all-null `String` column. These columns used to come
back as `String` columns that **panicked on the first read** (DuckDB results rejected dictionary,
duration and view types). A `uint64` value above `math.MaxInt64` — including a DuckDB `UBIGINT`,
which used to wrap to a negative number — a duration outside `time.Duration`'s range, and an
Arrow type with no mapping (map, union, run-end encoded, decimal256, …) now fail the import with
an error naming the column.

**How to check your code.** A schema check that expected `String` for such columns now sees
`Int64`/`Float64`/`Duration`.

### **BREAKING**: JSON readers keep keys that appear after the first record

`ReadJSON`/`ScanJSON` (array and NDJSON) return one column per distinct key in **any** record, in
first-appearance order, with nulls where a record lacks the key. Keys that first appeared after
the first record used to be dropped silently, and the column order was random on every read.
IPC files and Arrow tables now also keep the frame's column order (files written by older
versions read in alphabetical order).

NDJSON input is now read as one stream: blank lines are accepted and the 64 KiB per-line limit is
gone; data after the last record is still an error.

**How to check your code.** Frames read from JSON whose records have differing keys may be wider
than before.

### **BREAKING**: `-0.0` and `0.0` are one key

Group-by, `unique`, `n_unique`, `is_unique`/`is_duplicated`, `value_counts`, joins, `.over()`
partitions, pivot and set operations treat `-0.0` and `0.0` as the same key (all NaNs already
were one key), at every input size. They used to form separate groups and not join with each
other.

### Fixed without an API change (results change only where they were wrong)

- Lazy optimizer: `limit` before `sort` is no longer reordered; chained `select`s keep aliases;
  filters are no longer moved past a renaming or aggregating `select`, or before a window they
  would change.
- Lazy scans no longer drop columns that no operation removed, and no longer hoist a filter past
  an earlier `with_columns`, `limit` or rename.
- `CollectStreaming` returns exactly what `Collect` returns (`tail`, `with_row_index`, `limit`
  before `filter`, cumulative and whole-column expressions used to be computed per chunk).
- `LazyFrame.FillNaN` and `LazyFrame.Quantile` use their argument exactly (it was rounded to six
  decimals).
- `cum_sum().over()` and `cum_count().over()` restart in each partition.
- Sorting by, and `==`/`!=` on, list, struct and binary columns no longer panic.
- `polars.Melt` and `polars.Pivot` return their columns in Polars' order (id columns, then
  `variable`/`value`; index column, then pivoted values in first-appearance order) instead of a
  random order.
- `Int64` add/sub/mul on `Series`, and `Int64` sums in `LazyFrame.Sum`, pivot and window `SUM`,
  are exact; overflow wraps (two's complement) on every platform, as in expressions.
- `Quantile` never panics: eager APIs return NaN for a NaN probability and keep clamping
  out-of-range values; `LazyFrame.Quantile` makes `Collect` return an error for a NaN or
  out-of-range probability.
- `Replace`, `ExtendConstant`, `Explode` and `Flatten` never return an empty Series; values that
  do not fit the input dtype promote it (`Float64` for int/float, otherwise `String`), and
  `ReplaceStrict` returns its error.
- SQL: registered table names are always quoted identifiers; a name can no longer inject SQL.
- Concurrent operations on frames that share columns no longer race.
