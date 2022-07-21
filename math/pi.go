package math

import (
	"github.com/db47h/decimal"
)

var _pi = new(decimal.Decimal).SetPrec(0)

// Pi sets z to the value of π, with precision z.Prec(), and returns z. If
// z.Prec() is zero, it is set to decimal.DefaultDecimalPrec.
//
// Since many transcendental functions use π internally, Pi caches the computed
// value of π that has the highest precision. Access to this cached value is not
// guarded by a mutex, as a result, Pi, and most transcendental functions are
// not safe for concurrent use without taking precautionary measures.
//
// One strategy around this is to call Pi with at least decimal.DigitsPerWord*2
// additional digits of precision before starting any goroutines that may end up
// calling Pi.
func Pi(z *decimal.Decimal) *decimal.Decimal {
	if z.Prec() == 0 {
		z.SetPrec(decimal.DefaultDecimalPrec)
	}
	return z.Set(pi(z.Prec()))
}

// pi returns _pi with a precision that is guaranteed to be at least prec digits.
func pi(prec uint) *decimal.Decimal {
	if _pi.Prec() < prec {
		__pi(_pi.SetPrec(prec))
	}
	return _pi
}

// __pi computes π with the Gauss-Legendre algorithm to z.Prec() decimal digits of
// precision and returns z. If z.Prec() is zero, it is set to decimal.DefaultDecimalPrec.
func __pi(z *decimal.Decimal) *decimal.Decimal {
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

	// π = (a_n+1 + b_n+1)^2 / 4t
	// since at this point a ≈ b
	// π = (2a)^2/4t = a^2/t
	return z.Quo(u.Mul(a, a), t).SetPrec(prec)
}
