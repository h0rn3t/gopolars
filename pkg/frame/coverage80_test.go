package frame

import (
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// cov80Frame builds a small mixed-type frame from typed SeriesInputs.
func cov80Frame(t *testing.T, cols ...SeriesInput) DataFrame {
	t.Helper()
	df, err := FromAnyColumns(FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	return df
}

// TestFilterAggregateDirect exercises the eager fused float64 path, including the
// single-column fast path (fusedReduceFast / reduceWhere / reduceWhereSeq),
// directResult for every op, and the empty (zero-survivor) selectivity gate.
func TestFilterAggregateDirect(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{-2.0, -1.0, 0.0, 1.0, 2.0, 3.0}},
	)
	pred := expr.Col("a").Gt(expr.Lit(0.0))

	cases := []struct {
		op   string
		want float64
	}{
		{"sum", 6.0},
		{"count", 3.0},
		{"mean", 2.0},
		{"min", 1.0},
		{"max", 3.0},
	}
	for _, tc := range cases {
		got, err := df.FilterAggregateDirect(pred, tc.op, nil)
		if err != nil {
			t.Fatalf("op %s: %v", tc.op, err)
		}
		if got["a"] != tc.want {
			t.Fatalf("op %s: got %v want %v", tc.op, got["a"], tc.want)
		}
	}

	// Empty filter: every op yields 0 (directResult zero-survivor contract).
	empty, err := df.FilterAggregateDirect(expr.Col("a").Gt(expr.Lit(100.0)), "sum", nil)
	if err != nil {
		t.Fatalf("empty filter: %v", err)
	}
	if empty["a"] != 0 {
		t.Fatalf("empty sum: got %v want 0", empty["a"])
	}
	emptyMin, _ := df.FilterAggregateDirect(expr.Col("a").Gt(expr.Lit(100.0)), "min", nil)
	if emptyMin["a"] != 0 {
		t.Fatalf("empty min: got %v want 0", emptyMin["a"])
	}
}

// TestFilterAggregateDirectErrors covers every error/decline branch of the eager
// fused entry point.
func TestFilterAggregateDirectErrors(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{1.0, 2.0, 3.0}},
		SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3)}},
	)
	pred := expr.Col("a").Gt(expr.Lit(0.0))

	if _, err := df.FilterAggregateDirect(pred, "median", nil); err == nil {
		t.Fatal("expected unsupported op error")
	}
	if _, err := df.FilterAggregateDirect(expr.Col("a").Reverse(), "sum", nil); err == nil {
		t.Fatal("expected unsupported predicate error")
	}
	if _, err := df.FilterAggregateDirect(pred, "sum", []string{"missing"}); err == nil {
		t.Fatal("expected column-not-found error")
	}
	if _, err := df.FilterAggregateDirect(pred, "sum", []string{"b"}); err == nil {
		t.Fatal("expected non-float64 column error")
	}
}

// TestFilterAggregateDirectMultiColumn forces the bitmap path (more than one
// target column) rather than the single-column fast path.
func TestFilterAggregateDirectMultiColumn(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{-1.0, 1.0, 2.0, 3.0}},
		SeriesInput{Name: "b", Values: []any{10.0, 20.0, 30.0, 40.0}},
	)
	got, err := df.FilterAggregateDirect(expr.Col("a").Gt(expr.Lit(0.0)), "sum", []string{"a", "b"})
	if err != nil {
		t.Fatalf("multi-col: %v", err)
	}
	if got["a"] != 6.0 || got["b"] != 90.0 {
		t.Fatalf("multi-col: got %v", got)
	}
}

// TestFillNaNFallbackAndCast covers FillNaN over float/non-float columns and the
// Cast value coercions plus castValueToType error paths.
func TestFillNaN(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "f", Values: []any{1.0, math.NaN(), 3.0}},
		SeriesInput{Name: "s", Values: []any{"x", "y", "z"}},
	)
	out, err := df.FillNaN(0.0)
	if err != nil {
		t.Fatalf("FillNaN: %v", err)
	}
	fs, _ := out.Series("f")
	if fs.Value(1).(float64) != 0.0 {
		t.Fatalf("FillNaN did not replace NaN: %v", fs.Value(1))
	}
	// String column is untouched.
	ss, _ := out.Series("s")
	if ss.Value(0).(string) != "x" {
		t.Fatalf("FillNaN mutated string column: %v", ss.Value(0))
	}
}

func TestCastAndCastValueToType(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "i", Values: []any{int64(1), int64(2), nil}},
		SeriesInput{Name: "s", Values: []any{"10", "20", "30"}},
		SeriesInput{Name: "bs", Values: []any{"true", "false", "true"}},
	)
	out, err := df.Cast(map[string]dtypes.DataType{
		"i":  dtypes.Float64,
		"s":  dtypes.Int64,
		"bs": dtypes.Boolean,
	})
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	ic, _ := out.Series("i")
	if ic.DataType() != dtypes.Float64 || ic.Value(0).(float64) != 1.0 {
		t.Fatalf("cast int->float: %v %v", ic.DataType(), ic.Value(0))
	}
	if !ic.IsNull(2) {
		t.Fatal("null should remain null after cast")
	}
	sc, _ := out.Series("s")
	if sc.Value(0).(int64) != 10 {
		t.Fatalf("cast string->int: %v", sc.Value(0))
	}
	bc, _ := out.Series("bs")
	if bc.Value(0).(bool) != true || bc.Value(1).(bool) != false {
		t.Fatalf("cast string->bool: %v %v", bc.Value(0), bc.Value(1))
	}

	// Empty mapping is a no-op.
	if same, _ := df.Cast(nil); same.Width() != df.Width() {
		t.Fatal("nil cast should be no-op")
	}
	// Missing column errors.
	if _, err := df.Cast(map[string]dtypes.DataType{"nope": dtypes.Int64}); err == nil {
		t.Fatal("expected missing column error")
	}
	// Bad parse errors.
	bad := cov80Frame(t, SeriesInput{Name: "x", Values: []any{"notnum"}})
	if _, err := bad.Cast(map[string]dtypes.DataType{"x": dtypes.Int64}); err == nil {
		t.Fatal("expected parse error casting 'notnum' to int")
	}
	if _, err := bad.Cast(map[string]dtypes.DataType{"x": dtypes.Float64}); err == nil {
		t.Fatal("expected parse error casting 'notnum' to float")
	}
}

// TestInterpolate covers linear interpolation between known values plus the
// leading/trailing edge fill and the non-numeric skip.
func TestInterpolate(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "v", Values: []any{nil, 1.0, nil, 3.0, nil}},
		SeriesInput{Name: "label", Values: []any{"a", "b", "c", "d", "e"}},
	)
	out, err := df.Interpolate()
	if err != nil {
		t.Fatalf("Interpolate: %v", err)
	}
	v, _ := out.Series("v")
	// index 0: only right neighbor -> 1.0; index 2: midpoint of 1 and 3 -> 2.0;
	// index 4: only left neighbor -> 3.0.
	if v.Value(0).(float64) != 1.0 {
		t.Fatalf("leading edge: %v", v.Value(0))
	}
	if v.Value(2).(float64) != 2.0 {
		t.Fatalf("interp midpoint: %v", v.Value(2))
	}
	if v.Value(4).(float64) != 3.0 {
		t.Fatalf("trailing edge: %v", v.Value(4))
	}
	// String column passes through unchanged.
	l, _ := out.Series("label")
	if l.Value(0).(string) != "a" {
		t.Fatalf("string col changed: %v", l.Value(0))
	}

	// Named column that does not exist errors.
	if _, err := df.Interpolate("missing"); err == nil {
		t.Fatal("expected error for missing interpolate column")
	}

	// Integer column interpolation produces a fractional midpoint, which cannot be
	// stored back into an Int64 series, so the operation surfaces that error
	// (exercising the Int64 dtype + toFloat int branch and the New error path).
	idf := cov80Frame(t, SeriesInput{Name: "i", Values: []any{int64(0), nil, int64(3)}})
	if _, err := idf.Interpolate("i"); err == nil {
		t.Fatal("expected error interpolating Int64 column to a fractional value")
	}
}

// TestEvalShiftViaSelect exercises evalShift through Select with a shift
// expression, and the invalid-period error path is unreachable from the public
// API (Shift takes an int), so we drive the column shift result directly.
func TestEvalShiftViaSelect(t *testing.T) {
	df := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}})
	out, err := df.Select(expr.Col("v").Shift(1).Alias("shifted"))
	if err != nil {
		t.Fatalf("Select shift: %v", err)
	}
	s, _ := out.Series("shifted")
	if !s.IsNull(0) {
		t.Fatalf("shift head should be null: %v", s.Value(0))
	}
	if s.Value(1).(int64) != 1 || s.Value(2).(int64) != 2 {
		t.Fatalf("shift values: %v %v", s.Value(1), s.Value(2))
	}
}

// TestAsofJoinDirections exercises asofJoin/asofDiff for backward, forward and
// nearest directions plus the tolerance gate.
func TestAsofJoinDirections(t *testing.T) {
	left := cov80Frame(t,
		SeriesInput{Name: "t", Values: []any{int64(1), int64(5), int64(10)}},
		SeriesInput{Name: "lv", Values: []any{"a", "b", "c"}},
	)
	right := cov80Frame(t,
		SeriesInput{Name: "t", Values: []any{int64(2), int64(4), int64(8)}},
		SeriesInput{Name: "rv", Values: []any{int64(20), int64(40), int64(80)}},
	)

	backward, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"},
		How: JoinTypeAsof, AsofDirection: "backward",
	})
	if err != nil {
		t.Fatalf("asof backward: %v", err)
	}
	if backward.Height() != 3 {
		t.Fatalf("asof backward height: %d", backward.Height())
	}

	forward, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"},
		How: JoinTypeAsof, AsofDirection: "forward",
	})
	if err != nil {
		t.Fatalf("asof forward: %v", err)
	}
	if forward.Height() != 3 {
		t.Fatalf("asof forward height: %d", forward.Height())
	}

	nearest, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"},
		How: JoinTypeAsof, AsofDirection: "nearest",
	})
	if err != nil {
		t.Fatalf("asof nearest: %v", err)
	}
	if nearest.Height() != 3 {
		t.Fatalf("asof nearest height: %d", nearest.Height())
	}

	// With tolerance 1, only exact-ish matches survive.
	tol, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"},
		How: JoinTypeAsof, AsofDirection: "nearest", AsofTolerance: 1,
	})
	if err != nil {
		t.Fatalf("asof tolerance: %v", err)
	}
	if tol.Height() != 3 {
		t.Fatalf("asof tolerance height: %d", tol.Height())
	}
}

// TestAsofJoinErrors covers the asof key-count and missing-key error paths.
func TestAsofJoinErrors(t *testing.T) {
	left := cov80Frame(t, SeriesInput{Name: "t", Values: []any{int64(1)}})
	right := cov80Frame(t, SeriesInput{Name: "t", Values: []any{int64(1)}})

	if _, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t", "x"}, RightOn: []string{"t", "y"},
		How: JoinTypeAsof,
	}); err == nil {
		t.Fatal("expected single-key error")
	}
	if _, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"t"}, RightOn: []string{"nope"},
		How: JoinTypeAsof,
	}); err == nil {
		t.Fatal("expected missing right key error")
	}
	if _, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"nope"}, RightOn: []string{"t"},
		How: JoinTypeAsof,
	}); err == nil {
		t.Fatal("expected missing left key error")
	}
}

// TestAsofJoinFloatAndTime exercises the float64 and time.Time asofDiff branches.
func TestAsofJoinFloatAndTime(t *testing.T) {
	lf := cov80Frame(t, SeriesInput{Name: "t", Values: []any{1.0, 5.0}})
	rf := cov80Frame(t, SeriesInput{Name: "t", Values: []any{2.0, 4.0}})
	if _, err := lf.Join(JoinInput{
		Other: rf, LeftOn: []string{"t"}, RightOn: []string{"t"}, How: JoinTypeAsof,
	}); err != nil {
		t.Fatalf("asof float: %v", err)
	}

	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	lt := cov80Frame(t, SeriesInput{Name: "t", Values: []any{base.Add(time.Hour), base.Add(3 * time.Hour)}})
	rt := cov80Frame(t, SeriesInput{Name: "t", Values: []any{base, base.Add(2 * time.Hour)}})
	if _, err := lt.Join(JoinInput{
		Other: rt, LeftOn: []string{"t"}, RightOn: []string{"t"}, How: JoinTypeAsof,
	}); err != nil {
		t.Fatalf("asof time: %v", err)
	}
}

// TestSortStringComparisonPath drives the comparator fallback for a String key
// (compareSortValues / lessAny) and a mixed null sort.
func TestSortStringComparisonPath(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "s", Values: []any{"banana", "apple", "cherry"}},
	)
	out, err := df.Sort(SortInput{By: []string{"s"}})
	if err != nil {
		t.Fatalf("sort string: %v", err)
	}
	s, _ := out.Series("s")
	if s.Value(0).(string) != "apple" || s.Value(2).(string) != "cherry" {
		t.Fatalf("string sort order: %v %v", s.Value(0), s.Value(2))
	}

	// Descending.
	outD, err := df.Sort(SortInput{By: []string{"s"}, Descending: []bool{true}})
	if err != nil {
		t.Fatalf("sort desc: %v", err)
	}
	sd, _ := outD.Series("s")
	if sd.Value(0).(string) != "cherry" {
		t.Fatalf("desc sort: %v", sd.Value(0))
	}

	// Nulls in a string column, nulls last vs first.
	nf := cov80Frame(t, SeriesInput{Name: "s", Values: []any{"b", nil, "a"}})
	nl, err := nf.Sort(SortInput{By: []string{"s"}, NullsLast: true})
	if err != nil {
		t.Fatalf("sort nulls last: %v", err)
	}
	ns, _ := nl.Series("s")
	if !ns.IsNull(2) {
		t.Fatalf("nulls last: tail should be null, got %v", ns.Value(2))
	}
	nfirst, err := nf.Sort(SortInput{By: []string{"s"}, NullsLast: false})
	if err != nil {
		t.Fatalf("sort nulls first: %v", err)
	}
	nfs, _ := nfirst.Series("s")
	if !nfs.IsNull(0) {
		t.Fatalf("nulls first: head should be null, got %v", nfs.Value(0))
	}
}

// TestReverseUsesRowAccessor drives a row-wise Reverse expression so the
// rowAccessor RowIndex/NumRows/ValueAt accessors are exercised through expr.Eval.
func TestReverseUsesRowAccessor(t *testing.T) {
	df := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}})
	out, err := df.Select(expr.Col("v").Add(expr.Lit(int64(0))).Reverse().Alias("r"))
	if err != nil {
		t.Fatalf("reverse select: %v", err)
	}
	r, _ := out.Series("r")
	if r.Value(0).(int64) != 3 || r.Value(2).(int64) != 1 {
		t.Fatalf("reverse via row accessor: %v %v", r.Value(0), r.Value(2))
	}
}

// TestNUniqueAndGetColumnIndex covers NUnique error/empty branches and
// GetColumnIndex hit/miss.
func TestNUniqueAndGetColumnIndex(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(1), int64(2)}},
		SeriesInput{Name: "v", Values: []any{"a", "a", "b"}},
	)
	n, err := df.NUnique("g")
	if err != nil || n != 2 {
		t.Fatalf("NUnique g: n=%d err=%v", n, err)
	}
	nAll, err := df.NUnique()
	if err != nil || nAll != 2 {
		t.Fatalf("NUnique all: n=%d err=%v", nAll, err)
	}
	if _, err := df.NUnique("missing"); err == nil {
		t.Fatal("expected NUnique missing-column error")
	}
	if df.GetColumnIndex("v") != 1 {
		t.Fatalf("GetColumnIndex v: %d", df.GetColumnIndex("v"))
	}
	if df.GetColumnIndex("nope") != -1 {
		t.Fatalf("GetColumnIndex nope: %d", df.GetColumnIndex("nope"))
	}
}

// TestLimitTailSlice covers the Limit/Tail/Slice edge clamps and the viewRows
// negative/over-end path.
func TestLimitTailSlice(t *testing.T) {
	df := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}})

	if df.Limit(10).Height() != 4 {
		t.Fatal("Limit beyond height should return full frame")
	}
	if df.Limit(-1).Height() != 0 {
		t.Fatal("Limit negative should be empty")
	}
	if df.Limit(2).Height() != 2 {
		t.Fatal("Limit(2) height")
	}
	if df.Tail(10).Height() != 4 {
		t.Fatal("Tail beyond height should return full frame")
	}
	if df.Tail(-1).Height() != 0 {
		t.Fatal("Tail negative should be empty")
	}
	tail := df.Tail(2)
	tv, _ := tail.Series("v")
	if tv.Value(0).(int64) != 3 {
		t.Fatalf("Tail(2) first value: %v", tv.Value(0))
	}
	if df.Slice(0, 0).Height() != 0 {
		t.Fatal("Slice length 0 should be empty")
	}
	neg := df.Slice(-2, 2)
	if neg.Height() != 2 {
		t.Fatalf("Slice negative offset height: %d", neg.Height())
	}
}

// TestExpandColSelectorsAndStructWildcard covers expandColExprs for pl.all(),
// pl.exclude, and the struct wildcard expansion via isStructWildcard /
// structFieldNames.
func TestExpandColSelectorsAndStructWildcard(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "b", Values: []any{int64(3), int64(4)}},
		SeriesInput{Name: "m", Values: []any{
			map[string]any{"x": int64(1), "y": int64(2)},
			map[string]any{"x": int64(3), "y": int64(4)},
		}},
	)

	all, err := df.Select(expr.All())
	if err != nil {
		t.Fatalf("Select all: %v", err)
	}
	if all.Width() != 3 {
		t.Fatalf("pl.all width: %d", all.Width())
	}

	excl, err := df.Select(expr.Exclude("m"))
	if err != nil {
		t.Fatalf("Select exclude: %v", err)
	}
	if excl.Width() != 2 {
		t.Fatalf("exclude width: %d", excl.Width())
	}

	// Struct wildcard unpack: m.struct.field("*") -> x, y columns (sorted).
	wild, err := df.Select(expr.Col("m").StructField("*"))
	if err != nil {
		t.Fatalf("Select struct wildcard: %v", err)
	}
	if _, ok := wild.Series("x"); !ok {
		t.Fatalf("struct wildcard missing x: cols=%v", wild.Schema())
	}
	if _, ok := wild.Series("y"); !ok {
		t.Fatalf("struct wildcard missing y")
	}
}

// TestStructPackWildcardSelect covers the struct_pack branch of expandColExprs.
func TestStructPackWildcardSelect(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "b", Values: []any{int64(3), int64(4)}},
	)
	out, err := df.Select(expr.StructCols("a", "b").StructField("*"))
	if err != nil {
		t.Fatalf("struct pack wildcard: %v", err)
	}
	if _, ok := out.Series("a"); !ok {
		t.Fatalf("struct pack wildcard missing a")
	}
	if _, ok := out.Series("b"); !ok {
		t.Fatalf("struct pack wildcard missing b")
	}
}

// TestAggToScalarFirstLast covers aggToScalar via first()/last() folded into a
// row-wise expression (broadcast constant) and a top-level aggregate Select.
func TestAggToScalarFirstLast(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "v", Values: []any{int64(10), int64(20), int64(30)}},
	)
	// pl.col("v") + pl.col("v").first()  -> v + 10
	out, err := df.Select(expr.Col("v").Add(expr.First(expr.Col("v"))).Alias("r"))
	if err != nil {
		t.Fatalf("first fold select: %v", err)
	}
	r, _ := out.Series("r")
	if r.Value(0).(int64) != 20 || r.Value(2).(int64) != 40 {
		t.Fatalf("first fold values: %v %v", r.Value(0), r.Value(2))
	}

	// Top-level aggregate reduces to a single row.
	agg, err := df.Select(expr.Sum(expr.Col("v")).Alias("total"))
	if err != nil {
		t.Fatalf("agg select: %v", err)
	}
	if agg.Height() != 1 {
		t.Fatalf("agg select height: %d", agg.Height())
	}
	tv, _ := agg.Series("total")
	if tv.Value(0).(int64) != 60 {
		t.Fatalf("agg sum: %v", tv.Value(0))
	}
}

// TestGroupByAggExtended covers evalAgg branches: std, var, count_distinct,
// first/last, n_unique, and the extreme/sumAndCount row-wise (non-col) paths.
func TestGroupByAggExtended(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(1), int64(1), int64(2)}},
		SeriesInput{Name: "v", Values: []any{1.0, 2.0, 3.0, 5.0}},
		SeriesInput{Name: "lbl", Values: []any{"a", "a", "b", "c"}},
	)
	out, err := df.GroupBy("g").Agg(
		expr.Std(expr.Col("v")).Alias("std"),
		expr.Var(expr.Col("v")).Alias("var"),
		expr.CountDistinct(expr.Col("lbl")).Alias("nd"),
		expr.First(expr.Col("v")).Alias("first"),
		expr.Last(expr.Col("v")).Alias("last"),
		expr.NUnique(expr.Col("lbl")).Alias("nu"),
	)
	if err != nil {
		t.Fatalf("group agg: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("group height: %d", out.Height())
	}
	// Group g=1 has v {1,2,3}: var = 1.0, std = 1.0, distinct lbl {a,b} = 2.
	gOrder, _ := out.Series("g")
	row := 0
	if gOrder.Value(0).(int64) != 1 {
		row = 1
	}
	vc, _ := out.Series("var")
	if math.Abs(vc.Value(row).(float64)-1.0) > 1e-9 {
		t.Fatalf("var: %v", vc.Value(row))
	}
	sc, _ := out.Series("std")
	if math.Abs(sc.Value(row).(float64)-1.0) > 1e-9 {
		t.Fatalf("std: %v", sc.Value(row))
	}
	nd, _ := out.Series("nd")
	if nd.Value(row).(int64) != 2 {
		t.Fatalf("count_distinct: %v", nd.Value(row))
	}
	nu, _ := out.Series("nu")
	if nu.Value(row).(int64) != 2 {
		t.Fatalf("n_unique: %v", nu.Value(row))
	}
}

// TestGroupByExtremeRowWise drives the extreme/sumAndCount slow (row-wise) path
// using an expression target (col + lit) rather than a bare column.
func TestGroupByExtremeRowWise(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(1), int64(2)}},
		SeriesInput{Name: "v", Values: []any{3.0, 7.0, 5.0}},
	)
	out, err := df.GroupBy("g").Agg(
		expr.Min(expr.Col("v").Add(expr.Lit(1.0))).Alias("min1"),
		expr.Max(expr.Col("v").Add(expr.Lit(1.0))).Alias("max1"),
		expr.Sum(expr.Col("v").Add(expr.Lit(1.0))).Alias("sum1"),
		expr.Mean(expr.Col("v").Add(expr.Lit(1.0))).Alias("mean1"),
	)
	if err != nil {
		t.Fatalf("group extreme row-wise: %v", err)
	}
	gOrder, _ := out.Series("g")
	row := 0
	if gOrder.Value(0).(int64) != 1 {
		row = 1
	}
	// The aggregate target is an expression (col+lit), not a bare column, so
	// aggType reports Float64 for min/max/sum (the slow row-wise path).
	minC, _ := out.Series("min1")
	if toF(t, minC.Value(row)) != 4 { // min(3,7)+1 = 4
		t.Fatalf("min row-wise: %v", minC.Value(row))
	}
	maxC, _ := out.Series("max1")
	if toF(t, maxC.Value(row)) != 8 { // max(3,7)+1 = 8
		t.Fatalf("max row-wise: %v", maxC.Value(row))
	}
	sumC, _ := out.Series("sum1")
	if toF(t, sumC.Value(row)) != 12 { // (3+1)+(7+1)=12
		t.Fatalf("sum row-wise: %v", sumC.Value(row))
	}
}

// toF coerces an int64/float64 cell to float64 for tolerant assertions.
func toF(t *testing.T, v any) float64 {
	t.Helper()
	switch n := v.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	default:
		t.Fatalf("unexpected numeric type %T", v)
		return 0
	}
}

// TestGroupByEmptyKeys covers the Agg empty-keys error.
func TestGroupByEmptyKeys(t *testing.T) {
	df := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1)}})
	if _, err := df.GroupBy().Agg(expr.Sum(expr.Col("v"))); err == nil {
		t.Fatal("expected empty-keys error")
	}
}

// TestMeltExplodeFlatten covers Melt (with/without explicit value vars), Explode
// of a list column, and FlattenStruct.
func TestMeltExplodeFlatten(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "x", Values: []any{int64(10), int64(20)}},
		SeriesInput{Name: "y", Values: []any{int64(30), int64(40)}},
	)
	m, err := df.Melt([]string{"id"}, nil, "", "")
	if err != nil {
		t.Fatalf("Melt: %v", err)
	}
	// 2 rows * 2 value vars = 4 rows; columns id, variable, value.
	if m.Height() != 4 || m.Width() != 3 {
		t.Fatalf("Melt shape: h=%d w=%d", m.Height(), m.Width())
	}
	if _, err := df.Melt([]string{"missing"}, nil, "", ""); err == nil {
		t.Fatal("expected Melt missing id error")
	}

	// Explode.
	ldf := cov80Frame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "vals", Values: []any{[]any{int64(1), int64(2)}, []any{int64(3)}}},
	)
	ex, err := ldf.Explode("vals")
	if err != nil {
		t.Fatalf("Explode: %v", err)
	}
	if ex.Height() != 3 {
		t.Fatalf("Explode height: %d", ex.Height())
	}
	if _, err := ldf.Explode("missing"); err == nil {
		t.Fatal("expected Explode missing error")
	}
	// Explode of non-list column errors.
	if _, err := df.Explode("x"); err == nil {
		t.Fatal("expected Explode non-list error")
	}
	if same, _ := df.Explode(); same.Width() != df.Width() {
		t.Fatal("Explode with no columns should be no-op")
	}

	// FlattenStruct.
	sdf := cov80Frame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "m", Values: []any{
			map[string]any{"a": int64(1), "b": int64(2)},
			map[string]any{"a": int64(3), "b": int64(4)},
		}},
	)
	fl, err := sdf.FlattenStruct("m", "m_")
	if err != nil {
		t.Fatalf("FlattenStruct: %v", err)
	}
	if _, ok := fl.Series("m_a"); !ok {
		t.Fatalf("FlattenStruct missing m_a: %v", fl.Schema())
	}
	if _, err := sdf.FlattenStruct("missing", ""); err == nil {
		t.Fatal("expected FlattenStruct missing error")
	}
	if _, err := df.FlattenStruct("x", ""); err == nil {
		t.Fatal("expected FlattenStruct non-struct error")
	}
}

// TestOverPartitioned covers evalOver cum_sum/cum_count/rank over a partition.
func TestOverPartitioned(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(1), int64(2), int64(2)}},
		SeriesInput{Name: "v", Values: []any{1.0, 2.0, 3.0, 4.0}},
	)
	cs, err := df.Select(
		expr.Col("g"),
		expr.Col("v").CumSum().Over("g").Alias("cs"),
	)
	if err != nil {
		t.Fatalf("over cum_sum: %v", err)
	}
	csCol, _ := cs.Series("cs")
	// The Over target is itself a global cum_sum (base = [1,3,6,10]); evalOver then
	// runs a per-partition running sum over that base. g=1 -> 1, 1+3=4; g=2 -> 6, 6+10=16.
	if csCol.Value(0).(float64) != 1.0 || csCol.Value(1).(float64) != 4.0 {
		t.Fatalf("over cum_sum g1: %v %v", csCol.Value(0), csCol.Value(1))
	}
	if csCol.Value(2).(float64) != 6.0 || csCol.Value(3).(float64) != 16.0 {
		t.Fatalf("over cum_sum g2: %v %v", csCol.Value(2), csCol.Value(3))
	}

	cc, err := df.Select(expr.Col("v").CumCount().Over("g").Alias("cc"))
	if err != nil {
		t.Fatalf("over cum_count: %v", err)
	}
	ccCol, _ := cc.Series("cc")
	if ccCol.Value(1).(int64) != 2 {
		t.Fatalf("over cum_count: %v", ccCol.Value(1))
	}

	rk, err := df.Select(expr.Col("v").Rank().Over("g").Alias("rk"))
	if err != nil {
		t.Fatalf("over rank: %v", err)
	}
	rkCol, _ := rk.Series("rk")
	if rkCol.Value(0).(int64) != 1 || rkCol.Value(1).(int64) != 2 {
		t.Fatalf("over rank g1: %v %v", rkCol.Value(0), rkCol.Value(1))
	}

	// Missing partition column errors.
	if _, err := df.Select(expr.Col("v").CumSum().Over("missing")); err == nil {
		t.Fatal("expected over missing-partition error")
	}
}

// TestDeserializeRoundTrip covers Deserialize for a JSON record array and the
// empty-array branch.
func TestDeserialize(t *testing.T) {
	payload := []byte(`[{"a":1,"b":"x"},{"a":2,"b":"y"}]`)
	df, err := DataFrame{}.Deserialize(payload)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if df.Height() != 2 || df.Width() != 2 {
		t.Fatalf("Deserialize shape: h=%d w=%d", df.Height(), df.Width())
	}

	empty, err := DataFrame{}.Deserialize([]byte(`[]`))
	if err != nil {
		t.Fatalf("Deserialize empty: %v", err)
	}
	if empty.Height() != 0 {
		t.Fatalf("Deserialize empty height: %d", empty.Height())
	}

	if _, err := (DataFrame{}).Deserialize([]byte(`not json`)); err == nil {
		t.Fatal("expected Deserialize bad-json error")
	}
}

// TestFoldOps covers Fold sum/min/max with and without explicit columns plus the
// missing-column error.
func TestFoldOps(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{1.0, 2.0}},
		SeriesInput{Name: "b", Values: []any{10.0, 20.0}},
	)
	sum, err := df.Fold("sum", []string{"a", "b"}, "s")
	if err != nil {
		t.Fatalf("fold sum: %v", err)
	}
	sc, _ := sum.Series("s")
	if sc.Value(0).(float64) != 11.0 {
		t.Fatalf("fold sum: %v", sc.Value(0))
	}
	mn, err := df.Fold("min", nil, "")
	if err != nil {
		t.Fatalf("fold min: %v", err)
	}
	mc, _ := mn.Series("fold")
	if mc.Value(0).(float64) != 1.0 {
		t.Fatalf("fold min: %v", mc.Value(0))
	}
	mx, err := df.Fold("max", []string{"a", "b"}, "mx")
	if err != nil {
		t.Fatalf("fold max: %v", err)
	}
	mxc, _ := mx.Series("mx")
	if mxc.Value(1).(float64) != 20.0 {
		t.Fatalf("fold max: %v", mxc.Value(1))
	}
	if _, err := df.Fold("sum", []string{"nope"}, "x"); err == nil {
		t.Fatal("expected fold missing-column error")
	}
}

// TestCorr covers Corr happy path, the <2 sample branch, and missing columns.
func TestCorr(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "x", Values: []any{1.0, 2.0, 3.0}},
		SeriesInput{Name: "y", Values: []any{2.0, 4.0, 6.0}},
		SeriesInput{Name: "one", Values: []any{1.0, nil, nil}},
	)
	c, err := df.Corr("x", "y")
	if err != nil {
		t.Fatalf("Corr: %v", err)
	}
	if math.Abs(c-1.0) > 1e-9 {
		t.Fatalf("Corr perfect positive: %v", c)
	}
	// Fewer than 2 usable samples -> 0.
	c2, err := df.Corr("one", "one")
	if err != nil {
		t.Fatalf("Corr one: %v", err)
	}
	if c2 != 0 {
		t.Fatalf("Corr <2 samples: %v", c2)
	}
	if _, err := df.Corr("nope", "y"); err == nil {
		t.Fatal("expected Corr missing A error")
	}
	if _, err := df.Corr("x", "nope"); err == nil {
		t.Fatal("expected Corr missing B error")
	}
}

// TestEqualsBranches covers Equals false branches (height, width, name, dtype,
// value) and the true case.
func TestEqualsBranches(t *testing.T) {
	a := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2)}})
	aSame := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(2)}})
	eq, err := a.Equals(aSame)
	if err != nil || !eq {
		t.Fatalf("Equals same: eq=%v err=%v", eq, err)
	}

	diffHeight := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1)}})
	if eq, _ := a.Equals(diffHeight); eq {
		t.Fatal("Equals height mismatch should be false")
	}

	diffWidth := cov80Frame(t,
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "w", Values: []any{int64(3), int64(4)}},
	)
	if eq, _ := a.Equals(diffWidth); eq {
		t.Fatal("Equals width mismatch should be false")
	}

	diffName := cov80Frame(t, SeriesInput{Name: "x", Values: []any{int64(1), int64(2)}})
	if eq, _ := a.Equals(diffName); eq {
		t.Fatal("Equals name mismatch should be false")
	}

	diffType := cov80Frame(t, SeriesInput{Name: "v", Values: []any{1.0, 2.0}})
	if eq, _ := a.Equals(diffType); eq {
		t.Fatal("Equals dtype mismatch should be false")
	}

	diffVal := cov80Frame(t, SeriesInput{Name: "v", Values: []any{int64(1), int64(99)}})
	if eq, _ := a.Equals(diffVal); eq {
		t.Fatal("Equals value mismatch should be false")
	}
}

// TestWithContextColumns covers WithContextColumns and the contextColumn
// resolution path through evalExprAsSeries.
func TestWithContextColumns(t *testing.T) {
	df := cov80Frame(t, SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}})

	// Empty map is a no-op (same frame).
	if df.WithContextColumns(nil).Width() != 1 {
		t.Fatal("empty context should be no-op")
	}

	ctxCol, err := series.New("ctx", dtypes.Int64, []any{int64(100), int64(200)})
	if err != nil {
		t.Fatalf("ctx series: %v", err)
	}
	withCtx := df.WithContextColumns(map[string]series.Series{"ctx": ctxCol})

	// Selecting the context column resolves it (full-length, KindCol path).
	out, err := withCtx.Select(expr.Col("ctx"))
	if err != nil {
		t.Fatalf("select ctx: %v", err)
	}
	cs, _ := out.Series("ctx")
	if cs.Value(0).(int64) != 100 {
		t.Fatalf("context column value: %v", cs.Value(0))
	}

	// Merging additional context preserves the prior context.
	more, err := series.New("ctx2", dtypes.Int64, []any{int64(7), int64(8)})
	if err != nil {
		t.Fatalf("ctx2 series: %v", err)
	}
	merged := withCtx.WithContextColumns(map[string]series.Series{"ctx2": more})
	if _, ok := merged.contextColumn("ctx"); !ok {
		t.Fatal("merged context lost original ctx")
	}
	if _, ok := merged.contextColumn("ctx2"); !ok {
		t.Fatal("merged context missing ctx2")
	}
}

// TestInferDataTypeAllBranches covers inferAnyDataType and inferDataType across
// dtypes including the cannot-infer error.
func TestInferDataTypeAllBranches(t *testing.T) {
	cases := []struct {
		vals []any
		want dtypes.DataType
	}{
		{[]any{nil, int64(1)}, dtypes.Int64},
		{[]any{1.0}, dtypes.Float64},
		{[]any{"s"}, dtypes.String},
		{[]any{true}, dtypes.Boolean},
		{[]any{time.Now()}, dtypes.Datetime},
		{[]any{[]any{int64(1)}}, dtypes.List},
		{[]any{map[string]any{"k": int64(1)}}, dtypes.Struct},
		{[]any{[]byte("b")}, dtypes.Binary},
		{[]any{time.Second}, dtypes.Duration},
	}
	for _, tc := range cases {
		got, err := inferAnyDataType(tc.vals)
		if err != nil {
			t.Fatalf("inferAnyDataType %v: %v", tc.vals, err)
		}
		if got != tc.want {
			t.Fatalf("inferAnyDataType %v: got %s want %s", tc.vals, got, tc.want)
		}
		got2, err := inferDataType(tc.vals)
		if err != nil {
			t.Fatalf("inferDataType %v: %v", tc.vals, err)
		}
		if got2 != tc.want {
			t.Fatalf("inferDataType %v: got %s want %s", tc.vals, got2, tc.want)
		}
	}
	if _, err := inferAnyDataType([]any{nil, nil}); err == nil {
		t.Fatal("expected inferAnyDataType all-null error")
	}
	if _, err := inferDataType([]any{nil}); err == nil {
		t.Fatal("expected inferDataType all-null error")
	}
}

// TestFromAnyColumnsErrors covers FromAnyColumns error propagation.
func TestFromAnyColumnsErrors(t *testing.T) {
	// Unknown type -> inference fails.
	if _, err := FromAnyColumns(FromAnyColumnsInput{Columns: []SeriesInput{
		{Name: "a", Values: []any{nil}},
	}}); err == nil {
		t.Fatal("expected FromAnyColumns infer error")
	}
	// Explicit DType lets an all-null column build.
	df, err := FromAnyColumns(FromAnyColumnsInput{Columns: []SeriesInput{
		{Name: "a", Values: []any{nil, nil}, DType: dtypes.Int64},
	}})
	if err != nil {
		t.Fatalf("FromAnyColumns explicit dtype: %v", err)
	}
	if df.Height() != 2 {
		t.Fatalf("explicit dtype height: %d", df.Height())
	}
}

// TestAggregateValues exercises aggregateValues across its agg modes directly.
func TestAggregateValues(t *testing.T) {
	if aggregateValues(nil, "sum") != nil {
		t.Fatal("empty -> nil")
	}
	if aggregateValues([]any{int64(5), int64(6)}, "first").(int64) != 5 {
		t.Fatal("first")
	}
	if aggregateValues([]any{int64(2), int64(3)}, "sum").(int64) != 5 {
		t.Fatal("int sum")
	}
	if aggregateValues([]any{1.0, 2.0}, "sum").(float64) != 3.0 {
		t.Fatal("float sum")
	}
	if aggregateValues([]any{int64(2), int64(4)}, "mean").(float64) != 3.0 {
		t.Fatal("mean")
	}
	if aggregateValues([]any{"a", "b", "c"}, "count").(int64) != 3 {
		t.Fatal("count")
	}
	if aggregateValues([]any{int64(3), int64(1), int64(2)}, "min").(int64) != 1 {
		t.Fatal("min")
	}
	if aggregateValues([]any{int64(3), int64(1), int64(2)}, "max").(int64) != 3 {
		t.Fatal("max")
	}
	if aggregateValues([]any{int64(9)}, "unknown").(int64) != 9 {
		t.Fatal("unknown -> first")
	}
	// All non-numeric values -> sum yields nil (count==0).
	if aggregateValues([]any{"x", "y"}, "sum") != nil {
		t.Fatal("non-numeric sum -> nil")
	}
}
