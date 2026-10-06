package polars

import (
	"context"
	"maps"
	"math"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func charFrame(t *testing.T, cols ...frame.SeriesInput) DataFrame {
	t.Helper()
	d, err := NewDataFrame(NewDataFrameInput{Columns: cols})
	if err != nil {
		t.Fatalf("NewDataFrame: %v", err)
	}
	return d
}

func TestProjectedColumnsCharacterization(t *testing.T) {
	when := expr.When(expr.Col("c").Gt(expr.Lit(1)), expr.Col("d").Abs(), expr.Col("e"))
	tests := []struct {
		name  string
		nodes []logical.Node
		want  []string
	}{
		{"no projection", []logical.Node{{Type: logical.NodeLimit, IntValue: 1}}, nil},
		{"select", []logical.Node{{Type: logical.NodeSelect, Exprs: []expr.Expr{expr.Col("a"), expr.Lit(1), expr.Col("b").Alias("z")}}}, []string{"a", "b"}},
		{"filter without expr", []logical.Node{{Type: logical.NodeFilter}}, nil},
		{"filter expr tree", []logical.Node{{Type: logical.NodeFilter, Exprs: []expr.Expr{when}}}, []string{"c", "d", "e"}},
		{"column lists", []logical.Node{
			{Type: logical.NodeSort, Columns: []string{"s"}},
			{Type: logical.NodeAggregate, Columns: []string{"g"}},
			{Type: logical.NodeJoin, Columns: []string{"j"}},
			{Type: logical.NodeUnique, Columns: []string{"u"}},
			{Type: logical.NodeDropNulls, Columns: []string{"n"}},
			{Type: logical.NodeExplode, Columns: []string{"x"}},
			{Type: logical.NodeFlatten, Columns: []string{"f"}},
			{Type: logical.NodeMelt, Columns: []string{"m"}},
			{Type: logical.NodePivot, Columns: []string{"p", "s"}},
		}, []string{"f", "g", "j", "m", "n", "p", "s", "u", "x"}},
		{"rolling", []logical.Node{
			{Type: logical.NodeRolling, Columns: []string{"by", "", "out"}},
			{Type: logical.NodeRolling, Columns: []string{"v"}},
		}, []string{"by", "v"}},
		{"dynamic", []logical.Node{
			{Type: logical.NodeDynamic, Columns: []string{"", "w"}, Exprs: []expr.Expr{expr.Col("agg").Sum()}},
			{Type: logical.NodeDynamic, Columns: []string{"t"}},
		}, []string{"agg", "t"}},
		{"window", []logical.Node{{Type: logical.NodeWindow, Windows: []logical.WindowSpec{
			{Target: "*", PartitionBy: []string{"pb"}, OrderBy: []string{"ob"}},
			{Target: ""},
			{Target: "tg"},
		}}}, []string{"ob", "pb", "tg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectedColumns(tt.nodes)
			slices.Sort(got)
			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("projectedColumns() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExplainDiagnosticsCharacterization(t *testing.T) {
	scanned := &lf{
		nodes: []logical.Node{
			{Type: logical.NodeSort},
			{Type: logical.NodeWindow, Windows: make([]logical.WindowSpec, 2)},
			{Type: logical.NodeMelt},
			{Type: logical.NodePivot},
			{Type: logical.NodeSetOp},
			{Type: logical.NodeRolling},
			{Type: logical.NodeDynamic},
			{Type: logical.NodeLimit},
		},
		scan: &scanSource{format: "csv", path: "in.csv"},
	}
	want := map[string]any{
		"schema_version":             "v2",
		"logical_nodes":              8,
		"scan_source":                "csv",
		"scan_path":                  "in.csv",
		"optimized":                  false,
		"optimized_nodes":            8,
		"stateful_pipeline":          true,
		"window_expressions":         2,
		"reshape_operations":         2,
		"set_operations":             1,
		"temporal_window_operations": 2,
		"performance_markers": map[string]any{
			"stateful_nodes":        6,
			"window_nodes":          1,
			"temporal_window_nodes": 2,
		},
		"plan": "scan -> sort -> window -> melt -> pivot -> set_op -> rolling_mean -> group_by_dynamic -> limit",
	}
	if got := scanned.ExplainDiagnostics(false); !reflect.DeepEqual(got, want) {
		t.Errorf("ExplainDiagnostics(false) = %#v, want %#v", got, want)
	}

	wantEmpty := map[string]any{
		"schema_version":             "v2",
		"logical_nodes":              0,
		"scan_source":                "in_memory",
		"optimized":                  true,
		"optimized_nodes":            0,
		"stateful_pipeline":          false,
		"window_expressions":         0,
		"reshape_operations":         0,
		"set_operations":             0,
		"temporal_window_operations": 0,
		"performance_markers": map[string]any{
			"stateful_nodes":        0,
			"window_nodes":          0,
			"temporal_window_nodes": 0,
		},
		"plan": "scan",
	}
	if got := (&lf{}).ExplainDiagnostics(true); !reflect.DeepEqual(got, wantEmpty) {
		t.Errorf("ExplainDiagnostics(true) = %#v, want %#v", got, wantEmpty)
	}
	if got, want := scanned.Explain(false), "logical=["+want["plan"].(string)+"]"; got != want {
		t.Errorf("Explain(false) = %q, want %q", got, want)
	}
}

func TestProfileKeysCharacterization(t *testing.T) {
	ctx := context.Background()
	d := charFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3)}})

	out, profile, err := d.Lazy().Profile(ctx)
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	wantKeys := []string{"duration_ms", "memory_bytes", "operators", "output_rows", "schema_version", "source_rows", "temporal_ops"}
	if got := slices.Sorted(maps.Keys(profile)); !slices.Equal(got, wantKeys) {
		t.Errorf("Profile keys = %v, want %v", got, wantKeys)
	}
	if out.Height() != 3 || profile["output_rows"] != 3 || profile["source_rows"] != 3 || profile["schema_version"] != "v2" {
		t.Errorf("Profile out height %d, profile %v", out.Height(), profile)
	}
	if _, ok := profile["operators"].([]string); !ok {
		t.Errorf("operators type = %T, want []string", profile["operators"])
	}

	out, profile, err = d.Lazy().Select(Col("missing")).Profile(ctx)
	if err == nil || out != nil {
		t.Fatalf("Profile on missing column = (%v, %v), want error and nil frame", out, err)
	}
	wantKeys = []string{"duration_ms", "memory_bytes", "operators", "schema_version", "temporal_ops"}
	if got := slices.Sorted(maps.Keys(profile)); !slices.Equal(got, wantKeys) {
		t.Errorf("error Profile keys = %v, want %v", got, wantKeys)
	}
}

func TestCollectBatchesCharacterization(t *testing.T) {
	ctx := context.Background()
	d := charFrame(t, frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}})
	for _, tt := range []struct {
		chunk int
		want  []int
	}{{2, []int{2, 2, 1}}, {0, []int{5}}, {5, []int{5}}, {7, []int{5}}} {
		var heights []int
		for res := range d.Lazy().CollectBatches(ctx, tt.chunk) {
			if res.Error != nil {
				t.Fatalf("CollectBatches(%d): %v", tt.chunk, res.Error)
			}
			heights = append(heights, res.DataFrame.Height())
		}
		if !slices.Equal(heights, tt.want) {
			t.Errorf("CollectBatches(%d) heights = %v, want %v", tt.chunk, heights, tt.want)
		}
	}
	var results []AsyncCollectResult
	for res := range d.Lazy().Select(Col("missing")).CollectBatches(ctx, 2) {
		results = append(results, res)
	}
	if len(results) != 1 || results[0].Error == nil || results[0].DataFrame != nil {
		t.Errorf("CollectBatches error results = %#v, want one error result", results)
	}

	var sliceHeights []int
	for _, part := range d.IterSlices(2) {
		sliceHeights = append(sliceHeights, part.Height())
	}
	if !slices.Equal(sliceHeights, []int{2, 2, 1}) {
		t.Errorf("IterSlices(2) heights = %v, want [2 2 1]", sliceHeights)
	}
}

func TestScanJSONCollectCharacterization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.ndjson")
	d := charFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "b", Values: []any{"x", "y"}},
	)
	if err := d.WriteJSON(WriteJSONInput{Path: path, NDJSON: true}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	scan, err := NewIO().ScanJSON(ScanJSONInput{Path: path, NDJSON: true})
	if err != nil {
		t.Fatalf("ScanJSON: %v", err)
	}
	out, err := scan.Select(Col("b")).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	want := map[string][]any{"b": {"x", "y"}}
	if got := out.ToDict(); !reflect.DeepEqual(got, want) {
		t.Errorf("ScanJSON Select(b) = %v, want %v", got, want)
	}
}

func TestTransposeSupertypeCharacterization(t *testing.T) {
	mixed := charFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "b", Values: []any{"x", nil}},
	)
	out, err := mixed.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	want := map[string][]any{"column_0": {"1", "x"}, "column_1": {"2", nil}}
	if got := out.ToDict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Transpose(mixed) = %v, want %v", got, want)
	}

	numeric := charFrame(t,
		frame.SeriesInput{Name: "a", Values: []any{int64(1), int64(2)}},
		frame.SeriesInput{Name: "f", Values: []any{1.5, 2.5}},
	)
	out, err = numeric.Transpose()
	if err != nil {
		t.Fatalf("Transpose: %v", err)
	}
	want = map[string][]any{"column_0": {1.0, 1.5}, "column_1": {2.0, 2.5}}
	if got := out.ToDict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Transpose(numeric) = %v, want %v", got, want)
	}
}

func TestToDummiesAllColumnsCharacterization(t *testing.T) {
	d := charFrame(t,
		frame.SeriesInput{Name: "k", Values: []any{"a b", "c"}},
		frame.SeriesInput{Name: "n", Values: []any{int64(1), int64(1)}},
	)
	out, err := d.ToDummies()
	if err != nil {
		t.Fatalf("ToDummies: %v", err)
	}
	if got, want := out.Columns(), []string{"k_a_b", "k_c", "n_1"}; !slices.Equal(got, want) {
		t.Errorf("ToDummies() columns = %v, want %v", got, want)
	}
	want := map[string][]any{"k_a_b": {true, false}, "k_c": {false, true}, "n_1": {true, true}}
	if got := out.ToDict(); !reflect.DeepEqual(got, want) {
		t.Errorf("ToDummies() = %v, want %v", got, want)
	}
}

func TestHorizontalAggCharacterization(t *testing.T) {
	d := charFrame(t,
		frame.SeriesInput{Name: "a", DType: dtypes.Int64, Values: []any{int64(1), nil, int64(4)}},
		frame.SeriesInput{Name: "b", DType: dtypes.Float64, Values: []any{2.5, nil, math.NaN()}},
	)
	tests := []struct {
		name string
		agg  func(string) (DataFrame, error)
		want []any
	}{
		{"max_horizontal", d.MaxHorizontal, []any{2.5, nil, 4.0}},
		{"min_horizontal", d.MinHorizontal, []any{1.0, nil, 4.0}},
		{"sum_horizontal", d.SumHorizontal, []any{3.5, nil, 4.0}},
		{"mean_horizontal", d.MeanHorizontal, []any{1.75, nil, 4.0}},
	}
	for _, tt := range tests {
		out, err := tt.agg("")
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := out.ToDict()[tt.name]; !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got, want := d.ToJax(), [][]float64{{1, 2.5}, {0, 0}, {4, 0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("ToJax = %v, want %v", got, want)
	}
}

func TestFrameReductionsCharacterization(t *testing.T) {
	d := charFrame(t,
		frame.SeriesInput{Name: "n", DType: dtypes.Int64, Values: []any{int64(3), nil, int64(1), int64(3), int64(2)}},
		frame.SeriesInput{Name: "s", DType: dtypes.String, Values: []any{"b", "a", nil, "c", "b"}},
		frame.SeriesInput{Name: "f", DType: dtypes.Float64, Values: []any{1.5, math.NaN(), 2.5, nil, 4.0}},
		frame.SeriesInput{Name: "z", DType: dtypes.String, Values: []any{nil, nil, nil, nil, nil}},
	)
	// A NaN cell is not numeric to compareAnyLocal, so it is ordered by its "NaN"
	// text and wins Max over the numeric cells.
	gotMax := d.Max()
	if f, ok := gotMax["f"].(float64); !ok || !math.IsNaN(f) {
		t.Errorf("Max[f] = %#v, want NaN", gotMax["f"])
	}
	delete(gotMax, "f")
	if want := map[string]any{"n": int64(3), "s": "c", "z": nil}; !reflect.DeepEqual(gotMax, want) {
		t.Errorf("Max = %#v, want %#v", gotMax, want)
	}
	if got, want := d.Min(), map[string]any{"n": int64(1), "s": "a", "f": 1.5, "z": nil}; !reflect.DeepEqual(got, want) {
		t.Errorf("Min = %#v, want %#v", got, want)
	}
	if got, want := d.Median(), map[string]float64{"n": 2.5, "f": 2.5}; !reflect.DeepEqual(got, want) {
		t.Errorf("Median = %v, want %v", got, want)
	}
	odd := charFrame(t, frame.SeriesInput{Name: "f", Values: []any{3.0, 1.0, 2.0}})
	if got, want := odd.Median(), map[string]float64{"f": 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("Median(odd) = %v, want %v", got, want)
	}
	if got, want := d.Quantile(0.5), map[string]float64{"n": 3, "f": 2.5}; !reflect.DeepEqual(got, want) {
		t.Errorf("Quantile(0.5) = %v, want %v", got, want)
	}
	if got, want := d.Std(), map[string]float64{"n": 0.9574271077563381, "f": 1.2583057392117916}; !reflect.DeepEqual(got, want) {
		t.Errorf("Std = %v, want %v", got, want)
	}
	if got, want := odd.Var(), map[string]float64{"f": 1.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("Var(odd) = %v, want %v", got, want)
	}
	if got, want := odd.Reverse().ToDict(), map[string][]any{"f": {2.0, 1.0, 3.0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Reverse = %v, want %v", got, want)
	}
}
