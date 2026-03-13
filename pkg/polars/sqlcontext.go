package polars

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/eugeneshershen/gopolars/pkg/exec"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	gsql "github.com/eugeneshershen/gopolars/pkg/sql"
)

type sqlContext struct {
	mu     sync.RWMutex
	tables map[string]frame.DataFrame
	engine exec.Engine
}

func NewSQLContext() SQLContext {
	return &sqlContext{
		tables: map[string]frame.DataFrame{},
		engine: exec.New(),
	}
}

func (s *sqlContext) Register(name string, frameRef DataFrame) error {
	if name == "" {
		return fmt.Errorf("table name is empty")
	}
	internal, ok := frameRef.(*df)
	if !ok {
		return fmt.Errorf("unsupported dataframe implementation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tables[name] = internal.value
	return nil
}

func (s *sqlContext) RegisterMany(tables map[string]DataFrame) error {
	for name, df := range tables {
		if err := s.Register(name, df); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlContext) Execute(ctx context.Context, query string) (LazyFrame, error) {
	parsed, err := gsql.Parse(query)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	targetTable := parsed.Table
	if targetTable == "" {
		if len(s.tables) == 1 {
			for k := range s.tables {
				targetTable = k
			}
		}
	}
	if targetTable == "" {
		return nil, fmt.Errorf("table is required for SQLContext execute")
	}
	source, ok := s.tables[targetTable]
	if !ok {
		return nil, fmt.Errorf("table %s is not registered", targetTable)
	}
	return ParseSQL(ParseSQLInput{
		Context: ctx,
		Query:   query,
		Source:  source,
		Engine:  s.engine,
		Table:   targetTable,
	})
}

func (s *sqlContext) ExecuteGlobal(ctx context.Context, query string) (LazyFrame, error) {
	return s.Execute(ctx, query)
}

func (s *sqlContext) RegisterGlobals(tables map[string]DataFrame) error {
	return s.RegisterMany(tables)
}

func (s *sqlContext) Tables() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.tables))
	for name := range s.tables {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *sqlContext) Unregister(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tables, name)
}
