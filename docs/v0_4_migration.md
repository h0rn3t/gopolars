## v0.4.0 Migration Notes

Target release: **v0.4.0** — the next release after the `v0.3.0` tag. (Note: `v0.6`/`v0.7`/`v0.9`
elsewhere in `docs/` and `test/conformance/` are *conformance wave* numbers, not releases — see
[`versioning_policy.md`](versioning_policy.md).)

### **BREAKING**: `DataFrame.Clone` shares column buffers instead of deep-copying

`DataFrame.Clone` previously deep-copied every column buffer. It now returns a frame that **shares
the source frame's column buffers by pointer**, matching the semantics of Python Polars' `clone`
(a refcount operation, not a copy).

Measured at 1,000,000 rows: **3.02 ms / 44 MB / 17 allocations → 0.20 µs / 672 B / 5 allocations.**
The cost now scales with the column count instead of the row count.

**What is unchanged:**

- Reading is identical. The same cell read from a frame and from its clone yields the same value,
  including null positions.
- The clone is **structurally independent**: dropping, adding, renaming or reordering columns in
  the clone does not affect the source frame's schema or column set.
- Every query result is unchanged.

**Why this is safe.** No public method mutates a column in place, despite some Polars-inherited
names:

| method | actual behavior |
| --- | --- |
| `Series.Set` | copies into a fresh slice, returns a new `Series` |
| `DataFrame.DropInPlace` | returns a new frame |
| `DataFrame.SetSorted` | sorts and returns a new frame |
| `DataFrame.Shift`, `ReplaceColumn`, `InsertColumn` | return new frames |

Underneath, a `*chunk.Column` is immutable once it may be shared: slicing, gathering, filtering,
shifting, cloning and concatenating all allocate a new column and never write into the receiver.
Shared columns are marked (`MarkShared`), and the copy-on-write contract requires any future
in-place mutator to clone a shared column before writing (`CloneIfShared`). Column sharing was
already routine before this change — `Select`, `WithColumns` and `Series.Rename` have always shared
buffers.

The normative statement of the new behavior is the requirement *"Clone shares column buffers with
the source frame"* in `openspec/specs/zero-copy-projection/spec.md`, pinned by
`TestCloneDoesNotCopyBuffers`, `TestCloneIsStructurallyIndependent` and
`TestCloneReadsIdenticalValues` in `pkg/polars/rowop_perf_test.go`.

**How to check your code.** Ordinary use of `pkg/polars` needs no changes. Review only code that
steps outside the public API and relies on `Clone` having produced private buffers:

- passing `Series.Column()` into CGO or `unsafe` and writing through it;
- caching the typed slices returned by `Float64s()`, `Int64s()`, `Strings()`, `Bools()`, `Times()`
  or `Nulls()` with the intent of writing into them.

Both were already unsound — those accessors expose the column's live buffer, which other frames may
share regardless of `Clone`. If you genuinely need an independent buffer, please open an issue
rather than reintroducing a deep copy: it costs 3.02 ms and 44 MB per call at 1M rows.

### Highlights

Non-breaking, but user-visible:

- `DataFrame.Row`, `Rename` and `Drop` are now O(columns) instead of O(rows). `Row(i)` previously
  materialized *every* row to return one (153 ms / 381 MB / 5.6M allocations at 1M rows); it now
  reads only the requested row (0.19 µs). `Rename` and `Drop` share the untouched columns.
- Window and ordinal operations use all cores: `Expr.Over` went 18.7 ms → 4.89 ms at 1M
  (now ~2× faster than Polars), and `Expr.Rank` 21.2 ms → 14.1 ms.
- `WriteCSV` is ~7× faster (223.6 ms → 30.6 ms at 1M rows): cells are appended into a reused byte
  buffer instead of allocating a string each, and row ranges are formatted concurrently. Output is
  byte-for-byte identical to before, quoting included.
- **Sorting fix:** the radix argsort placed `-0.0` strictly before `0.0`. Since `<` treats them as
  equal, a stable sort must preserve their original order — so rows with `-0.0` and `0.0` in the
  sort key could come back in the wrong relative order. Both the sequential and parallel paths were
  affected; both are fixed.

### Migration guidance

- No API signatures changed; recompilation is enough for ordinary use.
- If you depended on `Clone` returning private buffers, audit the two patterns listed above.
- If you sort float columns containing both `-0.0` and `0.0` and asserted on the exact row order,
  update those expectations — the new order is the stable one.
