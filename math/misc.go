package math

import (
	"github.com/db47h/decimal"
)

// constants
var (
	one     = new(decimal.Decimal).SetUint64(1)
	two     = new(decimal.Decimal).SetUint64(2)
	four    = new(decimal.Decimal).SetUint64(4)
	half    = decimal.NewDecimal(5, -1)
	quarter = decimal.NewDecimal(25, -2)
)

func precWords(prec uint) uint { return (prec+(decimal.DigitsPerWord-1))/decimal.DigitsPerWord + 1 }

// dec pre-allocates storage for a decimal of precision prec with one
// additional word of storage and returns the new decimal with its precision set
// to prec.
func dec(prec uint) *decimal.Decimal {
	return new(decimal.Decimal).SetPrec(prec).SetBitsExp(make([]decimal.Word, 0, precWords(prec)), 0)
}

// pow sets z to the rounded value of x^n and returns z. The precision of z must
// be non zero and the caller is responsible for allocating guard digits and
// rounding down z.
func pow(z, x *decimal.Decimal, n uint64) *decimal.Decimal {
	if n == 0 {
		return z.SetUint64(1)
	}
	t := dec(z.Prec())
	y := dec(z.Prec()).SetUint64(1)
	z.Set(x)

	for n > 1 {
		if n%2 != 0 {
			y.Mul(t.Set(y), z)
		}
		z.Mul(t.Set(z), t)
		if z.IsInf() || z.IsZero() {
			return z
		}
		n /= 2
	}
	// TODO: implement a fastpath IsUint64(n) function.
	if y.Cmp(one) == 0 {
		return z
	}
	return z.Mul(t.Set(z), y)
}

func upow(x, n uint64) uint64 {
	if n == 0 {
		return 1
	}
	z := x
	y := uint64(1)
	for n > 1 {
		if n%2 != 0 {
			y *= z
		}
		z *= z
		n /= 2
	}
	return z * y
}
