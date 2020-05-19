package decimal

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import "math"

var (
	three = NewDecimal(3.0)
	two   = NewDecimal(2.0)
)

// Sqrt sets z to the rounded square root of x, and returns it.
//
// If z's precision is 0, it is changed to x's precision before the
// operation. Rounding is performed according to z's precision and
// rounding mode.
//
// The function panics if z < 0. The value of z is undefined in that
// case.
func (z *Decimal) Sqrt(x *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
	}

	if z.prec == 0 {
		z.prec = x.prec
	}

	if x.Sign() == -1 {
		// following IEEE754-2008 (section 7.2)
		panic(ErrNaN{"square root of negative operand"})
	}

	// handle ±0 and +∞
	if x.form != finite {
		z.acc = Exact
		z.form = x.form
		z.neg = x.neg // IEEE754-2008 requires √±0 = ±0
		return z
	}

	// MantExp sets the argument's precision to the receiver's, and
	// when z.prec > x.prec this will lower z.prec. Restore it after
	// the MantExp call.
	prec := z.prec
	b := x.MantExp(z)
	z.prec = prec

	// Compute √(z·10**b) as
	//   √( z)·10**(½b)     if b is even
	//   √(10z)·10**(⌊½b⌋)   if b > 0 is odd
	//   √(z/10)·10**(⌈½b⌉)   if b < 0 is odd
	switch b % 2 {
	case 0:
		// nothing to do
	case 1:
		z.exp++
	case -1:
		z.exp--
	}
	// 0.01 <= z < 10.0

	// Unlike with big.Float, solving x² - z = 0 directly is faster only for
	// very small precisions (<_DW/2).

	// Compute √x (to z.prec precision) by solving
	//   1/t² - x = 0
	// for t (using Newton's method), and then inverting.

	// let
	//   f(t) = 1/t² - x
	// then
	//   g(t) = f(t)/f'(t) = -½t(1 - xt²)
	// and the next guess is given by
	//   t2 = t - g(t) = ½t(3 - xt²)

	u := newDecimal(prec)
	v := newDecimal(prec)
	xf, _ := x.Float64()
	sqi := newDecimal(prec)
	sqi.SetFloat64(1 / math.Sqrt(xf))
	for prec := prec + _DW; sqi.prec < prec; {
		sqi.prec *= 2
		u.prec = sqi.prec
		v.prec = sqi.prec
		u.Mul(sqi, sqi) //   u = sqi²
		u.Mul(x, u)     //     = x.sqi²
		v.Sub(three, u) //   v = 3 - x.sqi²
		u.Mul(sqi, v)   //   u = sqi(3 - x.sqi²)
		sqi.Quo(u, two) // sqi = ½sqi(3 - x.sqi²)
	}
	// sqi = 1/√x

	// x/√x = √x
	z.Mul(x, sqi)

	// re-attach halved exponent
	return z.SetMantExp(z, b/2)
}

// newDecimal returns a new *Decimal with space for twice the given
// precision.
func newDecimal(prec2 uint32) *Decimal {
	z := new(Decimal)
	// dec.make ensures the slice length is > 0
	z.mant = z.mant.make(int(prec2/_DW) * 2)
	return z
}
