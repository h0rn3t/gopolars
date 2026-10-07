package exec

import (
	"math"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func mustFrame(t *testing.T, cols ...frame.SeriesInput) frame.DataFrame {
	t.Helper()
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return df
}

// TestSplitFrames covers the chunking helper, including the empty-frame and
// uneven-final-chunk cases.
func TestSplitFrames(t *testing.T) {
	t.Parallel()

	df := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}})

	parts := splitFrames(df, 2)
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3", len(parts))
	}
	if parts[0].Height() != 2 || parts[1].Height() != 2 || parts[2].Height() != 1 {
		t.Fatalf("part heights = %d/%d/%d, want 2/2/1", parts[0].Height(), parts[1].Height(), parts[2].Height())
	}

	// Empty source returns a single (empty) part.
	empty := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{}, DType: dtypes.Int64})
	emptyParts := splitFrames(empty, 4)
	if len(emptyParts) != 1 {
		t.Fatalf("empty parts = %d, want 1", len(emptyParts))
	}
}

// TestConcatFrames covers the 0-, 1-, and many-part branches and verifies a
// split followed by concat reconstructs the original.
func TestConcatFrames(t *testing.T) {
	t.Parallel()

	if df, err := concatFrames(nil); err != nil || df.Height() != 0 {
		t.Fatalf("concat(nil) = height %d err=%v", df.Height(), err)
	}

	df := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}})
	single := splitFrames(df, 100)
	if got, err := concatFrames(single); err != nil || got.Height() != 5 {
		t.Fatalf("concat(single) = height %d err=%v", got.Height(), err)
	}

	merged, err := concatFrames(splitFrames(df, 2))
	if err != nil {
		t.Fatalf("concat: %v", err)
	}
	eq, err := df.Equals(merged)
	if err != nil || !eq {
		t.Fatalf("split+concat != original (eq=%v err=%v)", eq, err)
	}
}

// TestUnionFrames covers union (dedup) and union-all (keep duplicates).
func TestUnionFrames(t *testing.T) {
	t.Parallel()

	left := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}})
	right := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(2), int64(3)}})

	all, err := unionFrames(left, right, true)
	if err != nil || all.Height() != 4 {
		t.Fatalf("union all height = %d err=%v, want 4", all.Height(), err)
	}

	deduped, err := unionFrames(left, right, false)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	if deduped.Height() != 3 {
		t.Fatalf("union dedup height = %d, want 3", deduped.Height())
	}
}

// TestExtractGlobalLimit pulls the smallest limit of the trailing limit run out
// of the pipeline and keeps every node before that run, earlier limits included.
func TestExtractGlobalLimit(t *testing.T) {
	t.Parallel()

	nodes := []logical.Node{
		{Type: logical.NodeFilter},
		{Type: logical.NodeLimit, IntValue: 10},
		{Type: logical.NodeSelect},
		{Type: logical.NodeLimit, IntValue: 4},
		{Type: logical.NodeLimit, IntValue: 6},
	}
	remaining, limit := extractGlobalLimit(nodes)
	if limit != 4 {
		t.Fatalf("limit = %d, want 4 (smallest trailing)", limit)
	}
	var types []logical.NodeType
	for _, n := range remaining {
		types = append(types, n.Type)
	}
	if want := []logical.NodeType{logical.NodeFilter, logical.NodeLimit, logical.NodeSelect}; !slices.Equal(types, want) {
		t.Fatalf("extractGlobalLimit remaining = %v, want %v", types, want)
	}

	// No limit -> sentinel -1.
	_, none := extractGlobalLimit([]logical.Node{{Type: logical.NodeFilter}})
	if none != -1 {
		t.Fatalf("no-limit sentinel = %d, want -1", none)
	}
}

// TestAppendSeries covers both appending a new column and replacing an existing
// one by name.
func TestAppendSeries(t *testing.T) {
	t.Parallel()

	df := mustFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}})

	newCol, err := series.New("b", dtypes.Int64, []any{int64(10), int64(20)})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	appended, err := appendSeries(df, newCol)
	if err != nil || appended.Width() != 2 {
		t.Fatalf("append width = %d err=%v, want 2", appended.Width(), err)
	}

	replacement, err := series.New("a", dtypes.Int64, []any{int64(7), int64(8)})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	replaced, err := appendSeries(df, replacement)
	if err != nil || replaced.Width() != 1 {
		t.Fatalf("replace width = %d err=%v, want 1", replaced.Width(), err)
	}
	col, _ := replaced.Series("a")
	if col.Value(0) != int64(7) {
		t.Fatalf("replaced value = %v, want 7", col.Value(0))
	}
}

// TestStreamable covers which plans may run chunk by chunk: row-local nodes
// with elementwise expressions, followed only by limits.
func TestStreamable(t *testing.T) {
	t.Parallel()

	a := expr.Col("a")
	aGt1 := []expr.Expr{a.Gt(expr.Lit(int64(1)))}
	tests := []struct {
		name  string
		nodes []logical.Node
		want  bool
	}{
		{"empty plan", nil, true},
		{"filter and select", []logical.Node{
			{Type: logical.NodeFilter, Exprs: aGt1},
			{Type: logical.NodeSelect, Exprs: []expr.Expr{a.Mul(expr.Lit(int64(2))).Alias("d")}},
		}, true},
		{"every row-local node", []logical.Node{
			{Type: logical.NodeScan},
			{Type: logical.NodeFilter, Exprs: aGt1},
			{Type: logical.NodeSelect, Exprs: []expr.Expr{a, expr.Col("s")}},
			{Type: logical.NodeWithCols, Exprs: []expr.Expr{a.Abs().Alias("b")}},
			{Type: logical.NodeRename, Strings: []string{"b", "c"}},
			{Type: logical.NodeDrop, Columns: []string{"c"}},
			{Type: logical.NodeCast, Strings: []string{"a", "float64"}},
			{Type: logical.NodeFillNull, Exprs: []expr.Expr{expr.Lit(0.0)}},
			{Type: logical.NodeFillNaN, Strings: []string{"0"}},
			{Type: logical.NodeDropNulls},
			{Type: logical.NodeDropNans},
			{Type: logical.NodeExplode, Columns: []string{"a"}},
			{Type: logical.NodeFlatten, Columns: []string{"s"}},
			{Type: logical.NodeUnnest, Columns: []string{"s"}},
		}, true},
		{"trailing limits", []logical.Node{
			{Type: logical.NodeFilter, Exprs: aGt1},
			{Type: logical.NodeLimit, IntValue: 3},
			{Type: logical.NodeLimit, IntValue: 2},
		}, true},
		{"limit before filter", []logical.Node{
			{Type: logical.NodeLimit, IntValue: 3},
			{Type: logical.NodeFilter, Exprs: aGt1},
		}, false},
		{"cumulative expression", []logical.Node{{Type: logical.NodeWithCols, Exprs: []expr.Expr{a.CumSum()}}}, false},
		{"aggregate expression", []logical.Node{{Type: logical.NodeSelect, Exprs: []expr.Expr{a, expr.Sum(a)}}}, false},
		{"window expression in filter", []logical.Node{{Type: logical.NodeFilter, Exprs: []expr.Expr{a.CumCount().Over("g").Gt(expr.Lit(int64(1)))}}}, false},
		{"tail", []logical.Node{{Type: logical.NodeTail, IntValue: 2}}, false},
		{"row index", []logical.Node{{Type: logical.NodeWithRowIdx, Strings: []string{"i"}}}, false},
		{"slice", []logical.Node{{Type: logical.NodeSlice, IntValue: 1}}, false},
		{"gather every", []logical.Node{{Type: logical.NodeGatherEvery}}, false},
		{"reverse", []logical.Node{{Type: logical.NodeReverse}}, false},
		{"unique", []logical.Node{{Type: logical.NodeUnique}}, false},
		{"shift", []logical.Node{{Type: logical.NodeShift, IntValue: 1}}, false},
		{"interpolate", []logical.Node{{Type: logical.NodeInterpolate}}, false},
		{"update", []logical.Node{{Type: logical.NodeUpdate}}, false},
		{"melt", []logical.Node{{Type: logical.NodeMelt}}, false},
		{"frame aggregation", []logical.Node{{Type: logical.NodeFrameAgg, Strings: []string{"sum"}}}, false},
		{"sort", []logical.Node{{Type: logical.NodeSort}}, false},
		{"aggregate", []logical.Node{{Type: logical.NodeAggregate}}, false},
		{"join", []logical.Node{{Type: logical.NodeJoin}}, false},
		{"window", []logical.Node{{Type: logical.NodeWindow}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := streamable(tt.nodes); got != tt.want {
				t.Errorf("streamable(%s) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}

// TestAggregateFrame exercises every aggregation kind on a fixed numeric column
// with hand-checked expected values.
func TestAggregateFrame(t *testing.T) {
	t.Parallel()

	// values: 1,2,3,4 (+ one null) -> sum 10, mean 2.5, median 2.5, min 1, max 4
	df := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), nil}})

	cases := []struct {
		op   string
		args []string
		want any
	}{
		{"sum", nil, int64(10)},
		{"mean", nil, 2.5},
		{"median", nil, 2.5},
		{"min", nil, int64(1)},
		{"max", nil, int64(4)},
		{"count", nil, int64(4)},      // nulls excluded
		{"null_count", nil, int64(1)}, // one null
		{"quantile", []string{"0.5"}, 2.5},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			out, err := aggregateFrame(df, tc.op, tc.args)
			if err != nil {
				t.Fatalf("aggregate %s: %v", tc.op, err)
			}
			col, _ := out.Series("v")
			if col.Value(0) != tc.want {
				t.Fatalf("%s = %v (%T), want %v (%T)", tc.op, col.Value(0), col.Value(0), tc.want, tc.want)
			}
		})
	}

	// std / var: variance of {1,2,3,4} (population) = 1.25, std = sqrt(1.25).
	varOut, err := aggregateFrame(df, "var", nil)
	if err != nil {
		t.Fatalf("var: %v", err)
	}
	varCol, _ := varOut.Series("v")
	if math.Abs(varCol.Value(0).(float64)-1.25) > 1e-9 {
		t.Fatalf("var = %v, want 1.25", varCol.Value(0))
	}
	stdOut, err := aggregateFrame(df, "std", nil)
	if err != nil {
		t.Fatalf("std: %v", err)
	}
	stdCol, _ := stdOut.Series("v")
	if math.Abs(stdCol.Value(0).(float64)-math.Sqrt(1.25)) > 1e-9 {
		t.Fatalf("std = %v, want %v", stdCol.Value(0), math.Sqrt(1.25))
	}
}

// TestAggregateFrameNullsAndOrder pins aggregateFrame on all-null and
// non-numeric columns (every numeric aggregate is null) and on an
// order-sensitive float column (sums run in row order, quantile hits exact and
// interpolated ranks).
func TestAggregateFrameNullsAndOrder(t *testing.T) {
	t.Parallel()

	empty := mustFrame(t,
		frame.SeriesInput{Name: "n", DType: dtypes.Float64, Values: []any{nil, nil}},
		frame.SeriesInput{Name: "s", Values: []any{"b", "a"}},
	)
	for _, op := range []string{"mean", "median", "std", "var", "quantile"} {
		out, err := aggregateFrame(empty, op, []string{"0.5"})
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		for _, name := range []string{"n", "s"} {
			if col, _ := out.Series(name); col.Value(0) != nil || col.DataType() != dtypes.Float64 {
				t.Errorf("%s(%s) = %v (%s), want null float64", op, name, col.Value(0), col.DataType())
			}
		}
	}
	for _, tc := range []struct {
		op     string
		n, s   any
		nt, st dtypes.DataType
	}{
		{"max", nil, "b", dtypes.Float64, dtypes.String},
		{"min", nil, "a", dtypes.Float64, dtypes.String},
		{"count", int64(0), int64(2), dtypes.Int64, dtypes.Int64},
		{"null_count", int64(2), int64(0), dtypes.Int64, dtypes.Int64},
	} {
		out, err := aggregateFrame(empty, tc.op, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		n, _ := out.Series("n")
		s, _ := out.Series("s")
		if n.Value(0) != tc.n || s.Value(0) != tc.s || n.DataType() != tc.nt || s.DataType() != tc.st {
			t.Errorf("%s = (%v %s, %v %s), want (%v %s, %v %s)", tc.op, n.Value(0), n.DataType(), s.Value(0), s.DataType(), tc.n, tc.nt, tc.s, tc.st)
		}
	}

	// Leading null and ties for max/min.
	ties := mustFrame(t, frame.SeriesInput{Name: "v", Values: []any{nil, int64(3), int64(1), int64(3), int64(1)}})
	for op, want := range map[string]any{"max": int64(3), "min": int64(1)} {
		out, err := aggregateFrame(ties, op, nil)
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if col, _ := out.Series("v"); col.Value(0) != want {
			t.Errorf("%s = %v, want %v", op, col.Value(0), want)
		}
	}

	vals := []any{1e16, 1.0, nil, -1e16, 3.0, 0.1}
	df := mustFrame(t, frame.SeriesInput{Name: "f", Values: vals})
	var nums []float64
	for _, v := range vals {
		if f, ok := v.(float64); ok {
			nums = append(nums, f)
		}
	}
	var sum float64
	for _, f := range nums {
		sum += f
	}
	mean := sum / float64(len(nums))
	var variance float64
	for _, f := range nums {
		d := f - mean
		variance += d * d
	}
	variance /= float64(len(nums))
	sorted := slices.Clone(nums)
	slices.Sort(sorted)
	idx := float64(len(sorted)-1) * 0.3
	w := idx - math.Floor(idx)
	interpolated := sorted[1]*(1.0-w) + sorted[2]*w

	for _, tc := range []struct {
		op   string
		args []string
		want float64
	}{
		{"mean", nil, mean},
		{"var", nil, variance},
		{"std", nil, math.Sqrt(variance)},
		{"median", nil, sorted[2]},
		{"quantile", []string{"0.25"}, sorted[1]},
		{"quantile", []string{"1"}, sorted[4]},
		{"quantile", []string{"0.3"}, interpolated},
	} {
		out, err := aggregateFrame(df, tc.op, tc.args)
		if err != nil {
			t.Fatalf("%s%v: %v", tc.op, tc.args, err)
		}
		if col, _ := out.Series("f"); col.Value(0) != tc.want {
			t.Errorf("%s%v = %v, want %v", tc.op, tc.args, col.Value(0), tc.want)
		}
	}
}
