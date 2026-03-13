package polars

import "strings"

type ChangeClass string

const (
	ChangeClassCompatible  ChangeClass = "compatible"
	ChangeClassDeprecating ChangeClass = "deprecating"
	ChangeClassBreaking    ChangeClass = "breaking"
)

func ClassifyChange(description string) ChangeClass {
	lower := strings.ToLower(description)
	switch {
	case strings.Contains(lower, "breaking"), strings.Contains(lower, "remove "), strings.Contains(lower, "drop "):
		return ChangeClassBreaking
	case strings.Contains(lower, "deprecat"):
		return ChangeClassDeprecating
	default:
		return ChangeClassCompatible
	}
}
