package unit

import (
	"encoding/json"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV08DataFrameSurfaceWave(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "b", Values: []any{float64(4), float64(3), float64(2), float64(1)}},
			{Name: "c", Values: []any{float64(1), nil, float64(3), nil}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	if len(df.CollectSchema()) != 3 || len(df.Dtypes()) != 3 {
		t.Fatalf("schema/dtypes mismatch")
	}
	if _, err := df.GetColumn("a"); err != nil {
		t.Fatalf("get_column failed: %v", err)
	}
	if df.GetColumnIndex("b") != 1 || len(df.GetColumns()) != 3 {
		t.Fatalf("get_column_index/get_columns mismatch")
	}
	if len(df.Flags()) == 0 || df.Glimpse(2) == "" {
		t.Fatalf("flags/glimpse mismatch")
	}
	if _, err := df.ApproxNUnique("a"); err != nil {
		t.Fatalf("approx_n_unique failed: %v", err)
	}
	if _, err := df.Corr("a", "b"); err != nil {
		t.Fatalf("corr failed: %v", err)
	}
	if len(df.Count()) != 3 {
		t.Fatalf("count mismatch")
	}
	if _, err := df.Describe(); err != nil {
		t.Fatalf("describe failed: %v", err)
	}
	if _, err := df.HashRows(42); err != nil {
		t.Fatalf("hash_rows failed: %v", err)
	}

	bottom, err := df.BottomK(2, "a")
	if err != nil || bottom.Height() != 2 {
		t.Fatalf("bottom_k failed")
	}
	casted, err := df.Cast(map[string]dtypes.DataType{"a": polars.Float64})
	if err != nil {
		t.Fatalf("cast failed: %v", err)
	}
	cloned := casted.Clone()
	eq, err := cloned.Equals(casted)
	if err != nil || !eq {
		t.Fatalf("equals/clone failed")
	}
	cleared := df.Clear()
	if cleared.Height() != 0 {
		t.Fatalf("clear failed")
	}
	dropped, err := df.DropInPlace("c")
	if err != nil || dropped.Width() != 2 {
		t.Fatalf("drop_in_place failed")
	}
	extended, err := dropped.Extend(dropped)
	if err != nil || extended.Height() != dropped.Height()*2 {
		t.Fatalf("extend failed")
	}
	col, _ := polars.NewSeries(polars.NewSeriesInput{Name: "z", DType: polars.Int64, Values: []any{int64(7), int64(8), int64(9), int64(10)}})
	hstacked, err := df.Hstack(col)
	if err != nil || hstacked.Width() != 4 {
		t.Fatalf("hstack failed")
	}
	inserted, err := df.InsertColumn(1, col)
	if err != nil || inserted.GetColumnIndex("z") != 1 {
		t.Fatalf("insert_column failed")
	}
	gathered := df.GatherEvery(2, 0)
	if gathered.Height() != 2 {
		t.Fatalf("gather_every failed")
	}
	folded, err := df.Fold("sum", []string{"a", "b"}, "ab_sum")
	if err != nil || folded.GetColumnIndex("ab_sum") < 0 {
		t.Fatalf("fold failed")
	}
	interpolated, err := df.Interpolate("c")
	if err != nil {
		t.Fatalf("interpolate failed: %v", err)
	}
	cCol, _ := interpolated.GetColumn("c")
	if cCol.Value(1) == nil || cCol.Value(3) == nil {
		t.Fatalf("interpolate produced nil")
	}

	payload, _ := json.Marshal([]map[string]any{
		{"x": 1.0, "y": "a"},
		{"x": 2.0, "y": "b"},
	})
	deserialized, err := df.Deserialize(payload)
	if err != nil || deserialized.Height() != 2 {
		t.Fatalf("deserialize failed")
	}
}
