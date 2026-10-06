package ipc

import (
	"encoding/gob"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	iarrow "github.com/h0rn3t/gopolars/pkg/io/arrow"
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
	maps.DeleteFunc(table.Columns, func(name string, _ []any) bool {
		return !slices.Contains(input.Columns, name)
	})
	return iarrow.FromTable(table)
}
