## Developer Guide v0.3

### DataFrame API focus

- Core: `Select`, `Filter`, `WithColumns`, `GroupBy().Agg`, `Sort`, `Limit`
- MVP additions: `Slice`, `Unique`, `Concat`, `FillNull`, `DropNulls`
- Join modes: `inner`, `left`, `right`, `full`

### Series API focus

- Constructor: `polars.NewSeries`
- Null-aware ops: `IsNull`, `IsNotNull`, `FillNull`, `DropNulls`
- Type ops: `Cast`
- Vector ops: `Add`, `Sub`, `Mul`, `Div`, `Eq`, `Ne`, `Gt`, `Ge`, `Lt`, `Le`

### LazyFrame API focus

- Source-level scans: `ScanCSV`, `ScanJSON`, `ScanParquet`, `ScanIPC`
- Plan ops: `Select`, `Filter`, `WithColumns`, `Slice`, `Unique`, `FillNull`, `DropNulls`
- Execution: `Collect`, `CollectStreaming`, `Explain`

### IO and semantics

- Object-store URI mapping supported by env:
  - `GOPOLARS_S3_ROOT`
  - `GOPOLARS_GCS_ROOT`
  - `GOPOLARS_AZURE_ROOT`
- Null/NaN semantics:
  - `FillNull` replaces only null values
  - `DropNulls` removes rows by selected columns or all columns
  - Sorting keeps explicit null ordering via sort input

### Performance and validation workflow

- CI/release run formatting, vet, unit tests, race tests and conformance threshold.
- Benchmarks run with `-benchmem`, repeated counts and longer benchtime for regression visibility.
- Differential/conformance test suite remains the source of semantic parity confidence.
