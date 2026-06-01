package polars

import (
	"context"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
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
	// SubSelectColumns відповідає Python DataFrame[columns] для підмножини колонок.
	SubSelectColumns(columns ...string) (DataFrame, error)
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
	FilterAggregateDirect(pred Expr, op string, cols []string) (map[string]float64, error)
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
	Max() LazyFrame
	Min() LazyFrame
	Mean() LazyFrame
	Median() LazyFrame
	Sum() LazyFrame
	Std() LazyFrame
	Var() LazyFrame
	Quantile(q float64) LazyFrame
	NullCount() LazyFrame
	Count() LazyFrame
	CollectSchema() dtypes.Schema
	Columns() []string
	Width() int
	Schema() dtypes.Schema
	Dtypes() []dtypes.DataType
	Describe() (DataFrame, error)
	FlattenStruct(column string, prefix string) LazyFrame
	Melt(input MeltInput) LazyFrame
	Pivot(input PivotInput) LazyFrame
	RollingMean(input RollingMeanInput) LazyFrame
	JoinAsof(input JoinInput) LazyFrame
	MapBatches(fn func(DataFrame) (DataFrame, error)) LazyFrame
	// MatchToSchema застосовує MatchToSchema після materialization (через MapBatches).
	MatchToSchema(schema dtypes.Schema) LazyFrame
	// SubSelectColumns відповідає Python LazyFrame[columns].
	SubSelectColumns(columns ...string) LazyFrame
	MergeSorted(other LazyFrame, by string) LazyFrame
	Pipe(fn func(LazyFrame) LazyFrame) LazyFrame
	PipeWithSchema(fn func(LazyFrame, dtypes.Schema) LazyFrame, schema dtypes.Schema) LazyFrame
	Rolling(input RollingMeanInput) LazyFrame
	SelectSeq(exprs ...Expr) LazyFrame
	WithColumnsSeq(exprs ...Expr) LazyFrame
	WithContext(other LazyFrame) LazyFrame
	Remove(column string) LazyFrame
	TopK(k int, by string) LazyFrame
	Show(maxRows int) string
	Lazy() LazyFrame
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
	SinkDelta(ctx context.Context, path string) error
	SinkIceberg(ctx context.Context, path string) error
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
	RollingStd(window int) Series
	RollingVar(window int) Series
	RollingMedian(window int) Series
	RollingQuantile(window int, q float64) Series
	Abs() Series
	Exp() Series
	Log() Series
	Sqrt() Series
	Sin() Series
	Cos() Series
	Tan() Series
	Sinh() Series
	Cosh() Series
	Tanh() Series
	Arcsin() Series
	Arccos() Series
	Arctan() Series
	Arcsinh() Series
	Arccosh() Series
	Arctanh() Series
	Cbrt() Series
	Ceil() Series
	Floor() Series
	Degrees() Series
	Sign() Series
	Log10() Series
	Log1p() Series
	Round() Series
	Pow(power float64) Series
	Shift(periods int) Series
	Reverse() Series
	Sum() float64
	Std() float64
	Max() float64
	Min() float64
	Mean() float64
	Median() float64
	Var() float64
	NUnique() int
	Mode() any
	Kurtosis() float64
	Skew() float64
	Quantile(q float64) float64
	Product() float64
	NanMax() float64
	NanMin() float64
	Alias(name string) Series
	Clone() Series
	Clear() Series
	Head(n int) Series
	Tail(n int) Series
	Limit(n int) Series
	Slice(offset int, length int) Series
	Sort(descending bool) Series
	Unique() Series
	ArgSort() Series
	ArgMax() int
	ArgMin() int
	ArgUnique() Series
	ArgTrue() Series
	Gather(indices []int) Series
	GatherEvery(step int) Series
	Sample(n int, seed int64) Series
	Shuffle(seed int64) Series
	Rechunk() Series
	ShrinkToFit() Series
	Shape() [1]int
	IsEmpty() bool
	IsSorted() bool
	HasNulls() bool
	HasValidity() bool
	NChunks() int
	ChunkLengths() []int
	All() bool
	Any() bool
	Not_() Series
	IsNan() Series
	IsNotNan() Series
	IsFinite() Series
	IsInfinite() Series
	IsDuplicated() Series
	IsUnique() Series
	IsFirstDistinct() Series
	IsLastDistinct() Series
	IsBetween(lower float64, upper float64) Series
	IsClose(other Series) (Series, error)
	IsIn(values []any) Series
	EqMissing(other Series) (Series, error)
	NeMissing(other Series) (Series, error)
	CumSum() Series
	CumMax() Series
	CumMin() Series
	CumProd() Series
	CumCount() Series
	EwmMean(alpha float64) Series
	EwmStd(alpha float64) Series
	EwmVar(alpha float64) Series
	Diff(n int) Series
	PctChange(n int) Series
	Dot(other Series) (float64, error)
	Entropy() float64
	ValueCounts() (DataFrame, error)
	UniqueCounts() Series
	TopK(k int) Series
	TopKBy(by Series, k int) (Series, error)
	BottomK(k int) Series
	BottomKBy(by Series, k int) (Series, error)
	PeakMax() Series
	PeakMin() Series
	InterpolateBy(by Series) (Series, error)
	ForwardFill() Series
	BackwardFill() Series
	Cut(breaks []float64) (Series, error)
	QCut(q int) (Series, error)
	Rle() (DataFrame, error)
	RleId() Series
	Rank() Series
	SearchSorted(value any) int
	LowerBound(value any) int
	UpperBound(value any) int
	Item(index int) (any, error)
	First() any
	Last() any
	IndexOf(value any) int
	MapElements(fn func(any) any) (Series, error)
	Replace(old any, new any) Series
	ReplaceStrict(old any, new any) (Series, error)
	Reshape(dims ...int) (Series, error)
	RepeatBy(n int) Series
	SetSorted(descending bool) Series
	ToFrame() (DataFrame, error)
	ToDummies() (DataFrame, error)
	ToArrow() (iarrow.Table, error)
	ToInitRepr() string
	ToJax() []float64
	ToTorch() []float64
	ToPhysical() Series
	Rename(name string) Series
	Serialize() ([]byte, error)
	Deserialize(payload []byte) (Series, error)
	Hash(seed uint64) Series
	Implode() Series
	Explode() Series
	Flatten() Series
	Extend(other Series) (Series, error)
	ExtendConstant(value any, n int) Series
	NewFromIndex(index int, n int) (Series, error)
	Scatter(indices []int, values []any) (Series, error)
	Set(mask Series, value any) (Series, error)
	ZipWith(mask Series, other Series) (Series, error)
	Describe() map[string]any
	Hist(bins int) (DataFrame, error)
	Interpolate() Series
	ToNumpy() []any
	ToPandas() []any
	// Str/Arr/Dt/Struct/Cat/Bin — простори імен як у Python Polars Series.
	Str() SeriesStrNS
	Arr() SeriesArrNS
	Dt() SeriesDtNS
	Struct() SeriesStructNS
	Cat() SeriesCatNS
	Bin() SeriesBinNS
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

	Count() int
	ApproxNUnique() int
	Equals(other Series) (bool, error)
	EstimatedSize() int64
	Filter(mask Series) (Series, error)
	Truncate(maxLen int) (Series, error)
	RoundSigFigs(sigFigs int) (Series, error)
	Clip(lower, upper float64) Series
	Cot() Series
	ShrinkDType() (Series, error)
	List() SeriesArrNS
	Append(other Series) (Series, error)
	GetChunks() ([]Series, error)
	Flags() map[string]bool
	MaxBy(by Series) (Series, error)
	MinBy(by Series) (Series, error)
	RollingMeanBy(by Series, window int) (Series, error)
	RollingSumBy(by Series, window int) (Series, error)
	RollingMinBy(by Series, window int) (Series, error)
	RollingMaxBy(by Series, window int) (Series, error)
	RollingStdBy(by Series, window int) (Series, error)
	RollingVarBy(by Series, window int) (Series, error)
	RollingMedianBy(by Series, window int) (Series, error)
	RollingQuantileBy(by Series, window int, q float64) (Series, error)
	RollingRank(window int) Series
	RollingRankBy(by Series, window int) (Series, error)
	RollingSkew(window int) Series
	RollingKurtosis(window int) Series
	RollingMap(window int, fn func([]float64) float64) (Series, error)
	EwmMeanBy(by Series, alpha float64) (Series, error)
	BitwiseAnd(other Series) (Series, error)
	BitwiseOr(other Series) (Series, error)
	BitwiseXor(other Series) (Series, error)
	BitwiseCountOnes() (Series, error)
	BitwiseCountZeros() (Series, error)
	BitwiseLeadingOnes() (Series, error)
	BitwiseLeadingZeros() (Series, error)
	BitwiseTrailingOnes() (Series, error)
	BitwiseTrailingZeros() (Series, error)
	Reinterpret(dt dtypes.DataType) (Series, error)
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
