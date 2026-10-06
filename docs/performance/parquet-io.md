# Parquet IO: projection, parallel decode, adaptive dictionary

Change: `accelerate-prod-io-paths`. Reference machine: Apple M4 Pro (arm64), Go 1.27.
Dataset: 200,000 rows — unique `Int64` / `Float64` / `Datetime` columns plus a
3-value `String` column (the column mix a production consumer writes and reads).

```bash
go test -run '^$' -bench 'BenchmarkReadParquetProjection|BenchmarkWriteParquetProdShape' \
  -benchtime=10x -count=6 ./pkg/io/parquet/
```

## Read (`ReadParquet` / `parquet.Read`)

The reader used to decode the whole file with `pqarrow.ReadTable`, convert every
column to a frame and only then drop the columns that were not requested. It now:

- detects the legacy JSON-envelope format from the schema, before decoding data;
- decodes only the leaf columns of the requested top-level fields
  (`FileReader.ReadRowGroups` with leaf indices; nested fields keep all their leaves);
- decodes columns in parallel (`ArrowReadProperties{Parallel: true}`, one goroutine
  per column, joined before the call returns);
- imports string columns from Arrow with one buffer copy per column instead of a
  `strings.Clone` per value, and numeric columns with one bulk copy and no
  all-false null mask when the column has no nulls.

| benchmark | before | after | delta |
|---|---|---|---|
| all 4 columns, time | 9.39 ms | 5.87 ms | −37% |
| 2 of 4 columns, time | 10.0 ms | 2.24 ms | −78% |
| all 4 columns, memory | 44.7 MiB | 43.8 MiB | −2% |
| 2 of 4 columns, memory | 44.7 MiB | 19.2 MiB | −57% |
| allocations | 202.8k | 1.6k–2.9k | −99% |

Observable behavior is unchanged: result columns follow file order, unknown
requested names are ignored, and an empty file yields an empty frame. Both differ
from py-polars 1.41.2 (requested order; `ColumnNotFoundError`) and are kept on
purpose; see the change's `design.md`, Open Questions.

## Write (`WriteParquet` / `parquet.Write`)

arrow-go dictionary-encodes every column by default. For a high-cardinality
numeric column it builds a hash table of values only to fall back to plain
encoding once the dictionary outgrows its 1 MiB page — that was ~96% of all write
allocations. The writer now samples up to 4096 non-null values per `Int64`,
`Float64` and `Datetime` column (strided, deterministic) and disables the
dictionary when more than half of the sample is distinct. String columns and all
other dtypes keep the arrow-go default.

| benchmark | before | after | delta |
|---|---|---|---|
| time | 54.9 ms | 26.2 ms | −52% |
| memory | 173.8 MiB | 50.9 MiB | −71% |
| allocations | 1,002k | 402k | −60% |
| file size | 4.27 MB | 2.99 MB | −30% |

Trade-off: a numeric column with roughly 3.5k–130k distinct values gets no
dictionary even where one would still have shrunk the file somewhat; that costs
file size only, never correctness. The remaining ~2 allocations per string value
happen inside arrow-go's dictionary encoder for byte arrays.

The frame→Arrow conversion owned by gopolars and the encoder time owned by
pqarrow are still split by `BenchmarkWriteParquetBreakdown`.
