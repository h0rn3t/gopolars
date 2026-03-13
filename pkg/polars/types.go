package polars

import (
	"context"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
)

type DataFrame interface {
	Schema() dtypes.Schema
	Height() int
	Width() int
	Columns() []string
	Series(name string) (Series, bool)
	Select(exprs ...Expr) (DataFrame, error)
	Filter(predicate Expr) (DataFrame, error)
	WithColumns(exprs ...Expr) (DataFrame, error)
	GroupBy(keys ...string) GroupBy
	GroupByDynamic(input DynamicGroupInput) (DataFrame, error)
	Join(input JoinInput) (DataFrame, error)
	Sort(input SortInput) (DataFrame, error)
	Concat(input ConcatInput) (DataFrame, error)
	Limit(n int) DataFrame
	Slice(offset int, length int) DataFrame
	Head(n int) DataFrame
	Tail(n int) DataFrame
	Unique(columns ...string) (DataFrame, error)
	FillNull(value any) (DataFrame, error)
	DropNulls(columns ...string) DataFrame
	Explode(columns ...string) (DataFrame, error)
	FlattenStruct(column string, prefix string) (DataFrame, error)
	Melt(input MeltInput) (DataFrame, error)
	Pivot(input PivotInput) (DataFrame, error)
	RollingMean(input RollingMeanInput) (DataFrame, error)
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
