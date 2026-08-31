//go:build !goexperiment.simd

package simd

// Default build: the portable vector kernels in vec_simd.go are not compiled, so
// every hook declines and its caller uses the scalar body. These one-line
// bodies inline to a constant false, which lets the compiler delete the whole
// vector branch at each call site — the default build is byte-for-byte the code
// it was before the hooks were introduced.

func sumWhereFloat64Vec([]float64, Cmp, float64) (float64, int, bool) { return 0, 0, false }

func minMaxWhereFloat64Vec([]float64, Cmp, float64) (float64, float64, int, bool) {
	return 0, 0, 0, false
}

func minFloat64Vec([]float64) (float64, bool) { return 0, false }

func maxFloat64Vec([]float64) (float64, bool) { return 0, false }

func minMaxFloat64Vec([]float64) (float64, float64, bool) { return 0, 0, false }
