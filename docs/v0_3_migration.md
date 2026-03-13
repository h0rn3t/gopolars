## v0.3 Migration Notes

### API alignment changes

- Added public `polars.Series` type and constructor `polars.NewSeries`.
- `LazyFrame` gained `Slice`, `Unique`, `FillNull`, `DropNulls`.
- `DataFrame` gained `Series`, `Slice`, `Unique`, `Concat`, `FillNull`, `DropNulls`.
- Join `How` now supports `right` and `full` in addition to `inner` and `left`.

### Lazy scan behavior changes

- `ScanCSV`, `ScanJSON`, `ScanParquet`, `ScanIPC` are source-level lazy and do not fail on missing files until `Collect`/`CollectStreaming`.
- Projection and supported predicate pushdown are applied before remaining plan operators.

### IO backend changes

- Parquet backend now uses `github.com/parquet-go/parquet-go v0.29.0`.
- Arrow compatibility target is Apache Arrow `23.0.1`.

### Compatibility guidance

- If your code relied on eager validation during `Scan*`, move error handling to `Collect` stage.
- If your wrappers assumed join type enum is only `inner`/`left`, extend validation to include `right`/`full`.
- For strict API wrappers, add new DataFrame/LazyFrame methods to adapter interfaces before upgrading.
