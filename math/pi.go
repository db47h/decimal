package math

import (
	"math/big"

	"github.com/db47h/decimal"
)

var _pi = pi(new(decimal.Decimal).SetPrec(decimal.DefaultDecimalPrec * 2))

func Pi(z *decimal.Decimal) *decimal.Decimal {
	if z.Prec() == 0 {
		z.SetPrec(decimal.DefaultDecimalPrec)
	}
	if z.Prec() > _pi.Prec() {
		pi(_pi)
	}
	return z.Set(_pi)
}

var (
	// constants for computePi
	one     = big.NewInt(1)
	twelve  = big.NewInt(12)
	sixteen = big.NewInt(16)
	la      = big.NewInt(545140134)
	xm      = new(decimal.Decimal).SetPrec(18).SetInt64(-262537412640768000)
	c1      = new(decimal.Decimal).SetPrec(9).SetUint64(426880)
	c2      = new(decimal.Decimal).SetPrec(9).SetUint64(10005)
)

// pi computes π with the Chudnovsky algorithm to z.Prec() decimal digits of precision and returns z.
//
func pi(z *decimal.Decimal) *decimal.Decimal {
	prec := z.Prec()
	if prec == 0 {
		prec = decimal.DefaultDecimalPrec
	}
	var (
		// q, k, l and m stay fairly small (storage-wise) compared to decimals
		// of precision prec, even m (~1/4 of z). Using big.Int for these
		// improves performance and reduces memory usage.
		// While x is also an integer, it grows much larger than prec decimal digits:
		// ~7e24500 for prec=20000, that's a big.Int made of 1250 Words vs.
		// ~1000 Words for decimals of this precision. Using a big.Int for x
		// triples the run time of the algorithm for prec=20000.
		q  = new(big.Int)
		k  = big.NewInt(-6)
		l  = big.NewInt(13591409)
		m  = big.NewInt(1)
		ti = new(big.Int) // temp int value
		// Increase precision. With only 2 or 4 additional digits there are
		// specific digit counts for which the last digit is off by one (eg. at
		// 57 and 761 respectively). Since increasing the precision may result
		// in increasing the decimals storage by one Word anyway, we just go
		// ahead and add a whole word of precision.
		p    = prec + decimal.DigitsPerWord
		x    = new(decimal.Decimal).SetPrec(p).SetUint64(1)
		sum  = new(decimal.Decimal).SetPrec(p)
		last = new(decimal.Decimal).SetPrec(p) // last sum value for comparison
	)

	// z is also used as a temp value, so increase its precision temporarily.
	z.SetPrec(p)

	for {
		// s = s + (m * l) / x
		sum.Add(sum, z.Quo(z.SetInt(ti.Mul(m, l)), x))
		if last.Cmp(sum) == 0 {
			break
		}
		// k = k + 12
		k.Add(k, twelve)
		// l = l + 545140134
		l.Add(l, la)
		// m = m * (k(k^2-16)) / (q+1)^3
		m.Mul(m, ti.Mul(k, ti.Sub(ti.Mul(k, k), sixteen)))
		q.Add(q, one)
		ti.Mul(ti.Mul(q, q), q)
		m.Quo(m, ti)
		// x = x × -262537412640768000
		x.Mul(x, xm)
		last.Copy(sum)
	}
	return z.Quo(z.Mul(z.Sqrt(c2), c1), sum).SetPrec(prec)
}
