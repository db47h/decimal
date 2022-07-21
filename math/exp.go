package math

import (
	"github.com/db47h/decimal"
)

func Exp(z, x *decimal.Decimal) *decimal.Decimal {
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
		p    = prec + decimal.DigitsPerWord
		mode = z.Mode()
	)

	z.SetMode(decimal.ToNearestEven).SetPrec(p)

	// solve:
	//	z = Exp(x) => Log(z) - x = 0
	// or, for ExpM1:
	//  z = Exp(x) - 1 => Log(z+1) - x = 0

	panic("not implemented")

	return z.SetMode(mode).SetPrec(prec)
}

// expm1T sets z to the rounded value of e^x-1, and returns z.
// The precision of z must be non zero and the caller is responsible
// for allocating guard digits and rounding down z.
//
// For example, get e^x-1 with the same precision as x:
//	z.SetPrec(x.Prec()+decimal.DigitsPreWord)
//	expm1T(z, x).SetPrec(x.Prec())
//
func expm1T(z, x *decimal.Decimal) *decimal.Decimal {
	if x.IsZero() {
		return z.SetUint64(1)
	}
	if x.IsInf() {
		return z.SetInf(x.Signbit())
	}

	var (
		p       = z.Prec()
		q       = dec(p).Set(one)
		fact    = new(decimal.Decimal).Set(one)
		t       = dec(p)
		xe      = dec(p).Set(x)
		s       = dec(p).Set(x) // first term
		epsilon = decimal.NewDecimal(1, -int(p))
	)
	z.SetPrec(p)
	for {
		xe.Set(t.Mul(xe, x))
		fact.Set(t.Mul(fact, q.Add(q, one)))
		z.Set(s)
		s.Add(z, t.Quo(xe, fact))
		if t.Sub(z, s).CmpAbs(epsilon) <= 0 {
			break
		}
	}
	return z
}
