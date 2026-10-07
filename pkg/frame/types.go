package frame

import (
	"github.com/h0rn3t/gopolars/pkg/series"
)

type NewInput struct {
	Series []series.Series
}

// SortInput configures a sort. The sort is stable: rows with equal keys keep
// their input order.
type SortInput struct {
	By []string
	// Descending reverses the order of non-null values of the key at the same
	// index; a missing entry is ascending.
	Descending []bool
	// NullsLast places nulls after all values of every key, otherwise before
	// them, whatever the key's direction.
	NullsLast bool
	// MaintainOrder is accepted for Polars compatibility. It does not change the
	// result, since the sort always keeps equal keys in input order.
	MaintainOrder bool
}

type JoinType string

const (
	JoinTypeInner JoinType = "inner"
	JoinTypeLeft  JoinType = "left"
	JoinTypeRight JoinType = "right"
	JoinTypeFull  JoinType = "full"
	JoinTypeSemi  JoinType = "semi"
	JoinTypeAnti  JoinType = "anti"
	JoinTypeCross JoinType = "cross"
	JoinTypeAsof  JoinType = "asof"
)

type JoinInput struct {
	Other         DataFrame
	LeftOn        []string
	RightOn       []string
	How           JoinType
	Suffix        string
	AsofDirection string
	AsofTolerance int64
}
