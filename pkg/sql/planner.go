package sql

import (
	"strings"

	"github.com/eugeneshershen/gopolars/pkg/expr"
	"github.com/eugeneshershen/gopolars/pkg/plan/logical"
)

func Plan(bound BoundQuery) []logical.Node {
	if bound.Parsed.CTE != nil && bound.Parsed.CTEName != "" && bound.Parsed.Table == bound.Parsed.CTEName {
		innerBound := BoundQuery{Parsed: *bound.Parsed.CTE, TableName: bound.Parsed.CTE.Table}
		nodes := Plan(innerBound)
		outer := bound.Parsed
		outer.CTE = nil
		outer.CTEName = ""
		nodes = append(nodes, planSingle(outer)...)
		return nodes
	}
	if bound.Parsed.Subquery != nil {
		innerBound := BoundQuery{Parsed: *bound.Parsed.Subquery, TableName: bound.Parsed.Subquery.Table}
		nodes := Plan(innerBound)
		outer := bound.Parsed
		outer.Subquery = nil
		nodes = append(nodes, planSingle(outer)...)
		return nodes
	}
	return planSingle(bound.Parsed)
}

func planSingle(parsed ParsedQuery) []logical.Node {
	nodes := make([]logical.Node, 0, 8)
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
		})
	}
	if len(windowSpecs) > 0 {
		nodes = append(nodes, logical.Node{Type: logical.NodeWindow, Windows: windowSpecs})
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
		if len(selectExprs) > 0 && !(len(selectExprs) == 1 && selectExprs[0].Kind() == expr.KindCol && selectExprs[0].ColName() == "*") {
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
		if len(selectExprs) > 0 {
			nodes = append(nodes, logical.Node{Type: logical.NodeSelect, Exprs: selectExprs})
		}
	}
	if len(parsed.OrderBy) > 0 {
		cols := make([]string, 0, len(parsed.OrderBy))
		desc := make([]bool, 0, len(parsed.OrderBy))
		for _, o := range parsed.OrderBy {
			cols = append(cols, o.Column)
			desc = append(desc, o.Desc)
		}
		nodes = append(nodes, logical.Node{Type: logical.NodeSort, Columns: cols, Descending: desc})
	}
	if parsed.Limit != nil {
		nodes = append(nodes, logical.Node{Type: logical.NodeLimit, IntValue: *parsed.Limit})
	}
	if parsed.SetOp != "" && parsed.SetRight != nil {
		rightNodes := planSingle(*parsed.SetRight)
		nodes = append(nodes, logical.Node{Type: logical.NodeSetOp, Strings: []string{strings.TrimSpace(parsed.SetOp)}, Plan: rightNodes})
	}
	return nodes
}
