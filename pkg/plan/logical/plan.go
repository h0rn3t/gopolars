package logical

import (
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

type NodeType string

const (
	NodeScan        NodeType = "scan"
	NodeSelect      NodeType = "select"
	NodeFilter      NodeType = "filter"
	NodeWithCols    NodeType = "with_columns"
	NodeJoin        NodeType = "join"
	NodeGroupBy     NodeType = "group_by"
	NodeSort        NodeType = "sort"
	NodeLimit       NodeType = "limit"
	NodeAggregate   NodeType = "aggregate"
	NodeSlice       NodeType = "slice"
	NodeUnique      NodeType = "unique"
	NodeFillNull    NodeType = "fill_null"
	NodeDropNulls   NodeType = "drop_nulls"
	NodeDropNans    NodeType = "drop_nans"
	NodeDrop        NodeType = "drop"
	NodeTail        NodeType = "tail"
	NodeGatherEvery NodeType = "gather_every"
	NodeReverse     NodeType = "reverse"
	NodeRename      NodeType = "rename"
	NodeWindow      NodeType = "window"
	NodeExplode     NodeType = "explode"
	NodeFlatten     NodeType = "flatten"
	NodeMelt        NodeType = "melt"
	NodeUnpivot     NodeType = "unpivot"
	NodeUnnest      NodeType = "unnest"
	NodeUpdate      NodeType = "update"
	NodeSetSorted   NodeType = "set_sorted"
	NodeShift       NodeType = "shift"
	NodeCast        NodeType = "cast"
	NodeFillNaN     NodeType = "fill_nan"
	NodeInterpolate NodeType = "interpolate"
	NodeWithRowIdx  NodeType = "with_row_index"
	NodeFrameAgg    NodeType = "frame_agg"
	NodePivot       NodeType = "pivot"
	NodeSetOp       NodeType = "set_op"
	NodeRolling     NodeType = "rolling_mean"
	NodeDynamic     NodeType = "group_by_dynamic"
)

type Node struct {
	Type       NodeType
	Exprs      []expr.Expr
	Columns    []string
	IntValue   int
	Strings    []string
	Descending []bool
	Join       *JoinSpec
	Windows    []WindowSpec
	Prefix     string
	Plan       []Node
}

type JoinSpec struct {
	Other         frame.DataFrame
	LeftOn        []string
	RightOn       []string
	How           frame.JoinType
	Suffix        string
	AsofDirection string
	AsofTolerance int64
}

type WindowSpec struct {
	Func        string
	Target      string
	Alias       string
	PartitionBy []string
	OrderBy     []string
	Descending  []bool
}
