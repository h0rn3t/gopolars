package chunk

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// stringifyBoxed renders a boxed value for use in a composite group key. Used
// only for non-primitive dtypes (Decimal/List/Struct) that have no typed
// backing slice.
func stringifyBoxed(v any) string { return fmt.Sprintf("%v", v) }

// NullCount returns the number of null rows. It computes the count lazily on the
// first request and caches it, returning the cached value in constant time
// thereafter. The cache is reset to the unknown sentinel at every site that
// produces or mutates a column's validity, so a recomputation always equals a
// full validity scan. Safe for concurrent callers on a shared column.
func (c *Column) NullCount() int {
	if c == nil {
		return 0
	}
	if v := atomic.LoadInt64(&c.nullCount); v != unknownNullCount {
		return int(v)
	}
	count := int64(0)
	for _, isNull := range c.nulls {
		if isNull {
			count++
		}
	}
	atomic.StoreInt64(&c.nullCount, count)
	return int(count)
}

// MarkShared records that the column may be referenced by more than one frame.
// A shared column is treated as immutable: any in-place mutator must clone it
// first via CloneIfShared. MarkShared is idempotent.
func (c *Column) MarkShared() {
	if c != nil {
		c.shared = true
	}
}

// IsShared reports whether the column has been marked as possibly shared.
func (c *Column) IsShared() bool { return c != nil && c.shared }

// CloneIfShared returns a private clone when the column is shared, otherwise the
// receiver. In-place mutators MUST route writes through this so a mutation never
// leaks across frames that share the buffer. (No in-place mutator exists today;
// this is the copy-on-write entry point that any future one must use.)
func (c *Column) CloneIfShared() *Column {
	if c != nil && c.shared {
		return c.Clone()
	}
	return c
}

// Gather builds a new Column from the rows at the given indices, preserving
// order. An index of -1 yields a null row (used to null-fill unmatched rows in
// outer joins); all other indices must be in [0, Len). The source column is not
// modified.
func (c *Column) Gather(indices []int) *Column {
	out := &Column{dtype: c.dtype, n: len(indices), nulls: make([]bool, len(indices)), nullCount: unknownNullCount}
	switch c.dtype {
	case dtypes.Int64:
		out.i64 = make([]int64, len(indices))
	case dtypes.Float64:
		out.f64 = make([]float64, len(indices))
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		out.str = make([]string, len(indices))
	case dtypes.Boolean:
		out.bln = make([]bool, len(indices))
	case dtypes.Datetime:
		out.tim = make([]time.Time, len(indices))
	default:
		out.boxed = make([]any, len(indices))
	}
	for dst, src := range indices {
		if src < 0 {
			out.nulls[dst] = true
			continue
		}
		c.copyRow(out, dst, src)
	}
	return out
}

// FillNullFloat64 returns a new Float64 column with every null entry replaced by
// fill; filled rows become non-null (matching fill_null semantics). The second
// result is false when the column is not Float64.
func (c *Column) FillNullFloat64(fill float64) (*Column, bool) {
	f64s, ok := c.Float64s()
	if !ok {
		return nil, false
	}
	out := make([]float64, c.n)
	copy(out, f64s)
	if c.nulls != nil {
		for i, isNull := range c.nulls {
			if isNull {
				out[i] = fill
			}
		}
	}
	// All nulls have been filled, so the result has no null entries.
	return NewFloat64(out, nil), true
}

// FillNaNFloat64 returns a new Float64 column with every non-null NaN value
// replaced by fill, preserving the validity mask (NaN is a value, not a null).
// The second result is false when the column is not Float64.
func (c *Column) FillNaNFloat64(fill float64) (*Column, bool) {
	f64s, ok := c.Float64s()
	if !ok {
		return nil, false
	}
	out := make([]float64, c.n)
	copy(out, f64s)
	nulls := c.nulls
	for i := range out {
		if nulls != nil && nulls[i] {
			continue
		}
		if math.IsNaN(out[i]) {
			out[i] = fill
		}
	}
	var outNulls []bool
	if nulls != nil {
		outNulls = make([]bool, c.n)
		copy(outNulls, nulls)
	}
	return NewFloat64(out, outNulls), true
}

// DropNaNFloat64 returns a new Float64 column with every non-null NaN row
// removed (matching drop_nans). The second result is false when the column is
// not Float64.
func (c *Column) DropNaNFloat64() (*Column, bool) {
	f64s, ok := c.Float64s()
	if !ok {
		return nil, false
	}
	out := make([]float64, 0, c.n)
	var outNulls []bool
	if c.nulls != nil {
		outNulls = make([]bool, 0, c.n)
	}
	for i, v := range f64s {
		isNull := c.nulls != nil && c.nulls[i]
		if !isNull && math.IsNaN(v) {
			continue
		}
		out = append(out, v)
		if outNulls != nil {
			outNulls = append(outNulls, isNull)
		}
	}
	return NewFloat64(out, outNulls), true
}

// canonicalNaNBits is the single bit pattern all NaNs collapse to when used as a
// group key, so that distinct NaN payloads group together (matching the %v/%g
// row-wise key which renders every NaN as "NaN").
const canonicalNaNBits = uint64(0x7ff8000000000000)

// GroupIDs assigns each of the first n rows a dense group id derived from the
// typed values of the given key columns, with no per-row interface boxing or
// fmt.Sprintf. ids[row] is the group id in [0, len(firstRow)); firstRow[g] is
// the first row index observed for group g, in encounter order. Null entries in
// a key form their own group value (all nulls in a column compare equal).
//
// Single primitive key columns hash a typed map keyed on the unboxed value;
// multi-key and boxed-dtype columns fold a typed byte encoding into a composite
// key, allocating per distinct group rather than per row.
func GroupIDs(cols []*Column, n int) (ids []int, firstRow []int) {
	if len(cols) == 1 {
		if ids, firstRow, ok := groupIDsSingle(cols[0], n); ok {
			return ids, firstRow
		}
	}
	ids = make([]int, n)
	idMap := make(map[string]int)
	var scratch []byte
	next := 0
	for row := range n {
		scratch = scratch[:0]
		for _, c := range cols {
			scratch = appendRowKey(scratch, c, row)
		}
		g, ok := idMap[string(scratch)]
		if !ok {
			g = next
			next++
			idMap[string(scratch)] = g
			firstRow = append(firstRow, row)
		}
		ids[row] = g
	}
	return ids, firstRow
}

// groupIDsSingle is the per-dtype typed fast path for a single key column. ok is
// false for boxed dtypes, which fall back to the composite encoder.
func groupIDsSingle(c *Column, n int) (ids []int, firstRow []int, ok bool) {
	ids = make([]int, n)
	nulls := c.nulls
	nullID := -1
	next := 0
	assignNull := func(row int) int {
		if nullID == -1 {
			nullID = next
			next++
			firstRow = append(firstRow, row)
		}
		return nullID
	}
	switch c.dtype {
	case dtypes.Int64:
		m := make(map[int64]int)
		for row := range n {
			if nulls != nil && nulls[row] {
				ids[row] = assignNull(row)
				continue
			}
			v := c.i64[row]
			g, seen := m[v]
			if !seen {
				g = next
				next++
				m[v] = g
				firstRow = append(firstRow, row)
			}
			ids[row] = g
		}
	case dtypes.Float64:
		m := make(map[uint64]int)
		for row := range n {
			if nulls != nil && nulls[row] {
				ids[row] = assignNull(row)
				continue
			}
			v := c.f64[row]
			bits := math.Float64bits(v)
			if math.IsNaN(v) {
				bits = canonicalNaNBits
			}
			g, seen := m[bits]
			if !seen {
				g = next
				next++
				m[bits] = g
				firstRow = append(firstRow, row)
			}
			ids[row] = g
		}
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		m := make(map[string]int)
		for row := range n {
			if nulls != nil && nulls[row] {
				ids[row] = assignNull(row)
				continue
			}
			v := c.str[row]
			g, seen := m[v]
			if !seen {
				g = next
				next++
				m[v] = g
				firstRow = append(firstRow, row)
			}
			ids[row] = g
		}
	case dtypes.Boolean:
		m := make(map[bool]int)
		for row := range n {
			if nulls != nil && nulls[row] {
				ids[row] = assignNull(row)
				continue
			}
			v := c.bln[row]
			g, seen := m[v]
			if !seen {
				g = next
				next++
				m[v] = g
				firstRow = append(firstRow, row)
			}
			ids[row] = g
		}
	case dtypes.Datetime:
		m := make(map[int64]int)
		for row := range n {
			if nulls != nil && nulls[row] {
				ids[row] = assignNull(row)
				continue
			}
			v := c.tim[row].UnixNano()
			g, seen := m[v]
			if !seen {
				g = next
				next++
				m[v] = g
				firstRow = append(firstRow, row)
			}
			ids[row] = g
		}
	default:
		return nil, nil, false
	}
	return ids, firstRow, true
}

// AppendRowKey appends a typed, collision-resistant encoding of the values at
// `row` across the given columns to dst and returns the extended slice. Reusing
// a scratch buffer lets callers build join/group keys without per-row
// fmt.Sprintf or interface boxing.
func AppendRowKey(dst []byte, cols []*Column, row int) []byte {
	for _, c := range cols {
		dst = appendRowKey(dst, c, row)
	}
	return dst
}

// appendRowKey appends a typed, collision-resistant encoding of c's value at
// row to dst and returns the extended slice. A leading tag byte distinguishes
// null from each dtype so values never alias across columns or types.
func appendRowKey(dst []byte, c *Column, row int) []byte {
	if c.nulls != nil && c.nulls[row] {
		return append(dst, 0) // null tag
	}
	switch c.dtype {
	case dtypes.Int64:
		dst = append(dst, 1)
		return appendUint64(dst, uint64(c.i64[row]))
	case dtypes.Float64:
		dst = append(dst, 2)
		bits := math.Float64bits(c.f64[row])
		if math.IsNaN(c.f64[row]) {
			bits = canonicalNaNBits
		}
		return appendUint64(dst, bits)
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		dst = append(dst, 3)
		s := c.str[row]
		dst = appendUint64(dst, uint64(len(s)))
		return append(dst, s...)
	case dtypes.Boolean:
		if c.bln[row] {
			return append(dst, 4, 1)
		}
		return append(dst, 4, 0)
	case dtypes.Datetime:
		dst = append(dst, 5)
		return appendUint64(dst, uint64(c.tim[row].UnixNano()))
	default:
		// Boxed dtypes (Decimal/List/Struct) are rare keys; encode via %v.
		dst = append(dst, 6)
		return append(dst, []byte(stringifyBoxed(c.boxed[row]))...)
	}
}

func appendUint64(dst []byte, v uint64) []byte {
	return append(dst,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}
