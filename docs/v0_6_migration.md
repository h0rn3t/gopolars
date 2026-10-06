## v0.6.0 Migration Notes

Target release: **v0.6.0** — the next release after the `v0.5.0` tag. This file also keeps the
older *conformance wave* `v0.6` notes at the end; the wave number and the release tag are
unrelated (see [`versioning_policy.md`](versioning_policy.md)).

### **BREAKING**: `polars.Series` gains five methods

The exported `Series` interface now also declares the typed accessors:

```go
Int64Values() ([]int64, []bool, error)
Float64Values() ([]float64, []bool, error)
StringValues() ([]string, []bool, error)
BoolValues() ([]bool, []bool, error)
DatetimeValues() ([]time.Time, []bool, error)
```

**Who is affected.** Only code that declares its own type implementing `polars.Series` — in
practice a hand-written test double. Such a type stops compiling until it has the five methods.
No gopolars function ever accepted a foreign `Series` implementation (they are rejected with
"unsupported series implementation"), so no working data path changes.

**How to check your code.**

```bash
go build ./... && go vet ./...   # against gopolars v0.6.0
```

A test double can embed `polars.Series` to pick up the new methods, or implement them.

Every other change in this release is additive or internal; no existing signature changed.

### Highlights

Measured on Apple M4 Pro (arm64), Go 1.27.1, 200,000 rows (unique Int64/Float64/Datetime columns
plus a 3-value String column); see [`performance/parquet-io.md`](performance/parquet-io.md).

- **Typed column exchange with Go code (new API).** `Series.Int64Values` / `Float64Values` /
  `StringValues` (also Categorical/Enum) / `BoolValues` / `DatetimeValues` return a copy of the
  values plus a null mask (nil when there are no nulls; null slots hold the zero value; a dtype
  mismatch wraps `polars.ErrDTypeMismatch`). `NewInt64Series` … `NewDatetimeSeries` and
  `NewDataFrameFromSeries` build frames from typed slices (inputs are copied; duplicate names and
  length mismatches are errors). Mapping a frame to `[]struct` drops from 30.9 ms / 1.2M
  allocations via `IterRows` to 4.2 ms / 17 allocations; building a frame from `[]struct` from
  7.9 ms / 800k allocations via `NewDataFrame([]any)` to 2.1 ms / 22 allocations.
- **`ReadParquet` decodes only the requested `Columns`, in parallel.** Two of four columns:
  10.0 ms → 2.2 ms (−78%), 44.7 MiB → 19.2 MiB; all columns: 9.4 ms → 5.9 ms (−37%); allocations
  −99%. The legacy JSON-envelope format is detected from the schema before any data is decoded.
- **`WriteParquet` uses dictionary encoding only where it pays off.** Int64/Float64/Datetime
  columns whose sampled values are more than half distinct are written without a dictionary;
  String columns keep it. 54.9 ms → 26.2 ms (−52%), 173.8 MiB → 50.9 MiB (−71%), file
  4.27 MB → 2.99 MB.
- **Arrow import** copies a string column's value buffer once instead of allocating per value
  (200k → 10 allocations, −69% time), and no longer materializes an all-false null mask.
- **`IterRows` / `ToDicts`** resolve columns once instead of per cell (−22%).

### Behavior notes (unchanged, recorded)

- `ReadParquet` with `Columns` returns the columns in **file order** and silently ignores names
  that are not in the file. Python Polars 1.41.2 returns the requested order and raises
  `ColumnNotFoundError`. gopolars keeps its existing behavior in this release.
- A numeric column with roughly 3.5k–130k distinct values is now written without a dictionary;
  the file can be somewhat larger than before for such columns. Values are unaffected.
- Strings read from parquet share one buffer per column: holding a single string keeps that
  column's buffer alive.

### Migration guidance

- Rebuild; only custom `polars.Series` implementations need the five new methods.
- To get the largest gain, replace `IterRows`-based row mapping with `GetColumn(name)` plus the
  typed accessors, and `NewDataFrame([]any)` with the typed constructors and
  `NewDataFrameFromSeries`.

### Conformance wave v0.6 (historical notes)

#### Highlights

- Расширены namespace операции для string/datetime/list (trim, starts_with, list_get, dt_hour, dt_weekday).
- Добавлены temporal window API: `GroupByDynamic` и расширенный `RollingMean` в eager/lazy путях.
- Explain diagnostics и execution report обновлены до schema v2 с temporal/performance markers.
- Добавлены performance budget и regression evidence артефакты для release gates.

#### Migration guidance

- Проверьте pipeline, использующие ручные временные бакеты, и перенесите их на `GroupByDynamic`.
- Для rolling аналитики зафиксируйте `Closed`, `Window`, `MinRows` в тестах для детерминированности.
- Обновите парсинг diagnostics/report output на schema v2 (`temporal_window_operations`, `performance_markers`).
- Подключите performance budget и regression report scripts в nightly/release workflows.
