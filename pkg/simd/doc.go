// Package simd provides the column kernels used by the vectorized expression
// and reduction engine. The float64 reduction and element-wise kernels select
// their implementation at runtime — no build tag required.
//
// On amd64, the reductions MinFloat64/MaxFloat64/MinMaxFloat64 dispatch to
// hand-written AVX2 assembly (reduce_amd64.s) when cpu.X86.HasAVX2 is true
// (kernels_amd64.go), and fall back to the scalar multiple-accumulator bodies in
// scalar.go on pre-AVX2 amd64. On every other architecture (simd_generic.go) the
// exported functions are those same scalar bodies. SumFloat64, DotProductFloat64,
// AddSlicesFloat64 and MulSlicesFloat64 stay scalar everywhere: the Go compiler
// already auto-vectorizes their float64 loops (a measured EPYC 7763 run found a
// hand-written AVX2 add/mul ~2.5x slower than the auto-vectorized scalar).
//
//	go build ./...   // one binary, AVX2 used at runtime on capable amd64 CPUs
//
// # Scalar reductions / fallback
//
// The scalar reductions (scalar.go) use multiple independent accumulators over
// an unrolled loop, with reslicing to hoist bounds checks. Breaking the
// single-accumulator dependency chain lets a superscalar core keep several
// FADD/FCMP in flight; on Apple M4 Pro (arm64, Go 1.26) this is ~6.7x faster for
// Sum and ~3.5-5x for min/max than a plain scalar loop, so no hand-written arm64
// assembly is shipped. Min/max are order-independent and bit-identical to the
// strict scalar reference (each accumulator is seeded from vals[0], so NaN is
// sticky-from-seed and later NaNs are ignored); the AVX2 kernels choose their
// VMINPD/VMAXPD operand order to reproduce exactly this NaN behaviour, and an
// equivalence test pins them to the scalar bodies. Sum reorders additions and
// therefore differs from a strict left-to-right sum within floating-point
// reduction-order tolerance.
//
// # Filter / reduce kernels (kernels.go)
//
// These are defined without a build tag so correctness never depends on the
// backend. CompressIndices uses a count-then-allocate strategy so a
// low-selectivity filter does not over-allocate an N-sized index buffer.
// MaskedReduceFloat64 is the kernel behind the fused filter+reduce path: it
// reduces (sum/min/max/count) the rows surviving a predicate in a single masked
// pass, avoiding both a surviving-index slice and a materialized filtered column.
package simd
