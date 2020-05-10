// +build !decimal_pure_go

package decimal

// implemented in arith_$GOARCH.s
// func add10VV(z, x, y []Word) (c Word)
// func sub10VV(z, x, y []Word) (c Word)
// func add10VW(z, x []Word, y Word) (c Word)
// func sub10VW(z, x []Word, y Word) (c Word)
// func shl10VU(z, x []Word, s uint) (c Word)
// func shr10VU(z, x []Word, s uint) (c Word)
// func mulAdd10VWW(z, x []Word, y, r Word) (c Word)
// func addMul10VVW(z, x []Word, y Word) (c Word)
// func div10WVW(z []Word, xn Word, x []Word, y Word) (r Word)
// func div10W(n1, n0 Word) (q, r Word)
