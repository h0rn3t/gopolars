package polars

import (
	"fmt"
	"sort"
	"time"

	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
)

type WindowSumInput struct {
	PartitionBy []string
	OrderBy     string
	Value       string
	Output      string
}

type RollingMeanInput struct {
	By      string
	Value   string
	Window  time.Duration
	MinRows int
	Output  string
	Closed  string
}

type DynamicGroupInput struct {
	By           string
	Every        time.Duration
	Period       time.Duration
	Offset       time.Duration
	Closed       string
	Label        string
	WindowColumn string
	AggExpr      Expr
}

type MeltInput struct {
	IDVars      []string
	ValueVars   []string
	VariableCol string
	ValueCol    string
}

type PivotInput struct {
	Index     string
	Columns   string
	Values    string
	Agg       string
	ValueName string
}

func WindowSum(df DataFrame, input WindowSumInput) (DataFrame, error) {
	table, err := df.ToArrow(ToArrowInput{})
	if err != nil {
		return nil, err
	}
	partKey := make([]string, 0, len(input.PartitionBy))
	rows := len(table.Columns[input.Value])
	orderValues, ok := table.Columns[input.OrderBy]
	if !ok {
		return nil, fmt.Errorf("order column not found")
	}
	valueValues, ok := table.Columns[input.Value]
	if !ok {
		return nil, fmt.Errorf("value column not found")
	}
	indexes := make([]int, rows)
	for i := 0; i < rows; i++ {
		indexes[i] = i
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		li := indexes[i]
		rj := indexes[j]
		for _, p := range input.PartitionBy {
			lv := fmt.Sprintf("%v", table.Columns[p][li])
			rv := fmt.Sprintf("%v", table.Columns[p][rj])
			if lv != rv {
				return lv < rv
			}
		}
		lt, lok := orderValues[li].(time.Time)
		rt, rok := orderValues[rj].(time.Time)
		if lok && rok {
			return lt.Before(rt)
		}
		return fmt.Sprintf("%v", orderValues[li]) < fmt.Sprintf("%v", orderValues[rj])
	})
	out := make([]any, rows)
	runningByPartition := map[string]float64{}
	for _, idx := range indexes {
		partKey = partKey[:0]
		for _, p := range input.PartitionBy {
			partKey = append(partKey, fmt.Sprintf("%v", table.Columns[p][idx]))
		}
		k := fmt.Sprintf("%v", partKey)
		v := valueValues[idx]
		switch t := v.(type) {
		case int64:
			runningByPartition[k] += float64(t)
		case float64:
			runningByPartition[k] += t
		default:
			return nil, fmt.Errorf("window sum supports int64/float64")
		}
		out[idx] = runningByPartition[k]
	}
	table.Columns[input.Output] = out
	return NewDataFrameFromArrow(table)
}

func RollingMean(df DataFrame, input RollingMeanInput) (DataFrame, error) {
	return df.RollingMean(input)
}

func GroupByDynamic(df DataFrame, input DynamicGroupInput) (DataFrame, error) {
	return df.GroupByDynamic(input)
}

func Melt(df DataFrame, input MeltInput) (DataFrame, error) {
	table, err := df.ToArrow(ToArrowInput{})
	if err != nil {
		return nil, err
	}
	if input.VariableCol == "" {
		input.VariableCol = "variable"
	}
	if input.ValueCol == "" {
		input.ValueCol = "value"
	}
	rowCount := len(table.Columns[input.IDVars[0]])
	outCols := map[string][]any{}
	for _, id := range input.IDVars {
		outCols[id] = make([]any, 0, rowCount*len(input.ValueVars))
	}
	outCols[input.VariableCol] = make([]any, 0, rowCount*len(input.ValueVars))
	outCols[input.ValueCol] = make([]any, 0, rowCount*len(input.ValueVars))
	for i := 0; i < rowCount; i++ {
		for _, vv := range input.ValueVars {
			for _, id := range input.IDVars {
				outCols[id] = append(outCols[id], table.Columns[id][i])
			}
			outCols[input.VariableCol] = append(outCols[input.VariableCol], vv)
			outCols[input.ValueCol] = append(outCols[input.ValueCol], table.Columns[vv][i])
		}
	}
	return NewDataFrameFromArrow(iarrow.Table{Columns: outCols})
}

func Pivot(df DataFrame, input PivotInput) (DataFrame, error) {
	table, err := df.ToArrow(ToArrowInput{})
	if err != nil {
		return nil, err
	}
	idxVals := table.Columns[input.Index]
	colVals := table.Columns[input.Columns]
	valVals := table.Columns[input.Values]
	colOrder := []string{}
	colSeen := map[string]struct{}{}
	type agg struct {
		sum   float64
		count int
	}
	aggMap := map[string]map[string]agg{}
	for i := range idxVals {
		idx := fmt.Sprintf("%v", idxVals[i])
		col := fmt.Sprintf("%v", colVals[i])
		if _, ok := colSeen[col]; !ok {
			colSeen[col] = struct{}{}
			colOrder = append(colOrder, col)
		}
		if _, ok := aggMap[idx]; !ok {
			aggMap[idx] = map[string]agg{}
		}
		current := aggMap[idx][col]
		switch t := valVals[i].(type) {
		case int64:
			current.sum += float64(t)
			current.count++
		case float64:
			current.sum += t
			current.count++
		}
		aggMap[idx][col] = current
	}
	indexOrder := make([]string, 0, len(aggMap))
	for idx := range aggMap {
		indexOrder = append(indexOrder, idx)
	}
	sort.Strings(indexOrder)
	outCols := map[string][]any{input.Index: make([]any, 0, len(indexOrder))}
	for _, c := range colOrder {
		outCols[c] = make([]any, 0, len(indexOrder))
	}
	for _, idx := range indexOrder {
		outCols[input.Index] = append(outCols[input.Index], idx)
		for _, c := range colOrder {
			a := aggMap[idx][c]
			switch input.Agg {
			case "mean":
				if a.count == 0 {
					outCols[c] = append(outCols[c], nil)
				} else {
					outCols[c] = append(outCols[c], a.sum/float64(a.count))
				}
			case "count":
				outCols[c] = append(outCols[c], int64(a.count))
			default:
				outCols[c] = append(outCols[c], a.sum)
			}
		}
	}
	return NewDataFrameFromArrow(iarrow.Table{Columns: outCols})
}
