package frame

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/series"
)

func TestInsertColumnValidatesIndexAndName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		index   int
		col     string
		want    []string
		wantErr string
	}{
		{"at end", 2, "c", []string{"a", "b", "c"}, ""},
		{"in middle", 1, "c", []string{"a", "c", "b"}, ""},
		{"at start", 0, "c", []string{"c", "a", "b"}, ""},
		{"index above width", 3, "c", nil, "out of bounds"},
		{"negative index", -1, "c", nil, "out of bounds"},
		{"duplicate name at end", 2, "a", nil, "duplicate"},
		{"duplicate name in place", 1, "b", nil, "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			df := mustFrame(t,
				SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
				SeriesInput{Name: "b", Values: []any{"x", "y"}},
			)
			col := series.FromFloat64(tt.col, []float64{7, 8}, nil)
			out, err := df.InsertColumn(tt.index, col)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("InsertColumn(%d, %s) error = %v, want error containing %q", tt.index, tt.col, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("InsertColumn(%d, %s) error = %v, want nil", tt.index, tt.col, err)
				}
				if got := out.Columns(); !slices.Equal(got, tt.want) {
					t.Errorf("InsertColumn(%d, %s) columns = %v, want %v", tt.index, tt.col, got, tt.want)
				}
				if got := columnValues(t, out, tt.col); !reflect.DeepEqual(got, []any{7.0, 8.0}) {
					t.Errorf("InsertColumn(%d, %s) values = %v, want [7 8]", tt.index, tt.col, got)
				}
			}
			if got := df.Columns(); !slices.Equal(got, []string{"a", "b"}) {
				t.Errorf("InsertColumn(%d, %s) changed the receiver's columns to %v", tt.index, tt.col, got)
			}
			if got := columnValues(t, df, "b"); !reflect.DeepEqual(got, []any{"x", "y"}) {
				t.Errorf("InsertColumn(%d, %s) changed the receiver's column b to %v", tt.index, tt.col, got)
			}
		})
	}
}
