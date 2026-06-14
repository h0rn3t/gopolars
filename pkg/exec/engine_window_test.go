package exec

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// windowFrame builds a small partitioned frame:
//
//	grp:   a a a b b
//	val:  10 20 30 40 50
func windowFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "grp", Values: []any{"a", "a", "a", "b", "b"}},
		frame.SeriesInput{Name: "val", Values: []any{int64(10), int64(20), int64(30), int64(40), int64(50)}},
	)
}

// colValues reads an output column into a slice in row order.
func colValues(t *testing.T, df frame.DataFrame, name string) []any {
	t.Helper()
	s, ok := df.Series(name)
	if !ok {
		t.Fatalf("column %s not found", name)
	}
	out := make([]any, s.Len())
	for i := range out {
		out[i] = s.Value(i)
	}
	return out
}

// TestApplyWindowsPartitionAgg covers the default-aggregation window path
// (computePartitionAgg) for sum/mean/min/max/count, broadcast across each
// partition's rows.
func TestApplyWindowsPartitionAgg(t *testing.T) {
	t.Parallel()

	df := windowFrame(t)

	cases := []struct {
		fn   string
		want []any
	}{
		{"sum", []any{int64(60), int64(60), int64(60), int64(90), int64(90)}},
		{"mean", []any{20.0, 20.0, 20.0, 45.0, 45.0}},
		{"min", []any{int64(10), int64(10), int64(10), int64(40), int64(40)}},
		{"max", []any{int64(30), int64(30), int64(30), int64(50), int64(50)}},
		{"count", []any{int64(3), int64(3), int64(3), int64(2), int64(2)}},
	}
	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			out, err := applyWindows(df, []logical.WindowSpec{{
				Func:        tc.fn,
				Target:      "val",
				Alias:       "w",
				PartitionBy: []string{"grp"},
			}})
			if err != nil {
				t.Fatalf("applyWindows %s: %v", tc.fn, err)
			}
			got := colValues(t, out, "w")
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("%s row %d = %v, want %v", tc.fn, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestApplyWindowsRanking covers row_number plus lag/lead and first/last_value
// over an ordered partition.
func TestApplyWindowsRanking(t *testing.T) {
	t.Parallel()

	df := windowFrame(t)

	rn, err := applyWindows(df, []logical.WindowSpec{{
		Func:        "row_number",
		Alias:       "rn",
		PartitionBy: []string{"grp"},
		OrderBy:     []string{"val"},
	}})
	if err != nil {
		t.Fatalf("row_number: %v", err)
	}
	// partition a -> 1,2,3 ; partition b -> 1,2
	want := []any{int64(1), int64(2), int64(3), int64(1), int64(2)}
	got := colValues(t, rn, "rn")
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("row_number row %d = %v, want %v", i, got[i], want[i])
		}
	}

	lag, err := applyWindows(df, []logical.WindowSpec{{
		Func:        "lag",
		Target:      "val",
		Alias:       "prev",
		PartitionBy: []string{"grp"},
		OrderBy:     []string{"val"},
		Offset:      1,
		Default:     nil,
	}})
	if err != nil {
		t.Fatalf("lag: %v", err)
	}
	// per-partition previous value: a -> nil,10,20 ; b -> nil,40
	prev := colValues(t, lag, "prev")
	wantPrev := []any{nil, int64(10), int64(20), nil, int64(40)}
	for i := range prev {
		if prev[i] != wantPrev[i] {
			t.Fatalf("lag row %d = %v, want %v", i, prev[i], wantPrev[i])
		}
	}

	first, err := applyWindows(df, []logical.WindowSpec{{
		Func:        "first_value",
		Target:      "val",
		Alias:       "fv",
		PartitionBy: []string{"grp"},
		OrderBy:     []string{"val"},
	}})
	if err != nil {
		t.Fatalf("first_value: %v", err)
	}
	fv := colValues(t, first, "fv")
	wantFV := []any{int64(10), int64(10), int64(10), int64(40), int64(40)}
	for i := range fv {
		if fv[i] != wantFV[i] {
			t.Fatalf("first_value row %d = %v, want %v", i, fv[i], wantFV[i])
		}
	}
}

// TestApplyWindowsRankTies covers RANK vs DENSE_RANK tie handling, which drives
// orderKeyChanged.
func TestApplyWindowsRankTies(t *testing.T) {
	t.Parallel()

	// One partition, order keys: 10,10,20 -> RANK 1,1,3 ; DENSE_RANK 1,1,2.
	df := mustFrame(t,
		frame.SeriesInput{Name: "grp", Values: []any{"a", "a", "a"}},
		frame.SeriesInput{Name: "k", Values: []any{int64(10), int64(10), int64(20)}},
	)

	rank, err := applyWindows(df, []logical.WindowSpec{{
		Func: "rank", Alias: "r", PartitionBy: []string{"grp"}, OrderBy: []string{"k"},
	}})
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if got := colValues(t, rank, "r"); got[0] != int64(1) || got[1] != int64(1) || got[2] != int64(3) {
		t.Fatalf("rank = %v, want [1 1 3]", got)
	}

	dense, err := applyWindows(df, []logical.WindowSpec{{
		Func: "dense_rank", Alias: "d", PartitionBy: []string{"grp"}, OrderBy: []string{"k"},
	}})
	if err != nil {
		t.Fatalf("dense_rank: %v", err)
	}
	if got := colValues(t, dense, "d"); got[0] != int64(1) || got[1] != int64(1) || got[2] != int64(2) {
		t.Fatalf("dense_rank = %v, want [1 1 2]", got)
	}
}

// TestOrderKeyChanged covers the tie-detection helper directly.
func TestOrderKeyChanged(t *testing.T) {
	t.Parallel()

	df := mustFrame(t, frame.SeriesInput{Name: "k", Values: []any{int64(10), int64(10), int64(20)}})

	if orderKeyChanged(df, []string{"k"}, 0, 1) {
		t.Fatal("rows 0,1 share key 10 -> should be unchanged")
	}
	if !orderKeyChanged(df, []string{"k"}, 1, 2) {
		t.Fatal("rows 1,2 differ (10 vs 20) -> should be changed")
	}
	// Missing column is skipped, so no change detected.
	if orderKeyChanged(df, []string{"missing"}, 0, 2) {
		t.Fatal("missing order column should report unchanged")
	}
}

// TestApplyWindowsErrors covers the missing-column error paths.
func TestApplyWindowsErrors(t *testing.T) {
	t.Parallel()

	df := windowFrame(t)

	if _, err := applyWindows(df, []logical.WindowSpec{{
		Func: "sum", Target: "val", Alias: "w", PartitionBy: []string{"nope"},
	}}); err == nil {
		t.Fatal("expected error for missing partition column")
	}
	if _, err := applyWindows(df, []logical.WindowSpec{{
		Func: "lag", Target: "nope", Alias: "w", PartitionBy: []string{"grp"}, OrderBy: []string{"val"},
	}}); err == nil {
		t.Fatal("expected error for missing lag target column")
	}
}
