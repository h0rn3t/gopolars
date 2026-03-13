package unit

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestV09WaveDComputeAndDataFrameLowMethods(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "g", Values: []any{"a", "a", "b"}},
			{Name: "x", Values: []any{float64(1), float64(2), float64(3)}},
			{Name: "ts", Values: []any{
				time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
			}},
		},
	})
	if err != nil {
		t.Fatalf("new df failed: %v", err)
	}
	if _, err := df.JoinWhere(polars.Col("x").Gt(polars.Lit(float64(1)))); err != nil {
		t.Fatalf("join_where failed: %v", err)
	}
	if _, err := df.MapColumns(func(name string, s polars.Series) (polars.Series, error) { return s, nil }); err != nil {
		t.Fatalf("map_columns failed: %v", err)
	}
	if _, err := df.MapRows(func(row map[string]any) (map[string]any, error) { return row, nil }); err != nil {
		t.Fatalf("map_rows failed: %v", err)
	}
	col, _ := polars.NewSeries(polars.NewSeriesInput{Name: "x", DType: polars.Float64, Values: []any{float64(3), float64(2), float64(1)}})
	if _, err := df.ReplaceColumn(1, col); err != nil {
		t.Fatalf("replace_column failed: %v", err)
	}
	if df.Reverse().Height() != df.Height() {
		t.Fatalf("reverse mismatch")
	}
	if _, err := df.Rolling("ts", "x", time.Hour, "rx"); err != nil {
		t.Fatalf("rolling failed: %v", err)
	}
	if _, err := df.Row(1); err != nil || len(df.Rows()) != df.Height() {
		t.Fatalf("row/rows failed")
	}
	if len(df.RowsByKey("g")) == 0 {
		t.Fatalf("rows_by_key failed")
	}
	if _, err := df.SelectSeq(polars.Col("x")); err != nil {
		t.Fatalf("select_seq failed: %v", err)
	}
	if _, err := df.Serialize(); err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if _, err := df.SetSorted("x"); err != nil {
		t.Fatalf("set_sorted failed: %v", err)
	}
	if df.Shape()[0] != df.Height() {
		t.Fatalf("shape failed")
	}
	if _, err := df.Shift(1); err != nil || df.Show(2) == "" || df.ShrinkToFit().Height() != df.Height() {
		t.Fatalf("shift/show/shrink failed")
	}
	if _, err := df.SQL(context.Background(), "SELECT * FROM self"); err != nil {
		t.Fatalf("sql failed: %v", err)
	}
	if _, err := df.Sql(context.Background(), "SELECT * FROM self"); err != nil || df.Style() == "" {
		t.Fatalf("sql/style alias failed")
	}
	if len(df.Std()) == 0 || len(df.Sum()) == 0 {
		t.Fatalf("std/sum failed")
	}
	if len(df.Var()) == 0 {
		t.Fatalf("var failed")
	}
	if _, err := df.SumHorizontal("sumh"); err != nil {
		t.Fatalf("sum_horizontal failed: %v", err)
	}
	if _, err := df.ToSeries("x"); err != nil || len(df.ToStruct()) == 0 || len(df.ToTorch()) != df.Height() {
		t.Fatalf("to_series/to_struct/to_torch failed")
	}
	if _, err := df.TopK(2, "x"); err != nil {
		t.Fatalf("top_k failed: %v", err)
	}
	if _, err := df.Transpose(); err != nil {
		t.Fatalf("transpose failed: %v", err)
	}
	if _, err := df.Unpivot(polars.MeltInput{IDVars: []string{"g"}, ValueVars: []string{"x"}, VariableCol: "var", ValueCol: "val"}); err != nil {
		t.Fatalf("unpivot failed: %v", err)
	}
	if _, err := df.Unstack("g"); err != nil {
		t.Fatalf("unstack failed: %v", err)
	}
	if _, err := df.Unnest("g"); err != nil {
		t.Fatalf("unnest failed: %v", err)
	}
	if _, err := df.VStack(df); err != nil {
		t.Fatalf("vstack failed: %v", err)
	}
	if _, err := df.Update(df); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if _, err := df.WithColumnsSeq(polars.Col("x").Alias("x2")); err != nil {
		t.Fatalf("with_columns_seq failed: %v", err)
	}
	if len(df.ToDict()) == 0 {
		t.Fatalf("to_dict failed")
	}
	if _, err := df.ToDummies("g"); err != nil {
		t.Fatalf("to_dummies failed: %v", err)
	}
	if df.ToInitRepr() == "" || len(df.ToJax()) != df.Height() {
		t.Fatalf("to_init_repr/to_jax failed")
	}
	if err := df.WriteCsv(polars.WriteCSVInput{Path: "/tmp/gopolars_wave_d.csv", IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("write_csv alias failed: %v", err)
	}
	if err := df.WriteJson(polars.WriteJSONInput{Path: "/tmp/gopolars_wave_d.json", Pretty: false}); err != nil {
		t.Fatalf("write_json alias failed: %v", err)
	}
	if err := df.WriteIpc(polars.WriteIPCInput{Path: "/tmp/gopolars_wave_d.ipc"}); err != nil {
		t.Fatalf("write_ipc alias failed: %v", err)
	}
	if err := df.WriteIpcStream(polars.WriteIPCInput{Path: "/tmp/gopolars_wave_d_stream.ipc"}); err != nil {
		t.Fatalf("write_ipc_stream failed: %v", err)
	}
	if err := df.WriteNdjson(polars.WriteJSONInput{Path: "/tmp/gopolars_wave_d.ndjson", NDJSON: true}); err != nil {
		t.Fatalf("write_ndjson failed: %v", err)
	}

	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "s", DType: polars.Float64, Values: []any{float64(-1), float64(0), float64(4)}})
	if err != nil {
		t.Fatalf("new series failed: %v", err)
	}
	if s.Abs().Len() != s.Len() || s.Exp().Len() != s.Len() || s.Log().Len() != s.Len() || s.Sqrt().Len() != s.Len() {
		t.Fatalf("series unary methods failed")
	}
	if s.Shift(1).Len() != s.Len() || s.Reverse().Len() != s.Len() {
		t.Fatalf("series shift/reverse failed")
	}
	if s.Sum() == 0 || s.Std() <= 0 {
		t.Fatalf("series sum/std failed")
	}

	out, err := df.Select(polars.Col("x").Abs().Alias("absx"), polars.Col("x").Exp().Alias("expx"))
	if err != nil {
		t.Fatalf("expr abs/exp failed: %v", err)
	}
	c, _ := out.GetColumn("absx")
	if math.Abs(c.Value(0).(float64)-1) > 1e-9 {
		t.Fatalf("expr abs value mismatch")
	}
}
