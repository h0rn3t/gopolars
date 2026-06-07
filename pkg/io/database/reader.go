package database

import (
	"sync/atomic"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

// defaultBatchSize is the streaming chunk size used when BatchSize <= 0.
const defaultBatchSize = 10000

var _ array.RecordReader = (*dfRecordReader)(nil)

// dfRecordReader implements array.RecordReader over a frame.DataFrame, yielding
// batchSize-row Arrow record batches built on demand via pkg/io/arrow so peak
// Arrow-side memory is bounded to a single batch rather than the whole frame.
type dfRecordReader struct {
	df        frame.DataFrame
	schema    *arrow.Schema
	height    int
	batchSize int

	pos  int
	cur  arrow.RecordBatch
	err  error
	refs int64
}

// newDFRecordReader derives a stable Arrow schema from a zero-row slice (the
// schema is independent of row count) and returns a reader positioned before
// the first batch.
func newDFRecordReader(df frame.DataFrame, batchSize int) (*dfRecordReader, error) {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	head, err := iarrow.ToArrowRecord(df.Slice(0, 0))
	if err != nil {
		return nil, err
	}
	schema := head.Schema()
	head.Release()
	return &dfRecordReader{df: df, schema: schema, height: df.Height(), batchSize: batchSize, refs: 1}, nil
}

func (r *dfRecordReader) Schema() *arrow.Schema { return r.schema }

func (r *dfRecordReader) Retain() { atomic.AddInt64(&r.refs, 1) }

func (r *dfRecordReader) Release() {
	if atomic.AddInt64(&r.refs, -1) == 0 && r.cur != nil {
		r.cur.Release()
		r.cur = nil
	}
}

func (r *dfRecordReader) Err() error { return r.err }

func (r *dfRecordReader) RecordBatch() arrow.RecordBatch { return r.cur }

// Record is the deprecated alias array.RecordReader still requires.
func (r *dfRecordReader) Record() arrow.RecordBatch { return r.cur }

func (r *dfRecordReader) Next() bool {
	if r.err != nil || r.pos >= r.height {
		return false
	}
	if r.cur != nil {
		r.cur.Release()
		r.cur = nil
	}
	end := r.pos + r.batchSize
	if end > r.height {
		end = r.height
	}
	rec, err := iarrow.ToArrowRecord(r.df.Slice(r.pos, end-r.pos))
	if err != nil {
		r.err = err
		return false
	}
	r.cur = rec
	r.pos = end
	return true
}
