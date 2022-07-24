package math

import (
	"github.com/db47h/decimal"
)

func Exp(z, x *decimal.Decimal) *decimal.Decimal {
	return z.Add(Expm1(z, x), one)
}

func Expm1(z, x *decimal.Decimal) *decimal.Decimal {
	if z == x {
		z = new(decimal.Decimal).SetMode(x.Mode()).SetPrec(x.Prec())
	}
	prec := z.Prec()
	if prec == 0 {
		prec = x.Prec()
	}

	// special cases
	if x.IsZero() {
		return z.SetUint64(0).SetPrec(prec)
	}
	if x.IsInf() {
		if x.Signbit() {
			return z.SetInt64(-1).SetPrec(prec)
		}
		return z.SetInf(false).SetPrec(prec)
	}

	var (
		p    = prec
		mode = z.Mode()
		exp  = x.MantExp(nil)
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

	// scale x down for x >= 1: e^(x×10^n) = (e^x)^(10^n)
	// this greatly improves performance for any x >= 1.
	if exp > 0 {
		x.MantExp(x)
	}

	// 0 < x < 1

	// If |x| < 1×10^1, we need to increase the precision more than DigitsPerWord.
	// i.e. If prec = 34, e^(1×10^-33) = 1.000000000000000000000000000000001e-33
	// That's 32 0s. In this extreme case, p = (3*prec) - 2 for expm1T
	// to return the proper result.
	// TODO: This is an attempt to scale the precision up only when needed.
	// Testing showed that this works, but we need to prove that this is correct.
	if exp < 0 && -exp < int(prec) {
		p += 2*uint(-exp) + 2
	} else {
		p += decimal.DigitsPerWord
	}

	z.SetMode(decimal.ToNearestEven).SetPrec(p)
	t.SetPrec(p)

	expm1T(z, x)

	// Scale back up. With x >= 1, we can safely add 1 and subtract again
	// without loss of precision in the final result.
	if exp > 0 {
		// Since the largest x for which e^x is representable as a Decimal is
		// about 4.944763833×10^9, z will overflow (+Inf) before n, so the
		// result accuracy of t.Uint64() can be ignored.
		n, _ := t.SetMantExp(t.SetUint64(1), exp).Uint64()
		z.Sub(pow(z, t.Add(z, one), n), one)
		// restore x
		x.SetMantExp(x, exp)
	}

	if neg {
		x.Neg(x)
		if z.IsInf() {
			return z.SetUint64(0)
		}
		t.SetPrec(p).Add(z, one)
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

	// 0 < x < +Inf

	var (
		p       = z.Prec()
		q       = new(decimal.Decimal).SetUint64(1)
		fq      = dec(p).Set(one) // q!
		xn      = dec(p)          // x^n
		sum     = dec(p)
		t       = dec(p)
		epsilon = decimal.NewDecimal(1, -int(p))
	)

	// term 1 for sum and x^n
	sum.Set(xn.Set(x))

	// Maclaurin expansion
	for {
		xn.Set(t.Mul(xn, x))
		fq.Set(t.Mul(fq, q.Add(q, one)))
		sum.Add(z.Set(sum), t.Quo(xn, fq))
		if t.Sub(z, sum).CmpAbs(epsilon) <= 0 {
			break
		}
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
