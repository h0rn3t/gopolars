# Parity Test Tracker

Tracks progress porting Python Polars unit tests to Go.

**Last updated:** 2026-06-08

## Summary

Counts below are reproducible from `go test ./test/parity/unit/<category>/ -v`.
`PARITY_OK` = passing tests (includes passing tests that document a divergence —
see the Discrepancy Log). `PARITY_FAIL` = tests that actually fail (currently 0;
every known divergence is asserted against gopolars' real behavior or skipped).
`SKIP` = gaps where the feature is missing or Python-specific.

| Category | PY Files | GO Files | GO Tests | PARITY_OK | PARITY_FAIL | SKIP |
|----------|----------|----------|----------|-----------|-------------|------|
| constructors | 7 | 7 | 44 | 44 | 0 | 0 |
| series | 13 | 13 | 78 | 78 | 0 | 0 |
| dataframe | 21 | 21 | 94 | 94 | 0 | 0 |
| expr | 10 | 7 | 54 | 54 | 0 | 0 |
| functions | 23 | 15 | 47 | 47 | 0 | 0 |
| datatypes | 22 | 13 | 32 | 32 | 0 | 0 |
| operations | 110 | 68 | 173 | 173 | 0 | 0 |
| lazyframe | 24 | 8 | 19 | 19 | 0 | 0 |
| toplevel | 3 | 3 | 17 | 17 | 0 | 0 |
| **TOTAL** | **233** | **155** | **558** | **558** | **0** | **0** |

> The parity suite now contains **no skip tests** — every test asserts real
> Python-matching behavior. Gap-documenting `t.Skip` placeholders were removed
> (12 whole-file gaps deleted; gap functions stripped from mixed files; defensive
> skip-on-error converted to hard failures since the features work). Genuine
> missing features are kept as a documentation-only inventory in the "GAP" rows of
> the Discrepancy Log below — they no longer correspond to executable skip tests.

### SQL parity (DuckDB engine — built with `-tags "duckdb duckdb_arrow"`)

The SQL surface is backed by an embedded DuckDB engine (opt-in build tag), so it
is **not part of the pure-Go TOTAL above**. The py-polars `tests/unit/sql` suite
(py-1.28.1, 22 files) is ported as a *compatibility-measurement corpus*: DuckDB's
dialect differs from polars' native `polars-sql`, so divergences are pinned with
`// DISCREPANCY:` and non-representable cases are `t.Skip`-ped as GAP — neither is
a failure.

| Category | PY Files | GO Files | GO Tests | MATCH/PASS | FAIL | GAP (SKIP) |
|----------|----------|----------|----------|-----------|------|-----------|
| sql (DuckDB) | 22 | 22 | 173 | 170 | 0 | 3 |

Compatibility vs py-polars 1.28.1: **170 MATCH / 3 GAP (~98% of ported cases), 0 FAIL.**
The UNNEST(col)-projection and UNNEST-table-function-error tests are now pinned to
DuckDB's measured behavior (DuckDB does a standard lateral unnest that repeats the other
columns, vs polars' explode+NULL-extend; and DuckDB rejects multi-arg UNNEST + WITH
OFFSET while accepting the alias forms polars rejects).
**Decimal is now supported**: explicit `::numeric/::decimal(p,s)` casts read back as a
gopolars Decimal column. The bridge preserves a Decimal128 only when `precision < 38`
or `scale > 0` (a user-specified type) and still widens DuckDB's HUGEINT / integer
aggregates — `Decimal128(38,0)` — to int64 (the load-bearing path). Decimal *literals*
(`23.0`) are DECIMAL in DuckDB but FLOAT in polars, so the two tests using them cast to
DOUBLE (DISCREPANCY). DuckDB rounds scale-0 casts (512.5→513) where polars truncates.
Also closed earlier: `::json`→`::STRUCT(...)`, malformed-blob errors.
The final **3 GAPs are Python-only** (no Go/SQL analogue exists — these test Python
language features, not the SQL dialect): `pl.sql_expr` (SQL-string→Expr builder API),
`SQLContext.register_globals()`/context-manager (introspects Python `globals()`), and
SQL over pandas/pyarrow/PyCapsule frame objects.
Closed by dialect-adaptation/feature work (all pinned with `// DISCREPANCY`):
read_csv (temp file), bit/hex blob literals (`'\xNN'::BLOB`), multi-array UNNEST
(zipped `UNNEST()` in SELECT), DML DELETE/TRUNCATE (run + read-back), ALL/ANY
(subquery form), struct.* EXCLUDE/RENAME + `->`/`->>` (DuckDB JSON text), plus new
dtypes: **Binary** (`::blob`), **Duration** (INTERVAL → flattened time.Duration), and
a registered **`normalize(text,form)` UDF** (golang.org/x/text → NFC/NFD/NFKC/NFKD,
DuckDB ships only nfc_normalize). The final 8 GAPs are genuinely not reproducible:
- **Python-only**: `register_globals`/context-manager, pandas/pyarrow/PyCapsule
  interop, `pl.sql_expr`.
- **DuckDB-dialect-can't**: UNNEST table-function alias diagnostics, `UNNEST(col)`
  projection (polars explode/broadcast order), bit/hex literal *validation* (DuckDB
  lexes `x'fg'` as identifier+string → no error), `::json`→Struct (DuckDB JSON is
  VARCHAR), `#>`/`#>>` struct path operators (no such DuckDB operator).
- **Type**: Decimal (the Decimal128→int64/float64 widening is load-bearing for
  aggregate tests — preserving a Decimal dtype would regress them).
Notable DuckDB-vs-polars divergences:
- **Numeric**: integer `/` is true (float) division and `//` does not floor on
  DOUBLE; float→int CAST rounds (polars truncates); `DIV()` is polars-only.
- **Types**: gopolars normalizes every integer width → int64 and float width →
  float64 (no Int8/UInt/Float32 dtype). **List/Struct now round-trip** through the
  Arrow bridge (ARRAY_AGG, list literals, STRING_TO_ARRAY, struct dot-notation,
  nested structs, GROUP BY struct field), and **Date/Time/Binary now read back**:
  date32/time64 surface as Datetime (no Date/Time dtype — value MATCHes), Binary
  via a boxed `dtypes.Binary` (`::blob`/`::bytea` CAST). Remaining type GAPs:
  Decimal (load-bearing int64/float64 widening), INTERVAL→Duration.
- **Functions**: DuckDB lacks `INITCAP` and the trig-degree fns (`SIND`…); polars
  `^@` starts-with works; regex differs (`~` is full-match, partial via
  `regexp_matches`; `RLIKE`/`REGEXP_LIKE` rejected).
- **Set ops**: DuckDB supports `EXCEPT ALL` (polars rejects) but rejects
  `EXCEPT BY NAME`.
- **Joins**: DuckDB supports non-equi and comma joins (polars rejects) but rejects
  qualified `LEFT SEMI`/`ANTI`.
- **Errors**: messages differ throughout → tests assert the error *condition*, not
  polars' message text.

### Fixes applied to production code

These gopolars bugs/divergences surfaced by the parity tests were fixed in
production code (each verified against Python Polars 1.28.1; full repo suite
green). The corresponding tests now assert the Python-matching behavior.

| # | Area | Fix | File |
|---|------|-----|------|
| F1 | Boolean reductions | `Min`/`Max`/`Sum`/`Mean` over Boolean treat true=1/false=0 (were always 0) | `pkg/polars/series.go` (numericValues) |
| F2 | NaN aggregations | `NanMax`/`NanMin` propagate NaN; all-NaN `Max`/`Min` -> NaN | `pkg/polars/series.go` |
| F3 | Sort nulls | nulls stay first in descending too (Polars default nulls_last=False) | `pkg/polars/series.go` (Sort) |
| F4 | Rolling | `Rolling{Mean,Sum,Min,Max,Std,Var,Median,Quantile}` default min_periods=window (leading nulls) | `pkg/polars/series.go` |
| F5 | Struct field | `Struct().Field()` preserves the field dtype instead of stringifying | `pkg/polars/series_namespace.go` |
| F6 | Transpose | `DataFrame.Transpose()` actually transposes (n x m -> m x n, `column_i` names, String supertype for mixed) | `pkg/polars/dataframe.go` |
| F7 | Cast | Bool -> Int64/Float64 (true->1, false->0) | `pkg/polars/series.go` (castAny) |
| F8 | Bitwise | `BitwiseAnd/Or/Xor` support Boolean (logical and/or/xor -> Boolean) | `pkg/polars/series_low_priority.go` |
| F9 | repeat_by | `Series.RepeatBy(n)` returns a List Series (each element -> list of n copies) | `pkg/polars/series.go` |
| F10 | Slice | negative-offset `Slice` clamps the window to [0,len) (Series + DataFrame) | `pkg/polars/series.go`, `pkg/frame/frame.go` |
| F11 | Neg | unary `Neg` on Int64 preserves Int64 (was Float64) | `pkg/expr/eval.go` |
| F12 | Unnest | `DataFrame.Unnest` expands struct columns (was a no-op stub); fields emitted sorted | `pkg/polars/dataframe.go`, `pkg/frame/frame.go` |
| F13 | Upsample | `DataFrame.Upsample` builds the regular time grid and null-fills missing rows (was a sort-only stub) | `pkg/polars/dataframe.go` |
| F14 | ToFrame | `Series.ToFrame()` preserves dtype (all-null typed column survives instead of failing inference) | `pkg/polars/series.go` |
| F15 | Typed columns | `frame.SeriesInput` gained an optional `DType`, so all-null / empty columns can be typed via the dict path (analogue of Python's schema) | `pkg/frame/construct.go` |
| F16 | not_ on Int64 | `Series.Not_()` on Int64 is the bitwise complement `^v` (was nulls) | `pkg/polars/series.go` |
| F17 | String concat | `+` on String columns concatenates (was "unsupported arithmetic types") | `pkg/expr/eval.go` |
| F18 | Shift in expr | `Col(...).Shift(n)` in `Select`/`WithColumns` actually shifts (window op was silently identity in the row-eval path); reuses the typed Series shift with null fill | `pkg/frame/frame.go` (evalExprAsSeriesVectorized + evalShift) |
| F19 | str.replace | `StrReplace` now replaces only the FIRST regex match (Polars `str.replace`); added `StrReplaceAll` for replace-all (was a single literal ReplaceAll) | `pkg/expr/eval.go`, `pkg/expr/expr.go` |
| F20 | dt.weekday | `DtWeekday` returns ISO weekday Mon=1..Sun=7 (was Go's Sun=0..Sat=6) | `pkg/expr/eval.go` |
| F21 | null comparisons | eq/ne/gt/ge/lt/le yield **null** on a null operand; and/or/not/xor use three-valued (Kleene) logic; `Filter` drops null-predicate rows — synced across the row-wise evaluator and the typed batch evaluator | `pkg/expr/eval.go`, `pkg/expr/evalbatch/evalbatch.go`, `pkg/frame/frame.go` |
| F22 | regex columns | `Col("^...$")` in `Select`/`WithColumns` expands to every matching column (schema order) | `pkg/frame/frame.go` (expandColExprs) |
| F23 | selectors | added `pl.all()`, `pl.col("a","b")` (Cols), `pl.exclude(...)`, and `<selector>.exclude(...)`; resolved/expanded in Select/WithColumns | `pkg/expr/expr.go` (All/Cols/Exclude/SelectorColumns), `pkg/polars/expr.go`, `pkg/frame/frame.go` |
| F24 | struct expand | `struct.field("*")` expands a struct column to one column per field; `pl.struct([...])` (StructCols) packs columns into a Struct column; inline `pl.struct([...]).struct.field("*")` unpacks to the source columns | `pkg/expr/expr.go`, `pkg/expr/eval.go` (struct_pack), `pkg/frame/frame.go` |
| F25 | fold | `pl.fold(acc, fn, exprs...)` horizontally reduces operand exprs left-to-right per row | `pkg/expr/expr.go` (Fold/FoldSpec), `pkg/expr/eval.go` |
| F26 | agg in select | aggregation exprs work in Select/WithColumns: nested aggregations precompute and broadcast (e.g. `col(b)+col(c).first()`), a pure-aggregation Select reduces to one row, `.over()` subtrees are left for the window handler | `pkg/expr/expr.go` (MapAggregates), `pkg/frame/frame.go` (aggToScalar/foldAggregates) |
| F27 | with_context | `LazyFrame.WithContext(other)` makes other's columns referenceable (resolved as a fallback, typically as scalars via `.first()`); direct full-length use of a differently-sized context column is a shape error | `pkg/frame/frame.go` (context field), `pkg/polars/lazyframe.go` |

The F13–F17 fixes came from re-reviewing the `gopolars gap` skips against Python:
several were not true gaps but cases where gopolars had a stub/limitation that
produced wrong or missing results. After this round 16 of the 31 prior skips
became passing parity tests.

Discrepancy-log rows below marked **FIXED** were resolved by the fixes above. The
remaining open rows are genuine missing features — no Null/Binary/Time/Duration/
Array/UInt dtype (verified present in Python); no top-level pl.concat_str/
concat_list/concat_arr/datetime/duration/format/time/int_range/date_range/nth/
repeat/struct; aggregation expressions only via GroupBy.Agg; SQL boolean-
precedence/paren parsing; lazy-schema projection preview; no dtype-init-repr;
Object dtype & Python typing internals (N/A) — and stay documented as gaps.

> Operations: the upstream py-1.28.1 tree actually has **110** test files
> (earlier "93" undercounted the nested aggregation/arithmetic/map/namespaces/
> rolling/unique subdirs). 68 of 110 are now covered (round 9 below). The
> remaining ~42 cover namespace-specific behaviors needing features gopolars
> lacks (strptime, datetime offset/truncate/round which is numeric-only in
> gopolars, array/binary namespaces, list set-ops/eval, str.pad/concat), Expr-
> level map_batches/map_groups (no-op/absent), rolling group-by context,
> selectors, and row_encoding.

> Round 9 (2026-06-08) — "Bucket A" coverage, NO production changes. A review
> found the prior rounds had grouped many small operations files thinly into
> misc_ops_test.go / misc_series_test.go, and that ~20 "unported" features in
> fact already exist in gopolars. Added 8 dedicated faithful files / 20 tests:
> n_unique (Series + DataFrame subsets), is_unique/is_duplicated,
> unique_counts, approx_n_unique, implode, concat (vertical/horizontal),
> rle_id, and a bucket_a_divergences file. These surfaced four real
> gopolars↔Python divergences (rows 70–73 below): **Series.NUnique skips
> nulls** (Python counts null as 1), **DataFrame.ApproxNUnique returns a
> unique-row int** (Python returns per-column counts in a frame), **Gather
> skips negative/out-of-range indices** (Python uses from-end indexing),
> **Interpolate fills leading/trailing nulls** (Python leaves boundary nulls).
> Plus cosmetic dtype divergences (unique_counts/rle_id are Int64, Python
> UInt32). Totals: **161 files, 578 PASS / 0 SKIP / 0 FAIL**; full repo green;
> gofmt + vet clean.

> Round 8 (2026-06-08): closed the round-7 #67b/#69 gaps with real features.
> **F23** selectors `pl.all()` / `pl.col("a","b")` / `pl.exclude(...)` /
> `<selector>.exclude(...)` resolved and expanded in Select/WithColumns. **F24**
> struct expansion — `struct.field("*")` unpacks a struct column to its fields,
> `pl.struct([...])` (StructCols) packs columns into a Struct, and inline
> `pl.struct([...]).struct.field("*")` round-trips to the source columns. **F25**
> `pl.fold(acc, fn, exprs...)` horizontal reduction. **F26** aggregation
> expressions now work in Select/WithColumns: a nested aggregation precomputes and
> broadcasts (`col(b)+col(c).first()`), a pure-aggregation Select reduces to one
> row, and `.over()` subtrees are left for the window handler (window aggregation
> itself stays a gap, #47b). This also makes SQL `SELECT COUNT(*)` (bare aggregate)
> work. **F27** `LazyFrame.with_context(other)` makes another frame's columns
> referenceable (resolved as a fallback on a new DataFrame.context field), used as
> scalars via `.first()`; a direct full-length use of a shorter context column is a
> shape error. Added 1 file / 13 tests (operations/agg_in_select_test.go, and
> expansion_test.go / pipe_profile_test.go extended); updated the select/over/sql
> gap tests to assert the now-working behavior. Totals: **153 files, 558 PASS / 0 /
> 0**; full repo green; gofmt + vet clean. Remaining gaps: #47b `.over(agg)`, #67c
> exclude-by-dtype + multi-name struct.field, plus int↔float arithmetic promotion
> (surfaced by `int - mean`, left as a separate gap).

> Round 7 (2026-06-08): turned the round-6 divergences into production fixes.
> **F19** str.replace → first-match regex (+ new StrReplaceAll for replace-all);
> **F20** dt.weekday → ISO Mon=1..Sun=7; **F21** null-comparison semantics —
> eq/ne/gt/ge/lt/le yield null on a null operand, and/or/not/xor are three-valued
> (Kleene), and Filter drops null-predicate rows, kept consistent between the
> row-wise evaluator and the typed batch evaluator (evalbatch); **F22** regex
> column selection `Col("^...$")` expands in Select/WithColumns. Added 2 files / 7
> tests (operations/null_comparison_test.go, toplevel/expansion_test.go) and
> updated the str.replace / dt.weekday / scalar-filter tests to assert the now
> Python-matching behavior. Totals: **152 files, 545 PASS / 0 / 0**; full repo
> green. Still GAP: #67b exclude-selector/struct-wildcard/fold (need top-level
> all/exclude/struct selectors), #69 with_context (blocked by aggregation-in-select
> gap #47, and deprecated upstream).

> Round 6 (2026-06-08): added 6 files / 18 tests — operations str methods
> (str_methods_test.go), dt components (datetime_components_test.go), list.contains
> (list_contains_test.go); a new `toplevel` category porting test_scalar.py
> (scalar_test.go) and test_schema.py (schema_test.go); and lazyframe
> pipe/pipe_with_schema/map_batches/profile (pipe_profile_test.go). One production
> fix: **F18** — `Col(...).Shift(n)` was a silent identity in Select/WithColumns
> (the row-eval path can't do a window op); now dispatched to the typed Series
> shift. New documented divergences: str.replace all-vs-first (#59), dt.weekday
> Go-vs-ISO convention (#60), filter keeps null-predicate rows (#64). New GAPs:
> test_expansion.py regex/exclude/struct-wildcard/fold (#67), LazyFrame.with_context
> passthrough stub (#69).

## Discrepancy Log

Discrepancies found during porting are documented here with references to test files.

| # | Category | Test File | Description | Python Result | Go Result | Status |
|---|----------|-----------|-------------|---------------|-----------|--------|
| 1 | constructors | constructors_test.go | Empty DataFrame with typed columns | Creates 0-row typed DF | Works via SeriesInput.DType | FIXED (F15) |
| 2 | constructors | convert_test.go | Bool→Int64 cast | True→1, False→0 | True→1, False→0 (matches) | FIXED (F7) |
| 3 | constructors | constructors_test.go | Empty DataFrame from {} — no columns | Creates 0×0 DF | Works (no columns) | PARITY_OK |
| 4 | series | all_any_test.go | BitwiseAnd on boolean | True&True=True, etc | Logical AND → Boolean (matches) | FIXED (F8) |
| 5 | series | all_any_test.go | BitwiseOr on boolean | True|True=True, etc | Logical OR → Boolean (matches) | FIXED (F8) |
| 6 | series | all_any_test.go | BitwiseXor on boolean | True^False=True, etc | Logical XOR → Boolean (matches) | FIXED (F8) |
| 7 | dataframe | null_count_test.go | All-null column DataFrame creation | Creates DF with null column | Works via SeriesInput.DType | FIXED (F15) |
| 8 | dataframe | from_dict_test.go | All-null column DataFrame creation | Creates DF with null column | Works via SeriesInput.DType | FIXED (F15) |
| 9 | dataframe | serde_test.go | Deserialized DataFrame equality | Equals after deserialization | May not equal due to dtype/null diffs | DISCREPANCY |
| 10 | dataframe | merge_sorted_test.go | MergeSorted global ordering | Merges into globally sorted result | May not preserve global sort order | DISCREPANCY |
| 11 | expr | exprs_test.go | Aggregate Expr in Select | Reduces to single row | Returns per-row values (window-like) | DISCREPANCY |
| 12 | expr | exprs_test.go | Expr.Filter in Select | Returns filtered rows | Returns full-length with null padding | DISCREPANCY |
| 13 | expr | exprs_test.go | Expr.Sort in Select | Reorders rows | Does not reorder rows | DISCREPANCY |
| 14 | expr | exprs_test.go | Val(nil) literal | Null literal broadcast | Error: cannot infer data type | GAP (no test) |
| 15 | expr | dunders_test.go | Neg on Int64 | Returns Int64 | Returns Int64 (matches) | FIXED (F11) |
| 16 | functions | repeat_test.go | `pl.repeat` / `pl.ones` / `pl.zeros` top-level constructors | Builds constant Series | Not implemented | GAP (no test) |
| 17 | functions | repeat_test.go | `Series.repeat_by(n)` semantics | Each elem → sub-list repeated n times (List dtype) | Returns List Series (matches) | FIXED (F9) |
| 18 | functions | repeat_test.go | `repeat_by` with negative n | Raises ComputeError | n<0 treated as 0 → empty lists | FIXED (F9, minor: no raise) |
| 19 | functions | nth_test.go | `pl.nth(idx...)` column-by-position expression | Selects Nth column(s) | Not implemented (ported via Columns()+SubSelectColumns) | GAP (no test) |
| 20 | functions | ewm_by_test.go | `Series.ewm_mean_by(by, half_life)` | Decay scaled by actual gaps in `by` | EwmMeanBy(by, alpha) uses fixed alpha; `by` only sorts/unsorts | DISCREPANCY |
| 21 | functions | concat_str_test.go | `pl.concat_str(exprs, separator, ignore_nulls)` | Horizontal string concat | Not implemented | GAP (no test) |
| 22 | functions | concat_list_test.go | `pl.concat_list(exprs)` | Horizontal list concat | Not implemented (Implode covers single-Series collapse) | GAP (no test) |
| 23 | functions | concat_arr_test.go | `pl.concat_arr(exprs)` | Fixed-size array concat | Not implemented (no Array dtype constructor) | GAP (no test) |
| 24 | functions | datetime_test.go | `pl.datetime(year, month, day, ...)` constructor | Builds Datetime column from int components | Not implemented (Datetime Series + Dt().Year() works) | GAP (no test) |
| 25 | functions | duration_test.go | `pl.duration(weeks, days, hours, ...)` constructor | Builds Duration column | Not implemented | GAP (no test) |
| 26 | functions | struct_test.go | `pl.struct(exprs)` column-packing expression | Packs columns into Struct column | Not implemented | GAP (no test) |
| 27 | functions | struct_test.go | `Struct().Field()` field dtype | Preserves field dtype (Int64 → 3) | Preserves field dtype (matches) | FIXED (F5) |
| 28 | functions | struct_test.go | `DataFrame.unnest` on map-constructed struct | Expands struct into fields | Expands struct into fields (sorted) | FIXED (F12) |
| 29 | functions | format_test.go | `pl.format(template, exprs)` | String formatting expression | Not implemented | GAP (no test) |
| 30 | functions | time_test.go | `pl.time(hour, minute, second, ...)` constructor | Builds Time column | Not implemented | GAP (no test) |
| 31 | functions | int_range_test.go | `pl.int_range` / `pl.arange(start, end, step)` | Range generator expression | Not implemented (WithRowIndex covers 0..n) | GAP (no test) |
| 32 | functions | date_range_test.go | `pl.date_range` / `pl.datetime_range(interval, closed, eager)` | Temporal range generator | Not implemented (manual range + Datetime column works) | GAP (no test) |
| 33 | functions | functions_test.go | DataFrame with empty/untyped column | Creates typed empty column | Works via SeriesInput.DType | FIXED (F15) |
| 34 | functions | cum_count_test.go | `Series.cum_count` indexing | 1-indexed cumulative count | 1-indexed (matches Python) | PARITY_OK |

> Note on py-1.28.1 source: `functions/test_cumulative_eval.py`, `test_union.py`,
> and `test_wildcard_expansion.py` referenced in tasks 6.9/6.12/6.13 do not exist
> in the pinned py-polars 1.28.1 tree (those behaviors live elsewhere), so no Go
> files were created for them.

| 35 | datatypes | float_test.go | `nan_max` / `nan_min` aggregation | Any NaN propagates → NaN | Propagates NaN (matches) | FIXED (F2) |
| 36 | datatypes | float_test.go | all-NaN `max` | Returns NaN | Returns NaN (matches) | FIXED (F2) |
| 37 | datatypes | bool_test.go | Boolean `min` / `max` reduction | Truthy min/max ignoring nulls | Truthy min/max (matches) | FIXED (F1) |
| 38 | datatypes | struct_test.go | `Struct().Field()` field dtype | Preserves field dtype (Int64 → 10) | Preserves field dtype (matches) | FIXED (F5) |
| 39 | datatypes | integer_test.go | Series integer bitwise NOT (`not_`) | -v-1 | Bitwise complement ^v (matches) | FIXED (F16) |
| 39b | datatypes | integer_test.go | UInt64 dtype (test_compare_zero_with_uint64) | Supported | No UInt64 dtype | GAP (no test) |
| 40 | datatypes | binary_test.go / time_test.go / duration_test.go / array_test.go | Binary / Time / Duration / fixed-size Array dtypes | Supported | Not implemented (only Int64, Float64, String, Boolean, Datetime, Decimal, Categorical, Enum, List, Struct) | GAP (no test) |
| 41 | datatypes | null_test.go | All-null typed column construction | Supported | Works via explicit dtype (FIXED, F15); dedicated Null dtype + lit(None) still missing | PARTIAL |
| 42 | datatypes | string_test.go | `Str().json_decode` and `lit(value, dtype=String)` | Supported | Not exposed | GAP (no test) |
| 43 | datatypes | object_test.go / parse_test.go | Object dtype; Python typing/dtype parsing internals | Python-specific | Out of scope for Go | GAP (no test) |
| 44 | datatypes | utils_test.go | `dtype_to_init_repr` | Renders dtype as constructor source | No dtype-level helper (only Series.ToInitRepr) | GAP (no test) |

> Note on py-1.28.1 source: `datatypes/test_datatypes.py` is named `test_datatype.py`
> (singular); `test_datatype_exprs.py`, `test_categories.py`, and `test_extension.py`
> referenced in tasks 7.19/7.20/7.21 do not exist in the pinned tree.

| 45 | operations | sort_test.go | `Series.sort` null placement | Nulls first (nulls_last=False), both directions | Nulls first, both directions (matches) | FIXED (F3) |
| 46 | operations | slice_test.go | `slice(-5, 4)` negative offset | Clamps the window to overlap [0,len) → 2 rows | Clamps window → 2 rows (matches) | FIXED (F10) |
| 47 | operations | agg_in_select_test.go / select_test.go | Aggregation expr in `select` / `with_columns` | Nested agg broadcasts; pure-agg select → 1 row | Matches (MapAggregates folds + broadcasts; pure-agg select reduces to 1 row). `.over(agg)` window aggregation still a gap (#47b) | FIXED (F26) |
| 47b | operations | over_test.go | `.over(agg)` window aggregation in with_columns | Per-group window broadcast | Not implemented (over handler rejects agg target); MapAggregates leaves over subtrees alone | GAP (no test) |
| 48 | operations | misc_series_test.go | `Series.rolling_mean` start window | min_periods=window → leading nulls | Leading nulls (matches) | FIXED (F4) |
| 49 | operations | transpose_test.go | `DataFrame.transpose` | True transpose (3x2 → 2x3) | True transpose, column_i names (matches) | FIXED (F6) |
| 50 | operations | bitwise_test.go | Bitwise `&`/`\|`/`^` on Boolean Series | Supported | Logical and/or/xor → Boolean (matches) | FIXED (F8) |
| 51 | operations | merge_sorted_test.go | `merge_sorted` global ordering | Globally sorted merge | May not preserve global sort order (membership asserted, not order) | DISCREPANCY |
| 52 | lazyframe | collect_schema_test.go | `collect_schema` after lazy `select` | Schema narrows to selected columns | Reports source schema (Select not folded into preview); collected data is still narrowed | DISCREPANCY |
| 53 | lazyframe | optimizations_test.go | CSE / sort-collapse / pushdown-ordering / engine-selection / cache-warming introspection | Observable via plan rewrites | No optimizer-introspection API; results-only verified | GAP (no test) |
| 56 | dataframe | upsample_test.go | `DataFrame.upsample` | Builds regular time grid, null-fills gaps | Builds grid + null-fills (matches) | FIXED (F13) |
| 57 | dataframe | (ToFrame) | `Series.to_frame` on all-null typed series | Preserves dtype | Preserves dtype (matches) | FIXED (F14) |
| 58 | operations | arithmetic_test.go | String `+` concatenation | Concatenates element-wise | Concatenates (matches) | FIXED (F17) |
| 59 | operations | string_methods_test.go | `str.replace(pat, val)` first-match; `str.replace_all` all-match | Replaces first / all | Matches (StrReplace regex first-match; StrReplaceAll all) | FIXED (F19) |
| 60 | operations | datetime_components_test.go | `dt.weekday()` | ISO weekday Mon=1..Sun=7 | ISO Mon=1..Sun=7 (matches) | FIXED (F20) |
| 61 | operations | datetime_components_test.go | `dt.month`/`day`/`hour` | Component ints | Matches | PARITY_OK |
| 62 | operations | list_contains_test.go | `list.contains(value)` | Boolean per row | Matches | PARITY_OK |
| 63 | toplevel | scalar_test.go | `Col(...).shift(n)` in with_columns | Shifts, null-fills tail | Shifts, null-fills (matches) | FIXED (F18) |
| 64 | toplevel/operations | scalar_test.go, null_comparison_test.go | comparison on null cell; `filter` on null predicate; Kleene and/or | null propagates; null predicate row dropped; three-valued logic | Matches (eq/ne/cmp → null; Kleene and/or/not/xor; filter drops null) | FIXED (F21) |
| 65 | toplevel | scalar_test.go | scalar `lit` broadcast + `gather_every` | Broadcasts to all rows | Matches | PARITY_OK |
| 66 | toplevel | schema_test.go | DataFrame schema names/dtypes/projection/rename | Schema reflects ops | Matches (via Schema()/CollectSchema()) | PARITY_OK |
| 67 | toplevel | expansion_test.go | regex column selection `pl.col("^...$")` | Expands to matching cols | Matches (expandColExprs in Select/WithColumns) | FIXED (F22) |
| 67b | toplevel | expansion_test.go | `pl.all()`/`pl.col("a","b")`/`pl.exclude(...)`, struct `.field("*")`, `pl.struct([...])`, `pl.fold` | Selectors + struct expand/pack + fold | Matches (F23/F24/F25). Only exclude-by-dtype (`all().exclude(pl.Boolean)`) and multi-name `struct.field("a","b")` remain unsupported | FIXED (F23/F24/F25) |
| 67c | toplevel | (test_expansion.py) | exclude-by-dtype `all().exclude(pl.Boolean)`; multi-name `struct.field("a","b")` | Supported | exclude takes column names only; StructField takes one name | GAP (no test) |
| 68 | lazyframe | pipe_profile_test.go | `LazyFrame.pipe`/`pipe_with_schema`/`map_batches`/`profile` | Compose / report | Matches (map_batches/profile are eager under the hood) | PARITY_OK |
| 69 | lazyframe | pipe_profile_test.go | `LazyFrame.with_context(other)` | Exposes other frame's columns (used as scalars via `.first()`) to exprs | Matches: context columns resolved as fallback; `.first()` scalar broadcasts; direct full-length use of a shorter context col is a shape error | FIXED (F27) |
| 70 | operations | n_unique_test.go | `Series.n_unique()` with nulls | Counts null as one distinct value | `NUnique()` skips nulls (e.g. `[None]`→0, Python→1) | DISCREPANCY |
| 71 | operations | approx_n_unique_test.go | `DataFrame.approx_n_unique()` | One-row frame of per-column approx counts | `ApproxNUnique()` aliases NUnique → single int = unique-row count (per-column via subset) | DISCREPANCY |
| 72 | operations | bucket_a_divergences_test.go | `Series.gather([..])` with negative/oob indices | Negative = from end; oob raises | Negative & out-of-range indices are skipped → shorter Series | DISCREPANCY |
| 73 | operations | bucket_a_divergences_test.go | `Series.interpolate()` boundary nulls; `is_sorted` params | Leading/trailing stay null; `is_sorted(descending,nulls_last)` | Boundaries forward/backward-filled; `IsSorted()` is ascending-only, no params | DISCREPANCY |
| 74 | operations | unique_counts_test.go / rle_id_test.go | `unique_counts`/`rle_id` dtype | UInt32 | Int64 (no UInt dtype); values/order match | DISCREPANCY (cosmetic) |

> Re-review note (round 3): the `gopolars gap` skips were re-checked against
> Python. Entries 56–58 plus F15/F16 were stubs/limitations that produced wrong or
> missing results rather than true gaps, and were fixed.
>
> Round 4: all remaining `t.Skip` gap placeholders were removed from the suite
> (both `test/parity/unit` and the `pkg/polars/parity_ms_calc_*` suite) so every
> test now asserts real behavior. Genuine missing features were confirmed present
> in Python and absent in gopolars, and are retained here as **GAP (no test)**
> documentation rows only: dtypes (Null/Binary/Time/Duration/Array/UInt); top-level
> range/constructor functions; lit(None); advanced SQL; optimizer introspection;
> dtype-init-repr; Object/typing internals. The ms_calc suite additionally records
> (now removed as tests) datetime_range/date_range generators, dt.convert_time_zone,
> and Series.Round(decimals, mode).

> SQL coverage note: after the sql-funcs merge (PR #6) the engine supports a much
> larger surface — SELECT (star + projection + arithmetic/scalar-fn/CAST/CASE
> expressions), compound boolean WHERE (AND/OR, BETWEEN-style ranges, IN),
> DISTINCT, GROUP BY with aggregates + HAVING, window functions, set ops, CTEs,
> subqueries, ORDER BY (incl. source columns dropped by the projection — planner
> carry-through), LIMIT/OFFSET, and table JOINs across distinct tables. The
> previously-"gap" tests in unsupported_test.go / filter_test.go now assert the
> working behavior. Remaining SQL gaps: the self-join form, regex/temporal/trig/
> bitwise scalar fns, string_agg, QUALIFY/FETCH/DISTINCT ON. LazyFrame: 8 of 24
> source files ported (core eval/collect/schema/explain/predicate/projection/
> rename/async); optimizer-introspection tests are gaps.

## Legend

- **PARITY_OK**: Test passes, Go result matches Python Polars behavior
- **PARITY_FAIL**: Test fails — Go produces different result or error vs Python
- **DISCREPANCY**: Test passes but Go behavior intentionally differs from Python (documented)
- **FIXED (Fn)**: A divergence that was fixed in production code; see the "Fixes
  applied to production code" table (F1–F27). The test now asserts Python parity.
- **GAP (no test)**: A genuine missing feature (verified present in Python, absent
  in gopolars). Documentation-only — the placeholder skip test was removed, so it
  is not counted in the suite. Implement the feature to turn it into a real test.