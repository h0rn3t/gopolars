package exec

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
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

// TestExtractGlobalLimit pulls the smallest limit out of the pipeline and
// returns the remaining (non-limit) nodes.
func TestExtractGlobalLimit(t *testing.T) {
	t.Parallel()

	nodes := []logical.Node{
		{Type: logical.NodeFilter},
		{Type: logical.NodeLimit, IntValue: 10},
		{Type: logical.NodeSelect},
		{Type: logical.NodeLimit, IntValue: 4},
	}
	remaining, limit := extractGlobalLimit(nodes)
	if limit != 4 {
		t.Fatalf("limit = %d, want 4 (smallest)", limit)
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining nodes = %d, want 2", len(remaining))
	}
	for _, n := range remaining {
		if n.Type == logical.NodeLimit {
			t.Fatal("limit nodes should be stripped")
		}
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

// TestHasStatefulNode covers the stateful/stateless classification.
func TestHasStatefulNode(t *testing.T) {
	t.Parallel()

	stateful := [][]logical.Node{
		{{Type: logical.NodeSort}},
		{{Type: logical.NodeAggregate}},
		{{Type: logical.NodeJoin}},
		{{Type: logical.NodeWindow}},
	}
	for _, nodes := range stateful {
		if !hasStatefulNode(nodes) {
			t.Fatalf("%v should be stateful", nodes[0].Type)
		}
	}

	if hasStatefulNode([]logical.Node{{Type: logical.NodeFilter}, {Type: logical.NodeSelect}}) {
		t.Fatal("filter+select should be stateless")
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
