package expr

import "testing"

type rasterRow struct {
	index int
	rows  int
	data  map[string][]any
}

func (r rasterRow) ValueByName(name string) (any, bool) {
	col, ok := r.data[name]
	if !ok || r.index < 0 || r.index >= len(col) {
		return nil, false
	}
	return col[r.index], true
}

func (r rasterRow) RowIndex() int { return r.index }

func (r rasterRow) NumRows() int { return r.rows }

func (r rasterRow) ValueAt(row int, column string) (any, bool) {
	col, ok := r.data[column]
	if !ok || row < 0 || row >= len(col) {
		return nil, false
	}
	return col[row], true
}

func TestReverseRowCtxReadsMirroredRow(t *testing.T) {
	t.Parallel()

	base := rasterRow{
		index: 0,
		rows:  3,
		data: map[string][]any{
			"v": {int64(10), int64(20), int64(30)},
		},
	}
	rev := reverseRowCtx{base: base}

	got, ok := rev.ValueByName("v")
	if !ok || got != int64(30) {
		t.Fatalf("reverse row 0: %v ok=%v", got, ok)
	}
	if rev.RowIndex() != 0 || rev.NumRows() != 3 {
		t.Fatalf("metadata: idx=%d rows=%d", rev.RowIndex(), rev.NumRows())
	}
	at, ok := rev.ValueAt(0, "v")
	if !ok || at != int64(30) {
		t.Fatalf("value at mirrored 0: %v ok=%v", at, ok)
	}
	if _, ok := rev.ValueAt(99, "v"); ok {
		t.Fatal("очікували false для індексу поза діапазоном")
	}
}
