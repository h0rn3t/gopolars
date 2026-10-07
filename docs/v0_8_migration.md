## v0.8.0 Migration Notes

Target release: **v0.8.0** — the next release after the `v0.7.0` tag. This release fixes Datetime
values outside 1677-09-21 … 2262-04-11 (OpenSpec change `fix-datetime-timestamp-range`). Every
release up to v0.7.0 wrote them as a different instant without an error, for example
`9999-12-31` as `1816-03-29`, and treated such dates as equal keys. No exported signature changed.

### **BREAKING**: Datetime is exported as `timestamp[us]`

`WriteParquet`, `SinkParquet`, `WriteDatabase` and DuckDB SQL (frames registered with
`SQLContext` or queried with `SQL`) now export Datetime columns, including `time.Time` values
inside List and Struct columns, as Arrow `timestamp[us]`: Parquet `TIMESTAMP(MICROS)` and DuckDB
`TIMESTAMP`. They used to be `timestamp[ns]` (Parquet `TIMESTAMP(NANOS)`, DuckDB
`TIMESTAMP_NS`), built with `time.Time.UnixNano`, which wraps outside 1677–2262. This is the
unit Polars uses by default (`Datetime("us")`), and the unit of PostgreSQL `timestamp`.

- Every instant from year −290308 to 294247 is written exactly. `0001-01-01` and `9999-12-31`
  survive a write and a read.
- Sub-microsecond digits are truncated toward the earlier instant:
  `12:30:45.123456789` is written as `12:30:45.123456`, as Polars does when it casts `ns` to `us`.
- An instant outside the microsecond range makes the write fail with an error that names the
  column (`column "ts": datetime … is outside the microsecond timestamp range`) instead of
  writing a different instant. For Parquet the error comes before the file is created.

Reading is unchanged: files and query results in any timestamp unit import as before.

**How to check your code.**

- Code or downstream readers that expect a nanosecond column (a schema check for
  `timestamp[ns]`, DuckDB SQL that names `TIMESTAMP_NS`, a consumer that compares nanoseconds)
  now see microseconds.
- Values that carry nanoseconds, such as `time.Now()`, no longer round-trip through Parquet,
  ADBC or DuckDB exactly; compare them with microsecond precision
  (`a.Truncate(time.Microsecond).Equal(b)`), or keep them in an `Int64` column of `UnixNano`.
- A `timestamp[ns]` file read and written back by gopolars becomes `timestamp[us]`: the Datetime
  dtype carries no unit, unlike Polars.
- **Files written by v0.7.0 or earlier** that held dates outside 1677–2262 contain the wrapped
  instant; the original value is lost. Re-export them from their source after upgrading.

### Fixed without an API change (results change only where they were wrong)

- `group_by`, `unique`, `n_unique` and joins on Datetime keys treat two different instants as
  different keys at any date. Dates outside 1677–2262 used to collide with the instant their
  nanosecond count wrapped to (`9999-12-31` and `1816-03-29T05:56:08.066277376Z` were one
  group, and joined each other). The same instant in two time zones is still one key.
