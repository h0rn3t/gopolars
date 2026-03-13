package series

func StringValues(s Series) []string {
	out := make([]string, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(string)
	}
	return out
}
