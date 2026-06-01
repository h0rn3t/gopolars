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
	"fmt"
	"math"
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
// fused-reduce / compress kernels walk it a word at a time.
//
// For the common predicate shapes — a numeric column compared to a literal,
// the AND of such predicates, or a bare boolean column — the Bitmap is produced
// directly by the simd compare kernels (CompareGTFloat64Bitmap /
// CompareEQInt64Bitmap / BitmapAnd) with a null operand folded to a 0 bit, so no
// intermediate []bool byte-mask is allocated. Any other shape falls back to the
// general column evaluator and packs the resulting boolean chunk into a Bitmap.
//
// The returned nulls slice is non-nil only on the fallback path, where it
// aliases the result chunk's validity buffer (read-only) so callers can detect a
// null-valued predicate result. Supported predicates never actually produce one
// (a comparison folds a null operand to false rather than null), so in practice
// it is all-false; the direct path returns nil. Either way the Bitmap already
// excludes null rows.
func (p *Plan) EvalBool(cols map[string]*chunk.Column, height int) (mask simd.Bitmap, nulls []bool, err error) {
	if bm, ok := evalBitmap(p.root, cols, height); ok {
		return bm, nil, nil
	}
	c, err := p.Eval(cols, height)
	if err != nil {
		return nil, nil, err
	}
	if c.DataType() != dtypes.Boolean {
		return nil, nil, fmt.Errorf("predicate did not evaluate to bool")
	}
	return packBoolColumn(c, height), c.Nulls(), nil
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
		// A nullable boolean used directly as a predicate has error semantics in
		// the row-wise evaluator (a null is not a bool); defer to the fallback so
		// that behavior is preserved exactly.
		if hasNulls(c) {
			return nil, false
		}
		return packBoolColumn(c, height), true
	case expr.KindBin:
		switch op := e.Op(); op {
		case "gt", "ge", "lt", "le", "eq":
			return cmpBitmap(op, e, cols, height)
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
	lr, lt := numericReader(l)
	rr, rt := numericReader(r)
	if lt == "other" || rt == "other" {
		return nil, false
	}
	if lr.isLit || !rr.isLit {
		return nil, false
	}
	switch {
	case op == "gt" && lt == "float64" && rt == "float64":
		b := simd.CompareGTFloat64Bitmap(lr.f, rr.litF)
		clearNullBits(b, lr.nulls, height)
		return b, true
	case op == "eq" && lt == "int64" && rt == "int64":
		b := simd.CompareEQInt64Bitmap(lr.i, rr.litI)
		clearNullBits(b, lr.nulls, height)
		return b, true
	case op == "eq":
		// float equality carries NaN subtleties handled by the row-wise path.
		return nil, false
	}
	// General gt/ge/lt/le over numeric column vs literal.
	b := simd.BitmapNew(height)
	useFloat := lt == "float64" || rt == "float64"
	for i := range height {
		if lr.nullAt(i) || rr.nullAt(i) {
			continue
		}
		if useFloat {
			a, c := lr.floatAt(i), rr.floatAt(i)
			if math.IsNaN(a) || math.IsNaN(c) {
				continue
			}
			if cmpFloat(op, a, c) {
				simd.BitmapSet(b, i)
			}
		} else if cmpInt(op, lr.intAt(i), rr.intAt(i)) {
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
			b[i>>6] |= 1 << (uint(i) & 63)
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
	case "add", "sub", "mul", "div":
		return arithNode(op, l, r, height)
	case "gt", "ge", "lt", "le":
		if lr, lt := numericReader(l); lt != "other" {
			if rr, rt := numericReader(r); rt != "other" {
				return numericCompare(op, lr, lt, rr, rt, height), nil
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
	lr, lt := numericReader(l)
	rr, rt := numericReader(r)
	switch {
	case lt == "int64" && rt == "int64":
		out := make([]int64, height)
		nulls := make([]bool, height)
		for i := 0; i < height; i++ {
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
	case lt == "float64" && rt == "float64":
		out := make([]float64, height)
		nulls := make([]bool, height)
		for i := 0; i < height; i++ {
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
// yields false (matching expr.compare), and the result never carries nulls.
func numericCompare(op string, lr numericReaderT, lt string, rr numericReaderT, rt string, height int) vresult {
	// SIMD fast path: float64 column compared to a float64 literal threshold.
	if op == "gt" && lt == "float64" && rt == "float64" && !lr.isLit && rr.isLit {
		mask := simd.CompareGTFloat64(lr.f, rr.litF)
		if lr.nulls != nil {
			for i := range mask {
				if lr.nulls[i] {
					mask[i] = false
				}
			}
		}
		return vresult{col: chunk.NewBool(mask, make([]bool, height))}
	}
	out := make([]bool, height)
	useFloat := lt == "float64" || rt == "float64"
	for i := 0; i < height; i++ {
		if lr.nullAt(i) || rr.nullAt(i) {
			continue
		}
		if useFloat {
			a, b := lr.floatAt(i), rr.floatAt(i)
			if math.IsNaN(a) || math.IsNaN(b) {
				continue
			}
			out[i] = cmpFloat(op, a, b)
		} else {
			out[i] = cmpInt(op, lr.intAt(i), rr.intAt(i))
		}
	}
	return vresult{col: chunk.NewBool(out, make([]bool, height))}
}

// boolBinNode handles eq/ne/and/or and non-numeric gt/ge/lt/le by replicating
// expr.evalBin per row. The result is always boolean with no nulls (these ops
// never return nil in the row-wise evaluator).
func boolBinNode(op string, l, r vresult, height int) (vresult, error) {
	// SIMD fast path: int64 column == int64 literal.
	if op == "eq" && !l.isLit && r.isLit && l.col.DataType() == dtypes.Int64 {
		if lit, ok := r.lit.(int64); ok {
			vals, _ := l.col.Int64s()
			mask := simd.CompareEQInt64(vals, lit)
			if nulls := l.col.Nulls(); nulls != nil {
				for i := range mask {
					if nulls[i] {
						mask[i] = false
					}
				}
			}
			return vresult{col: chunk.NewBool(mask, make([]bool, height))}, nil
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
	for i := 0; i < height; i++ {
		lv := readScalar(l, i)
		rv := readScalar(r, i)
		res, err := batchBin(op, lv, rv)
		if err != nil {
			return vresult{}, err
		}
		b, ok := res.(bool)
		if !ok {
			return vresult{}, fmt.Errorf("%s did not produce bool", op)
		}
		out[i] = b
	}
	return vresult{col: chunk.NewBool(out, make([]bool, height))}, nil
}

// floatBinNode handles mod/floordiv/pow which coerce to float64 and error on
// a null/non-numeric operand (matching the row-wise evaluator).
func floatBinNode(op string, l, r vresult, height int) (vresult, error) {
	out := make([]float64, height)
	for i := 0; i < height; i++ {
		lv := readScalar(l, i)
		rv := readScalar(r, i)
		res, err := batchBin(op, lv, rv)
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
	for i := 0; i < height; i++ {
		b, ok := readScalar(child, i).(bool)
		if !ok {
			return vresult{}, fmt.Errorf("not expects bool")
		}
		out[i] = !b
	}
	return vresult{col: chunk.NewBool(out, make([]bool, height))}, nil
}

func evalCast(e expr.Expr, cols map[string]*chunk.Column, height int) (vresult, error) {
	child, err := evalNode(*e.Target(), cols, height)
	if err != nil {
		return vresult{}, err
	}
	target := e.CastType()
	nulls := make([]bool, height)
	switch target {
	case dtypes.Int64:
		out := make([]int64, height)
		for i := 0; i < height; i++ {
			v := readScalar(child, i)
			if v == nil {
				nulls[i] = true
				continue
			}
			cv, err := batchCast(v, target)
			if err != nil {
				return vresult{}, err
			}
			out[i] = cv.(int64)
		}
		return vresult{col: chunk.NewInt64(out, nulls)}, nil
	case dtypes.Float64:
		out := make([]float64, height)
		for i := 0; i < height; i++ {
			v := readScalar(child, i)
			if v == nil {
				nulls[i] = true
				continue
			}
			cv, err := batchCast(v, target)
			if err != nil {
				return vresult{}, err
			}
			out[i] = cv.(float64)
		}
		return vresult{col: chunk.NewFloat64(out, nulls)}, nil
	case dtypes.String:
		out := make([]string, height)
		for i := 0; i < height; i++ {
			v := readScalar(child, i)
			if v == nil {
				nulls[i] = true
				continue
			}
			cv, err := batchCast(v, target)
			if err != nil {
				return vresult{}, err
			}
			out[i] = cv.(string)
		}
		return vresult{col: chunk.NewString(out, nulls)}, nil
	case dtypes.Boolean:
		out := make([]bool, height)
		for i := 0; i < height; i++ {
			v := readScalar(child, i)
			if v == nil {
				nulls[i] = true
				continue
			}
			cv, err := batchCast(v, target)
			if err != nil {
				return vresult{}, err
			}
			out[i] = cv.(bool)
		}
		return vresult{col: chunk.NewBool(out, nulls)}, nil
	case dtypes.Datetime:
		out := make([]time.Time, height)
		for i := 0; i < height; i++ {
			v := readScalar(child, i)
			if v == nil {
				nulls[i] = true
				continue
			}
			cv, err := batchCast(v, target)
			if err != nil {
				return vresult{}, err
			}
			out[i] = cv.(time.Time)
		}
		return vresult{col: chunk.NewTime(out, nulls)}, nil
	default:
		return vresult{}, fmt.Errorf("unsupported cast target %s", target)
	}
}

func hasNulls(c *chunk.Column) bool {
	nulls := c.Nulls()
	if nulls == nil {
		return false
	}
	for _, n := range nulls {
		if n {
			return true
		}
	}
	return false
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
		out := make([]int64, height)
		for i := range out {
			out[i] = x
		}
		return chunk.NewInt64(out, nulls)
	case float64:
		out := make([]float64, height)
		for i := range out {
			out[i] = x
		}
		return chunk.NewFloat64(out, nulls)
	case string:
		out := make([]string, height)
		for i := range out {
			out[i] = x
		}
		return chunk.NewString(out, nulls)
	case bool:
		out := make([]bool, height)
		for i := range out {
			out[i] = x
		}
		return chunk.NewBool(out, nulls)
	case time.Time:
		out := make([]time.Time, height)
		for i := range out {
			out[i] = x
		}
		return chunk.NewTime(out, nulls)
	default:
		// nil or unknown literal -> all-null float column (matches all-nil infer)
		for i := range nulls {
			nulls[i] = true
		}
		return chunk.NewFloat64(make([]float64, height), nulls)
	}
}

// --- replicas of pkg/expr eval helpers (keep in sync with eval.go) ---

func batchBin(op string, left, right any) (any, error) {
	switch op {
	case "eq":
		if left == nil || right == nil {
			return left == nil && right == nil, nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return false, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return false, nil
		}
		return left == right, nil
	case "ne":
		if left == nil || right == nil {
			return left != nil || right != nil, nil
		}
		if lf, ok := left.(float64); ok && math.IsNaN(lf) {
			return true, nil
		}
		if rf, ok := right.(float64); ok && math.IsNaN(rf) {
			return true, nil
		}
		return left != right, nil
	case "gt", "ge", "lt", "le":
		return compareAny(op, left, right)
	case "and":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("and expects bool")
		}
		return l && r, nil
	case "or":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, fmt.Errorf("or expects bool")
		}
		return l || r, nil
	case "floordiv":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, fmt.Errorf("floordiv expects numeric non-zero divisor")
		}
		return math.Floor(lf / rf), nil
	case "mod":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, fmt.Errorf("mod expects numeric non-zero divisor")
		}
		return math.Mod(lf, rf), nil
	case "pow":
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok {
			return nil, fmt.Errorf("pow expects numeric")
		}
		return math.Pow(lf, rf), nil
	default:
		return nil, fmt.Errorf("unsupported binary op %s", op)
	}
}

func compareAny(op string, left, right any) (any, error) {
	if left == nil || right == nil {
		return false, nil
	}
	switch l := left.(type) {
	case int64:
		r, ok := right.(int64)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		return cmpInt(op, l, r), nil
	case float64:
		r, ok := right.(float64)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		if math.IsNaN(l) || math.IsNaN(r) {
			return false, nil
		}
		return cmpFloat(op, l, r), nil
	case string:
		r, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		return cmpString(op, l, r), nil
	case time.Time:
		r, ok := right.(time.Time)
		if !ok {
			return false, fmt.Errorf("compare type mismatch")
		}
		switch op {
		case "gt":
			return l.After(r), nil
		case "ge":
			return l.After(r) || l.Equal(r), nil
		case "lt":
			return l.Before(r), nil
		case "le":
			return l.Before(r) || l.Equal(r), nil
		}
	}
	return false, fmt.Errorf("unsupported compare types")
}

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

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func cmpInt(op string, l, r int64) bool {
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

func cmpFloat(op string, l, r float64) bool {
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

func cmpString(op string, l, r string) bool {
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
