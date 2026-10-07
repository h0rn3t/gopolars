package frame

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func oneColumn(t *testing.T, col *chunk.Column) DataFrame {
	t.Helper()
	df, err := New(NewInput{Series: []series.Series{series.FromColumn("x", col)}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func TestConcatVerticalRejectsMismatchedDtypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		left, right *chunk.Column
		wantInErr   []string
	}{
		{"int64 and float64", chunk.NewInt64([]int64{1, 2}, nil), chunk.NewFloat64([]float64{3.5, 4.5}, nil), []string{`"x"`, "int64", "float64"}},
		{"string and datetime", chunk.NewString([]string{"x"}, nil), chunk.NewTime([]time.Time{noonUTC}, nil), []string{`"x"`}},
	}
	for _, tt := range tests {
		_, err := ConcatVertical(oneColumn(t, tt.left), oneColumn(t, tt.right))
		if err == nil {
			t.Errorf("ConcatVertical(%s) error = nil, want a dtype mismatch", tt.name)
			continue
		}
		for _, want := range tt.wantInErr {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("ConcatVertical(%s) error = %q, want it to mention %s", tt.name, err, want)
			}
		}
	}
}

func TestConcatVerticalAdoptsAllNullColumns(t *testing.T) {
	t.Parallel()
	allNullString := func(n int) *chunk.Column {
		return chunk.NewString(make([]string, n), slices.Repeat([]bool{true}, n))
	}
	tests := []struct {
		name  string
		parts []*chunk.Column
		want  []any
	}{
		{"all-null first", []*chunk.Column{allNullString(2), chunk.NewInt64([]int64{5}, nil)}, []any{nil, nil, int64(5)}},
		{"all-null last", []*chunk.Column{chunk.NewInt64([]int64{5}, nil), allNullString(1)}, []any{int64(5), nil}},
		{"zero rows of another dtype", []*chunk.Column{chunk.NewInt64([]int64{5}, nil), chunk.NewString([]string{}, nil)}, []any{int64(5)}},
		{"matching dtypes", []*chunk.Column{chunk.NewInt64([]int64{1, 0}, []bool{false, true}), chunk.NewInt64([]int64{3}, nil)}, []any{int64(1), nil, int64(3)}},
	}
	for _, tt := range tests {
		frames := make([]DataFrame, len(tt.parts))
		for i, c := range tt.parts {
			frames[i] = oneColumn(t, c)
		}
		got, err := ConcatVertical(frames[0], frames[1:]...)
		if err != nil {
			t.Errorf("ConcatVertical(%s) error = %v, want nil", tt.name, err)
			continue
		}
		s, _ := got.Series("x")
		if s.DataType() != dtypes.Int64 {
			t.Errorf("ConcatVertical(%s) dtype = %s, want int64", tt.name, s.DataType())
		}
		vals := make([]any, s.Len())
		for i := range vals {
			vals[i] = s.Value(i)
		}
		if !slices.Equal(vals, tt.want) {
			t.Errorf("ConcatVertical(%s) = %v, want %v", tt.name, vals, tt.want)
		}
	}
}
