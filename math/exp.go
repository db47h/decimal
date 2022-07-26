package math

import (
	"math"

	"github.com/db47h/decimal"
)

func Exp(z, x *decimal.Decimal) *decimal.Decimal {
	return z.Add(Expm1(z, x), one)
}

var maxExpExp = int(math.Ceil(math.Log10(math.Ln10 * decimal.MaxExp)))

func Expm1(z, x *decimal.Decimal) *decimal.Decimal {
	// Implementation based on:
	//  T. E. Hull and A. Abrham. 1986. Variable precision exponential function.
	//  ACM Trans. Math. Softw. 12, 2 (June 1986), 79–91.
	//  https://doi.org/10.1145/6497.6498
	//
	// Other than returning e^x - 1 instead of e^x, this implementation differs
	// from Hull & Abrham's paper on the following points:
	//  - The upper bound check for x cannot be done as in the paper. Instead,
	//    we check that x.exp > 10 since the maximum x for which e^x is finite
	//    is 0.4945×10^10. For values in-between, the last step where we raise z
	//	  to the power of 10^exp will just overflow to +Inf as expected.
	//  - For values of x where |x| <= 0.9^-prec, Hull & Abrham return 1 (0 for
	//    e^x-1). Here we return x as this would be the result returned if this
	//    short-circuit was disabled.
	//	- The test |x| <= 0.9e-prec is replaced by the more practical x.exp<=-int(prec)
	//	  which is equivalent to |x| <= 0.99999...×10^-prec or |x| <= (1-10^-prec)×10^prec.
	//	  The error E in this case still satisfies |E| < 10^-prec:
	//	  (1-10^-prec)×10^-prec / (1 - (1-10^-prec)×10^-prec) < 10^-prec
	//	  simplifies to 1-10^-prec < 1, which is true for prec > 0.
	//  - x is normalized to the interval (0, .1) for which the series converge faster.
	//  - For x.exp < 0, the precision needs to be increased by -exp*2+2 in
	//	  order to preserve the leading fixed digits. eg.:
	//    e^(1e-4)-1 = 1.000050001...e-4, Rounded to 5 digits: 1.0001e-4.
	//	  TODO: Need mathematical proof for p += -exp*2+2. Test results are however
	//	  correct for the extreme case where -exp=prec-1 with p >= z.prec-exp*2+2.

	if z == x {
		z = new(decimal.Decimal).SetMode(x.Mode()).SetPrec(x.Prec())
	}

	prec := z.Prec()
	if prec == 0 {
		prec = x.Prec()
	}
	exp := x.MantExp(nil)

	// special cases
	if x.IsZero() {
		return z.SetUint64(0).SetPrec(prec)
	}
	if x.IsInf() || exp > maxExpExp {
		if x.Signbit() {
			return z.SetInt64(-1).SetPrec(prec)
		}
		return z.SetInf(false).SetPrec(prec)
	}
	// |x.mant×10^x.exp| <= 0.999999999999999...×10^-prec => e^x-1 = x
	if exp <= -int(prec) {
		return z.Set(x).SetPrec(prec)
	}

	var (
		p    = prec
		mode = z.Mode()
		t    = new(decimal.Decimal)
	)

	// 0.999..×10^-prec < |x| < 1e10

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
	} else {
		// 0 < -exp < prec
		// this is specific to the case of e^x-1 and is not covered by Hull & Abrham.
		p += 2*uint(-exp) + 2
	}
	if p < 4 {
		p = 4
	}

	z.SetMode(decimal.ToNearestEven).SetPrec(p)
	t.SetPrec(p)

	expm1T(z, x)

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
		if z.IsInf() {
			// nothing to do
		} else if e := z.MantExp(nil); e < -int(p) {
			// z too small for z.Sub(z, 1) to return a result != -1:
			// with p=4, -1+0.9999e-5 = -0.999990001, rounds to -1
			z.SetInt64(-1)
		} else if e <= int(p) {
			// If we subtract a 1 to the right of z.mant, skip the subtraction
			// with p=4, 1234e1-1 = 1233.9e1, rounded to 4 digits again -> 1234e1 = 0.1234e5
			z.Sub(z, one)
		}
	}

	return z.SetMode(mode).SetPrec(prec)
}

// expm1T sets z to the rounded value of e^x-1, and returns z.
// |x| must be in the open interval (0, 0.1).
// expm1T uses the Maclaurin series expansion for e^x-1:
//
//  x + x^2/2! + x^3/3! + ...
//
// The precision of z must be non zero and the caller is responsible for
// allocating guard digits and rounding down z.
func expm1T(z, x *decimal.Decimal) *decimal.Decimal {

	// 0 < |x| < 0.1

	var (
		p  = z.Prec()
		q  = new(decimal.Decimal).SetUint64(2)
		xn = dec(p) // x^n / !q
		t  = dec(p)
	)

	// term 1 for sum and x^n
	z.Set(xn.Set(x))

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

	return z
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

	newton(z, expm1T(guess, x), func(z, guess *decimal.Decimal) *decimal.Decimal {
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
