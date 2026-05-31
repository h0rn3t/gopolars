package unit

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestSeriesCountApproxEqualsFilter(t *testing.T) {
	s, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  dtypes.Int64,
		Values: []any{int64(1), nil, int64(2), int64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Count() != 3 {
		t.Fatalf("Count: got %d want 3", s.Count())
	}
	if s.ApproxNUnique() != 2 {
		t.Fatalf("ApproxNUnique: got %d want 2", s.ApproxNUnique())
	}
	s2, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  dtypes.Int64,
		Values: []any{int64(1), nil, int64(2), int64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	eq, err := s.Equals(s2)
	if err != nil || !eq {
		t.Fatalf("Equals: %v %v", eq, err)
	}
	mask, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "m",
		DType:  dtypes.Boolean,
		Values: []any{true, false, true, false},
	})
	f, err := s.Filter(mask)
	if err != nil {
		t.Fatal(err)
	}
	if f.Len() != 2 {
		t.Fatalf("Filter len: %d", f.Len())
	}
}

func TestSeriesTruncateRoundClipCotShrink(t *testing.T) {
	s, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "s",
		DType:  dtypes.String,
		Values: []any{"привіт", "ab"},
	})
	tr, err := s.Truncate(2)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Value(0).(string) != "пр" {
		t.Fatalf("truncate runes: %q", tr.Value(0))
	}
	num, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "n",
		DType:  dtypes.Float64,
		Values: []any{12.345, 0.001234},
	})
	r, err := num.RoundSigFigs(3)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r.Value(0).(float64)-12.3) > 0.05 {
		t.Fatalf("sig figs: %v", r.Value(0))
	}
	cl := num.Clip(0, 1)
	if cl.Value(0).(float64) != 1.0 || cl.Value(1).(float64) != 0.001234 {
		t.Fatalf("clip: %#v %#v", cl.Value(0), cl.Value(1))
	}
	ang, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "a",
		DType:  dtypes.Float64,
		Values: []any{math.Pi / 4},
	})
	cot := ang.Cot()
	if math.Abs(cot.Value(0).(float64)-1) > 1e-9 {
		t.Fatalf("cot: %v", cot.Value(0))
	}
	fint, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "f",
		DType:  dtypes.Float64,
		Values: []any{float64(1), float64(2)},
	})
	sh, err := fint.ShrinkDType()
	if err != nil {
		t.Fatal(err)
	}
	if sh.DataType() != dtypes.Int64 {
		t.Fatalf("shrink dtype: %s", sh.DataType())
	}
}

func TestSeriesAppendGetChunksFlagsBitwiseRollingBy(t *testing.T) {
	a, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: dtypes.Int64, Values: []any{int64(1)}})
	b, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: dtypes.Int64, Values: []any{int64(2)}})
	c, err := a.Append(b)
	if err != nil || c.Len() != 2 {
		t.Fatalf("append: %v len=%d", err, c.Len())
	}
	ch, err := a.GetChunks()
	if err != nil || len(ch) != 1 {
		t.Fatalf("chunks: %v", err)
	}
	an, _ := polars.NewSeries(polars.NewSeriesInput{Name: "n", DType: dtypes.Int64, Values: []any{int64(1), nil}})
	fl := an.Flags()
	if !fl["has_nulls"] {
		t.Fatalf("flags: %#v", fl)
	}
	x, _ := polars.NewSeries(polars.NewSeriesInput{Name: "x", DType: dtypes.Int64, Values: []any{int64(3), int64(5)}})
	y, _ := polars.NewSeries(polars.NewSeriesInput{Name: "y", DType: dtypes.Int64, Values: []any{int64(1), int64(2)}})
	and, err := x.BitwiseAnd(y)
	if err != nil {
		t.Fatal(err)
	}
	if and.Value(0).(int64) != 1 || and.Value(1).(int64) != 0 {
		t.Fatalf("bitwise and: %v %v", and.Value(0), and.Value(1))
	}
	v, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "v",
		DType:  dtypes.Float64,
		Values: []any{1.0, 2.0, 4.0},
	})
	by, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "t",
		DType:  dtypes.Int64,
		Values: []any{int64(0), int64(1), int64(2)},
	})
	rm, err := v.RollingMeanBy(by, 2)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Len() != 3 {
		t.Fatalf("rolling mean by len %d", rm.Len())
	}
}

func TestSeriesCatCodes(t *testing.T) {
	s, _ := polars.NewSeries(polars.NewSeriesInput{
		Name:   "c",
		DType:  dtypes.Categorical,
		Values: []any{"x", "y", "x"},
	})
	codes, err := s.Cat().Codes()
	if err != nil {
		t.Fatal(err)
	}
	if codes.Value(0).(int64) != 0 || codes.Value(1).(int64) != 1 || codes.Value(2).(int64) != 0 {
		t.Fatalf("codes: %v %v %v", codes.Value(0), codes.Value(1), codes.Value(2))
	}
}

func TestSeriesReinterpretError(t *testing.T) {
	s, _ := polars.NewSeries(polars.NewSeriesInput{Name: "a", DType: dtypes.Int64, Values: []any{int64(1)}})
	_, err := s.Reinterpret(dtypes.Float64)
	if err == nil {
		t.Fatal("expected error")
	}
}
