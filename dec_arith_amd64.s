// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !decimal_pure_go

#include "textflag.h"

#define _DB  10000000000000000000
#define _DMax 9999999999999999999
#define _DW 19

// This file provides fast assembly versions for the elementary
// arithmetic operations on vectors implemented in arith.go.

// func mul10WW(x, y Word) (z1, z0 Word)
TEXT ·mul10WW(SB),NOSPLIT,$0
	MOVQ x+0(FP), AX
	MULQ y+8(FP)

	// inlined version of div10W with DX = n1, AX = n0
	// trashes R13, R14, AX, BX, CX, DX
	MOVQ DX, R13
	MOVQ AX, R14
	MOVQ AX, BX
	SARQ $63, BX		// _n1
	MOVQ DX, AX
	SUBQ BX, AX			// AX == n1-_n1
	MOVQ $0xd83c94fb6d2ac34a, CX
	MULQ CX				// DX:AX = m' * (n1-_n1)
	MOVQ $_DB, CX 
	ANDQ CX, BX			// BX = d&_n1
	ADDQ R14, BX 		// nAdj
	ADDQ BX, AX
	ADCQ R13, DX			// q1 + n1 + carry
	MOVQ DX, BX
	NOTQ BX				// t
	MOVQ BX, AX
	MULQ CX				// DX:AX = t * d
	ADDQ R14, AX
	ADCQ R13, DX
	SUBQ CX, DX			// DX:AX = dr
	ANDQ DX, CX
	ADDQ CX, AX			// r
	SUBQ BX, DX			// q

	MOVQ DX, z1+16(FP)
	MOVQ AX, z0+24(FP)
	RET


// func div10WW(x1, x0, y Word) (q, r Word)
TEXT ·div10WW(SB),NOSPLIT,$0
	// mulAddWWW(x1, _DB, x0)
	MOVQ x1+0(FP), AX
	MOVQ $_DB, CX
	MULQ CX            // hi: DX, lo: AX
	ADDQ x0+8(FP), AX  // AX += v
	ADCQ $0, DX        // DX += carry
	DIVQ y+16(FP)
	MOVQ AX, q+24(FP)
	MOVQ DX, r+32(FP)
	RET


// func div10W(n1, n0 Word) (q, r Word)
TEXT ·div10W(SB),NOSPLIT,$0
	// N = l     = 64
	// d = dNorm = _DB == 1e19
	// m'        = 0xd83c94fb6d2ac34a
	// n2, n10   = n1, n0
	MOVQ n1+0(FP), R8
	MOVQ n0+8(FP), R9
	MOVQ R9, BX
	SARQ $63, BX		// _n1
	MOVQ R8, AX
	SUBQ BX, AX			// AX == n1-_n1
	MOVQ $0xd83c94fb6d2ac34a, CX
	MULQ CX				// DX:AX = m' * (n1-_n1)
	MOVQ $_DB, CX 
	ANDQ CX, BX			// BX = d&_n1
	ADDQ R9, BX 		// nAdj
	ADDQ BX, AX
	ADCQ R8, DX			// q1 + n1 + carry
	MOVQ DX, BX
	NOTQ BX				// t
	MOVQ BX, AX
	MULQ CX				// DX:AX = t * d
	ADDQ R9, AX
	ADCQ R8, DX
	SUBQ CX, DX			// DX:AX = dr
	ANDQ DX, CX
	ADDQ CX, AX			// r
	SUBQ BX, DX			// q
	MOVQ DX, q+16(FP)
	MOVQ AX, r+24(FP)
	RET


// The carry bit is saved with SBBQ Rx, Rx: if the carry was set, Rx is -1, otherwise it is 0.
// It is restored with ADDQ Rx, Rx: if Rx was -1 the carry is set, otherwise it is cleared.
// This is faster than using rotate instructions.

// func add10VV(z, x, y []Word) (c Word)
TEXT ·add10VV(SB),NOSPLIT,$0
	MOVQ z_len+8(FP), DI
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), R9
	MOVQ z+0(FP), R10

	MOVQ $0, CX		// c = 0
	MOVQ $0, SI		// i = 0
	MOVQ $_DMax, DX

	// s/JL/JMP/ below to disable the unrolled loop
	SUBQ $4, DI		// n -= 4
	JL V1			// if n < 0 goto V1

U1:	// n >= 0
	// regular loop body unrolled 4x
	MOVQ 0(R8)(SI*8), R11
	MOVQ 8(R8)(SI*8), R12
	MOVQ 16(R8)(SI*8), R13
	MOVQ 24(R8)(SI*8), R14
	ADDQ CX, CX			// restore CF
	ADCQ 0(R9)(SI*8), R11
	SBBQ CX, CX
	CMPQ DX, R11
	SBBQ BX, BX
	ORQ BX, CX
	LEAQ 1(DX), AX
	ANDQ CX, AX
	SUBQ AX, R11
	ADDQ CX, CX			// restore CF
	ADCQ 8(R9)(SI*8), R12
	SBBQ CX, CX
	CMPQ DX, R12
	SBBQ BX, BX
	ORQ BX, CX
	LEAQ 1(DX), AX
	ANDQ CX, AX
	SUBQ AX, R12
	ADDQ CX, CX			// restore CF
	ADCQ 16(R9)(SI*8), R13
	SBBQ CX, CX
	CMPQ DX, R13
	SBBQ BX, BX
	ORQ BX, CX
	LEAQ 1(DX), AX
	ANDQ CX, AX
	SUBQ AX, R13
	ADDQ CX, CX			// restore CF
	ADCQ 24(R9)(SI*8), R14
	SBBQ CX, CX
	CMPQ DX, R14
	SBBQ BX, BX
	ORQ BX, CX
	LEAQ 1(DX), AX
	ANDQ CX, AX
	SUBQ AX, R14
	MOVQ R11, 0(R10)(SI*8)
	MOVQ R12, 8(R10)(SI*8)
	MOVQ R13, 16(R10)(SI*8)
	MOVQ R14, 24(R10)(SI*8)

	ADDQ $4, SI		// i += 4
	SUBQ $4, DI		// n -= 4
	JGE U1			// if n >= 0 goto U1

V1:	ADDQ $4, DI		// n += 4
	JLE E1			// if n <= 0 goto E1

L1:	// n > 0
	ADDQ CX, CX		// restore CF
	MOVQ 0(R8)(SI*8), R11
	ADCQ 0(R9)(SI*8), R11
	SBBQ CX, CX
	CMPQ DX, R11
	SBBQ BX, BX
	ORQ BX, CX
	LEAQ 1(DX), AX
	ANDQ CX, AX
	SUBQ AX, R11
	MOVQ R11, 0(R10)(SI*8)

	ADDQ $1, SI		// i++
	SUBQ $1, DI		// n--
	JG L1			// if n > 0 goto L1

E1: NEGQ CX
	MOVQ CX, c+72(FP)	// return c
	RET


// func sub10VV(z, x, y []Word) (c Word)
TEXT ·sub10VV(SB),NOSPLIT,$0
	MOVQ z_len+8(FP), DI
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), R9
	MOVQ z+0(FP), R10

	MOVQ $0, CX		// c = 0
	MOVQ $0, SI		// i = 0
	MOVQ $_DB, DX

	// s/JL/JMP/ below to disable the unrolled loop
	SUBQ $4, DI		// n -= 4
	JL V2			// if n < 0 goto V2

U2:	// n >= 0
	// regular loop body unrolled 4x
	MOVQ 0(R8)(SI*8), R11
	MOVQ 8(R8)(SI*8), R12
	MOVQ 16(R8)(SI*8), R13
	MOVQ 24(R8)(SI*8), R14
	ADDQ CX, CX		// restore CF	
	SBBQ 0(R9)(SI*8), R11
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R11
	ADDQ CX, CX		// restore CF
	SBBQ 8(R9)(SI*8), R12
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R12
	ADDQ CX, CX		// restore CF
	SBBQ 16(R9)(SI*8), R13
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R13
	ADDQ CX, CX		// restore CF
	SBBQ 24(R9)(SI*8), R14
	SBBQ CX, CX		// save CF
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R14
	MOVQ R11, 0(R10)(SI*8)
	MOVQ R12, 8(R10)(SI*8)
	MOVQ R13, 16(R10)(SI*8)
	MOVQ R14, 24(R10)(SI*8)

	ADDQ $4, SI		// i += 4
	SUBQ $4, DI		// n -= 4
	JGE U2			// if n >= 0 goto U2

V2:	ADDQ $4, DI		// n += 4
	JLE E2			// if n <= 0 goto E2

L2:	// n > 0
	ADDQ CX, CX		// restore CF
	MOVQ 0(R8)(SI*8), R11
	SBBQ 0(R9)(SI*8), R11
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R11
	MOVQ R11, 0(R10)(SI*8)

	ADDQ $1, SI		// i++
	SUBQ $1, DI		// n--
	JG L2			// if n > 0 goto L2

E2:	NEGQ CX
	MOVQ CX, c+72(FP)	// return c
	RET


// func add10VW(z, x []Word, y Word) (c Word)
TEXT ·add10VW(SB),NOSPLIT,$0
	MOVQ z_len+8(FP), DI
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), CX	// c = y
	MOVQ z+0(FP), R10

	MOVQ $0, SI			// i = 0
	MOVQ $_DB, DX

	// Once we start looping, we won't handle the hardware carry since
	// x[i] < _DB, so x[i] + 1 < 1<<64-1 always.
	// This still needs to be handled for the first element.

	SUBQ $1, DI		// n--
	JL E3			// abort if n < 0
	ADDQ 0(R8)(SI*8), CX
	LEAQ -1(DX), AX
	SBBQ BX, BX
	CMPQ AX, CX
	SBBQ AX, AX
	ORQ AX, BX
	MOVQ DX, AX
	ANDQ BX, AX
	SUBQ AX, CX
	NEGQ BX			// convert to C = 0/1
	MOVQ CX, 0(R10)(SI*8)
	MOVQ BX, CX		// save c
	ADDQ $1, SI		// i++

	// z[1:] ...
	// We do not check the carry value to switch to a memcpy for small slices
	// (len < 4). This seems to be a good trade-off to the cost of additional
	// jumps/function calls.

	// s/JL/JMP/ below to disable the unrolled loop and memcpy
	SUBQ $4, DI		// n -= 4
	JL V3			// if n < 4 goto V3.

	TESTQ CX, CX
	JNE U3			// if c != 0 propagate
	CMPQ R8, R10
	JEQ E3			// don't copy if &x[0] == &z[0]
	ADDQ $4, DI
	MOVQ CX, c+56(FP)
	JMP decCpy(SB)

U3:	// n >= 0
	// regular loop body unrolled 4x
	ADDQ 0(R8)(SI*8), CX
	CMPQ CX, DX
	SBBQ BX, BX
	ANDQ BX, CX
	MOVQ CX, 0(R10)(SI*8)
	LEAQ 1(BX), CX
	ADDQ 8(R8)(SI*8), CX
	CMPQ CX, DX
	SBBQ BX, BX
	ANDQ BX, CX
	MOVQ CX, 8(R10)(SI*8)
	LEAQ 1(BX), CX
	ADDQ 16(R8)(SI*8), CX
	CMPQ CX, DX
	SBBQ BX, BX
	ANDQ BX, CX
	MOVQ CX, 16(R10)(SI*8)
	LEAQ 1(BX), CX
	ADDQ 24(R8)(SI*8), CX
	CMPQ CX, DX
	SBBQ BX, BX
	ANDQ BX, CX
	MOVQ CX, 24(R10)(SI*8)
	LEAQ 1(BX), CX

	ADDQ $4, SI		// i += 4
	TESTQ BX, BX
	JL C3
	SUBQ $4, DI		// n -= 4
	JGE U3			// if n >= 0 goto U3

V3:	ADDQ $4, DI		// n += 4
	JLE E3			// if n <= 0 goto E3

L3:	// n > 0
	ADDQ 0(R8)(SI*8), CX
	CMPQ CX, DX
	SBBQ BX, BX		// BX = _DB > CX ? -1 : 0
	ANDQ BX, CX		// sets CX to 0 if CX >= _DB
	MOVQ CX, 0(R10)(SI*8)
	LEAQ 1(BX), CX	// eqv to NOTQ BX, NEGQ BX, MOVQ BX CX

	ADDQ $1, SI		// i++
	SUBQ $1, DI		// n--
	JG L3			// if n > 0 goto L3

E3:	MOVQ CX, c+56(FP)	// return c
	RET

C3: // memcpy
	CMPQ R8, R10	// don't copy if &x[0] == &z[0]
	JEQ E3
	MOVQ CX, c+56(FP)
	JMP decCpy(SB)


// func sub10VW(z, x []Word, y Word) (c Word)
// (same as add10VW except for SUBQ/SBBQ instead of ADDQ/ADCQ and label names)
TEXT ·sub10VW(SB),NOSPLIT,$0
	MOVQ z_len+8(FP), DI
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), CX	// c = y
	MOVQ z+0(FP), R10

	XORQ SI, SI			// i = 0
	MOVQ $_DB, DX

	// s/JL/JMP/ below to disable the unrolled loop
	SUBQ $4, DI		// n -= 4
	JL V4			// if n < 4 goto V4

U4:	// n >= 0
	// regular loop body unrolled 4x
	MOVQ 0(R8)(SI*8), BX
	SUBQ CX, BX
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, BX
	MOVQ BX, 0(R10)(SI*8)
	NEGQ CX
	MOVQ 8(R8)(SI*8), BX
	SUBQ CX, BX
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, BX
	MOVQ BX, 8(R10)(SI*8)
	NEGQ CX
	MOVQ 16(R8)(SI*8), BX
	SUBQ CX, BX
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, BX
	MOVQ BX, 16(R10)(SI*8)
	NEGQ CX
	MOVQ 24(R8)(SI*8), BX
	SUBQ CX, BX
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, BX
	MOVQ BX, 24(R10)(SI*8)
	NEGQ CX

	LEAQ 4(SI), SI	// i += 4
	JCC C4			// if CX == 0 goto C4 (memcpy)
	SUBQ $4, DI		// n -= 4
	JGE U4			// if n >= 0 goto U4

V4:	ADDQ $4, DI		// n += 4
	JLE E4			// if n <= 0 goto E4

L4:	// n > 0
	MOVQ 0(R8)(SI*8), R11
	SUBQ CX, R11
	SBBQ CX, CX
	MOVQ DX, AX
	ANDQ CX, AX
	ADDQ AX, R11
	NEGQ CX
	MOVQ R11, 0(R10)(SI*8)

	ADDQ $1, SI		// i++
	SUBQ $1, DI		// n--
	JG L4			// if n > 0 goto L4

E4:	MOVQ CX, c+56(FP)	// return c
	RET

C4: // memcpy
	CMPQ R8, R10	// don't copy if &x[0] == &z[0]
	JEQ E4
	MOVQ CX, c+56(FP)
	JMP decCpy(SB)


// func decCpy(dst = R10, src = R8, i = SI, n = DI)
TEXT decCpy(SB),NOSPLIT,$0
	SUBQ $4, DI		// n -= 4
	JL CV			// if n < 0 goto CV

CU: // n >= 4
	MOVQ 0(R8)(SI*8), AX
	MOVQ 8(R8)(SI*8), BX
	MOVQ 16(R8)(SI*8), CX
	MOVQ 24(R8)(SI*8), DX
	MOVQ AX, 0(R10)(SI*8)
	MOVQ BX, 8(R10)(SI*8)
	MOVQ CX, 16(R10)(SI*8)
	MOVQ DX, 24(R10)(SI*8)
	ADDQ $4, SI		// i += 4
	SUBQ $4, DI		// n -= 4
	JGE CU			// if n >= 0 goto C4
CV:
	ADDQ $4, DI
	JLE CE
CLoop:
	MOVQ 0(R8)(SI*8), AX
	MOVQ AX, 0(R10)(SI*8)
	ADDQ $1, SI
	SUBQ $1, DI
	JG CLoop
CE:
	RET


// func decCpyInv(dst = R10, src = R8, n = SI)
// copies from high to low address
TEXT decCpyInv(SB),NOSPLIT,$0
	SUBQ $4, SI
	JL CV

CU: // n >= 4
	MOVQ 0(R8)(SI*8), AX
	MOVQ 8(R8)(SI*8), BX
	MOVQ 16(R8)(SI*8), CX
	MOVQ 24(R8)(SI*8), DX
	MOVQ AX, 0(R10)(SI*8)
	MOVQ BX, 8(R10)(SI*8)
	MOVQ CX, 16(R10)(SI*8)
	MOVQ DX, 24(R10)(SI*8)
	SUBQ $4, SI		// n -= 4
	JGE CU			// if n >= 0 goto C4
CV:
	ADDQ $3, SI
	JL CE
CLoop:
	MOVQ 0(R8)(SI*8), AX
	MOVQ AX, 0(R10)(SI*8)
	SUBQ $1, SI
	JGE CLoop
CE:
	RET


// func shl10VU(z, x []Word, s uint) (c Word)
TEXT ·shl10VU(SB),NOSPLIT,$0
	// TODO(db47h): unroll loop
	MOVQ z_len+8(FP), SI	// i = z
	SUBQ $1, SI				// i--
	JL X8b					// i < 0	(n <= 0)

	// n > 0
 	MOVQ z+0(FP), R10
 	MOVQ x+24(FP), R8
 	MOVQ s+48(FP), BX
	TESTQ BX, BX
	JEQ X8c		// copy if s = 0

	MOVQ $_DW, AX
	LEAQ ·pow10DivTab64(SB), DI
	SUBQ BX, AX
	LEAQ -3(AX)(AX*2), AX
	LEAQ -3(BX)(BX*2), BX
	MOVQ 0(DI)(BX*8), R11 		// m = pow10DivTab64(s-1).d
	MOVQ 0(DI)(AX*8), R12		// d
	MOVQ 8(DI)(AX*8), R13		// m'
	MOVWLZX 16(DI)(AX*8), CX	// post|pre

	// r, l = x[len(x)-1] / d
	MOVQ 0(R8)(SI*8), AX
	MOVQ AX, BX				// x[i]
	SHRQ CX, AX				// x[i] >> pre
	MULQ R13				// *m'
	RORW $8, CX				// post
	MOVQ DX, AX
	SHRQ CX, AX				// r
	MOVQ AX, c+56(FP)		// save r
	MULQ R12				// DX:AX = r*d
	SUBQ AX, BX				// l = x[i]-r*d
	MOVQ BX, AX				// AX = l

	TESTQ SI, SI
	JEQ X8a
L8:
	MULQ R11
	MOVQ AX, R9				// z[i] = l*m
	MOVQ -8(R8)(SI*8), AX	
	MOVQ AX, BX				// x[i-1]
	RORW $8, CX				// pre
	SHRQ CX, AX				// x[i-1] >> pre
	MULQ R13				// *m'
	RORW $8, CX				// post
	MOVQ DX, AX			
	SHRQ CX, AX				// h
	ADDQ AX, R9				// z[i] += h
	MOVQ R9, 0(R10)(SI*8)
	MULQ R12				// DX:AX = d*h
	SUBQ AX, BX				// l = x[i-1]-d*h
	MOVQ BX, AX
	SUBQ $1, SI
	JG L8
X8a:
	MULQ R11
	MOVQ AX, 0(R10)(SI*8)
	RET
X8b:
	MOVQ $0, c+56(FP)
	RET
X8c:
	CMPQ R10, R8
	JEQ X8b
	ADDQ $1, SI
	MOVQ $0, c+56(FP)
	JMP decCpyInv(SB)


// func shr10VU(z, x []Word, s uint) (c Word)
TEXT ·shr10VU(SB),NOSPLIT,$0
	// TODO(db47h): implement constant division by multiplication, unroll loop.
	MOVQ z_len+8(FP), DI
	SUBQ $1, DI		// n--
	JL X9b			// n < 0	(n <= 0)

	// n > 0
 	MOVQ z+0(FP), R10
 	MOVQ x+24(FP), R8
 	MOVQ s+48(FP), BX
	TESTQ BX, BX
	JEQ X9c		// copy if s = 0

	MOVQ $_DW, AX
	LEAQ ·pow10DivTab64(SB), SI
	SUBQ BX, AX
	LEAQ -3(AX)(AX*2), AX
	LEAQ -3(BX)(BX*2), BX
	MOVQ 0(SI)(AX*8), R11 		// m = pow10DivTab64(_DW-s-1).d
	MOVQ 0(SI)(BX*8), R12		// d
	MOVQ 8(SI)(BX*8), R13		// m'
	MOVWLZX 16(SI)(BX*8), CX	// post|pre

	MOVQ 0(R8), AX		// x[0]
	MOVQ AX, R9			
	SHRQ CX, AX			// x[0] >> pre
	MULQ R13			// *m'
	RORW $8, CX			// post
	SHRQ CX, DX			// h
	MOVQ DX, BX
	MOVQ R12, AX
	MULQ DX				// DX:AX = h*d
	SUBQ AX, R9			// r = x[0]-h*d
	MOVQ R11, AX
	MULQ R9
	MOVQ AX, c+56(FP)	// save r*m

	MOVQ $0, SI
	CMPQ SI, DI
	JGE X9a			// if i >= len(x)-1 goto X9a

L9:
	MOVQ BX, R9		// z[i] = h
	MOVQ 8(R8)(SI*8), AX
	MOVQ AX, R14	// x[i+1]
	RORW $8, CX
	SHRQ CX, AX
	MULQ R13
	RORW $8, CX 
	SHRQ CX, DX		
	MOVQ DX, BX		// BX = h
	MOVQ R12, AX
	MULQ DX			// d*h
	SUBQ AX, R14	// l = x[i+1]-d*h
	MOVQ R11, AX
	MULQ R14		// l*m
	ADDQ AX, R9
	MOVQ R9, 0(R10)(SI*8)
	ADDQ $1, SI
	CMPQ SI, DI
	JL L9
X9a:
	MOVQ BX, 0(R10)(SI*8)
	RET
X9b:
	MOVQ $0, c+56(FP)
 	RET
X9c:
	CMPQ R10, R8
	JEQ X9b
	ADDQ $1, DI
	XORQ SI, SI
	MOVQ $0, c+56(FP)
	JMP decCpy(SB)


// func mulAdd10VWW(z, x []Word, y, r Word) (c Word)
TEXT ·mulAdd10VWW(SB),NOSPLIT,$0
	MOVQ z+0(FP), R10
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), R9
	MOVQ r+56(FP), R11	 // c = r
	MOVQ z_len+8(FP), DI // n
	MOVQ $0, SI			 // i = 0

	CMPQ SI, DI
	JGE E10
L10:
	MOVQ 0(R8)(SI*8), AX
	MULQ R9
	ADDQ R11, AX
	ADCQ $0, DX

	// inlined version of div10W with DX = n1, AX = n0
	// trashes R13, R14, AX, BX, CX, DX
	MOVQ DX, R13
	MOVQ AX, R14
	MOVQ AX, BX
	SARQ $63, BX		// _n1
	MOVQ DX, AX
	SUBQ BX, AX			// AX == n1-_n1
	MOVQ $0xd83c94fb6d2ac34a, CX
	MULQ CX				// DX:AX = m' * (n1-_n1)
	MOVQ $_DB, CX 
	ANDQ CX, BX			// BX = d&_n1
	ADDQ R14, BX 		// nAdj
	ADDQ BX, AX
	ADCQ R13, DX		// q1 + n1 + carry
	MOVQ DX, BX
	NOTQ BX				// t
	MOVQ BX, AX
	MULQ CX				// DX:AX = t * d
	ADDQ R14, AX
	ADCQ R13, DX
	SUBQ CX, DX			// DX:AX = dr
	ANDQ DX, CX
	ADDQ CX, AX			// r
	SUBQ BX, DX			// q

	MOVQ DX, R11
	MOVQ AX, 0(R10)(SI*8)

	ADDQ $1, SI
	CMPQ SI, DI
	JL L10
E10:
	MOVQ R11, c+64(FP)
	RET


// func addMmul10VVW(z, x []Word, y Word) (c Word)
TEXT ·addMul10VVW(SB),NOSPLIT,$0
	MOVQ z+0(FP), R10
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), R9
	MOVQ z_len+8(FP), DI	// n
	MOVQ $0, SI				// i = 0
	XORQ R11, R11			// c = 0

	CMPQ SI, DI
	JGE E11
L11:
	// xi * y + zi
	MOVQ 0(R8)(SI*8), AX
	MULQ R9
	ADDQ 0(R10)(SI*8), AX
	ADCQ $0, DX
	ADDQ R11, AX			// lo
	ADCQ $0, DX				// hi

	// inlined version of div10W with DX = n1, AX = n0
	// trashes R13, R14, AX, BX, CX, DX
	MOVQ DX, R13
	MOVQ AX, R14
	MOVQ AX, BX
	SARQ $63, BX		// _n1
	MOVQ DX, AX
	SUBQ BX, AX			// AX == n1-_n1
	MOVQ $0xd83c94fb6d2ac34a, CX
	MULQ CX				// DX:AX = m' * (n1-_n1)
	MOVQ $_DB, CX 
	ANDQ CX, BX			// BX = d&_n1
	ADDQ R14, BX 		// nAdj
	ADDQ BX, AX
	ADCQ R13, DX		// q1 + n1 + carry
	MOVQ DX, BX
	NOTQ BX				// t
	MOVQ BX, AX
	MULQ CX				// DX:AX = t * d
	ADDQ R14, AX
	ADCQ R13, DX
	SUBQ CX, DX			// DX:AX = dr
	ANDQ DX, CX
	ADDQ CX, AX			// r | l
	SUBQ BX, DX			// q | h

	MOVQ DX, R11
	MOVQ AX, 0(R10)(SI*8)

	ADDQ $1, SI
	CMPQ SI, DI
	JL L11
E11:
	MOVQ R11, c+56(FP)
	RET


// func div10VWW(z, x []Word, y, xn Word) (r Word)
TEXT ·div10VWW(SB),NOSPLIT,$0
	MOVQ z+0(FP), R10
	MOVQ x+24(FP), R8
	MOVQ y+48(FP), R9
	MOVQ xn+56(FP), DX		// r = xn
	MOVQ z_len+8(FP), SI	// i = z
	MOVQ $_DB, CX
	JMP E7

L7: MOVQ DX, AX
	MULQ CX
	ADDQ (R8)(SI*8), AX
	ADCQ $0, DX
	DIVQ R9
	MOVQ AX, (R10)(SI*8)

E7:	SUBQ $1, SI		// i--
	JGE L7			// i >= 0

	MOVQ DX, r+64(FP)
	RET
