//go:build !duckdb || !duckdb_arrow

package polars

import (
	"context"
	"errors"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// errNoDuckDB is returned by the SQL surface when the binary was not built with
// the DuckDB engine. Build with: go build -tags "duckdb duckdb_arrow".
var errNoDuckDB = errors.New(`polars: SQL requires building with -tags "duckdb duckdb_arrow"`)

func execSQL(_ context.Context, _ string, _ map[string]frame.DataFrame) (frame.DataFrame, error) {
	return frame.DataFrame{}, errNoDuckDB
}
