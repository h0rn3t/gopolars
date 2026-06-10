package polars

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/frame"
	icsv "github.com/h0rn3t/gopolars/pkg/io/csv"
	iipc "github.com/h0rn3t/gopolars/pkg/io/ipc"
	ijson "github.com/h0rn3t/gopolars/pkg/io/json"
	iparquet "github.com/h0rn3t/gopolars/pkg/io/parquet"
	gsql "github.com/h0rn3t/gopolars/pkg/sql"
)

// readTableFn loads the frame behind a FROM-clause table function using the
// matching pkg/io reader with default options.
func readTableFn(fn *gsql.TableFn) (frame.DataFrame, error) {
	switch fn.Name {
	case "read_csv":
		return icsv.Read(icsv.ReadInput{Path: fn.Path, HasHeader: true})
	case "read_parquet":
		return iparquet.Read(iparquet.ReadInput{Path: fn.Path})
	case "read_json":
		lower := strings.ToLower(fn.Path)
		ndjson := strings.HasSuffix(lower, ".ndjson") || strings.HasSuffix(lower, ".jsonl")
		return ijson.Read(ijson.ReadInput{Path: fn.Path, NDJSON: ndjson})
	case "read_ipc":
		return iipc.Read(iipc.ReadInput{Path: fn.Path})
	}
	return frame.DataFrame{}, fmt.Errorf("unknown table function %s", fn.Name)
}

// resolveTableFns eagerly loads every table-function reference in the query
// tree, registers the loaded frame in tables under the reference's alias (or a
// unique synthetic name), and rewrites the reference to a plain table name so
// the binder and planner treat it like any registered table.
func resolveTableFns(q *gsql.ParsedQuery, tables map[string]frame.DataFrame) error {
	if q == nil {
		return nil
	}
	if err := resolveTableFnRef(&q.From.Primary, tables); err != nil {
		return err
	}
	q.Table = q.From.Primary.Name
	for i := range q.From.Joins {
		if err := resolveTableFnRef(&q.From.Joins[i].Table, tables); err != nil {
			return err
		}
	}
	for _, sub := range []*gsql.ParsedQuery{q.CTE, q.Subquery, q.SetRight} {
		if err := resolveTableFns(sub, tables); err != nil {
			return err
		}
	}
	return nil
}

func resolveTableFnRef(ref *gsql.TableRef, tables map[string]frame.DataFrame) error {
	if ref.Subquery != nil {
		return resolveTableFns(ref.Subquery, tables)
	}
	if ref.Fn == nil {
		return nil
	}
	loaded, err := readTableFn(ref.Fn)
	if err != nil {
		return fmt.Errorf("%s: %w", ref.Fn.Name, err)
	}
	name := ref.Alias
	if name == "" {
		name = ref.Fn.Name
		for i := 2; ; i++ {
			if _, taken := tables[name]; !taken {
				break
			}
			name = ref.Fn.Name + "_" + strconv.Itoa(i)
		}
	}
	tables[name] = loaded
	ref.Name = name
	ref.Fn = nil
	return nil
}
