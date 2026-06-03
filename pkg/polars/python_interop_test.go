package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestPyArrayDataFrameReturnsRowSnapshot verifies PyArrayDataFrame returns a
// 2D slice matching the frame's rows and values.
func TestPyArrayDataFrameReturnsRowSnapshot(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "b", Values: []any{1.5, 2.5, 3.5}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	got := PyArrayDataFrame(df)
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3", len(got))
	}
	for i, row := range got {
		if len(row) != 2 {
			t.Errorf("row %d width = %d, want 2", i, len(row))
		}
	}
	// Check one cell value to confirm conversion.
	if v, _ := got[0][0].(int64); v != 1 {
		t.Errorf("got[0][0] = %v, want int64(1)", got[0][0])
	}
	if v, _ := got[0][1].(float64); v != 1.5 {
		t.Errorf("got[0][1] = %v, want float64(1.5)", got[0][1])
	}
}

// TestPyArrayDataFrameNilReceiver returns nil (defensive) when df is nil.
func TestPyArrayDataFrameNilReceiver(t *testing.T) {
	if got := PyArrayDataFrame(nil); got != nil {
		t.Fatalf("PyArrayDataFrame(nil) = %v, want nil", got)
	}
}

// TestPyArraySeriesReturnsValues verifies PyArraySeries returns a flat slice
// matching the series length and values.
func TestPyArraySeriesReturnsValues(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "x", DType: dtypes.Int64, Values: []any{int64(10), int64(20), int64(30)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	got := PyArraySeries(s)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	want := []int64{10, 20, 30}
	for i, w := range want {
		if v, _ := got[i].(int64); v != w {
			t.Errorf("got[%d] = %v, want %d", i, got[i], w)
		}
	}
}

// TestPyArraySeriesNilReceiver returns nil (defensive) when s is nil.
func TestPyArraySeriesNilReceiver(t *testing.T) {
	if got := PyArraySeries(nil); got != nil {
		t.Fatalf("PyArraySeries(nil) = %v, want nil", got)
	}
}

// TestPyArrowTableDataFrameSchema verifies the returned arrow table's columns
// match the frame's columns and that each column has the expected row count.
func TestPyArrowTableDataFrameSchema(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	tbl, err := PyArrowTableDataFrame(df)
	if err != nil {
		t.Fatalf("PyArrowTableDataFrame: %v", err)
	}
	if len(tbl.Columns) != 2 {
		t.Errorf("len(Columns) = %d, want 2", len(tbl.Columns))
	}
	if rows := len(tbl.Columns["a"]); rows != 2 {
		t.Errorf("rows in column a = %d, want 2", rows)
	}
	if rows := len(tbl.Columns["b"]); rows != 2 {
		t.Errorf("rows in column b = %d, want 2", rows)
	}
}

// TestPyArrowTableDataFrameNil returns an error (not a panic) when df is nil.
func TestPyArrowTableDataFrameNil(t *testing.T) {
	_, err := PyArrowTableDataFrame(nil)
	if err == nil {
		t.Fatalf("PyArrowTableDataFrame(nil) returned nil error, want non-nil")
	}
}

// TestPyArrowTableSeriesSchema verifies the returned arrow table's column for
// a single series.
func TestPyArrowTableSeriesSchema(t *testing.T) {
	s, err := NewSeries(NewSeriesInput{Name: "x", DType: dtypes.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("new series: %v", err)
	}
	tbl, err := PyArrowTableSeries(s)
	if err != nil {
		t.Fatalf("PyArrowTableSeries: %v", err)
	}
	if len(tbl.Columns) != 1 {
		t.Errorf("len(Columns) = %d, want 1", len(tbl.Columns))
	}
	if rows := len(tbl.Columns["x"]); rows != 3 {
		t.Errorf("rows in column x = %d, want 3", rows)
	}
}

// TestPyArrowTableSeriesNil returns an error (not a panic) when s is nil.
func TestPyArrowTableSeriesNil(t *testing.T) {
	_, err := PyArrowTableSeries(nil)
	if err == nil {
		t.Fatalf("PyArrowTableSeries(nil) returned nil error, want non-nil")
	}
}

// TestPyDataFrameInterchangeReturnsDF verifies PyDataFrameInterchange returns
// a non-nil DataFrame with the same shape.
func TestPyDataFrameInterchangeReturnsDF(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	out := PyDataFrameInterchange(df)
	if out == nil {
		t.Fatalf("PyDataFrameInterchange returned nil")
	}
	if out.Height() != 2 || out.Width() != 1 {
		t.Errorf("interchange shape = (%d, %d), want (2, 1)", out.Height(), out.Width())
	}
}

// TestErrPySetItemNotSupported is exported as a sentinel for callers that
// intentionally compare against the documented error.
func TestErrPySetItemNotSupported(t *testing.T) {
	if ErrPySetItemNotSupported == nil {
		t.Fatalf("ErrPySetItemNotSupported is nil")
	}
	if ErrPySetItemNotSupported.Error() == "" {
		t.Errorf("ErrPySetItemNotSupported has empty message")
	}
}
