package datatypes

// Ported from py-polars/tests/unit/datatypes/test_temporal.py (py-1.28.1, representative subset)
//
// gopolars exposes a single Datetime dtype (no separate Date/Time/Duration). We
// port the behaviors that survive: Datetime round-trip, ordering, and the Dt
// namespace Year() extraction.

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func newDatetimeSeries(t *testing.T, name string, vals []any) polars.Series {
	t.Helper()
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: name, DType: polars.Datetime, Values: vals})
	if err != nil {
		t.Fatalf("new datetime series: %v", err)
	}
	return s
}

func TestTemporalDatetimeRoundTrip(t *testing.T) {
	t.Parallel()
	want := time.Date(2023, 5, 17, 8, 30, 0, 0, time.UTC)
	s := newDatetimeSeries(t, "t", []any{want})
	got, ok := s.Value(0).(time.Time)
	if !ok || !got.Equal(want) {
		t.Fatalf("value[0]: got %v, want %v", s.Value(0), want)
	}
	if s.DataType() != polars.Datetime {
		t.Fatalf("dtype: got %v, want Datetime", s.DataType())
	}
}

func TestTemporalYearExtraction(t *testing.T) {
	t.Parallel()
	s := newDatetimeSeries(t, "t", []any{
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	year, err := s.Dt().Year()
	if err != nil {
		t.Fatalf("dt.year: %v", err)
	}
	for i, w := range []int64{2020, 2021, 2022} {
		if v, ok := year.Value(i).(int64); !ok || v != w {
			t.Fatalf("year[%d]: got %v, want %d", i, year.Value(i), w)
		}
	}
}

func TestTemporalSort(t *testing.T) {
	t.Parallel()
	s := newDatetimeSeries(t, "t", []any{
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	sorted := s.Sort(false)
	first, _ := sorted.Value(0).(time.Time)
	if first.Year() != 2020 {
		t.Fatalf("sorted first year: got %d, want 2020", first.Year())
	}
}
