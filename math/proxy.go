package math

import "github.com/db47h/decimal"

// FMA sets z to x * y + u, computed with only one rounding. (That is, FMA
// performs the fused multiply-add of x, y, and u.) If z's precision is 0, it is
// changed to the larger of x's, y's, or u's precision before the operation.
// Rounding, and accuracy reporting are as for Add. FMA panics with ErrNaN if
// multiplying zero with an infinity, or if adding two infinities with opposite
// signs. The value of z is undefined in that case.
//
// This function is a proxy for z.FMA(x, y, u)
func FMA(z, x, y, u *decimal.Decimal) *decimal.Decimal {
	return z.FMA(x, y, u)
}

// Sqrt sets z to the rounded square root of x, and returns it.
//
// If z's precision is 0, it is changed to x's precision before the
// operation. Rounding is performed according to z's precision and
// rounding mode.
//
// The function panics if z < 0. The value of z is undefined in that
// case.
func Sqrt(z, x *decimal.Decimal) *decimal.Decimal {
	return z.Sqrt(x)
}
