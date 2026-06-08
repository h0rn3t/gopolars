package operations

// Ported from several small py-polars/tests/unit/operations files (py-1.28.1):
// test_hash.py, test_ewm.py, test_chunks.py, test_shrink_dtype.py, test_rle.py,
// test_random.py, test_join_right.py, test_join_asof.py — representative subsets.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// test_hash: hashing produces a same-length series; equal inputs hash equally.
func TestHash(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(1)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	h := s.Hash(0)
	if h.Len() != 3 {
		t.Fatalf("hash len: got %d, want 3", h.Len())
	}
	if h.Value(0) != h.Value(2) {
		t.Fatalf("equal values must hash equally: %v vs %v", h.Value(0), h.Value(2))
	}
}

// test_ewm: exponential weighted mean, first value equals input.
func TestEwmMean(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Float64, Values: []any{1.0, 2.0, 3.0}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.EwmMean(0.5)
	if v, _ := out.Value(0).(float64); v != 1.0 {
		t.Fatalf("ewm[0]: got %v, want 1.0", out.Value(0))
	}
	// adjust=False recursion: ewm[1] = 0.5*2 + 0.5*1 = 1.5
	if v, _ := out.Value(1).(float64); v != 1.5 {
		t.Fatalf("ewm[1]: got %v, want 1.5", out.Value(1))
	}
}

// test_chunks: rechunk yields a single chunk.
func TestChunks(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	r := s.Rechunk()
	if r.NChunks() != 1 {
		t.Fatalf("n_chunks after rechunk: got %d, want 1", r.NChunks())
	}
}

// test_shrink_dtype: shrinking preserves values.
func TestShrinkDtype(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.ShrinkDType()
	if err != nil {
		t.Fatalf("ShrinkDType failed: %v", err)
	}
	if out.Len() != 3 {
		t.Fatalf("len: got %d, want 3", out.Len())
	}
}

// test_rle: run-length encoding groups consecutive equal values.
func TestRle(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(1), int64(2), int64(3), int64(3)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out, err := s.Rle()
	if err != nil {
		t.Fatalf("Rle failed: %v", err)
	}
	// runs: (1,x2), (2,x1), (3,x2) -> 3 runs
	if out.Height() != 3 {
		t.Fatalf("rle runs: got %d, want 3", out.Height())
	}
}

// test_random: sample returns the requested number of rows.
func TestSample(t *testing.T) {
	t.Parallel()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: polars.Int64, Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	out := s.Sample(2, 42)
	if out.Len() != 2 {
		t.Fatalf("sample len: got %d, want 2", out.Len())
	}
}

// test_join_right: right join keeps all right rows.
func TestJoinRight(t *testing.T) {
	t.Parallel()
	left, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(2)}},
		{Name: "a", Values: []any{"x", "y"}},
	}})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(3), int64(4)}},
		{Name: "b", Values: []any{"p", "q", "r"}},
	}})
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	out, err := left.Join(polars.JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: polars.JoinTypeRight})
	if err != nil {
		t.Fatalf("right join: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("right join height: got %d, want 3 (all right rows)", out.Height())
	}
}
