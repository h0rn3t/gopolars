package micro

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// The float-key benchmarks measure the hash paths that key on float64Key:
// single-key group ids, composite (byte-encoded) group ids, the packed join
// key and the byte-encoded join key. Run before and after a change to the key
// encoding with:
//
//	go test ./bench/micro -run '^$' -bench 'FloatKey' -count 6 -benchtime 300ms

const (
	floatKeyRows     = 1_000_000
	floatKeyDistinct = 1000
)

// floatKeyFrame builds a frame with a float64 key f (floatKeyDistinct values,
// zero included), an int64 key i and a float64 value v.
func floatKeyFrame(b *testing.B, n int) frame.DataFrame {
	b.Helper()
	f := make([]float64, n)
	i64 := make([]int64, n)
	v := make([]float64, n)
	for r := range n {
		f[r] = float64(r%floatKeyDistinct) * 0.5
		i64[r] = int64(r % 7)
		v[r] = float64(r)
	}
	df, err := frame.New(frame.NewInput{Series: []series.Series{
		series.FromFloat64("f", f, nil),
		series.FromInt64("i", i64, nil),
		series.FromFloat64("v", v, nil),
	}})
	if err != nil {
		b.Fatalf("frame.New: %v", err)
	}
	return df
}

func BenchmarkGroupByFloatKey_1M(b *testing.B) {
	df := floatKeyFrame(b, floatKeyRows)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.GroupBy("f").Agg(expr.Sum(expr.Col("v"))); err != nil {
			b.Fatalf("GroupBy(f): %v", err)
		}
	}
}

func BenchmarkGroupByFloatKey2_1M(b *testing.B) {
	df := floatKeyFrame(b, floatKeyRows)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.GroupBy("f", "i").Agg(expr.Sum(expr.Col("v"))); err != nil {
			b.Fatalf("GroupBy(f, i): %v", err)
		}
	}
}

func BenchmarkJoinFloatKey_1M(b *testing.B) {
	left := floatKeyFrame(b, floatKeyRows)
	right := floatKeyFrame(b, floatKeyDistinct)
	in := frame.JoinInput{Other: right, LeftOn: []string{"f"}, RightOn: []string{"f"}, How: frame.JoinTypeInner}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := left.Join(in); err != nil {
			b.Fatalf("Join(f): %v", err)
		}
	}
}

func BenchmarkJoinFloatKey2_1M(b *testing.B) {
	left := floatKeyFrame(b, floatKeyRows)
	right := floatKeyFrame(b, floatKeyDistinct)
	in := frame.JoinInput{Other: right, LeftOn: []string{"f", "i"}, RightOn: []string{"f", "i"}, How: frame.JoinTypeInner}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := left.Join(in); err != nil {
			b.Fatalf("Join(f, i): %v", err)
		}
	}
}
