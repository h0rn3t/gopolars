package polars

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/series"
	gsql "github.com/h0rn3t/gopolars/pkg/sql"
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
	stmt, err := gsql.ParseStatement(query)
	if err != nil {
		return nil, err
	}
	switch st := stmt.(type) {
	case gsql.SelectStatement:
		return s.executeSelect(st.Query)
	case gsql.ExplainSelect:
		lfr, err := s.executeSelect(st.Query)
		if err != nil {
			return nil, err
		}
		return s.literalLazy(stringFrame("plan", []string{lfr.Explain(true)}))
	case gsql.CreateTableAs:
		lfr, err := s.executeSelect(st.Query)
		if err != nil {
			return nil, err
		}
		out, err := lfr.Collect(ctx)
		if err != nil {
			return nil, err
		}
		internal, ok := out.(*df)
		if !ok {
			return nil, fmt.Errorf("unsupported dataframe implementation")
		}
		s.mu.Lock()
		s.tables[st.Name] = internal.value
		s.mu.Unlock()
		return s.literalLazy(stringFrame("response", []string{"CREATE TABLE"}))
	case gsql.DropTable:
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.tables[st.Name]; !ok {
			if !st.IfExists {
				return nil, fmt.Errorf("table %s is not registered", st.Name)
			}
			return s.literalLazy(stringFrame("response", []string{"DROP TABLE"}))
		}
		delete(s.tables, st.Name)
		return s.literalLazy(stringFrame("response", []string{"DROP TABLE"}))
	case gsql.TruncateTable:
		s.mu.Lock()
		defer s.mu.Unlock()
		t, ok := s.tables[st.Name]
		if !ok {
			return nil, fmt.Errorf("table %s is not registered", st.Name)
		}
		s.tables[st.Name] = t.Clear()
		return s.literalLazy(stringFrame("response", []string{"TRUNCATE TABLE"}))
	case gsql.ShowTables:
		return s.literalLazy(stringFrame("name", s.Tables()))
	default:
		return nil, fmt.Errorf("unsupported SQL statement")
	}
}

// executeSelect binds and plans a parsed SELECT against a snapshot of the
// registered tables, resolving FROM-clause table functions first.
func (s *sqlContext) executeSelect(parsed gsql.ParsedQuery) (LazyFrame, error) {
	s.mu.RLock()
	tables := make(map[string]frame.DataFrame, len(s.tables))
	for name, df := range s.tables {
		tables[name] = df
	}
	s.mu.RUnlock()
	if err := resolveTableFns(&parsed, tables); err != nil {
		return nil, err
	}
	targetTable := parsed.Table
	if targetTable == "" && len(tables) == 1 {
		for k := range tables {
			targetTable = k
		}
	}
	if targetTable == "" {
		return nil, fmt.Errorf("table is required for SQLContext execute")
	}
	source, ok := tables[targetTable]
	if !ok {
		// CTE and FROM-subquery names are not registered; the plan scans the
		// innermost real table instead.
		if inner := sourceTableOf(parsed); inner != targetTable {
			source, ok = tables[inner]
		}
	}
	if !ok {
		return nil, fmt.Errorf("table %s is not registered", targetTable)
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
	return &lf{source: source, engine: s.engine, nodes: nodes}, nil
}

// sourceTableOf walks through CTE and FROM-subquery wrappers to the innermost
// scanned table name.
func sourceTableOf(parsed gsql.ParsedQuery) string {
	q := parsed
	for {
		switch {
		case q.CTE != nil && q.CTEName != "" && q.Table == q.CTEName:
			q = *q.CTE
		case q.Subquery != nil:
			q = *q.Subquery
		default:
			return q.Table
		}
	}
}

func (s *sqlContext) literalLazy(f frame.DataFrame, err error) (LazyFrame, error) {
	if err != nil {
		return nil, err
	}
	return &lf{source: f, engine: s.engine}, nil
}

// stringFrame builds a single-column String frame from the given values.
func stringFrame(column string, values []string) (frame.DataFrame, error) {
	boxed := make([]any, len(values))
	for i, v := range values {
		boxed[i] = v
	}
	col, err := series.New(column, dtypes.String, boxed)
	if err != nil {
		return frame.DataFrame{}, err
	}
	return frame.New(frame.NewInput{Series: []series.Series{col}})
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
