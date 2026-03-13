package series

func BoolValues(s Series) []bool {
	out := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(bool)
	}
	return out
}
