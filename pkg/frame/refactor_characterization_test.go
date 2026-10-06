package frame

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/expr/evalbatch"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// columnValues returns the boxed values of column name in d.
func columnValues(t *testing.T, d DataFrame, name string) []any {
	t.Helper()
	s, ok := d.Series(name)
	if !ok {
		t.Fatalf("column %s missing", name)
	}
	out := make([]any, s.Len())
	for i := range out {
		out[i] = s.Value(i)
	}
	return out
}

func seriesValues(s series.Series) []any {
	out := make([]any, s.Len())
	for i := range out {
		out[i] = s.Value(i)
	}
	return out
}

func TestCompareSortValuesCharacterization(t *testing.T) {
	nan := math.NaN()
	negZero := math.Copysign(0, -1)
	cases := []struct {
		l, r      any
		nullsLast bool
		want      int
	}{
		{nil, nil, false, 0},
		{nil, nil, true, 0},
		{nil, int64(1), false, -1},
		{nil, int64(1), true, 1},
		{int64(1), nil, false, 1},
		{int64(1), nil, true, -1},
		{nan, nan, false, 0},
		{nan, 1.0, false, 1},
		{1.0, nan, false, -1},
		{1.0, 2.0, false, -1},
		{2.0, 1.0, false, 1},
		{negZero, 0.0, false, 0},
		{math.Inf(-1), math.Inf(1), false, -1},
		{int64(1), int64(2), false, -1},
		{int64(2), int64(1), false, 1},
		{int64(3), int64(3), false, 0},
		{"a", "b", false, -1},
		{"b", "a", false, 1},
		{"a", "a", false, 0},
		{false, true, false, -1},
		{true, false, false, 1},
		{true, true, false, 0},
		{int64(1), 1.0, false, 0},
		{nan, int64(1), false, 0},
		{int64(1), nan, false, 0},
		{"a", int64(1), false, 0},
		{time.Unix(1, 0), time.Unix(2, 0), false, 0},
	}
	for _, tc := range cases {
		if got := compareSortValues(tc.l, tc.r, tc.nullsLast); got != tc.want {
			t.Errorf("compareSortValues(%v, %v, %v) = %d, want %d", tc.l, tc.r, tc.nullsLast, got, tc.want)
		}
	}
}

func TestLessAnyCharacterization(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		l, r any
		want bool
	}{
		{false, true, true},
		{true, false, false},
		{false, false, false},
		{true, true, false},
		{false, int64(1), false},
		{int64(1), int64(2), true},
		{int64(2), int64(1), false},
		{int64(1), 1.0, false},
		{1.0, 2.0, true},
		{2.0, 1.0, false},
		{nan, 1.0, false},
		{1.0, nan, false},
		{1.0, int64(2), false},
		{"a", "b", true},
		{"b", "a", false},
		{"a", false, false},
		{nil, int64(1), false},
		{int64(1), nil, false},
		{time.Unix(1, 0), time.Unix(2, 0), false},
	}
	for _, tc := range cases {
		if got := lessAny(tc.l, tc.r); got != tc.want {
			t.Errorf("lessAny(%v, %v) = %v, want %v", tc.l, tc.r, got, tc.want)
		}
	}
}

// TestColumnComparatorsCharacterization checks the typed float/int comparators
// against an explicit ordering: non-null values ascending, NaN after every
// value, nulls first or last per nullsLast.
func TestColumnComparatorsCharacterization(t *testing.T) {
	nan := math.NaN()
	f := series.FromFloat64("f", []float64{1, nan, 0, -1, nan, math.Copysign(0, -1), 2}, []bool{false, false, true, false, false, false, false})
	i := series.FromInt64("i", []int64{3, 1, 0, 2, 1}, []bool{false, false, true, false, false})
	s, err := series.New("s", dtypes.String, []any{"b", nil, "a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	rank := func(s series.Series, row int, nullsLast bool) (int, float64) {
		if s.IsNull(row) {
			if nullsLast {
				return 3, 0
			}
			return 0, 0
		}
		switch v := s.Value(row).(type) {
		case float64:
			if math.IsNaN(v) {
				return 2, 0
			}
			return 1, v
		case int64:
			return 1, float64(v)
		}
		return 1, 0
	}
	for _, col := range []series.Series{f, i} {
		cmps := buildColumnComparators([]series.Series{col})
		for _, nullsLast := range []bool{false, true} {
			for a := range col.Len() {
				for b := range col.Len() {
					ra, va := rank(col, a, nullsLast)
					rb, vb := rank(col, b, nullsLast)
					want := 0
					switch {
					case ra < rb:
						want = -1
					case ra > rb:
						want = 1
					case va < vb:
						want = -1
					case va > vb:
						want = 1
					}
					if got := cmps[0](a, b, nullsLast); got != want {
						t.Errorf("%s cmp(%d,%d,%v) = %d, want %d", col.Name(), a, b, nullsLast, got, want)
					}
				}
			}
		}
	}
	cmps := buildColumnComparators([]series.Series{s})
	if got := cmps[0](0, 2, false); got != 1 {
		t.Errorf("string cmp b>a = %d", got)
	}
	if got := cmps[0](1, 0, true); got != 1 {
		t.Errorf("string cmp null last = %d", got)
	}
	if got := cmps[0](0, 3, false); got != 0 {
		t.Errorf("string cmp equal = %d", got)
	}
}

func TestSortMultiKeyDescendingCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{"x", "y", "x", nil, "y", "x"}},
		SeriesInput{Name: "b", Values: []any{int64(1), int64(2), int64(3), int64(4), nil, int64(3)}},
		SeriesInput{Name: "id", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4), int64(5)}},
	)
	cases := []struct {
		in   SortInput
		want []any
	}{
		{SortInput{By: []string{"a", "b"}}, []any{int64(3), int64(0), int64(2), int64(5), int64(4), int64(1)}},
		{SortInput{By: []string{"a", "b"}, Descending: []bool{true, false}}, []any{int64(4), int64(1), int64(0), int64(2), int64(5), int64(3)}},
		{SortInput{By: []string{"a", "b"}, Descending: []bool{false, true}, NullsLast: true}, []any{int64(2), int64(5), int64(0), int64(4), int64(1), int64(3)}},
		{SortInput{By: []string{"a", "b"}, Descending: []bool{true}}, []any{int64(4), int64(1), int64(0), int64(2), int64(5), int64(3)}},
	}
	for _, tc := range cases {
		out, err := df.Sort(tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := columnValues(t, out, "id"); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Sort(%+v) ids = %v, want %v", tc.in, got, tc.want)
		}
	}
	if _, err := df.Sort(SortInput{By: []string{"nope"}}); err == nil || err.Error() != "column nope not found" {
		t.Errorf("sort missing column err = %v", err)
	}
}

// TestRadixSecondaryTiesCharacterization drives the radix leading-key path with
// descending secondary keys over a frame above the radix threshold.
func TestRadixSecondaryTiesCharacterization(t *testing.T) {
	n := radixSortThreshold * 2
	lead := make([]any, n)
	sec := make([]any, n)
	id := make([]any, n)
	for r := range n {
		lead[r] = int64(r % 3)
		if r%5 == 0 {
			sec[r] = nil
		} else {
			sec[r] = int64(r % 7)
		}
		id[r] = int64(r)
	}
	df := cov80Frame(t,
		SeriesInput{Name: "lead", Values: lead},
		SeriesInput{Name: "sec", Values: sec},
		SeriesInput{Name: "id", Values: id},
	)
	for _, in := range []SortInput{
		{By: []string{"lead", "sec"}, Descending: []bool{false, true}},
		{By: []string{"lead", "sec"}, Descending: []bool{true, false}, NullsLast: true},
		{By: []string{"lead", "sec"}},
	} {
		out, err := df.Sort(in)
		if err != nil {
			t.Fatal(err)
		}
		ids := columnValues(t, out, "id")
		got := make([]int, n)
		for k, v := range ids {
			got[k] = int(v.(int64))
		}
		// The radix path reverses a stable ascending argsort for a descending
		// leading key, so its ties start in descending row order.
		want := make([]int, n)
		for k := range want {
			want[k] = k
		}
		if len(in.Descending) > 0 && in.Descending[0] {
			slices.Reverse(want)
		}
		cmpKey := func(a, b any, desc bool) int {
			c := compareSortValues(a, b, in.NullsLast)
			if desc {
				return -c
			}
			return c
		}
		slices.SortStableFunc(want, func(p, q int) int {
			if c := cmpKey(lead[p], lead[q], len(in.Descending) > 0 && in.Descending[0]); c != 0 {
				return c
			}
			return cmpKey(sec[p], sec[q], len(in.Descending) > 1 && in.Descending[1])
		})
		if !slices.Equal(got, want) {
			t.Errorf("Sort(%+v) order mismatch", in)
		}
	}
}

func TestCrossJoinCharacterization(t *testing.T) {
	left := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "k", Values: []any{"x", nil}},
	)
	right := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{10.0, 20.0, 30.0}},
		SeriesInput{Name: "r", Values: []any{true, false, nil}},
	)
	out, err := left.Join(JoinInput{Other: right, How: JoinTypeCross})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.Columns(), []string{"a", "k", "a_right", "r"}; !slices.Equal(got, want) {
		t.Fatalf("cross columns = %v, want %v", got, want)
	}
	if got, want := out.Dtypes(), []dtypes.DataType{dtypes.Int64, dtypes.String, dtypes.Float64, dtypes.Boolean}; !slices.Equal(got, want) {
		t.Fatalf("cross dtypes = %v, want %v", got, want)
	}
	want := map[string][]any{
		"a":       {int64(1), int64(1), int64(1), int64(2), int64(2), int64(2)},
		"k":       {"x", "x", "x", nil, nil, nil},
		"a_right": {10.0, 20.0, 30.0, 10.0, 20.0, 30.0},
		"r":       {true, false, nil, true, false, nil},
	}
	for name, w := range want {
		if got := columnValues(t, out, name); !reflect.DeepEqual(got, w) {
			t.Errorf("cross %s = %v, want %v", name, got, w)
		}
	}
	suffixed, err := left.Join(JoinInput{Other: right, How: JoinTypeCross, Suffix: "_r"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := suffixed.Columns(), []string{"a", "k", "a_r", "r"}; !slices.Equal(got, want) {
		t.Fatalf("cross suffix columns = %v, want %v", got, want)
	}

	emptyLeft := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{}, DType: dtypes.Int64},
		SeriesInput{Name: "k", Values: []any{}, DType: dtypes.String},
	)
	emptyRight := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{}, DType: dtypes.Float64},
		SeriesInput{Name: "r", Values: []any{}, DType: dtypes.Boolean},
	)
	for _, tc := range []struct {
		name        string
		left, right DataFrame
	}{
		{"empty left", emptyLeft, right},
		{"empty right", left, emptyRight},
		{"both empty", emptyLeft, emptyRight},
	} {
		out, err := tc.left.Join(JoinInput{Other: tc.right, How: JoinTypeCross})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if out.Height() != 0 {
			t.Errorf("%s: height %d", tc.name, out.Height())
		}
		if got, want := out.Columns(), []string{"a", "k", "a_right", "r"}; !slices.Equal(got, want) {
			t.Errorf("%s: columns = %v, want %v", tc.name, got, want)
		}
		if got, want := out.Dtypes(), []dtypes.DataType{dtypes.Int64, dtypes.String, dtypes.Float64, dtypes.Boolean}; !slices.Equal(got, want) {
			t.Errorf("%s: dtypes = %v, want %v", tc.name, got, want)
		}
		for _, s := range out.GetColumns() {
			if s.Column() == nil {
				t.Errorf("%s: column %s has no typed chunk", tc.name, s.Name())
			}
		}
	}
}

func TestAsofJoinNearestCharacterization(t *testing.T) {
	left := cov80Frame(t,
		SeriesInput{Name: "t", Values: []any{int64(5), int64(10), int64(20), int64(100)}},
	)
	right := cov80Frame(t,
		SeriesInput{Name: "t", Values: []any{int64(3), int64(7), int64(10), int64(12), int64(8)}},
		SeriesInput{Name: "v", Values: []any{"a", "b", "c", "d", "e"}},
	)
	cases := []struct {
		dir  string
		tol  int64
		want []any
	}{
		{"", 0, []any{"a", "c", "d", "d"}},
		{"backward", 0, []any{"a", "c", "d", "d"}},
		{"forward", 0, []any{"b", "c", nil, nil}},
		{"nearest", 0, []any{"a", "c", "d", "d"}},
		{"nearest", 2, []any{"a", "c", nil, nil}},
	}
	for _, tc := range cases {
		out, err := left.Join(JoinInput{Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"}, How: JoinTypeAsof, AsofDirection: tc.dir, AsofTolerance: tc.tol})
		if err != nil {
			t.Fatal(err)
		}
		if got := columnValues(t, out, "v"); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("asof %q tol=%d: v = %v, want %v", tc.dir, tc.tol, got, tc.want)
		}
		if got, want := out.Columns(), []string{"t", "t_right", "v"}; !slices.Equal(got, want) {
			t.Errorf("asof columns = %v, want %v", got, want)
		}
	}
	empty := cov80Frame(t, SeriesInput{Name: "t", Values: []any{}, DType: dtypes.Int64})
	out, err := empty.Join(JoinInput{Other: right, LeftOn: []string{"t"}, RightOn: []string{"t"}, How: JoinTypeAsof})
	if err != nil || out.Height() != 0 || out.Width() != 3 {
		t.Errorf("asof empty left: err=%v h=%d w=%d", err, out.Height(), out.Width())
	}
	if _, err := left.Join(JoinInput{Other: right, LeftOn: []string{"t"}, RightOn: []string{"nope"}, How: JoinTypeAsof}); err == nil || err.Error() != "join key nope not found" {
		t.Errorf("asof missing right key err = %v", err)
	}
	if _, err := left.Join(JoinInput{Other: right, LeftOn: []string{"nope"}, RightOn: []string{"t"}, How: JoinTypeAsof}); err == nil || err.Error() != "join key nope not found" {
		t.Errorf("asof missing left key err = %v", err)
	}
}

func TestKeyColumnErrorsCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "v", Values: []any{1.0, 2.0}},
	)
	other := cov80Frame(t, SeriesInput{Name: "g", Values: []any{int64(1)}})
	check := func(what string, err error, want string) {
		t.Helper()
		if err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", what, err, want)
		}
	}
	_, err := df.Unique("g", "nope")
	check("unique", err, "column nope not found")
	_, err = df.NUnique("nope")
	check("n_unique", err, "column nope not found")
	_, err = df.GroupBy("g", "nope").Agg(expr.Count())
	check("group by", err, "group key nope not found")
	_, err = df.Join(JoinInput{Other: other, LeftOn: []string{"g"}, RightOn: []string{"nope"}})
	check("join right", err, "join key nope not found")
	_, err = df.Join(JoinInput{Other: other, LeftOn: []string{"nope"}, RightOn: []string{"g"}})
	check("join left", err, "join key nope not found")
	_, err = df.evalOver(expr.Col("v"), "g,nope", "o")
	check("over", err, "partition column nope not found")
}

func TestInsertColumnCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "b", Values: []any{"x", "y"}},
		SeriesInput{Name: "c", Values: []any{1.0, 2.0}},
	)
	mk := func(name string, dt dtypes.DataType, vals ...any) series.Series {
		s, err := series.New(name, dt, vals)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	schemaNames := func(d DataFrame) []string {
		out := []string{}
		for _, f := range d.Schema() {
			out = append(out, f.Name+":"+string(f.Type))
		}
		return out
	}
	cases := []struct {
		index int
		col   series.Series
		want  []string
	}{
		{-5, mk("n", dtypes.Boolean, true, false), []string{"n:bool", "a:int64", "b:string", "c:float64"}},
		{99, mk("n", dtypes.Boolean, true, false), []string{"a:int64", "b:string", "c:float64", "n:bool"}},
		{1, mk("n", dtypes.Boolean, true, false), []string{"a:int64", "n:bool", "b:string", "c:float64"}},
		{0, mk("c", dtypes.String, "p", "q"), []string{"c:string", "a:int64", "b:string"}},
		{2, mk("a", dtypes.Int64, int64(7), int64(8)), []string{"b:string", "c:float64", "a:int64"}},
		{1, mk("b", dtypes.Boolean, true, true), []string{"a:int64", "b:bool", "c:float64"}},
	}
	for _, tc := range cases {
		out, err := df.InsertColumn(tc.index, tc.col)
		if err != nil {
			t.Fatal(err)
		}
		if got := schemaNames(out); !slices.Equal(got, tc.want) {
			t.Errorf("InsertColumn(%d, %s) schema = %v, want %v", tc.index, tc.col.Name(), got, tc.want)
		}
		wantOrder := make([]string, len(tc.want))
		for i, s := range tc.want {
			wantOrder[i], _, _ = strings.Cut(s, ":")
		}
		if got := out.Columns(); !slices.Equal(got, wantOrder) {
			t.Errorf("InsertColumn(%d, %s) order = %v", tc.index, tc.col.Name(), got)
		}
		if got := columnValues(t, out, tc.col.Name()); !reflect.DeepEqual(got, seriesValues(tc.col)) {
			t.Errorf("InsertColumn(%d, %s) values = %v", tc.index, tc.col.Name(), got)
		}
	}
	if _, err := df.InsertColumn(0, mk("n", dtypes.Int64, int64(1))); err == nil || err.Error() != "column n has invalid length" {
		t.Errorf("InsertColumn invalid length err = %v", err)
	}
	if got := df.Columns(); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("InsertColumn mutated source order: %v", got)
	}
}

func TestRowWindowClampsCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "id", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
		SeriesInput{Name: "s", Values: []any{"a", "b", nil, "d", "e"}},
	)
	ids := func(d DataFrame) []any { return columnValues(t, d, "id") }
	if got := ids(df.viewRows(-3, 100)); len(got) != 5 {
		t.Errorf("viewRows(-3,100) = %v", got)
	}
	if got := ids(df.viewRows(4, 2)); len(got) != 0 {
		t.Errorf("viewRows(4,2) = %v", got)
	}
	if got := ids(df.viewRows(1, 3)); !reflect.DeepEqual(got, []any{int64(1), int64(2)}) {
		t.Errorf("viewRows(1,3) = %v", got)
	}
	if got := ids(df.GatherEvery(0, -2)); !reflect.DeepEqual(got, ids(df)) {
		t.Errorf("GatherEvery(0,-2) = %v", got)
	}
	if got := ids(df.GatherEvery(2, 1)); !reflect.DeepEqual(got, []any{int64(1), int64(3)}) {
		t.Errorf("GatherEvery(2,1) = %v", got)
	}
	if got := ids(df.GatherEvery(-1, 9)); len(got) != 0 {
		t.Errorf("GatherEvery(-1,9) = %v", got)
	}
	if got := df.Sample(0, 7); got.Height() != 0 || got.Width() != 2 {
		t.Errorf("Sample(0) h=%d w=%d", got.Height(), got.Width())
	}
	want := []any{int64(2), int64(1), int64(3), int64(0), int64(4)}
	if got := ids(df.Sample(100, 7)); !reflect.DeepEqual(got, want) {
		t.Errorf("Sample(100,7) = %v, want %v", got, want)
	}
	if got := ids(df.Sample(3, 7)); !reflect.DeepEqual(got, want[:3]) {
		t.Errorf("Sample(3,7) = %v, want %v", got, want[:3])
	}
	if got, want := df.Glimpse(0), "rows=5 cols=2\nid:int64 s:string \n[0] id=0 s=a \n[1] id=1 s=b \n[2] id=2 s=<nil> \n[3] id=3 s=d \n[4] id=4 s=e"; got != want {
		t.Errorf("Glimpse(0) = %q", got)
	}
	if got, want := df.Glimpse(2), "rows=5 cols=2\nid:int64 s:string \n[0] id=0 s=a \n[1] id=1 s=b"; got != want {
		t.Errorf("Glimpse(2) = %q", got)
	}
}

func TestEvalRollingRowWiseCharacterization(t *testing.T) {
	boxed := series.FromColumn("b", chunk.NewBoxed(dtypes.List, []any{int64(1), 2.5, nil, "x", 4.0, int64(-1)}, nil))
	df, err := New(NewInput{Series: []series.Series{boxed}})
	if err != nil {
		t.Fatal(err)
	}
	window1 := []any{1.0, 2.5, nil, nil, 4.0, -1.0}
	cases := []struct {
		op   string
		want []any
	}{
		{"rolling_sum:3", []any{1.0, 3.5, 3.5, 2.5, 4.0, 3.0}},
		{"rolling_mean:3", []any{1.0, 1.75, 1.75, 2.5, 4.0, 1.5}},
		{"rolling_mean:10", []any{1.0, 1.75, 1.75, 1.75, 2.5, 1.625}},
		{"rolling_min:3", []any{1.0, 1.0, 1.0, 2.5, 4.0, -1.0}},
		{"rolling_max:3", []any{1.0, 2.5, 2.5, 2.5, 4.0, 4.0}},
		{"rolling_std:2", []any{0.0, 0.75, 0.0, nil, 0.0, 2.5}},
		{"rolling_var:2", []any{0.0, 0.5625, 0.0, nil, 0.0, 6.25}},
		{"rolling_median:2", []any{1.0, 2.5, 2.5, nil, 4.0, -1.0}},
		{"rolling_sum", window1},
		{"rolling_sum:", window1},
		{"rolling_sum:0", window1},
		{"rolling_sum:-1", window1},
		{"rolling_sum:abc", window1},
		{"rolling_sum:2:3", window1},
	}
	for _, tc := range cases {
		s, err := df.evalRolling(expr.Col("b"), tc.op, "out")
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if s.Name() != "out" || s.DataType() != dtypes.Float64 {
			t.Errorf("%s: name=%s dtype=%s", tc.op, s.Name(), s.DataType())
		}
		if got := seriesValues(s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.op, got, tc.want)
		}
	}
}

func TestEvalOverCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{"a", "b", "a", "b"}},
		SeriesInput{Name: "h", Values: []any{int64(1), int64(1), int64(1), int64(2)}},
		SeriesInput{Name: "v", Values: []any{int64(4), int64(3), nil, int64(1)}},
	)
	cases := []struct {
		target expr.Expr
		spec   string
		want   []any
	}{
		{expr.Col("v"), "g", []any{int64(4), int64(3), nil, int64(1)}},
		{expr.Col("v"), "  ", []any{int64(4), int64(3), nil, int64(1)}},
		{expr.Col("v"), " , ", []any{int64(4), int64(3), nil, int64(1)}},
		{expr.Col("v").CumSum(), " g , ,h ", []any{4.0, 7.0, 11.0, 8.0}},
		{expr.Col("v").CumCount(), "g", []any{int64(1), int64(1), int64(2), int64(2)}},
		{expr.Col("v").Rank(), "g,h", []any{int64(1), int64(1), int64(2), int64(1)}},
		{expr.Col("v").Reverse(), "g", []any{int64(1), nil, int64(3), int64(4)}},
	}
	for _, tc := range cases {
		s, err := df.evalOver(tc.target, tc.spec, "o")
		if err != nil {
			t.Fatalf("over %s %q: %v", tc.target.Name(), tc.spec, err)
		}
		if got := seriesValues(s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("over %s/%s %q = %v, want %v", tc.target.Op(), tc.target.Name(), tc.spec, got, tc.want)
		}
	}
}

func TestInTemporalBoundsCharacterization(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	points := []time.Time{start.Add(-time.Second), start, start.Add(time.Minute), end, end.Add(time.Second)}
	want := map[string][]bool{
		"":      {false, true, true, false, false},
		"left":  {false, true, true, false, false},
		"right": {false, false, true, true, false},
		"both":  {false, true, true, true, false},
		"none":  {false, false, true, false, false},
		"other": {false, true, true, false, false},
	}
	for closed, w := range want {
		for i, ts := range points {
			if got := inTemporalBounds(ts, start, end, closed); got != w[i] {
				t.Errorf("inTemporalBounds(%v, %q) = %v, want %v", ts, closed, got, w[i])
			}
		}
	}
	// A monotonic reading on one side compares by wall clock.
	now := time.Now()
	if !inTemporalBounds(now, now.Round(0), now.Add(time.Second), "left") {
		t.Error("monotonic vs wall start should be in bounds")
	}
}

func TestFusedResultCharacterization(t *testing.T) {
	empty := colReduction{}
	full := colReduction{sum: 6, min: -1, max: 4, count: 3}
	cases := []struct {
		op    string
		r     colReduction
		want  any
		dtype dtypes.DataType
	}{
		{"count", empty, int64(0), dtypes.Int64},
		{"count", full, int64(3), dtypes.Int64},
		{"mean", empty, nil, dtypes.Float64},
		{"mean", full, 2.0, dtypes.Float64},
		{"min", empty, nil, dtypes.Float64},
		{"min", full, -1.0, dtypes.Float64},
		{"max", empty, nil, dtypes.Float64},
		{"max", full, 4.0, dtypes.Float64},
		{"sum", empty, nil, dtypes.Float64},
		{"sum", full, 6.0, dtypes.Float64},
		{"other", full, 6.0, dtypes.Float64},
	}
	for _, tc := range cases {
		got, dt := fusedResult(tc.op, tc.r)
		if got != tc.want || dt != tc.dtype {
			t.Errorf("fusedResult(%s, %+v) = %v, %s; want %v, %s", tc.op, tc.r, got, dt, tc.want, tc.dtype)
		}
	}
}

func TestAggregateValuesCharacterization(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		vals []any
		agg  string
		want any
	}{
		{[]any{"q", int64(1)}, "", "q"},
		{[]any{nil, int64(1)}, "first", nil},
		{[]any{3.0, nan, 1.0}, "min", 1.0},
		{[]any{3.0, nan, 5.0}, "max", 5.0},
		{[]any{"b", "c", "a"}, "min", "a"},
		{[]any{"b", "c", "a"}, "max", "c"},
		{[]any{int64(2), 1.0, int64(5)}, "max", int64(5)},
		{[]any{int64(2), 1.0, int64(1)}, "min", int64(1)},
		{[]any{nil, int64(2)}, "min", nil},
		{[]any{int64(2), int64(2)}, "max", int64(2)},
		{[]any{int64(1), 2.5}, "sum", 3.5},
		{[]any{"x"}, "mean", nil},
	}
	for _, tc := range cases {
		got := aggregateValues(tc.vals, tc.agg)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("aggregateValues(%v, %q) = %v, want %v", tc.vals, tc.agg, got, tc.want)
		}
	}
}

func TestFoldCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: []any{1.0, nil, 5.0, nil}},
		SeriesInput{Name: "i", Values: []any{int64(2), nil, nil, nil}},
		SeriesInput{Name: "b", Values: []any{3.0, nil, 2.0, 4.0}},
		SeriesInput{Name: "s", Values: []any{"x", "y", "z", "w"}},
	)
	cases := []struct {
		op   string
		want []any
	}{
		{"sum", []any{6.0, nil, 7.0, 4.0}},
		{"other", []any{6.0, nil, 7.0, 4.0}},
		{"max", []any{3.0, nil, 5.0, 4.0}},
		{"min", []any{1.0, nil, 2.0, 4.0}},
	}
	for _, tc := range cases {
		out, err := df.Fold(tc.op, []string{"a", "s", "i", "b"}, "")
		if err != nil {
			t.Fatal(err)
		}
		if got := columnValues(t, out, "fold"); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Fold(%s) = %v, want %v", tc.op, got, tc.want)
		}
	}
	if _, err := df.Fold("sum", []string{"a", "nope"}, "x"); err == nil || err.Error() != "column nope not found" {
		t.Errorf("Fold missing column err = %v", err)
	}
}

func TestExplodeCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		SeriesInput{Name: "l", Values: []any{[]any{int64(1), int64(2)}, nil, []any{}}},
		SeriesInput{Name: "m", Values: []any{[]any{"a"}, []any{"b", "c"}, nil}},
	)
	out, err := df.Explode("l", "m")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]any{
		"id": {int64(1), int64(1), int64(2), int64(2), int64(3)},
		"l":  {int64(1), int64(2), nil, nil, nil},
		"m":  {"a", nil, "b", "c", nil},
	}
	for name, w := range want {
		if got := columnValues(t, out, name); !reflect.DeepEqual(got, w) {
			t.Errorf("explode %s = %v, want %v", name, got, w)
		}
	}
	if got, want := out.Dtypes(), []dtypes.DataType{dtypes.Int64, dtypes.Int64, dtypes.String}; !slices.Equal(got, want) {
		t.Errorf("explode dtypes = %v, want %v", got, want)
	}
}

func TestFlagsCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "asc", Values: []any{int64(1), int64(2), int64(2)}},
		SeriesInput{Name: "desc", Values: []any{3.0, math.NaN(), 1.0}},
		SeriesInput{Name: "nulls", Values: []any{"a", nil, "b"}},
		SeriesInput{Name: "uns", Values: []any{int64(2), int64(1), int64(3)}},
	)
	// compareAny treats NaN as equal to everything, so "desc" reads as sorted.
	want := map[string]map[string]bool{
		"asc":   {"sorted_asc": true, "has_nulls": false, "has_nan": false, "is_monotonic": true},
		"desc":  {"sorted_asc": true, "has_nulls": false, "has_nan": true, "is_monotonic": true},
		"uns":   {"sorted_asc": false, "has_nulls": false, "has_nan": false, "is_monotonic": false},
		"nulls": {"sorted_asc": true, "has_nulls": true, "has_nan": false, "is_monotonic": true},
	}
	if got := df.Flags(); !reflect.DeepEqual(got, want) {
		t.Errorf("Flags = %v, want %v", got, want)
	}
}

func TestGroupByDistinctAndTypedPathsCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "g", Values: []any{int64(1), int64(1), int64(1), int64(2), int64(2)}},
		SeriesInput{Name: "s", Values: []any{"a", nil, "a", nil, nil}},
		SeriesInput{Name: "f", Values: []any{1.5, 2.5, nil, nil, nil}},
		SeriesInput{Name: "i", Values: []any{int64(4), nil, int64(2), nil, nil}},
	)
	out, err := df.GroupBy("g").Agg(
		expr.NUnique(expr.Col("s")).Alias("nu"),
		expr.CountDistinct(expr.Col("s")).Alias("cd"),
		expr.Sum(expr.Col("f")).Alias("fsum"),
		expr.Mean(expr.Col("f")).Alias("fmean"),
		expr.Sum(expr.Col("i")).Alias("isum"),
		expr.Mean(expr.Col("i")).Alias("imean"),
		expr.Min(expr.Col("i")).Alias("imin"),
		expr.Max(expr.Col("f")).Alias("fmax"),
		expr.Median(expr.Col("i")).Alias("imed"),
		expr.Median(expr.Col("f")).Alias("fmed"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]any{
		"nu":    {int64(2), int64(1)},
		"cd":    {int64(1), int64(0)},
		"fsum":  {4.0, nil},
		"fmean": {2.0, nil},
		"isum":  {int64(6), nil},
		"imean": {3.0, nil},
		"imin":  {int64(2), nil},
		"fmax":  {2.5, nil},
		"imed":  {3.0, nil},
		"fmed":  {2.0, nil},
	}
	for name, w := range want {
		if got := columnValues(t, out, name); !reflect.DeepEqual(got, w) {
			t.Errorf("%s = %v, want %v", name, got, w)
		}
	}
	g := df.GroupBy("g")
	if _, err := g.evalAgg(expr.Sum(expr.Col("nope")), []int{0, 1}); err == nil {
		t.Error("sum over missing column should error")
	}
	if _, err := g.evalAgg(expr.Min(expr.Col("nope")), []int{0, 1}); err == nil {
		t.Error("min over missing column should error")
	}
	if v, err := g.evalAgg(expr.Max(expr.Col("s")), []int{0, 1, 2}); err != nil || v != "a" {
		t.Errorf("max over string column = %v, %v", v, err)
	}
	if v, err := g.evalAgg(expr.Median(expr.Col("f")), []int{0, 1, 2}); err != nil || v != 2.0 {
		t.Errorf("median = %v, %v", v, err)
	}
	if v, err := g.evalAgg(expr.Median(expr.Col("i")), []int{0, 2, 3}); err != nil || v != 3.0 {
		t.Errorf("median even = %v, %v", v, err)
	}
	if v, err := g.evalAgg(expr.Median(expr.Col("f")), []int{3, 4}); err != nil || v != nil {
		t.Errorf("median empty = %v, %v", v, err)
	}
}

func TestResolveParallelAggsCharacterization(t *testing.T) {
	df := cov80Frame(t,
		SeriesInput{Name: "f", Values: []any{1.0, nil}},
		SeriesInput{Name: "i", Values: []any{int64(1), int64(2)}},
		SeriesInput{Name: "s", Values: []any{"a", "b"}},
	)
	g := df.GroupBy("s")
	for _, e := range []expr.Expr{
		expr.Sum(expr.Col("f").Add(expr.Lit(1.0))),
		expr.Sum(expr.Col("nope")),
		expr.Max(expr.Col("s")),
		expr.NUnique(expr.Col("i")),
		expr.Col("i"),
	} {
		if _, ok := g.resolveParallelAggs([]expr.Expr{e}); ok {
			t.Errorf("resolveParallelAggs(%s/%s) should decline", e.Op(), e.Name())
		}
	}
	specs, ok := g.resolveParallelAggs([]expr.Expr{
		expr.Count().Alias("n"),
		expr.Sum(expr.Col("f")).Alias("fs"),
		expr.Mean(expr.Col("i")).Alias("im"),
		expr.Min(expr.Col("i")).Alias("imin"),
		expr.Max(expr.Col("f")).Alias("fmax"),
	})
	if !ok {
		t.Fatal("resolveParallelAggs declined supported aggs")
	}
	type view struct {
		kind     aggKind
		name     string
		dtype    dtypes.DataType
		colFloat bool
		nulls    bool
	}
	got := make([]view, len(specs))
	for k, s := range specs {
		got[k] = view{s.kind, s.name, s.dtype, s.colFloat, s.nulls != nil}
	}
	want := []view{
		{aggCount, "n", dtypes.Int64, false, false},
		{aggSum, "fs", dtypes.Float64, true, true},
		{aggMean, "im", dtypes.Float64, false, false},
		{aggMin, "imin", dtypes.Int64, false, false},
		{aggMax, "fmax", dtypes.Float64, true, true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("specs = %+v, want %+v", got, want)
	}
}

func TestParallelMaskPathsCharacterization(t *testing.T) {
	n := 1000
	av := make([]any, n)
	bv := make([]any, n)
	cv := make([]any, n)
	for i := range n {
		cv[i] = float64(i%13) - 6
		av[i] = cv[i]
		if i%9 == 0 {
			av[i] = nil
		}
		bv[i] = int64(i)
	}
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: av},
		SeriesInput{Name: "b", Values: bv},
		SeriesInput{Name: "c", Values: cv},
	)
	cols := df.chunkColumns()
	pred := expr.Col("c").Gt(expr.Lit(0.0)).Or(expr.Col("b").Lt(expr.Lit(int64(10))))
	plan, ok := evalbatch.Compile(pred)
	if !ok {
		t.Fatal("compile predicate")
	}
	seq, _, err := plan.EvalBool(cols, n)
	if err != nil {
		t.Fatal(err)
	}
	row, err := df.Filter(pred)
	if err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{1, 3, 4, 7} {
		par, ok := filterMaskParallel(plan, cols, n, workers)
		if !ok || !slices.Equal(par, seq) {
			t.Errorf("filterMaskParallel workers=%d ok=%v mismatch", workers, ok)
		}
		fused, ok, err := df.filterFused(plan, cols, workers)
		if err != nil || !ok {
			t.Fatalf("filterFused workers=%d ok=%v err=%v", workers, ok, err)
		}
		if eq, _ := fused.Equals(row); !eq {
			t.Errorf("filterFused workers=%d differs from Filter", workers)
		}
	}
	// Kleene OR yields null where a is null and b >= 10: every path declines.
	nullable, ok := evalbatch.Compile(expr.Col("a").Gt(expr.Lit(0.0)).Or(expr.Col("b").Lt(expr.Lit(int64(10)))))
	if !ok {
		t.Fatal("compile nullable predicate")
	}
	bad, ok := evalbatch.Compile(expr.Col("a").Add(expr.Lit(1.0)))
	if !ok {
		t.Fatal("compile non-bool")
	}
	for _, p := range []*evalbatch.Plan{nullable, bad} {
		if _, ok := filterMaskParallel(p, cols, n, 4); ok {
			t.Error("filterMaskParallel should decline")
		}
		if out, ok, err := df.filterFused(p, cols, 4); ok || err != nil || out.Width() != 0 {
			t.Errorf("filterFused decline: ok=%v err=%v w=%d", ok, err, out.Width())
		}
		if _, ok := df.fusedReduceParallel(p, cols, []string{"c"}, 4); ok {
			t.Error("fusedReduceParallel should decline")
		}
	}
	red, ok := df.fusedReduceParallel(plan, map[string]*chunk.Column{"b": cols["b"], "c": cols["c"]}, []string{"c"}, 4)
	if !ok || len(red) != 1 {
		t.Fatalf("fusedReduceParallel: ok=%v len=%d", ok, len(red))
	}
	want, _ := df.fusedReduce(plan, cols, []string{"c"})
	if red[0] != want[0] {
		t.Errorf("fusedReduceParallel = %+v, want %+v", red[0], want[0])
	}
}

func TestDropNullsFusedCharacterization(t *testing.T) {
	n := 700
	av := make([]any, n)
	bv := make([]any, n)
	for i := range n {
		av[i] = int64(i)
		if i%4 == 0 {
			av[i] = nil
		}
		bv[i] = "v"
		if i%11 == 0 {
			bv[i] = nil
		}
	}
	df := cov80Frame(t,
		SeriesInput{Name: "a", Values: av},
		SeriesInput{Name: "b", Values: bv},
	)
	cols := df.chunkColumns()
	for _, targets := range [][]string{{"a"}, {"b"}, {"a", "b"}} {
		want := df.gatherRows(keepFromDropped(func() []bool {
			dropped := make([]bool, n)
			for _, name := range targets {
				for r, isNull := range cols[name].Nulls() {
					dropped[r] = dropped[r] || isNull
				}
			}
			return dropped
		}(), n))
		for _, workers := range []int{2, 5} {
			got := df.dropNullsFused(targets, cols, workers)
			if eq, _ := got.Equals(want); !eq {
				t.Errorf("dropNullsFused(%v, %d) mismatch", targets, workers)
			}
		}
	}
}
