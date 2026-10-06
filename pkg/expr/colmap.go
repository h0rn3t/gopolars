package expr

// MapColumnNames returns a copy of e with every column reference's name rewritten
// by fn. The "*" wildcard column is left untouched. fn may return an error to
// abort the rewrite (e.g. an ambiguous or unknown column). The expression tree is
// rebuilt structurally so all other attributes (op, alias, dtype, literal value)
// are preserved.
func MapColumnNames(e Expr, fn func(name string) (string, error)) (Expr, error) {
	switch e.kind {
	case KindCol:
		if e.name == selectorAll {
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
	for _, slot := range []**Expr{&out.target, &out.left, &out.right, &out.extra} {
		if *slot != nil {
			child, err := MapColumnNames(**slot, fn)
			if err != nil {
				return Expr{}, err
			}
			*slot = &child
		}
	}
	return out, nil
}
