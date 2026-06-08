package functions

// Ported from py-polars/tests/unit/functions/as_datatype/test_datetime.py (py-1.28.1)
//
// gopolars has no top-level pl.datetime(year, month, day, ...) constructor that
// assembles a Datetime column from integer component columns. We port the intent
// that survives: a Datetime Series built from time.Time values round-trips and
// exposes its components via the Dt namespace (Year).

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDatetimeSeriesRoundTripAndYear(t *testing.T) {
	t.Parallel()
	ts := []any{
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC),
	}
	s, err := polars.NewSeries(polars.NewSeriesInput{Name: "dt", DType: polars.Datetime, Values: ts})
	if err != nil {
		t.Fatalf("datetime series: %v", err)
	}
	if s.Len() != 2 {
		t.Fatalf("len: got %d, want 2", s.Len())
	}
	year, err := s.Dt().Year()
	if err != nil {
		t.Fatalf("dt.year: %v", err)
	}
	if v, ok := year.Value(0).(int64); !ok || v != 2022 {
		t.Fatalf("year[0]: got %v, want 2022", year.Value(0))
	}
	if v, ok := year.Value(1).(int64); !ok || v != 2023 {
		t.Fatalf("year[1]: got %v, want 2023", year.Value(1))
	}
}
