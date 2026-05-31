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
	bound, err := gsql.Bind(parsed)
	if err != nil {
		return nil, err
	}
	nodes := gsql.Plan(bound)
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
