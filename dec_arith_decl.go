// +build !decimal_pure_go,amd64

package decimal

// implemented in dec_arith_$GOARCH.s
func mul10WW(x, y Word) (z1, z0 Word)

func div10WW(x1, x0, y Word) (q, r Word)

func add10VV(z, x, y []Word) (c Word)

func div10W(n1, n0 Word) (q, r Word)

func sub10VV(z, x, y []Word) (c Word)

func add10VW(z, x []Word, y Word) (c Word)

func sub10VW(z, x []Word, y Word) (c Word)

func shl10VU(z, x []Word, s uint) (c Word)

func shr10VU(z, x []Word, s uint) (c Word)

func mulAdd10VWW(z, x []Word, y, r Word) (c Word)

func addMul10VVW(z, x []Word, y Word) (c Word)

func div10VWW(z, x []Word, y, xn Word) (r Word)
