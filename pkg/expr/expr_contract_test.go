package expr

import (
	"testing"
)

// TestPassthroughBuilders evaluates the builders that the row-wise evaluator
// treats as pass-throughs (group/window/series-context ops). Building each and
// evaluating it must return the input value unchanged — that is the contract the
// frame and group layers rely on. This exercises the one-line builders in
// expr.go together with their Eval branches.
func TestPassthroughBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{"v": int64(42)}
	builders := map[string]Expr{
		"agg_groups":      Col("v").AggGroups(),
		"approx_n_unique": Col("v").ApproxNUnique(),
		"arg_max":         Col("v").ArgMax(),
		"arg_min":         Col("v").ArgMin(),
		"arg_sort":        Col("v").ArgSort(),
		"arg_true":        Col("v").ArgTrue(),
		"arg_unique":      Col("v").ArgUnique(),
		"backward_fill":   Col("v").BackwardFill(),
		"count":           Col("v").Count(),
		"cum_max":         Col("v").CumMax(),
		"cum_min":         Col("v").CumMin(),
		"cum_prod":        Col("v").CumProd(),
		"cumulative_eval": Col("v").CumulativeEval(),
		"diff":            Col("v").Diff(),
		"drop_nans":       Col("v").DropNans(),
		"drop_nulls":      Col("v").DropNulls(),
		"entropy":         Col("v").Entropy(),
		"ewm_mean":        Col("v").EwmMean(),
		"ewm_std":         Col("v").EwmStd(),
		"ewm_var":         Col("v").EwmVar(),
		"first":           Col("v").First(),
		"flatten":         Col("v").Flatten(),
		"forward_fill":    Col("v").ForwardFill(),
		"hash":            Col("v").Hash(),
		"implode":         Col("v").Implode(),
		"interpolate":     Col("v").Interpolate(),
		"is_duplicated":   Col("v").IsDuplicated(),
		"is_first":        Col("v").IsFirstDistinct(),
		"is_last":         Col("v").IsLastDistinct(),
		"is_unique":       Col("v").IsUnique(),
		"item":            Col("v").Item(),
		"kurtosis":        Col("v").Kurtosis(),
		"last":            Col("v").Last(),
		"lower_bound":     Col("v").LowerBound(),
		"map_batches":     Col("v").MapBatches(),
		"map_elements":    Col("v").MapElements(),
		"max":             Col("v").Max(),
		"mean":            Col("v").Mean(),
		"median":          Col("v").Median(),
		"meta":            Col("v").Meta(),
		"min":             Col("v").Min(),
		"mode":            Col("v").Mode(),
		"n_unique":        Col("v").NUnique(),
		"nan_max":         Col("v").NanMax(),
		"nan_min":         Col("v").NanMin(),
		"null_count":      Col("v").NullCount(),
		"pct_change":      Col("v").PctChange(),
		"peak_max":        Col("v").PeakMax(),
		"peak_min":        Col("v").PeakMin(),
		"pipe":            Col("v").Pipe(),
		"product":         Col("v").Product(),
		"qcut":            Col("v").QCut(),
		"quantile":        Col("v").Quantile(),
		"rechunk":         Col("v").Rechunk(),
		"reinterpret":     Col("v").Reinterpret(),
		"rle":             Col("v").Rle(),
		"rle_id":          Col("v").RleId(),
		"shrink_dtype":    Col("v").ShrinkDtype(),
		"truncate":        Col("v").Truncate(),
		"unique":          Col("v").Unique(),
		"unique_counts":   Col("v").UniqueCounts(),
		"value_counts":    Col("v").ValueCounts(),
		"head":            Col("v").Head(3),
		"tail":            Col("v").Tail(3),
		"limit":           Col("v").Limit(3),
		"slice":           Col("v").Slice(0, 2),
		"gather_every":    Col("v").GatherEvery(2),
		"bottom_k":        Col("v").BottomK(2),
		"top_k_prefix":    Col("v").TopK(2),
		"shift":           Col("v").Shift(1),
		"reshape":         Col("v").Reshape(2, 1),
		"sample":          Col("v").Sample(1, 7),
		"shuffle":         Col("v").Shuffle(7),
		"set_sorted":      Col("v").SetSorted(false),
		"sort":            Col("v").Sort(false),
		"repeat_by":       Col("v").RepeatBy(2),
	}
	for name, e := range builders {
		got, err := Eval(e, row)
		if err != nil {
			t.Errorf("%s: unexpected error %v", name, err)
			continue
		}
		if got != int64(42) {
			t.Errorf("%s = %v, want pass-through 42", name, got)
		}
	}
}

// TestNamespaceBuilders covers the namespace accessor builders (pass-throughs).
func TestNamespaceBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{"v": int64(1)}
	for _, e := range []Expr{
		Col("v").Dt(), Col("v").Str(), Col("v").List(), Col("v").Struct(),
		Col("v").Arr(), Col("v").Cat(), Col("v").Bin(), Col("v").Ext(),
	} {
		if got, err := Eval(e, row); err != nil || got != int64(1) {
			t.Errorf("namespace %q = %v err=%v", e.Op(), got, err)
		}
	}
}

// TestAggregationConstructors covers the SQL-catalog aggregation constructors,
// which the group-by layer dispatches on by Kind/Op.
func TestAggregationConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		e      Expr
		wantOp string
	}{
		{"std", Std(Col("v")), "std"},
		{"var", Var(Col("v")), "var"},
		{"median", Median(Col("v")), "median"},
		{"first", First(Col("v")), "first"},
		{"last", Last(Col("v")), "last"},
		{"count_distinct", CountDistinct(Col("v")), "count_distinct"},
	}
	for _, tc := range cases {
		if tc.e.Kind() != KindAgg {
			t.Errorf("%s kind = %q, want agg", tc.name, tc.e.Kind())
		}
		if tc.e.Op() != tc.wantOp {
			t.Errorf("%s op = %q, want %q", tc.name, tc.e.Op(), tc.wantOp)
		}
		if tc.e.Target() == nil || tc.e.Target().ColName() != "v" {
			t.Errorf("%s target not preserved", tc.name)
		}
	}
}

// TestSelectors covers the multi-column selector machinery: Cols, All, Exclude,
// regex names, and StructCols, via SelectorColumns.
func TestSelectors(t *testing.T) {
	t.Parallel()

	available := []string{"a", "b", "city"}

	cols, ok := SelectorColumns(Cols("a", "city"), available)
	if !ok || len(cols) != 2 || cols[0] != "a" || cols[1] != "city" {
		t.Fatalf("Cols selector = %v ok=%v", cols, ok)
	}

	all, ok := SelectorColumns(All(), available)
	if !ok || len(all) != 3 {
		t.Fatalf("All selector = %v ok=%v", all, ok)
	}

	excl, ok := SelectorColumns(Exclude("b"), available)
	if !ok || len(excl) != 2 || excl[0] != "a" || excl[1] != "city" {
		t.Fatalf("Exclude selector = %v ok=%v", excl, ok)
	}

	// Regex column selector (^...$).
	rx, ok := SelectorColumns(Col("^c.*$"), available)
	if !ok || len(rx) != 1 || rx[0] != "city" {
		t.Fatalf("regex selector = %v ok=%v", rx, ok)
	}

	// A plain column is not a selector.
	if _, ok := SelectorColumns(Col("a"), available); ok {
		t.Fatal("plain column should not be a selector")
	}

	if !isRegexName("^a$") || isRegexName("a") {
		t.Fatal("isRegexName classification wrong")
	}

	// StructCols packs named columns into a struct value.
	packed, err := Eval(StructCols("a", "b"), mapRow{"a": int64(1), "b": int64(2)})
	if err != nil {
		t.Fatalf("struct_pack: %v", err)
	}
	m, ok := packed.(map[string]any)
	if !ok || m["a"] != int64(1) || m["b"] != int64(2) {
		t.Fatalf("struct_pack = %v", packed)
	}
}

// TestFold horizontally reduces operand exprs left to right.
func TestFold(t *testing.T) {
	t.Parallel()

	// Sum a + b + c starting from 0.
	add := func(acc any, next any) (any, error) {
		return acc.(int64) + next.(int64), nil
	}
	e := Fold(int64(0), add, Col("a"), Col("b"), Col("c"))
	got, err := Eval(e, mapRow{"a": int64(1), "b": int64(2), "c": int64(3)})
	if err != nil || got != int64(6) {
		t.Fatalf("fold = %v err=%v, want 6", got, err)
	}
}
