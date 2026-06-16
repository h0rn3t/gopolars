//go:build amd64

#include "textflag.h"

// AVX2 element-wise float64 add/mul: dst[i] = a[i] op b[i] for i in [0, len(dst)).
// The Go wrapper pre-allocates dst with len = min(len(a), len(b)) and slices a
// and b to that length, so dst_len drives the loop and all three pointers are
// valid for that many elements. 16 elements (4 YMM) per iteration, scalar tail.

// func addSlicesFloat64AVX2(dst, a, b []float64)
TEXT ·addSlicesFloat64AVX2(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ dst_len+8(FP), CX
	MOVQ a_base+24(FP), SI
	MOVQ b_base+48(FP), BX

add_big:
	CMPQ CX, $16
	JL   add_small
	VMOVUPD 0(SI), Y0
	VMOVUPD 32(SI), Y1
	VMOVUPD 64(SI), Y2
	VMOVUPD 96(SI), Y3
	VADDPD  0(BX), Y0, Y0
	VADDPD  32(BX), Y1, Y1
	VADDPD  64(BX), Y2, Y2
	VADDPD  96(BX), Y3, Y3
	VMOVUPD Y0, 0(DI)
	VMOVUPD Y1, 32(DI)
	VMOVUPD Y2, 64(DI)
	VMOVUPD Y3, 96(DI)
	ADDQ $128, SI
	ADDQ $128, BX
	ADDQ $128, DI
	SUBQ $16, CX
	JMP  add_big

add_small:
	CMPQ CX, $4
	JL   add_tail
	VMOVUPD 0(SI), Y0
	VADDPD  0(BX), Y0, Y0
	VMOVUPD Y0, 0(DI)
	ADDQ $32, SI
	ADDQ $32, BX
	ADDQ $32, DI
	SUBQ $4, CX
	JMP  add_small

add_tail:
	TESTQ CX, CX
	JZ    add_done
	VMOVSD 0(SI), X0
	VADDSD 0(BX), X0, X0
	VMOVSD X0, 0(DI)
	ADDQ $8, SI
	ADDQ $8, BX
	ADDQ $8, DI
	SUBQ $1, CX
	JMP  add_tail

add_done:
	VZEROUPPER
	RET

// func mulSlicesFloat64AVX2(dst, a, b []float64)
TEXT ·mulSlicesFloat64AVX2(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ dst_len+8(FP), CX
	MOVQ a_base+24(FP), SI
	MOVQ b_base+48(FP), BX

mul_big:
	CMPQ CX, $16
	JL   mul_small
	VMOVUPD 0(SI), Y0
	VMOVUPD 32(SI), Y1
	VMOVUPD 64(SI), Y2
	VMOVUPD 96(SI), Y3
	VMULPD  0(BX), Y0, Y0
	VMULPD  32(BX), Y1, Y1
	VMULPD  64(BX), Y2, Y2
	VMULPD  96(BX), Y3, Y3
	VMOVUPD Y0, 0(DI)
	VMOVUPD Y1, 32(DI)
	VMOVUPD Y2, 64(DI)
	VMOVUPD Y3, 96(DI)
	ADDQ $128, SI
	ADDQ $128, BX
	ADDQ $128, DI
	SUBQ $16, CX
	JMP  mul_big

mul_small:
	CMPQ CX, $4
	JL   mul_tail
	VMOVUPD 0(SI), Y0
	VMULPD  0(BX), Y0, Y0
	VMOVUPD Y0, 0(DI)
	ADDQ $32, SI
	ADDQ $32, BX
	ADDQ $32, DI
	SUBQ $4, CX
	JMP  mul_small

mul_tail:
	TESTQ CX, CX
	JZ    mul_done
	VMOVSD 0(SI), X0
	VMULSD 0(BX), X0, X0
	VMOVSD X0, 0(DI)
	ADDQ $8, SI
	ADDQ $8, BX
	ADDQ $8, DI
	SUBQ $1, CX
	JMP  mul_tail

mul_done:
	VZEROUPPER
	RET
