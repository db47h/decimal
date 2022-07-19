package math

import (
	"github.com/db47h/decimal"
)

var _pi = pi(new(decimal.Decimal).SetPrec(decimal.DefaultDecimalPrec * 2))

func Pi(z *decimal.Decimal) *decimal.Decimal {
	// TODO: the _pi cache needs locking. Or document that it's not safe for concurrent use
	// should it need to be re-computed.
	if z.Prec() == 0 {
		z.SetPrec(decimal.DefaultDecimalPrec)
	}
	if z.Prec() > _pi.Prec() {
		pi(_pi)
	}
	return z.Set(_pi)
}

// constants for pi()
var (
	one     = new(decimal.Decimal).SetUint64(1)
	two     = new(decimal.Decimal).SetUint64(2)
	four    = new(decimal.Decimal).SetUint64(4)
	half    = decimal.NewDecimal(5, -1)
	quarter = decimal.NewDecimal(25, -2)
)

func precWords(prec uint) uint { return (prec+(decimal.DigitsPerWord-1))/decimal.DigitsPerWord + 1 }

// allocDec pre-allocates storage for a decimal of precision prec with one
// additional word of storage and returns the new decimal with its precision set
// to prec.
func allocDec(prec uint) *decimal.Decimal {
	return new(decimal.Decimal).SetPrec(prec).SetBitsExp(make([]decimal.Word, 0, precWords(prec)), 0)
}

// pi computes π with the Gauss-Legendre algorithm to z.Prec() decimal digits of
// precision and returns z. If z.Prec() is zero, it is set to decimal.DefaultDecimalPrec.
func pi(z *decimal.Decimal) *decimal.Decimal {
	prec := z.Prec()
	if prec == 0 {
		prec = decimal.DefaultDecimalPrec
	}

	var (
		// Increase precision. With only 2 or 4 additional digits there are
		// specific digit counts for which the last digit is off by one (eg. at
		// 57 and 761 respectively). Since increasing the precision may result
		// in increasing the decimals storage by one Word anyway, we just go
		// ahead and add a whole word of precision.
		pp = prec + decimal.DigitsPerWord
		a  = allocDec(pp).SetUint64(1)
		u  = allocDec(pp).Sqrt(two)
		b  = allocDec(pp).Quo(one, u)
		t  = allocDec(pp).Set(quarter)
		// while p is an int, just use a decimal. This causes less mallocs.
		p       = new(decimal.Decimal).SetPrec(pp).SetUint64(1)
		epsilon = decimal.NewDecimal(1, -int(pp))
	)

	// z is also used as a temp value, pre-allocate it if necessary and increase its precision temporarily.

	words := precWords(pp)
	if bits, _ := z.SetPrec(pp).BitsExp(); cap(bits) < int(words) {
		z.SetBitsExp(make([]decimal.Word, 0, words), 0)
	}

	for {
		u.Set(a)                 // a_n
		a.Mul(z.Add(a, b), half) // a_n+1
		b.Sqrt(z.Mul(u, b))      // b_n+1

		// t = t - p×(a_n - a_n+1)^2 could be computed as:
		// t.Sub(t, u.Mul(pd, u.Mul(u.Sub(u, a), u)))
		// but we shuffle temp vars in order to avoid using arguments as targets
		// in operations, which may result in memory allocations for temp
		// storage in the decimal package.
		t.Set(u.Sub(t, z.Mul(u.Mul(z.Sub(u, a), z), p)))

		if z.Sub(a, b).CmpAbs(epsilon) <= 0 {
			break
		}

		p.Set(z.Mul(p, two))
	}
	z.Add(a, b)
	a.Mul(z, z)
	t.Mul(t, four)
	return z.Quo(a, t).SetPrec(prec)
}
