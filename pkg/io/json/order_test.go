package json

import (
	"reflect"
	"slices"
	"testing"
)

// TestReadKeepsKeyOrder reads every document twenty times: a column order
// taken from map iteration differs between reads.
func TestReadKeepsKeyOrder(t *testing.T) {
	cases := []struct {
		name    string
		content string
		ndjson  bool
		columns []string
		want    []string
		values  map[string][]any
	}{
		{
			name:    "array",
			content: `[{"z": 1, "a": 2, "m": 3}]`,
			want:    []string{"z", "a", "m"},
		},
		{
			name:    "ndjson",
			content: "{\"z\": 1, \"a\": 2, \"m\": 3}\n{\"z\": 4, \"a\": 5, \"m\": 6}\n",
			ndjson:  true,
			want:    []string{"z", "a", "m"},
		},
		{
			name:    "late key",
			content: `[{"a": 1}, {"a": 2, "b": 3}]`,
			want:    []string{"a", "b"},
			values:  map[string][]any{"a": {int64(1), int64(2)}, "b": {nil, int64(3)}},
		},
		{
			name:    "late key ndjson",
			content: "{\"a\": 1}\n{\"b\": 3, \"a\": 2}\n",
			ndjson:  true,
			want:    []string{"a", "b"},
			values:  map[string][]any{"a": {int64(1), int64(2)}, "b": {nil, int64(3)}},
		},
		{
			name:    "later records reorder keys",
			content: `[{"b": 1, "a": 2}, {"a": 3, "c": "x", "b": 4}]`,
			want:    []string{"b", "a", "c"},
			values:  map[string][]any{"c": {nil, "x"}},
		},
		{
			name:    "null record",
			content: `[{"a": 1}, null]`,
			want:    []string{"a"},
			values:  map[string][]any{"a": {int64(1), nil}},
		},
		{
			name:    "projection",
			content: `[{"z": 1, "a": 2, "c": 3}]`,
			columns: []string{"c", "z"},
			want:    []string{"z", "c"},
		},
		{
			name:    "projection ndjson",
			content: "{\"z\": 1, \"a\": 2, \"c\": 3}\n",
			ndjson:  true,
			columns: []string{"c", "z"},
			want:    []string{"z", "c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, "data.json", tc.content)
			for range 20 {
				df, err := Read(ReadInput{Path: path, NDJSON: tc.ndjson, Columns: tc.columns})
				if err != nil {
					t.Fatalf("Read(%q) error = %v, want nil", tc.content, err)
				}
				if got := df.Columns(); !slices.Equal(got, tc.want) {
					t.Fatalf("Read(%q, columns %v) columns = %v, want %v", tc.content, tc.columns, got, tc.want)
				}
				for name, want := range tc.values {
					s, _ := df.Series(name)
					got := make([]any, s.Len())
					for i := range got {
						got[i] = s.Value(i)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("Read(%q) column %q = %#v, want %#v", tc.content, name, got, want)
					}
				}
			}
		})
	}
}

// TestReadRejectsMalformedRecords checks the inputs the token reader must
// still reject, as decoding the whole document did.
func TestReadRejectsMalformedRecords(t *testing.T) {
	cases := []struct {
		name    string
		content string
		ndjson  bool
	}{
		{name: "object document", content: `{"a": 1}`},
		{name: "scalar record", content: `[{"a": 1}, 2]`},
		{name: "truncated array", content: `[{"a": 1}`},
		{name: "truncated record", content: `[{"a": 1`},
		{name: "data after array", content: `[{"a": 1}] {"a": 2}`},
		{name: "truncated ndjson", content: "{\"a\": 1}\n{\"a\":", ndjson: true},
		{name: "scalar ndjson record", content: "{\"a\": 1}\n2\n", ndjson: true},
		{name: "stray bracket ndjson", content: "{\"a\": 1}\n]\n", ndjson: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, "bad.json", tc.content)
			if df, err := Read(ReadInput{Path: path, NDJSON: tc.ndjson}); err == nil {
				t.Errorf("Read(%q, ndjson=%t) = %v columns, error = nil, want an error", tc.content, tc.ndjson, df.Columns())
			}
		})
	}
}
