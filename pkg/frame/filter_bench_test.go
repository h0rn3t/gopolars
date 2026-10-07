package frame

import (
	"fmt"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

func BenchmarkFilterIsIn(b *testing.B) {
	df := aggBenchFrame(b, 100_000)
	ids := make([]int64, 50)
	names := make([]string, 50)
	for i := range ids {
		ids[i] = int64(i * 7)
		names[i] = fmt.Sprintf("zone-%d", i*5)
	}
	cases := []struct {
		name string
		pred expr.Expr
	}{
		{"int64", expr.Col("id").IsIn(expr.Lit(ids))},
		{"string", expr.Col("name").IsIn(expr.Lit(names))},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := df.Filter(c.pred); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFilterNullablePredicate filters by a predicate that is null where
// iw is null and v is not above the threshold (null OR false).
func BenchmarkFilterNullablePredicate(b *testing.B) {
	df := aggBenchFrame(b, 100_000)
	pred := expr.Col("iw").Eq(expr.Lit(int64(1))).Or(expr.Col("v").Gt(expr.Lit(100_000.0)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := df.Filter(pred); err != nil {
			b.Fatal(err)
		}
	}
}
