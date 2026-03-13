package series

func Int64Values(s Series) []int64 {
	out := make([]int64, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(int64)
	}
	return out
}

func Float64Values(s Series) []float64 {
	out := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(float64)
	}
	return out
}
