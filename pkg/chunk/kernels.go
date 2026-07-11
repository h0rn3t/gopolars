package chunk

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// parallelFillThreshold is the row count above which the element-wise float64
// fill/drop kernels split work across GOMAXPROCS workers. Below it the
// goroutine-coordination overhead outweighs the gain, so the sequential path
// (a single shard) runs inline.
const parallelFillThreshold = 1 << 13 // 8192 rows

// forEachShard splits [0,n) into up to GOMAXPROCS disjoint contiguous ranges and
// runs fn on each concurrently, blocking until all complete. For n at or below
// parallelFillThreshold (or a single worker) it runs fn(0,n) inline.
func forEachShard(n int, fn func(lo, hi int)) {
	if n == 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if n <= parallelFillThreshold || workers <= 1 {
		fn(0, n)
		return
	}
	if workers > n {
		workers = n
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(lo, hi int) {
			defer wg.Done()
			fn(lo, hi)
		}(w*n/workers, (w+1)*n/workers)
	}
	wg.Wait()
}

// shardBounds returns workers+1 boundaries splitting [0,n) into balanced disjoint
// ranges, collapsing to a single shard at or below parallelFillThreshold.
func shardBounds(n int) []int {
	workers := runtime.GOMAXPROCS(0)
	if n <= parallelFillThreshold || workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}
	bounds := make([]int, workers+1)
	for w := 0; w <= workers; w++ {
		bounds[w] = w * n / workers
	}
	return bounds
}

// forEachBound runs fn(workerIndex, lo, hi) concurrently for each consecutive
// pair in bounds (length workers+1), blocking until all complete. With a single
// worker it runs inline.
func forEachBound(bounds []int, fn func(w, lo, hi int)) {
	workers := len(bounds) - 1
	if workers <= 1 {
		if workers == 1 {
			fn(0, bounds[0], bounds[1])
		}
		return
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			fn(w, bounds[w], bounds[w+1])
		}(w)
	}
	wg.Wait()
}

// hasFillableNaNFloat64 reports whether any non-null entry is NaN — exactly the
// rows fill_nan replaces and drop_nans removes. It is read-only and sharded, so
// the common "no NaN present" case costs one parallel scan rather than a full
// allocate-and-copy.
func hasFillableNaNFloat64(f64s []float64, nulls []bool) bool {
	var found int32
	forEachShard(len(f64s), func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if math.IsNaN(f64s[i]) && (nulls == nil || !nulls[i]) {
				atomic.StoreInt32(&found, 1)
				return
			}
		}
	})
	return atomic.LoadInt32(&found) != 0
}

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
	return gatherTyped(c, indices, true)
}

// GatherInt32 is Gather over an int32 index buffer. Joins keep their
// (leftIdx, rightIdx) pair buffers as int32 — half the memory of []int at the
// 1M-row reference scale — and gather output columns directly from them without
// first widening to []int. An index of -1 yields a null row, as in Gather.
func (c *Column) GatherInt32(indices []int32) *Column {
	return gatherTyped(c, indices, true)
}

// FillNullFloat64 returns a new Float64 column with every null entry replaced by
// fill; filled rows become non-null (matching fill_null semantics). The second
// result is false when the column is not Float64.
func (c *Column) FillNullFloat64(fill float64) (*Column, bool) {
	f64s, ok := c.Float64s()
	if !ok {
		return nil, false
	}
	// No nulls to fill: the result equals the input, so share its value buffer
	// instead of allocating and copying a fresh 8N-byte one. Copy-on-write
	// (MarkShared / CloneIfShared) keeps any future in-place mutator safe.
	if c.NullCount() == 0 {
		c.MarkShared()
		out := NewFloat64(f64s, nil)
		out.MarkShared()
		return out, true
	}
	nulls := c.nulls // non-nil here: NullCount() > 0 implies a validity mask
	out := make([]float64, c.n)
	forEachShard(c.n, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if nulls[i] {
				out[i] = fill
			} else {
				out[i] = f64s[i]
			}
		}
	})
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
	nulls := c.nulls
	// No non-null NaN present: the result equals the input, so share both buffers
	// instead of copying. The validity mask is preserved (NaN is a value, not a
	// null) by reusing c.nulls.
	if !hasFillableNaNFloat64(f64s, nulls) {
		c.MarkShared()
		out := NewFloat64(f64s, nulls)
		out.MarkShared()
		return out, true
	}
	out := make([]float64, c.n)
	var outNulls []bool
	if nulls != nil {
		outNulls = make([]bool, c.n)
	}
	forEachShard(c.n, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			v := f64s[i]
			if nulls != nil {
				outNulls[i] = nulls[i]
				if nulls[i] {
					out[i] = v // null slot: keep its payload, validity preserved
					continue
				}
			}
			if math.IsNaN(v) {
				v = fill
			}
			out[i] = v
		}
	})
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
	nulls := c.nulls
	// Nothing to drop (no non-null NaN): share the input buffers unchanged.
	if !hasFillableNaNFloat64(f64s, nulls) {
		c.MarkShared()
		out := NewFloat64(f64s, nulls)
		out.MarkShared()
		return out, true
	}
	// Two-pass count -> scatter so survivors keep their original order under
	// parallel execution: a row is kept unless it is a non-null NaN.
	keep := func(i int) bool {
		if nulls != nil && nulls[i] {
			return true
		}
		return !math.IsNaN(f64s[i])
	}
	bounds := shardBounds(c.n)
	workers := len(bounds) - 1
	// Pass 1: per-shard survivor count.
	counts := make([]int, workers)
	forEachBound(bounds, func(w, lo, hi int) {
		cnt := 0
		for i := lo; i < hi; i++ {
			if keep(i) {
				cnt++
			}
		}
		counts[w] = cnt
	})
	// Exclusive prefix offsets + total survivor count.
	offsets := make([]int, workers)
	total := 0
	for w := 0; w < workers; w++ {
		offsets[w] = total
		total += counts[w]
	}
	out := make([]float64, total)
	var outNulls []bool
	if nulls != nil {
		outNulls = make([]bool, total)
	}
	// Pass 2: each shard scatters its survivors into out[offset:] in order.
	forEachBound(bounds, func(w, lo, hi int) {
		pos := offsets[w]
		for i := lo; i < hi; i++ {
			if !keep(i) {
				continue
			}
			out[pos] = f64s[i]
			if outNulls != nil {
				outNulls[pos] = nulls[i]
			}
			pos++
		}
	})
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

// FirstRows returns the first-seen row index of each distinct key across the
// given key columns, in encounter order — exactly the rows unique() keeps —
// without allocating the length-N ids array GroupIDs builds. Null entries in a
// key form a single group (all nulls compare equal), matching GroupIDs. Use this
// for Unique/NUnique, which only need firstRow; keep GroupIDs for GroupBy, which
// needs the per-row ids to bucket rows into groups.
func FirstRows(cols []*Column, n int) []int {
	if len(cols) == 1 {
		if firstRow, ok := firstRowsSingle(cols[0], n); ok {
			return firstRow
		}
	}
	var firstRow []int
	idMap := make(map[string]struct{})
	var scratch []byte
	for row := range n {
		scratch = scratch[:0]
		for _, c := range cols {
			scratch = appendRowKey(scratch, c, row)
		}
		if _, ok := idMap[string(scratch)]; !ok {
			idMap[string(scratch)] = struct{}{}
			firstRow = append(firstRow, row)
		}
	}
	return firstRow
}

// firstRowsSingle is the per-dtype typed fast path for a single key column,
// mirroring groupIDsSingle but recording only the first-seen row per key. ok is
// false for boxed dtypes, which fall back to the composite encoder.
func firstRowsSingle(c *Column, n int) (firstRow []int, ok bool) {
	nulls := c.nulls
	nullSeen := false
	assignNull := func(row int) {
		if !nullSeen {
			nullSeen = true
			firstRow = append(firstRow, row)
		}
	}
	switch c.dtype {
	case dtypes.Int64:
		m := make(map[int64]struct{})
		for row := range n {
			if nulls != nil && nulls[row] {
				assignNull(row)
				continue
			}
			v := c.i64[row]
			if _, seen := m[v]; !seen {
				m[v] = struct{}{}
				firstRow = append(firstRow, row)
			}
		}
	case dtypes.Float64:
		m := make(map[uint64]struct{})
		for row := range n {
			if nulls != nil && nulls[row] {
				assignNull(row)
				continue
			}
			v := c.f64[row]
			bits := math.Float64bits(v)
			if math.IsNaN(v) {
				bits = canonicalNaNBits
			}
			if _, seen := m[bits]; !seen {
				m[bits] = struct{}{}
				firstRow = append(firstRow, row)
			}
		}
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		m := make(map[string]struct{})
		for row := range n {
			if nulls != nil && nulls[row] {
				assignNull(row)
				continue
			}
			v := c.str[row]
			if _, seen := m[v]; !seen {
				m[v] = struct{}{}
				firstRow = append(firstRow, row)
			}
		}
	case dtypes.Boolean:
		m := make(map[bool]struct{})
		for row := range n {
			if nulls != nil && nulls[row] {
				assignNull(row)
				continue
			}
			v := c.bln[row]
			if _, seen := m[v]; !seen {
				m[v] = struct{}{}
				firstRow = append(firstRow, row)
			}
		}
	case dtypes.Datetime:
		m := make(map[int64]struct{})
		for row := range n {
			if nulls != nil && nulls[row] {
				assignNull(row)
				continue
			}
			v := c.tim[row].UnixNano()
			if _, seen := m[v]; !seen {
				m[v] = struct{}{}
				firstRow = append(firstRow, row)
			}
		}
	default:
		return nil, false
	}
	return firstRow, true
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

// CanPackJoinKey reports whether c's dtype packs losslessly into a uint64, so an
// equi-join can key a map[uint64] directly instead of allocating a byte-encoded
// Go string per row. Int64, Float64, Boolean, and Datetime each fit a 64-bit
// slot bijectively (with NaN canonicalized); String/Categorical/Enum and boxed
// dtypes do not and must use the AppendRowKey byte fallback.
func CanPackJoinKey(c *Column) bool {
	if c == nil {
		return false
	}
	switch c.dtype {
	case dtypes.Int64, dtypes.Float64, dtypes.Boolean, dtypes.Datetime:
		return true
	default:
		return false
	}
}

// PackKeyFunc returns a closure that maps a row index to a uint64 encoding of a
// single fixed-width key column's value — an exact (collision-free) join-table
// key — with the per-dtype switch hoisted out of the row loop (the closure
// captures the typed backing slice). This avoids materializing a whole-column
// []uint64 key buffer (O(rows) memory) just to pack keys the build/probe read
// once. ok is false for dtypes that do not pack losslessly (see CanPackJoinKey),
// in which case the caller uses the byte-encoded AppendRowKey fallback. Null rows
// are NOT distinguished here: callers detect nulls via c.Nulls() and key them
// separately, exactly as the byte path's null tag keeps null keys apart from
// every real value. NaN is canonicalized so all NaN payloads pack equal,
// matching appendRowKey.
func PackKeyFunc(c *Column) (keyAt func(int) uint64, ok bool) {
	switch c.dtype {
	case dtypes.Int64:
		v := c.i64
		return func(i int) uint64 { return uint64(v[i]) }, true
	case dtypes.Float64:
		v := c.f64
		return func(i int) uint64 {
			bits := math.Float64bits(v[i])
			if math.IsNaN(v[i]) {
				bits = canonicalNaNBits
			}
			return bits
		}, true
	case dtypes.Boolean:
		v := c.bln
		return func(i int) uint64 {
			if v[i] {
				return 1
			}
			return 0
		}, true
	case dtypes.Datetime:
		v := c.tim
		return func(i int) uint64 { return uint64(v[i].UnixNano()) }, true
	default:
		return nil, false
	}
}
