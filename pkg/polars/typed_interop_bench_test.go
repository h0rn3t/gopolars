package polars

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

const interopRows = 200_000

// interopRow is the shape a consumer maps parquet rows into.
type interopRow struct {
	ID int64
	V  float64
	S  string
	TS time.Time
}

func interopRows200k() []interopRow {
	labels := []string{"alpha", "beta", "gamma"}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]interopRow, interopRows)
	for i := range rows {
		rows[i] = interopRow{
			ID: int64(i),
			V:  float64(i) * 1.5,
			S:  labels[i%len(labels)],
			TS: base.Add(time.Duration(i) * time.Minute),
		}
	}
	return rows
}

// frameFromRowsAny builds a frame the way the boxed NewDataFrame API requires.
func frameFromRowsAny(rows []interopRow) (DataFrame, error) {
	id := make([]any, len(rows))
	v := make([]any, len(rows))
	s := make([]any, len(rows))
	ts := make([]any, len(rows))
	for i, r := range rows {
		id[i], v[i], s[i], ts[i] = r.ID, r.V, r.S, r.TS
	}
	return NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: id, DType: dtypes.Int64},
		{Name: "v", Values: v, DType: dtypes.Float64},
		{Name: "s", Values: s, DType: dtypes.String},
		{Name: "ts", Values: ts, DType: dtypes.Datetime},
	}})
}

func BenchmarkRowsToFrameAny(b *testing.B) {
	rows := interopRows200k()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := frameFromRowsAny(rows); err != nil {
			b.Fatalf("frameFromRowsAny: %v", err)
		}
	}
}

// frameFromRowsTyped builds the same frame through the typed constructors.
func frameFromRowsTyped(rows []interopRow) (DataFrame, error) {
	id := make([]int64, len(rows))
	v := make([]float64, len(rows))
	s := make([]string, len(rows))
	ts := make([]time.Time, len(rows))
	for i, r := range rows {
		id[i], v[i], s[i], ts[i] = r.ID, r.V, r.S, r.TS
	}
	idS, err := NewInt64Series("id", id, nil)
	if err != nil {
		return nil, err
	}
	vS, err := NewFloat64Series("v", v, nil)
	if err != nil {
		return nil, err
	}
	sS, err := NewStringSeries("s", s, nil)
	if err != nil {
		return nil, err
	}
	tsS, err := NewDatetimeSeries("ts", ts, nil)
	if err != nil {
		return nil, err
	}
	return NewDataFrameFromSeries(idS, vS, sS, tsS)
}

func BenchmarkRowsToFrameTyped(b *testing.B) {
	rows := interopRows200k()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := frameFromRowsTyped(rows); err != nil {
			b.Fatalf("frameFromRowsTyped: %v", err)
		}
	}
}

func BenchmarkFrameToRowsTyped(b *testing.B) {
	df, err := frameFromRowsTyped(interopRows200k())
	if err != nil {
		b.Fatalf("frameFromRowsTyped: %v", err)
	}
	column := func(name string) Series {
		s, err := df.GetColumn(name)
		if err != nil {
			b.Fatalf("GetColumn(%q): %v", name, err)
		}
		return s
	}
	b.ReportAllocs()
	for b.Loop() {
		ids, _, err := column("id").Int64Values()
		if err != nil {
			b.Fatalf("Int64Values: %v", err)
		}
		vs, _, err := column("v").Float64Values()
		if err != nil {
			b.Fatalf("Float64Values: %v", err)
		}
		ss, _, err := column("s").StringValues()
		if err != nil {
			b.Fatalf("StringValues: %v", err)
		}
		tss, _, err := column("ts").DatetimeValues()
		if err != nil {
			b.Fatalf("DatetimeValues: %v", err)
		}
		out := make([]interopRow, len(ids))
		for i := range out {
			out[i] = interopRow{ID: ids[i], V: vs[i], S: ss[i], TS: tss[i]}
		}
		if len(out) != interopRows {
			b.Fatalf("mapped %d rows, want %d", len(out), interopRows)
		}
	}
}

func BenchmarkFrameToRowsIterRows(b *testing.B) {
	df, err := frameFromRowsAny(interopRows200k())
	if err != nil {
		b.Fatalf("frameFromRowsAny: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		dicts := df.IterRows()
		out := make([]interopRow, 0, len(dicts))
		for _, d := range dicts {
			id, _ := d["id"].(int64)
			v, _ := d["v"].(float64)
			s, _ := d["s"].(string)
			ts, _ := d["ts"].(time.Time)
			out = append(out, interopRow{ID: id, V: v, S: s, TS: ts})
		}
		if len(out) != interopRows {
			b.Fatalf("mapped %d rows, want %d", len(out), interopRows)
		}
	}
}
