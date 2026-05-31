package frame

import (
	"github.com/h0rn3t/gopolars/pkg/series"
)

type NewInput struct {
	Series []series.Series
}

type SortInput struct {
	By            []string
	Descending    []bool
	NullsLast     bool
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
