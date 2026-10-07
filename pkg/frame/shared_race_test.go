package frame

import (
	"reflect"
	"sync"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/expr"
)

// TestConcurrentDerivationsOfOneFrame derives frames that share the source's
// columns from several goroutines at once; run with -race. Each result must
// equal the result of the same operation run alone.
func TestConcurrentDerivationsOfOneFrame(t *testing.T) {
	t.Parallel()
	build := func() DataFrame {
		return mustFrame(t,
			SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			SeriesInput{Name: "b", Values: []any{1.5, nil, 3.5, nil}},
			SeriesInput{Name: "s", Values: []any{"w", "x", "y", "z"}},
		)
	}
	ops := []struct {
		name string
		run  func(DataFrame) (DataFrame, error)
	}{
		{"select with alias", func(d DataFrame) (DataFrame, error) {
			return d.Select(expr.Col("a").Alias("x"), expr.Col("s"))
		}},
		{"with_columns", func(d DataFrame) (DataFrame, error) {
			return d.WithColumns(expr.Col("a").Alias("a2"))
		}},
		{"limit", func(d DataFrame) (DataFrame, error) { return d.Limit(2), nil }},
		{"rename", func(d DataFrame) (DataFrame, error) {
			return d.Rename(map[string]string{"a": "z"})
		}},
		{"fill_null", func(d DataFrame) (DataFrame, error) { return d.FillNull(0.0) }},
	}
	want := make([][]map[string]any, len(ops))
	for i, op := range ops {
		out, err := op.run(build())
		if err != nil {
			t.Fatalf("%s alone: %v", op.name, err)
		}
		want[i] = out.ToDicts()
	}

	df := build()
	const rounds = 4
	got := make([][]map[string]any, len(ops)*rounds)
	errs := make([]error, len(ops)*rounds)
	var wg sync.WaitGroup
	for r := range rounds {
		for i, op := range ops {
			wg.Go(func() {
				out, err := op.run(df)
				errs[r*len(ops)+i] = err
				got[r*len(ops)+i] = out.ToDicts()
			})
		}
	}
	wg.Wait()
	for k, rows := range got {
		op := ops[k%len(ops)]
		if errs[k] != nil {
			t.Errorf("%s concurrently: error = %v", op.name, errs[k])
			continue
		}
		if !reflect.DeepEqual(rows, want[k%len(ops)]) {
			t.Errorf("%s concurrently = %v, want %v", op.name, rows, want[k%len(ops)])
		}
	}
}
