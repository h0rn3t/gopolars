package expr

// MapColumnNames returns a copy of e with every column reference's name rewritten
// by fn. The "*" wildcard column is left untouched. fn may return an error to
// abort the rewrite (e.g. an ambiguous or unknown column). The expression tree is
// rebuilt structurally so all other attributes (op, alias, dtype, literal value)
// are preserved.
func MapColumnNames(e Expr, fn func(name string) (string, error)) (Expr, error) {
	switch e.kind {
	case KindCol:
		if e.name == "*" {
			return e, nil
		}
		nn, err := fn(e.name)
		if err != nil {
			return Expr{}, err
		}
		out := e
		out.name = nn
		return out, nil
	case KindLit:
		return e, nil
	}
	out := e
	if e.target != nil {
		t, err := MapColumnNames(*e.target, fn)
		if err != nil {
			return Expr{}, err
		}
		out.target = &t
	}
	if e.left != nil {
		l, err := MapColumnNames(*e.left, fn)
		if err != nil {
			return Expr{}, err
		}
		out.left = &l
	}
	if e.right != nil {
		r, err := MapColumnNames(*e.right, fn)
		if err != nil {
			return Expr{}, err
		}
		out.right = &r
	}
	if e.extra != nil {
		x, err := MapColumnNames(*e.extra, fn)
		if err != nil {
			return Expr{}, err
		}
		out.extra = &x
	}
	return out, nil
}
