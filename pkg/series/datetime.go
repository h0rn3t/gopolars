package series

import "time"

func DatetimeValues(s Series) []time.Time {
	out := make([]time.Time, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(time.Time)
	}
	return out
}
