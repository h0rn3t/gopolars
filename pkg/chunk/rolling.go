package chunk

import "math"

// Linear-time rolling-window kernels. Each computes a trailing window aggregate
// over a typed []float64 plus an optional validity mask (nulls; nil == no
// nulls) in O(n) time independent of the window size w, writing a single
// preallocated output buffer plus O(w) working state.
//
// The start of the series uses an expanding window (positions clamped to 0),
// matching gopolars' established rolling behavior. minPeriods is the minimum
// number of observations a window must contain to emit a value; with the
// default of 1, a position is null only when the window has no observations.
//
// skipNaN selects between the two historical NaN policies: the DataFrame
// expression path included NaN observations (so a NaN propagates into the
// sum/mean), whereas the Series facade excluded them. min/max always exclude
// NaN (a NaN can never be the running extremum).

// rollSumState is a running Neumaier-compensated accumulator over the non-null,
// finite values currently inside the window, plus counts used for null and NaN
// handling. Compensated summation bounds the floating-point drift of the
// add/subtract running sum on mixed-sign data.
type rollSumState struct {
	sum        float64
	comp       float64
	validCount int // non-null observations in the window
	nanCount   int // non-null NaN observations in the window
}

func (s *rollSumState) add(v float64, isNull bool) {
	if isNull {
		return
	}
	s.validCount++
	if math.IsNaN(v) {
		s.nanCount++
		return
	}
	t := s.sum + v
	if math.Abs(s.sum) >= math.Abs(v) {
		s.comp += (s.sum - t) + v
	} else {
		s.comp += (v - t) + s.sum
	}
	s.sum = t
}

func (s *rollSumState) remove(v float64, isNull bool) {
	if isNull {
		return
	}
	s.validCount--
	if math.IsNaN(v) {
		s.nanCount--
		return
	}
	nv := -v
	t := s.sum + nv
	if math.Abs(s.sum) >= math.Abs(nv) {
		s.comp += (s.sum - t) + nv
	} else {
		s.comp += (nv - t) + s.sum
	}
	s.sum = t
}

func (s *rollSumState) total() float64 { return s.sum + s.comp }

func normWindowParams(window, minPeriods int) (int, int) {
	if window <= 0 {
		window = 1
	}
	if minPeriods < 1 {
		minPeriods = 1
	}
	return window, minPeriods
}

// RollingSum computes the trailing-window sum. Returns the result values and a
// validity mask (true == null).
func RollingSum(values []float64, nulls []bool, window, minPeriods int, skipNaN bool) ([]float64, []bool) {
	window, minPeriods = normWindowParams(window, minPeriods)
	n := len(values)
	out := make([]float64, n)
	outNulls := make([]bool, n)
	var st rollSumState
	isNull := func(i int) bool { return nulls != nil && nulls[i] }
	for i := 0; i < n; i++ {
		st.add(values[i], isNull(i))
		if i >= window {
			st.remove(values[i-window], isNull(i-window))
		}
		obs := st.validCount
		if skipNaN {
			obs -= st.nanCount
		}
		if obs < minPeriods {
			outNulls[i] = true
			continue
		}
		if !skipNaN && st.nanCount > 0 {
			out[i] = math.NaN()
			continue
		}
		out[i] = st.total()
	}
	return out, outNulls
}

// RollingMean computes the trailing-window mean (sum / observation count).
func RollingMean(values []float64, nulls []bool, window, minPeriods int, skipNaN bool) ([]float64, []bool) {
	window, minPeriods = normWindowParams(window, minPeriods)
	n := len(values)
	out := make([]float64, n)
	outNulls := make([]bool, n)
	var st rollSumState
	isNull := func(i int) bool { return nulls != nil && nulls[i] }
	for i := 0; i < n; i++ {
		st.add(values[i], isNull(i))
		if i >= window {
			st.remove(values[i-window], isNull(i-window))
		}
		obs := st.validCount
		if skipNaN {
			obs -= st.nanCount
		}
		if obs < minPeriods {
			outNulls[i] = true
			continue
		}
		if !skipNaN && st.nanCount > 0 {
			out[i] = math.NaN()
			continue
		}
		out[i] = st.total() / float64(obs)
	}
	return out, outNulls
}

// RollingMin computes the trailing-window minimum via a monotonic-index deque.
func RollingMin(values []float64, nulls []bool, window, minPeriods int) ([]float64, []bool) {
	return rollingExtreme(values, nulls, window, minPeriods, true)
}

// RollingMax computes the trailing-window maximum via a monotonic-index deque.
func RollingMax(values []float64, nulls []bool, window, minPeriods int) ([]float64, []bool) {
	return rollingExtreme(values, nulls, window, minPeriods, false)
}

func rollingExtreme(values []float64, nulls []bool, window, minPeriods int, isMin bool) ([]float64, []bool) {
	window, minPeriods = normWindowParams(window, minPeriods)
	n := len(values)
	out := make([]float64, n)
	outNulls := make([]bool, n)
	// deque holds indices of valid observations whose values are monotonic; the
	// front is always the current extremum within the window.
	deque := make([]int, 0, window+1)
	head := 0 // logical front index into deque (avoids reslicing churn)
	validCount := 0
	isValid := func(i int) bool {
		return (nulls == nil || !nulls[i]) && !math.IsNaN(values[i])
	}
	for i := 0; i < n; i++ {
		if isValid(i) {
			validCount++
			for len(deque) > head {
				last := deque[len(deque)-1]
				if (isMin && values[last] >= values[i]) || (!isMin && values[last] <= values[i]) {
					deque = deque[:len(deque)-1]
				} else {
					break
				}
			}
			deque = append(deque, i)
		}
		if i >= window && isValid(i-window) {
			validCount--
		}
		lo := i - window + 1
		if lo < 0 {
			lo = 0
		}
		for len(deque) > head && deque[head] < lo {
			head++
		}
		if validCount < minPeriods {
			outNulls[i] = true
			continue
		}
		out[i] = values[deque[head]]
	}
	return out, outNulls
}
