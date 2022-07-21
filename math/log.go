package math

import (
	"github.com/db47h/decimal"
)

// Log sets z to the natural logarithm of x, and returns z.
//
// If z's precision is 0, it is changed to x's precision before the operation.
// Rounding is performed according to z's precision and rounding mode.
//
// The function panics if z < 0. The value of z is undefined in that case.
func Log(z *decimal.Decimal, x *decimal.Decimal) *decimal.Decimal {
	// Log uses the Salamin algorithm described in Michael Beeler, R. William
	// Gosper, Richard Schroeppel, HAKMEM, Artificial Intelligence Memo No. 239,
	// Item 143.
	//
	// Another source describes a possibly faster algorithm that builds on top of
	// this by variable substitution and a different pre-scaling, but I first need
	// to understand how to get it right for decimal floats. See Sasaki, T.; Kanada,
	// Y. (1982). "Practically fast multiple-precision evaluation of log(x)".
	// Journal of Information Processing. 5 (4): 247–250
	if z == x {
		z = new(decimal.Decimal).SetMode(z.Mode()).SetPrec(z.Prec())
	}

	prec := z.Prec()
	if prec == 0 {
		prec = x.Prec()
	}
	p := prec + decimal.DigitsPerWord

	// special cases
	switch x.Sign() {
	case -1: // x < 0
		panic(decimal.ErrNaN{Msg: "natural logarithm of a negative number"})
	case 0: // log(0) = -inf
		return z.SetInf(true).SetPrec(prec)
	}
	// ln(+inf) = +inf
	if x.IsInf() {
		return z.SetInf(false).SetPrec(prec)
	}

	// save z mode and switch to ToNearestEven
	mode := z.Mode()
	z.SetMode(decimal.ToNearestEven).SetPrec(p)

	// more special cases.
	neg := false
	switch x.Cmp(one) {
	case 0: // ln(1) = 0
		return z.SetUint64(0).SetMode(mode).SetPrec(prec)
	case -1: // x < 0, log(x) = -log(1/x)
		neg = true
		z.Quo(one, x)
	default:
		z.Set(x)
	}

	// scale z by 10^m so that z×10^m > 2/sqrt(epsilon)
	// with epsilon = 1×10^-p, 2/sqrt(epsilon) = 2×10^(p/2).
	// In order to account for odd precisions, we will scale to 2×10^((p+1)/2)
	// z is mant×10^exp where mant < 1 or mant1×10^(exp-1) and 1 <= mant1 < 10
	// Supposing a worst case where mant1 <= 2, scaling the exponent so that
	// m+exp-1 > (p+1)/2 gives m > (p+1)/2-exp+1 => m = (p+1)/2-exp+2
	m := (int(p)+1)/2 - z.MantExp(nil) + 2
	if m > 0 {
		z.SetMantExp(z, m)
	}

	t := new(decimal.Decimal).SetPrec(p).SetUint64(1)
	u := new(decimal.Decimal).SetPrec(p).Quo(four, z)
	agm(z, t, u).Quo(pi(p), t.Mul(z, two))
	if m > 0 {
		// scale back: z-m×log(10)
		t.Mul(u.SetUint64(uint64(m)), log10(p))
		z.Sub(z, t)
	}
	if neg {
		z.Neg(z)
	}
	return z.SetMode(mode).SetPrec(prec)
}

var _log10 = new(decimal.Decimal).SetPrec(0)

// log10 returns log(10) with a precision that is guaranteed to be at least prec digits.
func log10(prec uint) *decimal.Decimal {
	if _log10.Prec() < prec {
		__log10(_log10.SetPrec(prec))
	}
	return _log10
}

// __log10 computes log(10) to z.Prec() decimal digits of precision and
// returns z. If z.Prec() is zero, it is set to decimal.DefaultDecimalPrec.
//
// log10 is a special case of log() where no actual value for log(10) is needed.
// For log(10) we can easily pre-scale x by doing x=x^10 until x < 1/sqrt(epsilon);
// scaling the result back is done by simply adjustiung its exponent.
func __log10(z *decimal.Decimal) *decimal.Decimal {
	prec := z.Prec()
	if prec == 0 {
		prec = decimal.DefaultDecimalPrec
	}
	p := prec + decimal.DigitsPerWord

	mode := z.Mode()
	z.SetMode(decimal.ToNearestEven).SetPrec(p)

	// see the general case for details about pre-scaling x. In this specific
	// case,  with x = 10^n, we need to scale so that 1×10^n > 2×10^((prec+1)/2)
	//  => n > (prec+1)/2...
	exp := 1
	k := 0
	for eps := int(p+1) / 2; exp <= eps; exp *= 10 {
		k++
	}
	x := decimal.NewDecimal(1, exp).SetPrec(p)
	agm(z,
		new(decimal.Decimal).SetPrec(p).SetUint64(1),
		new(decimal.Decimal).SetPrec(p).Quo(four, x))
	z.Quo(pi(p), x.Mul(z, two))

	// reverse scaling
	return z.SetMantExp(z, -k).SetMode(mode).SetPrec(prec)
}

// agm sets z to the algebraic-geometric-mean of a, b and returns z.
// a, b and z must be distinct decimals. a and b are not preserved.
func agm(z, a, b *decimal.Decimal) *decimal.Decimal {
	var (
		prec    = z.Prec()
		t       = new(decimal.Decimal).SetPrec(prec)
		epsilon = decimal.NewDecimal(1, -int(prec))
	)

	for {
		t.Copy(a)
		a.Mul(z.Add(a, b), half) // a_n+1 = (a_n+b_n)/2
		b.Sqrt(z.Mul(t, b))      // b_n+1 = sqrt(a_n × b_n)
		if z.Sub(a, b).CmpAbs(epsilon) <= 0 {
			break
		}
	}
	return z.Copy(a)
}
