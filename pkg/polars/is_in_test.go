package polars

import (
	"slices"
	"testing"
)

func TestFilterIsInTypedSlice(t *testing.T) {
	t.Parallel()
	d := mscFrame(t, mscCol("id", int64(1), int64(2), int64(3)), mscCol("name", "a", "b", "c"))
	tests := []struct {
		name string
		pred Expr
		want []any
	}{
		{"[]int64", Col("id").IsIn(Lit([]int64{1, 3})), []any{int64(1), int64(3)}},
		{"[]any", Col("id").IsIn(Lit([]any{int64(1), int64(3)})), []any{int64(1), int64(3)}},
		{"[]string", Col("name").IsIn(Lit([]string{"b"})), []any{int64(2)}},
	}
	for _, tt := range tests {
		got, err := d.Filter(tt.pred)
		if err != nil {
			t.Fatalf("Filter(is_in %s) error = %v", tt.name, err)
		}
		if ids := frameColumnValues(t, got, "id"); !slices.Equal(ids, tt.want) {
			t.Errorf("Filter(is_in %s) id = %v, want %v", tt.name, ids, tt.want)
		}
	}
}
