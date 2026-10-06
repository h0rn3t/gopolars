package polars

import (
	"context"
	"fmt"
	"slices"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// sqlResult wraps an eagerly-computed result frame as a LazyFrame so the public
// SQL methods keep the LazyFrame-returning contract; a non-nil err is returned
// as is.
func sqlResult(res frame.DataFrame, err error) (LazyFrame, error) {
	if err != nil {
		return nil, err
	}
	return (&df{value: res}).Lazy(), nil
}

// SQL runs a SQL query against this DataFrame, which is addressable as `self`.
func (d *df) SQL(ctx context.Context, query string) (LazyFrame, error) {
	return sqlResult(execSQL(ctx, query, map[string]frame.DataFrame{"self": d.value}))
}

// Sql is the lowercase alias of SQL.
func (d *df) Sql(ctx context.Context, query string) (LazyFrame, error) {
	return d.SQL(ctx, query)
}

// SQL collects this LazyFrame and runs a SQL query against it under the given
// table name.
func (l *lf) SQL(ctx context.Context, query string, table string) (LazyFrame, error) {
	collected, err := l.collectFrame(ctx)
	if err != nil {
		return nil, err
	}
	return sqlResult(execSQL(ctx, query, map[string]frame.DataFrame{table: collected}))
}

// SQL runs a SQL query with no registered source table (e.g. `SELECT 1 AS x`).
func (f ioFacade) SQL(ctx context.Context, query string) (LazyFrame, error) {
	return sqlResult(execSQL(ctx, query, nil))
}

// NewSQLContext returns a SQLContext for multi-table SQL over registered frames.
func NewSQLContext() SQLContext {
	return &sqlContext{tables: map[string]frame.DataFrame{}}
}

type sqlContext struct {
	tables map[string]frame.DataFrame
}

func (c *sqlContext) Register(name string, d DataFrame) error {
	if name == "" {
		return fmt.Errorf("table name is required")
	}
	inner, ok := d.(*df)
	if !ok {
		return fmt.Errorf("unsupported dataframe implementation")
	}
	c.tables[name] = inner.value
	return nil
}

func (c *sqlContext) RegisterMany(tables map[string]DataFrame) error {
	for name, d := range tables {
		if err := c.Register(name, d); err != nil {
			return err
		}
	}
	return nil
}

// RegisterGlobals is an alias of RegisterMany kept for API compatibility.
func (c *sqlContext) RegisterGlobals(tables map[string]DataFrame) error {
	return c.RegisterMany(tables)
}

func (c *sqlContext) Unregister(name string) {
	delete(c.tables, name)
}

func (c *sqlContext) Tables() []string {
	names := make([]string, 0, len(c.tables))
	for name := range c.tables {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (c *sqlContext) Execute(ctx context.Context, query string) (LazyFrame, error) {
	return sqlResult(execSQL(ctx, query, c.tables))
}

// ExecuteGlobal is an alias of Execute kept for API compatibility.
func (c *sqlContext) ExecuteGlobal(ctx context.Context, query string) (LazyFrame, error) {
	return c.Execute(ctx, query)
}
