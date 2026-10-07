package polars

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

func TestLazyFillNaNKeepsExactValue(t *testing.T) {
	t.Parallel()

	for _, value := range []float64{1e-7, 0.1234567891, 1e300, 5e-324, math.Inf(1), math.Inf(-1)} {
		t.Run(strconv.FormatFloat(value, 'g', -1, 64), func(t *testing.T) {
			t.Parallel()

			d := mscFrame(t, mscCol("x", 1.5, math.NaN()))
			eager, err := d.FillNaN(value)
			if err != nil {
				t.Fatalf("FillNaN(%v) error = %v", value, err)
			}
			lazy, err := d.Lazy().FillNaN(value).Collect(t.Context())
			if err != nil {
				t.Fatalf("Lazy().FillNaN(%v).Collect() error = %v", value, err)
			}
			want := []any{1.5, value}
			if got := frameColumnValues(t, eager, "x"); !slices.Equal(got, want) {
				t.Errorf("FillNaN(%v) x = %v, want %v", value, got, want)
			}
			if got := frameColumnValues(t, lazy, "x"); !slices.Equal(got, want) {
				t.Errorf("Lazy().FillNaN(%v) x = %v, want %v", value, got, want)
			}
		})
	}
}

func TestLazyQuantileKeepsExactProbability(t *testing.T) {
	t.Parallel()

	for _, q := range []float64{0.1234567891, 1e-7, 0, 1} {
		t.Run(strconv.FormatFloat(q, 'g', -1, 64), func(t *testing.T) {
			t.Parallel()

			d := mscFrame(t, mscCol("v", 0.0, 1e6))
			out, err := d.Lazy().Quantile(q).Collect(t.Context())
			if err != nil {
				t.Fatalf("Lazy().Quantile(%v).Collect() error = %v", q, err)
			}
			s, ok := d.Series("v")
			if !ok {
				t.Fatal("Series(v) not found")
			}
			want := []any{s.Quantile(q)}
			if got := frameColumnValues(t, out, "v"); !slices.Equal(got, want) {
				t.Errorf("Lazy().Quantile(%v) v = %v, want %v", q, got, want)
			}
		})
	}
}

func TestUpdateOverwritesByRowPosition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		left  []frame.SeriesInput
		other []frame.SeriesInput
		want  []frame.SeriesInput
	}{
		{
			name: "non-null values replace",
			left: []frame.SeriesInput{
				mscCol("id", int64(1), int64(2), int64(3)),
				mscCol("v", int64(10), int64(20), int64(30)),
			},
			other: []frame.SeriesInput{
				mscCol("v", int64(100), nil, int64(300)),
				mscCol("w", int64(7), int64(8), int64(9)),
			},
			want: []frame.SeriesInput{
				{Name: "id", DType: dtypes.Int64, Values: []any{int64(1), int64(2), int64(3)}},
				{Name: "v", DType: dtypes.Int64, Values: []any{int64(100), int64(20), int64(300)}},
			},
		},
		{
			name:  "shorter other",
			left:  []frame.SeriesInput{mscCol("v", int64(10), int64(20), int64(30))},
			other: []frame.SeriesInput{mscCol("v", int64(100))},
			want:  []frame.SeriesInput{{Name: "v", DType: dtypes.Int64, Values: []any{int64(100), int64(20), int64(30)}}},
		},
		{
			name:  "longer other",
			left:  []frame.SeriesInput{mscCol("v", int64(10), int64(20))},
			other: []frame.SeriesInput{mscCol("v", int64(1), int64(2), int64(3))},
			want:  []frame.SeriesInput{{Name: "v", DType: dtypes.Int64, Values: []any{int64(1), int64(2)}}},
		},
		{
			name: "same height appends no rows",
			left: []frame.SeriesInput{
				mscCol("id", int64(1), int64(2), int64(3)),
				mscCol("v", int64(10), int64(20), int64(30)),
			},
			other: []frame.SeriesInput{
				mscCol("id", int64(1), int64(2), int64(3)),
				mscCol("v", int64(100), int64(200), int64(300)),
			},
			want: []frame.SeriesInput{
				{Name: "id", DType: dtypes.Int64, Values: []any{int64(1), int64(2), int64(3)}},
				{Name: "v", DType: dtypes.Int64, Values: []any{int64(100), int64(200), int64(300)}},
			},
		},
		{
			name:  "int64 from float64",
			left:  []frame.SeriesInput{mscCol("v", int64(1), int64(2), int64(3))},
			other: []frame.SeriesInput{mscCol("v", 1.5, nil, 3.5)},
			want:  []frame.SeriesInput{{Name: "v", DType: dtypes.Float64, Values: []any{1.5, 2.0, 3.5}}},
		},
		{
			name:  "float64 from int64",
			left:  []frame.SeriesInput{mscCol("v", 1.5, 2.5)},
			other: []frame.SeriesInput{mscCol("v", int64(7), nil)},
			want:  []frame.SeriesInput{{Name: "v", DType: dtypes.Float64, Values: []any{7.0, 2.5}}},
		},
		{
			name:  "null left value",
			left:  []frame.SeriesInput{mscCol("s", "a", nil, "c")},
			other: []frame.SeriesInput{mscCol("s", nil, "b", nil)},
			want:  []frame.SeriesInput{{Name: "s", DType: dtypes.String, Values: []any{"a", "b", "c"}}},
		},
		{
			name:  "no shared columns",
			left:  []frame.SeriesInput{mscCol("v", int64(1), int64(2))},
			other: []frame.SeriesInput{mscCol("w", "x", "y")},
			want:  []frame.SeriesInput{{Name: "v", DType: dtypes.Int64, Values: []any{int64(1), int64(2)}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			left := mscFrame(t, tc.left...)
			other := mscFrame(t, tc.other...)

			eager, err := left.Update(other)
			checkUpdated(t, "Update", eager, err, tc.want)
			lazy, err := left.Lazy().Update(other.Lazy()).Collect(t.Context())
			checkUpdated(t, "Lazy().Update().Collect", lazy, err, tc.want)
			streamed, err := left.Lazy().Update(other.Lazy()).CollectStreaming(t.Context(), 1)
			checkUpdated(t, "Lazy().Update().CollectStreaming(1)", streamed, err, tc.want)
		})
	}
}

func TestUpdateRejectsIncompatibleDtypes(t *testing.T) {
	t.Parallel()

	left := mscFrame(t, mscCol("v", int64(1), int64(2)))
	other := mscFrame(t, mscCol("v", "a", "b"))

	if _, err := left.Update(other); err == nil {
		t.Error("Update(Int64 v, String v) error = nil, want an error")
	}
	if _, err := left.Lazy().Update(other.Lazy()).Collect(t.Context()); err == nil {
		t.Error("Lazy().Update(Int64 v, String v).Collect() error = nil, want an error")
	}
}

func TestLazyUpdateUsesOtherFramesData(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "other.csv")
	if err := os.WriteFile(path, []byte("v\n100\n200\n300\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	scan, err := NewIO().ScanCSV(ScanCSVInput{Path: path, HasHeader: true})
	if err != nil {
		t.Fatalf("ScanCSV(%s) error = %v", path, err)
	}
	inMemory := mscFrame(t,
		mscCol("id", int64(1), int64(2), int64(3)),
		mscCol("v", int64(100), int64(200), int64(300)),
	).Lazy()
	renamed := mscFrame(t, mscCol("w", int64(7), int64(8), int64(9))).Lazy().Select(Col("w").Alias("v"))

	cases := []struct {
		name  string
		other LazyFrame
		want  []any
	}{
		{"in-memory frame", inMemory, []any{int64(100), int64(200), int64(300)}},
		{"other plan applied", renamed, []any{int64(7), int64(8), int64(9)}},
		{"csv scan", scan, []any{int64(100), int64(200), int64(300)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			left := mscFrame(t,
				mscCol("id", int64(1), int64(2), int64(3)),
				mscCol("v", int64(10), int64(20), int64(30)),
			).Lazy().Update(tc.other)

			out, err := left.Collect(t.Context())
			if err != nil {
				t.Fatalf("Update(%s).Collect() error = %v", tc.name, err)
			}
			if got := frameColumnValues(t, out, "v"); !slices.Equal(got, tc.want) {
				t.Errorf("Update(%s).Collect() v = %v, want %v", tc.name, got, tc.want)
			}
			streamed, err := left.CollectStreaming(t.Context(), 1)
			if err != nil {
				t.Fatalf("Update(%s).CollectStreaming(1) error = %v", tc.name, err)
			}
			if got := frameColumnValues(t, streamed, "v"); !slices.Equal(got, tc.want) {
				t.Errorf("Update(%s).CollectStreaming(1) v = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestLazyUpdateReportsOtherCollectError(t *testing.T) {
	t.Parallel()

	missing, err := NewIO().ScanCSV(ScanCSVInput{Path: filepath.Join(t.TempDir(), "missing.csv"), HasHeader: true})
	if err != nil {
		t.Fatalf("ScanCSV(missing.csv) error = %v", err)
	}
	_, otherErr := missing.Collect(t.Context())
	if otherErr == nil {
		t.Fatal("Collect(missing.csv) error = nil, want an error")
	}

	left := mscFrame(t, mscCol("v", int64(1), int64(2))).Lazy().Update(missing)
	out, err := left.Collect(t.Context())
	if err == nil || !strings.Contains(err.Error(), otherErr.Error()) {
		t.Errorf("Update(missing.csv).Collect() = %v, %v; want error containing %q", out, err, otherErr)
	}
	if _, err := left.CollectStreaming(t.Context(), 1); err == nil {
		t.Error("Update(missing.csv).CollectStreaming(1) error = nil, want an error")
	}
}

// checkUpdated compares an updated frame column by column — names, order,
// dtypes and values — with want.
func checkUpdated(t *testing.T, call string, got DataFrame, err error, want []frame.SeriesInput) {
	t.Helper()
	if err != nil {
		t.Errorf("%s error = %v", call, err)
		return
	}
	names := make([]string, len(want))
	for i, c := range want {
		names[i] = c.Name
	}
	if cols := got.Columns(); !slices.Equal(cols, names) {
		t.Errorf("%s columns = %v, want %v", call, cols, names)
		return
	}
	for _, c := range want {
		s, _ := got.Series(c.Name)
		if s.DataType() != c.DType || !slices.Equal(s.ToList(), c.Values) {
			t.Errorf("%s %s = %s %v, want %s %v", call, c.Name, s.DataType(), s.ToList(), c.DType, c.Values)
		}
	}
}

// frameColumnValues returns the values of column name of d.
func frameColumnValues(t *testing.T, d DataFrame, name string) []any {
	t.Helper()
	s, ok := d.Series(name)
	if !ok {
		t.Fatalf("Series(%q) not found in %v", name, d.Columns())
	}
	return s.ToList()
}
