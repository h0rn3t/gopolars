package chunk

import (
	"math"
	"math/rand"
	"testing"
)

// refRolling is an O(n·w) full-window recompute reference used to validate the
// linear kernels. agg is one of "sum","mean","min","max". It mirrors the kernel
// semantics: expanding window at the start, min_periods, skipNaN policy.
func refRolling(agg string, values []float64, nulls []bool, window, minPeriods int, skipNaN bool) ([]float64, []bool) {
	n := len(values)
	out := make([]float64, n)
	outNulls := make([]bool, n)
	for i := 0; i < n; i++ {
		lo := i - window + 1
		if lo < 0 {
			lo = 0
		}
		obs := 0
		sum := 0.0
		nan := false
		var ext float64
		hasExt := false
		for j := lo; j <= i; j++ {
			if nulls != nil && nulls[j] {
				continue
			}
			v := values[j]
			if math.IsNaN(v) {
				if skipNaN {
					continue
				}
				nan = true
				obs++ // counted as an observation in the include-NaN policy
				continue
			}
			obs++
			sum += v
			if !hasExt {
				ext = v
				hasExt = true
			} else if (agg == "min" && v < ext) || (agg == "max" && v > ext) {
				ext = v
			}
		}
		// min/max use a finite-only observation count.
		effObs := obs
		if agg == "min" || agg == "max" {
			effObs = 0
			for j := lo; j <= i; j++ {
				if nulls != nil && nulls[j] {
					continue
				}
				if math.IsNaN(values[j]) {
					continue
				}
				effObs++
			}
		}
		if effObs < minPeriods {
			outNulls[i] = true
			continue
		}
		switch agg {
		case "sum":
			if !skipNaN && nan {
				out[i] = math.NaN()
			} else {
				out[i] = sum
			}
		case "mean":
			if !skipNaN && nan {
				out[i] = math.NaN()
			} else {
				out[i] = sum / float64(obs)
			}
		case "min", "max":
			out[i] = ext
		}
	}
	return out, outNulls
}

func approxEqual(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	d := math.Abs(a - b)
	if d <= tol {
		return true
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	return d <= tol*scale
}

func compareRolling(t *testing.T, name string, gotV []float64, gotN []bool, wantV []float64, wantN []bool, exact bool) {
	t.Helper()
	for i := range wantV {
		if gotN[i] != wantN[i] {
			t.Errorf("%s[%d] null = %v, want %v", name, i, gotN[i], wantN[i])
			continue
		}
		if wantN[i] {
			continue
		}
		ok := false
		if exact {
			ok = gotV[i] == wantV[i] || (math.IsNaN(gotV[i]) && math.IsNaN(wantV[i]))
		} else {
			ok = approxEqual(gotV[i], wantV[i], 1e-9)
		}
		if !ok {
			t.Errorf("%s[%d] = %v, want %v", name, i, gotV[i], wantV[i])
		}
	}
}

func TestRollingSumMeanMixedSignParity(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	n := 5000
	vals := make([]float64, n)
	for i := range vals {
		// Mixed-sign, large magnitude to stress compensated summation.
		vals[i] = (r.Float64()*2 - 1) * 1e8
	}
	for _, w := range []int{1, 7, 100, 999} {
		gotS, gotSN := RollingSum(vals, nil, w, 1, false)
		wantS, wantSN := refRolling("sum", vals, nil, w, 1, false)
		compareRolling(t, "sum", gotS, gotSN, wantS, wantSN, false)

		gotM, gotMN := RollingMean(vals, nil, w, 1, false)
		wantM, wantMN := refRolling("mean", vals, nil, w, 1, false)
		compareRolling(t, "mean", gotM, gotMN, wantM, wantMN, false)
	}
}

func TestRollingMinMaxExactParity(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	n := 3000
	vals := make([]float64, n)
	nulls := make([]bool, n)
	for i := range vals {
		vals[i] = r.Float64()*200 - 100
		if r.Float32() < 0.15 {
			nulls[i] = true
		}
	}
	for _, w := range []int{1, 5, 64, 500} {
		gotMin, gotMinN := RollingMin(vals, nulls, w, 1)
		wantMin, wantMinN := refRolling("min", vals, nulls, w, 1, true)
		compareRolling(t, "min", gotMin, gotMinN, wantMin, wantMinN, true)

		gotMax, gotMaxN := RollingMax(vals, nulls, w, 1)
		wantMax, wantMaxN := refRolling("max", vals, nulls, w, 1, true)
		compareRolling(t, "max", gotMax, gotMaxN, wantMax, wantMaxN, true)
	}
}

func TestRollingNullAndMinPeriods(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 5}
	nulls := []bool{true, true, false, false, false}
	// window 3, min_periods 2: positions with <2 valid obs are null.
	got, gotN := RollingSum(vals, nulls, 3, 2, true)
	want, wantN := refRolling("sum", vals, nulls, 3, 2, true)
	compareRolling(t, "sum", got, gotN, want, wantN, false)
	// Position 0: 0 valid -> null; pos1: 0 valid -> null; pos2: window[0..2]=1 valid -> null;
	// pos3: window[1..3]={null,3,4}=2 valid -> 7; pos4: window[2..4]={3,4,5}=3 valid -> 12.
	if !gotN[0] || !gotN[1] || !gotN[2] {
		t.Errorf("expected nulls for positions with <2 valid obs: %v", gotN)
	}
	if gotN[3] || got[3] != 7 {
		t.Errorf("pos3 = (%v,null=%v), want 7", got[3], gotN[3])
	}
	if gotN[4] || got[4] != 12 {
		t.Errorf("pos4 = (%v,null=%v), want 12", got[4], gotN[4])
	}
}

func TestRollingSumNaNPropagation(t *testing.T) {
	vals := []float64{1, math.NaN(), 3, 4}
	// include-NaN policy: any NaN in window -> NaN output.
	got, gotN := RollingSum(vals, nil, 2, 1, false)
	if gotN[1] || !math.IsNaN(got[1]) {
		t.Errorf("window containing NaN should output NaN, got %v", got[1])
	}
	// skip-NaN policy: NaN excluded.
	gotSkip, _ := RollingSum(vals, nil, 2, 1, true)
	if gotSkip[1] != 1 { // window {1, NaN-excluded} -> 1
		t.Errorf("skipNaN sum[1] = %v, want 1", gotSkip[1])
	}
}
