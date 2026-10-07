package frame

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// sortOracle returns the "seq" values of df in the order a stable sort with
// boxed value comparisons gives for in: nulls placed by NullsLast alone,
// Descending reversing only non-null values.
func sortOracle(t *testing.T, df DataFrame, in SortInput) []int64 {
	t.Helper()
	keys := make([]series.Series, len(in.By))
	for k, name := range in.By {
		keys[k], _ = df.Series(name)
	}
	rows := make([]int, df.Height())
	for i := range rows {
		rows[i] = i
	}
	slices.SortStableFunc(rows, func(a, b int) int {
		for k, s := range keys {
			na, nb := s.IsNull(a), s.IsNull(b)
			if na || nb {
				if c := compareNulls(na, nb, in.NullsLast); c != 0 {
					return c
				}
				continue
			}
			c := compareSortValues(s.Value(a), s.Value(b), in.NullsLast)
			if k < len(in.Descending) && in.Descending[k] {
				c = -c
			}
			if c != 0 {
				return c
			}
		}
		return 0
	})
	seq := columnInt64s(t, df, "seq")
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = seq[r]
	}
	return out
}

func checkSortMatchesOracle(t *testing.T, df DataFrame, in SortInput) {
	t.Helper()
	got, err := df.Sort(in)
	if err != nil {
		t.Fatalf("Sort(%+v): %v", in, err)
	}
	want := sortOracle(t, df, in)
	if seq := columnInt64s(t, got, "seq"); !slices.Equal(seq, want) {
		i := 0
		for i < len(seq) && seq[i] == want[i] {
			i++
		}
		t.Errorf("Sort(%+v) on %d rows: row order differs from the boxed stable sort at position %d (got row %d, want %d)", in, df.Height(), i, seq[i], want[i])
	}
}

// typedKeysFrame has n rows with a nullable Boolean "flag", Datetime "ts" and
// String "name" taking few values, and the row number "seq".
func typedKeysFrame(t *testing.T, n int) DataFrame {
	t.Helper()
	flag := make([]bool, n)
	ts := make([]time.Time, n)
	name := make([]string, n)
	flagNulls := make([]bool, n)
	tsNulls := make([]bool, n)
	nameNulls := make([]bool, n)
	seq := make([]int64, n)
	for i := range n {
		h := i * 2654435761
		flag[i] = h%2 == 0
		ts[i] = noonUTC.Add(time.Duration(h%5) * time.Hour)
		name[i] = fmt.Sprintf("n%d", h%4)
		flagNulls[i] = i%7 == 0
		tsNulls[i] = i%11 == 0
		nameNulls[i] = i%13 == 0
		seq[i] = int64(i)
	}
	df, err := New(NewInput{Series: []series.Series{
		series.FromColumn("flag", chunk.NewBool(flag, flagNulls)),
		series.FromColumn("ts", chunk.NewTime(ts, tsNulls)),
		series.FromColumn("name", chunk.NewString(name, nameNulls)),
		series.FromInt64("seq", seq, nil),
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func TestSortTypedKeysMatchBoxedOrder(t *testing.T) {
	t.Parallel()
	for _, n := range []int{200, 1000} {
		df := typedKeysFrame(t, n)
		for _, desc := range [][]bool{{false, true, false}, {true, false, true}} {
			for _, nullsLast := range []bool{false, true} {
				checkSortMatchesOracle(t, df, SortInput{By: []string{"flag", "ts", "name"}, Descending: desc, NullsLast: nullsLast})
				checkSortMatchesOracle(t, df, SortInput{By: []string{"name", "flag"}, Descending: desc, NullsLast: nullsLast})
			}
		}
	}
}

// numericLeadFrame has n rows whose "key" column of the given kind takes 7
// values, null on every 5th row when nullable, and the row number "seq".
func numericLeadFrame(t *testing.T, n int, kind string, nullable bool) DataFrame {
	t.Helper()
	nulls := make([]bool, n)
	seq := make([]int64, n)
	i64s := make([]int64, n)
	f64s := make([]float64, n)
	tims := make([]time.Time, n)
	for i := range n {
		v := (i * 2654435761) % 7
		i64s[i] = int64(v) - 3
		f64s[i] = float64(v) - 3.5
		tims[i] = noonUTC.Add(time.Duration(v-3) * time.Hour)
		nulls[i] = nullable && i%5 == 0
		seq[i] = int64(i)
	}
	var key *chunk.Column
	switch kind {
	case "int64":
		key = chunk.NewInt64(i64s, nulls)
	case "float64":
		key = chunk.NewFloat64(f64s, nulls)
	default:
		key = chunk.NewTime(tims, nulls)
	}
	df, err := New(NewInput{Series: []series.Series{series.FromColumn("key", key), series.FromInt64("seq", seq, nil)}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func TestSortNumericLeadingKeysMatchBoxedOrder(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"int64", "float64", "datetime"} {
		for _, nullable := range []bool{false, true} {
			df := numericLeadFrame(t, 1000, kind, nullable)
			if _, ok := df.radixArgsort(SortInput{By: []string{"key"}}); !ok {
				t.Errorf("radixArgsort(%s key, nullable=%v) declined, want the radix path", kind, nullable)
			}
			for _, desc := range []bool{false, true} {
				for _, nullsLast := range []bool{false, true} {
					checkSortMatchesOracle(t, df, SortInput{By: []string{"key"}, Descending: []bool{desc}, NullsLast: nullsLast})
				}
			}
		}
	}
}

func TestSortDatetimeOutsideNanosecondRange(t *testing.T) {
	t.Parallel()
	n := 1000
	ts := make([]time.Time, n)
	seq := make([]int64, n)
	for i := range n {
		ts[i] = noonUTC.Add(time.Duration(i%10) * time.Hour)
		seq[i] = int64(i)
	}
	ts[3] = farDate
	df, err := New(NewInput{Series: []series.Series{series.FromColumn("ts", chunk.NewTime(ts, nil)), series.FromInt64("seq", seq, nil)}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := df.radixArgsort(SortInput{By: []string{"ts"}}); ok {
		t.Error("radixArgsort(ts with 9999-12-31) took the radix path, want the comparison path")
	}
	got, err := df.Sort(SortInput{By: []string{"ts"}})
	if err != nil {
		t.Fatalf("Sort(ts): %v", err)
	}
	if last := columnTimes(t, got, "ts")[n-1]; !last.Equal(farDate) {
		t.Errorf("Sort(ts) last value = %v, want %v", last, farDate)
	}
	checkSortMatchesOracle(t, df, SortInput{By: []string{"ts"}, Descending: []bool{true}})
}

func TestSortStringKeyAllocations(t *testing.T) {
	const n = 20_000
	df := lowCardinalityFrame(t, n)
	allocs := testing.AllocsPerRun(3, func() {
		if _, err := df.Sort(SortInput{By: []string{"s300"}}); err != nil {
			t.Fatal(err)
		}
	})
	if allocs >= n {
		t.Errorf("Sort(String key) on %d rows made %.0f allocations, want fewer than one per row", n, allocs)
	}
}

func TestSortNaNKeyStillUsesComparisonOrder(t *testing.T) {
	t.Parallel()
	df := numericLeadFrame(t, 1000, "float64", true)
	s, _ := df.Series("key")
	f64s, _ := s.Column().Float64s()
	f64s[7] = math.NaN() // NaN keys keep the comparison path: NaN is the largest value
	for _, desc := range []bool{false, true} {
		checkSortMatchesOracle(t, df, SortInput{By: []string{"key"}, Descending: []bool{desc}, NullsLast: true})
	}
}
