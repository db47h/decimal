package decimal

import (
	"fmt"
	"math"
	"math/big"
)

// DefaultDecimalPrec is the default minimum precision used when creating a new
// Decimal from any other type. An uint64 requires up to 20 digits, which
// amounts to 2 x 19-digits Words (64 bits) or 3 x 9-digits Words (32 bits).
// Forcing the precision to 20 digits would result in 18 or 7 unused digits.
// Using 34 instead gives a higher precision at no performance or memory cost
// and gives room for 2 to 4 extra digits of extra precision for internal
// computations at no performance or memory cost either. Also 34 digits matches
// the precision of IEEE-754 decimal128.
const DefaultDecimalPrec = 34

var decimalZero Decimal

var _ fmt.Scanner = &decimalZero   // *Decimal must implement fmt.Scanner
var _ fmt.Formatter = &decimalZero // *Float must implement fmt.Formatter

type Decimal struct {
	mant dec
	exp  int32
	prec uint32
	mode RoundingMode
	acc  Accuracy
	form form
	neg  bool
}

// NewDecimal allocates and returns a new Float set to x,
// with precision DefaultDecimalPrec and rounding mode ToNearestEven.
// NewDecimal panics with ErrNaN if x is a NaN.
func NewDecimal(x float64) *Decimal {
	if math.IsNaN(x) {
		panic(ErrNaN{"NewFloat(NaN)"})
	}
	return new(Decimal).SetFloat64(x)
}

func (z *Decimal) Abs(x *Decimal) *Decimal {
	panic("not implemented")
}

// Acc returns the accuracy of x produced by the most recent operation.
func (x *Decimal) Acc() Accuracy {
	return x.acc
}

// Handling of sign bit as defined by IEEE 754-2008, section 6.3:
//
// When neither the inputs nor result are NaN, the sign of a product or
// quotient is the exclusive OR of the operands’ signs; the sign of a sum,
// or of a difference x−y regarded as a sum x+(−y), differs from at most
// one of the addends’ signs; and the sign of the result of conversions,
// the quantize operation, the roundToIntegral operations, and the
// roundToIntegralExact (see 5.3.1) is the sign of the first or only operand.
// These rules shall apply even when operands or results are zero or infinite.
//
// When the sum of two operands with opposite signs (or the difference of
// two operands with like signs) is exactly zero, the sign of that sum (or
// difference) shall be +0 in all rounding-direction attributes except
// roundTowardNegative; under that attribute, the sign of an exact zero
// sum (or difference) shall be −0. However, x+x = x−(−x) retains the same
// sign as x even when x is zero.
//
// See also: https://play.golang.org/p/RtH3UCt5IH

// Add sets z to the rounded sum x+y and returns z. If z's precision is 0,
// it is changed to the larger of x's or y's precision before the operation.
// Rounding is performed according to z's precision and rounding mode; and
// z's accuracy reports the result error relative to the exact (not rounded)
// result. Add panics with ErrNaN if x and y are infinities with opposite
// signs. The value of z is undefined in that case.
func (z *Decimal) Add(x, y *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
		y.validate()
	}

	if z.prec == 0 {
		z.prec = umax32(x.prec, y.prec)
	}

	if x.form == finite && y.form == finite {
		// x + y (common case)

		// Below we set z.neg = x.neg, and when z aliases y this will
		// change the y operand's sign. This is fine, because if an
		// operand aliases the receiver it'll be overwritten, but we still
		// want the original x.neg and y.neg values when we evaluate
		// x.neg != y.neg, so we need to save y.neg before setting z.neg.
		yneg := y.neg

		z.neg = x.neg
		if x.neg == yneg {
			// x + y == x + y
			// (-x) + (-y) == -(x + y)
			z.uadd(x, y)
		} else {
			// x + (-y) == x - y == -(y - x)
			// (-x) + y == y - x == -(x - y)
			if x.ucmp(y) > 0 {
				z.usub(x, y)
			} else {
				z.neg = !z.neg
				z.usub(y, x)
			}
		}
		if z.form == zero && z.mode == ToNegativeInf && z.acc == Exact {
			z.neg = true
		}
		return z
	}

	if x.form == inf && y.form == inf && x.neg != y.neg {
		// +Inf + -Inf
		// -Inf + +Inf
		// value of z is undefined but make sure it's valid
		z.acc = Exact
		z.form = zero
		z.neg = false
		panic(ErrNaN{"addition of infinities with opposite signs"})
	}

	if x.form == zero && y.form == zero {
		// ±0 + ±0
		z.acc = Exact
		z.form = zero
		z.neg = x.neg && y.neg // -0 + -0 == -0
		return z
	}

	if x.form == inf || y.form == zero {
		// ±Inf + y
		// x + ±0
		return z.Set(x)
	}

	// ±0 + y
	// x + ±Inf
	return z.Set(y)
}

// z = x + y, ignoring signs of x and y for the addition
// but using the sign of z for rounding the result.
// x and y must have a non-empty mantissa and valid exponent.
func (z *Decimal) uadd(x, y *Decimal) {
	// Note: This implementation requires 2 shifts most of the
	// time. It is also inefficient if exponents or precisions
	// differ by wide margins. The following article describes
	// an efficient (but much more complicated) implementation
	// compatible with the internal representation used here:
	//
	// Vincent Lefèvre: "The Generic Multiple-Precision Floating-
	// Point Addition With Exact Rounding (as in the MPFR Library)"
	// http://www.vinc17.net/research/papers/rnc6.pdf

	if debugDecimal {
		validateBinaryOperands(x, y)
	}

	// compute exponents ex, ey for mantissa with decimal point
	// on the right (mantissa.0) - use int64 to avoid overflow
	ex := int64(x.exp) - int64(len(x.mant))*_DW
	ey := int64(y.exp) - int64(len(y.mant))*_DW

	al := alias(z.mant, x.mant) || alias(z.mant, y.mant)

	// TODO(gri) having a combined add-and-shift primitive
	//           could make this code significantly faster
	switch {
	case ex < ey:
		if al {
			t := dec(nil).shl(y.mant, uint(ey-ex))
			z.mant = z.mant.add(x.mant, t)
		} else {
			z.mant = z.mant.shl(y.mant, uint(ey-ex))
			z.mant = z.mant.add(x.mant, z.mant)
		}
	default:
		// ex == ey, no shift needed
		z.mant = z.mant.add(x.mant, y.mant)
	case ex > ey:
		if al {
			t := dec(nil).shl(x.mant, uint(ex-ey))
			z.mant = z.mant.add(t, y.mant)
		} else {
			z.mant = z.mant.shl(x.mant, uint(ex-ey))
			z.mant = z.mant.add(z.mant, y.mant)
		}
		ex = ey
	}
	// len(z.mant) > 0

	z.setExpAndRound(ex+int64(len(z.mant))*_DW-dnorm(z.mant), 0)
}

// z = x - y for |x| > |y|, ignoring signs of x and y for the subtraction
// but using the sign of z for rounding the result.
// x and y must have a non-empty mantissa and valid exponent.
func (z *Decimal) usub(x, y *Decimal) {
	// This code is symmetric to uadd.
	// We have not factored the common code out because
	// eventually uadd (and usub) should be optimized
	// by special-casing, and the code will diverge.

	if debugDecimal {
		validateBinaryOperands(x, y)
	}

	ex := int64(x.exp) - int64(len(x.mant))*_DW
	ey := int64(y.exp) - int64(len(y.mant))*_DW

	al := alias(z.mant, x.mant) || alias(z.mant, y.mant)

	switch {
	case ex < ey:
		if al {
			t := dec(nil).shl(y.mant, uint(ey-ex))
			z.mant = t.sub(x.mant, t)
		} else {
			z.mant = z.mant.shl(y.mant, uint(ey-ex))
			z.mant = z.mant.sub(x.mant, z.mant)
		}
	default:
		// ex == ey, no shift needed
		z.mant = z.mant.sub(x.mant, y.mant)
	case ex > ey:
		if al {
			t := dec(nil).shl(x.mant, uint(ex-ey))
			z.mant = t.sub(t, y.mant)
		} else {
			z.mant = z.mant.shl(x.mant, uint(ex-ey))
			z.mant = z.mant.sub(z.mant, y.mant)
		}
		ex = ey
	}

	// operands may have canceled each other out
	if len(z.mant) == 0 {
		z.acc = Exact
		z.form = zero
		z.neg = false
		return
	}
	// len(z.mant) > 0

	z.setExpAndRound(ex+int64(len(z.mant))*_DW-dnorm(z.mant), 0)
}

// Cmp compares x and y and returns:
//
//   -1 if x <  y
//    0 if x == y (incl. -0 == 0, -Inf == -Inf, and +Inf == +Inf)
//   +1 if x >  y
//
func (x *Decimal) Cmp(y *Decimal) int {
	if debugDecimal {
		x.validate()
		y.validate()
	}

	mx := x.ord()
	my := y.ord()
	switch {
	case mx < my:
		return -1
	case mx > my:
		return +1
	}
	// mx == my

	// only if |mx| == 1 we have to compare the mantissae
	switch mx {
	case -1:
		return y.ucmp(x)
	case +1:
		return x.ucmp(y)
	}

	return 0
}

// ord classifies x and returns:
//
//	-2 if -Inf == x
//	-1 if -Inf < x < 0
//	 0 if x == 0 (signed or unsigned)
//	+1 if 0 < x < +Inf
//	+2 if x == +Inf
//
func (x *Decimal) ord() int {
	var m int
	switch x.form {
	case finite:
		m = 1
	case zero:
		return 0
	case inf:
		m = 2
	}
	if x.neg {
		m = -m
	}
	return m
}

// ucmp returns -1, 0, or +1, depending on whether
// |x| < |y|, |x| == |y|, or |x| > |y|.
// x and y must have a non-empty mantissa and valid exponent.
func (x *Decimal) ucmp(y *Decimal) int {
	if debugDecimal {
		validateBinaryOperands(x, y)
	}

	switch {
	case x.exp < y.exp:
		return -1
	case x.exp > y.exp:
		return +1
	}
	// x.exp == y.exp

	// compare mantissas
	i := len(x.mant)
	j := len(y.mant)
	for i > 0 || j > 0 {
		var xm, ym Word
		if i > 0 {
			i--
			xm = x.mant[i]
		}
		if j > 0 {
			j--
			ym = y.mant[j]
		}
		switch {
		case xm < ym:
			return -1
		case xm > ym:
			return +1
		}
	}

	return 0
}

// Copy sets z to x, with the same precision, rounding mode, and
// accuracy as x, and returns z. x is not changed even if z and
// x are the same.
func (z *Decimal) Copy(x *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
	}
	if z != x {
		z.prec = x.prec
		z.mode = x.mode
		z.acc = x.acc
		z.form = x.form
		z.neg = x.neg
		if z.form == finite {
			z.mant = z.mant.set(x.mant)
			z.exp = x.exp
		}
	}
	return z
}

// Float returns the big.Float value nearest to x with precision prec and
// RoundingMode set to that of x. The returned accuracy is the accuracy if the
// conversion from base 10 to base 2.
//
// If prec is 0, the result precision will be set to the precision of x
// (converting precision in decimal digits to bits).
//
// Note that for high enough exponents, the result might overflow and be set to
// ±Inf. In this case, accuracy will be either Above or Below, depending on the
// sign of x.
func (x *Decimal) Float(prec uint) (*big.Float, Accuracy) {
	p := uint64(prec)
	if p == 0 {
		p = uint64(float64(x.prec)*log2_10) + 1
		if p < 64 {
			p = 64
		}
	}
	if p > MaxPrec {
		p = MaxPrec
	}

	z := new(big.Float).SetMode(big.RoundingMode(x.mode)).SetPrec(uint(p))

	switch x.form {
	case zero:
		return z, Exact
	case inf:
		return z.SetInf(x.neg), Exact
	}

	// big.Float has no SetBits. Need to use a temp Int.
	var i big.Int
	i.SetBits(decToNat(nil, x.mant))

	exp := int64(x.exp) - int64(len(x.mant)*_DW)
	if exp < MinExp {
		// overflow.
		return z, makeAcc(x.neg)
	}

	z = z.SetInt(&i)

	// z = x.mant * 2**exp * 5**exp
	// Set 2 exponent
	z.SetMantExp(z, int(exp))

	// now multiply/divide by 5**exp
	// add a full Word of precision for exponent conversion
	tp := ((p+_W-1)/_W + 1) * _W
	if tp > MaxPrec {
		tp = MaxPrec
	}
	t := new(big.Float).SetPrec(uint(tp))
	if exp < 0 {
		z.Quo(z, pow5Float(t, uint64(-exp)))
	} else {
		z.Mul(z, pow5Float(t, uint64(exp)))
	}
	if z.IsInf() {
		// inaccurate result
		return z, makeAcc(!x.neg)
	}
	return z, Accuracy(z.Acc())
}

func (x *Decimal) Float32() (float32, Accuracy) {
	panic("not implemented")
}

func (x *Decimal) Float64() (float64, Accuracy) {
	panic("not implemented")
}

func (z *Decimal) GobDecode(buf []byte) error {
	panic("not implemented")
}

func (x *Decimal) GobEncode() ([]byte, error) {
	panic("not implemented")
}

func (x *Decimal) Int(z *big.Int) (*big.Int, Accuracy) {
	panic("not implemented")
}

func (x *Decimal) Int64() (int64, Accuracy) {
	panic("not implemented")
}

// IsInf reports whether x is +Inf or -Inf.
func (x *Decimal) IsInf() bool {
	return x.form == inf
}

func (x *Decimal) IsInt() bool {
	if debugDecimal {
		x.validate()
	}
	// special cases
	if x.form != finite {
		return x.form == zero
	}
	// x.form == finite
	if x.exp <= 0 {
		return false
	}
	// x.exp > 0
	// mant[0:prec] * 10**exp >= 0 || mant[0:mant.MinPrec()]*10**exp >= 0
	return x.prec <= uint32(x.exp) || x.MinPrec() <= uint(x.exp)
}

func (x *Decimal) MantExp(mant *Decimal) (exp int) {
	panic("not implemented")
}

func (x *Decimal) MarshalText() (text []byte, err error) {
	panic("not implemented")
}

// MinPrec returns the minimum precision required to represent x exactly
// (i.e., the smallest prec before x.SetPrec(prec) would start rounding x).
// The result is 0 for |x| == 0 and |x| == Inf.
func (x *Decimal) MinPrec() uint {
	if x.form != finite {
		return 0
	}
	return uint(len(x.mant))*_DW - x.mant.ntz()
}

// Mode returns the rounding mode of x.
func (x *Decimal) Mode() RoundingMode {
	return x.mode
}

// Mul sets z to the rounded product x*y and returns z.
// Precision, rounding, and accuracy reporting are as for Add.
// Mul panics with ErrNaN if one operand is zero and the other
// operand an infinity. The value of z is undefined in that case.
func (z *Decimal) Mul(x, y *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
		y.validate()
	}

	if z.prec == 0 {
		z.prec = umax32(x.prec, y.prec)
	}

	z.neg = x.neg != y.neg

	if x.form == finite && y.form == finite {
		// x * y (common case)
		z.umul(x, y)
		return z
	}

	z.acc = Exact
	if x.form == zero && y.form == inf || x.form == inf && y.form == zero {
		// ±0 * ±Inf
		// ±Inf * ±0
		// value of z is undefined but make sure it's valid
		z.form = zero
		z.neg = false
		panic(ErrNaN{"multiplication of zero with infinity"})
	}

	if x.form == inf || y.form == inf {
		// ±Inf * y
		// x * ±Inf
		z.form = inf
		return z
	}

	// ±0 * y
	// x * ±0
	z.form = zero
	return z
}

// Neg sets z to the (possibly rounded) value of x with its sign negated,
// and returns z.
func (z *Decimal) Neg(x *Decimal) *Decimal {
	z.Set(x)
	z.neg = !z.neg
	return z
}

// Prec returns the mantissa precision of x in bits.
// The result may be 0 for |x| == 0 and |x| == Inf.
func (x *Decimal) Prec() uint {
	return uint(x.prec)
}

// Quo sets z to the rounded quotient x/y and returns z.
// Precision, rounding, and accuracy reporting are as for Add.
// Quo panics with ErrNaN if both operands are zero or infinities.
// The value of z is undefined in that case.
func (z *Decimal) Quo(x, y *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
		y.validate()
	}

	if z.prec == 0 {
		z.prec = umax32(x.prec, y.prec)
	}

	z.neg = x.neg != y.neg

	if x.form == finite && y.form == finite {
		// x / y (common case)
		z.uquo(x, y)
		return z
	}

	z.acc = Exact
	if x.form == zero && y.form == zero || x.form == inf && y.form == inf {
		// ±0 / ±0
		// ±Inf / ±Inf
		// value of z is undefined but make sure it's valid
		z.form = zero
		z.neg = false
		panic(ErrNaN{"division of zero by zero or infinity by infinity"})
	}

	if x.form == zero || y.form == inf {
		// ±0 / y
		// x / ±Inf
		z.form = zero
		return z
	}

	// x / ±0
	// ±Inf / y
	z.form = inf
	return z
}

// z = x / y, ignoring signs of x and y for the division
// but using the sign of z for rounding the result.
// x and y must have a non-empty mantissa and valid exponent.
func (z *Decimal) uquo(x, y *Decimal) {
	if debugDecimal {
		validateBinaryOperands(x, y)
	}

	// mantissa length in words for desired result precision + 1
	// (at least one extra bit so we get the rounding bit after
	// the division)
	n := int(z.prec/_DW) + 1

	// compute adjusted x.mant such that we get enough result precision
	xadj := x.mant
	if d := n - len(x.mant) + len(y.mant); d > 0 {
		// d extra words needed => add d "0 digits" to x
		xadj = make(dec, len(x.mant)+d)
		copy(xadj[d:], x.mant)
	}
	// TODO(db47h): If we have too many digits (d < 0), we should be able
	// to shorten x for faster division. But we must be extra careful
	// with rounding in that case.

	// Compute d before division since there may be aliasing of x.mant
	// (via xadj) or y.mant with z.mant.
	d := len(xadj) - len(y.mant)

	// divide
	var r dec
	z.mant, r = z.mant.div(nil, xadj, y.mant)
	e := int64(x.exp) - int64(y.exp) - int64(d-len(z.mant))*_DW

	// The result is long enough to include (at least) the rounding bit.
	// If there's a non-zero remainder, the corresponding fractional part
	// (if it were computed), would have a non-zero sticky bit (if it were
	// zero, it couldn't have a non-zero remainder).
	var sbit uint
	if len(r) > 0 {
		sbit = 1
	}

	z.setExpAndRound(e-dnorm(z.mant), sbit)
}

func (x *Decimal) Rat(z *big.Rat) (*big.Rat, Accuracy) {
	panic("not implemented")
}

// Set sets z to the (possibly rounded) value of x and returns z.
// If z's precision is 0, it is changed to the precision of x
// before setting z (and rounding will have no effect).
// Rounding is performed according to z's precision and rounding
// mode; and z's accuracy reports the result error relative to the
// exact (not rounded) result.
func (z *Decimal) Set(x *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
	}
	z.acc = Exact
	if z != x {
		z.form = x.form
		z.neg = x.neg
		if x.form == finite {
			z.exp = x.exp
			z.mant = z.mant.set(x.mant)
		}
		if z.prec == 0 {
			z.prec = x.prec
		} else if z.prec < x.prec {
			z.round(0)
		}
	}
	return z
}

func (z *Decimal) SetFloat64(x float64) *Decimal {
	panic("not implemented")
}

// SetInf sets z to the infinite Decimal -Inf if signbit is
// set, or +Inf if signbit is not set, and returns z. The
// precision of z is unchanged and the result is always
// Exact.
func (z *Decimal) SetInf(signbit bool) *Decimal {
	z.acc = Exact
	z.form = inf
	z.neg = signbit
	return z
}

const log2_10 = math.Ln10 / math.Ln2

// SetInt sets z to the (possibly rounded) value of x and returns z.
// If z's precision is 0, it is changed to the larger of x.BitLen()
// or DefaultDecimalPrec (and rounding will have no effect).
func (z *Decimal) SetInt(x *big.Int) *Decimal {
	bits := uint32(x.BitLen())
	prec := uint32(float64(bits)/log2_10) + 1 // off by 1 at most
	// TODO(db47h): adjust precision if needed
	if z.prec == 0 {
		z.prec = umax32(prec, DefaultDecimalPrec)
	}
	// TODO(db47h) truncating x could be more efficient if z.prec > 0
	// but small compared to the size of x, or if there are many trailing 0's.
	z.acc = Exact
	z.neg = x.Sign() < 0
	if bits == 0 {
		z.form = zero
		return z
	}
	// x != 0
	z.mant = z.mant.make((int(prec) + _DW - 1) / _DW).setNat(x.Bits())
	exp := dnorm(z.mant)
	z.setExpAndRound(int64(len(z.mant))*_DW-exp, 0)
	return z
}

func (z *Decimal) setBits64(neg bool, x uint64) *Decimal {
	if z.prec == 0 {
		z.prec = DefaultDecimalPrec
	}
	z.acc = Exact
	z.neg = neg
	if x == 0 {
		z.form = zero
		return z
	}
	// x != 0
	z.form = finite
	z.mant, z.exp = z.mant.setUint64(x)
	dnorm(z.mant)
	if z.prec < 20 {
		z.round(0)
	}
	return z
}

// SetInt64 sets z to the (possibly rounded) value of x and returns z. If z's
// precision is 0, it is changed to DefaultDecimalPrec (and rounding will have
// no effect).
func (z *Decimal) SetInt64(x int64) *Decimal {
	u := x
	if u < 0 {
		u = -u
	}
	// We cannot simply call z.SetUint64(uint64(u)) and change
	// the sign afterwards because the sign affects rounding.
	return z.setBits64(x < 0, uint64(u))
}

func (z *Decimal) setExpAndRound(exp int64, sbit uint) {
	if exp < MinExp {
		// underflow
		z.acc = makeAcc(z.neg)
		z.form = zero
		return
	}

	if exp > MaxExp {
		// overflow
		z.acc = makeAcc(!z.neg)
		z.form = inf
		return
	}

	z.form = finite
	z.exp = int32(exp)
	z.round(sbit)
}

func (z *Decimal) SetMantExp(mant *Decimal, exp int) *Decimal {
	panic("not implemented")
}

// SetMode sets z's rounding mode to mode and returns an exact z.
// z remains unchanged otherwise.
// z.SetMode(z.Mode()) is a cheap way to set z's accuracy to Exact.
func (z *Decimal) SetMode(mode RoundingMode) *Decimal {
	z.mode = mode
	z.acc = Exact
	return z
}

// SetPrec sets z's precision to prec and returns the (possibly) rounded
// value of z. Rounding occurs according to z's rounding mode if the mantissa
// cannot be represented in prec digits without loss of precision.
// SetPrec(0) maps all finite values to ±0; infinite values remain unchanged.
// If prec > MaxPrec, it is set to MaxPrec.
func (z *Decimal) SetPrec(prec uint) *Decimal {
	z.acc = Exact // optimistically assume no rounding is needed

	// special case
	if prec == 0 {
		z.prec = 0
		if z.form == finite {
			// truncate z to 0
			z.acc = makeAcc(z.neg)
			z.form = zero
		}
		return z
	}

	// general case
	if prec > MaxPrec {
		prec = MaxPrec
	}
	old := z.prec
	z.prec = uint32(prec)
	if z.prec < old {
		z.round(0)
	}
	return z
}

func (z *Decimal) SetRat(x *big.Rat) *Decimal {
	panic("not implemented")
}

// SetUint64 sets z to the (possibly rounded) value of x and returns z. If z's
// precision is 0, it is changed to DefaultDecimalPrec (and rounding will have
// no effect).
func (z *Decimal) SetUint64(x uint64) *Decimal {
	return z.setBits64(false, x)
}

// Sign returns:
//
//	-1 if x <   0
//	 0 if x is ±0
//	+1 if x >   0
//
func (x *Decimal) Sign() int {
	if debugDecimal {
		x.validate()
	}
	if x.form == zero {
		return 0
	}
	if x.neg {
		return -1
	}
	return 1
}

// Signbit reports whether x is negative or negative zero.
func (x *Decimal) Signbit() bool {
	return x.neg
}

func (z *Decimal) Sqrt(x *Decimal) *Decimal {
	panic("not implemented")
}

// Sub sets z to the rounded difference x-y and returns z.
// Precision, rounding, and accuracy reporting are as for Add.
// Sub panics with ErrNaN if x and y are infinities with equal
// signs. The value of z is undefined in that case.
func (z *Decimal) Sub(x, y *Decimal) *Decimal {
	if debugDecimal {
		x.validate()
		y.validate()
	}

	if z.prec == 0 {
		z.prec = umax32(x.prec, y.prec)
	}

	if x.form == finite && y.form == finite {
		// x - y (common case)
		yneg := y.neg
		z.neg = x.neg
		if x.neg != yneg {
			// x - (-y) == x + y
			// (-x) - y == -(x + y)
			z.uadd(x, y)
		} else {
			// x - y == x - y == -(y - x)
			// (-x) - (-y) == y - x == -(x - y)
			if x.ucmp(y) > 0 {
				z.usub(x, y)
			} else {
				z.neg = !z.neg
				z.usub(y, x)
			}
		}
		if z.form == zero && z.mode == ToNegativeInf && z.acc == Exact {
			z.neg = true
		}
		return z
	}

	if x.form == inf && y.form == inf && x.neg == y.neg {
		// +Inf - +Inf
		// -Inf - -Inf
		// value of z is undefined but make sure it's valid
		z.acc = Exact
		z.form = zero
		z.neg = false
		panic(ErrNaN{"subtraction of infinities with equal signs"})
	}

	if x.form == zero && y.form == zero {
		// ±0 - ±0
		z.acc = Exact
		z.form = zero
		z.neg = x.neg && !y.neg // -0 - +0 == -0
		return z
	}

	if x.form == inf || y.form == zero {
		// ±Inf - y
		// x - ±0
		return z.Set(x)
	}

	// ±0 - y
	// x - ±Inf
	return z.Neg(y)
}

func (x *Decimal) Uint64() (uint64, Accuracy) {
	panic("not implemented")
}

func (z *Decimal) UnmarshalText(text []byte) error {
	panic("not implemented")
}

func (x *Decimal) validate() {
	if !debugDecimal {
		// avoid performance bugs
		panic("validate called but debugDecimal is not set")
	}
	if x.form != finite {
		return
	}
	m := len(x.mant)
	if m == 0 {
		panic("nonzero finite number with empty mantissa")
	}
	if msw := x.mant[m-1]; !(_DB/10 <= msw && msw < _DB) {
		panic(fmt.Sprintf("last word of %s is not within [%d %d)", x.Text('p', 0), uint(_DB/10), uint(_DB)))
	}
	if x.prec == 0 {
		panic("zero precision finite number")
	}

}

func validateBinaryOperands(x, y *Decimal) {
	if !debugDecimal {
		// avoid performance bugs
		panic("validateBinaryOperands called but debugFloat is not set")
	}
	if len(x.mant) == 0 {
		panic("empty mantissa for x")
	}
	if len(y.mant) == 0 {
		panic("empty mantissa for y")
	}
}

// round rounds z according to z.mode to z.prec digits and sets z.acc accordingly.
// z's mantissa must be normalized or empty.
//
// CAUTION: The rounding modes ToNegativeInf, ToPositiveInf are affected by the
// sign of z. For correct rounding, the sign of z must be set correctly before
// calling round.
func (z *Decimal) round(sbit uint) {
	if debugDecimal {
		z.validate()
	}

	z.acc = Exact
	if z.form != finite {
		// ±0 or ±Inf => nothing left to do
		return
	}
	// z.form == finite && len(z.mant) > 0
	// m > 0 implies z.prec > 0 (checked by validate)
	m := uint32(len(z.mant)) // present mantissa length in words
	digits := m * _DW
	if digits <= z.prec {
		// mantissa fits => nothing to do
		return
	}

	// digits > z.prec: mantissa too large => round
	r := uint(digits - z.prec - 1) // rounding digit position r >= 0
	rdigit := z.mant.digit(r)      // rounding digit

	if sbit == 0 && (rdigit == 0 || z.mode == ToNearestEven) {
		// The sticky bit is only needed for rounding ToNearestEven
		// or when the rounding bit is zero. Avoid computation otherwise.
		sbit = z.mant.sticky(r)
	}
	sbit &= 1 // be safe and ensure it's a single bit	// cut off extra words

	n := (z.prec + (_DW - 1)) / _DW // mantissa length in words for desired precision
	if m > n {
		copy(z.mant, z.mant[m-n:]) // move n last words to front
		z.mant = z.mant[:n]
	}

	// determine number of trailing zero digits (ntz) and compute lsd of mantissa's least-significant word
	ntz := uint(n*_DW - z.prec) // 0 <= ntz < _W
	lsd := pow10(ntz)

	// round if result is inexact
	if rdigit|sbit != 0 {
		// Make rounding decision: The result mantissa is truncated ("rounded down")
		// by default. Decide if we need to increment, or "round up", the (unsigned)
		// mantissa.
		inc := false
		switch z.mode {
		case ToNegativeInf:
			inc = z.neg
		case ToZero:
			// nothing to do
		case ToNearestEven:
			inc = rdigit > 5 || (rdigit == 5 && (sbit != 0 || z.mant.digit(ntz)&1 != 0))
		case ToNearestAway:
			inc = rdigit >= 5
		case AwayFromZero:
			inc = true
		case ToPositiveInf:
			inc = !z.neg
		default:
			panic("unreachable")
		}
		z.acc = makeAcc(inc != z.neg)
		if inc {
			// add 1 to mantissa
			if add10VW(z.mant, z.mant, Word(lsd)) != 0 {
				// mantissa overflow => adjust exponent
				if z.exp >= MaxExp {
					// exponent overflow
					z.form = inf
					return
				}
				z.exp++
				// mantissa overflow means that the mantissa before increment
				// was all nines. In that case, the result is 1**(z.exp+1)
				z.mant[n-1] = _DB / 10
			}
		}
	}

	// zero out trailing digits in least-significant word
	z.mant[0] -= z.mant[0] % Word(lsd)

	if debugDecimal {
		z.validate()
	}
}

// dnorm normalizes mantissa m by shifting it to the left
// such that the msd of the most-significant word (msw) is != 0.
// It returns the shift amount. It assumes that len(m) != 0.
func dnorm(m dec) int64 {
	if debugDecimal && (len(m) == 0 || m[len(m)-1] == 0) {
		panic("msw of mantissa is 0")
	}
	s := _DW - decDigits(uint(m[len(m)-1]))
	// partial shift
	if s > 0 {
		c := shl10VU(m, m, s)
		if debugDecimal && c != 0 {
			panic("nlz or shlVU incorrect")
		}
	}
	return int64(s)
}

// z = x * y, ignoring signs of x and y for the multiplication
// but using the sign of z for rounding the result.
// x and y must have a non-empty mantissa and valid exponent.
func (z *Decimal) umul(x, y *Decimal) {
	if debugDecimal {
		validateBinaryOperands(x, y)
	}

	// Note: This is doing too much work if the precision
	// of z is less than the sum of the precisions of x
	// and y which is often the case (e.g., if all floats
	// have the same precision).
	// TODO(db47h) Optimize this for the common case.

	e := int64(x.exp) + int64(y.exp)
	if x == y {
		z.mant = z.mant.sqr(x.mant)
	} else {
		z.mant = z.mant.mul(x.mant, y.mant)
	}
	z.setExpAndRound(e-dnorm(z.mant), 0)
}
