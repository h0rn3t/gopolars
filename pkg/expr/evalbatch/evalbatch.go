// Package evalbatch evaluates a supported subset of expression trees over
// entire typed columns in one vectorized pass, instead of calling the row-wise
// expr.Eval for every row. Unsupported expressions are rejected by Compile so
// callers can fall back to the row-wise evaluator.
//
// For every supported operation the batch result MUST match the semantics of
// the row-wise evaluator in pkg/expr (eval.go), including its null and type
// rules. The helpers below intentionally mirror that logic.
package evalbatch

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/h0rn3t/gopolars/pkg/chunk"
	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/simd"
)

// Plan is a validated expression ready for batch evaluation.
type Plan struct {
	root expr.Expr
}

// Compile validates that e is composed only of batch-supported operations and
// returns a Plan. The second result is false when e must fall back to the
// row-wise evaluator.
func Compile(e expr.Expr) (*Plan, bool) {
	if !supported(e) {
		return nil, false
	}
	return &Plan{root: e}, true
}

// AsColCmpLit reports whether the plan's root is a single `col <cmp> lit`
// comparison with the column on the left and a numeric literal on the right —
// the shape the single-pass filter-reduce fast path accelerates. It returns the
// column name, the simd comparison code, the literal as float64, and ok.
//
// Only gt/ge/lt/le are reported: for these a NaN operand fails the comparison
// under Go's native float semantics exactly as the bitmap path's NaN-exclusion
// does, so the single-pass kernel is bit-for-bit equivalent. eq/ne (whose NaN
// handling the row-wise evaluator treats specially) return ok=false and fall
// back to the bitmap path. The column dtype is checked by the caller, which
// holds the typed columns.
func (p *Plan) AsColCmpLit() (col string, cmpCode simd.Cmp, litF float64, ok bool) {
	e := p.root
	if e.Kind() != expr.KindBin {
		return "", 0, 0, false
	}
	var c simd.Cmp
	switch e.Op() {
	case "gt":
		c = simd.CmpGT
	case "ge":
		c = simd.CmpGE
	case "lt":
		c = simd.CmpLT
	case "le":
		c = simd.CmpLE
	default:
		return "", 0, 0, false
	}
	l, r := e.Left(), e.Right()
	if l == nil || r == nil || l.Kind() != expr.KindCol || r.Kind() != expr.KindLit {
		return "", 0, 0, false
	}
	f, ok := numericLitToFloat(r.Value())
	if !ok {
		return "", 0, 0, false
	}
	return l.ColName(), c, f, true
}

// numericLitToFloat converts a numeric literal value to float64, reporting false
// for non-numeric (e.g. string/bool) literals.
func numericLitToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

var batchBinOps = map[string]bool{
	"eq": true, "ne": true, "gt": true, "ge": true, "lt": true, "le": true,
	"add": true, "sub": true, "mul": true, "div": true,
	"mod": true, "floordiv": true, "pow": true,
	"and": true, "or": true,
}

func supported(e expr.Expr) bool {
	switch e.Kind() {
	case expr.KindCol, expr.KindLit:
		return true
	case expr.KindCast:
		switch e.CastType() {
		case dtypes.Int64, dtypes.Float64, dtypes.String, dtypes.Boolean, dtypes.Datetime:
			return e.Target() != nil && supported(*e.Target())
		}
		return false
	case expr.KindUnary:
		return e.Op() == "not" && e.Target() != nil && supported(*e.Target())
	case expr.KindBin:
		if e.Op() == "fill_null_expr" || e.Op() == "fill_nan_expr" {
			// Coalesce a column against a float64 literal fill value.
			return e.Left() != nil && e.Right() != nil &&
				supported(*e.Left()) && e.Right().Kind() == expr.KindLit
		}
		if e.Op() == "is_in" {
			// Membership in a literal list; against a column it stays row-wise.
			if e.Left() == nil || e.Right() == nil || e.Right().Kind() != expr.KindLit {
				return false
			}
			_, isList := e.Right().Value().([]any)
			return isList && supported(*e.Left())
		}
		if !batchBinOps[e.Op()] {
			return false
		}
		return e.Left() != nil && e.Right() != nil && supported(*e.Left()) && supported(*e.Right())
	default:
		return false
	}
}

// vresult is an intermediate evaluation result: either a column or a literal.
type vresult struct {
	col   *chunk.Column
	lit   any
	isLit bool
}

// Eval evaluates the plan against the given columns and returns a typed result
// column of length height.
func (p *Plan) Eval(cols map[string]*chunk.Column, height int) (*chunk.Column, error) {
	r, err := evalNode(p.root, cols, height)
	if err != nil {
		return nil, err
	}
	if r.isLit {
		return broadcast(r.lit, height), nil
	}
	return r.col, nil
}

// EvalBool evaluates a predicate plan into a packed simd.Bitmap: bit i is set
// iff the predicate is true and non-null at row i. One bit per row replaces the
// old one-byte-per-row []bool mask, cutting the mask 8x and letting the
// fused-reduce / compress kernels walk it a word at a time. A null predicate
// value leaves its bit clear, which is how the row-wise Filter treats it.
//
// For the common predicate shapes — a numeric column compared to a literal,
// the AND of such predicates, a bare boolean column, or is_in against a
// literal list — the Bitmap is produced directly by the simd compare kernels
// (CompareGTFloat64Bitmap / CompareEQInt64Bitmap / BitmapAnd) with a null
// operand folded to a 0 bit, so no intermediate []bool byte-mask is allocated.
// Any other shape falls back to the general column evaluator and packs the
// resulting boolean chunk into a Bitmap.
func (p *Plan) EvalBool(cols map[string]*chunk.Column, height int) (simd.Bitmap, error) {
	if bm, ok := evalBitmap(p.root, cols, height); ok {
		return bm, nil
	}
	c, err := p.Eval(cols, height)
	if err != nil {
		return nil, err
	}
	if c.DataType() != dtypes.Boolean {
		return nil, fmt.Errorf("predicate did not evaluate to bool")
	}
	return packBoolColumn(c, height), nil
}

// evalBitmap tries to evaluate predicate e directly to a Bitmap without going
// through the []bool column engine. ok is false when e is not one of the
// directly-supported predicate shapes and the caller must use the general
// evaluator (p.Eval) and pack its result.
func evalBitmap(e expr.Expr, cols map[string]*chunk.Column, height int) (simd.Bitmap, bool) {
	switch e.Kind() {
	case expr.KindCol:
		c, found := cols[e.ColName()]
		if !found || c.DataType() != dtypes.Boolean {
			return nil, false
		}
		// A nullable boolean goes through the general evaluator, which packs
		// its nulls as clear bits.
		if hasNulls(c) {
			return nil, false
		}
		return packBoolColumn(c, height), true
	case expr.KindBin:
		switch op := e.Op(); op {
		case "gt", "ge", "lt", "le", "eq":
			return cmpBitmap(op, e, cols, height)
		case "is_in":
			l, err := evalNode(*e.Left(), cols, height)
			if err != nil || l.isLit {
				return nil, false
			}
			mask, err := isInMask(l.col, e.Right().Value().([]any), height)
			return mask, err == nil
		case "and":
			la, lok := evalBitmap(*e.Left(), cols, height)
			if !lok {
				return nil, false
			}
			rb, rok := evalBitmap(*e.Right(), cols, height)
			if !rok {
				return nil, false
			}
			return simd.BitmapAnd(la, rb, height), true
		}
	}
	return nil, false
}

// cmpBitmap evaluates a numeric "column op literal" comparison directly into a
// Bitmap, mirroring numericCompare's semantics (a null or NaN operand yields a
// 0 bit). It handles only column-on-left / literal-on-right; any other operand
// shape returns ok=false so the caller falls back. The gt-float and eq-int
// cases route through the dedicated simd kernels.
func cmpBitmap(op string, e expr.Expr, cols map[string]*chunk.Column, height int) (simd.Bitmap, bool) {
	l, err := evalNode(*e.Left(), cols, height)
	if err != nil {
		return nil, false
	}
	r, err := evalNode(*e.Right(), cols, height)
	if err != nil {
		return nil, false
	}
	lr, lTag := numericReader(l)
	rr, rTag := numericReader(r)
	if lTag == "other" || rTag == "other" {
		return nil, false
	}
	if lr.isLit || !rr.isLit {
		return nil, false
	}
	switch {
	case op == "gt" && lTag == "float64" && rTag == "float64":
		b := simd.CompareGTFloat64Bitmap(lr.f, rr.litF)
		clearNullBits(b, lr.nulls, height)
		return b, true
	case op == "eq" && lTag == "int64" && rTag == "int64":
		b := simd.CompareEQInt64Bitmap(lr.i, rr.litI)
		clearNullBits(b, lr.nulls, height)
		return b, true
	case op == "eq":
		// float equality carries NaN subtleties handled by the row-wise path.
		return nil, false
	}
	// General gt/ge/lt/le over numeric column vs literal.
	b := simd.BitmapNew(height)
	useFloat := lTag == "float64" || rTag == "float64"
	for i := range height {
		if lr.nullAt(i) || rr.nullAt(i) {
			continue
		}
		if useFloat {
			a, c := lr.floatAt(i), rr.floatAt(i)
			if math.IsNaN(a) || math.IsNaN(c) {
				continue
			}
			if cmpOrdered(op, a, c) {
				simd.BitmapSet(b, i)
			}
		} else if cmpOrdered(op, lr.intAt(i), rr.intAt(i)) {
			simd.BitmapSet(b, i)
		}
	}
	return b, true
}

// clearNullBits clears the bitmap bit for every null row, so a kernel that set a
// bit from a null operand's zero value (e.g. 0 > -1, 0 == 0) is corrected to the
// row-wise "null operand yields false" semantics.
func clearNullBits(b simd.Bitmap, nulls []bool, height int) {
	if nulls == nil {
		return
	}
	for i := range height {
		if nulls[i] {
			b[i>>6] &^= 1 << (uint(i) & 63)
		}
	}
}

// packBoolColumn packs a boolean result chunk into a Bitmap, setting bit i iff
// the value is true and non-null. It is the fallback bridge from the column
// engine to the Bitmap predicate representation.
func packBoolColumn(c *chunk.Column, height int) simd.Bitmap {
	bln, _ := c.Bools()
	nulls := c.Nulls()
	b := simd.BitmapNew(height)
	for i := range height {
		if bln[i] && (nulls == nil || !nulls[i]) {
			simd.BitmapSet(b, i)
		}
	}
	return b
}

func evalNode(e expr.Expr, cols map[string]*chunk.Column, height int) (vresult, error) {
	switch e.Kind() {
	case expr.KindCol:
		c, ok := cols[e.ColName()]
		if !ok {
			return vresult{}, fmt.Errorf("column %s not found", e.ColName())
		}
		return vresult{col: c}, nil
	case expr.KindLit:
		return vresult{lit: e.Value(), isLit: true}, nil
	case expr.KindCast:
		return evalCast(e, cols, height)
	case expr.KindUnary:
		return evalNot(e, cols, height)
	case expr.KindBin:
		return evalBinNode(e, cols, height)
	default:
		return vresult{}, fmt.Errorf("unsupported expr kind %s", e.Kind())
	}
}

// coalesceNode evaluates fill_null_expr / fill_nan_expr: the left operand is a
// column and the right operand is a float64 literal fill value. It produces a
// typed result column via the chunk coalesce kernels (reusing kernel 1.4),
// never round-tripping through []any.
func coalesceNode(op string, l, r vresult) (vresult, error) {
	if l.isLit || l.col == nil {
		return vresult{}, fmt.Errorf("%s target must be a column", op)
	}
	fill, ok := r.lit.(float64)
	if !r.isLit || !ok {
		return vresult{}, fmt.Errorf("%s value must be a float64 literal", op)
	}
	var out *chunk.Column
	var supported bool
	if op == "fill_null_expr" {
		out, supported = l.col.FillNullFloat64(fill)
	} else {
		out, supported = l.col.FillNaNFloat64(fill)
	}
	if !supported {
		return vresult{}, fmt.Errorf("%s requires a float64 column", op)
	}
	return vresult{col: out}, nil
}

// isInMask sets the bit of every row of c whose value is in list, mirroring
// the row-wise is_in: a value matches an element of the same Go type that
// equals it (a datetime, one denoting the same instant), and a null row
// matches when list holds nil. The result is never null.
func isInMask(c *chunk.Column, list []any, height int) (simd.Bitmap, error) {
	mask := simd.BitmapNew(height)
	nullsMatch := slices.Contains(list, nil)
	nulls := c.Nulls()
	switch c.DataType() {
	case dtypes.Int64:
		vals, _ := c.Int64s()
		markMembers(mask, vals, nulls, nullsMatch, memberSet(list, identity[int64]), identity[int64])
	case dtypes.Float64:
		// A NaN element never matches, as NaN != NaN; ±0 are one map key, as -0 == 0.
		vals, _ := c.Float64s()
		markMembers(mask, vals, nulls, nullsMatch, memberSet(list, identity[float64]), identity[float64])
	case dtypes.String, dtypes.Categorical, dtypes.Enum:
		vals, _ := c.Strings()
		markMembers(mask, vals, nulls, nullsMatch, memberSet(list, identity[string]), identity[string])
	case dtypes.Boolean:
		vals, _ := c.Bools()
		markMembers(mask, vals, nulls, nullsMatch, memberSet(list, identity[bool]), identity[bool])
	case dtypes.Datetime:
		vals, _ := c.Times()
		markMembers(mask, vals, nulls, nullsMatch, memberSet(list, chunk.TimeKeyOf), chunk.TimeKeyOf)
	default:
		return nil, fmt.Errorf("is_in: unsupported dtype %s", c.DataType())
	}
	return mask, nil
}

func identity[T any](v T) T { return v }

// memberSet returns the keys of the list elements of type T.
func memberSet[T any, K comparable](list []any, key func(T) K) map[K]struct{} {
	set := make(map[K]struct{}, len(list))
	for _, v := range list {
		if x, ok := v.(T); ok {
			set[key(x)] = struct{}{}
		}
	}
	return set
}

// markMembers sets the bit of every row whose key is in set, and of every null
// row when nullsMatch.
func markMembers[T any, K comparable](mask simd.Bitmap, vals []T, nulls []bool, nullsMatch bool, set map[K]struct{}, key func(T) K) {
	for i, v := range vals {
		if nulls != nil && nulls[i] {
			if nullsMatch {
				simd.BitmapSet(mask, i)
			}
			continue
		}
		if _, ok := set[key(v)]; ok {
			simd.BitmapSet(mask, i)
		}
	}
}

func evalBinNode(e expr.Expr, cols map[string]*chunk.Column, height int) (vresult, error) {
	l, err := evalNode(*e.Left(), cols, height)
	if err != nil {
		return vresult{}, err
	}
	r, err := evalNode(*e.Right(), cols, height)
	if err != nil {
		return vresult{}, err
	}
	op := e.Op()
	switch op {
	case "fill_null_expr", "fill_nan_expr":
		return coalesceNode(op, l, r)
	case "is_in":
		if l.isLit {
			return vresult{}, fmt.Errorf("is_in needs a column operand")
		}
		mask, err := isInMask(l.col, r.lit.([]any), height)
		if err != nil {
			return vresult{}, err
		}
		out := make([]bool, height)
		for i := range out {
			out[i] = simd.BitmapGet(mask, i)
		}
		return vresult{col: chunk.NewBool(out, nil)}, nil
	case "add", "sub", "mul", "div":
		return arithNode(op, l, r, height)
	case "gt", "ge", "lt", "le":
		if lr, lTag := numericReader(l); lTag != "other" {
			if rr, rTag := numericReader(r); rTag != "other" {
				return numericCompare(op, lr, lTag, rr, rTag, height), nil
			}
		}
		return boolBinNode(op, l, r, height)
	case "eq", "ne", "and", "or":
		return boolBinNode(op, l, r, height)
	case "mod", "floordiv", "pow":
		return floatBinNode(op, l, r, height)
	default:
		return vresult{}, fmt.Errorf("unsupported binary op %s", op)
	}
}

// numericReader exposes typed access to a numeric operand without per-element
// boxing. The returned tag is "int64", "float64", or "other".
type numericReaderT struct {
	isLit   bool
	isFloat bool
	i       []int64
	f       []float64
	nulls   []bool
	litI    int64
	litF    float64
}

func numericReader(v vresult) (numericReaderT, string) {
	if v.isLit {
		switch x := v.lit.(type) {
		case int64:
			return numericReaderT{isLit: true, litI: x, litF: float64(x)}, "int64"
		case float64:
			return numericReaderT{isLit: true, isFloat: true, litF: x}, "float64"
		default:
			return numericReaderT{}, "other"
		}
	}
	switch v.col.DataType() {
	case dtypes.Int64:
		s, _ := v.col.Int64s()
		return numericReaderT{i: s, nulls: v.col.Nulls()}, "int64"
	case dtypes.Float64:
		s, _ := v.col.Float64s()
		return numericReaderT{isFloat: true, f: s, nulls: v.col.Nulls()}, "float64"
	default:
		return numericReaderT{}, "other"
	}
}

func (n numericReaderT) nullAt(i int) bool {
	if n.isLit {
		return false
	}
	return n.nulls != nil && n.nulls[i]
}

func (n numericReaderT) intAt(i int) int64 {
	if n.isLit {
		return n.litI
	}
	return n.i[i]
}

func (n numericReaderT) floatAt(i int) float64 {
	if n.isLit {
		return n.litF
	}
	if n.isFloat {
		return n.f[i]
	}
	return float64(n.i[i])
}

// arithNode mirrors expr.arith: int64+int64 stays int64 (integer division),
// float64+float64 stays float64, any other type combination is an error, a null
// operand yields null, and division by zero is an error.
func arithNode(op string, l, r vresult, height int) (vresult, error) {
	lr, lTag := numericReader(l)
	rr, rTag := numericReader(r)
	switch {
	case lTag == "int64" && rTag == "int64":
		out := make([]int64, height)
		nulls := make([]bool, height)
		for i := range height {
			if lr.nullAt(i) || rr.nullAt(i) {
				nulls[i] = true
				continue
			}
			a, b := lr.intAt(i), rr.intAt(i)
			switch op {
			case "add":
				out[i] = a + b
			case "sub":
				out[i] = a - b
			case "mul":
				out[i] = a * b
			case "div":
				if b == 0 {
					return vresult{}, fmt.Errorf("division by zero")
				}
				out[i] = a / b
			}
		}
		return vresult{col: chunk.NewInt64(out, nulls)}, nil
	case lTag == "float64" && rTag == "float64":
		out := make([]float64, height)
		nulls := make([]bool, height)
		for i := range height {
			if lr.nullAt(i) || rr.nullAt(i) {
				nulls[i] = true
				continue
			}
			a, b := lr.floatAt(i), rr.floatAt(i)
			switch op {
			case "add":
				out[i] = a + b
			case "sub":
				out[i] = a - b
			case "mul":
				out[i] = a * b
			case "div":
				if b == 0 {
					return vresult{}, fmt.Errorf("division by zero")
				}
				out[i] = a / b
			}
		}
		return vresult{col: chunk.NewFloat64(out, nulls)}, nil
	default:
		return vresult{}, fmt.Errorf("arith type mismatch")
	}
}

// numericCompare evaluates gt/ge/lt/le on numeric operands. A null operand
// yields null (matching expr.EvalBin), carried in the result's null mask.
func numericCompare(op string, lr numericReaderT, lTag string, rr numericReaderT, rTag string, height int) vresult {
	// SIMD fast path: float64 column compared to a float64 literal threshold.
	if op == "gt" && lTag == "float64" && rTag == "float64" && !lr.isLit && rr.isLit {
		mask := simd.CompareGTFloat64(lr.f, rr.litF)
		return vresult{col: nullMaskedBool(mask, lr.nulls, height)}
	}
	out := make([]bool, height)
	nulls := make([]bool, height)
	useFloat := lTag == "float64" || rTag == "float64"
	for i := range height {
		if lr.nullAt(i) || rr.nullAt(i) {
			nulls[i] = true
			continue
		}
		if useFloat {
			a, b := lr.floatAt(i), rr.floatAt(i)
			if math.IsNaN(a) || math.IsNaN(b) {
				continue
			}
			out[i] = cmpOrdered(op, a, b)
		} else {
			out[i] = cmpOrdered(op, lr.intAt(i), rr.intAt(i))
		}
	}
	return vresult{col: chunk.NewBool(out, nulls)}
}

// nullMaskedBool adopts a SIMD comparison mask as a bool column in which every
// row null in operandNulls is null with a false value.
func nullMaskedBool(mask, operandNulls []bool, height int) *chunk.Column {
	nulls := make([]bool, height)
	if operandNulls != nil {
		for i := range mask {
			if operandNulls[i] {
				mask[i] = false
				nulls[i] = true
			}
		}
	}
	return chunk.NewBool(mask, nulls)
}

// boolBinNode handles eq/ne/and/or and non-numeric gt/ge/lt/le by applying
// expr.EvalBin per row. Comparisons propagate nulls (a null operand yields null)
// and and/or use three-valued Kleene logic, matching the row-wise evaluator.
func boolBinNode(op string, l, r vresult, height int) (vresult, error) {
	// SIMD fast path: int64 column == int64 literal (null operand -> null).
	if op == "eq" && !l.isLit && r.isLit && l.col.DataType() == dtypes.Int64 {
		if lit, ok := r.lit.(int64); ok {
			vals, _ := l.col.Int64s()
			mask := simd.CompareEQInt64(vals, lit)
			return vresult{col: nullMaskedBool(mask, l.col.Nulls(), height)}, nil
		}
	}
	// SIMD fast path: AND of two boolean columns with no nulls.
	if op == "and" && !l.isLit && !r.isLit &&
		l.col.DataType() == dtypes.Boolean && r.col.DataType() == dtypes.Boolean &&
		!hasNulls(l.col) && !hasNulls(r.col) {
		la, _ := l.col.Bools()
		rb, _ := r.col.Bools()
		return vresult{col: chunk.NewBool(simd.AndMask(la, rb), make([]bool, height))}, nil
	}
	out := make([]bool, height)
	nulls := make([]bool, height)
	for i := range height {
		lv := readScalar(l, i)
		rv := readScalar(r, i)
		res, err := expr.EvalBin(op, lv, rv)
		if err != nil {
			return vresult{}, err
		}
		if res == nil {
			nulls[i] = true
			continue
		}
		b, ok := res.(bool)
		if !ok {
			return vresult{}, fmt.Errorf("%s did not produce bool", op)
		}
		out[i] = b
	}
	return vresult{col: chunk.NewBool(out, nulls)}, nil
}

// floatBinNode handles mod/floordiv/pow which coerce to float64 and error on
// a null/non-numeric operand (matching the row-wise evaluator).
func floatBinNode(op string, l, r vresult, height int) (vresult, error) {
	out := make([]float64, height)
	for i := range height {
		lv := readScalar(l, i)
		rv := readScalar(r, i)
		res, err := expr.EvalBin(op, lv, rv)
		if err != nil {
			return vresult{}, err
		}
		f, ok := res.(float64)
		if !ok {
			return vresult{}, fmt.Errorf("%s did not produce float", op)
		}
		out[i] = f
	}
	return vresult{col: chunk.NewFloat64(out, make([]bool, height))}, nil
}

func evalNot(e expr.Expr, cols map[string]*chunk.Column, height int) (vresult, error) {
	child, err := evalNode(*e.Target(), cols, height)
	if err != nil {
		return vresult{}, err
	}
	out := make([]bool, height)
	nulls := make([]bool, height)
	for i := range height {
		sv := readScalar(child, i)
		if sv == nil {
			nulls[i] = true
			continue
		}
		b, ok := sv.(bool)
		if !ok {
			return vresult{}, fmt.Errorf("not expects bool")
		}
		out[i] = !b
	}
	return vresult{col: chunk.NewBool(out, nulls)}, nil
}

// castRows casts every row of child to target, into the typed buffer the
// target dtype is backed by. A null row stays null and leaves the buffer's zero
// value in place. T must be the type batchCast returns for target.
func castRows[T any](child vresult, target dtypes.DataType, height int, nulls []bool) ([]T, error) {
	out := make([]T, height)
	for i := range out {
		v := readScalar(child, i)
		if v == nil {
			nulls[i] = true
			continue
		}
		cv, err := batchCast(v, target)
		if err != nil {
			return nil, err
		}
		out[i] = cv.(T)
	}
	return out, nil
}

func evalCast(e expr.Expr, cols map[string]*chunk.Column, height int) (vresult, error) {
	child, err := evalNode(*e.Target(), cols, height)
	if err != nil {
		return vresult{}, err
	}
	target := e.CastType()
	nulls := make([]bool, height)
	// Each arm differs only in the buffer type and the constructor that adopts
	// it, so the per-row cast loop lives in castRows.
	switch target {
	case dtypes.Int64:
		out, err := castRows[int64](child, target, height, nulls)
		if err != nil {
			return vresult{}, err
		}
		return vresult{col: chunk.NewInt64(out, nulls)}, nil
	case dtypes.Float64:
		out, err := castRows[float64](child, target, height, nulls)
		if err != nil {
			return vresult{}, err
		}
		return vresult{col: chunk.NewFloat64(out, nulls)}, nil
	case dtypes.String:
		out, err := castRows[string](child, target, height, nulls)
		if err != nil {
			return vresult{}, err
		}
		return vresult{col: chunk.NewString(out, nulls)}, nil
	case dtypes.Boolean:
		out, err := castRows[bool](child, target, height, nulls)
		if err != nil {
			return vresult{}, err
		}
		return vresult{col: chunk.NewBool(out, nulls)}, nil
	case dtypes.Datetime:
		out, err := castRows[time.Time](child, target, height, nulls)
		if err != nil {
			return vresult{}, err
		}
		return vresult{col: chunk.NewTime(out, nulls)}, nil
	default:
		return vresult{}, fmt.Errorf("unsupported cast target %s", target)
	}
}

func hasNulls(c *chunk.Column) bool {
	return slices.Contains(c.Nulls(), true)
}

func readScalar(v vresult, i int) any {
	if v.isLit {
		return v.lit
	}
	return v.col.ValueAt(i)
}

func broadcast(lit any, height int) *chunk.Column {
	nulls := make([]bool, height)
	switch x := lit.(type) {
	case int64:
		return chunk.NewInt64(slices.Repeat([]int64{x}, height), nulls)
	case float64:
		return chunk.NewFloat64(slices.Repeat([]float64{x}, height), nulls)
	case string:
		return chunk.NewString(slices.Repeat([]string{x}, height), nulls)
	case bool:
		return chunk.NewBool(slices.Repeat([]bool{x}, height), nulls)
	case time.Time:
		return chunk.NewTime(slices.Repeat([]time.Time{x}, height), nulls)
	default:
		// nil or unknown literal -> all-null float column (matches all-nil infer)
		for i := range nulls {
			nulls[i] = true
		}
		return chunk.NewFloat64(make([]float64, height), nulls)
	}
}

// --- replicas of pkg/expr eval helpers (keep in sync with eval.go) ---

func batchCast(v any, dt dtypes.DataType) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch dt {
	case dtypes.Int64:
		switch t := v.(type) {
		case int64:
			return t, nil
		case float64:
			return int64(t), nil
		}
	case dtypes.Float64:
		switch t := v.(type) {
		case float64:
			return t, nil
		case int64:
			return float64(t), nil
		}
	case dtypes.String:
		return fmt.Sprintf("%v", v), nil
	case dtypes.Boolean:
		if b, ok := v.(bool); ok {
			return b, nil
		}
	case dtypes.Datetime:
		if t, ok := v.(time.Time); ok {
			return t, nil
		}
	}
	return nil, fmt.Errorf("cannot cast value")
}

// cmpOrdered applies one of the four ordering operators. An op outside the set
// (equality is handled by the caller) is false, not an error.
func cmpOrdered[T cmp.Ordered](op string, l, r T) bool {
	switch op {
	case "gt":
		return l > r
	case "ge":
		return l >= r
	case "lt":
		return l < r
	case "le":
		return l <= r
	}
	return false
}
