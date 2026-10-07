package ipc

import (
	"encoding/gob"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// legacyTable is the shape of the gob table written before column order was
// stored.
type legacyTable struct {
	Columns map[string][]any
}

func orderFrame(t *testing.T) frame.DataFrame {
	t.Helper()
	var cols []frame.SeriesInput
	for i, name := range []string{"z", "a", "m", "b", "c"} {
		cols = append(cols, frame.SeriesInput{Name: name, Values: []any{int64(i), int64(i * 10)}})
	}
	df, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{Columns: cols})
	if err != nil {
		t.Fatalf("FromAnyColumns: %v", err)
	}
	return df
}

// TestReadKeepsColumnOrder reads the file twenty times: an order taken from
// map iteration differs between reads.
func TestReadKeepsColumnOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order.ipc")
	if err := Write(orderFrame(t), WriteInput{Path: path}); err != nil {
		t.Fatalf("Write(%q): %v", path, err)
	}
	cases := []struct {
		name    string
		columns []string
		want    []string
	}{
		{name: "all", want: []string{"z", "a", "m", "b", "c"}},
		{name: "projection", columns: []string{"c", "z"}, want: []string{"z", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for range 20 {
				df, err := Read(ReadInput{Path: path, Columns: tc.columns})
				if err != nil {
					t.Fatalf("Read(%q, %v) error = %v, want nil", path, tc.columns, err)
				}
				if got := df.Columns(); !slices.Equal(got, tc.want) {
					t.Fatalf("Read(%q, %v) columns = %v, want %v", path, tc.columns, got, tc.want)
				}
			}
		})
	}
}

// TestLegacyFilesReadBothWays checks gob compatibility with the table shape
// that has no stored column order, in both directions.
func TestLegacyFilesReadBothWays(t *testing.T) {
	t.Run("old file, new reader", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "legacy.ipc")
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %q: %v", path, err)
		}
		old := legacyTable{Columns: map[string][]any{"b": {int64(1)}, "a": {"x"}}}
		if err := gob.NewEncoder(f).Encode(old); err != nil {
			t.Fatalf("encode legacy table: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close %q: %v", path, err)
		}
		for range 20 {
			df, err := Read(ReadInput{Path: path})
			if err != nil {
				t.Fatalf("Read(%q) error = %v, want nil", path, err)
			}
			if got, want := df.Columns(), []string{"a", "b"}; !slices.Equal(got, want) {
				t.Fatalf("Read(%q) columns = %v, want %v", path, got, want)
			}
		}
	})
	t.Run("new file, old reader", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "new.ipc")
		if err := Write(orderFrame(t), WriteInput{Path: path}); err != nil {
			t.Fatalf("Write(%q): %v", path, err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %q: %v", path, err)
		}
		defer func() { _ = f.Close() }() // read-only file
		var old legacyTable
		if err := gob.NewDecoder(f).Decode(&old); err != nil {
			t.Fatalf("decode %q as legacy table: %v", path, err)
		}
		got := slices.Sorted(maps.Keys(old.Columns))
		if want := []string{"a", "b", "c", "m", "z"}; !slices.Equal(got, want) {
			t.Errorf("legacy decode of %q columns = %v, want %v", path, got, want)
		}
		if got, want := old.Columns["m"], []any{int64(2), int64(20)}; !slices.Equal(got, want) {
			t.Errorf("legacy decode of %q column m = %v, want %v", path, got, want)
		}
	})
}
