package polars

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/exec"
	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	icsv "github.com/eugeneshershen/gopolars/pkg/io/csv"
	iipc "github.com/eugeneshershen/gopolars/pkg/io/ipc"
	ijson "github.com/eugeneshershen/gopolars/pkg/io/json"
	iparquet "github.com/eugeneshershen/gopolars/pkg/io/parquet"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
	"github.com/eugeneshershen/gopolars/pkg/plan/optimizer"
	"github.com/eugeneshershen/gopolars/pkg/plan/physical"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type lf struct {
	source frame.DataFrame
	engine exec.Engine
	nodes  []logical.Node
	scan   *scanSource
}

func (l *lf) Select(exprs ...Expr) LazyFrame {
	internal := make([]expr.Expr, len(exprs))
	copy(internal, exprs)
	return l.withNode(logical.Node{Type: logical.NodeSelect, Exprs: internal})
}

func (l *lf) Filter(predicate Expr) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{predicate}})
}

func (l *lf) WithColumns(exprs ...Expr) LazyFrame {
	internal := make([]expr.Expr, len(exprs))
	copy(internal, exprs)
	return l.withNode(logical.Node{Type: logical.NodeWithCols, Exprs: internal})
}

func (l *lf) GroupBy(keys ...string) LazyGroupBy {
	return lazyGroupBy{lf: l, keys: keys}
}

func (l *lf) GroupByDynamic(input DynamicGroupInput) LazyFrame {
	return l.withNode(logical.Node{
		Type:    logical.NodeDynamic,
		Columns: []string{input.By, input.WindowColumn},
		Strings: []string{
			fmt.Sprintf("%d", input.Every.Nanoseconds()),
			fmt.Sprintf("%d", input.Period.Nanoseconds()),
			fmt.Sprintf("%d", input.Offset.Nanoseconds()),
			input.Closed,
			input.Label,
		},
		Exprs: []expr.Expr{input.AggExpr},
	})
}

func (l *lf) Join(input JoinInput) LazyFrame {
	other, ok := input.Other.(*df)
	if !ok {
		return l
	}
	return l.withNode(logical.Node{
		Type: logical.NodeJoin,
		Join: &logical.JoinSpec{
			Other:         other.value,
			LeftOn:        input.LeftOn,
			RightOn:       input.RightOn,
			How:           frame.JoinType(input.How),
			Suffix:        input.Suffix,
			AsofDirection: input.AsofDirection,
			AsofTolerance: input.AsofTolerance,
		},
	})
}

func (l *lf) Sort(input SortInput) LazyFrame {
	return l.withNode(logical.Node{
		Type:       logical.NodeSort,
		Columns:    input.By,
		Descending: input.Descending,
	})
}

func (l *lf) Limit(n int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeLimit, IntValue: n})
}

func (l *lf) Slice(offset int, length int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeSlice, IntValue: offset, Strings: []string{fmt.Sprintf("%d", length)}})
}

func (l *lf) Unique(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeUnique, Columns: columns})
}

func (l *lf) FillNull(value any) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFillNull, Exprs: []expr.Expr{expr.Lit(value)}})
}

func (l *lf) DropNulls(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeDropNulls, Columns: columns})
}

func (l *lf) Explode(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeExplode, Columns: columns})
}

func (l *lf) FlattenStruct(column string, prefix string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFlatten, Columns: []string{column}, Prefix: prefix})
}

func (l *lf) Melt(input MeltInput) LazyFrame {
	return l.withNode(logical.Node{
		Type:    logical.NodeMelt,
		Columns: append(append([]string{}, input.IDVars...), input.ValueVars...),
		Strings: []string{input.VariableCol, input.ValueCol, fmt.Sprintf("%d", len(input.IDVars))},
	})
}

func (l *lf) Pivot(input PivotInput) LazyFrame {
	idx := input.Index
	return l.withNode(logical.Node{
		Type:    logical.NodePivot,
		Columns: []string{idx, input.Columns, input.Values},
		Strings: []string{input.Agg, input.ValueName},
	})
}

func (l *lf) RollingMean(input RollingMeanInput) LazyFrame {
	return l.withNode(logical.Node{
		Type:    logical.NodeRolling,
		Columns: []string{input.By, input.Value, input.Output},
		Strings: []string{
			fmt.Sprintf("%d", input.Window.Nanoseconds()),
			fmt.Sprintf("%d", input.MinRows),
			input.Closed,
		},
	})
}

func (l *lf) Collect(ctx context.Context) (DataFrame, error) {
	source, nodes, err := l.resolveSource()
	if err != nil {
		return nil, err
	}
	df, err := l.engine.Execute(ctx, source, nodes)
	return fromFrame(df, err)
}

func (l *lf) CollectStreaming(ctx context.Context, chunkSize int) (DataFrame, error) {
	source, nodes, err := l.resolveSource()
	if err != nil {
		return nil, err
	}
	df, err := l.engine.ExecuteStreaming(ctx, source, nodes, chunkSize)
	return fromFrame(df, err)
}

func (l *lf) SinkCSV(ctx context.Context, input WriteCSVInput) error {
	df, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return df.WriteCSV(input)
}

func (l *lf) SinkParquet(ctx context.Context, input WriteParquetInput) error {
	df, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return df.WriteParquet(input)
}

func (l *lf) SinkIPC(ctx context.Context, input WriteIPCInput) error {
	df, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return df.WriteIPC(input)
}

func (l *lf) Explain(optimized bool) string {
	logicalStage := renderStage(l.nodes)
	if !optimized {
		return "logical=[" + logicalStage + "]"
	}
	optimizedNodes := optimizer.Optimize(l.nodes)
	optimizedStage := renderStage(optimizedNodes)
	physicalPlan := physical.Build(optimizedNodes)
	physicalOps := make([]logical.Node, 0, len(physicalPlan.Operators))
	for _, op := range physicalPlan.Operators {
		physicalOps = append(physicalOps, op.Node)
	}
	physicalStage := renderStage(physicalOps)
	return "logical=[" + logicalStage + "] optimized=[" + optimizedStage + "] physical=[" + physicalStage + "]"
}

func (l *lf) ExplainDiagnostics(optimized bool) map[string]any {
	diag := map[string]any{
		"schema_version": "v2",
		"logical_nodes":  len(l.nodes),
		"scan_source":    "in_memory",
	}
	if l.scan != nil {
		diag["scan_source"] = l.scan.format
		diag["scan_path"] = l.scan.path
	}
	optimizedNodes := l.nodes
	if optimized {
		optimizedNodes = optimizer.Optimize(l.nodes)
	}
	stateful := false
	window := 0
	reshape := 0
	setOps := 0
	temporal := 0
	perfMarkers := map[string]any{
		"stateful_nodes":        0,
		"window_nodes":          0,
		"temporal_window_nodes": 0,
	}
	for _, n := range optimizedNodes {
		if n.Type == logical.NodeSort || n.Type == logical.NodeJoin || n.Type == logical.NodeAggregate || n.Type == logical.NodeWindow || n.Type == logical.NodePivot || n.Type == logical.NodeSetOp || n.Type == logical.NodeRolling || n.Type == logical.NodeDynamic {
			stateful = true
			perfMarkers["stateful_nodes"] = perfMarkers["stateful_nodes"].(int) + 1
		}
		if n.Type == logical.NodeWindow {
			window += len(n.Windows)
			perfMarkers["window_nodes"] = perfMarkers["window_nodes"].(int) + 1
		}
		if n.Type == logical.NodeMelt || n.Type == logical.NodePivot {
			reshape++
		}
		if n.Type == logical.NodeSetOp {
			setOps++
		}
		if n.Type == logical.NodeRolling || n.Type == logical.NodeDynamic {
			temporal++
			perfMarkers["temporal_window_nodes"] = perfMarkers["temporal_window_nodes"].(int) + 1
		}
	}
	diag["optimized"] = optimized
	diag["optimized_nodes"] = len(optimizedNodes)
	diag["stateful_pipeline"] = stateful
	diag["window_expressions"] = window
	diag["reshape_operations"] = reshape
	diag["set_operations"] = setOps
	diag["temporal_window_operations"] = temporal
	diag["performance_markers"] = perfMarkers
	diag["plan"] = renderStage(optimizedNodes)
	return diag
}

func (l *lf) withNode(node logical.Node) *lf {
	next := &lf{
		source: l.source,
		engine: l.engine,
		nodes:  make([]logical.Node, 0, len(l.nodes)+1),
		scan:   l.scan,
	}
	next.nodes = append(next.nodes, l.nodes...)
	next.nodes = append(next.nodes, node)
	return next
}

type scanSource struct {
	format string
	path   string
	csv    ScanCSVInput
	json   ScanJSONInput
	ips    ScanIPCInput
	parq   ScanParquetInput
}

func (l *lf) resolveSource() (frame.DataFrame, []logical.Node, error) {
	if l.scan == nil {
		return l.source, l.nodes, nil
	}
	columns := projectedColumns(l.nodes)
	pushed, remaining := splitPushdownFilters(l.nodes)
	path, err := resolveObjectStorePath(l.scan.path)
	if err != nil {
		return frame.DataFrame{}, nil, err
	}
	var base frame.DataFrame
	switch l.scan.format {
	case "csv":
		c := l.scan.csv
		c.Path = path
		base, err = icsv.Read(icsv.ReadInput{
			Path:      c.Path,
			HasHeader: c.HasHeader,
			Separator: c.Separator,
			Schema:    c.Schema,
			Columns:   columns,
		})
	case "json":
		j := l.scan.json
		j.Path = path
		base, err = ijson.Read(ijson.ReadInput{
			Path:    j.Path,
			NDJSON:  j.NDJSON,
			Schema:  j.Schema,
			Columns: columns,
		})
	case "ipc":
		base, err = iipc.Read(iipc.ReadInput{Path: path, Columns: columns})
	case "parquet":
		p := l.scan.parq
		p.Path = path
		if len(columns) > 0 {
			p.Columns = columns
		}
		base, err = readParquetSource(p.Path, p.Columns, pushed)
	default:
		err = fmt.Errorf("unsupported scan source format %s", l.scan.format)
	}
	if err != nil {
		return frame.DataFrame{}, nil, err
	}
	for _, p := range pushed {
		base, err = base.Filter(p)
		if err != nil {
			return frame.DataFrame{}, nil, err
		}
	}
	return base, remaining, nil
}

func readParquetSource(path string, columns []string, pushed []expr.Expr) (frame.DataFrame, error) {
	info, err := os.Stat(path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	if !info.IsDir() {
		return iparquet.Read(iparquet.ReadInput{Path: path, Columns: columns})
	}
	files := make([]string, 0)
	if err := filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if i.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".parquet") {
			files = append(files, p)
		}
		return nil
	}); err != nil {
		return frame.DataFrame{}, err
	}
	if len(files) == 0 {
		return frame.New(frame.NewInput{})
	}
	constraints := partitionConstraints(pushed)
	parts := make([]frame.DataFrame, 0, len(files))
	for _, f := range files {
		meta := partitionFromPath(path, f)
		if !matchPartition(meta, constraints) {
			continue
		}
		readCols := make([]string, 0, len(columns))
		for _, c := range columns {
			if _, isPartition := meta[c]; isPartition {
				continue
			}
			readCols = append(readCols, c)
		}
		if len(readCols) == 0 {
			readCols = nil
		}
		df, err := iparquet.Read(iparquet.ReadInput{Path: f, Columns: readCols})
		if err != nil {
			return frame.DataFrame{}, err
		}
		if len(meta) > 0 {
			df, err = addPartitionColumns(df, meta)
			if err != nil {
				return frame.DataFrame{}, err
			}
		}
		if len(columns) > 0 {
			exprs := make([]expr.Expr, 0, len(columns))
			for _, c := range columns {
				if _, ok := meta[c]; ok || hasColumn(df, c) {
					exprs = append(exprs, expr.Col(c))
				}
			}
			if len(exprs) > 0 {
				df, err = df.Select(exprs...)
				if err != nil {
					return frame.DataFrame{}, err
				}
			}
		}
		parts = append(parts, df)
	}
	if len(parts) == 0 {
		return frame.New(frame.NewInput{})
	}
	base := parts[0]
	if len(parts) == 1 {
		return base, nil
	}
	return frame.ConcatVertical(base, parts[1:]...)
}

func partitionFromPath(root string, file string) map[string]string {
	out := map[string]string{}
	rel, err := filepath.Rel(root, filepath.Dir(file))
	if err != nil || rel == "." {
		return out
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		out[pair[0]] = pair[1]
	}
	return out
}

func partitionConstraints(filters []expr.Expr) map[string]string {
	out := map[string]string{}
	for _, f := range filters {
		if f.Op() != "eq" || f.Left() == nil || f.Right() == nil {
			continue
		}
		if f.Left().Kind() == expr.KindCol && f.Right().Kind() == expr.KindLit {
			out[f.Left().ColName()] = fmt.Sprintf("%v", f.Right().Value())
			continue
		}
		if f.Left().Kind() == expr.KindLit && f.Right().Kind() == expr.KindCol {
			out[f.Right().ColName()] = fmt.Sprintf("%v", f.Left().Value())
		}
	}
	return out
}

func matchPartition(meta map[string]string, constraints map[string]string) bool {
	for key, expected := range constraints {
		if got, ok := meta[key]; ok && got != expected {
			return false
		}
	}
	return true
}

func addPartitionColumns(df frame.DataFrame, meta map[string]string) (frame.DataFrame, error) {
	existing := map[string]struct{}{}
	for _, name := range df.Columns() {
		existing[name] = struct{}{}
	}
	out := make([]series.Series, 0, df.Width()+len(meta))
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		out = append(out, s.Clone())
	}
	for key, value := range meta {
		if _, ok := existing[key]; ok {
			continue
		}
		values := make([]any, df.Height())
		for i := range values {
			values[i] = value
		}
		s, err := series.New(key, dtypes.String, values)
		if err != nil {
			return frame.DataFrame{}, err
		}
		out = append(out, s)
	}
	return frame.New(frame.NewInput{Series: out})
}

func hasColumn(df frame.DataFrame, column string) bool {
	for _, c := range df.Columns() {
		if c == column {
			return true
		}
	}
	return false
}

func projectedColumns(nodes []logical.Node) []string {
	required := map[string]struct{}{}
	for _, n := range nodes {
		switch n.Type {
		case logical.NodeSelect, logical.NodeWithCols:
			for _, e := range n.Exprs {
				if e.Kind() == expr.KindCol {
					required[e.ColName()] = struct{}{}
				}
			}
		case logical.NodeFilter:
			if len(n.Exprs) == 0 {
				continue
			}
			for _, c := range collectExprColumns(n.Exprs[0]) {
				required[c] = struct{}{}
			}
		case logical.NodeSort, logical.NodeAggregate, logical.NodeJoin, logical.NodeUnique, logical.NodeDropNulls:
			for _, c := range n.Columns {
				required[c] = struct{}{}
			}
		case logical.NodeExplode, logical.NodeFlatten:
			for _, c := range n.Columns {
				required[c] = struct{}{}
			}
		case logical.NodeMelt, logical.NodePivot:
			for _, c := range n.Columns {
				required[c] = struct{}{}
			}
		case logical.NodeRolling:
			limit := len(n.Columns)
			if limit > 2 {
				limit = 2
			}
			for i := 0; i < limit; i++ {
				c := n.Columns[i]
				if c == "" {
					continue
				}
				required[c] = struct{}{}
			}
		case logical.NodeDynamic:
			if len(n.Columns) > 0 && n.Columns[0] != "" {
				required[n.Columns[0]] = struct{}{}
			}
			for _, e := range n.Exprs {
				for _, c := range collectExprColumns(e) {
					required[c] = struct{}{}
				}
			}
		case logical.NodeWindow:
			for _, w := range n.Windows {
				for _, c := range w.PartitionBy {
					required[c] = struct{}{}
				}
				for _, c := range w.OrderBy {
					required[c] = struct{}{}
				}
				if w.Target != "" && w.Target != "*" {
					required[w.Target] = struct{}{}
				}
			}
		}
	}
	if len(required) == 0 {
		return nil
	}
	out := make([]string, 0, len(required))
	for c := range required {
		out = append(out, c)
	}
	return out
}

func collectExprColumns(e expr.Expr) []string {
	out := map[string]struct{}{}
	var walk func(x expr.Expr)
	walk = func(x expr.Expr) {
		if x.Kind() == expr.KindCol {
			out[x.ColName()] = struct{}{}
		}
		if x.Left() != nil {
			walk(*x.Left())
		}
		if x.Right() != nil {
			walk(*x.Right())
		}
		if x.Target() != nil {
			walk(*x.Target())
		}
		if x.Extra() != nil {
			walk(*x.Extra())
		}
	}
	walk(e)
	values := make([]string, 0, len(out))
	for c := range out {
		values = append(values, c)
	}
	return values
}

func splitPushdownFilters(nodes []logical.Node) ([]expr.Expr, []logical.Node) {
	pushed := make([]expr.Expr, 0)
	remaining := make([]logical.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Type != logical.NodeFilter || len(n.Exprs) == 0 {
			remaining = append(remaining, n)
			continue
		}
		if isPushdownFilter(n.Exprs[0]) {
			pushed = append(pushed, n.Exprs[0])
			continue
		}
		remaining = append(remaining, n)
	}
	return pushed, remaining
}

func isPushdownFilter(e expr.Expr) bool {
	if e.Kind() != expr.KindBin {
		return false
	}
	switch e.Op() {
	case "eq", "ne", "gt", "ge", "lt", "le", "and":
	default:
		return false
	}
	if e.Op() == "and" && e.Left() != nil && e.Right() != nil {
		return isPushdownFilter(*e.Left()) && isPushdownFilter(*e.Right())
	}
	if e.Left() == nil || e.Right() == nil {
		return false
	}
	leftColRightLit := e.Left().Kind() == expr.KindCol && e.Right().Kind() == expr.KindLit
	rightColLeftLit := e.Left().Kind() == expr.KindLit && e.Right().Kind() == expr.KindCol
	return leftColRightLit || rightColLeftLit
}

type lazyGroupBy struct {
	lf   *lf
	keys []string
}

func (g lazyGroupBy) Agg(exprs ...Expr) LazyFrame {
	internal := make([]expr.Expr, len(exprs))
	copy(internal, exprs)
	return g.lf.withNode(logical.Node{
		Type:    logical.NodeAggregate,
		Columns: g.keys,
		Exprs:   internal,
	})
}

func (l *lf) ensureState() error {
	if l == nil {
		return fmt.Errorf("lazyframe is nil")
	}
	return nil
}

func renderStage(nodes []logical.Node) string {
	out := make([]string, 0, len(nodes)+1)
	out = append(out, "scan")
	for _, n := range nodes {
		out = append(out, string(n.Type))
	}
	return strings.Join(out, " -> ")
}
