package frame

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func mustFrame(t *testing.T, cols ...SeriesInput) DataFrame {
	t.Helper()
	df, err := FromAnyColumns(FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("побудова dataframe: %v", err)
	}
	return df
}

func TestFrameMetadataAndAccessors(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), nil, int64(3)}},
		SeriesInput{Name: "tag", Values: []any{"go", "rs", "py"}},
		SeriesInput{Name: "meta", Values: []any{map[string]any{"k": "v"}, map[string]any{"k": "w"}, nil}},
		SeriesInput{Name: "items", Values: []any{[]any{int64(1)}, []any{int64(2), int64(3)}, []any{}}},
	)

	if df.IsEmpty() || df.Width() != 4 || df.Height() != 3 {
		t.Fatalf("розміри: empty=%v w=%d h=%d", df.IsEmpty(), df.Width(), df.Height())
	}
	if df.EstimatedSize() <= 0 {
		t.Fatal("estimated size має бути > 0")
	}
	if len(df.Dtypes()) != 4 {
		t.Fatalf("dtypes: %d", len(df.Dtypes()))
	}

	col, err := df.GetColumn("id")
	if err != nil || col.Len() != 3 {
		t.Fatalf("get column: %v len=%d", err, col.Len())
	}
	if df.GetColumnIndex("tag") != 1 {
		t.Fatalf("index tag: %d", df.GetColumnIndex("tag"))
	}
	if _, err := df.GetColumn("missing"); err == nil {
		t.Fatal("очікували помилку для відсутньої колонки")
	}

	flags := df.Flags()
	if flags == nil {
		t.Fatal("flags nil")
	}
	if glimpse := df.Glimpse(2); glimpse == "" {
		t.Fatal("glimpse порожній")
	}
	if len(df.ToDicts()) != 3 {
		t.Fatalf("todicts: %d", len(df.ToDicts()))
	}
	if df.NullCount()["id"] != 1 || df.Count()["id"] != 2 {
		t.Fatalf("null/count: %+v %+v", df.NullCount(), df.Count())
	}
	nu, err := df.NUnique("tag")
	if err != nil || nu != 3 {
		t.Fatalf("nunique: %d err=%v", nu, err)
	}
	if approx, _ := df.ApproxNUnique("tag"); approx != 3 {
		t.Fatalf("approx nunique: %d", approx)
	}
}

func TestFrameBottomKSliceTailEdges(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "v", Values: []any{int64(30), int64(10), int64(20)}},
	)

	if empty, err := df.BottomK(0, "v"); err != nil || empty.Height() != 0 {
		t.Fatalf("bottomk 0: err=%v h=%d", err, empty.Height())
	}
	bottom, err := df.BottomK(2, "v")
	if err != nil || bottom.Height() != 2 {
		t.Fatalf("bottomk: err=%v h=%d", err, bottom.Height())
	}
	vCol, _ := bottom.Series("v")
	if vCol.Value(0) != int64(10) {
		t.Fatalf("bottomk sort: %v", vCol.Value(0))
	}

	if df.Tail(10).Height() != 3 {
		t.Fatal("tail n>=height")
	}
	if df.Slice(5, 2).Height() != 0 {
		t.Fatal("slice offset>=height")
	}
	sliced := df.Slice(1, 2)
	if sliced.Height() != 2 {
		t.Fatalf("slice: %d", sliced.Height())
	}
}

func TestFrameCastFoldAndInsert(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "s", Values: []any{"42", "7"}},
		SeriesInput{Name: "f", Values: []any{float64(1.5), float64(2.5)}},
		SeriesInput{Name: "b", Values: []any{"true", "false"}},
	)

	casted, err := df.Cast(map[string]dtypes.DataType{
		"s": dtypes.Int64,
		"f": dtypes.Float64,
		"b": dtypes.Boolean,
	})
	if err != nil {
		t.Fatalf("cast: %v", err)
	}
	sCol, _ := casted.Series("s")
	if sCol.Value(0) != int64(42) {
		t.Fatalf("cast string->int: %v", sCol.Value(0))
	}

	folded, err := df.Fold("min", []string{"f"}, "fold_min")
	if err != nil {
		t.Fatalf("fold min: %v", err)
	}
	foldCol, _ := folded.Series("fold_min")
	if foldCol.Value(0) != float64(1.5) {
		t.Fatalf("fold: %v", foldCol.Value(0))
	}
	if _, err := df.Fold("sum", []string{"missing"}, "x"); err == nil {
		t.Fatal("fold missing column")
	}

	extra, err := series.New("extra", dtypes.Int64, []any{int64(9), int64(8)})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	withExtra, err := df.InsertColumn(0, extra)
	if err != nil || withExtra.Columns()[0] != "extra" {
		t.Fatalf("insert: err=%v cols=%v", err, withExtra.Columns())
	}
	replaced, err := withExtra.InsertColumn(1, extra)
	if err != nil || len(replaced.Columns()) != 4 {
		t.Fatalf("replace insert: err=%v w=%d", err, replaced.Width())
	}

	dropped, err := df.DropInPlace("b")
	if err != nil || dropped.Width() != 2 {
		t.Fatalf("drop in place: err=%v w=%d", err, dropped.Width())
	}
	if _, err := df.DropInPlace("nope"); err == nil {
		t.Fatal("drop in place missing")
	}
}

func TestFrameSelectFilterWithExpressions(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "g", Values: []any{"a", "a", "b"}},
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
	)

	selected, err := df.Select(
		expr.Col("g"),
		expr.Col("v").Add(expr.Lit(int64(10))).Alias("v_plus"),
		expr.Col("v").CumSum().Alias("cum"),
		expr.Col("v").Rank().Alias("rk"),
		expr.Col("v").RollingSum(2).Alias("roll"),
		expr.Col("v").CumSum().Over("g").Alias("cum_g"),
	)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if selected.Width() != 6 {
		t.Fatalf("select width: %d", selected.Width())
	}

	filtered, err := df.Filter(expr.Col("v").Gt(expr.Lit(int64(1))))
	if err != nil || filtered.Height() != 2 {
		t.Fatalf("filter: err=%v h=%d", err, filtered.Height())
	}

	withCols, err := df.WithColumns(expr.Col("v").Mul(expr.Lit(int64(2))).Alias("v2"))
	if err != nil {
		t.Fatalf("with columns: %v", err)
	}
	v2, _ := withCols.Series("v2")
	if v2.Value(0) != int64(2) {
		t.Fatalf("with columns val: %v", v2.Value(0))
	}
}

func TestFrameFillInterpolateUniqueEquals(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "x", Values: []any{float64(1), math.NaN(), float64(3)}},
		SeriesInput{Name: "k", Values: []any{"a", "a", "b"}},
	)

	filled, err := df.FillNaN(0)
	if err != nil {
		t.Fatalf("fill nan: %v", err)
	}
	xCol, _ := filled.Series("x")
	if math.IsNaN(xCol.Value(1).(float64)) {
		t.Fatal("fill nan не замінив NaN")
	}

	interp, err := df.Interpolate("x")
	if err != nil {
		t.Fatalf("interpolate: %v", err)
	}
	xInterp, _ := interp.Series("x")
	if xInterp.Value(1) == nil {
		t.Fatal("interpolate очікував значення")
	}

	uniq, err := df.Unique("k")
	if err != nil || uniq.Height() != 2 {
		t.Fatalf("unique: err=%v h=%d", err, uniq.Height())
	}

	clone := df.Clone()
	eq, err := df.Equals(clone)
	if err != nil || !eq {
		t.Fatalf("equals: %v err=%v", eq, err)
	}
}

func TestFrameHashCorrDescribeDeserialize(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		SeriesInput{Name: "b", Values: []any{int64(2), int64(4), int64(6)}},
	)

	hashes, err := df.HashRows(99)
	if err != nil || len(hashes) != 3 {
		t.Fatalf("hash rows: err=%v n=%d", err, len(hashes))
	}
	corr, err := df.Corr("a", "b")
	if err != nil || corr < 0.99 {
		t.Fatalf("corr: %v err=%v", corr, err)
	}
	desc, err := df.Describe()
	if err != nil || desc.Height() == 0 {
		t.Fatalf("describe: err=%v h=%d", err, desc.Height())
	}

	payload, _ := json.Marshal([]map[string]any{
		{"id": int64(1), "city": "kyiv"},
		{"id": int64(2), "city": "lviv"},
	})
	restored, err := df.Deserialize(payload)
	if err != nil || restored.Height() != 2 {
		t.Fatalf("deserialize: err=%v h=%d", err, restored.Height())
	}
}

func TestFrameGroupByAllAggregations(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "g", Values: []any{"a", "a", "b", "b"}},
		SeriesInput{Name: "v", Values: []any{float64(1), float64(3), math.NaN(), float64(5)}},
		SeriesInput{Name: "tag", Values: []any{"x", "x", "y", "z"}},
	)

	out, err := df.GroupBy("g").Agg(
		expr.Sum(expr.Col("v")).Alias("sum_v"),
		expr.Mean(expr.Col("v")).Alias("mean_v"),
		expr.Count().Alias("cnt"),
		expr.NUnique(expr.Col("tag")).Alias("nu"),
		expr.Min(expr.Col("v")).Alias("min_v"),
		expr.Max(expr.Col("v")).Alias("max_v"),
	)
	if err != nil {
		t.Fatalf("groupby agg: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("groups: %d", out.Height())
	}
	gCol, _ := out.Series("g")
	minCol, _ := out.Series("min_v")
	var minA any
	for i := 0; i < out.Height(); i++ {
		if gCol.Value(i) == "a" {
			minA = minCol.Value(i)
			break
		}
	}
	if minA != float64(1) {
		t.Fatalf("min group a: %v", minA)
	}
}

func TestFramePivotMeltExplodeHstackExtend(t *testing.T) {
	t.Parallel()

	long := mustFrame(t,
		SeriesInput{Name: "k", Values: []any{"r1", "r1", "r2"}},
		SeriesInput{Name: "dim", Values: []any{"x", "y", "x"}},
		SeriesInput{Name: "val", Values: []any{int64(10), int64(20), int64(30)}},
	)
	pivoted, err := long.Pivot([]string{"k"}, "dim", "val", "sum")
	if err != nil || pivoted.Width() != 3 {
		t.Fatalf("pivot sum: err=%v w=%d", err, pivoted.Width())
	}
	pivotedMean, err := long.Pivot([]string{"k"}, "dim", "val", "mean")
	if err != nil {
		t.Fatalf("pivot mean: %v", err)
	}
	_ = pivotedMean

	melted, err := long.Melt([]string{"k"}, []string{"val"}, "variable", "value")
	if err != nil || melted.Height() != 3 {
		t.Fatalf("melt: err=%v h=%d", err, melted.Height())
	}

	listDF := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "tags", Values: []any{[]any{"a", "b"}, []any{"c"}}},
	)
	exploded, err := listDF.Explode("tags")
	if err != nil || exploded.Height() != 3 {
		t.Fatalf("explode: err=%v h=%d", err, exploded.Height())
	}

	extra, _ := series.New("z", dtypes.Int64, []any{int64(9), int64(8)})
	stacked, err := listDF.Hstack(extra)
	if err != nil || stacked.Width() != 3 {
		t.Fatalf("hstack: err=%v w=%d", err, stacked.Width())
	}
	extended, err := listDF.Extend(listDF)
	if err != nil || extended.Height() != 4 {
		t.Fatalf("extend: err=%v h=%d", err, extended.Height())
	}
}

func TestFrameTemporalRollingAndDynamicGroup(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	df := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{
			base,
			base.Add(time.Hour),
			base.Add(2 * time.Hour),
			base.Add(3 * time.Hour),
		}},
		SeriesInput{Name: "v", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
	)

	rolled, err := df.RollingMean("ts", "v", 2*time.Hour, 1, "rm", "both")
	if err != nil || rolled.Width() != 3 {
		t.Fatalf("rolling mean: err=%v w=%d", err, rolled.Width())
	}
	rm, _ := rolled.Series("rm")
	if rm.Value(2) == nil {
		t.Fatal("rolling mean очікував значення на 3-му рядку")
	}

	dyn, err := df.GroupByDynamic("ts", time.Hour, time.Hour, 0, "right", "right", "win", expr.Sum(expr.Col("v")))
	if err != nil || dyn.Height() == 0 {
		t.Fatalf("group by dynamic: err=%v h=%d", err, dyn.Height())
	}
}

func TestJoinVariants(t *testing.T) {
	t.Parallel()

	left := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		SeriesInput{Name: "lv", Values: []any{"a", "b", "c"}},
	)
	right := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(2), int64(3), int64(4)}},
		SeriesInput{Name: "rv", Values: []any{"B", "C", "D"}},
	)
	keys := JoinInput{
		Other:   right,
		LeftOn:  []string{"id"},
		RightOn: []string{"id"},
	}

	inner, err := left.Join(keys)
	if err != nil || inner.Height() != 2 {
		t.Fatalf("inner join: err=%v h=%d", err, inner.Height())
	}

	semiKeys := keys
	semiKeys.How = JoinTypeSemi
	semi, err := left.Join(semiKeys)
	if err != nil || semi.Height() != 2 {
		t.Fatalf("semi join: err=%v h=%d", err, semi.Height())
	}

	antiKeys := keys
	antiKeys.How = JoinTypeAnti
	anti, err := left.Join(antiKeys)
	idAnti, _ := anti.Series("id")
	if err != nil || anti.Height() != 1 || idAnti.Value(0) != int64(1) {
		t.Fatalf("anti join: err=%v h=%d id=%v", err, anti.Height(), idAnti.Value(0))
	}

	fullKeys := keys
	fullKeys.How = JoinTypeFull
	full, err := left.Join(fullKeys)
	if err != nil || full.Height() != 4 {
		t.Fatalf("full join: err=%v h=%d", err, full.Height())
	}

	cross, err := left.Join(JoinInput{Other: right, How: JoinTypeCross})
	if err != nil || cross.Height() != 9 {
		t.Fatalf("cross join: err=%v h=%d", err, cross.Height())
	}
}

func TestAsofJoinWithTimeKeys(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	left := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{t0, t0.Add(time.Minute)}},
		SeriesInput{Name: "lv", Values: []any{int64(1), int64(2)}},
	)
	right := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{t0.Add(-30 * time.Second), t0.Add(30 * time.Second)}},
		SeriesInput{Name: "rv", Values: []any{int64(10), int64(20)}},
	)

	joined, err := left.Join(JoinInput{
		Other:         right,
		LeftOn:        []string{"ts"},
		RightOn:       []string{"ts"},
		How:           JoinTypeAsof,
		AsofDirection: "forward",
		AsofTolerance: int64((2 * time.Minute).Nanoseconds()),
	})
	if err != nil || joined.Height() != 2 {
		t.Fatalf("asof join: err=%v h=%d", err, joined.Height())
	}
}

func TestFrameDropFillGatherUtilities(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), nil, int64(3)}},
		SeriesInput{Name: "b", Values: []any{float64(1), math.NaN(), float64(3)}},
		SeriesInput{Name: "c", Values: []any{"x", "y", "z"}},
	)

	filled, err := df.FillNull(int64(0))
	if err != nil {
		t.Fatalf("fill null: %v", err)
	}
	aFilled, _ := filled.Series("a")
	if aFilled.Value(1) != int64(0) {
		t.Fatalf("fill null: %v", aFilled.Value(1))
	}

	dropped, err := df.Drop("c")
	if err != nil || dropped.Width() != 2 {
		t.Fatalf("drop: err=%v w=%d", err, dropped.Width())
	}
	dropNulls := df.DropNulls("a")
	if dropNulls.Height() != 2 {
		t.Fatalf("drop nulls: %d", dropNulls.Height())
	}
	dropNaNs := df.DropNaNs("b")
	if dropNaNs.Height() != 2 {
		t.Fatalf("drop nans: %d", dropNaNs.Height())
	}

	every := df.GatherEvery(2, 1)
	if every.Height() != 1 {
		t.Fatalf("gather every: %d", every.Height())
	}
	if cols := df.GetColumns(); len(cols) != 3 {
		t.Fatalf("get columns: %d", len(cols))
	}
	if schema := df.CollectSchema(); len(schema) != 3 {
		t.Fatalf("collect schema: %d", len(schema))
	}
}

func TestFrameSelectReverseRollingStdPivotAgg(t *testing.T) {
	t.Parallel()

	df := mustFrame(t,
		SeriesInput{Name: "g", Values: []any{"a", "a", "b"}},
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3)}},
	)

	rev, err := df.Select(expr.Col("v").Reverse().Alias("rev"))
	if err != nil {
		t.Fatalf("reverse select: %v", err)
	}
	revCol, _ := rev.Series("rev")
	if revCol.Value(0) != int64(3) {
		t.Fatalf("reverse: %v", revCol.Value(0))
	}

	withRoll, err := df.Select(
		expr.Col("v").RollingStd(2).Alias("rstd"),
		expr.Col("v").RollingVar(2).Alias("rvar"),
	)
	if err != nil {
		t.Fatalf("rolling std/var: %v", err)
	}
	if withRoll.Width() != 2 {
		t.Fatalf("rolling width: %d", withRoll.Width())
	}

	long := mustFrame(t,
		SeriesInput{Name: "k", Values: []any{"r1", "r1"}},
		SeriesInput{Name: "dim", Values: []any{"x", "y"}},
		SeriesInput{Name: "val", Values: []any{int64(10), int64(20)}},
	)
	pivMax, err := long.Pivot([]string{"k"}, "dim", "val", "max")
	if err != nil {
		t.Fatalf("pivot max: %v", err)
	}
	_ = pivMax
}

func TestFrameJoinLeftRightAsofBackward(t *testing.T) {
	t.Parallel()

	left := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "lv", Values: []any{"a", "b"}},
	)
	right := mustFrame(t,
		SeriesInput{Name: "id", Values: []any{int64(2), int64(3)}},
		SeriesInput{Name: "rv", Values: []any{"B", "C"}},
	)

	leftJoin, err := left.Join(JoinInput{
		Other: right, LeftOn: []string{"id"}, RightOn: []string{"id"}, How: JoinTypeLeft,
	})
	if err != nil || leftJoin.Height() != 2 {
		t.Fatalf("left join: err=%v h=%d", err, leftJoin.Height())
	}

	rightJoin, err := right.Join(JoinInput{
		Other: left, LeftOn: []string{"id"}, RightOn: []string{"id"}, How: JoinTypeRight,
	})
	if err != nil || rightJoin.Height() != 2 {
		t.Fatalf("right join: err=%v h=%d", err, rightJoin.Height())
	}

	t0 := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	lts := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{t0, t0.Add(time.Minute)}},
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2)}},
	)
	rts := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{t0.Add(-time.Minute), t0}},
		SeriesInput{Name: "rv", Values: []any{int64(10), int64(20)}},
	)
	asof, err := lts.Join(JoinInput{
		Other:         rts,
		LeftOn:        []string{"ts"},
		RightOn:       []string{"ts"},
		How:           JoinTypeAsof,
		AsofDirection: "backward",
	})
	if err != nil || asof.Height() != 2 {
		t.Fatalf("asof backward: err=%v h=%d", err, asof.Height())
	}
}

func TestFrameEqualsFalseAndCastError(t *testing.T) {
	t.Parallel()

	a := mustFrame(t, SeriesInput{Name: "x", Values: []any{int64(1)}})
	b := mustFrame(t, SeriesInput{Name: "x", Values: []any{int64(2)}})
	eq, err := a.Equals(b)
	if err != nil || eq {
		t.Fatalf("equals false: %v err=%v", eq, err)
	}

	_, err = a.Cast(map[string]dtypes.DataType{"x": dtypes.Boolean})
	if err == nil {
		t.Fatal("cast int->bool без конверсії має повернути помилку")
	}
}

func TestFrameRollingMeanClosedNone(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	df := mustFrame(t,
		SeriesInput{Name: "ts", Values: []any{base, base.Add(time.Hour)}},
		SeriesInput{Name: "v", Values: []any{int64(5), int64(15)}},
	)
	if _, err := df.RollingMean("ts", "v", time.Hour, 1, "rm", "none"); err != nil {
		t.Fatalf("rolling closed=none: %v", err)
	}
}

func TestConcatErrorsAndClearSample(t *testing.T) {
	t.Parallel()

	a := mustFrame(t, SeriesInput{Name: "x", Values: []any{int64(1)}})
	b := mustFrame(t,
		SeriesInput{Name: "x", Values: []any{int64(2)}},
		SeriesInput{Name: "y", Values: []any{int64(3)}},
	)
	if _, err := ConcatVertical(a, b); err == nil {
		t.Fatal("vertical concat має впасти на різній схемі")
	}
	short := mustFrame(t, SeriesInput{Name: "x", Values: []any{int64(1), int64(2)}})
	if _, err := ConcatHorizontal(a, short); err == nil {
		t.Fatal("horizontal concat має впасти на різній висоті")
	}

	cleared := a.Clear()
	if cleared.Height() != 0 || cleared.Width() != 1 {
		t.Fatalf("clear: h=%d w=%d", cleared.Height(), cleared.Width())
	}
	sampled := mustFrame(t,
		SeriesInput{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
	).Sample(2, 42)
	if sampled.Height() != 2 {
		t.Fatalf("sample: %d", sampled.Height())
	}
}
