package polars

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/exec"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	icsv "github.com/h0rn3t/gopolars/pkg/io/csv"
	iipc "github.com/h0rn3t/gopolars/pkg/io/ipc"
	ijson "github.com/h0rn3t/gopolars/pkg/io/json"
	iparquet "github.com/h0rn3t/gopolars/pkg/io/parquet"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
	"github.com/h0rn3t/gopolars/pkg/plan/optimizer"
	"github.com/h0rn3t/gopolars/pkg/series"
)

type lf struct {
	source frame.DataFrame
	engine exec.Engine
	nodes  []logical.Node
	scan   *scanSource
}

func newLazy(source frame.DataFrame, scan *scanSource) *lf {
	return &lf{source: source, engine: exec.New(), nodes: []logical.Node{}, scan: scan}
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

func (l *lf) ApproxNUnique(columns ...string) LazyFrame {
	if len(columns) == 0 {
		return l
	}
	return l.Unique(columns...)
}

func (l *lf) BottomK(k int, by string) LazyFrame {
	return l.Sort(SortInput{By: []string{by}}).Limit(k)
}

func (l *lf) Limit(n int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeLimit, IntValue: n})
}

func (l *lf) Head(n int) LazyFrame {
	return l.Limit(n)
}

func (l *lf) Tail(n int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeTail, IntValue: n})
}

func (l *lf) First() LazyFrame {
	return l.Head(1)
}

func (l *lf) Last() LazyFrame {
	return l.Tail(1)
}

func (l *lf) Slice(offset int, length int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeSlice, IntValue: offset, Strings: []string{fmt.Sprintf("%d", length)}})
}

func (l *lf) GatherEvery(step int, offset int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeGatherEvery, IntValue: offset, Strings: []string{fmt.Sprintf("%d", step)}})
}

func (l *lf) Clear() LazyFrame {
	return l.Limit(0)
}

func (l *lf) Clone() LazyFrame {
	next := &lf{
		source: l.source,
		engine: l.engine,
		nodes:  make([]logical.Node, len(l.nodes)),
		scan:   l.scan,
	}
	copy(next.nodes, l.nodes)
	return next
}

func (l *lf) Reverse() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeReverse})
}

func (l *lf) Rename(mapping map[string]string) LazyFrame {
	if len(mapping) == 0 {
		return l
	}
	stringsArr := make([]string, 0, len(mapping)*2)
	for k, v := range mapping {
		stringsArr = append(stringsArr, k, v)
	}
	return l.withNode(logical.Node{Type: logical.NodeRename, Strings: stringsArr})
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

func (l *lf) DropNaNs(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeDropNans, Columns: columns})
}

func (l *lf) Drop(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeDrop, Columns: columns})
}

func (l *lf) Explode(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeExplode, Columns: columns})
}

func (l *lf) Unnest(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeUnnest, Columns: columns})
}

func (l *lf) FlattenStruct(column string, prefix string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFlatten, Columns: []string{column}, Prefix: prefix})
}

func (l *lf) Melt(input MeltInput) LazyFrame {
	return l.withNode(meltNode(logical.NodeMelt, input))
}

func (l *lf) Unpivot(input MeltInput) LazyFrame {
	return l.withNode(meltNode(logical.NodeUnpivot, input))
}

func meltNode(t logical.NodeType, in MeltInput) logical.Node {
	return logical.Node{
		Type:    t,
		Columns: append(append([]string{}, in.IDVars...), in.ValueVars...),
		Strings: []string{in.VariableCol, in.ValueCol, fmt.Sprintf("%d", len(in.IDVars))},
	}
}

func (l *lf) WithRowIndex(name string, offset int64) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeWithRowIdx, Strings: []string{name, fmt.Sprintf("%d", offset)}})
}

func (l *lf) WithRowCount(name string, offset int64) LazyFrame {
	return l.WithRowIndex(name, offset)
}

func (l *lf) Shift(periods int) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeShift, IntValue: periods})
}

func (l *lf) SetSorted(by string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeSetSorted, Columns: []string{by}})
}

func (l *lf) Cast(mapping map[string]dtypes.DataType) LazyFrame {
	if len(mapping) == 0 {
		return l
	}
	stringsArr := make([]string, 0, len(mapping)*2)
	for k, v := range mapping {
		stringsArr = append(stringsArr, k, string(v))
	}
	return l.withNode(logical.Node{Type: logical.NodeCast, Strings: stringsArr})
}

func (l *lf) FillNaN(value float64) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFillNaN, Strings: []string{strconv.FormatFloat(value, 'g', -1, 64)}})
}

func (l *lf) Interpolate(columns ...string) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeInterpolate, Columns: columns})
}

// Update collects other now, so its own scan and operations apply, and records
// the result (or the collect error, which Collect then returns) on the node.
func (l *lf) Update(other LazyFrame) LazyFrame {
	node := logical.Node{Type: logical.NodeUpdate}
	otherLf, ok := other.(*lf)
	if !ok {
		node.Err = fmt.Errorf("update: unsupported lazyframe implementation")
		return l.withNode(node)
	}
	otherFrame, err := otherLf.collectFrame(context.Background())
	if err != nil {
		node.Err = fmt.Errorf("update: collect other frame: %w", err)
	} else {
		node.Other = &otherFrame
	}
	return l.withNode(node)
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

func (l *lf) Max() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"max"}})
}

func (l *lf) Min() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"min"}})
}

func (l *lf) Mean() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"mean"}})
}

func (l *lf) Median() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"median"}})
}

func (l *lf) Sum() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"sum"}})
}

func (l *lf) Std() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"std"}})
}

func (l *lf) Var() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"var"}})
}

func (l *lf) Quantile(q float64) LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"quantile", strconv.FormatFloat(q, 'g', -1, 64)}})
}

func (l *lf) NullCount() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"null_count"}})
}

func (l *lf) Count() LazyFrame {
	return l.withNode(logical.Node{Type: logical.NodeFrameAgg, Strings: []string{"count"}})
}

func (l *lf) CollectSchema() dtypes.Schema {
	return l.source.CollectSchema()
}

func (l *lf) Columns() []string {
	return l.source.Columns()
}

func (l *lf) Width() int {
	return l.source.Width()
}

func (l *lf) Schema() dtypes.Schema {
	return l.source.Schema()
}

func (l *lf) Dtypes() []dtypes.DataType {
	return l.source.Dtypes()
}

func (l *lf) Describe() (DataFrame, error) {
	return (&df{value: l.source}).Describe()
}

func (l *lf) JoinAsof(input JoinInput) LazyFrame {
	if input.How == "" {
		input.How = "asof"
	}
	return l.Join(input)
}

func (l *lf) MapBatches(fn func(DataFrame) (DataFrame, error)) LazyFrame {
	if fn == nil {
		return l
	}
	current, err := l.Collect(context.Background())
	if err != nil {
		return l
	}
	next, err := fn(current)
	if err != nil || next == nil {
		return l
	}
	return next.Lazy()
}

func (l *lf) MatchToSchema(schema dtypes.Schema) LazyFrame {
	return l.MapBatches(func(in DataFrame) (DataFrame, error) {
		return in.MatchToSchema(schema)
	})
}

func (l *lf) SubSelectColumns(columns ...string) LazyFrame {
	if len(columns) == 0 {
		return l.Clone()
	}
	exprs := make([]Expr, 0, len(columns))
	for _, c := range columns {
		exprs = append(exprs, Col(c))
	}
	return l.Select(exprs...)
}

func (l *lf) MergeSorted(other LazyFrame, by string) LazyFrame {
	current, err := l.Collect(context.Background())
	if err != nil {
		return l
	}
	otherDF, err := other.Collect(context.Background())
	if err != nil {
		return l
	}
	next, err := current.MergeSorted(otherDF, by)
	if err != nil {
		return l
	}
	return next.Lazy()
}

func (l *lf) Pipe(fn func(LazyFrame) LazyFrame) LazyFrame {
	if fn == nil {
		return l
	}
	return fn(l)
}

func (l *lf) PipeWithSchema(fn func(LazyFrame, dtypes.Schema) LazyFrame, schema dtypes.Schema) LazyFrame {
	if fn == nil {
		return l
	}
	return fn(l, schema)
}

func (l *lf) Rolling(input RollingMeanInput) LazyFrame {
	return l.RollingMean(input)
}

func (l *lf) SelectSeq(exprs ...Expr) LazyFrame {
	return l.Select(exprs...)
}

func (l *lf) WithColumnsSeq(exprs ...Expr) LazyFrame {
	return l.WithColumns(exprs...)
}

// WithContext makes the columns of other referenceable by expressions in
// subsequent operations (pl.LazyFrame.with_context). The context columns are not
// added to this frame's own columns and may have a different length; they are
// resolved as a fallback, typically as scalars via an aggregation like .first().
func (l *lf) WithContext(other LazyFrame) LazyFrame {
	if other == nil {
		return l
	}
	collected, err := other.Collect(context.Background())
	if err != nil {
		return l
	}
	od, ok := collected.(*df)
	if !ok {
		return l
	}
	ctxCols := map[string]series.Series{}
	for _, name := range od.value.Columns() {
		ctxCols[name], _ = od.value.Series(name)
	}
	next := &lf{
		source: l.source.WithContextColumns(ctxCols),
		engine: l.engine,
		nodes:  append([]logical.Node(nil), l.nodes...),
		scan:   l.scan,
	}
	return next
}

func (l *lf) Remove(column string) LazyFrame {
	return l.Drop(column)
}

func (l *lf) TopK(k int, by string) LazyFrame {
	return l.Sort(SortInput{By: []string{by}, Descending: []bool{true}}).Limit(k)
}

func (l *lf) Show(maxRows int) string {
	out, err := l.Collect(context.Background())
	if err != nil {
		return err.Error()
	}
	return out.Show(maxRows)
}

func (l *lf) Lazy() LazyFrame {
	return l.Clone()
}

func (l *lf) Cache() LazyFrame {
	return l
}

func (l *lf) ShowGraph() string {
	return l.Explain(true)
}

func (l *lf) Serialize() ([]byte, error) {
	return json.Marshal(l.ExplainDiagnostics(true))
}

func (l *lf) Deserialize(payload []byte) (LazyFrame, error) {
	var m map[string]any
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *lf) Remote(endpoint string) LazyFrame {
	return l
}

func (l *lf) SinkBatches(ctx context.Context, chunkSize int) <-chan AsyncCollectResult {
	return l.CollectBatches(ctx, chunkSize)
}

func (l *lf) CollectAsync(ctx context.Context) <-chan AsyncCollectResult {
	ch := make(chan AsyncCollectResult, 1)
	go func() {
		defer close(ch)
		out, err := l.Collect(ctx)
		ch <- AsyncCollectResult{DataFrame: out, Error: err}
	}()
	return ch
}

func (l *lf) CollectBatches(ctx context.Context, chunkSize int) <-chan AsyncCollectResult {
	ch := make(chan AsyncCollectResult, 1)
	go func() {
		defer close(ch)
		if chunkSize <= 0 {
			chunkSize = 1024
		}
		collected, err := l.collectFrame(ctx)
		if err != nil {
			ch <- AsyncCollectResult{Error: err}
			return
		}
		for start := 0; start < collected.Height(); start += chunkSize {
			ch <- AsyncCollectResult{DataFrame: &df{value: collected.Slice(start, chunkSize)}}
		}
	}()
	return ch
}

func (l *lf) Inspect() LazyFrame {
	return l
}

func (l *lf) Profile(ctx context.Context) (DataFrame, map[string]any, error) {
	source, nodes, err := l.resolveSource()
	if err != nil {
		return nil, nil, err
	}
	out, report, err := l.engine.ExecuteWithReport(ctx, source, nodes)
	profile := map[string]any{
		"schema_version": report.SchemaVersion,
		"operators":      report.Operators,
		"duration_ms":    report.DurationMS,
		"memory_bytes":   report.MemoryBytes,
		"temporal_ops":   report.TemporalOps,
	}
	if err != nil {
		return nil, profile, err
	}
	profile["source_rows"] = report.SourceRows
	profile["output_rows"] = report.OutputRows
	return &df{value: out}, profile, nil
}

func (l *lf) JoinWhere(predicate Expr) LazyFrame {
	return l.Filter(predicate)
}

func (l *lf) SinkNDJSON(ctx context.Context, input WriteJSONInput) error {
	input.NDJSON = true
	out, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return out.WriteJSON(input)
}

func (l *lf) Collect(ctx context.Context) (DataFrame, error) {
	return fromFrame(l.collectFrame(ctx))
}

// collectFrame resolves the scan source and executes the plan, returning the
// unwrapped frame.
func (l *lf) collectFrame(ctx context.Context) (frame.DataFrame, error) {
	source, nodes, err := l.resolveSource()
	if err != nil {
		return frame.DataFrame{}, err
	}
	return l.engine.Execute(ctx, source, nodes)
}

func (l *lf) CollectStreaming(ctx context.Context, chunkSize int) (DataFrame, error) {
	source, nodes, err := l.resolveSource()
	if err != nil {
		return nil, err
	}
	return fromFrame(l.engine.ExecuteStreaming(ctx, source, nodes, chunkSize))
}

func (l *lf) SinkCSV(ctx context.Context, input WriteCSVInput) error {
	out, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return out.WriteCSV(input)
}

func (l *lf) SinkParquet(ctx context.Context, input WriteParquetInput) error {
	out, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return out.WriteParquet(input)
}

func (l *lf) SinkIPC(ctx context.Context, input WriteIPCInput) error {
	out, err := l.Collect(ctx)
	if err != nil {
		return err
	}
	return out.WriteIPC(input)
}

func (l *lf) SinkDelta(ctx context.Context, path string) error {
	return fmt.Errorf("not supported: sink_delta %s", path)
}

func (l *lf) SinkIceberg(ctx context.Context, path string) error {
	return fmt.Errorf("not supported: sink_iceberg %s", path)
}

func (l *lf) Explain(optimized bool) string {
	logicalStage := renderStage(l.nodes)
	if !optimized {
		return "logical=[" + logicalStage + "]"
	}
	optimizedStage := renderStage(optimizer.Optimize(l.nodes))
	return "logical=[" + logicalStage + "] optimized=[" + optimizedStage + "] physical=[" + optimizedStage + "]"
}

func (l *lf) ExplainDiagnostics(optimized bool) map[string]any {
	optimizedNodes := l.nodes
	if optimized {
		optimizedNodes = optimizer.Optimize(l.nodes)
	}
	var stateful, windowNodes, window, reshape, setOps, temporal int
	for _, n := range optimizedNodes {
		switch n.Type {
		case logical.NodeSort, logical.NodeJoin, logical.NodeAggregate, logical.NodeWindow, logical.NodePivot, logical.NodeSetOp, logical.NodeRolling, logical.NodeDynamic:
			stateful++
		}
		switch n.Type {
		case logical.NodeWindow:
			window += len(n.Windows)
			windowNodes++
		case logical.NodeMelt, logical.NodePivot:
			reshape++
		case logical.NodeSetOp:
			setOps++
		case logical.NodeRolling, logical.NodeDynamic:
			temporal++
		}
	}
	diag := map[string]any{
		"schema_version":             "v2",
		"logical_nodes":              len(l.nodes),
		"scan_source":                "in_memory",
		"optimized":                  optimized,
		"optimized_nodes":            len(optimizedNodes),
		"stateful_pipeline":          stateful > 0,
		"window_expressions":         window,
		"reshape_operations":         reshape,
		"set_operations":             setOps,
		"temporal_window_operations": temporal,
		"performance_markers": map[string]any{
			"stateful_nodes":        stateful,
			"window_nodes":          windowNodes,
			"temporal_window_nodes": temporal,
		},
		"plan": renderStage(optimizedNodes),
	}
	if l.scan != nil {
		diag["scan_source"] = l.scan.format
		diag["scan_path"] = l.scan.path
	}
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
	parq   ScanParquetInput
}

func (l *lf) resolveSource() (frame.DataFrame, []logical.Node, error) {
	if l.scan == nil {
		return l.source, l.nodes, nil
	}
	columns := projectedColumns(l.nodes)
	pushed, remaining := splitPushdownFilters(optimizer.PredicatePushdown(l.nodes))
	path, err := resolveObjectStorePath(l.scan.path)
	if err != nil {
		return frame.DataFrame{}, nil, err
	}
	var base frame.DataFrame
	switch l.scan.format {
	case "csv":
		c := l.scan.csv
		base, err = icsv.Read(icsv.ReadInput{
			Path:      path,
			HasHeader: c.HasHeader,
			Separator: c.Separator,
			Schema:    c.Schema,
			Columns:   columns,
		})
	case "json":
		j := l.scan.json
		base, err = ijson.Read(ijson.ReadInput{
			Path:    path,
			NDJSON:  j.NDJSON,
			Schema:  j.Schema,
			Columns: columns,
		})
	case "ipc":
		base, err = iipc.Read(iipc.ReadInput{Path: path, Columns: columns})
	case "parquet":
		p := l.scan.parq
		if len(columns) > 0 {
			p.Columns = columns
		}
		base, err = readParquetSource(path, p.Columns, pushed)
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
		part, err := iparquet.Read(iparquet.ReadInput{Path: f, Columns: readCols})
		if err != nil {
			return frame.DataFrame{}, err
		}
		if len(meta) > 0 {
			part, err = addPartitionColumns(part, meta)
			if err != nil {
				return frame.DataFrame{}, err
			}
		}
		if len(columns) > 0 {
			exprs := make([]expr.Expr, 0, len(columns))
			for _, c := range columns {
				if _, ok := meta[c]; ok || hasColumn(part, c) {
					exprs = append(exprs, expr.Col(c))
				}
			}
			if len(exprs) > 0 {
				part, err = part.Select(exprs...)
				if err != nil {
					return frame.DataFrame{}, err
				}
			}
		}
		parts = append(parts, part)
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
	for part := range strings.SplitSeq(rel, string(os.PathSeparator)) {
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

func addPartitionColumns(f frame.DataFrame, meta map[string]string) (frame.DataFrame, error) {
	existing := map[string]struct{}{}
	for _, name := range f.Columns() {
		existing[name] = struct{}{}
	}
	out := make([]series.Series, 0, f.Width()+len(meta))
	for _, name := range f.Columns() {
		s, _ := f.Series(name)
		out = append(out, s.Clone())
	}
	for key, value := range meta {
		if _, ok := existing[key]; ok {
			continue
		}
		values := make([]any, f.Height())
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

func hasColumn(f frame.DataFrame, column string) bool {
	return slices.Contains(f.Columns(), column)
}

// projectedColumns returns the source columns a scan must read for nodes, or
// nil to read every column. It prunes only when the plan reaches a projecting
// barrier, the first Select or Aggregate, through nodes whose column use is
// known; the barrier decides which columns survive, so nodes after it read
// nothing from the source. Names created before the barrier by WithColumns or
// WithRowIndex are not requested from the source.
func projectedColumns(nodes []logical.Node) []string {
	var required []string
	defined := map[string]struct{}{}
	request := func(names ...string) {
		for _, name := range names {
			if _, ok := defined[name]; !ok && !slices.Contains(required, name) {
				required = append(required, name)
			}
		}
	}
	requestInputs := func(exprs []expr.Expr) bool {
		for _, e := range exprs {
			cols, ok := expr.InputColumns(e)
			if !ok {
				return false
			}
			request(cols...)
		}
		return true
	}
	for _, n := range nodes {
		switch n.Type {
		case logical.NodeSelect, logical.NodeAggregate:
			request(n.Columns...)
			if !requestInputs(n.Exprs) || len(required) == 0 {
				return nil
			}
			return required
		case logical.NodeFilter:
			if !requestInputs(n.Exprs) {
				return nil
			}
		case logical.NodeWithCols:
			if !requestInputs(n.Exprs) {
				return nil
			}
			for _, e := range n.Exprs {
				defined[e.Name()] = struct{}{}
			}
		case logical.NodeWithRowIdx:
			if len(n.Strings) > 0 {
				defined[n.Strings[0]] = struct{}{}
			}
		case logical.NodeSort, logical.NodeDrop, logical.NodeSetSorted, logical.NodeExplode:
			request(n.Columns...)
		case logical.NodeCast:
			for i := 0; i+1 < len(n.Strings); i += 2 {
				request(n.Strings[i])
			}
		case logical.NodeDropNulls, logical.NodeDropNans, logical.NodeUnique:
			// Without columns these read every column to decide which rows stay.
			if len(n.Columns) == 0 {
				return nil
			}
			request(n.Columns...)
		case logical.NodeLimit, logical.NodeTail, logical.NodeSlice, logical.NodeReverse,
			logical.NodeGatherEvery, logical.NodeShift, logical.NodeFillNull, logical.NodeFillNaN:
		default:
			return nil
		}
	}
	return nil
}

// splitPushdownFilters hoists the leading run of simple column-versus-literal
// filters, which the scan applies right after reading, and returns the rest of
// the plan. A filter behind any other node stays in the plan: that node may
// change which rows exist or the values the filter reads. Callers run
// optimizer.PredicatePushdown first so that filters are as early as is safe.
func splitPushdownFilters(nodes []logical.Node) ([]expr.Expr, []logical.Node) {
	var pushed []expr.Expr
	for len(nodes) > 0 && nodes[0].Type == logical.NodeFilter && len(nodes[0].Exprs) > 0 && isPushdownFilter(nodes[0].Exprs[0]) {
		pushed = append(pushed, nodes[0].Exprs[0])
		nodes = nodes[1:]
	}
	return pushed, nodes
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

func renderStage(nodes []logical.Node) string {
	out := make([]string, 0, len(nodes)+1)
	out = append(out, "scan")
	for _, n := range nodes {
		out = append(out, string(n.Type))
	}
	return strings.Join(out, " -> ")
}
