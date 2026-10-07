//go:build duckdb && duckdb_arrow

package polars

import (
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestDuckDBResultTypes pins how DuckDB result types map to gopolars dtypes:
// narrow and unsigned integers widen to Int64, REAL to Float64, and a UBIGINT
// above math.MaxInt64 is an error rather than a wrapped negative number.
func TestDuckDBResultTypes(t *testing.T) {
	tests := []struct {
		query   string
		dtype   dtypes.DataType
		want    any
		wantErr string
	}{
		{query: "SELECT 42::TINYINT AS x", dtype: dtypes.Int64, want: int64(42)},
		{query: "SELECT 42::UINTEGER AS x", dtype: dtypes.Int64, want: int64(42)},
		{query: "SELECT 42::UBIGINT AS x", dtype: dtypes.Int64, want: int64(42)},
		{query: "SELECT 1.5::REAL AS x", dtype: dtypes.Float64, want: 1.5},
		{query: "SELECT 18446744073709551615::UBIGINT AS x", wantErr: "overflows int64"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			// The query runs inside SQL, so a conversion error can surface there.
			var out DataFrame
			lf, err := NewIO().SQL(t.Context(), tt.query)
			if err == nil {
				out, err = lf.Collect(t.Context())
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("SQL(%q) error = %v, want error containing %q", tt.query, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SQL(%q) error = %v", tt.query, err)
			}

			col, err := out.GetColumn("x")
			if err != nil {
				t.Fatalf("GetColumn(x) error = %v", err)
			}
			if col.DataType() != tt.dtype || col.Value(0) != tt.want {
				t.Errorf("SQL(%q) = %s %v, want %s %v", tt.query, col.DataType(), col.Value(0), tt.dtype, tt.want)
			}
		})
	}
}
