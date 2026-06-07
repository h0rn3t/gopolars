package polars

import (
	"context"
	"fmt"

	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/frame"
	gsql "github.com/h0rn3t/gopolars/pkg/sql"
)

type ParseSQLInput struct {
	Context context.Context
	Query   string
	Source  frame.DataFrame
	Engine  exec.Engine
	Table   string
	// Tables holds every table available to the query (for multi-table joins).
	// Source is always added under Table if not already present.
	Tables map[string]frame.DataFrame
}

func ParseSQL(input ParseSQLInput) (LazyFrame, error) {
	parsed, err := gsql.Parse(input.Query)
	if err != nil {
		return nil, err
	}
	if input.Table != "" && parsed.Table == "" {
		parsed.Table = input.Table
	}
	if parsed.Table == "" {
		return nil, fmt.Errorf("table is required")
	}
	tables := make(map[string]frame.DataFrame, len(input.Tables)+1)
	for k, v := range input.Tables {
		tables[k] = v
	}
	if input.Table != "" {
		if _, ok := tables[input.Table]; !ok {
			tables[input.Table] = input.Source
		}
	}
	catalog := gsql.NewCatalog(tables)
	bound, err := gsql.Bind(parsed, catalog)
	if err != nil {
		return nil, err
	}
	nodes, err := gsql.Plan(bound, catalog)
	if err != nil {
		return nil, err
	}
	return &lf{
		source: input.Source,
		engine: input.Engine,
		nodes:  nodes,
	}, nil
}

func SQLFromDataFrame(ctx context.Context, source DataFrame, query string, table string) (LazyFrame, error) {
	internal, ok := source.(*df)
	if !ok {
		return nil, fmt.Errorf("unsupported dataframe implementation")
	}
	return ParseSQL(ParseSQLInput{
		Context: ctx,
		Query:   query,
		Source:  internal.value,
		Engine:  exec.New(),
		Table:   table,
	})
}
