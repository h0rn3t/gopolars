## v0.9.0 Migration Notes

Target release: **v0.9.0**, the next release after the `v0.8.0` tag. This release fixes sorting,
concatenation and `is_in` results that were wrong without an error (OpenSpec change
`fix-sort-groupby-concat-correctness`). No exported signature changed. Two fixes change results
that were not wrong by accident but by rule; they are marked **BREAKING**. Behavior was checked
against Polars 2.0.0.

### **BREAKING**: `NullsLast` alone decides where nulls go in `Sort`

`DataFrame.Sort` used to reverse null placement for a descending key: with `NullsLast: false` the
nulls of a descending key came last, and with `NullsLast: true` they came first. Now, as in
Polars, `NullsLast` places nulls for every key whatever its direction:

| `Descending` | `NullsLast` | v0.8.0 | v0.9.0 |
|---|---|---|---|
| false | false | `null, 1, 2, 3` | `null, 1, 2, 3` |
| false | true | `1, 2, 3, null` | `1, 2, 3, null` |
| true | false | `3, 2, 1, null` | `null, 3, 2, 1` |
| true | true | `null, 3, 2, 1` | `3, 2, 1, null` |

NaN is still the largest Float64 value: last ascending, first descending.

**How to check your code.** Look for `Sort` calls with a `Descending: true` key on a column
that can hold nulls. If the code relied on nulls landing at the end of a descending sort, set
`NullsLast: true`.

### **BREAKING**: `Concat` rejects a column whose dtype differs between frames

Vertical `Concat` used to take each column's dtype from the first frame and read the other
frames' values as that dtype. A Float64 part concatenated after an Int64 part became zeros:
`[1, 2]` + `[3.5, 4.5]` gave `[1, 2, 0, 0]`. Now it fails with an error that names the column
and both dtypes:

```
concat vertical: column "x" has dtype int64 in frame 0 and float64 in frame 1
```

The same check applies to reading a directory of Parquet files (`ReadParquet` and `ScanParquet`
on a directory) and to DuckDB SQL results that are concatenated.

A column that is entirely null in one frame is still accepted whatever its dtype and contributes
nulls, so partitions where a column is all null (an Arrow `null` column imports as an all-null
String column) keep working. Polars 2.0.0 rejects this case unless `how="vertical_relaxed"`;
gopolars has no Null dtype, so it accepts it.

**How to check your code.** A new error from `Concat` means that frame's data was corrupted
before. Cast the column to one dtype before concatenating, for example with
`WithColumns(Col("x").Cast(dtypes.Float64))`.

### Fixed without an API change (results change only where they were wrong)

Re-check outputs that went through these operations with v0.8.0 or earlier:

- **Sort by a Datetime or Duration column did not sort.** Every value compared as equal, so rows
  stayed in input order. Datetime now orders by instant (the same instant in two time zones
  ties), Duration by length.
- **Datetime `min`/`max` in `group_by`** returned the group's first value. They now return the
  earliest and latest instant.
- **`rank` of a Datetime or Boolean column**, plain or inside `over`, gave every row the rank of
  its position. `Pivot` with `min`/`max` of a Boolean or Datetime column and the `sorted_asc` /
  `is_monotonic` flags of `Flags` had the same defect.
- **`Sort` is stable.** Rows with equal keys keep their input order on every path. A descending
  Int64/Float64 key of 256 rows or more used to reverse them, and String, Boolean and nullable
  keys gave an arbitrary order. `MaintainOrder` is accepted and has no further effect.
- **`LazyFrame.Sort` ignored `NullsLast`.** It now places nulls as `DataFrame.Sort` does.
- **`is_in` with a typed slice** (`Lit([]int64{1, 3})`, `[]float64`, `[]string`, `[]bool`,
  `[]time.Time`) matched no row. It now means the same as the `[]any` of its elements. A Datetime
  value matches the same instant in another time zone.
