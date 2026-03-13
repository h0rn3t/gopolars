package polars

import (
	"context"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
)

type DataFrame interface {
	Schema() dtypes.Schema
	CollectSchema() dtypes.Schema
	Height() int
	IsEmpty() bool
	EstimatedSize() int64
	Width() int
	Columns() []string
	Dtypes() []dtypes.DataType
	ToDicts() []map[string]any
	GetColumn(name string) (Series, error)
	GetColumnIndex(name string) int
	GetColumns() []Series
	NullCount() map[string]int
	Count() map[string]int
	NUnique(columns ...string) (int, error)
	ApproxNUnique(columns ...string) (int, error)
	Series(name string) (Series, bool)
	Flags() map[string]map[string]bool
	Glimpse(maxRows int) string
	Select(exprs ...Expr) (DataFrame, error)
	Filter(predicate Expr) (DataFrame, error)
	WithColumns(exprs ...Expr) (DataFrame, error)
	WithRowCount(name string, offset int64) (DataFrame, error)
	WithRowIndex(name string, offset int64) (DataFrame, error)
	GroupBy(keys ...string) GroupBy
	GroupByDynamic(input DynamicGroupInput) (DataFrame, error)
	Join(input JoinInput) (DataFrame, error)
	Sort(input SortInput) (DataFrame, error)
	Concat(input ConcatInput) (DataFrame, error)
	Sample(n int, seed int64) DataFrame
	Limit(n int) DataFrame
	Slice(offset int, length int) DataFrame
	GatherEvery(step int, offset int) DataFrame
	Head(n int) DataFrame
	Tail(n int) DataFrame
	BottomK(k int, by string) (DataFrame, error)
	Unique(columns ...string) (DataFrame, error)
	Cast(mapping map[string]dtypes.DataType) (DataFrame, error)
	Clear() DataFrame
	Clone() DataFrame
	DropInPlace(column string) (DataFrame, error)
	Equals(other DataFrame) (bool, error)
	Extend(other DataFrame) (DataFrame, error)
	Hstack(columns ...Series) (DataFrame, error)
	InsertColumn(index int, column Series) (DataFrame, error)
	FillNull(value any) (DataFrame, error)
	FillNaN(value float64) (DataFrame, error)
	Interpolate(columns ...string) (DataFrame, error)
	DropNaNs(columns ...string) DataFrame
	DropNulls(columns ...string) DataFrame
	Fold(op string, columns []string, alias string) (DataFrame, error)
	HashRows(seed uint64) ([]uint64, error)
	Corr(columnA string, columnB string) (float64, error)
	Describe() (DataFrame, error)
	Explode(columns ...string) (DataFrame, error)
	FlattenStruct(column string, prefix string) (DataFrame, error)
	Melt(input MeltInput) (DataFrame, error)
	Pivot(input PivotInput) (DataFrame, error)
	RollingMean(input RollingMeanInput) (DataFrame, error)
	Deserialize(payload []byte) (DataFrame, error)
	Drop(columns ...string) (DataFrame, error)
	Rename(mapping map[string]string) (DataFrame, error)
	Lazy() LazyFrame
	WriteCSV(input WriteCSVInput) error
	WriteJSON(input WriteJSONInput) error
	WriteParquet(input WriteParquetInput) error
	WriteIPC(input WriteIPCInput) error
	ToArrow(input ToArrowInput) (iarrow.Table, error)
}

type LazyFrame interface {
	Select(exprs ...Expr) LazyFrame
	Filter(predicate Expr) LazyFrame
	WithColumns(exprs ...Expr) LazyFrame
	GroupBy(keys ...string) LazyGroupBy
	GroupByDynamic(input DynamicGroupInput) LazyFrame
	Join(input JoinInput) LazyFrame
	Sort(input SortInput) LazyFrame
	Limit(n int) LazyFrame
	Slice(offset int, length int) LazyFrame
	Unique(columns ...string) LazyFrame
	FillNull(value any) LazyFrame
	DropNulls(columns ...string) LazyFrame
	Explode(columns ...string) LazyFrame
	FlattenStruct(column string, prefix string) LazyFrame
	Melt(input MeltInput) LazyFrame
	Pivot(input PivotInput) LazyFrame
	RollingMean(input RollingMeanInput) LazyFrame
	CollectAsync(ctx context.Context) <-chan AsyncCollectResult
	CollectBatches(ctx context.Context, chunkSize int) <-chan AsyncCollectResult
	Inspect() LazyFrame
	Profile(ctx context.Context) (DataFrame, map[string]any, error)
	JoinWhere(predicate Expr) LazyFrame
	SinkNDJSON(ctx context.Context, input WriteJSONInput) error
	SQL(ctx context.Context, query string, table string) (LazyFrame, error)
	Collect(ctx context.Context) (DataFrame, error)
	CollectStreaming(ctx context.Context, chunkSize int) (DataFrame, error)
	SinkCSV(ctx context.Context, input WriteCSVInput) error
	SinkParquet(ctx context.Context, input WriteParquetInput) error
	SinkIPC(ctx context.Context, input WriteIPCInput) error
	Explain(optimized bool) string
	ExplainDiagnostics(optimized bool) map[string]any
}

type Series interface {
	Name() string
	DataType() dtypes.DataType
	Len() int
	Value(i int) any
	IsNull() Series
	IsNotNull() Series
	FillNull(value any) (Series, error)
	DropNulls() Series
	Cast(dt dtypes.DataType) (Series, error)
	Add(other Series) (Series, error)
	Sub(other Series) (Series, error)
	Mul(other Series) (Series, error)
	Div(other Series) (Series, error)
	Eq(other Series) (Series, error)
	Ne(other Series) (Series, error)
	Gt(other Series) (Series, error)
	Ge(other Series) (Series, error)
	Lt(other Series) (Series, error)
	Le(other Series) (Series, error)
}

type GroupBy interface {
	Agg(exprs ...Expr) (DataFrame, error)
}

type LazyGroupBy interface {
	Agg(exprs ...Expr) LazyFrame
}

type AsyncCollectResult struct {
	DataFrame DataFrame
	Error     error
}

type SQLContext interface {
	Register(name string, df DataFrame) error
	Execute(ctx context.Context, query string) (LazyFrame, error)
	Tables() []string
	Unregister(name string)
}
