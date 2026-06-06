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

// TestParityDatetimeRangeGeneratorUnsupported records the gap: gopolars exposes no top-level
// pl.datetime_range / pl.date_range generator (closed/interval/eager).
func TestParityDatetimeRangeGeneratorUnsupported(t *testing.T) {
	skipGap(t, "datetime_range / date_range",
		"polars generates ranges via pl.datetime_range/date_range(interval, closed, eager); gopolars has no top-level generator")
}

// TestParityTimezoneConvertDate documents the gap for profiling.py:301 —
// dt.convert_time_zone("Europe/Kyiv").dt.date(). The gopolars Dt namespace exposes only Year(); there
// is no convert_time_zone / date. The expected Kyiv-local date is computed via Go's time package as
// the reference the native op would need to match.
func TestParityTimezoneConvertDate(t *testing.T) {
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Skipf("Europe/Kyiv tzdata unavailable: %v", err)
	}
	// 2024-07-01 22:30 UTC is 2024-07-02 01:30 in Kyiv (UTC+3 DST) -> date 2024-07-02.
	utc := time.Date(2024, 7, 1, 22, 30, 0, 0, time.UTC)
	wantDate := utc.In(kyiv).Format("2006-01-02")
	if wantDate != "2024-07-02" {
		t.Fatalf("reference Kyiv date = %s, want 2024-07-02", wantDate)
	}
	skipGap(t, "Series.dt.convert_time_zone(...).dt.date()",
		"polars converts tz then extracts the local date; gopolars Dt namespace has no convert_time_zone/date (expected 2024-07-02)")
}
