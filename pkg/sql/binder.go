package sql

import "fmt"

type BoundQuery struct {
	Parsed    ParsedQuery
	TableName string
}

func Bind(parsed ParsedQuery) (BoundQuery, error) {
	for _, item := range parsed.Select {
		if item.Window == nil {
			continue
		}
		if len(item.Window.PartitionBy) == 0 && len(item.Window.OrderBy) == 0 {
			return BoundQuery{}, fmt.Errorf("window expression requires PARTITION BY or ORDER BY")
		}
	}
	return BoundQuery{
		Parsed:    parsed,
		TableName: parsed.Table,
	}, nil
}
