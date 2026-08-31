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
// # Portable vector kernels (GOEXPERIMENT=simd, Go 1.27+)
//
// Building with GOEXPERIMENT=simd additionally compiles vec_simd.go, which
// implements the reductions and the fused filter-reduce kernels on the stdlib
// simd package. Those types are vector-length-agnostic and backed by NEON on
// arm64, AVX/AVX2/AVX512 on amd64, and a pure-Go emulation elsewhere, so one
// body of Go covers every architecture — arm64 in particular, which ships no
// hand-written assembly.
//
//	GOEXPERIMENT=simd go build ./...   // adds the portable vector kernels
//
// The vector code is reached through hooks that return ok=false when they
// decline an input (nulls present, or too short to fill the accumulators);
// vec_generic.go supplies the same signatures returning false in the default
// build, so the branch folds away and correctness never depends on the
// experiment. On amd64 the vector path sits BELOW the AVX2 gate: the assembly
// keeps priority and the portable kernels serve as the pre-AVX2 fallback.
//
// Measured on Apple M4 Pro (arm64/NEON, 2 float64 lanes; wider vectors on x86
// should do better), GOEXPERIMENT=simd vs the default build, 1M float64:
//
//	SumWhereFloat64      547.5µs -> 267.8µs   (-51%)
//	MinMaxWhereFloat64   4178.3µs -> 435.6µs  (-90%)
//	MaxFloat64           277.8µs -> 133.8µs   (-52%)
//	MinFloat64           272.0µs -> 151.0µs   (-44%)
//	MinMaxFloat64        324.7µs -> 239.6µs   (-26%)
//
// Two implementation rules are load-bearing and must survive any rewrite:
// comparisons switch on the loop-invariant Cmp INLINE in the loop body (routing
// it through a func value is not inlined and costs most of the win), and min/max
// use IfElse(Less(...)) rather than Float64s.Min/Max, because the latter map to
// NEON FMIN/FMAX which propagate NaN and would break the sticky-from-seed
// contract in a way that depends on which lane the NaN lands in.
//
// Kernels deliberately left scalar, with the measurement behind each:
//
//	SumFloat64                      1.05x — already at the memory bound
//	AddSlicesFloat64/MulSlices      0.64x — SLOWER; the compiler's
//	                                auto-vectorization beats a hand-written
//	                                vector loop here (same finding as the
//	                                earlier EPYC 7763 AVX2 experiment)
//	CompareGT/EQ...Bitmap           no portable mask->bitmap: ToBits() exists
//	                                only in archsimd on amd64; and the existing
//	                                loop already runs at memory bandwidth
//	CompressIndices, BitmapAnd      already bit-parallel, 64 bits per word
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
