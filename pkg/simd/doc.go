// Package simd provides the column kernels used by the vectorized expression
// and reduction engine, with two backends selected at build time.
//
// On amd64 built with GOEXPERIMENT=simd the float64 reductions and elementwise
// ops delegate to turboslice (simd_amd64.go). Every other build — including
// arm64, and amd64 without the experiment — uses the generic implementations in
// simd_generic.go.
//
// Build with the turboslice SIMD backend:
//
//	GOEXPERIMENT=simd go build ./...
//
// Build with the generic backend (auto-fallback):
//
//	go build ./...
//
// # arm64 / generic reductions
//
// The generic reductions (SumFloat64, MinFloat64, MaxFloat64, MinMaxFloat64)
// are written with multiple independent accumulators over an unrolled loop, with
// reslicing to hoist bounds checks. Breaking the single-accumulator dependency
// chain lets a superscalar core keep several FADD/FCMP in flight; on Apple M4
// Pro (arm64, Go 1.26) this is ~6.7x faster for Sum and ~3.5-5x for min/max than
// the previous scalar loop. Go 1.26 does not emit NEON here (the hot loop is
// independent scalar FADDD instructions), but the instruction-level parallelism
// is sufficient, so no hand-written arm64 assembly is shipped. Min/max remain
// order-independent and bit-identical to the scalar reference (each accumulator
// is seeded from vals[0], so NaN is sticky-from-seed and later NaNs are ignored,
// exactly as before); Sum reorders additions and therefore differs from a strict
// left-to-right sum within floating-point reduction-order tolerance.
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
