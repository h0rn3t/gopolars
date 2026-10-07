package chunk

import (
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

func TestConcatColumnsValidity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cols      []*Column
		wantNulls []bool // nil: the output carries no validity
	}{
		{"null-free", []*Column{NewInt64([]int64{1, 2}, nil), NewInt64([]int64{3}, nil)}, nil},
		{"one input with nulls", []*Column{NewInt64([]int64{1, 2}, nil), NewInt64([]int64{0, 4}, []bool{true, false})}, []bool{false, false, true, false}},
		{"adopted all-null chunk", []*Column{NewInt64([]int64{1}, nil), NewString([]string{"", ""}, []bool{true, true})}, []bool{false, true, true}},
	}
	for _, tt := range tests {
		out := ConcatColumns(tt.cols, dtypes.Int64)
		if !slices.Equal(out.Nulls(), tt.wantNulls) {
			t.Errorf("ConcatColumns(%s) nulls = %v, want %v", tt.name, out.Nulls(), tt.wantNulls)
		}
		if tt.wantNulls == nil && out.NullCount() != 0 {
			t.Errorf("ConcatColumns(%s) NullCount() = %d, want 0", tt.name, out.NullCount())
		}
	}
}

func BenchmarkConcatColumns(b *testing.B) {
	parts := make([]*Column, 4)
	for i := range parts {
		parts[i] = NewInt64(make([]int64, 100_000), nil)
	}
	b.ReportAllocs()
	for b.Loop() {
		ConcatColumns(parts, dtypes.Int64)
	}
}
