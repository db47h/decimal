package math

import (
	"github.com/db47h/decimal"
)

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
		return z.SetUint64(1).SetPrec(prec)
	}
	if x.IsInf() {
		return z.SetInf(x.Signbit()).SetPrec(prec)
	}

	var (
		p    = prec
		mode = z.Mode()
	)

	// TODO: if |x| < 1×10^1, we need to increase the precision more than DigitsPerWord.
	// i.e. If prec = 34, e^(1×10^-33) = 1.000000000000000000000000000000001e-33
	// That's 32 0s. In this extreme case, p = (3*prec) - 2 for expm1T
	// to return the proper result. This is an attempt to scale the precision up
	// only when needed.
	if exp := x.MantExp(nil); exp < 0 && -exp < int(prec) {
		p += 2*uint(-exp) + 2
	} else {
		p += decimal.DigitsPerWord
	}

	z.SetMode(decimal.ToNearestEven).SetPrec(p)

	// solve:
	//	z = Exp(x) => Log(z) - x = 0
	// or, for ExpM1:
	//  z = Exp(x) - 1 => Log(z+1) - x = 0

	// fallback: taylor series.

	return expm1T(z, x).SetMode(mode).SetPrec(prec)
}

// expm1T sets z to the rounded value of e^x-1, and returns z.
// expm1T uses the Maclaurin series expansion for e^x-1:
//
//	x + x^2/2! + x^3/3! + ...
//
// The precision of z must be non zero and the caller is responsible
// for allocating guard digits and rounding down z.
//
// For example, get e^x-1 with the same precision as x:
//	z.SetPrec(x.Prec()+decimal.DigitsPreWord)
//	expm1T(z, x).SetPrec(x.Prec())
//
func expm1T(z, x *decimal.Decimal) *decimal.Decimal {
	// Large negative values require a lot of precision if applying the series
	// expansion or end up with very wrong results. In order to solve this, we
	// substitute exp(-x) with 1/exp(x):
	//	z = e^x - 1
	//	If x < 0: z = (1/e^-x) - 1
	//	let E(x) = (e^-x) - 1
	//	=> z = 1/(E(x)+1) - 1
	// This second step preserves the property (e^x)-1 = x for |x| < 1×10^prec:
	//	=> z = 1/(E(x)+1) - (E(x)+1)/(E(x)+1) => z = -E(x) / (E(x) + 1).
	// This will preserve precision.
	if x.Signbit() {
		expm1T(z, x.Neg(x))
		t := new(decimal.Decimal).SetPrec(z.Prec()).Add(z, one)
		z.Quo(z.Neg(z), t)
		x.Neg(x)
		return z
	}

	if x.IsZero() {
		return z.SetUint64(1)
	}
	if x.IsInf() {
		return z.SetInf(x.Signbit())
	}

	// 0 < x < +Inf

	var (
		p       = z.Prec()
		q       = new(decimal.Decimal).SetUint64(1)
		fq      = dec(p).Set(one) // q!
		xn      = dec(p)          // x^n
		sum     = dec(p)
		t       = dec(p)
		epsilon = decimal.NewDecimal(1, -int(p))
		exp     int
	)

	// scale x down for x >= 1: e^(x×10^n) = (e^x)^(10^n)
	// this greatly improves performance for any x >= 1.
	if exp = x.MantExp(nil); exp > 0 {
		x.MantExp(x)
	}

	// 0 < x < 1

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

	return z
}
