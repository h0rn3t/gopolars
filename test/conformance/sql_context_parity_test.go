package conformance

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// sqlParityCase is one differential check: the same data and query are run in
// gopolars SQLContext and Python polars SQLContext and their results compared.
type sqlParityCase struct {
	name  string
	query string
}

// pythonSQLScript builds the shared dataset in Python polars, runs the query via
// SQLContext, and prints the result rows as JSON.
func pythonSQLScript(query string) string {
	return `
import json
import polars as pl

ctx = pl.SQLContext()
ctx.register("customers", pl.DataFrame({"id": [1, 2, 3], "name": ["a", "b", "c"]}))
ctx.register("orders", pl.DataFrame({"cust_id": [1, 1, 2], "amount": [10, 20, 30]}))
ctx.register("t", pl.DataFrame({"x": [-1, 0, 5], "v": [10.0, 20.0, 30.0], "city": ["A", "B", "C"]}))
df = ctx.execute(""" ` + query + ` """, eager=True)
print(json.dumps(df.to_dicts()))
`
}

func gopolarsParityRows(t *testing.T, query string) []map[string]any {
	t.Helper()
	ctx := polars.NewSQLContext()
	customers, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
		{Name: "name", Values: []any{"a", "b", "c"}},
	}})
	orders, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "cust_id", Values: []any{int64(1), int64(1), int64(2)}},
		{Name: "amount", Values: []any{int64(10), int64(20), int64(30)}},
	}})
	tt, _ := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{int64(-1), int64(0), int64(5)}},
		{Name: "v", Values: []any{10.0, 20.0, 30.0}},
		{Name: "city", Values: []any{"A", "B", "C"}},
	}})
	if err := ctx.RegisterMany(map[string]polars.DataFrame{"customers": customers, "orders": orders, "t": tt}); err != nil {
		t.Fatalf("register: %v", err)
	}
	lf, err := ctx.Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("gopolars Execute(%q): %v", query, err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("gopolars Collect(%q): %v", query, err)
	}
	return out.Rows()
}

// canonical re-marshals rows through encoding/json so numeric types and key
// ordering match across the Go/Python boundary.
func canonical(t *testing.T, rows []map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var normalized []map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	return string(out)
}

func TestSQLContextDifferentialAgainstPythonPolars(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import polars").Run(); err != nil {
		t.Skip("python polars is not installed")
	}

	cases := []sqlParityCase{
		{"inner_join", "SELECT name, amount FROM customers JOIN orders ON customers.id = orders.cust_id ORDER BY amount"},
		{"left_join", "SELECT name, amount FROM customers LEFT JOIN orders ON customers.id = orders.cust_id ORDER BY name, amount"},
		{"boolean_where", "SELECT x FROM t WHERE x > 0 OR x = 0 ORDER BY x"},
		{"case", "SELECT CASE WHEN x > 0 THEN 'pos' WHEN x < 0 THEN 'neg' ELSE 'zero' END AS sign FROM t ORDER BY x"},
		{"in_list", "SELECT city FROM t WHERE city IN ('A', 'C') ORDER BY city"},
		{"between", "SELECT v FROM t WHERE v BETWEEN 15.0 AND 25.0 ORDER BY v"},
		{"distinct", "SELECT DISTINCT name FROM customers ORDER BY name"},
		{"offset", "SELECT id FROM customers ORDER BY id LIMIT 2 OFFSET 1"},
		{"scalar_fn", "SELECT UPPER(name) AS u FROM customers ORDER BY u"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := exec.Command("python3", "-c", pythonSQLScript(tc.query)).Output()
			if err != nil {
				t.Fatalf("python polars failed for %q: %v", tc.query, err)
			}
			var pyRows []map[string]any
			if err := json.Unmarshal(out, &pyRows); err != nil {
				t.Fatalf("parse python output %q: %v\noutput=%s", tc.query, err, string(out))
			}
			goRows := gopolarsParityRows(t, tc.query)
			if got, want := canonical(t, goRows), canonical(t, pyRows); got != want {
				t.Fatalf("differential mismatch for %q:\ngopolars=%s\npython  =%s", tc.query, got, want)
			}
		})
	}
}
