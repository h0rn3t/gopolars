package polars

import (
	"context"
	"time"

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
	IterColumns() []Series
	IterRows() []map[string]any
	IterSlices(size int) []DataFrame
	Dtypes() []dtypes.DataType
	ToDicts() []map[string]any
	Item(row int, column string) (any, error)
	GetColumn(name string) (Series, error)
	GetColumnIndex(name string) int
	GetColumns() []Series
	NullCount() map[string]int
	Count() map[string]int
	NUnique(columns ...string) (int, error)
	ApproxNUnique(columns ...string) (int, error)
	Series(name string) (Series, bool)
	IsDuplicated() Series
	IsUnique() Series
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
	JoinAsof(input JoinInput) (DataFrame, error)
	JoinWhere(predicate Expr) (DataFrame, error)
	Sort(input SortInput) (DataFrame, error)
	Concat(input ConcatInput) (DataFrame, error)
	Sample(n int, seed int64) DataFrame
	Limit(n int) DataFrame
	Slice(offset int, length int) DataFrame
	GatherEvery(step int, offset int) DataFrame
	Rechunk() DataFrame
	ToNumpy() [][]any
	ToPandas() []map[string]any
	PartitionBy(columns ...string) ([]DataFrame, error)
	MatchToSchema(schema dtypes.Schema) (DataFrame, error)
	MergeSorted(other DataFrame, by string) (DataFrame, error)
	SelectSeq(exprs ...Expr) (DataFrame, error)
	ReplaceColumn(index int, column Series) (DataFrame, error)
	Reverse() DataFrame
	Rolling(by string, value string, window time.Duration, output string) (DataFrame, error)
	Row(index int) (map[string]any, error)
	Rows() []map[string]any
	RowsByKey(column string) map[any][]map[string]any
	NChunks() int
	Pipe(fn func(DataFrame) (DataFrame, error)) (DataFrame, error)
	MapColumns(fn func(name string, s Series) (Series, error)) (DataFrame, error)
	MapRows(fn func(row map[string]any) (map[string]any, error)) (DataFrame, error)
	Plot(x string, y string) map[string]any
	Serialize() ([]byte, error)
	SetSorted(by string) (DataFrame, error)
	Shape() [2]int
	Shift(periods int) (DataFrame, error)
	Show(maxRows int) string
	ShrinkToFit() DataFrame
	SQL(ctx context.Context, query string) (LazyFrame, error)
	Sql(ctx context.Context, query string) (LazyFrame, error)
	Style() string
	Upsample(by string, every time.Duration) (DataFrame, error)
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
	Max() map[string]any
	Min() map[string]any
	Mean() map[string]float64
	Median() map[string]float64
	Product() map[string]float64
	Quantile(q float64) map[string]float64
	Std() map[string]float64
	Var() map[string]float64
	Sum() map[string]float64
	MaxHorizontal(alias string) (DataFrame, error)
	MinHorizontal(alias string) (DataFrame, error)
	MeanHorizontal(alias string) (DataFrame, error)
	SumHorizontal(alias string) (DataFrame, error)
	Remove(column string) (DataFrame, error)
	Explode(columns ...string) (DataFrame, error)
	FlattenStruct(column string, prefix string) (DataFrame, error)
	Melt(input MeltInput) (DataFrame, error)
	Unpivot(input MeltInput) (DataFrame, error)
	Unstack(by string) ([]DataFrame, error)
	Unnest(columns ...string) (DataFrame, error)
	Pivot(input PivotInput) (DataFrame, error)
	Transpose() (DataFrame, error)
	ToSeries(column string) (Series, error)
	ToStruct() map[string]any
	ToTorch() [][]float64
	TopK(k int, by string) (DataFrame, error)
	Vstack(other DataFrame) (DataFrame, error)
	VStack(other DataFrame) (DataFrame, error)
	Update(other DataFrame) (DataFrame, error)
	WithColumnsSeq(exprs ...Expr) (DataFrame, error)
	RollingMean(input RollingMeanInput) (DataFrame, error)
	Deserialize(payload []byte) (DataFrame, error)
	Drop(columns ...string) (DataFrame, error)
	Rename(mapping map[string]string) (DataFrame, error)
	Lazy() LazyFrame
	WriteCSV(input WriteCSVInput) error
	WriteCsv(input WriteCSVInput) error
	WriteJSON(input WriteJSONInput) error
	WriteJson(input WriteJSONInput) error
	WriteParquet(input WriteParquetInput) error
	WriteIPC(input WriteIPCInput) error
	WriteIpc(input WriteIPCInput) error
	WriteIpcStream(input WriteIPCInput) error
	WriteNDJSON(input WriteJSONInput) error
	WriteNdjson(input WriteJSONInput) error
	WriteAvro(path string) error
	WriteClipboard() error
	WriteDatabase(target string) error
	WriteDelta(path string) error
	WriteExcel(path string) error
	WriteIceberg(path string) error
	ToArrow(input ToArrowInput) (iarrow.Table, error)
	ToDict() map[string][]any
	ToDummies(columns ...string) (DataFrame, error)
	ToInitRepr() string
	ToJax() [][]float64
}

type LazyFrame interface {
	Select(exprs ...Expr) LazyFrame
	Filter(predicate Expr) LazyFrame
	WithColumns(exprs ...Expr) LazyFrame
	GroupBy(keys ...string) LazyGroupBy
	GroupByDynamic(input DynamicGroupInput) LazyFrame
	Join(input JoinInput) LazyFrame
	Sort(input SortInput) LazyFrame
	ApproxNUnique(columns ...string) LazyFrame
	BottomK(k int, by string) LazyFrame
	Limit(n int) LazyFrame
	Head(n int) LazyFrame
	Tail(n int) LazyFrame
	First() LazyFrame
	Last() LazyFrame
	Slice(offset int, length int) LazyFrame
	GatherEvery(step int, offset int) LazyFrame
	Clear() LazyFrame
	Clone() LazyFrame
	Reverse() LazyFrame
	Rename(mapping map[string]string) LazyFrame
	Unique(columns ...string) LazyFrame
	FillNull(value any) LazyFrame
	DropNulls(columns ...string) LazyFrame
	DropNaNs(columns ...string) LazyFrame
	Drop(columns ...string) LazyFrame
	WithRowIndex(name string, offset int64) LazyFrame
	WithRowCount(name string, offset int64) LazyFrame
	Shift(periods int) LazyFrame
	SetSorted(by string) LazyFrame
	Cast(mapping map[string]dtypes.DataType) LazyFrame
	FillNaN(value float64) LazyFrame
	Interpolate(columns ...string) LazyFrame
	Unnest(columns ...string) LazyFrame
	Unpivot(input MeltInput) LazyFrame
	Update(other LazyFrame) LazyFrame
	Explode(columns ...string) LazyFrame
	FlattenStruct(column string, prefix string) LazyFrame
	Melt(input MeltInput) LazyFrame
	Pivot(input PivotInput) LazyFrame
	RollingMean(input RollingMeanInput) LazyFrame
	Cache() LazyFrame
	ShowGraph() string
	Serialize() ([]byte, error)
	Deserialize(payload []byte) (LazyFrame, error)
	Remote(endpoint string) LazyFrame
	SinkBatches(ctx context.Context, chunkSize int) <-chan AsyncCollectResult
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
	ToList() []any
	NullCount() int
	IsNull() Series
	IsNotNull() Series
	FillNull(value any) (Series, error)
	FillNan(value float64) (Series, error)
	DropNans() Series
	DropNulls() Series
	RollingMean(window int) Series
	RollingSum(window int) Series
	RollingMin(window int) Series
	RollingMax(window int) Series
	Abs() Series
	Exp() Series
	Log() Series
	Sqrt() Series
	Shift(periods int) Series
	Reverse() Series
	Sum() float64
	Std() float64
	Describe() map[string]any
	Hist(bins int) (DataFrame, error)
	Interpolate() Series
	ToNumpy() []any
	ToPandas() []any
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
	RegisterMany(tables map[string]DataFrame) error
	Execute(ctx context.Context, query string) (LazyFrame, error)
	ExecuteGlobal(ctx context.Context, query string) (LazyFrame, error)
	RegisterGlobals(tables map[string]DataFrame) error
	Tables() []string
	Unregister(name string)
}
