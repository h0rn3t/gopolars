package parquet

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	pqgo "github.com/parquet-go/parquet-go"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

func mixedFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	ts := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), nil}},
			{Name: "score", Values: []any{float64(1.5), math.NaN(), nil}},
			{Name: "active", Values: []any{true, nil, false}},
			{Name: "label", Values: []any{"alpha", "", "gamma"}},
			{Name: "ts", Values: []any{ts, ts.Add(time.Hour), nil}},
		},
	})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	return df
}

// TestWriteParquetIsColumnar asserts the writer produces a real columnar
// parquet file (one leaf per DataFrame column), not the legacy single "payload"
// envelope.
func TestWriteParquetIsColumnar(t *testing.T) {
	df := mixedFrame(t)
	path := filepath.Join(t.TempDir(), "columnar.parquet")
	if err := Write(df, WriteInput{Path: path}); err != nil {
		t.Fatalf("write: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	info, _ := f.Stat()
	pf, err := pqgo.OpenFile(f, info.Size())
	if err != nil {
		t.Fatalf("open parquet: %v", err)
	}

	got := make([]string, 0)
	for _, fld := range pf.Schema().Fields() {
		got = append(got, fld.Name())
	}
	want := df.Columns()
	if len(got) != len(want) {
		t.Fatalf("schema has %d leaf columns %v, want %d %v", len(got), got, len(want), want)
	}
	for i, name := range want {
		if got[i] != name {
			t.Fatalf("column %d on disk = %q, want %q (got=%v)", i, got[i], name, got)
		}
	}
	if len(got) == 1 && got[0] == "payload" {
		t.Fatal("writer produced a legacy single-payload envelope, not columnar parquet")
	}
}

// TestWriteParquetRoundTripNullsNaN pins value/type/null/NaN fidelity across the
// five core dtypes.
func TestWriteParquetRoundTripNullsNaN(t *testing.T) {
	df := mixedFrame(t)
	path := filepath.Join(t.TempDir(), "rt.parquet")
	if err := Write(df, WriteInput{Path: path, Compression: "zstd", RowGroupSize: 2}); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Height() != df.Height() || got.Width() != df.Width() {
		t.Fatalf("shape changed: %dx%d want %dx%d", got.Height(), got.Width(), df.Height(), df.Width())
	}
	for _, name := range df.Columns() {
		orig, _ := df.Series(name)
		back, ok := got.Series(name)
		if !ok {
			t.Fatalf("column %q missing after round-trip", name)
		}
		for i := 0; i < df.Height(); i++ {
			ov, bv := orig.Value(i), back.Value(i)
			if ov == nil || bv == nil {
				if (ov == nil) != (bv == nil) {
					t.Fatalf("col %q row %d null mismatch: orig=%v back=%v", name, i, ov, bv)
				}
				continue
			}
			if of, isF := ov.(float64); isF {
				bf := bv.(float64)
				if math.IsNaN(of) != math.IsNaN(bf) || (!math.IsNaN(of) && of != bf) {
					t.Fatalf("col %q row %d float mismatch: orig=%v back=%v", name, i, of, bf)
				}
				continue
			}
			if ot, isT := ov.(time.Time); isT {
				if !ot.Equal(bv.(time.Time)) {
					t.Fatalf("col %q row %d time mismatch: orig=%v back=%v", name, i, ot, bv)
				}
				continue
			}
			if ov != bv {
				t.Fatalf("col %q row %d mismatch: orig=%v back=%v", name, i, ov, bv)
			}
		}
	}
}

// TestReadLegacyEnvelopeFixture builds a file in the previous single-column
// JSON-envelope parquet format and asserts the reader still decodes it.
func TestReadLegacyEnvelopeFixture(t *testing.T) {
	payload := parquetPayload{Columns: []parquetColumn{
		{Name: "id", Type: "int64", Values: []any{float64(1), float64(2)}},
		{Name: "label", Type: "string", Values: []any{"a", "b"}},
	}}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy.parquet")
	if err := pqgo.WriteFile(path, []parquetEnvelope{{Payload: string(raw)}}); err != nil {
		t.Fatalf("write legacy envelope: %v", err)
	}

	got, err := Read(ReadInput{Path: path})
	if err != nil {
		t.Fatalf("read legacy: %v", err)
	}
	if got.Height() != 2 || got.Width() != 2 {
		t.Fatalf("legacy decode shape = %dx%d, want 2x2", got.Height(), got.Width())
	}
	if s, ok := got.Series("label"); !ok || s.Value(1) != "b" {
		t.Fatalf("legacy label column wrong: ok=%v v=%v", ok, func() any { s, _ := got.Series("label"); return s.Value(1) }())
	}
}
