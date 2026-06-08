// Package database implements gopolars write_database / read_database against
// external SQL databases via an ADBC (Arrow Database Connectivity) Arrow-native
// engine.
//
// Writes stream a DataFrame to the driver as Arrow record batches
// (adbc_ingest: BindStream + ExecuteUpdate); the driver creates the table from
// the Arrow schema and bulk-loads the rows. Reads execute a SQL query and build
// a DataFrame from the Arrow RecordReader the driver returns. The gopolars
// <-> Arrow conversion reuses pkg/io/arrow (both the core and this engine are
// on github.com/apache/arrow-go/v18).
//
// Connections are resolved either from a caller-provided adbc.Connection or by
// opening one from a driver name + options through the ADBC driver manager
// (CGO; see conn_cgo.go / conn_nocgo.go).
//
// The package depends on ADBC + arrow-go/v18, which require CGO for the
// SQLite/PostgreSQL drivers (FlightSQL/Snowflake are pure Go).
package database
