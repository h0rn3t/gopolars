package polars

import (
	"fmt"

	"github.com/eugeneshershen/gopolars/pkg/dtypes"
	"github.com/eugeneshershen/gopolars/pkg/exec"
	"github.com/eugeneshershen/gopolars/pkg/frame"
	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
	icsv "github.com/eugeneshershen/gopolars/pkg/io/csv"
	iipc "github.com/eugeneshershen/gopolars/pkg/io/ipc"
	ijson "github.com/eugeneshershen/gopolars/pkg/io/json"
	iparquet "github.com/eugeneshershen/gopolars/pkg/io/parquet"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
	"github.com/eugeneshershen/gopolars/pkg/series"
)

type df struct {
	value frame.DataFrame
}

func NewDataFrame(input NewDataFrameInput) (DataFrame, error) {
	return fromFrame(frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: input.Columns}))
}

func NewDataFrameFromArrow(table iarrow.Table) (DataFrame, error) {
	return fromFrame(iarrow.FromTable(table))
}

type NewDataFrameInput struct {
	Columns []frame.SeriesInput
}

func fromFrame(v frame.DataFrame, err error) (DataFrame, error) {
	if err != nil {
		return nil, err
	}
	return &df{value: v}, nil
}

func (d *df) Schema() dtypes.Schema {
	return d.value.Schema()
}

func (d *df) Height() int {
	return d.value.Height()
}

func (d *df) Width() int {
	return d.value.Width()
}

func (d *df) Columns() []string {
	return d.value.Columns()
}

func (d *df) Series(name string) (Series, bool) {
	s, ok := d.value.Series(name)
	if !ok {
		return nil, false
	}
	return fromInternalSeries(s), true
}

func (d *df) Select(exprs ...Expr) (DataFrame, error) {
	next, err := d.value.Select(exprs...)
	return fromFrame(next, err)
}

func (d *df) Filter(predicate Expr) (DataFrame, error) {
	next, err := d.value.Filter(predicate)
	return fromFrame(next, err)
}

func (d *df) WithColumns(exprs ...Expr) (DataFrame, error) {
	next, err := d.value.WithColumns(exprs...)
	return fromFrame(next, err)
}

func (d *df) GroupBy(keys ...string) GroupBy {
	return groupBy{value: d.value.GroupBy(keys...)}
}

func (d *df) GroupByDynamic(input DynamicGroupInput) (DataFrame, error) {
	next, err := d.value.GroupByDynamic(
		input.By,
		input.Every,
		input.Period,
		input.Offset,
		input.Closed,
		input.Label,
		input.WindowColumn,
		input.AggExpr,
	)
	return fromFrame(next, err)
}

func (d *df) Join(input JoinInput) (DataFrame, error) {
	other, ok := input.Other.(*df)
	if !ok {
		return nil, fmt.Errorf("unsupported dataframe implementation")
	}
	next, err := d.value.Join(frame.JoinInput{
		Other:         other.value,
		LeftOn:        input.LeftOn,
		RightOn:       input.RightOn,
		How:           frame.JoinType(input.How),
		Suffix:        input.Suffix,
		AsofDirection: input.AsofDirection,
		AsofTolerance: input.AsofTolerance,
	})
	return fromFrame(next, err)
}

func (d *df) Sort(input SortInput) (DataFrame, error) {
	next, err := d.value.Sort(frame.SortInput{
		By:            input.By,
		Descending:    input.Descending,
		NullsLast:     input.NullsLast,
		MaintainOrder: input.MaintainOrder,
	})
	return fromFrame(next, err)
}

func (d *df) Concat(input ConcatInput) (DataFrame, error) {
	others := make([]frame.DataFrame, 0, len(input.Others))
	for _, other := range input.Others {
		next, ok := other.(*df)
		if !ok {
			return nil, fmt.Errorf("unsupported dataframe implementation")
		}
		others = append(others, next.value)
	}
	if input.How == "horizontal" {
		return fromFrame(frame.ConcatHorizontal(d.value, others...))
	}
	return fromFrame(frame.ConcatVertical(d.value, others...))
}

func (d *df) Limit(n int) DataFrame {
	return &df{value: d.value.Limit(n)}
}

func (d *df) Slice(offset int, length int) DataFrame {
	return &df{value: d.value.Slice(offset, length)}
}

func (d *df) Head(n int) DataFrame {
	return d.Limit(n)
}

func (d *df) Tail(n int) DataFrame {
	if n >= d.value.Height() {
		return d
	}
	if n <= 0 {
		return d.Limit(0)
	}
	start := d.value.Height() - n
	indexes := make([]int, 0, n)
	for i := start; i < d.value.Height(); i++ {
		indexes = append(indexes, i)
	}
	out := make([]series.Series, 0, len(d.value.Columns()))
	for _, name := range d.value.Columns() {
		s, _ := d.value.Series(name)
		out = append(out, s.Slice(indexes))
	}
	next, err := frame.New(frame.NewInput{Series: out})
	if err != nil {
		return d.Limit(0)
	}
	return &df{value: next}
}

func (d *df) Unique(columns ...string) (DataFrame, error) {
	next, err := d.value.Unique(columns...)
	return fromFrame(next, err)
}

func (d *df) FillNull(value any) (DataFrame, error) {
	next, err := d.value.FillNull(value)
	return fromFrame(next, err)
}

func (d *df) DropNulls(columns ...string) DataFrame {
	return &df{value: d.value.DropNulls(columns...)}
}

func (d *df) Explode(columns ...string) (DataFrame, error) {
	next, err := d.value.Explode(columns...)
	return fromFrame(next, err)
}

func (d *df) FlattenStruct(column string, prefix string) (DataFrame, error) {
	next, err := d.value.FlattenStruct(column, prefix)
	return fromFrame(next, err)
}

func (d *df) Melt(input MeltInput) (DataFrame, error) {
	next, err := d.value.Melt(input.IDVars, input.ValueVars, input.VariableCol, input.ValueCol)
	return fromFrame(next, err)
}

func (d *df) Pivot(input PivotInput) (DataFrame, error) {
	idx := []string{}
	if input.Index != "" {
		idx = append(idx, input.Index)
	}
	next, err := d.value.Pivot(idx, input.Columns, input.Values, input.Agg)
	if err != nil {
		return nil, err
	}
	if input.ValueName != "" {
		return (&df{value: next}).Rename(map[string]string{input.Values: input.ValueName})
	}
	return fromFrame(next, nil)
}

func (d *df) RollingMean(input RollingMeanInput) (DataFrame, error) {
	next, err := d.value.RollingMean(input.By, input.Value, input.Window, input.MinRows, input.Output, input.Closed)
	return fromFrame(next, err)
}

func (d *df) Drop(columns ...string) (DataFrame, error) {
	dropSet := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		dropSet[c] = struct{}{}
	}
	out := make([]series.Series, 0, d.value.Width())
	for _, name := range d.value.Columns() {
		if _, shouldDrop := dropSet[name]; shouldDrop {
			continue
		}
		s, _ := d.value.Series(name)
		out = append(out, s.Clone())
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Rename(mapping map[string]string) (DataFrame, error) {
	if len(mapping) == 0 {
		return d, nil
	}
	out := make([]series.Series, 0, d.value.Width())
	for _, name := range d.value.Columns() {
		s, _ := d.value.Series(name)
		if nextName, ok := mapping[name]; ok && nextName != "" {
			out = append(out, s.Rename(nextName))
			continue
		}
		out = append(out, s.Clone())
	}
	return fromFrame(frame.New(frame.NewInput{Series: out}))
}

func (d *df) Lazy() LazyFrame {
	return &lf{
		source: d.value,
		engine: exec.New(),
		nodes:  []logical.Node{},
	}
}

func (d *df) WriteCSV(input WriteCSVInput) error {
	return icsv.Write(d.value, icsv.WriteInput{
		Path:          input.Path,
		IncludeHeader: input.IncludeHeader,
		Separator:     input.Separator,
	})
}

func (d *df) WriteJSON(input WriteJSONInput) error {
	return ijson.Write(d.value, ijson.WriteInput{
		Path:   input.Path,
		Pretty: input.Pretty,
		NDJSON: input.NDJSON,
	})
}

func (d *df) WriteParquet(input WriteParquetInput) error {
	return iparquet.Write(d.value, iparquet.WriteInput{
		Path:         input.Path,
		Compression:  input.Compression,
		RowGroupSize: input.RowGroupSize,
	})
}

func (d *df) WriteIPC(input WriteIPCInput) error {
	return iipc.Write(d.value, iipc.WriteInput{Path: input.Path})
}

func (d *df) ToArrow(input ToArrowInput) (iarrow.Table, error) {
	_ = input
	return iarrow.ToTable(d.value), nil
}
