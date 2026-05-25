package expr

// RasterRowAccess розширює RowValueGetter випадковим читанням рядка — потрібно для
// виразів на кшталт Reverse() у Filter, де значення залежить від іншого індексу рядка.
type RasterRowAccess interface {
	RowValueGetter
	RowIndex() int
	NumRows() int
	ValueAt(row int, column string) (any, bool)
}

type reverseRowCtx struct {
	base RasterRowAccess
}

func (r reverseRowCtx) ValueByName(name string) (any, bool) {
	h := r.base.NumRows()
	br := r.base.RowIndex()
	if h <= 0 || br < 0 || br >= h {
		return nil, false
	}
	return r.base.ValueAt(h-1-br, name)
}

func (r reverseRowCtx) RowIndex() int { return r.base.RowIndex() }

func (r reverseRowCtx) NumRows() int { return r.base.NumRows() }

func (r reverseRowCtx) ValueAt(row int, column string) (any, bool) {
	h := r.base.NumRows()
	if h <= 0 || row < 0 || row >= h {
		return nil, false
	}
	return r.base.ValueAt(h-1-row, column)
}
