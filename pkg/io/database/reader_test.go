package database

import (
	"context"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

func mustFrame(t *testing.T, cols ...frame.SeriesInput) frame.DataFrame {
	t.Helper()
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("build frame: %v", err)
	}
	return df
}

// sampleFrame is a small frame covering the natively-supported dtypes
// (Int64/Float64/String/Boolean/Datetime), shared by the engine tests.
func sampleFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	return mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		frame.SeriesInput{Name: "score", Values: []any{1.5, 2.5, 3.5}},
		frame.SeriesInput{Name: "name", Values: []any{"a", "b", "c"}},
		frame.SeriesInput{Name: "ok", Values: []any{true, false, true}},
		frame.SeriesInput{Name: "ts", Values: []any{
			time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC(), time.Unix(3, 0).UTC()}},
	)
}

// TestDFRecordReaderBatching drives the write-path streaming reader without a
// database: it chunks a frame into batchSize records and, reassembled via
// pkg/io/arrow, reproduces the original rows, schema and values.
func TestDFRecordReaderBatching(t *testing.T) {
	df := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
		frame.SeriesInput{Name: "score", Values: []any{1.5, 2.5, nil, 4.5, 5.5}},
		frame.SeriesInput{Name: "name", Values: []any{"a", "b", "c", "d", "e"}},
		frame.SeriesInput{Name: "ok", Values: []any{true, false, true, nil, false}},
	)

	rr, err := newDFRecordReader(df, 2)
	if err != nil {
		t.Fatalf("newDFRecordReader: %v", err)
	}
	defer rr.Release()

	// Schema must be present before the first batch.
	if got := len(rr.Schema().Fields()); got != 4 {
		t.Fatalf("schema fields = %d, want 4", got)
	}

	var batches int
	var combined frame.DataFrame
	have := false
	for rr.Next() {
		batches++
		b, err := iarrow.FromArrowRecord(rr.RecordBatch())
		if err != nil {
			t.Fatalf("FromArrowRecord: %v", err)
		}
		if !have {
			combined, have = b, true
			continue
		}
		if combined, err = combined.Extend(b); err != nil {
			t.Fatalf("extend: %v", err)
		}
	}
	if err := rr.Err(); err != nil {
		t.Fatalf("reader err: %v", err)
	}
	if batches != 3 { // ceil(5/2)
		t.Fatalf("batches = %d, want 3", batches)
	}
	if combined.Height() != 5 {
		t.Fatalf("reassembled height = %d, want 5", combined.Height())
	}

	idCol, _ := combined.GetColumn("id")
	for i := 0; i < 5; i++ {
		if got := idCol.Value(i); got != int64(i+1) {
			t.Fatalf("id[%d] = %v, want %d", i, got, i+1)
		}
	}
	scoreCol, _ := combined.GetColumn("score")
	if scoreCol.Value(2) != nil {
		t.Fatalf("score[2] = %v, want nil", scoreCol.Value(2))
	}
	okCol, _ := combined.GetColumn("ok")
	if okCol.Value(0) != true || okCol.Value(3) != nil {
		t.Fatalf("ok column round-trip mismatch: [0]=%v [3]=%v", okCol.Value(0), okCol.Value(3))
	}
}

func TestDFRecordReaderEmpty(t *testing.T) {
	df := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
	).Slice(0, 0)

	rr, err := newDFRecordReader(df, 100)
	if err != nil {
		t.Fatalf("newDFRecordReader: %v", err)
	}
	defer rr.Release()

	if rr.Next() {
		t.Fatalf("empty frame should yield no batches")
	}
	if err := rr.Err(); err != nil {
		t.Fatalf("reader err: %v", err)
	}
	if got := len(rr.Schema().Fields()); got != 1 {
		t.Fatalf("schema fields = %d, want 1", got)
	}
}

func TestValidateWriteSchemaRejectsNested(t *testing.T) {
	listFrame := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
		frame.SeriesInput{Name: "tags", Values: []any{[]any{int64(1), int64(2)}}},
	)
	if err := validateWriteSchema(listFrame); err == nil {
		t.Fatalf("expected error for List column")
	}

	okFrame := mustFrame(t,
		frame.SeriesInput{Name: "id", Values: []any{int64(1)}},
		frame.SeriesInput{Name: "name", Values: []any{"a"}},
	)
	if err := validateWriteSchema(okFrame); err != nil {
		t.Fatalf("unexpected error for scalar columns: %v", err)
	}
}

func TestWriteAndReadMissingConnection(t *testing.T) {
	df := mustFrame(t, frame.SeriesInput{Name: "id", Values: []any{int64(1)}})

	if _, err := Write(context.Background(), df, WriteInput{TableName: "t"}); err == nil {
		t.Fatalf("Write with no connection should error")
	}
	if _, err := Read(context.Background(), ReadInput{Query: "SELECT 1"}); err == nil {
		t.Fatalf("Read with no connection should error")
	}
	if _, err := Read(context.Background(), ReadInput{Query: ""}); err == nil {
		t.Fatalf("Read with empty query should error")
	}
	// DriverName set but pointing at no real driver: must error rather than panic
	// (a load failure with CGO, or a clear CGO-required message without it).
	if _, err := Read(context.Background(), ReadInput{Query: "SELECT 1", DriverName: "nonexistent-driver-xyz"}); err == nil {
		t.Fatalf("Read open-by-unknown-driver should error")
	}
}
