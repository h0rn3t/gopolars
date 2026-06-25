// Package sql holds parity tests that compare gopolars' DuckDB-backed SQL engine
// against the py-polars SQL test suite (py-1.28.1). Every test file is gated on
// the `duckdb && duckdb_arrow` build tags, so the suite only runs when built with
// `-tags "duckdb duckdb_arrow"`. Without those tags this package is intentionally
// empty (this file keeps it buildable in the default, pure-Go configuration).
package sql
