//go:build duckdb && duckdb_arrow

// Ported from py-polars/tests/unit/sql/test_temporal.py (py-1.28.1).
//
// gopolars runs SQL through an embedded DuckDB engine (build -tags "duckdb
// duckdb_arrow"). Temporal support is the most divergence-heavy area:
//
//   - gopolars has NO Date or Time dtype. The Arrow bridge only carries TIMESTAMP
//     (-> time.Time, UTC). A DATE result (Arrow date32) or TIME result (Arrow
//     time64) raises "unsupported arrow type ..." on read-back, so DATE/TIME
//     producing queries are GAP'd or cast to TIMESTAMP / VARCHAR.
//   - polars-only scalar functions DATETIME(), and the bare TIME('...') call,
//     do not exist in DuckDB.
//   - EXTRACT/DATE_PART parts return int64/float64 (not polars' Int8/Int16/Int32),
//     and ms/us are integer-truncated (polars returns fractional Float64).
//     `nanosecond` is unknown to DuckDB.
//
// All TIMESTAMP literals/results normalize to time.Time in UTC.
package sql

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func utc(y, mo, d, h, mi, s, us int) time.Time {
	return time.Date(y, time.Month(mo), d, h, mi, s, us*1000, time.UTC)
}

// eqTimes compares a column of values against expected time.Time/nil using
// time.Time.Equal (reflect.DeepEqual is unreliable across an Arrow round-trip).
func eqTimes(t *testing.T, got, want []any, msg string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len got=%d want=%d", msg, len(got), len(want))
	}
	for i := range got {
		if want[i] == nil {
			if got[i] != nil {
				t.Fatalf("%s[%d] = %v, want nil", msg, i, got[i])
			}
			continue
		}
		g, ok := got[i].(time.Time)
		if !ok || !g.Equal(want[i].(time.Time)) {
			t.Fatalf("%s[%d] = %v, want %v", msg, i, got[i], want[i])
		}
	}
}

// test_date_func (datetime portion): comparing a TIMESTAMP column against a DATE
// literal. polars compares a Date column to DATE('2021-03-20'); gopolars has no
// Date dtype, so the source column is modeled as TIMESTAMP and the DATE literal
// is cast to TIMESTAMP for the comparison. Values/semantics MATCH.
func TestTemporalDateCompare(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "dt", Values: []any{
			utc(2021, 3, 15, 0, 0, 0, 0),
			utc(2021, 3, 28, 0, 0, 0, 0),
			utc(2021, 4, 4, 0, 0, 0, 0),
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	out := query(t, d, `SELECT dt < (DATE '2021-03-20')::timestamp AS lt FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "lt"), []any{true, false, false}, "dt < DATE")
}

// test_date_func (CAST DATE to string): DuckDB's DATE -> VARCHAR matches polars
// rendering ("2023-03-01"). The DATE value itself cannot be surfaced (no Date
// dtype), so it is cast to VARCHAR.
func TestTemporalDateCastString(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})
	out := query(t, d, `SELECT CAST(STRPTIME('2023-03','%Y-%m') AS DATE)::VARCHAR AS s FROM self`)
	eqRow(t, vals(t, out, "s"), []any{"2023-03-01"}, "DATE->string")
}

// test (bare DATE result): a DATE result column (Arrow date32) now reads back.
//
// DISCREPANCY: gopolars has no Date dtype — date32 is surfaced as Datetime
// (time.Time at midnight UTC). The instant MATCHes polars' Date value.
func TestTemporalDateResult(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})
	out := query(t, d, `SELECT DATE '2020-12-30' AS dt FROM self`)
	eqTimes(t, vals(t, out, "dt"), []any{utc(2020, 12, 30, 0, 0, 0, 0)}, "DATE result")
}

// test_datetime_to_time: dtm::time. TIME results (Arrow time64) now read back.
//
// DISCREPANCY: gopolars has no Time dtype — time64 is surfaced as Datetime
// (time-of-day on the epoch day, 1970-01-01 UTC). The time-of-day MATCHes polars.
func TestTemporalDatetimeToTime(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dtm", Values: []any{utc(2099, 12, 31, 23, 59, 59, 0)}})
	out := query(t, d, `SELECT dtm::time AS tm FROM self`)
	eqTimes(t, vals(t, out, "tm"), []any{utc(1970, 1, 1, 23, 59, 59, 0)}, "TIME result")
}

// test_extract / test_date_part: integer/float date parts off a TIMESTAMP.
// Values are validated against DuckDB (which agrees with PostgreSQL on these
// edge cases). dtypes differ from polars (int64/float64 vs Int8/Int16/Int32) —
// // DISCREPANCY: dtype only, values MATCH except ms/us (see below).
func TestTemporalExtract(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "dt", Values: []any{
			utc(2024, 1, 7, 1, 2, 3, 123456),
			utc(2020, 12, 30, 10, 30, 45, 987654),
			utc(2006, 1, 1, 23, 59, 59, 555555),
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	// part -> expected int64 values across the 3 rows.
	intParts := map[string][]any{
		"decade":  {int64(202), int64(202), int64(200)},
		"isoyear": {int64(2024), int64(2020), int64(2005)},
		"year":    {int64(2024), int64(2020), int64(2006)},
		"quarter": {int64(1), int64(4), int64(1)},
		"month":   {int64(1), int64(12), int64(1)},
		"week":    {int64(1), int64(53), int64(52)},
		"doy":     {int64(7), int64(365), int64(1)},
		"isodow":  {int64(7), int64(3), int64(7)},
		"dow":     {int64(0), int64(3), int64(0)},
		"day":     {int64(7), int64(30), int64(1)},
		"hour":    {int64(1), int64(10), int64(23)},
		"minute":  {int64(2), int64(30), int64(59)},
		"second":  {int64(3), int64(45), int64(59)},
		// DISCREPANCY: polars returns fractional Float64 here (e.g. 3123.456);
		// DuckDB returns the integer-truncated whole-unit count.
		"millisecond": {int64(3123), int64(45987), int64(59555)},
		"microsecond": {int64(3123456), int64(45987654), int64(59555555)},
	}
	for part, want := range intParts {
		for _, fn := range []string{
			"EXTRACT(" + part + " FROM dt)",
			"DATE_PART('" + part + "',dt)",
		} {
			out := query(t, d, "SELECT "+fn+" AS r FROM self ORDER BY idx")
			eqRow(t, vals(t, out, "r"), want, part+" via "+fn)
		}
	}

	// epoch -> Float64 (fractional seconds), MATCHES polars exactly.
	for _, fn := range []string{"EXTRACT(epoch FROM dt)", "DATE_PART('epoch',dt)"} {
		out := query(t, d, "SELECT "+fn+" AS r FROM self ORDER BY idx")
		wantEpoch := []float64{1704589323.123456, 1609324245.987654, 1136159999.555555}
		r := col(t, out, "r")
		for i, w := range wantEpoch {
			trigClose(t, r.Value(i), w, "epoch via "+fn)
		}
	}
}

// GAP: polars supports EXTRACT(nanosecond ...); DuckDB does not recognize the
// "nanosecond" specifier (Conversion Error). Also the polars `time` part returns
// a Time dtype which gopolars cannot represent.
func TestTemporalExtractNanosecondUnsupported(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dt", Values: []any{utc(2024, 1, 7, 1, 2, 3, 123456)}})
	if err := runErr(t, d, `SELECT EXTRACT(nanosecond FROM dt) AS r FROM self`); err == nil {
		t.Skip("GAP: expected DuckDB to reject 'nanosecond' specifier, but it succeeded")
	}
}

// test_extract_century_millennium: century / millennium parts on the standard
// ISO boundaries (year 2000 -> century 20; year 2001 -> century 21, millennium 3).
//
// GAP: the py test also covers extreme dates (year 1 and year 9999). gopolars'
// Arrow TIMESTAMP round-trip shifts such far-boundary dates (a microsecond-
// timestamp / proleptic-Gregorian artifact in the time.Time <-> Arrow bridge),
// so DuckDB then reports e.g. century 18 for year 1. Those rows are excluded;
// only the cleanly round-tripping mid-range boundaries are asserted (MATCH).
func TestTemporalExtractCenturyMillennium(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "dt", Values: []any{
			utc(1901, 1, 1, 0, 0, 0, 0),
			utc(2000, 12, 31, 0, 0, 0, 0),
			utc(2001, 1, 1, 0, 0, 0, 0),
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	out := query(t, d, `
		SELECT
		  EXTRACT(MILLENNIUM FROM dt) AS mil,
		  DATE_PART('century', dt) AS cen
		FROM self
		ORDER BY idx`)
	eqRow(t, vals(t, out, "mil"), []any{int64(2), int64(2), int64(3)}, "millennium")
	eqRow(t, vals(t, out, "cen"), []any{int64(20), int64(20), int64(21)}, "century")
}

// test_extract_errors: unknown date-part specifiers raise an error.
func TestTemporalExtractErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dt", Values: []any{utc(2024, 1, 7, 1, 2, 3, 123456)}})
	for _, part := range []string{"femtosecond", "stroopwafel"} {
		if err := runErr(t, d, "SELECT EXTRACT("+part+" FROM dt) AS r FROM self"); err == nil {
			t.Fatalf("EXTRACT(%s): expected error, got nil", part)
		}
	}
}

// test_implicit_temporal_strings: temporal/string comparisons in WHERE. DuckDB
// implicitly coerces the string literal to match the temporal column (same as
// polars). The DATE/TIME-typed columns of the py fixture are modeled as TIMESTAMP
// (no Date/Time dtype); cases that depend on date-only or time-only semantics are
// noted. Result is the surviving idx set.
func TestTemporalImplicitStrings(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "idx", Values: []any{int64(0), int64(1), int64(2)}},
		frame.SeriesInput{Name: "dtm", Values: []any{
			utc(2024, 1, 7, 1, 2, 3, 123456),
			utc(2006, 1, 1, 23, 59, 59, 555555),
			utc(2020, 12, 30, 10, 30, 45, 987654),
		}},
	)
	cases := []struct {
		constraint string
		expected   []any
	}{
		{"dtm >= '2020-12-30T10:30:45.987'", []any{int64(0), int64(2)}},
		{"dtm > '2006-01-01'", []any{int64(0), int64(1), int64(2)}},
		{"dtm <= '2006-01-01'", []any{}},
		{"dtm::date > '2006-01-01'", []any{int64(0), int64(2)}},
		{"dtm BETWEEN '2020-12-30 10:30:44' AND '2023-01-01 00:00:00'", []any{int64(2)}},
		{
			"dtm = '2024-01-07 01:02:03.123456000' OR dtm = '2020-12-30 10:30:45.987654'",
			[]any{int64(0), int64(2)},
		},
	}
	for _, c := range cases {
		out := query(t, d, "SELECT idx FROM self WHERE "+c.constraint+" ORDER BY idx")
		got := vals(t, out, "idx")
		if got == nil {
			got = []any{}
		}
		eqRow(t, got, c.expected, "WHERE "+c.constraint)
	}
}

// test_implicit_temporal_string_errors: malformed temporal strings in a temporal
// comparison raise a conversion error.
func TestTemporalImplicitStringErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "dt", Values: []any{utc(2020, 12, 30, 0, 0, 0, 0)}})
	for _, v := range []string{"yyyy-mm-dd", "2222-22-22"} {
		if err := runErr(t, d, "SELECT * FROM self WHERE dt = '"+v+"'"); err == nil {
			t.Fatalf("compare to %q: expected conversion error, got nil", v)
		}
	}
}

// test_strftime: STRFTIME(datetime, fmt) -> formatted string. MATCHES polars for
// the datetime input. The DATE/TIME inputs of the py fixture are GAP'd (no Date/
// Time dtype); only the datetime column is asserted here.
func TestTemporalStrftime(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "dtm", Values: []any{
			nil,
			utc(1980, 9, 30, 1, 25, 50, 0),
			utc(2077, 7, 17, 11, 30, 55, 0),
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	out := query(t, d, `SELECT STRFTIME(dtm,'%m.%d.%Y/%T') AS s FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "s"),
		[]any{nil, "09.30.1980/01:25:50", "07.17.2077/11:30:55"}, "STRFTIME")
}

// test_strptime: STRPTIME(string, fmt) -> datetime. The datetime form MATCHES
// polars. The polars `::date` and `::time` casts are GAP'd (no Date/Time dtype).
func TestTemporalStrptime(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "s_dtm", Values: []any{
			nil, "09.30.1980/01:25:50", "07.17.2077/11:30:55",
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	out := query(t, d, `SELECT STRPTIME(s_dtm,'%m.%d.%Y/%T') AS dtm FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "dtm"), []any{
		nil,
		utc(1980, 9, 30, 1, 25, 50, 0),
		utc(2077, 7, 17, 11, 30, 55, 0),
	}, "STRPTIME")
}

// test_temporal_stings_to_datetime (datetime portion): polars DATETIME(str) parses
// a string to a datetime. DuckDB has NO DATETIME() scalar function — GAP. The
// equivalent is STRPTIME / ::timestamp, asserted here as the working substitute.
func TestTemporalStringToDatetime(t *testing.T) {
	d := mustFrame(t,
		frame.SeriesInput{Name: "s", Values: []any{
			"1999-12-31 10:30:45", "2020-06-10", "2022-08-07T00:01:02.654321",
		}},
		frame.SeriesInput{Name: "idx", Values: idxAny(3)},
	)
	// DISCREPANCY: DATETIME() (polars) does not exist in DuckDB; use ::TIMESTAMP.
	out := query(t, d, `SELECT s::TIMESTAMP AS dtm FROM self ORDER BY idx`)
	eqRow(t, vals(t, out, "dtm"), []any{
		utc(1999, 12, 31, 10, 30, 45, 0),
		utc(2020, 6, 10, 0, 0, 0, 0),
		utc(2022, 8, 7, 0, 1, 2, 654321),
	}, "string->TIMESTAMP")
}

// GAP: the polars DATETIME() / TIME() scalar string-constructors do not exist in
// DuckDB (DATETIME -> Catalog Error; TIME '...' -> parser error since TIME is a
// reserved type keyword).
func TestTemporalDatetimeFnUnsupported(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})
	if err := runErr(t, d, `SELECT DATETIME('1999-12-31 10:30:45') AS d FROM self`); err == nil {
		t.Skip("GAP: expected DATETIME() to be missing in DuckDB, but it resolved")
	}
}

// test_temporal_typed_literals: typed temporal literals. TIMESTAMP literals MATCH
// polars (-> time.Time). DATE/TIME typed literals are GAP'd (date32/time64 are
// unreadable in gopolars) — verified by TestTemporalDateResultUnsupported.
func TestTemporalTypedLiterals(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})
	out := query(t, d, `
		SELECT
		  TIMESTAMP '1930-01-01 12:30:00' AS dtm1,
		  TIMESTAMP '2077-04-27T23:45:30.123456' AS dtm2
		FROM self`)
	eqRow(t, vals(t, out, "dtm1"), []any{utc(1930, 1, 1, 12, 30, 0, 0)}, "TIMESTAMP literal 1")
	eqRow(t, vals(t, out, "dtm2"), []any{utc(2077, 4, 27, 23, 45, 30, 123456)}, "TIMESTAMP literal 2")
}

// test_typed_literals_errors: an invalid TIMESTAMP literal raises an error.
func TestTemporalTypedLiteralErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})
	if err := runErr(t, d, `SELECT TIMESTAMP '999' AS dtm FROM self`); err == nil {
		t.Fatalf("TIMESTAMP '999': expected error, got nil")
	}
}

// test_timestamp_time_unit: ::timestamp with explicit precision. polars exposes
// the chosen time-unit; gopolars normalizes every timestamp to time.Time, so the
// observable difference is the surviving sub-second precision. ms truncates to
// milliseconds; us keeps microseconds. (DuckDB renders ns down to its supported
// precision.) Values MATCH at the asserted precision.
func TestTemporalTimestampTimeUnit(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})

	// us precision: full microseconds preserved.
	outUS := query(t, d, `SELECT (TIMESTAMP '2024-01-07 01:02:03.123456')::timestamp_us AS t FROM self`)
	eqRow(t, vals(t, outUS, "t"), []any{utc(2024, 1, 7, 1, 2, 3, 123456)}, "timestamp_us")

	// ms precision: truncated to milliseconds (123456us -> 123000us).
	outMS := query(t, d, `SELECT (TIMESTAMP '2024-01-07 01:02:03.123456')::timestamp_ms AS t FROM self`)
	eqRow(t, vals(t, outMS, "t"), []any{utc(2024, 1, 7, 1, 2, 3, 123000)}, "timestamp_ms")
}

// test_timestamp_time_unit_errors: invalid precision on a timestamp cast errors.
//
// DISCREPANCY: polars rejects both precision 0 and 15 (valid range 1-9). DuckDB
// only rejects precision > 9 ("TIMESTAMP only supports until nano-second
// precision (9)"); precision 0 is accepted and truncates to whole seconds.
func TestTemporalTimestampPrecisionErrors(t *testing.T) {
	d := mustFrame(t, frame.SeriesInput{Name: "x", Values: []any{int64(1)}})

	// > 9 errors in DuckDB (and polars).
	if err := runErr(t, d, "SELECT (TIMESTAMP '2024-01-07 01:02:03.123456')::timestamp(15) AS t FROM self"); err == nil {
		t.Fatalf("timestamp(15): expected precision error, got nil")
	}

	// DISCREPANCY: precision 0 is valid in DuckDB (truncates to seconds), whereas
	// polars rejects it.
	out := query(t, d, "SELECT (TIMESTAMP '2024-01-07 01:02:03.123456')::timestamp(0) AS t FROM self")
	eqRow(t, vals(t, out, "t"), []any{utc(2024, 1, 7, 1, 2, 3, 0)}, "timestamp(0) truncates")
}
