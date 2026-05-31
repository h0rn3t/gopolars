package ipc

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	iarrow "github.com/eugeneshershen/gopolars/pkg/io/arrow"
)

type WriteInput struct {
	Path string
}

type ReadInput struct {
	Path    string
	Columns []string
}

func init() {
	gob.Register(int64(0))
	gob.Register(float64(0))
	gob.Register("")
	gob.Register(false)
	gob.Register(time.Time{})
}

func Write(df frame.DataFrame, input WriteInput) error {
	f, err := os.Create(input.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := gob.NewEncoder(f)
	return enc.Encode(iarrow.ToTable(df))
}

func Read(input ReadInput) (frame.DataFrame, error) {
	f, err := os.Open(input.Path)
	if err != nil {
		return frame.DataFrame{}, err
	}
	defer func() { _ = f.Close() }()
	var table iarrow.Table
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&table); err != nil {
		return frame.DataFrame{}, err
	}
	if len(input.Columns) == 0 {
		return iarrow.FromTable(table)
	}
	selected := map[string]struct{}{}
	for _, c := range input.Columns {
		selected[c] = struct{}{}
	}
	cols := map[string][]any{}
	for name, values := range table.Columns {
		if _, ok := selected[name]; !ok {
			continue
		}
		cols[name] = values
	}
	return iarrow.FromTable(iarrow.Table{Columns: cols})
}
