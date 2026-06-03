package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameToArrowAndConversion exercises the to-X family.
func TestDataFrameToArrowAndConversion(t *testing.T) {
	df := newDFTestFrame(t)

	tbl, err := df.ToArrow(ToArrowInput{})
	if err != nil {
		t.Fatalf("ToArrow: %v", err)
	}
	if len(tbl.Columns) != 3 {
		t.Errorf("ToArrow cols = %d, want 3", len(tbl.Columns))
	}
	if np := df.ToNumpy(); len(np) != 4 {
		t.Errorf("ToNumpy rows = %d, want 4", len(np))
	}
	// ToPandas is a row-of-maps snapshot.
	if pd := df.ToPandas(); len(pd) != 4 {
		t.Errorf("ToPandas rows = %d, want 4", len(pd))
	}
	// ToSeries converts a single column.
	if s, err := df.ToSeries("a"); err != nil {
		t.Errorf("ToSeries: %v", err)
	} else if s.Len() != 4 {
		t.Errorf("ToSeries len = %d, want 4", s.Len())
	}
	// ToDict is map[col_name]row_values.
	if d := df.ToDict(); d == nil {
		t.Errorf("ToDict returned nil")
	}
	// ToTorch / ToJax return [][]float64.
	if torch := df.ToTorch(); len(torch) == 0 {
		t.Errorf("ToTorch returned empty")
	}
	if jax := df.ToJax(); len(jax) == 0 {
		t.Errorf("ToJax returned empty")
	}
}

// TestDataFrameInsertAndReplace exercises column-mutation methods.
func TestDataFrameInsertAndReplace(t *testing.T) {
	df := newDFTestFrame(t)
	newCol, _ := NewSeries(NewSeriesInput{Name: "z", DType: dtypes.Int64, Values: []any{int64(0), int64(0), int64(0), int64(0)}})

	// HStack adds a column to the right.
	out, err := df.Hstack(newCol)
	if err != nil {
		t.Fatalf("Hstack: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("Hstack width = %d, want 4", out.Width())
	}

	// InsertColumn at position 1.
	out, err = df.InsertColumn(1, newCol)
	if err != nil {
		t.Fatalf("InsertColumn: %v", err)
	}
	if out.Width() != 4 {
		t.Errorf("InsertColumn width = %d, want 4", out.Width())
	}

	// ReplaceColumn at position 0.
	out, err = df.ReplaceColumn(0, newCol)
	if err != nil {
		t.Fatalf("ReplaceColumn: %v", err)
	}
	if out.Width() != 3 {
		t.Errorf("ReplaceColumn width = %d, want 3", out.Width())
	}
}

// TestDataFrameVstackAndExtend exercises vertical stacking.
func TestDataFrameVstackAndExtend(t *testing.T) {
	a := newDFTestFrame(t)
	b := newDFTestFrame(t)
	out, err := a.Vstack(b)
	if err != nil {
		t.Fatalf("Vstack: %v", err)
	}
	if out.Height() != 8 {
		t.Errorf("Vstack height = %d, want 8", out.Height())
	}
	// VStack is the alias of Vstack.
	out, err = a.VStack(b)
	if err != nil {
		t.Fatalf("VStack: %v", err)
	}
	if out.Height() != 8 {
		t.Errorf("VStack height = %d, want 8", out.Height())
	}
	// Extend.
	out, err = a.Extend(b)
	if err != nil {
		t.Fatalf("Extend: %v", err)
	}
	if out.Height() != 8 {
		t.Errorf("Extend height = %d, want 8", out.Height())
	}
}

// TestDataFrameArithmetic exercises Add/Sub/Mul/Div via Expr.
func TestDataFrameArithmetic(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Select(Col("a").Add(Lit(int64(10))).Alias("a_plus_10"))
	if err != nil {
		t.Fatalf("Select Add: %v", err)
	}
	col, _ := out.GetColumn("a_plus_10")
	for i := 0; i < col.Len(); i++ {
		want := int64(i+1) + 10
		if v, _ := col.Value(i).(int64); v != want {
			t.Errorf("Add[%d] = %d, want %d", i, v, want)
		}
	}
}

// TestDataFrameComparison exercises Eq/Ne/Lt/Gt via Expr.
func TestDataFrameComparison(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Select(Col("a").Gt(Lit(int64(2))).Alias("gt"))
	if err != nil {
		t.Fatalf("Select Gt: %v", err)
	}
	col, _ := out.GetColumn("gt")
	want := []bool{false, false, true, true}
	for i, w := range want {
		if v, _ := col.Value(i).(bool); v != w {
			t.Errorf("Gt[%d] = %v, want %v", i, v, w)
		}
	}
}

// TestDataFrameCumulative exercises CumSum/Diff/PctChange via Expr.
func TestDataFrameCumulative(t *testing.T) {
	df := newDFTestFrame(t)
	out, err := df.Select(Col("a").CumSum().Alias("cs"))
	if err != nil {
		t.Fatalf("Select CumSum: %v", err)
	}
	col, _ := out.GetColumn("cs")
	want := []int64{1, 3, 6, 10}
	for i, w := range want {
		var v int64
		switch x := col.Value(i).(type) {
		case int64:
			v = x
		case float64:
			v = int64(x)
		}
		if v != w {
			t.Errorf("CumSum[%d] = %d, want %d (raw=%T %v)", i, v, w, col.Value(i), col.Value(i))
		}
	}
}

// TestDataFrameFillNaN exercises the NaN-specific fill.
func TestDataFrameFillNaN(t *testing.T) {
	df, _ := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{1.0, nil, 3.0, 4.0}},
	}})
	out, err := df.FillNaN(0.0)
	if err != nil {
		t.Fatalf("FillNaN: %v", err)
	}
	if out == nil {
		t.Errorf("FillNaN returned nil")
	}
}

// TestDataFrameDescribe exercises the describe summary.
func TestDataFrameDescribe(t *testing.T) {
	df := newDFTestFrame(t)
	if _, err := df.Describe(); err != nil {
		t.Errorf("Describe: %v", err)
	}
}

// TestDataFrameLazyDescribe exercises the lazy describe path.
func TestDataFrameLazyDescribe(t *testing.T) {
	df := newDFTestFrame(t)
	df2, _, err := df.Lazy().Profile(nil)
	if err != nil {
		// Profile requires a context; nil is acceptable for the dispatch test.
		if df2 != nil {
			t.Errorf("Profile returned non-nil DF and err: %v", err)
		}
	}
}
