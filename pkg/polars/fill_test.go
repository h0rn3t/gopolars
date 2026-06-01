package polars

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestFillNullTypedSemantics(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, nil, math.NaN(), nil, 5.0})
	out, err := s.FillNull(float64(0))
	if err != nil {
		t.Fatalf("fill_null: %v", err)
	}
	// nulls filled with 0; NaN (a value, not null) left as NaN.
	want := []float64{1, 0, math.NaN(), 0, 5}
	for i, w := range want {
		got := out.Value(i).(float64)
		if math.IsNaN(w) {
			if !math.IsNaN(got) {
				t.Errorf("fill_null[%d] = %v, want NaN", i, got)
			}
			continue
		}
		if got != w {
			t.Errorf("fill_null[%d] = %v, want %v", i, got, w)
		}
	}
	if out.NullCount() != 0 {
		t.Errorf("fill_null result should have no nulls, got %d", out.NullCount())
	}
}

func TestFillNanTypedSemantics(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, nil, math.NaN(), 4.0})
	out, err := s.FillNan(7)
	if err != nil {
		t.Fatalf("fill_nan: %v", err)
	}
	// NaN -> 7; null preserved as null.
	if out.Value(2).(float64) != 7 {
		t.Errorf("fill_nan should replace NaN with 7, got %v", out.Value(2))
	}
	if out.NullCount() != 1 {
		t.Errorf("fill_nan must preserve the null, got NullCount %d", out.NullCount())
	}
}

func TestDropNansTypedSemantics(t *testing.T) {
	s := mkFloatSeries(t, []any{1.0, math.NaN(), nil, 4.0})
	out := s.DropNans()
	// NaN row dropped; null row kept.
	if out.Len() != 3 {
		t.Fatalf("drop_nans len = %d, want 3", out.Len())
	}
	if out.NullCount() != 1 {
		t.Errorf("drop_nans should keep the null row, got NullCount %d", out.NullCount())
	}
}

func mkFillBenchSeries(b *testing.B) Series {
	vals := make([]any, 1_000_000)
	for i := range vals {
		switch i % 3 {
		case 0:
			vals[i] = nil
		case 1:
			vals[i] = math.NaN()
		default:
			vals[i] = float64(i)
		}
	}
	s, err := NewSeries(NewSeriesInput{Name: "v", DType: dtypes.Float64, Values: vals})
	if err != nil {
		b.Fatal(err)
	}
	return s
}

func BenchmarkFillNullSeries(b *testing.B) {
	s := mkFillBenchSeries(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.FillNull(float64(0)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFillNanSeries(b *testing.B) {
	s := mkFillBenchSeries(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.FillNan(0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDropNansSeries(b *testing.B) {
	s := mkFillBenchSeries(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.DropNans()
	}
}
