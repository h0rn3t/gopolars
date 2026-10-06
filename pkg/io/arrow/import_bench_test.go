package arrow_test

import (
	"testing"

	goarrow "github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"

	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

func BenchmarkFromArrowRecordStringFloat(b *testing.B) {
	const n = 200_000
	mem := memory.NewGoAllocator()
	sb := array.NewStringBuilder(mem)
	fb := array.NewFloat64Builder(mem)
	labels := []string{"alpha", "beta", "gamma"}
	for i := range n {
		sb.Append(labels[i%len(labels)])
		fb.Append(float64(i))
	}
	strs, floats := sb.NewArray(), fb.NewArray()
	defer strs.Release()
	defer floats.Release()
	schema := goarrow.NewSchema([]goarrow.Field{
		{Name: "s", Type: goarrow.BinaryTypes.String, Nullable: true},
		{Name: "v", Type: goarrow.PrimitiveTypes.Float64, Nullable: true},
	}, nil)
	rec := array.NewRecordBatch(schema, []goarrow.Array{strs, floats}, n)
	defer rec.Release()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := iarrow.FromArrowRecord(rec); err != nil {
			b.Fatalf("FromArrowRecord: %v", err)
		}
	}
}
