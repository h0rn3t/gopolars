package frame

import (
	"reflect"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// nestedFrame builds a frame from explicitly typed columns, so list, struct
// and binary values keep their dtype.
func nestedFrame(t *testing.T, cols ...series.Series) DataFrame {
	t.Helper()
	df, err := New(NewInput{Series: cols})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return df
}

func nestedSeries(t *testing.T, name string, dt dtypes.DataType, values []any) series.Series {
	t.Helper()
	s, err := series.New(name, dt, values)
	if err != nil {
		t.Fatalf("series.New(%s, %s): %v", name, dt, err)
	}
	return s
}

func TestSortNestedColumns(t *testing.T) {
	t.Parallel()
	l := func(v ...any) []any { return v }
	type st = map[string]any
	tests := []struct {
		name  string
		dt    dtypes.DataType
		in    []any
		input SortInput
		want  []any
	}{
		{"list ascending with prefix", dtypes.List,
			[]any{l(int64(2)), l(int64(1), int64(2)), l(int64(1))},
			SortInput{}, []any{l(int64(1)), l(int64(1), int64(2)), l(int64(2))}},
		{"list descending", dtypes.List,
			[]any{l(int64(1)), l(int64(2)), l(int64(1), int64(2))},
			SortInput{Descending: []bool{true}}, []any{l(int64(2)), l(int64(1), int64(2)), l(int64(1))}},
		{"list of strings", dtypes.List,
			[]any{l("b"), l("a", "z"), l("a")},
			SortInput{}, []any{l("a"), l("a", "z"), l("b")}},
		{"list nulls last", dtypes.List,
			[]any{l(int64(2)), nil, l(int64(1))},
			SortInput{NullsLast: true}, []any{l(int64(1)), l(int64(2)), nil}},
		{"list nulls first", dtypes.List,
			[]any{l(int64(2)), nil, l(int64(1))},
			SortInput{}, []any{nil, l(int64(1)), l(int64(2))}},
		{"struct ascending", dtypes.Struct,
			[]any{st{"a": int64(2)}, st{"a": int64(1)}},
			SortInput{}, []any{st{"a": int64(1)}, st{"a": int64(2)}}},
		{"struct fields in name order", dtypes.Struct,
			[]any{st{"b": int64(1), "a": int64(2)}, st{"b": int64(2), "a": int64(1)}, st{"b": int64(0), "a": int64(1)}},
			SortInput{}, []any{st{"a": int64(1), "b": int64(0)}, st{"a": int64(1), "b": int64(2)}, st{"a": int64(2), "b": int64(1)}}},
		{"struct nulls last", dtypes.Struct,
			[]any{nil, st{"a": int64(1)}},
			SortInput{NullsLast: true}, []any{st{"a": int64(1)}, nil}},
		{"binary", dtypes.Binary,
			[]any{[]byte("b"), []byte("ab"), []byte("a")},
			SortInput{}, []any{[]byte("a"), []byte("ab"), []byte("b")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df := nestedFrame(t, nestedSeries(t, "k", tt.dt, tt.in))
			tt.input.By = []string{"k"}
			out, err := df.Sort(tt.input)
			if err != nil {
				t.Fatalf("Sort(%v) error = %v", tt.in, err)
			}
			if got := columnValues(t, out, "k"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sort(%v, %+v) = %v, want %v", tt.in, tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterEqualityOnNestedColumns(t *testing.T) {
	t.Parallel()
	l := func(v ...any) []any { return v }
	type st = map[string]any
	tests := []struct {
		name string
		dt   dtypes.DataType
		a, b []any
		pred expr.Expr
		want int
	}{
		{"list eq self", dtypes.List, []any{l(int64(1)), l(int64(2))}, nil,
			expr.Col("a").Eq(expr.Col("a")), 2},
		{"list eq", dtypes.List, []any{l(int64(1)), l(int64(2), int64(3))}, []any{l(int64(1)), l(int64(2))},
			expr.Col("a").Eq(expr.Col("b")), 1},
		{"list ne", dtypes.List, []any{l(int64(1)), l(int64(2), int64(3))}, []any{l(int64(1)), l(int64(2))},
			expr.Col("a").Ne(expr.Col("b")), 1},
		{"list eq_missing", dtypes.List, []any{l(int64(1)), nil}, []any{l(int64(1)), nil},
			expr.Col("a").EqMissing(expr.Col("b")), 2},
		{"struct eq", dtypes.Struct, []any{st{"x": int64(1)}, st{"x": int64(2)}}, []any{st{"x": int64(1)}, st{"x": int64(3)}},
			expr.Col("a").Eq(expr.Col("b")), 1},
		{"binary eq", dtypes.Binary, []any{[]byte("x"), []byte("y")}, []any{[]byte("x"), []byte("z")},
			expr.Col("a").Eq(expr.Col("b")), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := tt.b
			if b == nil {
				b = tt.a
			}
			df := nestedFrame(t, nestedSeries(t, "a", tt.dt, tt.a), nestedSeries(t, "b", tt.dt, b))
			out, err := df.Filter(tt.pred)
			if err != nil {
				t.Fatalf("Filter(%s) error = %v", tt.name, err)
			}
			if got := out.Height(); got != tt.want {
				t.Errorf("Filter(%s) on a=%v b=%v kept %d rows, want %d", tt.name, tt.a, b, got, tt.want)
			}
		})
	}
}
