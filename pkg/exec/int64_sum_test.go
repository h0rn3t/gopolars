package exec

import (
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

// TestInt64SumsAreExact checks the frame and window sums of Int64 values above
// 2^53, which a float64 accumulator rounds.
func TestInt64SumsAreExact(t *testing.T) {
	t.Parallel()

	const want = int64(9007199254740995)
	df := mustFrame(t,
		frame.SeriesInput{Name: "g", Values: []any{"a", "a"}},
		frame.SeriesInput{Name: "v", Values: []any{int64(9007199254740993), int64(2)}},
	)

	summed, err := aggregateFrame(df, "sum", nil)
	if err != nil {
		t.Fatalf("aggregateFrame(sum) error = %v", err)
	}
	if got := colValues(t, summed, "v"); !slices.Equal(got, []any{want}) {
		t.Errorf("aggregateFrame(sum) v = %v, want [%d]", got, want)
	}

	windowed, err := applyWindows(df, []logical.WindowSpec{{Func: "sum", Target: "v", Alias: "w", PartitionBy: []string{"g"}}})
	if err != nil {
		t.Fatalf("applyWindows(sum) error = %v", err)
	}
	if got := colValues(t, windowed, "w"); !slices.Equal(got, []any{want, want}) {
		t.Errorf("applyWindows(sum) w = %v, want [%d %d]", got, want, want)
	}
}
