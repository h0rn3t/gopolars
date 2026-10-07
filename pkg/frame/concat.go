package frame

import (
	"fmt"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/series"
)

func ConcatVertical(base DataFrame, others ...DataFrame) (DataFrame, error) {
	frames := append([]DataFrame{base}, others...)
	columns := frames[0].Columns()
	for _, f := range frames {
		if len(f.Columns()) != len(columns) {
			return DataFrame{}, fmt.Errorf("concat vertical schema mismatch")
		}
	}
	out := make([]series.Series, 0, len(columns))
	for _, field := range frames[0].Schema() {
		chunks := make([]*chunk.Column, len(frames))
		for i, f := range frames {
			s, ok := f.Series(field.Name)
			if !ok {
				return DataFrame{}, fmt.Errorf("column %s not found", field.Name)
			}
			chunks[i] = s.Column()
		}
		// The output dtype is that of the first part not entirely null; a part
		// that is entirely null, as an Arrow null-typed column imports, fits any
		// dtype and contributes only nulls.
		dtype, typed := chunks[0].DataType(), -1
		for i, c := range chunks {
			if c.NullCount() == c.Len() {
				continue
			}
			if typed < 0 {
				dtype, typed = c.DataType(), i
				continue
			}
			if c.DataType() != dtype {
				return DataFrame{}, fmt.Errorf("concat vertical: column %q has dtype %s in frame %d and %s in frame %d", field.Name, dtype, typed, c.DataType(), i)
			}
		}
		out = append(out, series.FromColumn(field.Name, chunk.ConcatColumns(chunks, dtype)))
	}
	return New(NewInput{Series: out})
}

func ConcatHorizontal(base DataFrame, others ...DataFrame) (DataFrame, error) {
	frames := append([]DataFrame{base}, others...)
	height := frames[0].Height()
	out := make([]series.Series, 0)
	seen := map[string]struct{}{}
	for _, f := range frames {
		if f.Height() != height {
			return DataFrame{}, fmt.Errorf("concat horizontal height mismatch")
		}
		for _, name := range f.Columns() {
			s, _ := f.Series(name)
			outName := name
			if _, ok := seen[outName]; ok {
				i := 1
				for {
					candidate := fmt.Sprintf("%s_%d", name, i)
					if _, exists := seen[candidate]; !exists {
						outName = candidate
						break
					}
					i++
				}
			}
			seen[outName] = struct{}{}
			out = append(out, s.Rename(outName))
		}
	}
	return New(NewInput{Series: out})
}
