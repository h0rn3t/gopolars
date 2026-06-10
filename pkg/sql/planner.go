package sql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func Plan(bound BoundQuery, catalog Catalog) ([]logical.Node, error) {
	if bound.Parsed.CTE != nil && bound.Parsed.CTEName != "" && bound.Parsed.Table == bound.Parsed.CTEName {
		innerBound := BoundQuery{Parsed: *bound.Parsed.CTE, TableName: bound.Parsed.CTE.Table}
		nodes, err := Plan(innerBound, catalog)
		if err != nil {
			return nil, err
		}
		outer := bound.Parsed
		outer.CTE = nil
		outer.CTEName = ""
		outerNodes, err := planSingle(outer, catalog)
		if err != nil {
			return nil, err
		}
		return append(nodes, outerNodes...), nil
	}
	if bound.Parsed.Subquery != nil {
		innerBound := BoundQuery{Parsed: *bound.Parsed.Subquery, TableName: bound.Parsed.Subquery.Table}
		nodes, err := Plan(innerBound, catalog)
		if err != nil {
			return nil, err
		}
		outer := bound.Parsed
		outer.Subquery = nil
		outerNodes, err := planSingle(outer, catalog)
		if err != nil {
			return nil, err
		}
		return append(nodes, outerNodes...), nil
	}
	return planSingle(bound.Parsed, catalog)
}

func planSingle(parsed ParsedQuery, catalog Catalog) ([]logical.Node, error) {
	nodes := make([]logical.Node, 0, 8)

	// Joins augment the source columns before any filter/projection.
	joinNodes, err := planJoins(parsed.From, catalog)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, joinNodes...)

	if parsed.HasWhere && parsed.Where != nil {
		nodes = append(nodes, logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{*parsed.Where}})
	}
	windowSpecs := make([]logical.WindowSpec, 0)
	for _, item := range parsed.Select {
		if item.Window == nil {
			continue
		}
		by := make([]string, 0, len(item.Window.OrderBy))
		desc := make([]bool, 0, len(item.Window.OrderBy))
		for _, o := range item.Window.OrderBy {
			by = append(by, o.Column)
			desc = append(desc, o.Desc)
		}
		windowSpecs = append(windowSpecs, logical.WindowSpec{
			Func:        item.Window.Func,
			Target:      item.Window.Target,
			Alias:       item.Window.Alias,
			PartitionBy: item.Window.PartitionBy,
			OrderBy:     by,
			Descending:  desc,
			Offset:      item.Window.Offset,
			Default:     item.Window.Default,
		})
	}
	if len(windowSpecs) > 0 {
		nodes = append(nodes, logical.Node{Type: logical.NodeWindow, Windows: windowSpecs})
	}
	// When a non-`*` projection would drop a source column that ORDER BY
	// references, carry that column through the projection and restore the
	// original output schema after the sort. SQL allows ORDER BY to reference
	// source columns not present in the SELECT list; without this the sort would
	// fail with "column not found". The GROUP BY / window / DISTINCT paths are
	// excluded (ordering by a dropped source column there is ill-defined).
	var carryCols, projOutputs []string
	if len(parsed.OrderBy) > 0 && len(parsed.GroupBy) == 0 && len(windowSpecs) == 0 && !parsed.Distinct {
		out := make(map[string]bool)
		hasProjection := false
		for _, item := range parsed.Select {
			if item.Expr.Kind() == expr.KindCol && item.Expr.ColName() == "*" {
				continue
			}
			hasProjection = true
			n := item.Expr.Name()
			projOutputs = append(projOutputs, n)
			out[n] = true
		}
		if hasProjection {
			seen := make(map[string]bool)
			for _, o := range parsed.OrderBy {
				if !out[o.Column] && !seen[o.Column] {
					carryCols = append(carryCols, o.Column)
					seen[o.Column] = true
				}
			}
		}
	}
	if len(parsed.GroupBy) > 0 {
		aggExprs := make([]expr.Expr, 0, len(parsed.Select))
		for _, item := range parsed.Select {
			if item.Expr.Kind() == expr.KindAgg {
				aggExprs = append(aggExprs, item.Expr)
			}
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeAggregate, Columns: parsed.GroupBy, Exprs: aggExprs})
		if parsed.Having != nil {
			nodes = append(nodes, logical.Node{Type: logical.NodeFilter, Exprs: []expr.Expr{*parsed.Having}})
		}
		selectExprs := make([]expr.Expr, 0, len(parsed.Select))
		for _, item := range parsed.Select {
			if item.Expr.Kind() == expr.KindAgg {
				selectExprs = append(selectExprs, expr.Col(item.Expr.Name()))
				continue
			}
			selectExprs = append(selectExprs, item.Expr)
		}
		if len(selectExprs) > 0 && (len(selectExprs) != 1 || selectExprs[0].Kind() != expr.KindCol || selectExprs[0].ColName() != "*") {
			nodes = append(nodes, logical.Node{Type: logical.NodeSelect, Exprs: selectExprs})
		}
	} else {
		selectExprs := make([]expr.Expr, 0, len(parsed.Select))
		for _, item := range parsed.Select {
			if item.Expr.Kind() == expr.KindCol && item.Expr.ColName() == "*" {
				continue
			}
			selectExprs = append(selectExprs, item.Expr)
		}
		for _, c := range carryCols {
			selectExprs = append(selectExprs, expr.Col(c))
		}
		if len(selectExprs) > 0 {
			nodes = append(nodes, logical.Node{Type: logical.NodeSelect, Exprs: selectExprs})
		}
	}
	if parsed.Distinct {
		nodes = append(nodes, logical.Node{Type: logical.NodeUnique})
	}
	if len(parsed.OrderBy) > 0 {
		cols := make([]string, 0, len(parsed.OrderBy))
		desc := make([]bool, 0, len(parsed.OrderBy))
		for _, o := range parsed.OrderBy {
			cols = append(cols, o.Column)
			desc = append(desc, o.Desc)
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeSort, Columns: cols, Descending: desc})
		if len(carryCols) > 0 {
			restore := make([]expr.Expr, 0, len(projOutputs))
			for _, n := range projOutputs {
				restore = append(restore, expr.Col(n))
			}
			nodes = append(nodes, logical.Node{Type: logical.NodeSelect, Exprs: restore})
		}
	}
	if parsed.Offset != nil {
		length := 1<<31 - 1
		if parsed.Limit != nil {
			length = *parsed.Limit
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeSlice, IntValue: *parsed.Offset, Strings: []string{strconv.Itoa(length)}})
	} else if parsed.Limit != nil {
		nodes = append(nodes, logical.Node{Type: logical.NodeLimit, IntValue: *parsed.Limit})
	}
	if parsed.SetOp != "" && parsed.SetRight != nil {
		rightNodes, err := planSingle(*parsed.SetRight, catalog)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeSetOp, Strings: []string{strings.TrimSpace(parsed.SetOp)}, Plan: rightNodes})
	}
	return nodes, nil
}

// planJoins emits a NodeJoin for each join in the FROM clause, resolving the
// right-side table from the catalog and deriving the join keys from the ON
// predicate.
func planJoins(from FromClause, catalog Catalog) ([]logical.Node, error) {
	if len(from.Joins) == 0 {
		return nil, nil
	}
	if catalog == nil {
		return nil, fmt.Errorf("joins require a table catalog")
	}
	nodes := make([]logical.Node, 0, len(from.Joins))
	for _, jc := range from.Joins {
		if jc.Table.Subquery != nil {
			return nil, fmt.Errorf("subquery join sources are not supported")
		}
		if jc.Table.Fn != nil {
			return nil, fmt.Errorf("table function %s is not resolved; execute through a SQLContext", jc.Table.Fn.Name)
		}
		other, ok := catalog.Resolve(jc.Table.Name)
		if !ok {
			return nil, fmt.Errorf("table %s is not registered", jc.Table.Name)
		}
		how, err := joinTypeOf(jc.Type)
		if err != nil {
			return nil, err
		}
		spec := &logical.JoinSpec{Other: other, How: how, Suffix: "_right"}
		if jc.Type != "cross" {
			if jc.On == nil {
				return nil, fmt.Errorf("%s join requires an ON predicate", jc.Type)
			}
			leftOn, rightOn, err := joinKeys(*jc.On, rightKeySet(jc.Table))
			if err != nil {
				return nil, err
			}
			spec.LeftOn = leftOn
			spec.RightOn = rightOn
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeJoin, Join: spec})
	}
	return nodes, nil
}

func joinTypeOf(t string) (frame.JoinType, error) {
	switch t {
	case "inner":
		return frame.JoinTypeInner, nil
	case "left":
		return frame.JoinTypeLeft, nil
	case "right":
		return frame.JoinTypeRight, nil
	case "full":
		return frame.JoinTypeFull, nil
	case "cross":
		return frame.JoinTypeCross, nil
	default:
		return "", fmt.Errorf("unsupported join type: %s", t)
	}
}

func rightKeySet(t TableRef) map[string]bool {
	keys := map[string]bool{}
	if t.Name != "" {
		keys[t.Name] = true
	}
	if t.Alias != "" {
		keys[t.Alias] = true
	}
	return keys
}

// joinKeys decomposes an ON predicate (equalities combined with AND) into the
// left-side and right-side key column names.
func joinKeys(on expr.Expr, rightKeys map[string]bool) ([]string, []string, error) {
	var leftOn, rightOn []string
	var walk func(e expr.Expr) error
	walk = func(e expr.Expr) error {
		if e.Kind() == expr.KindBin && e.Op() == "and" {
			if e.Left() == nil || e.Right() == nil {
				return fmt.Errorf("invalid AND in ON predicate")
			}
			if err := walk(*e.Left()); err != nil {
				return err
			}
			return walk(*e.Right())
		}
		if e.Kind() != expr.KindBin || e.Op() != "eq" {
			return fmt.Errorf("ON predicate supports only equality keys combined with AND")
		}
		l, r := e.Left(), e.Right()
		if l == nil || r == nil || l.Kind() != expr.KindCol || r.Kind() != expr.KindCol {
			return fmt.Errorf("ON predicate keys must be columns")
		}
		lq, lc := splitQualified(l.ColName())
		rq, rc := splitQualified(r.ColName())
		switch {
		case rightKeys[rq]:
			leftOn = append(leftOn, lc)
			rightOn = append(rightOn, rc)
		case rightKeys[lq]:
			leftOn = append(leftOn, rc)
			rightOn = append(rightOn, lc)
		default:
			// Unqualified keys: assume the same column name on both sides.
			leftOn = append(leftOn, lc)
			rightOn = append(rightOn, rc)
		}
		return nil
	}
	if err := walk(on); err != nil {
		return nil, nil, err
	}
	return leftOn, rightOn, nil
}

func splitQualified(name string) (string, string) {
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		return name[:dot], name[dot+1:]
	}
	return "", name
}
