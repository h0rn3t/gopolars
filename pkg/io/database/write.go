package database

import (
	"context"
	"fmt"

	"github.com/apache/arrow-adbc/go/adbc"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// IfTableExists selects what write_database does when the target table exists.
type IfTableExists string

const (
	IfTableExistsFail    IfTableExists = "fail"
	IfTableExistsAppend  IfTableExists = "append"
	IfTableExistsReplace IfTableExists = "replace"
)

// WriteInput configures Write. Conn, when non-nil, must be an adbc.Connection;
// otherwise DriverName + DriverOptions are used to open one.
type WriteInput struct {
	TableName     string
	IfTableExists IfTableExists
	Conn          any
	DriverName    string
	DriverOptions map[string]string
	BatchSize     int
}

// Write streams df to an external SQL table via ADBC ingest and returns the
// number of rows affected (-1 when the driver does not report it).
func Write(ctx context.Context, df frame.DataFrame, in WriteInput) (int64, error) {
	catalog, schema, table, err := parseTableName(in.TableName)
	if err != nil {
		return 0, err
	}
	mode, err := ingestMode(in.IfTableExists)
	if err != nil {
		return 0, err
	}
	if err := validateWriteSchema(df); err != nil {
		return 0, err
	}

	rc, err := resolveConnection(ctx, in.Conn, in.DriverName, in.DriverOptions)
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	// newDFRecordReader yields the frame as batchSize-row Arrow batches.
	// IngestStream binds the stream (transferring ownership to the driver, which
	// releases it when it closes the statement); we must not Release it here.
	rr, err := newDFRecordReader(df, in.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("build record stream: %w", err)
	}

	n, err := adbc.IngestStream(ctx, rc.conn, rr, table, mode, adbc.IngestStreamOptions{
		Catalog:  catalog,
		DBSchema: schema,
	})
	if err != nil {
		return 0, fmt.Errorf("write_database ingest: %w", err)
	}
	return n, nil
}

// validateWriteSchema rejects dtypes the Arrow converter cannot ingest without
// silent corruption. Nested types (List, Struct) are stringified by the boxed
// fallback in pkg/io/arrow, so they error here instead. Decimal and
// Categorical/Enum are allowed: they ingest as text (a documented type-fidelity
// gap, but value-preserving).
func validateWriteSchema(df frame.DataFrame) error {
	for _, f := range df.Schema() {
		switch f.Type {
		case dtypes.List, dtypes.Struct:
			return fmt.Errorf(
				"write_database: column %q has unsupported dtype %q for the ADBC engine", f.Name, f.Type)
		}
	}
	return nil
}

// ingestMode maps if_table_exists to the ADBC ingest mode option value.
func ingestMode(m IfTableExists) (string, error) {
	switch m {
	case "", IfTableExistsFail:
		return adbc.OptionValueIngestModeCreate, nil
	case IfTableExistsAppend:
		return adbc.OptionValueIngestModeCreateAppend, nil
	case IfTableExistsReplace:
		return adbc.OptionValueIngestModeReplace, nil
	default:
		return "", fmt.Errorf("unknown if_table_exists mode %q (want fail, append or replace)", m)
	}
}
