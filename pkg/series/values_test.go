package series

import "time"

// seriesValues materializes a Series into a typed slice, leaving the zero value
// in null positions. Test-only helper used across the series tests.
func seriesValues[T any](s Series) []T {
	out := make([]T, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		out[i], _ = s.Value(i).(T)
	}
	return out
}

func Int64Values(s Series) []int64        { return seriesValues[int64](s) }
func Float64Values(s Series) []float64    { return seriesValues[float64](s) }
func StringValues(s Series) []string      { return seriesValues[string](s) }
func BoolValues(s Series) []bool          { return seriesValues[bool](s) }
func DatetimeValues(s Series) []time.Time { return seriesValues[time.Time](s) }
