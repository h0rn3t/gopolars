package arrow

import "github.com/eugeneshershen/gopolars/pkg/frame"

type Table struct {
	Columns map[string][]any
}

func ToTable(df frame.DataFrame) Table {
	out := Table{Columns: map[string][]any{}}
	for _, name := range df.Columns() {
		s, _ := df.Series(name)
		values := make([]any, 0, df.Height())
		for i := 0; i < df.Height(); i++ {
			values = append(values, s.Value(i))
		}
		out.Columns[name] = values
	}
	return out
}

func FromTable(t Table) (frame.DataFrame, error) {
	seriesInput := make([]frame.SeriesInput, 0)
	for k, v := range t.Columns {
		seriesInput = append(seriesInput, frame.SeriesInput{Name: k, Values: v})
	}
	return frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: seriesInput})
}
