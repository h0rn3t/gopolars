package database

import (
	"context"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
	"github.com/h0rn3t/gopolars/pkg/series"
)

// ReadInput configures Read. Conn, when non-nil, must be an adbc.Connection;
// otherwise DriverName + DriverOptions are used to open one.
type ReadInput struct {
	Query         string
	Conn          any
	DriverName    string
	DriverOptions map[string]string
	// BatchSize is advisory and currently not applied on read: the ADBC driver
	// controls result-set batching. Reserved for a future iter-batches form.
	BatchSize int
}

// Read executes a SQL query over an ADBC connection and builds a DataFrame from
// the returned Arrow record stream. An empty result yields a typed zero-row
// frame whose schema reflects the query's columns.
func Read(ctx context.Context, in ReadInput) (frame.DataFrame, error) {
	if in.Query == "" {
		return frame.DataFrame{}, fmt.Errorf("read_database: empty query")
	}

	rc, err := resolveConnection(ctx, in.Conn, in.DriverName, in.DriverOptions)
	if err != nil {
		return frame.DataFrame{}, err
	}
	defer rc.Close()

	stmt, err := rc.conn.NewStatement()
	if err != nil {
		return frame.DataFrame{}, fmt.Errorf("new statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	if err := stmt.SetSqlQuery(in.Query); err != nil {
		return frame.DataFrame{}, fmt.Errorf("set query: %w", err)
	}
	rr, _, err := stmt.ExecuteQuery(ctx)
	if err != nil {
		return frame.DataFrame{}, fmt.Errorf("execute query: %w", err)
	}
	defer rr.Release()

	var out frame.DataFrame
	have := false
	for rr.Next() {
		batch, err := iarrow.FromArrowRecord(rr.RecordBatch())
		if err != nil {
			return frame.DataFrame{}, fmt.Errorf("convert record batch: %w", err)
		}
		if !have {
			out, have = batch, true
			continue
		}
		out, err = out.Extend(batch)
		if err != nil {
			return frame.DataFrame{}, fmt.Errorf("concat record batches: %w", err)
		}
	}
	if err := rr.Err(); err != nil {
		return frame.DataFrame{}, fmt.Errorf("read record stream: %w", err)
	}
	if !have {
		return emptyFrameFromSchema(rr.Schema())
	}
	return out, nil
}

// emptyFrameFromSchema builds a zero-row DataFrame whose columns match the Arrow
// schema, so an empty query result still carries its column names and dtypes.
func emptyFrameFromSchema(schema *arrow.Schema) (frame.DataFrame, error) {
	cols := make([]series.Series, 0, len(schema.Fields()))
	for _, f := range schema.Fields() {
		s, err := series.New(f.Name, gopolarsDtype(f.Type), nil)
		if err != nil {
			return frame.DataFrame{}, fmt.Errorf("build empty column %q: %w", f.Name, err)
		}
		cols = append(cols, s)
	}
	return frame.New(frame.NewInput{Series: cols})
}

// gopolarsDtype mirrors the dtype pkg/io/arrow.FromArrowRecord produces for a
// given Arrow type, so empty-result frames stay consistent with populated ones
// (Int64/Float64/Boolean/Datetime native; everything else boxes to String).
func gopolarsDtype(t arrow.DataType) dtypes.DataType {
	switch t.ID() {
	case arrow.INT64:
		return dtypes.Int64
	case arrow.FLOAT64:
		return dtypes.Float64
	case arrow.BOOL:
		return dtypes.Boolean
	case arrow.TIMESTAMP:
		return dtypes.Datetime
	default:
		return dtypes.String
	}
}
