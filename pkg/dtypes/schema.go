package dtypes

type Field struct {
	Name string
	Type DataType
}

type Schema []Field

func (s Schema) IndexOf(name string) int {
	for i, f := range s {
		if f.Name == name {
			return i
		}
	}
	return -1
}
