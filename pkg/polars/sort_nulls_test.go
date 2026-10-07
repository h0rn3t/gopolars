package polars

import (
	"fmt"
	"slices"
	"testing"
)

func TestLazySortHonorsNullsLast(t *testing.T) {
	t.Parallel()
	d := mscFrame(t, mscCol("k", int64(2), nil, int64(3), int64(1)))
	for _, in := range []SortInput{
		{By: []string{"k"}, NullsLast: true},
		{By: []string{"k"}, NullsLast: true, Descending: []bool{true}},
		{By: []string{"k"}, Descending: []bool{true}},
	} {
		t.Run(fmt.Sprintf("%v", in), func(t *testing.T) {
			t.Parallel()
			eager, err := d.Sort(in)
			if err != nil {
				t.Fatalf("Sort(%+v) error = %v", in, err)
			}
			lazy, err := d.Lazy().Sort(in).Collect(t.Context())
			if err != nil {
				t.Fatalf("Lazy().Sort(%+v).Collect() error = %v", in, err)
			}
			want := frameColumnValues(t, eager, "k")
			if got := frameColumnValues(t, lazy, "k"); !slices.Equal(got, want) {
				t.Errorf("Lazy().Sort(%+v) k = %v, want %v (eager)", in, got, want)
			}
			if gotNullLast := want[len(want)-1] == nil; gotNullLast != in.NullsLast {
				t.Errorf("Sort(%+v) k = %v, null last = %v, want %v", in, want, gotNullLast, in.NullsLast)
			}
		})
	}
}
