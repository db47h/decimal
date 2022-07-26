package math

import (
	"math"

	"github.com/db47h/decimal"
)

// Exponential functions.
//
// Implementation based on:
//
//    T. E. Hull and A. Abrham. 1986. Variable precision exponential function.
//    ACM Trans. Math. Softw. 12, 2 (June 1986), 79–91.
//    https://doi.org/10.1145/6497.6498
//
// The algorithm has been tweaked to support both e^x and e^x-1. The functions Exp
// and Expm1 perform basic sanity checks and handle special cases (stage 1), stages
// 2-5 are performed in expT.
//
// This implementation differs from Hull & Abrham's paper on the following
// points:
// - The upper bound check for x cannot be done as in the paper. Instead, we check
//   if x.exp > 10 since the maximum x for which e^x is finite is 0.4945×10^10.
//   For 0.4945×10^10 < |x| < 1e10, the last stage where we raise z to the power
//   of 10^exp will just overflow to +Inf as expected.
// - The test |x| <= 0.9e-prec is replaced by the more practical x.exp<=-int(prec)
//   which is equivalent to |x| <= 0.99999...×10^-prec or |x| <= (1-10^-prec)×10^prec.
//   The error E in this case still satisfies |E| < 10^-prec for both e^x and e^x-1:
//
//   (1-10^-prec)×10^-prec / (1 - (1-10^-prec)×10^-prec) < 10^-prec simplifies to
//   10^-prec > 0 for prec > 0.
//
// - For values of x where |x| <= 0.9...^-prec, Hull & Abrham return 1 (0 for e^x-1).
//   For the e^x-1 case, we return x as this would be the result returned if this
//   short-circuit was disabled.
// - |x| is normalized to the interval (0, .1) for which the series converge with
//	 about a third less iterations than for (0, 1).
// - For e^x-1, when x.exp < 0, the precision needs to be increased by -exp*2+2 in
//   order to preserve the leading fixed digits.
//
//   eg.: e^(1e-4)-1 = 1.000050001...e-4, rounded to 5 digits: 1.0001e-4.
//
// - TODO: Need mathematical proof for p += -exp*2+2. Test results are however
//   correct for the extreme case where -exp=prec-1 with p >= z.prec-exp*2+2.

func Exp(z, x *decimal.Decimal) *decimal.Decimal {
	if z == x {
		z = new(decimal.Decimal).SetMode(x.Mode()).SetPrec(x.Prec())
	}

	prec := z.Prec()
	if prec == 0 {
		prec = x.Prec()
		z.SetPrec(prec)
	}
	exp := x.MantExp(nil)

	// special cases
	if x.IsZero() {
		return z.SetUint64(1)
	}
	if x.IsInf() || exp > maxExpExp {
		if x.Signbit() {
			return z.SetInt64(0)
		}
		return z.SetInf(false)
	}
	// |x.mant×10^x.exp| <= 0.999999999999999...×10^-prec => e^x-1 = x
	if exp <= -int(prec) {
		return z.SetUint64(1)
	}

	return expT(z, x, false)
}

var maxExpExp = int(math.Ceil(math.Log10(math.Ln10 * decimal.MaxExp)))

func Expm1(z, x *decimal.Decimal) *decimal.Decimal {
	if z == x {
		z = new(decimal.Decimal).SetMode(x.Mode()).SetPrec(x.Prec())
	}

	prec := z.Prec()
	if prec == 0 {
		prec = x.Prec()
		z.SetPrec(prec)
	}
	exp := x.MantExp(nil)

	// special cases
	if x.IsZero() {
		return z.SetUint64(0)
	}
	if x.IsInf() || exp > maxExpExp {
		if x.Signbit() {
			return z.SetInt64(-1)
		}
		return z.SetInf(false)
	}
	// |x.mant×10^x.exp| <= 0.999999999999999...×10^-prec => e^x-1 = x
	if exp <= -int(prec) {
		return z.Set(x)
	}

	return expT(z, x, true)
}

// expT sets z to the rounded value of e^x (if m1 is false), or e^x-1 (if m1 is
// true), and returns z. z and x must be distinct entities and z's precision
// must be > 0.
//
// |x| must be a finite non-zero decimal in the open interval ((1-10^-p)×10^-p, 10^10)
// where p = z.Prec().
//
// expT uses the Maclaurin series expansion for e^x:
//
//  1 + x + x^2/2! + x^3/3! + ...
//
func expT(z, x *decimal.Decimal, m1 bool) *decimal.Decimal {

	// 0.999..×10^-prec < |x| < 1e10

	var (
		exp  = x.MantExp(nil)
		prec = z.Prec()
		p    = prec
	)

	// scale x down for x >= 0.1: e^(x×10^n) = (e^x)^(10^n)
	// Scaling x to be < 0.1 instead of < 1 reduces iterations by about a third.
	if exp >= 0 {
		x.MantExp(x)
		x.SetMantExp(x, -1)
		exp++
	}

	// 0.999..×10^-prec < |x| < 0.1

	if exp >= 0 {
		p += uint(exp) + 2
	} else if m1 {
		// this is specific to the case of e^x-1
		// 0 < -exp < prec
		p += 2*uint(-exp) + 2
	}
	if p < 4 {
		p = 4
	}

	var (
		mode = z.Mode()
		q    = new(decimal.Decimal).SetUint64(2)
		xn   = dec(p).Set(x) // x^n / !q
		t    = dec(p)
	)

	// use z as sum accumulator. Init to 1st term.
	z.SetMode(decimal.ToNearestEven).SetPrec(p).Set(x)

	// Maclaurin expansion
	for {
		xn.Quo(t.Mul(xn, x), q)
		if xn.IsZero() || xn.MantExp(nil) < z.MantExp(nil)-int(p) {
			// xn too small for z.Add(z, xn) to change z.
			break
		}
		z.Add(z, xn)
		q.Add(q, one)
	}

	// Scale back up. If exp > 0, |x| >= 1 and we can safely add 1 and subtract
	// again without loss of precision in the final result.
	if exp > 0 {
		// restore x
		x.SetMantExp(x, exp)
		// with exp clamped at 11, upow will not overflow.
		pow(z, t.Add(z, one), upow(10, uint64(exp)))
		// If z.exp is large enough, z.Sub(z, 1) will not change z's value
		// because of the rounding mode ToNearestEven. On the other hand, if z
		// is close to 0, it will return -1. In both cases, z.Sub will allocate
		// and shift very large mantissae to perform the operation (this is to
		// support other rounding modes but could be optimized).
		// TODO: add this to the TODO list of the decimal package.
		if !m1 || z.IsInf() {
			// e^x, or z=+Inf: nothing to do
		} else if e := z.MantExp(nil); e < -int(p) {
			// z too small for z.Sub(z, 1) to return a result != -1:
			// with p=4, -1+0.9999e-5 = -0.999990001, rounds to -1
			z.SetInt64(-1)
		} else if e <= int(p) {
			// If we subtract a 1 to the right of z.mant, skip the subtraction
			// with p=4, 1234e1-1 = 1233.9e1, rounded to 4 digits again -> 1234e1 = 0.1234e5
			z.Sub(z, one)
		}
	} else if !m1 {
		// we want e^x for 0.999..×10^-prec < |x| < 0.1. e^x-1 is within acceptable bounds
		// for z.Add(z, one) to operate without over shifting mantissae.
		z.Add(z, one)
	}

	return z.SetMode(mode).SetPrec(prec)
}

// expm1 is a prototype that would calculate z=e^x-1 by solving Log(z+1)-x=0 for
// z. initial tests show that this starts to be as fast as the Maclaurin series
// expansion at 10000 digits.
func expm1(z, x *decimal.Decimal) *decimal.Decimal {
	// 0 < x < 1
	var (
		prec = z.Prec()
		p    = prec + decimal.DigitsPerWord
		t    = new(decimal.Decimal).SetPrec(p)
		u    = new(decimal.Decimal).SetPrec(p)
	)
	z.SetPrec(p)

	// initial guess, calculate DigitsPerWord digits using Maclaurin series.
	guess := new(decimal.Decimal).SetPrec(decimal.DigitsPerWord)

	newton(z, expT(guess, x, true), func(z, guess *decimal.Decimal) *decimal.Decimal {
		// f(z)/f'(z) = (Log(z+1)-x)(z+1)
		Log(t, u.Add(guess, one))
		t.Sub(t, x)
		return z.Mul(t, u)
	})
	return z.SetPrec(prec)
}

func newton(z, guess *decimal.Decimal, fOverDf func(z, x *decimal.Decimal) *decimal.Decimal) *decimal.Decimal {
	var (
		prec = z.Prec()
		p    = guess.Prec()
		t    = new(decimal.Decimal).SetMode(z.Mode()).SetPrec(prec)
	)
	guess.SetPrec(prec)
	for {
		z.Sub(guess, fOverDf(t, guess))
		if p *= 2; p >= prec {
			return z
		}
		guess.Set(z)
	}
}
