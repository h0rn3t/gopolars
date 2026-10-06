package dtypes

import "slices"

type Field struct {
	Name string
	Type DataType
}

type Schema []Field

func (s Schema) IndexOf(name string) int {
	return slices.IndexFunc(s, func(f Field) bool { return f.Name == name })
}
