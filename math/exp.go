package math

import (
	"math"

	"github.com/db47h/decimal"
)

func Exp(z, x *decimal.Decimal) *decimal.Decimal {
	return z.Add(Expm1(z, x), one)
}

var maxExpExp = 1 + int(math.Ceil(math.Log10(math.Ln10*decimal.MaxExp)))

func Expm1(z, x *decimal.Decimal) *decimal.Decimal {
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
	if x.IsInf() || exp >= maxExpExp {
		if x.Signbit() {
			return z.SetInt64(-1).SetPrec(prec)
		}
		return z.SetInf(false).SetPrec(prec)
	}

	// Exp(x) <= 10^-prec
	// x^10exp <= -prec(log(10))
	//

	// TODO: confirm this and find a way to avoid the NewDecimal.
	if x.CmpAbs(decimal.NewDecimal(9, -int(prec)-1)) <= 0 {
		if exp < 0 {
			return z.Set(x).SetPrec(prec)
		} else {
			return z.SetInt64(-1).SetPrec(prec)
		}
	}

	var (
		p    uint
		mode = z.Mode()
		neg  = x.Signbit()
		t    = new(decimal.Decimal)
	)

	// Large negative values require a lot of precision if applying the series
	// expansion or end up with very wrong results. In order to solve this, we
	// substitute exp(-x) with 1/exp(x):
	//	z = e^x - 1
	//	If x < 0: z = (1/e^-x) - 1
	//	let E(x) = (e^-x) - 1
	//	=> z = 1/(E(x)+1) - 1
	// This second step preserves the property (e^x)-1 = x for |x| < 1×10^prec:
	//	=> z = 1/(E(x)+1) - (E(x)+1)/(E(x)+1) => z = -E(x) / (E(x) + 1).
	// This will preserve precision for e^x - 1.
	if neg {
		x.Neg(x)
	}

	// 0 < x < +inf

	// scale x down for x >= 0.1: e^(x×10^n) = (e^x)^(10^n)
	// Scaling x to be < 0.1 instead of < 1 reduces iterations by about a third.
	if exp >= 0 {
		x.MantExp(x)
		x.SetMantExp(x, -1)
		exp++
	}

	// 0 < x < 1

	// If |x| < 1×10^1, we need to increase the precision more than DigitsPerWord.
	// i.e. If prec = 34, e^(1×10^-33) = 1.000000000000000000000000000000001e-33
	// That's 32 0s. In this extreme case, p = (3*prec) - 2 for expm1T
	// to return the proper result.
	// TODO: This is an attempt to scale the precision up only when needed.
	// Testing showed that this works, but we need to prove that this is correct.
	// TODO: gate x^exp such that too small exponents return 0 and too large return +Inf
	// largest number is 1×10^MaxExp. So max X is ln(10^MaxExp) = MaxExp×ln(10). Anything with exp >= 11 is +Inf

	if exp < 0 {
		if -exp < int(prec) {
			p = 2*uint(-exp) + 2
		} else {
			p = 2
		}
	} else {
		p = uint(exp) + 2
	}
	p += prec
	if p < 4 {
		p = 4
	}

	z.SetMode(decimal.ToNearestEven).SetPrec(p)
	t.SetPrec(p)

	expm1T(z, x)

	// Scale back up. With x >= 1, we can safely add 1 and subtract again
	// without loss of precision in the final result.
	if exp > 0 {
		// with exp clamped at 11, upow will not overflow. z may overflow,
		// but this is expected and will produce +Inf.
		pow(z, t.Add(z, one), upow(10, uint64(exp)))
		// z.Sub(z×10^LARGE, 1) does not check that z-1 will not change z's value. Subtract only if needed.
		// TODO: test edge case and proof
		if z.MantExp(nil) < int(prec) {
			z.Sub(z, one)
		}
		// restore x
		x.SetMantExp(x, exp)
	}

	if neg {
		x.Neg(x)
		if z.IsInf() {
			return z.SetUint64(0).SetPrec(prec)
		}
		// same as above, add only if z is small enough.
		// TODO: test edge case and proof
		if e := z.MantExp(nil); e <= int(prec) {
			t.Add(z, one)
		} else {
			t.Set(z)
		}
		z.Quo(z.Neg(z), t)
	}

	return z.SetMode(mode).SetPrec(prec)
}

// expm1T sets z to the rounded value of e^x-1, and returns z.
// x must be in the open interval (0, 1).
// expm1T uses the Maclaurin series expansion for e^x-1:
//
//  x + x^2/2! + x^3/3! + ...
//
// The precision of z must be non zero and the caller is responsible for
// allocating guard digits and rounding down z.
//
// For example, get e^x-1 with the same precision as x:
//  z.SetPrec(x.Prec()+decimal.DigitsPreWord)
//  expm1T(z, x).SetPrec(x.Prec())
//
func expm1T(z, x *decimal.Decimal) *decimal.Decimal {

	// 0 < x < 0.1

	var (
		p   = z.Prec()
		q   = new(decimal.Decimal).SetUint64(2)
		xn  = dec(p) // x^n / !q
		sum = dec(p)
		t   = dec(p)
	)

	// term 1 for sum and x^n
	sum.Set(xn.Set(x))

	// Maclaurin expansion
	for {
		xn.Quo(t.Mul(xn, x), q)
		// TODO: checking xn should be enough to quit (see z.Sub(z, one) gating above when x < 0)
		sum.Add(z.Set(sum), xn)
		if z.Cmp(sum) == 0 {
			break
		}
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
