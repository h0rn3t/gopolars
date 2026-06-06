package polars

import (
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestDataFrameWriteStubsReturnError pins the contract of the not-yet-supported
// writer sinks: each reports an error rather than silently succeeding, so
// callers can detect the unsupported format.
func TestDataFrameWriteStubsReturnError(t *testing.T) {
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
	}})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}

	cases := []struct {
		name string
		call func() error
	}{
		{"WriteAvro", func() error { return df.WriteAvro("/tmp/x.avro") }},
		{"WriteClipboard", func() error { return df.WriteClipboard() }},
		{"WriteDatabase", func() error { return df.WriteDatabase("postgres://x") }},
		{"WriteDelta", func() error { return df.WriteDelta("/tmp/delta") }},
		{"WriteExcel", func() error { return df.WriteExcel("/tmp/x.xlsx") }},
		{"WriteIceberg", func() error { return df.WriteIceberg("/tmp/iceberg") }},
	}
	for _, tc := range cases {
		if err := tc.call(); err == nil {
			t.Errorf("%s: expected an unsupported error, got nil", tc.name)
		}
	}
}
