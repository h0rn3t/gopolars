package exec

import (
	"context"
	"runtime"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
	"github.com/eugeneshershen/gopolars/pkg/plan/optimizer"
)

type ExecutionReport struct {
	SchemaVersion    string         `json:"schema_version"`
	Operators        []string       `json:"operators"`
	OptimizedNodes   int            `json:"optimized_nodes"`
	StatefulPipeline bool           `json:"stateful_pipeline"`
	DurationMS       int64          `json:"duration_ms"`
	MemoryBytes      uint64         `json:"memory_bytes"`
	TemporalOps      int            `json:"temporal_ops"`
	SourceRows       int            `json:"source_rows"`
	OutputRows       int            `json:"output_rows"`
	Metadata         map[string]any `json:"metadata"`
}

func (e Engine) ExecuteWithReport(ctx context.Context, source frame.DataFrame, nodes []logical.Node) (frame.DataFrame, ExecutionReport, error) {
	start := time.Now()
	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)
	optimized := optimizer.Optimize(nodes)
	out, err := executeOptimized(source, optimized)
	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)
	temporal := 0
	for _, n := range optimized {
		if n.Type == logical.NodeRolling || n.Type == logical.NodeDynamic {
			temporal++
		}
	}
	report := ExecutionReport{
		SchemaVersion:    "v2",
		Operators:        make([]string, 0, len(optimized)),
		OptimizedNodes:   len(optimized),
		StatefulPipeline: hasStatefulNode(optimized),
		DurationMS:       time.Since(start).Milliseconds(),
		MemoryBytes:      memEnd.TotalAlloc - memStart.TotalAlloc,
		TemporalOps:      temporal,
		SourceRows:       source.Height(),
		Metadata:         map[string]any{},
	}
	if err == nil {
		report.OutputRows = out.Height()
	}
	for _, n := range optimized {
		report.Operators = append(report.Operators, string(n.Type))
	}
	return out, report, err
}
