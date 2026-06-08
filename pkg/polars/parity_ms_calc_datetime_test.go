package polars

// Parity: date/time ranges and timezone conversion (spec: Date/time range and timezone parity).
// Mirrors pl.datetime_range(..., closed="left") and dt.convert_time_zone("Europe/Kyiv").dt.date()
// in ../ms-calculations (mga_balance.py:312, profiling.py:301).

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// hourlyRangeLeftClosed builds a left-closed hourly range [start, end): start included, end excluded.
func hourlyRangeLeftClosed(start, end time.Time) []time.Time {
	var out []time.Time
	for ts := start; ts.Before(end); ts = ts.Add(time.Hour) {
		out = append(out, ts)
	}
	return out
}

// TestParityHourlyRangeLeftClosed pins mga_balance.py:312 — an hourly, left-closed range fed into a
// Datetime column. gopolars has no top-level datetime_range generator (see
// TestParityDatetimeRangeGeneratorUnsupported), so the range is built in Go; this still exercises
// gopolars Datetime column construction and value round-tripping.
func TestParityHourlyRangeLeftClosed(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(4 * time.Hour)
	rng := hourlyRangeLeftClosed(start, end)

	vals := make([]any, len(rng))
	for i, ts := range rng {
		vals[i] = ts
	}
	df := mscFrame(t, mscCol("yd_time", vals...))

	if df.Height() != 4 {
		t.Fatalf("range height = %d, want 4 (00,01,02,03; end excluded)", df.Height())
	}
	col, err := df.GetColumn("yd_time")
	if err != nil {
		t.Fatalf("GetColumn: %v", err)
	}
	first, ok := col.Value(0).(time.Time)
	if !ok || !first.Equal(start) {
		t.Errorf("first = %v, want %v (start included)", col.Value(0), start)
	}
	last, ok := col.Value(3).(time.Time)
	if !ok || !last.Equal(end.Add(-time.Hour)) {
		t.Errorf("last = %v, want %v (end excluded, 1h step)", col.Value(3), end.Add(-time.Hour))
	}
	for _, f := range df.Schema() {
		if f.Name == "yd_time" && f.Type != dtypes.Datetime {
			t.Errorf("dtype = %s, want Datetime", f.Type)
		}
	}
}
