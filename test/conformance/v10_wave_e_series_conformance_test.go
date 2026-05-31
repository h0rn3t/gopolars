package conformance

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestV10WaveESeries(t *testing.T) {
	trig, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "angles",
		DType:  polars.Float64,
		Values: []any{float64(0), math.Pi / 2},
	})
	if err != nil {
		t.Fatalf("new trig series failed: %v", err)
	}
	sin := trig.Sin()
	if sin.Len() != 2 {
		t.Fatalf("unexpected sin len: got %d want 2", sin.Len())
	}
	if got := sin.Value(1).(float64); math.Abs(got-1) > 1e-9 {
		t.Fatalf("unexpected sin value: got %.12f want 1.0", got)
	}

	stats, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "stats",
		DType:  polars.Float64,
		Values: []any{float64(1), float64(2), float64(2), float64(3)},
	})
	if err != nil {
		t.Fatalf("new stats series failed: %v", err)
	}
	if got := stats.Max(); got != 3 {
		t.Fatalf("unexpected max: got %.12f want 3", got)
	}
	if got := stats.Min(); got != 1 {
		t.Fatalf("unexpected min: got %.12f want 1", got)
	}
	if got := stats.NUnique(); got != 3 {
		t.Fatalf("unexpected n_unique: got %d want 3", got)
	}
	if got := stats.Sort(false); got.Value(0).(float64) != 1 {
		t.Fatalf("unexpected sort first value: got %v want 1", got.Value(0))
	}
	if got := stats.Head(2); got.Len() != 2 {
		t.Fatalf("unexpected head len: got %d want 2", got.Len())
	}

	mask, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "mask",
		DType:  polars.Boolean,
		Values: []any{true, false, true, false},
	})
	if err != nil {
		t.Fatalf("new mask series failed: %v", err)
	}
	if !mask.Any() || mask.All() {
		t.Fatalf("unexpected boolean reductions: any=%v all=%v", mask.Any(), mask.All())
	}
	if got := stats.IsBetween(1, 2); got.Len() != stats.Len() {
		t.Fatalf("unexpected is_between len: got %d want %d", got.Len(), stats.Len())
	}
	if got := mask.Not_(); got.Value(1) != true {
		t.Fatalf("unexpected not value: got %v want true", got.Value(1))
	}

	if got := stats.CumSum(); got.Value(3).(float64) != 8 {
		t.Fatalf("unexpected cumsum tail: got %v want 8", got.Value(3))
	}
	if got := stats.RollingStd(2); got.Len() != stats.Len() {
		t.Fatalf("unexpected rolling_std len: got %d want %d", got.Len(), stats.Len())
	}
	if got := stats.EwmMean(0.5); got.Len() != stats.Len() {
		t.Fatalf("unexpected ewm_mean len: got %d want %d", got.Len(), stats.Len())
	}

	other, err := polars.NewSeries(polars.NewSeriesInput{
		Name:   "other",
		DType:  polars.Float64,
		Values: []any{float64(2), float64(3), float64(4), float64(5)},
	})
	if err != nil {
		t.Fatalf("new other series failed: %v", err)
	}
	dot, err := stats.Dot(other)
	if err != nil {
		t.Fatalf("dot failed: %v", err)
	}
	if dot != 31 {
		t.Fatalf("unexpected dot: got %.12f want 31", dot)
	}
	counts, err := stats.ValueCounts()
	if err != nil {
		t.Fatalf("value_counts failed: %v", err)
	}
	if counts.Width() != 2 || counts.Height() == 0 {
		t.Fatalf("unexpected value_counts shape: %v", counts.Shape())
	}
	rle, err := stats.Rle()
	if err != nil {
		t.Fatalf("rle failed: %v", err)
	}
	if rle.Width() != 2 {
		t.Fatalf("unexpected rle width: got %d want 2", rle.Width())
	}
	mapped, err := stats.MapElements(func(v any) any {
		return v.(float64) * 10
	})
	if err != nil {
		t.Fatalf("map_elements failed: %v", err)
	}
	if mapped.Value(0).(float64) != 10 {
		t.Fatalf("unexpected mapped value: got %v want 10", mapped.Value(0))
	}
	replaced, err := stats.ReplaceStrict(float64(2), float64(20))
	if err != nil {
		t.Fatalf("replace_strict failed: %v", err)
	}
	if replaced.Value(1).(float64) != 20 {
		t.Fatalf("unexpected replaced value: got %v want 20", replaced.Value(1))
	}

	frame, err := stats.Rename("renamed").ToFrame()
	if err != nil {
		t.Fatalf("to_frame failed: %v", err)
	}
	if _, err := frame.GetColumn("renamed"); err != nil {
		t.Fatalf("expected renamed column: %v", err)
	}
	if _, err := stats.ToArrow(); err != nil {
		t.Fatalf("to_arrow failed: %v", err)
	}
	payload, err := stats.Serialize()
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	roundtrip, err := stats.Deserialize(payload)
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if roundtrip.Len() != stats.Len() {
		t.Fatalf("unexpected deserialize len: got %d want %d", roundtrip.Len(), stats.Len())
	}
	if got := stats.Hash(7); got.Len() != stats.Len() {
		t.Fatalf("unexpected hash len: got %d want %d", got.Len(), stats.Len())
	}
	zip, err := stats.ZipWith(mask, other)
	if err != nil {
		t.Fatalf("zip_with failed: %v", err)
	}
	if zip.Value(0).(float64) != 1 || zip.Value(1).(float64) != 3 {
		t.Fatalf("unexpected zip_with values: got [%v %v]", zip.Value(0), zip.Value(1))
	}
}
