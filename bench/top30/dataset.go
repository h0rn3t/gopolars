package top30

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

type dataset struct {
	G []string
	V []float64
	N []*float64
	I []int64
}

func GenerateDataset(n int, seed int64) dataset {
	r := rand.New(rand.NewSource(seed))
	g := make([]string, n)
	v := make([]float64, n)
	nvals := make([]*float64, n)
	i := make([]int64, n)

	groups := []string{"a", "b", "c", "d", "e"}

	for idx := 0; idx < n; idx++ {
		g[idx] = groups[r.Intn(len(groups))]
		v[idx] = r.Float64()*100 - 50
		if r.Float32() < 0.1 {
			nvals[idx] = nil
		} else {
			fv := r.Float64()*100 - 50
			nvals[idx] = &fv
		}
		i[idx] = r.Int63n(1000)
	}

	return dataset{G: g, V: v, N: nvals, I: i}
}

func WriteArrowIPC(path string, ds dataset) error {
	alloc := memory.NewGoAllocator()

	gBuilder := array.NewStringBuilder(alloc)
	defer gBuilder.Release()
	gBuilder.AppendValues(ds.G, nil)
	gArr := gBuilder.NewStringArray()
	defer gArr.Release()

	vBuilder := array.NewFloat64Builder(alloc)
	defer vBuilder.Release()
	vBuilder.AppendValues(ds.V, nil)
	vArr := vBuilder.NewFloat64Array()
	defer vArr.Release()

	nValid := make([]bool, len(ds.N))
	nData := make([]float64, len(ds.N))
	for i, ptr := range ds.N {
		if ptr != nil {
			nValid[i] = true
			nData[i] = *ptr
		}
	}
	nBuilder := array.NewFloat64Builder(alloc)
	defer nBuilder.Release()
	nBuilder.AppendValues(nData, nValid)
	nArr := nBuilder.NewFloat64Array()
	defer nArr.Release()

	iBuilder := array.NewInt64Builder(alloc)
	defer iBuilder.Release()
	iBuilder.AppendValues(ds.I, nil)
	iArr := iBuilder.NewInt64Array()
	defer iArr.Release()

	fields := []arrow.Field{
		{Name: "g", Type: arrow.BinaryTypes.String},
		{Name: "v", Type: arrow.PrimitiveTypes.Float64},
		{Name: "n", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
		{Name: "i", Type: arrow.PrimitiveTypes.Int64},
	}
	schema := arrow.NewSchema(fields, nil)

	arrays := []arrow.Array{gArr, vArr, nArr, iArr}
	rec := array.NewRecordBatch(schema, arrays, int64(len(ds.G)))
	defer rec.Release()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create arrow ipc file: %w", err)
	}
	defer func() { _ = f.Close() }()

	writer, err := ipc.NewFileWriter(f, ipc.WithSchema(schema))
	if err != nil {
		return fmt.Errorf("create arrow writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	if err := writer.Write(rec); err != nil {
		return fmt.Errorf("write arrow record: %w", err)
	}
	return nil
}
