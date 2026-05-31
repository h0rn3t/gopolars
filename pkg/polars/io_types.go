package polars

import "github.com/h0rn3t/gopolars/pkg/dtypes"

type JoinType string

const (
	JoinTypeInner JoinType = "inner"
	JoinTypeLeft  JoinType = "left"
	JoinTypeRight JoinType = "right"
	JoinTypeFull  JoinType = "full"
	JoinTypeSemi  JoinType = "semi"
	JoinTypeAnti  JoinType = "anti"
	JoinTypeCross JoinType = "cross"
	JoinTypeAsof  JoinType = "asof"
)

type JoinInput struct {
	Other         DataFrame
	LeftOn        []string
	RightOn       []string
	How           JoinType
	Suffix        string
	AsofDirection string
	AsofTolerance int64
}

type SortInput struct {
	By            []string
	Descending    []bool
	NullsLast     bool
	MaintainOrder bool
}

type ConcatInput struct {
	Others []DataFrame
	How    string
}

type ScanCSVInput struct {
	Path              string
	HasHeader         bool
	Separator         rune
	Schema            dtypes.Schema
	InferSchemaLength int
	IgnoreErrors      bool
}

type ScanParquetInput struct {
	Path    string
	Columns []string
}

type ScanIPCInput struct {
	Path string
}

type ScanJSONInput struct {
	Path   string
	NDJSON bool
	Schema dtypes.Schema
}

type ReadCSVInput struct {
	Path      string
	HasHeader bool
	Separator rune
	Schema    dtypes.Schema
}

type ReadParquetInput struct {
	Path    string
	Columns []string
}

type ReadIPCInput struct {
	Path string
}

type ReadJSONInput struct {
	Path   string
	NDJSON bool
	Schema dtypes.Schema
}

type WriteCSVInput struct {
	Path          string
	IncludeHeader bool
	Separator     rune
}

type WriteJSONInput struct {
	Path   string
	Pretty bool
	NDJSON bool
}

type WriteParquetInput struct {
	Path         string
	Compression  string
	RowGroupSize int
}

type WriteIPCInput struct {
	Path string
}

type ToArrowInput struct {
	ChunkSize int
}
