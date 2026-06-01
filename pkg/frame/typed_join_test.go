package frame

import (
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

func mkJoinFrames() (DataFrame, DataFrame) {
	left, _ := New(NewInput{Series: []series.Series{
		series.FromInt64("k", []int64{1, 2, 3, 4}, nil),
		series.FromString("lv", []string{"a", "b", "c", "d"}, nil),
	}})
	right, _ := New(NewInput{Series: []series.Series{
		series.FromInt64("k", []int64{2, 3, 3, 5}, nil),
		series.FromString("rv", []string{"x", "y", "z", "w"}, nil),
	}})
	return left, right
}

// joinRowSet renders each output row as a string so results can be compared
// independent of ordering.
func joinRowSet(t *testing.T, df DataFrame) []string {
	t.Helper()
	rows := make([]string, 0, df.height)
	for i := 0; i < df.height; i++ {
		s := ""
		for _, name := range df.order {
			v := df.cols[name].Value(i)
			s += name + "="
			if v == nil {
				s += "<null>"
			} else {
				s += stringifyJoinVal(v)
			}
			s += ";"
		}
		rows = append(rows, s)
	}
	sort.Strings(rows)
	return rows
}

func stringifyJoinVal(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int64:
		return string(rune('0' + int(t)))
	default:
		return "?"
	}
}

func TestJoinInnerColumnar(t *testing.T) {
	left, right := mkJoinFrames()
	got, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeInner})
	if err != nil {
		t.Fatal(err)
	}
	// k=2 (1 match), k=3 (2 matches) -> 3 rows.
	if got.height != 3 {
		t.Fatalf("inner rows = %d, want 3", got.height)
	}
	// Output keeps both key columns (left k, right k_right) plus lv, rv — same
	// columns as the prior per-cell implementation.
	if len(got.order) != 4 {
		t.Fatalf("columns = %v, want [k lv k_right rv]", got.order)
	}
}

func TestJoinLeftColumnarNullFill(t *testing.T) {
	left, right := mkJoinFrames()
	got, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeLeft})
	if err != nil {
		t.Fatal(err)
	}
	// All 4 left rows, k=3 doubles -> 5 rows; unmatched (k=1,4) get null rv.
	if got.height != 5 {
		t.Fatalf("left rows = %d, want 5", got.height)
	}
	nullRv := 0
	for i := 0; i < got.height; i++ {
		if got.cols["rv"].Value(i) == nil {
			nullRv++
		}
	}
	if nullRv != 2 {
		t.Errorf("expected 2 null rv (unmatched left rows), got %d", nullRv)
	}
}

func TestJoinRightFullColumnar(t *testing.T) {
	left, right := mkJoinFrames()
	rj, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeRight})
	if err != nil {
		t.Fatal(err)
	}
	// right rows: k=2(1),3(2),3,5(unmatched) -> matches: 2->1, 3->2 (each of 2 right 3s), 5->null left.
	// left k=3 has 1 row; right has two k=3 rows -> 2 pairs. plus k=2 ->1, k=5 -> 1 (null left). total 4.
	if rj.height != 4 {
		t.Fatalf("right join rows = %d, want 4", rj.height)
	}
	fj, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeFull})
	if err != nil {
		t.Fatal(err)
	}
	// full: inner matches (3) + unmatched left (k=1,4 -> 2) + unmatched right (k=5 -> 1) = 6.
	if fj.height != 6 {
		t.Fatalf("full join rows = %d, want 6", fj.height)
	}
}

func TestJoinSemiAntiColumnar(t *testing.T) {
	left, right := mkJoinFrames()
	semi, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeSemi})
	if err != nil {
		t.Fatal(err)
	}
	// left rows with a match: k=2,3 -> 2 rows. Only left columns.
	if semi.height != 2 {
		t.Fatalf("semi rows = %d, want 2", semi.height)
	}
	if len(semi.order) != 2 { // k, lv only
		t.Errorf("semi columns = %v, want left-only", semi.order)
	}
	anti, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeAnti})
	if err != nil {
		t.Fatal(err)
	}
	// left rows without a match: k=1,4 -> 2 rows.
	if anti.height != 2 {
		t.Fatalf("anti rows = %d, want 2", anti.height)
	}
}

func TestJoinCrossColumnar(t *testing.T) {
	left, right := mkJoinFrames()
	got, err := left.Join(JoinInput{Other: right, How: JoinTypeCross})
	if err != nil {
		t.Fatal(err)
	}
	if got.height != left.height*right.height {
		t.Fatalf("cross rows = %d, want %d", got.height, left.height*right.height)
	}
}

func TestJoinStringKeyColumnar(t *testing.T) {
	left, _ := New(NewInput{Series: []series.Series{
		series.FromString("g", []string{"a", "b", "c"}, nil),
		series.FromFloat64("v", []float64{1, 2, 3}, nil),
	}})
	right, _ := New(NewInput{Series: []series.Series{
		series.FromString("g", []string{"b", "c", "c"}, nil),
		series.FromFloat64("w", []float64{10, 20, 30}, nil),
	}})
	got, err := left.Join(JoinInput{Other: right, LeftOn: []string{"g"}, RightOn: []string{"g"}, How: JoinTypeInner})
	if err != nil {
		t.Fatal(err)
	}
	// b->1, c->2 = 3 rows.
	if got.height != 3 {
		t.Fatalf("string-key inner rows = %d, want 3", got.height)
	}
	_ = joinRowSet(t, got)
}

func BenchmarkJoinColumnar(b *testing.B) {
	// 1M-row fact frame joined against a 1000-row dimension frame on an int64
	// key — completes (output ~1M rows) and exercises the typed gather path.
	n := 1_000_000
	dim := 1000
	k := make([]int64, n)
	v := make([]float64, n)
	for i := range k {
		k[i] = int64(i % dim)
		v[i] = float64(i)
	}
	left, _ := New(NewInput{Series: []series.Series{
		series.FromInt64("k", k, nil),
		series.FromFloat64("v", v, nil),
	}})
	dk := make([]int64, dim)
	dv := make([]float64, dim)
	for i := range dk {
		dk[i] = int64(i)
		dv[i] = float64(i) * 10
	}
	right, _ := New(NewInput{Series: []series.Series{
		series.FromInt64("k", dk, nil),
		series.FromFloat64("dv", dv, nil),
	}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := left.Join(JoinInput{Other: right, LeftOn: []string{"k"}, RightOn: []string{"k"}, How: JoinTypeInner}); err != nil {
			b.Fatal(err)
		}
	}
}
