package arrow_test

import (
	"slices"
	"testing"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"

	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// stringArray builds a String (or LargeString) array where a nil entry is null.
func stringArray(large bool, vals []*string) goarrow.Array {
	mem := memory.NewGoAllocator()
	var b interface {
		array.Builder
		Append(string)
	}
	if large {
		b = array.NewLargeStringBuilder(mem)
	} else {
		b = array.NewStringBuilder(mem)
	}
	defer b.Release()
	for _, v := range vals {
		if v == nil {
			b.AppendNull()
			continue
		}
		b.Append(*v)
	}
	return b.NewArray()
}

// importColumn imports arr as column "s", releases the record and the array,
// and returns the column's boxed values (nil for null).
func importColumn(t *testing.T, arr goarrow.Array) []any {
	t.Helper()
	schema := goarrow.NewSchema([]goarrow.Field{{Name: "s", Type: arr.DataType(), Nullable: true}}, nil)
	rec := array.NewRecordBatch(schema, []goarrow.Array{arr}, int64(arr.Len()))
	df, err := iarrow.FromArrowRecord(rec)
	rec.Release()
	arr.Release()
	if err != nil {
		t.Fatalf("FromArrowRecord: %v", err)
	}
	s, ok := df.Series("s")
	if !ok {
		t.Fatalf("FromArrowRecord: column %q missing", "s")
	}
	got := make([]any, s.Len())
	for i := range got {
		got[i] = s.Value(i)
	}
	return got
}

func TestFromArrowStringColumn(t *testing.T) {
	t.Parallel()
	values := []*string{new("alpha"), nil, new(""), new("β-юнікод"), nil, new("gamma")}
	want := []any{"alpha", nil, "", "β-юнікод", nil, "gamma"}

	tests := []struct {
		name  string
		large bool
	}{
		{"string", false},
		{"large_string", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := importColumn(t, stringArray(tt.large, values)); !slices.Equal(got, want) {
				t.Errorf("FromArrowRecord(%v) values = %v, want %v", tt.name, got, want)
			}
		})
	}
}

func TestFromArrowStringSlice(t *testing.T) {
	t.Parallel()
	full := stringArray(false, []*string{new("a"), new("bb"), nil, new("ccc"), new("dddd")})
	defer full.Release()
	sliced := array.NewSlice(full, 1, 4) // "bb", null, "ccc"

	want := []any{"bb", nil, "ccc"}
	if got := importColumn(t, sliced); !slices.Equal(got, want) {
		t.Errorf("FromArrowRecord(slice[1:4]) values = %v, want %v", got, want)
	}
}

func TestFromArrowStringWithoutValueBytes(t *testing.T) {
	t.Parallel()
	want := []any{"", nil, ""}
	if got := importColumn(t, stringArray(false, []*string{new(""), nil, new("")})); !slices.Equal(got, want) {
		t.Errorf("FromArrowRecord(empty strings) values = %v, want %v", got, want)
	}
}

func TestFromArrowStringAllocsIndependentOfLength(t *testing.T) {
	allocsFor := func(n int) float64 {
		vals := make([]*string, n)
		for i := range vals {
			vals[i] = new("value")
		}
		arr := stringArray(false, vals)
		defer arr.Release()
		schema := goarrow.NewSchema([]goarrow.Field{{Name: "s", Type: arr.DataType(), Nullable: true}}, nil)
		rec := array.NewRecordBatch(schema, []goarrow.Array{arr}, int64(n))
		defer rec.Release()
		return testing.AllocsPerRun(5, func() {
			if _, err := iarrow.FromArrowRecord(rec); err != nil {
				t.Fatalf("FromArrowRecord: %v", err)
			}
		})
	}
	small, large := allocsFor(100), allocsFor(10_000)
	if large != small {
		t.Errorf("FromArrowRecord allocs: %v for 100 rows, %v for 10000 rows, want equal", small, large)
	}
}
