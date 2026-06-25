//go:build duckdb && duckdb_arrow

package polars

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	duckdb "github.com/marcboeker/go-duckdb/v2"
	"golang.org/x/text/unicode/norm"
)

// normalizeUDF implements a DuckDB scalar function `normalize(text, form)` that
// applies Unicode normalization (NFC/NFD/NFKC/NFKD). DuckDB only ships
// nfc_normalize (NFC); polars' SQL exposes NORMALIZE(text, FORM) for all four
// forms, so this UDF fills the gap using Go's golang.org/x/text/unicode/norm.
type normalizeUDF struct{}

func (*normalizeUDF) Config() duckdb.ScalarFuncConfig {
	vc, err := duckdb.NewTypeInfo(duckdb.TYPE_VARCHAR)
	if err != nil {
		panic(err)
	}
	return duckdb.ScalarFuncConfig{
		InputTypeInfos: []duckdb.TypeInfo{vc, vc},
		ResultTypeInfo: vc,
	}
}

func (*normalizeUDF) Executor() duckdb.ScalarFuncExecutor {
	return duckdb.ScalarFuncExecutor{RowExecutor: func(values []driver.Value) (any, error) {
		text, _ := values[0].(string)
		form, _ := values[1].(string)
		var f norm.Form
		switch strings.ToUpper(strings.TrimSpace(form)) {
		case "NFC":
			f = norm.NFC
		case "NFD":
			f = norm.NFD
		case "NFKC":
			f = norm.NFKC
		case "NFKD":
			f = norm.NFKD
		default:
			return nil, fmt.Errorf("normalize: unknown form %q", form)
		}
		return f.String(text), nil
	}}
}

// registerUDFs registers gopolars' supplementary scalar functions on the
// connection (currently the Unicode `normalize`).
func registerUDFs(conn *sql.Conn) error {
	if err := duckdb.RegisterScalarUDF(conn, "normalize", &normalizeUDF{}); err != nil {
		return fmt.Errorf("register normalize udf: %w", err)
	}
	return nil
}
