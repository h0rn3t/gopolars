package polars

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// covTimeFrame builds a frame with a datetime "ts" column and float "v" column.
func covTimeFrame(t *testing.T) DataFrame {
	t.Helper()
	base := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "ts", DType: dtypes.Datetime, Values: []any{
			base,
			base.Add(1 * time.Hour),
			base.Add(2 * time.Hour),
			base.Add(3 * time.Hour),
		}},
		{Name: "v", Values: []any{1.0, 2.0, 3.0, 4.0}},
	}})
	if err != nil {
		t.Fatalf("covTimeFrame: %v", err)
	}
	return d
}

// TestDataFrameRolling covers df.Rolling -> RollingMean over a datetime key.
func TestDataFrameRolling(t *testing.T) {
	d := covTimeFrame(t)
	out, err := d.Rolling("ts", "v", 90*time.Minute, "roll")
	if err != nil {
		t.Fatalf("Rolling: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Rolling height = %d, want 4", out.Height())
	}
	if _, ok := out.Series("roll"); !ok {
		t.Errorf("Rolling did not produce output column 'roll'")
	}
}

// TestDataFrameRollingBadKeyError covers the error path (non-datetime by).
func TestDataFrameRollingBadKeyError(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2)}},
		{Name: "v", Values: []any{1.0, 2.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if _, err := d.Rolling("k", "v", time.Hour, "roll"); err == nil {
		t.Fatalf("Rolling on non-datetime key returned nil error, want non-nil")
	}
}

// TestDataFrameJoinAsof covers JoinAsof (defaults How to "asof").
func TestDataFrameJoinAsof(t *testing.T) {
	left, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(5), int64(10)}},
		{Name: "lv", Values: []any{1.0, 5.0, 10.0}},
	}})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(6)}},
		{Name: "rv", Values: []any{100.0, 600.0}},
	}})
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	out, err := left.JoinAsof(JoinInput{
		Other:   right,
		LeftOn:  []string{"k"},
		RightOn: []string{"k"},
	})
	if err != nil {
		t.Fatalf("JoinAsof: %v", err)
	}
	if out.Height() != 3 {
		t.Errorf("JoinAsof height = %d, want 3 (all left rows)", out.Height())
	}
}

// TestDataFrameUpsample covers Upsample filling gaps on a datetime grid.
func TestDataFrameUpsample(t *testing.T) {
	base := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "ts", DType: dtypes.Datetime, Values: []any{
			base,
			base.Add(3 * time.Hour),
		}},
		{Name: "v", Values: []any{1.0, 4.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.Upsample("ts", time.Hour)
	if err != nil {
		t.Fatalf("Upsample: %v", err)
	}
	// grid: 0,1,2,3 hours -> 4 rows.
	if out.Height() != 4 {
		t.Errorf("Upsample height = %d, want 4", out.Height())
	}
}

// TestDataFrameUpsampleNonPositiveEvery covers the early-return branch.
func TestDataFrameUpsampleNonPositiveEvery(t *testing.T) {
	d := covTimeFrame(t)
	out, err := d.Upsample("ts", 0)
	if err != nil {
		t.Fatalf("Upsample: %v", err)
	}
	if out.Height() != d.Height() {
		t.Errorf("Upsample(every=0) height = %d, want %d (unchanged)", out.Height(), d.Height())
	}
}

// TestDataFrameUpsampleNonDatetimeError covers the type-check error.
func TestDataFrameUpsampleNonDatetimeError(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2)}},
		{Name: "v", Values: []any{1.0, 2.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if _, err := d.Upsample("k", time.Hour); err == nil {
		t.Fatalf("Upsample on non-datetime column returned nil error, want non-nil")
	}
}

// TestDataFramePlot covers the Plot stub.
func TestDataFramePlot(t *testing.T) {
	d := covTimeFrame(t)
	out := d.Plot("ts", "v")
	if out["type"] != "line" {
		t.Errorf("Plot type = %v, want line", out["type"])
	}
	if out["x"] != "ts" || out["y"] != "v" {
		t.Errorf("Plot x/y = %v/%v, want ts/v", out["x"], out["y"])
	}
	if out["rows"].(int) != 4 {
		t.Errorf("Plot rows = %v, want 4", out["rows"])
	}
}

// TestDataFrameStyle covers Style -> ToInitRepr.
func TestDataFrameStyle(t *testing.T) {
	d := covTimeFrame(t)
	s := d.Style()
	if s == "" {
		t.Errorf("Style returned empty string")
	}
}

// TestDataFrameTranspose covers Transpose of a numeric (int) frame.
func TestDataFrameTransposeInt(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	// 2x2 -> 2 columns (one per original row).
	if out.Width() != 2 {
		t.Errorf("Transpose width = %d, want 2", out.Width())
	}
	if out.Height() != 2 {
		t.Errorf("Transpose height = %d, want 2", out.Height())
	}
}

// TestDataFrameTransposeFloat covers the anyFloat supertype branch.
func TestDataFrameTransposeFloat(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{3.0, 4.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	if out.Width() != 2 {
		t.Errorf("Transpose width = %d, want 2", out.Width())
	}
}

// TestDataFrameTransposeString covers the stringify supertype branch.
func TestDataFrameTransposeString(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	col, err := out.GetColumn("column_0")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}
	if col.DataType() != dtypes.String {
		t.Errorf("Transpose mixed-type column dtype = %s, want String", col.DataType())
	}
}

// TestDataFrameTransposeEmpty covers the empty frame early return.
func TestDataFrameTransposeEmpty(t *testing.T) {
	d := covTimeFrame(t).Limit(0)
	out, err := d.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	if out.Height() != 0 {
		t.Errorf("Transpose empty height = %d, want 0", out.Height())
	}
}

// TestDataFrameToDummiesSubset covers ToDummies on a subset, exercising
// anySlice for the passthrough column.
func TestDataFrameToDummiesSubset(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "color", Values: []any{"red", "blue", "red"}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.ToDummies("color")
	if err != nil {
		t.Fatalf("ToDummies: %v", err)
	}
	// id passthrough + color_blue + color_red.
	if _, ok := out.Series("id"); !ok {
		t.Errorf("ToDummies dropped passthrough 'id' column")
	}
	if _, ok := out.Series("color_red"); !ok {
		t.Errorf("ToDummies missing color_red")
	}
	if _, ok := out.Series("color_blue"); !ok {
		t.Errorf("ToDummies missing color_blue")
	}
}

// TestDataFrameTopK covers TopK (sort descending then limit).
func TestDataFrameTopK(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "v", Values: []any{int64(3), int64(1), int64(5), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.TopK(2, "v")
	if err != nil {
		t.Fatalf("TopK: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("TopK height = %d, want 2", out.Height())
	}
	col, _ := out.GetColumn("v")
	if v, _ := col.Value(0).(int64); v != 5 {
		t.Errorf("TopK[0] = %v, want 5", col.Value(0))
	}
}

// TestDataFrameHorizontalAggs covers Sum/Max/Min/MeanHorizontal.
func TestDataFrameHorizontalAggs(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{1.0, 4.0}},
		{Name: "b", Values: []any{3.0, 2.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}

	sum, err := d.SumHorizontal("s")
	if err != nil {
		t.Fatalf("SumHorizontal: %v", err)
	}
	col, _ := sum.GetColumn("s")
	if v, _ := col.Value(0).(float64); v != 4.0 {
		t.Errorf("SumHorizontal[0] = %v, want 4", col.Value(0))
	}

	mx, err := d.MaxHorizontal("mx")
	if err != nil {
		t.Fatalf("MaxHorizontal: %v", err)
	}
	col, _ = mx.GetColumn("mx")
	if v, _ := col.Value(1).(float64); v != 4.0 {
		t.Errorf("MaxHorizontal[1] = %v, want 4", col.Value(1))
	}

	mn, err := d.MinHorizontal("mn")
	if err != nil {
		t.Fatalf("MinHorizontal: %v", err)
	}
	col, _ = mn.GetColumn("mn")
	if v, _ := col.Value(0).(float64); v != 1.0 {
		t.Errorf("MinHorizontal[0] = %v, want 1", col.Value(0))
	}

	mean, err := d.MeanHorizontal("mean")
	if err != nil {
		t.Fatalf("MeanHorizontal: %v", err)
	}
	col, _ = mean.GetColumn("mean")
	if v, _ := col.Value(0).(float64); v != 2.0 {
		t.Errorf("MeanHorizontal[0] = %v, want 2", col.Value(0))
	}
}

// TestDataFrameHorizontalAggDefaultAlias covers the empty-alias default branch.
func TestDataFrameHorizontalAggDefaultAlias(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{1.0}},
		{Name: "b", Values: []any{2.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.SumHorizontal("")
	if err != nil {
		t.Fatalf("SumHorizontal: %v", err)
	}
	if _, ok := out.Series("sum_horizontal"); !ok {
		t.Errorf("default alias 'sum_horizontal' not present")
	}
}

// TestDataFrameToJaxTorch covers ToJax/ToTorch numeric extraction and the
// non-numeric fallback to 0.
func TestDataFrameToJaxTorch(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}}, // non-numeric -> 0
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	jx := d.ToJax()
	if len(jx) != 2 || len(jx[0]) != 2 {
		t.Fatalf("ToJax shape = %dx? want 2x2", len(jx))
	}
	if jx[0][0] != 1.0 {
		t.Errorf("ToJax[0][0] = %v, want 1", jx[0][0])
	}
	if jx[0][1] != 0.0 {
		t.Errorf("ToJax[0][1] = %v, want 0 (non-numeric fallback)", jx[0][1])
	}
	tt := d.ToTorch()
	if len(tt) != 2 {
		t.Errorf("ToTorch len = %d, want 2", len(tt))
	}
}

// TestDataFramePivot covers Pivot with a value rename (ValueName branch).
func TestDataFramePivot(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "idx", Values: []any{"a", "a", "b"}},
		{Name: "key", Values: []any{"x", "y", "x"}},
		{Name: "val", Values: []any{1.0, 2.0, 3.0}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.Pivot(PivotInput{Index: "idx", Columns: "key", Values: "val", Agg: "sum"})
	if err != nil {
		t.Fatalf("Pivot: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("Pivot height = %d, want 2 (idx a,b)", out.Height())
	}
}

// TestDataFrameMapColumns covers MapColumns transform path.
func TestDataFrameMapColumns(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.MapColumns(func(name string, s Series) (Series, error) {
		return s.Rename(name + "_x"), nil
	})
	if err != nil {
		t.Fatalf("MapColumns: %v", err)
	}
	if _, ok := out.Series("a_x"); !ok {
		t.Errorf("MapColumns did not rename to a_x")
	}
}

// TestDataFrameMapRows covers MapRows transform path.
func TestDataFrameMapRows(t *testing.T) {
	d, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	out, err := d.MapRows(func(row map[string]any) (map[string]any, error) {
		row["a"] = row["a"].(int64) * 10
		return row, nil
	})
	if err != nil {
		t.Fatalf("MapRows: %v", err)
	}
	col, _ := out.GetColumn("a")
	if v, _ := col.Value(0).(int64); v != 10 {
		t.Errorf("MapRows[0] = %v, want 10", col.Value(0))
	}
}

// TestDataFramePipe covers the Pipe wrapper including the nil-fn branch.
func TestDataFramePipe(t *testing.T) {
	d := covTimeFrame(t)
	out, err := d.Pipe(func(in DataFrame) (DataFrame, error) {
		return in.Limit(1), nil
	})
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	if out.Height() != 1 {
		t.Errorf("Pipe height = %d, want 1", out.Height())
	}

	out2, err := d.Pipe(nil)
	if err != nil {
		t.Fatalf("Pipe(nil): %v", err)
	}
	if out2.Height() != d.Height() {
		t.Errorf("Pipe(nil) height = %d, want %d", out2.Height(), d.Height())
	}
}
