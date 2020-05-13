// +build !decimal_pure_go,amd64

package decimal

// implemented in dec_arith_$GOARCH.s
func mul10WW(x, y Word) (z1, z0 Word)

func div10WW(x1, x0, y Word) (q, r Word)

func add10VV(z, x, y []Word) (c Word)

func div10W(n1, n0 Word) (q, r Word)

func sub10VV(z, x, y []Word) (c Word)

// Not implemented yet

func add10VW(z, x []Word, y Word) (c Word) {
	return add10VW_g(z, x, y)
}

func sub10VW(z, x []Word, y Word) (c Word) {
	return sub10VW_g(z, x, y)
}

func shl10VU(z, x []Word, s uint) (c Word) {
	return shl10VU_g(z, x, s)
}

func shr10VU(z, x []Word, s uint) (c Word) {
	return shr10VU_g(z, x, s)
}

func mulAdd10VWW(z, x []Word, y, r Word) (c Word) {
	return mulAdd10VWW_g(z, x, y, r)
}

func addMul10VVW(z, x []Word, y Word) (c Word) {
	return addMul10VVW_g(z, x, y)
}

func div10WVW(z []Word, xn Word, x []Word, y Word) (r Word) {
	return div10WVW_g(z, xn, x, y)
}
