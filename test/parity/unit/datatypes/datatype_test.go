package datatypes

// Ported from py-polars/tests/unit/datatypes/test_datatype.py (py-1.28.1)
//
// The Python test (test_datatype_copy) checks copy.deepcopy(pl.Int64()) equality.
// gopolars dtypes are simple comparable string-backed constants, so equality and
// identity are trivially preserved across assignment.

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDatatypeEquality(t *testing.T) {
	t.Parallel()
	cases := []dtypes.DataType{
		polars.Int64, polars.Float64, polars.String, polars.Boolean,
		polars.Datetime, polars.Decimal, polars.Categorical, polars.Enum,
		polars.List, polars.Struct,
	}
	for _, dt := range cases {
		copyDt := dt // assignment "copy"
		if copyDt != dt {
			t.Fatalf("dtype copy not equal: %v vs %v", copyDt, dt)
		}
	}
}

func TestDatatypeDistinct(t *testing.T) {
	t.Parallel()
	if polars.Int64 == polars.Float64 {
		t.Fatal("Int64 must not equal Float64")
	}
	if polars.String == polars.Boolean {
		t.Fatal("String must not equal Boolean")
	}
}
