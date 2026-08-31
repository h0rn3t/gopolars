//go:build goexperiment.simd

package simd

import "simd"

// Portable hardware-SIMD kernels, built only under GOEXPERIMENT=simd (Go 1.27+).
//
// The stdlib simd package is vector-length-agnostic: simd.Float64s is backed by
// NEON on arm64, AVX/AVX2/AVX512 on amd64, and a pure-Go emulation elsewhere,
// with the lane count fixed for a program execution and read back via Len(). One
// body of Go therefore covers every architecture this module builds for.
//
// Every function here is a *hook*: it returns ok=false when it declines the
// input (too short for its accumulator set, or a vector wider than maxLanes),
// and the untagged caller in kernels.go / kernels_amd64.go / simd_generic.go
// falls through to the scalar body. vec_generic.go supplies the same signatures
// returning ok=false, so correctness never depends on this file being compiled.
//
// Two rules govern the code below; both are load-bearing, see the change's
// design.md (D2-D4):
//
//   - Comparisons are switched on the loop-invariant Cmp *inline* in the loop
//     body, never behind a func value. The branch is perfectly predicted and
//     amortized over a whole vector; routing it through a closure is not
//     inlined and gives back most of the win (1.12x vs 1.40x on a measured 1M
//     float64 filter+sum).
//   - min/max never use Float64s.Min/Max. Those map to NEON FMIN/FMAX, which
//     *propagate* NaN, while this package's contract is sticky-from-seed: NaN
//     only when the seed is NaN, later NaNs ignored. IfElse(Less(...)) keeps the
//     accumulator when the comparison is false, which is exactly the scalar
//     semantics. IfElse costs three vector ops where Min costs one, so the
//     reductions run four independent accumulator sets to hide the latency.
//
// Generics cannot abstract over these types: the compiler specializes every
// function mentioning simd.Float64s per vector width, which collides with
// generic instantiation ("C does not satisfy vcmp@simd128").

// maxLanes is the widest float64 vector any supported backend produces (AVX512:
// 512/64). It sizes the stack buffers used for the horizontal tail of each
// reduction, keeping these kernels allocation-free. A backend wider than this
// makes every hook decline rather than misbehave.
const maxLanes = 8

// whereMaskVec returns the lane mask for `v <op> lit`. Callers inline the switch
// by construction — this is a leaf the compiler folds into the loop body.
func whereMaskVec(v, lit simd.Float64s, op Cmp) simd.Mask64s {
	switch op {
	case CmpGT:
		return v.Greater(lit)
	case CmpGE:
		return v.GreaterEqual(lit)
	case CmpLT:
		return v.Less(lit)
	case CmpLE:
		return v.LessEqual(lit)
	case CmpEQ:
		return v.Equal(lit)
	default: // CmpNE
		return v.NotEqual(lit)
	}
}

// sumWhereFloat64Vec is the vector path of SumWhereFloat64 for the null-free
// case. Passing values are selected with Masked (which zeroes non-passing lanes,
// so a non-passing NaN cannot poison the sum) and the survivor count is
// accumulated by subtracting the mask's -1/0 lanes into an integer vector.
func sumWhereFloat64Vec(vals []float64, op Cmp, lit float64) (sum float64, count int, ok bool) {
	w := simd.Float64s{}.Len()
	if w > maxLanes || len(vals) < 2*w {
		return 0, 0, false
	}
	l := simd.BroadcastFloat64s(lit)
	var s0, s1 simd.Float64s
	var c0, c1 simd.Int64s
	rest := vals
	for len(rest) >= 2*w {
		v0 := simd.LoadFloat64s(rest)
		v1 := simd.LoadFloat64s(rest[w:])
		m0 := whereMaskVec(v0, l, op)
		m1 := whereMaskVec(v1, l, op)
		s0 = s0.Add(v0.Masked(m0))
		s1 = s1.Add(v1.Masked(m1))
		c0 = c0.Sub(m0.ToInt64s())
		c1 = c1.Sub(m1.ToInt64s())
		rest = rest[2*w:]
	}
	sacc, cacc := s0.Add(s1), c0.Add(c1)
	if len(rest) >= w {
		v := simd.LoadFloat64s(rest)
		m := whereMaskVec(v, l, op)
		sacc = sacc.Add(v.Masked(m))
		cacc = cacc.Sub(m.ToInt64s())
		rest = rest[w:]
	}
	var sbuf [maxLanes]float64
	var cbuf [maxLanes]int64
	sacc.Store(sbuf[:w])
	cacc.Store(cbuf[:w])
	for i := range w {
		sum += sbuf[i]
		count += int(cbuf[i])
	}
	for _, v := range rest {
		if whereKeep(v, lit, op) {
			sum += v
			count++
		}
	}
	return sum, count, true
}

// minMaxWhereFloat64Vec is the vector path of MinMaxWhereFloat64 for the
// null-free case.
//
// min/max are seeded on the first passing row in ascending index order, exactly
// as the scalar body does, so the sticky-from-seed NaN contract holds. The seed
// is found with a scalar scan and the vector loop then resumes from that row's
// vector boundary: rows before the seed do not pass, so skipping them changes
// nothing, and the scan plus the vector loop together still touch each element
// at most once.
func minMaxWhereFloat64Vec(vals []float64, op Cmp, lit float64) (mn, mx float64, count int, ok bool) {
	w := simd.Float64s{}.Len()
	if w > maxLanes || len(vals) < 4*w {
		return 0, 0, 0, false
	}
	seed := -1
	for i, v := range vals {
		if whereKeep(v, lit, op) {
			seed = i
			break
		}
	}
	if seed < 0 {
		return 0, 0, 0, true // no passing row: empty reduction
	}
	l := simd.BroadcastFloat64s(lit)
	sv := simd.BroadcastFloat64s(vals[seed])
	mn0, mn1 := sv, sv
	mx0, mx1 := sv, sv
	var c0, c1 simd.Int64s
	rest := vals[seed/w*w:]
	for len(rest) >= 2*w {
		v0 := simd.LoadFloat64s(rest)
		v1 := simd.LoadFloat64s(rest[w:])
		m0 := whereMaskVec(v0, l, op)
		m1 := whereMaskVec(v1, l, op)
		mn0 = v0.IfElse(v0.Less(mn0).And(m0), mn0)
		mn1 = v1.IfElse(v1.Less(mn1).And(m1), mn1)
		mx0 = v0.IfElse(v0.Greater(mx0).And(m0), mx0)
		mx1 = v1.IfElse(v1.Greater(mx1).And(m1), mx1)
		c0 = c0.Sub(m0.ToInt64s())
		c1 = c1.Sub(m1.ToInt64s())
		rest = rest[2*w:]
	}
	mnv, mxv, cacc := mn0, mx0, c0.Add(c1)
	if len(rest) >= w {
		v := simd.LoadFloat64s(rest)
		m := whereMaskVec(v, l, op)
		mnv = v.IfElse(v.Less(mnv).And(m), mnv)
		mxv = v.IfElse(v.Greater(mxv).And(m), mxv)
		cacc = cacc.Sub(m.ToInt64s())
		rest = rest[w:]
	}
	var buf [2 * maxLanes]float64
	var cbuf [maxLanes]int64
	mnv.Store(buf[0:w])
	mn1.Store(buf[w : 2*w])
	mn = buf[0]
	for _, v := range buf[1 : 2*w] {
		if v < mn {
			mn = v
		}
	}
	mxv.Store(buf[0:w])
	mx1.Store(buf[w : 2*w])
	mx = buf[0]
	for _, v := range buf[1 : 2*w] {
		if v > mx {
			mx = v
		}
	}
	cacc.Store(cbuf[:w])
	for i := range w {
		count += int(cbuf[i])
	}
	for _, v := range rest {
		if !whereKeep(v, lit, op) {
			continue
		}
		if v < mn {
			mn = v
		} else if v > mx {
			mx = v
		}
		count++
	}
	return mn, mx, count, true
}

// minFloat64Vec is the vector path of MinFloat64: four independent accumulators
// updated with IfElse(Less(...)) so NaN is sticky-from-seed, never propagated.
func minFloat64Vec(vals []float64) (float64, bool) {
	w := simd.Float64s{}.Len()
	if w > maxLanes || len(vals) < 4*w {
		return 0, false
	}
	seed := simd.BroadcastFloat64s(vals[0])
	a0, a1, a2, a3 := seed, seed, seed, seed
	rest := vals
	for len(rest) >= 4*w {
		v0 := simd.LoadFloat64s(rest)
		v1 := simd.LoadFloat64s(rest[w:])
		v2 := simd.LoadFloat64s(rest[2*w:])
		v3 := simd.LoadFloat64s(rest[3*w:])
		a0 = v0.IfElse(v0.Less(a0), a0)
		a1 = v1.IfElse(v1.Less(a1), a1)
		a2 = v2.IfElse(v2.Less(a2), a2)
		a3 = v3.IfElse(v3.Less(a3), a3)
		rest = rest[4*w:]
	}
	var buf [4 * maxLanes]float64
	a0.Store(buf[0:w])
	a1.Store(buf[w : 2*w])
	a2.Store(buf[2*w : 3*w])
	a3.Store(buf[3*w : 4*w])
	mn := buf[0]
	for _, v := range buf[1 : 4*w] {
		if v < mn {
			mn = v
		}
	}
	for _, v := range rest {
		if v < mn {
			mn = v
		}
	}
	return mn, true
}

// maxFloat64Vec is the vector path of MaxFloat64, mirroring minFloat64Vec.
func maxFloat64Vec(vals []float64) (float64, bool) {
	w := simd.Float64s{}.Len()
	if w > maxLanes || len(vals) < 4*w {
		return 0, false
	}
	seed := simd.BroadcastFloat64s(vals[0])
	a0, a1, a2, a3 := seed, seed, seed, seed
	rest := vals
	for len(rest) >= 4*w {
		v0 := simd.LoadFloat64s(rest)
		v1 := simd.LoadFloat64s(rest[w:])
		v2 := simd.LoadFloat64s(rest[2*w:])
		v3 := simd.LoadFloat64s(rest[3*w:])
		a0 = v0.IfElse(v0.Greater(a0), a0)
		a1 = v1.IfElse(v1.Greater(a1), a1)
		a2 = v2.IfElse(v2.Greater(a2), a2)
		a3 = v3.IfElse(v3.Greater(a3), a3)
		rest = rest[4*w:]
	}
	var buf [4 * maxLanes]float64
	a0.Store(buf[0:w])
	a1.Store(buf[w : 2*w])
	a2.Store(buf[2*w : 3*w])
	a3.Store(buf[3*w : 4*w])
	mx := buf[0]
	for _, v := range buf[1 : 4*w] {
		if v > mx {
			mx = v
		}
	}
	for _, v := range rest {
		if v > mx {
			mx = v
		}
	}
	return mx, true
}

// minMaxFloat64Vec is the vector path of MinMaxFloat64: one pass, four
// independent accumulators per reduction (eight dependency chains in flight).
func minMaxFloat64Vec(vals []float64) (mn, mx float64, ok bool) {
	w := simd.Float64s{}.Len()
	if w > maxLanes || len(vals) < 4*w {
		return 0, 0, false
	}
	seed := simd.BroadcastFloat64s(vals[0])
	n0, n1, n2, n3 := seed, seed, seed, seed
	x0, x1, x2, x3 := seed, seed, seed, seed
	rest := vals
	for len(rest) >= 4*w {
		v0 := simd.LoadFloat64s(rest)
		v1 := simd.LoadFloat64s(rest[w:])
		v2 := simd.LoadFloat64s(rest[2*w:])
		v3 := simd.LoadFloat64s(rest[3*w:])
		n0 = v0.IfElse(v0.Less(n0), n0)
		n1 = v1.IfElse(v1.Less(n1), n1)
		n2 = v2.IfElse(v2.Less(n2), n2)
		n3 = v3.IfElse(v3.Less(n3), n3)
		x0 = v0.IfElse(v0.Greater(x0), x0)
		x1 = v1.IfElse(v1.Greater(x1), x1)
		x2 = v2.IfElse(v2.Greater(x2), x2)
		x3 = v3.IfElse(v3.Greater(x3), x3)
		rest = rest[4*w:]
	}
	var buf [4 * maxLanes]float64
	n0.Store(buf[0:w])
	n1.Store(buf[w : 2*w])
	n2.Store(buf[2*w : 3*w])
	n3.Store(buf[3*w : 4*w])
	mn = buf[0]
	for _, v := range buf[1 : 4*w] {
		if v < mn {
			mn = v
		}
	}
	x0.Store(buf[0:w])
	x1.Store(buf[w : 2*w])
	x2.Store(buf[2*w : 3*w])
	x3.Store(buf[3*w : 4*w])
	mx = buf[0]
	for _, v := range buf[1 : 4*w] {
		if v > mx {
			mx = v
		}
	}
	for _, v := range rest {
		if v < mn {
			mn = v
		} else if v > mx {
			mx = v
		}
	}
	return mn, mx, true
}
