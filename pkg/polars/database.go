package polars

import (
	"context"

	idatabase "github.com/h0rn3t/gopolars/pkg/io/database"
)

// ReadDatabase executes a SQL query against an external database and returns the
// result as a DataFrame, mirroring Python polars read_database. The connection
// is taken from ReadDatabaseInput.Conn (an open adbc.Connection) or opened from
// DriverName + DriverOptions via the ADBC driver manager (CGO).
func ReadDatabase(ctx context.Context, input ReadDatabaseInput) (DataFrame, error) {
	return fromFrame(idatabase.Read(ctx, idatabase.ReadInput{
		Query:         input.Query,
		Conn:          input.Conn,
		DriverName:    input.DriverName,
		DriverOptions: input.DriverOptions,
		BatchSize:     input.BatchSize,
	}))
}
