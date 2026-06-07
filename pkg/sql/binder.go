package sql

import (
	"fmt"
	"strings"

	"github.com/h0rn3t/gopolars/pkg/expr"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// Catalog resolves a registered table name to its DataFrame. It is used by the
// binder (to learn table schemas for qualified-column resolution) and by the
// planner (to embed join right-side frames).
type Catalog interface {
	Resolve(name string) (frame.DataFrame, bool)
}

// tableCatalog is the default map-backed Catalog.
type tableCatalog map[string]frame.DataFrame

func (c tableCatalog) Resolve(name string) (frame.DataFrame, bool) {
	d, ok := c[name]
	return d, ok
}

// NewCatalog builds a Catalog from a name->DataFrame map.
func NewCatalog(tables map[string]frame.DataFrame) Catalog {
	c := make(tableCatalog, len(tables))
	for k, v := range tables {
		c[k] = v
	}
	return c
}

type BoundQuery struct {
	Parsed    ParsedQuery
	TableName string
}

// Bind validates the parsed query and, when a catalog with resolvable table
// schemas is available, rewrites qualified/ambiguous column references in the
// SELECT/WHERE/HAVING/GROUP BY/ORDER BY clauses to their physical post-join
// column names.
func Bind(parsed ParsedQuery, catalog Catalog) (BoundQuery, error) {
	for _, item := range parsed.Select {
		if item.Window == nil {
			continue
		}
		if len(item.Window.PartitionBy) == 0 && len(item.Window.OrderBy) == 0 {
			return BoundQuery{}, fmt.Errorf("window expression requires PARTITION BY or ORDER BY")
		}
	}
	resolved, err := resolveQueryColumns(parsed, catalog)
	if err != nil {
		return BoundQuery{}, err
	}
	return BoundQuery{
		Parsed:    resolved,
		TableName: resolved.Table,
	}, nil
}

// resolveQueryColumns rewrites column references using table schemas from the
// catalog. If any referenced table cannot be resolved (e.g. a CTE or subquery
// source) and there are no joins, resolution is skipped and the query is left
// untouched (single-table column names need no rewriting).
func resolveQueryColumns(parsed ParsedQuery, catalog Catalog) (ParsedQuery, error) {
	if catalog == nil {
		return parsed, nil
	}
	refs := append([]TableRef{parsed.From.Primary}, joinTableRefs(parsed.From.Joins)...)
	hasJoins := len(parsed.From.Joins) > 0

	type tableInfo struct {
		key  string
		cols []string
	}
	infos := make([]tableInfo, 0, len(refs))
	for _, tr := range refs {
		if tr.Subquery != nil {
			// Subquery sources have no catalog schema; skip column resolution.
			return parsed, nil
		}
		df, ok := catalog.Resolve(tr.Name)
		if !ok {
			if hasJoins {
				return ParsedQuery{}, fmt.Errorf("table %s is not registered", tr.Name)
			}
			return parsed, nil
		}
		infos = append(infos, tableInfo{key: tr.key(), cols: df.Columns()})
	}

	physical := make(map[string]map[string]string)
	owners := make(map[string][]string)
	accumulated := make(map[string]bool)
	for _, info := range infos {
		physical[info.key] = make(map[string]string, len(info.cols))
		for _, col := range info.cols {
			phys := col
			if accumulated[col] {
				phys = col + "_right"
			}
			physical[info.key][col] = phys
			accumulated[phys] = true
			owners[col] = append(owners[col], info.key)
		}
	}

	resolver := func(name string) (string, error) {
		if dot := strings.IndexByte(name, '.'); dot >= 0 {
			alias := name[:dot]
			col := name[dot+1:]
			m, ok := physical[alias]
			if !ok {
				return "", fmt.Errorf("unknown table qualifier %q in column %q", alias, name)
			}
			phys, ok := m[col]
			if !ok {
				return "", fmt.Errorf("column %q not found", name)
			}
			return phys, nil
		}
		owns := owners[name]
		switch len(owns) {
		case 0:
			// Not a base column (alias / computed); leave as-is.
			return name, nil
		case 1:
			return physical[owns[0]][name], nil
		default:
			return "", fmt.Errorf("ambiguous column: %s", name)
		}
	}

	out := parsed
	if out.Where != nil {
		e, err := expr.MapColumnNames(*out.Where, resolver)
		if err != nil {
			return ParsedQuery{}, err
		}
		out.Where = &e
	}
	if out.Having != nil {
		e, err := expr.MapColumnNames(*out.Having, resolver)
		if err != nil {
			return ParsedQuery{}, err
		}
		out.Having = &e
	}
	items := make([]SelectItem, len(out.Select))
	for i, item := range out.Select {
		if item.Window != nil {
			items[i] = item
			continue
		}
		e, err := expr.MapColumnNames(item.Expr, resolver)
		if err != nil {
			return ParsedQuery{}, err
		}
		items[i] = SelectItem{Expr: e}
	}
	out.Select = items
	if len(out.GroupBy) > 0 {
		groups := make([]string, len(out.GroupBy))
		for i, g := range out.GroupBy {
			ng, err := resolver(g)
			if err != nil {
				return ParsedQuery{}, err
			}
			groups[i] = ng
		}
		out.GroupBy = groups
	}
	if len(out.OrderBy) > 0 {
		order := make([]OrderItem, len(out.OrderBy))
		for i, o := range out.OrderBy {
			nc, err := resolver(o.Column)
			if err != nil {
				return ParsedQuery{}, err
			}
			order[i] = OrderItem{Column: nc, Desc: o.Desc}
		}
		out.OrderBy = order
	}
	return out, nil
}

func joinTableRefs(joins []JoinClause) []TableRef {
	refs := make([]TableRef, 0, len(joins))
	for _, j := range joins {
		refs = append(refs, j.Table)
	}
	return refs
}
