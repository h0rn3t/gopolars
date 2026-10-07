package arrow_test

import (
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// TestFromTableColumnOrder converts every table twenty times: an order taken
// from map iteration differs between conversions.
func TestFromTableColumnOrder(t *testing.T) {
	var cols []frame.SeriesInput
	for i, name := range []string{"z", "a", "m", "b", "c"} {
		cols = append(cols, frame.SeriesInput{Name: name, Values: []any{int64(i)}})
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	one := []any{int64(1)}
	cases := []struct {
		name  string
		table iarrow.Table
		want  []string
	}{
		{name: "round trip", table: iarrow.ToTable(df), want: []string{"z", "a", "m", "b", "c"}},
		{
			name:  "no stored order",
			table: iarrow.Table{Columns: map[string][]any{"b": one, "a": one}},
			want:  []string{"a", "b"},
		},
		{
			name:  "projected-out name",
			table: iarrow.Table{Columns: map[string][]any{"z": one, "c": one}, Names: []string{"z", "a", "c"}},
			want:  []string{"z", "c"},
		},
		{
			name:  "columns added after ToTable",
			table: iarrow.Table{Columns: map[string][]any{"b": one, "a": one, "y": one, "x": one}, Names: []string{"b", "a"}},
			want:  []string{"b", "a", "x", "y"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for range 20 {
				got, err := iarrow.FromTable(tc.table)
				if err != nil {
					t.Fatalf("FromTable(%v) error = %v, want nil", tc.table, err)
				}
				if !slices.Equal(got.Columns(), tc.want) {
					t.Fatalf("FromTable(%v) columns = %v, want %v", tc.table, got.Columns(), tc.want)
				}
			}
		})
	}
}
