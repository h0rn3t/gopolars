package polars

import (
	"context"
	"strings"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestWriteDatabaseWiring checks df.WriteDatabase reaches the ADBC engine: with
// no connection supplied it returns the engine's clear "no connection" error
// rather than the old "not supported" stub.
func TestWriteDatabaseWiring(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("new frame: %v", err)
	}
	_, err = df.WriteDatabase(WriteDatabaseInput{TableName: "t"})
	if err == nil {
		t.Fatalf("WriteDatabase with no connection should error")
	}
	if !strings.Contains(err.Error(), "no connection") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadDatabaseWiring(t *testing.T) {
	_, err := ReadDatabase(context.Background(), ReadDatabaseInput{Query: "SELECT 1"})
	if err == nil {
		t.Fatalf("ReadDatabase with no connection should error")
	}
	if !strings.Contains(err.Error(), "no connection") {
		t.Fatalf("unexpected error: %v", err)
	}
}
