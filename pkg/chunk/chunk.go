// Package chunk provides gopolars' canonical typed column storage.
//
// A Column holds a dense, typed value buffer plus a validity mask (one bool
// per row, true == null) instead of the legacy boxed []any representation.
// Primitive dtypes (Int64, Float64, String, Boolean, Datetime, and the
// string-backed Categorical/Enum) use typed slices so kernels can operate on
// []float64 / []int64 without per-element interface boxing. Unsupported
// dtypes (Decimal, List, Struct) fall back to a boxed []any backing.
package chunk

import (
	"fmt"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// Column is a typed, single-column value buffer with a validity mask.
//
// Exactly one typed backing slice is populated for a given dtype; boxed holds
// values for dtypes without a typed mapping. nulls has length n (true == null).
//
// A Column is treated as immutable once it may be shared by more than one
// frame (see MarkShared / zero-copy projection). Slicing, gathering, filtering,
// shifting, cloning and concatenating all allocate new columns and never mutate
// the receiver, so read-only sharing by pointer is safe; any future in-place
// mutator must clone a shared column first (cloneIfShared).
type Column struct {
	dtype dtypes.DataType
	n     int
	nulls []bool

	// nullCount caches the number of null rows. unknownNullCount (-1) means the
	// count has not been computed yet; it is recomputed lazily on first request
	// and is left unknown at every site that produces or mutates validity.
	// Accessed atomically (NullCount) so concurrent readers of a shared column
	// are race-free.
	nullCount int64

	// shared marks a column that may be referenced by more than one frame. A
	// shared column must be treated as read-only; in-place mutators clone first.
	shared bool

	f64   []float64
	i64   []int64
	str   []string
	bln   []bool
	tim   []time.Time
	boxed []any
}

// unknownNullCount is the sentinel for a not-yet-computed cached null count
// (mirrors Arrow's kUnknownNullCount).
const unknownNullCount = -1

// normalizeNulls returns a validity slice of length n. A nil input means no
// nulls. The returned slice is owned by the Column.
func normalizeNulls(nulls []bool, n int) []bool {
	if nulls == nil {
		return make([]bool, n)
	}
	if len(nulls) != n {
		out := make([]bool, n)
		copy(out, nulls)
		return out
	}
	return nulls
}

// NewFloat64 builds a Float64 column. The caller transfers ownership of values
// and nulls (no copy is made).
func NewFloat64(values []float64, nulls []bool) *Column {
	return &Column{dtype: dtypes.Float64, n: len(values), f64: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// NewInt64 builds an Int64 column (ownership transferred).
func NewInt64(values []int64, nulls []bool) *Column {
	return &Column{dtype: dtypes.Int64, n: len(values), i64: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// NewString builds a String column (ownership transferred).
func NewString(values []string, nulls []bool) *Column {
	return &Column{dtype: dtypes.String, n: len(values), str: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// NewBool builds a Boolean column (ownership transferred).
func NewBool(values []bool, nulls []bool) *Column {
	return &Column{dtype: dtypes.Boolean, n: len(values), bln: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// NewTime builds a Datetime column (ownership transferred).
func NewTime(values []time.Time, nulls []bool) *Column {
	return &Column{dtype: dtypes.Datetime, n: len(values), tim: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// NewBoxed builds a boxed column for a dtype without a typed mapping
// (ownership transferred).
func NewBoxed(dtype dtypes.DataType, values []any, nulls []bool) *Column {
	return &Column{dtype: dtype, n: len(values), boxed: values, nulls: normalizeNulls(nulls, len(values)), nullCount: unknownNullCount}
}

// FromAny converts a legacy []any input into a typed Column for the given
// dtype, validating each non-nil value. nil entries become nulls. This is the
// one-time ingest cost paid by series.New.
func FromAny(dtype dtypes.DataType, values []any) (*Column, error) {
	n := len(values)
	nulls := make([]bool, n)
	switch dtype {
	case dtypes.Int64:
		buf := make([]int64, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			x, ok := v.(int64)
			if !ok {
				return nil, fmt.Errorf("expected int64 at index %d", i)
			}
			buf[i] = x
		}
		return &Column{dtype: dtype, n: n, i64: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.Float64:
		buf := make([]float64, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			x, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("expected float64 at index %d", i)
			}
			buf[i] = x
		}
		return &Column{dtype: dtype, n: n, f64: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		buf := make([]string, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			x, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("expected string at index %d", i)
			}
			buf[i] = x
		}
		return &Column{dtype: dtype, n: n, str: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.Boolean:
		buf := make([]bool, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			x, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool at index %d", i)
			}
			buf[i] = x
		}
		return &Column{dtype: dtype, n: n, bln: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.Datetime:
		buf := make([]time.Time, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			x, ok := v.(time.Time)
			if !ok {
				return nil, fmt.Errorf("expected time.Time at index %d", i)
			}
			buf[i] = x
		}
		return &Column{dtype: dtype, n: n, tim: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.Decimal:
		buf := make([]any, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			switch v.(type) {
			case dtypes.DecimalValue, string:
				buf[i] = v
			default:
				return nil, fmt.Errorf("expected decimal value at index %d", i)
			}
		}
		return &Column{dtype: dtype, n: n, boxed: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.List:
		buf := make([]any, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			if _, ok := v.([]any); !ok {
				return nil, fmt.Errorf("expected []any at index %d", i)
			}
			buf[i] = v
		}
		return &Column{dtype: dtype, n: n, boxed: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	case dtypes.Struct:
		buf := make([]any, n)
		for i, v := range values {
			if v == nil {
				nulls[i] = true
				continue
			}
			if _, ok := v.(map[string]any); !ok {
				return nil, fmt.Errorf("expected map[string]any at index %d", i)
			}
			buf[i] = v
		}
		return &Column{dtype: dtype, n: n, boxed: buf, nulls: nulls, nullCount: unknownNullCount}, nil
	default:
		return nil, fmt.Errorf("unsupported data type %s", dtype)
	}
}

// DataType returns the column dtype.
func (c *Column) DataType() dtypes.DataType { return c.dtype }

// Len returns the number of rows.
func (c *Column) Len() int {
	if c == nil {
		return 0
	}
	return c.n
}

// IsNull reports whether row i is null.
func (c *Column) IsNull(i int) bool { return c.nulls != nil && c.nulls[i] }

// Nulls returns the underlying validity slice (true == null). Callers must not
// mutate it.
func (c *Column) Nulls() []bool { return c.nulls }

// ValueAt returns the boxed value at row i, or nil when the row is null.
func (c *Column) ValueAt(i int) any {
	if c.IsNull(i) {
		return nil
	}
	switch c.dtype {
	case dtypes.Int64:
		return c.i64[i]
	case dtypes.Float64:
		return c.f64[i]
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		return c.str[i]
	case dtypes.Boolean:
		return c.bln[i]
	case dtypes.Datetime:
		return c.tim[i]
	default:
		return c.boxed[i]
	}
}

// Float64s returns the typed backing for Float64 columns.
func (c *Column) Float64s() ([]float64, bool) {
	if c.dtype != dtypes.Float64 {
		return nil, false
	}
	return c.f64, true
}

// Int64s returns the typed backing for Int64 columns.
func (c *Column) Int64s() ([]int64, bool) {
	if c.dtype != dtypes.Int64 {
		return nil, false
	}
	return c.i64, true
}

// Strings returns the typed backing for string-backed columns.
func (c *Column) Strings() ([]string, bool) {
	switch c.dtype {
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		return c.str, true
	default:
		return nil, false
	}
}

// Bools returns the typed backing for Boolean columns.
func (c *Column) Bools() ([]bool, bool) {
	if c.dtype != dtypes.Boolean {
		return nil, false
	}
	return c.bln, true
}

// Times returns the typed backing for Datetime columns.
func (c *Column) Times() ([]time.Time, bool) {
	if c.dtype != dtypes.Datetime {
		return nil, false
	}
	return c.tim, true
}

// Slice gathers rows at the given indices into a new Column, preserving order.
// All indices must be in [0, Len); for outer-join null-fill (-1 sentinels) use
// Gather instead.
func (c *Column) Slice(indices []int) *Column {
	return gatherTyped(c, indices, false)
}

// indexInteger is the set of index integer widths the gather kernels accept.
// int is the canonical row index; int32 halves the index memory for wide joins,
// where the (leftIdx, rightIdx) pair buffers dominate the join's working set.
type indexInteger interface{ ~int | ~int32 }

// gatherSlice copies src[indices[d]] into dst[d] for every d. When needNulls is
// set it also gathers validity: a -1 index or a null source row marks the
// output row null (and leaves the typed value at its zero). When needNulls is
// clear the tight value loop runs with no per-row branching — the case that
// dominates null-free numeric gathers.
func gatherSlice[T any, I indexInteger](dst, src []T, indices []I, needNulls bool, outNulls, srcNulls []bool) {
	if !needNulls {
		for d, s := range indices {
			dst[d] = src[s]
		}
		return
	}
	for d, s := range indices {
		if s < 0 || (srcNulls != nil && srcNulls[s]) {
			outNulls[d] = true
			continue
		}
		dst[d] = src[s]
	}
}

// gatherTyped builds a new Column by gathering rows at the given indices,
// preserving order. It switches on dtype once and runs a tight typed loop,
// rather than the per-row dtype switch that copyRow performs.
//
// When allowNullFill is true an index of -1 yields a null row (outer-join
// null-fill); when false every index must be in range. If the source carries no
// nulls and no null-fill is needed, the output validity slice is left nil — no
// []bool is allocated and no per-row null bookkeeping runs — and the null count
// is recorded as a known zero. This is the common case for filter/sort/unique
// materialization over null-free columns.
//
// indices is generic over int / int32 so a join can gather over int32 pair
// buffers (half the memory of []int) without first widening them to []int.
func gatherTyped[I indexInteger](c *Column, indices []I, allowNullFill bool) *Column {
	n := len(indices)
	out := &Column{dtype: c.dtype, n: n, nullCount: unknownNullCount}

	// The gather can only introduce a null when the source actually has null
	// rows or when a -1 sentinel is present. NullCount() is cached, so a
	// null-free column (including one carrying an all-false validity slice from
	// construction) takes the no-validity fast path. Probe for a sentinel only
	// when null-fill is allowed and the source is otherwise null-free, so the
	// common in-range gather stays a single pass.
	needNulls := c.NullCount() != 0
	if allowNullFill && !needNulls {
		for _, s := range indices {
			if s < 0 {
				needNulls = true
				break
			}
		}
	}
	if needNulls {
		out.nulls = make([]bool, n)
	} else {
		out.nullCount = 0
	}

	switch c.dtype {
	case dtypes.Int64:
		out.i64 = make([]int64, n)
		gatherSlice(out.i64, c.i64, indices, needNulls, out.nulls, c.nulls)
	case dtypes.Float64:
		out.f64 = make([]float64, n)
		gatherSlice(out.f64, c.f64, indices, needNulls, out.nulls, c.nulls)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str = make([]string, n)
		gatherSlice(out.str, c.str, indices, needNulls, out.nulls, c.nulls)
	case dtypes.Boolean:
		out.bln = make([]bool, n)
		gatherSlice(out.bln, c.bln, indices, needNulls, out.nulls, c.nulls)
	case dtypes.Datetime:
		out.tim = make([]time.Time, n)
		gatherSlice(out.tim, c.tim, indices, needNulls, out.nulls, c.nulls)
	default:
		out.boxed = make([]any, n)
		gatherSlice(out.boxed, c.boxed, indices, needNulls, out.nulls, c.nulls)
	}
	return out
}

// View returns a Column over rows [start,end) that SHARES this column's backing
// arrays (no copy is made). It is intended for read-only, range-parallel
// evaluation (e.g. a parallel Filter splitting a column into worker chunks):
// callers must treat the view — and must not mutate the source — for the view's
// lifetime. start and end must satisfy 0 <= start <= end <= c.Len().
func (c *Column) View(start, end int) *Column {
	out := &Column{dtype: c.dtype, n: end - start, nullCount: unknownNullCount}
	if c.nulls != nil {
		out.nulls = c.nulls[start:end]
	}
	switch c.dtype {
	case dtypes.Int64:
		out.i64 = c.i64[start:end]
	case dtypes.Float64:
		out.f64 = c.f64[start:end]
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str = c.str[start:end]
	case dtypes.Boolean:
		out.bln = c.bln[start:end]
	case dtypes.Datetime:
		out.tim = c.tim[start:end]
	default:
		out.boxed = c.boxed[start:end]
	}
	return out
}

// Filter keeps rows where mask[i] is true, producing a new Column.
func (c *Column) Filter(mask []bool) *Column {
	keep := make([]int, 0, len(mask))
	for i, m := range mask {
		if m {
			keep = append(keep, i)
		}
	}
	return c.Slice(keep)
}

// Shift moves values by periods rows, filling vacated positions with nulls.
func (c *Column) Shift(periods int) *Column {
	if periods == 0 {
		return c.Clone()
	}
	out := &Column{dtype: c.dtype, n: c.n, nulls: make([]bool, c.n), nullCount: unknownNullCount}
	switch c.dtype {
	case dtypes.Int64:
		out.i64 = make([]int64, c.n)
	case dtypes.Float64:
		out.f64 = make([]float64, c.n)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str = make([]string, c.n)
	case dtypes.Boolean:
		out.bln = make([]bool, c.n)
	case dtypes.Datetime:
		out.tim = make([]time.Time, c.n)
	default:
		out.boxed = make([]any, c.n)
	}
	for i := range out.nulls {
		out.nulls[i] = true
	}
	if periods > 0 {
		for src := 0; src < c.n-periods; src++ {
			c.copyRow(out, src+periods, src)
		}
	} else {
		abs := -periods
		for src := abs; src < c.n; src++ {
			c.copyRow(out, src-abs, src)
		}
	}
	return out
}

// Clone returns a deep copy of the column. The copy is private (not shared) and
// its cached null count is left unknown.
func (c *Column) Clone() *Column {
	out := &Column{dtype: c.dtype, n: c.n, nullCount: unknownNullCount}
	if c.nulls != nil {
		out.nulls = make([]bool, len(c.nulls))
		copy(out.nulls, c.nulls)
	}
	if c.f64 != nil {
		out.f64 = append([]float64(nil), c.f64...)
	}
	if c.i64 != nil {
		out.i64 = append([]int64(nil), c.i64...)
	}
	if c.str != nil {
		out.str = append([]string(nil), c.str...)
	}
	if c.bln != nil {
		out.bln = append([]bool(nil), c.bln...)
	}
	if c.tim != nil {
		out.tim = append([]time.Time(nil), c.tim...)
	}
	if c.boxed != nil {
		out.boxed = append([]any(nil), c.boxed...)
	}
	return out
}

// ConcatColumns appends multiple same-dtype columns into one without per-element
// boxing. Columns must all share the same dtype. Returns an empty column when
// cols is empty.
func ConcatColumns(cols []*Column) *Column {
	if len(cols) == 0 {
		return &Column{nullCount: unknownNullCount}
	}
	dtype := cols[0].dtype
	total := 0
	for _, c := range cols {
		total += c.n
	}
	nulls := make([]bool, total)
	off := 0
	for _, c := range cols {
		if c.nulls != nil {
			copy(nulls[off:], c.nulls)
		}
		off += c.n
	}
	switch dtype {
	case dtypes.Float64:
		buf := make([]float64, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.f64)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, f64: buf, nulls: nulls, nullCount: unknownNullCount}
	case dtypes.Int64:
		buf := make([]int64, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.i64)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, i64: buf, nulls: nulls, nullCount: unknownNullCount}
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		buf := make([]string, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.str)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, str: buf, nulls: nulls, nullCount: unknownNullCount}
	case dtypes.Boolean:
		buf := make([]bool, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.bln)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, bln: buf, nulls: nulls, nullCount: unknownNullCount}
	case dtypes.Datetime:
		buf := make([]time.Time, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.tim)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, tim: buf, nulls: nulls, nullCount: unknownNullCount}
	default:
		buf := make([]any, total)
		off = 0
		for _, c := range cols {
			copy(buf[off:], c.boxed)
			off += c.n
		}
		return &Column{dtype: dtype, n: total, boxed: buf, nulls: nulls, nullCount: unknownNullCount}
	}
}

// copyRow copies row src of c into row dst of out (same dtype), including null.
func (c *Column) copyRow(out *Column, dst, src int) {
	if c.nulls != nil && c.nulls[src] {
		out.nulls[dst] = true
		return
	}
	out.nulls[dst] = false
	switch c.dtype {
	case dtypes.Int64:
		out.i64[dst] = c.i64[src]
	case dtypes.Float64:
		out.f64[dst] = c.f64[src]
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str[dst] = c.str[src]
	case dtypes.Boolean:
		out.bln[dst] = c.bln[src]
	case dtypes.Datetime:
		out.tim[dst] = c.tim[src]
	default:
		out.boxed[dst] = c.boxed[src]
	}
}
