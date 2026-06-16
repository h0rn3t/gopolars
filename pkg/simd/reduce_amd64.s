//go:build amd64

#include "textflag.h"

// AVX2 float64 reductions. Operand order is chosen so the accumulator update is
//   acc = (candidate < acc) ? candidate : acc      (VMINPD)
//   acc = (candidate > acc) ? candidate : acc      (VMAXPD)
// which keeps the running extreme on a NaN/unordered candidate, exactly like the
// scalar `if v < m`. Every accumulator is seeded by broadcasting vals[0], so a
// NaN at vals[0] is sticky and a later NaN is ignored — identical to the scalar
// reference. Re-reading vals[0] in the main/tail loops is harmless for min/max.
//
// Callers (kernels_amd64.go) only invoke these for len >= 1; the Go side returns
// the scalar empty default for len 0.

// func minFloat64AVX2(vals []float64) float64
TEXT ·minFloat64AVX2(SB), NOSPLIT, $0-32
	MOVQ vals_base+0(FP), AX
	MOVQ vals_len+8(FP), CX

	VBROADCASTSD (AX), Y0
	VMOVUPD Y0, Y1
	VMOVUPD Y0, Y2
	VMOVUPD Y0, Y3

min_big:
	CMPQ CX, $16
	JL   min_small
	VMOVUPD 0(AX), Y4
	VMOVUPD 32(AX), Y5
	VMOVUPD 64(AX), Y6
	VMOVUPD 96(AX), Y7
	VMINPD  Y0, Y4, Y0
	VMINPD  Y1, Y5, Y1
	VMINPD  Y2, Y6, Y2
	VMINPD  Y3, Y7, Y3
	ADDQ $128, AX
	SUBQ $16, CX
	JMP  min_big

min_small:
	CMPQ CX, $4
	JL   min_reduce
	VMOVUPD 0(AX), Y4
	VMINPD  Y0, Y4, Y0
	ADDQ $32, AX
	SUBQ $4, CX
	JMP  min_small

min_reduce:
	VMINPD Y0, Y1, Y0
	VMINPD Y0, Y2, Y0
	VMINPD Y0, Y3, Y0
	VEXTRACTF128 $1, Y0, X1
	VMINPD X0, X1, X0
	VSHUFPD $1, X0, X0, X1
	VMINSD X0, X1, X0

min_tail:
	TESTQ CX, CX
	JZ    min_done
	VMOVSD 0(AX), X2
	VMINSD X0, X2, X0
	ADDQ $8, AX
	SUBQ $1, CX
	JMP  min_tail

min_done:
	VMOVSD X0, ret+24(FP)
	VZEROUPPER
	RET

// func maxFloat64AVX2(vals []float64) float64
TEXT ·maxFloat64AVX2(SB), NOSPLIT, $0-32
	MOVQ vals_base+0(FP), AX
	MOVQ vals_len+8(FP), CX

	VBROADCASTSD (AX), Y0
	VMOVUPD Y0, Y1
	VMOVUPD Y0, Y2
	VMOVUPD Y0, Y3

max_big:
	CMPQ CX, $16
	JL   max_small
	VMOVUPD 0(AX), Y4
	VMOVUPD 32(AX), Y5
	VMOVUPD 64(AX), Y6
	VMOVUPD 96(AX), Y7
	VMAXPD  Y0, Y4, Y0
	VMAXPD  Y1, Y5, Y1
	VMAXPD  Y2, Y6, Y2
	VMAXPD  Y3, Y7, Y3
	ADDQ $128, AX
	SUBQ $16, CX
	JMP  max_big

max_small:
	CMPQ CX, $4
	JL   max_reduce
	VMOVUPD 0(AX), Y4
	VMAXPD  Y0, Y4, Y0
	ADDQ $32, AX
	SUBQ $4, CX
	JMP  max_small

max_reduce:
	VMAXPD Y0, Y1, Y0
	VMAXPD Y0, Y2, Y0
	VMAXPD Y0, Y3, Y0
	VEXTRACTF128 $1, Y0, X1
	VMAXPD X0, X1, X0
	VSHUFPD $1, X0, X0, X1
	VMAXSD X0, X1, X0

max_tail:
	TESTQ CX, CX
	JZ    max_done
	VMOVSD 0(AX), X2
	VMAXSD X0, X2, X0
	ADDQ $8, AX
	SUBQ $1, CX
	JMP  max_tail

max_done:
	VMOVSD X0, ret+24(FP)
	VZEROUPPER
	RET

// func minMaxFloat64AVX2(vals []float64) (min float64, max float64)
// Y0..Y3 hold min accumulators, Y4..Y7 hold max accumulators.
TEXT ·minMaxFloat64AVX2(SB), NOSPLIT, $0-40
	MOVQ vals_base+0(FP), AX
	MOVQ vals_len+8(FP), CX

	VBROADCASTSD (AX), Y0
	VMOVUPD Y0, Y1
	VMOVUPD Y0, Y2
	VMOVUPD Y0, Y3
	VMOVUPD Y0, Y4
	VMOVUPD Y0, Y5
	VMOVUPD Y0, Y6
	VMOVUPD Y0, Y7

mm_big:
	CMPQ CX, $16
	JL   mm_small
	VMOVUPD 0(AX), Y8
	VMOVUPD 32(AX), Y9
	VMOVUPD 64(AX), Y10
	VMOVUPD 96(AX), Y11
	VMINPD  Y0, Y8, Y0
	VMINPD  Y1, Y9, Y1
	VMINPD  Y2, Y10, Y2
	VMINPD  Y3, Y11, Y3
	VMAXPD  Y4, Y8, Y4
	VMAXPD  Y5, Y9, Y5
	VMAXPD  Y6, Y10, Y6
	VMAXPD  Y7, Y11, Y7
	ADDQ $128, AX
	SUBQ $16, CX
	JMP  mm_big

mm_small:
	CMPQ CX, $4
	JL   mm_reduce
	VMOVUPD 0(AX), Y8
	VMINPD  Y0, Y8, Y0
	VMAXPD  Y4, Y8, Y4
	ADDQ $32, AX
	SUBQ $4, CX
	JMP  mm_small

mm_reduce:
	// fold min accumulators into Y0
	VMINPD Y0, Y1, Y0
	VMINPD Y0, Y2, Y0
	VMINPD Y0, Y3, Y0
	VEXTRACTF128 $1, Y0, X1
	VMINPD X0, X1, X0
	VSHUFPD $1, X0, X0, X1
	VMINSD X0, X1, X0
	// fold max accumulators into Y4
	VMAXPD Y4, Y5, Y4
	VMAXPD Y4, Y6, Y4
	VMAXPD Y4, Y7, Y4
	VEXTRACTF128 $1, Y4, X5
	VMAXPD X4, X5, X4
	VSHUFPD $1, X4, X4, X5
	VMAXSD X4, X5, X4

mm_tail:
	TESTQ CX, CX
	JZ    mm_done
	VMOVSD 0(AX), X2
	VMINSD X0, X2, X0
	VMAXSD X4, X2, X4
	ADDQ $8, AX
	SUBQ $1, CX
	JMP  mm_tail

mm_done:
	VMOVSD X0, ret+24(FP)
	VMOVSD X4, ret1+32(FP)
	VZEROUPPER
	RET
