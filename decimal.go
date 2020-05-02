package decimal

import (
	"fmt"
	"math"
	"math/big"
)

type Decimal struct {
	mant dec
	exp  int32
	prec uint32
	dig  uint32
	mode RoundingMode
	acc  Accuracy
	form form
	neg  bool
}

func NewDecimal(x float64) *Decimal {
	panic("not implemented")
}

func (z *Decimal) Abs(x *Decimal) *Decimal {
	panic("not implemented")
}

// Acc returns the accuracy of x produced by the most recent operation.
func (x *Decimal) Acc() Accuracy {
	return x.acc
}

func (z *Decimal) Add(x, y *Decimal) *Decimal {
	panic("not implemented")
}

func (x *Decimal) Append(buf []byte, fmt byte, prec int) []byte {
	panic("not implemented")
}

func (x *Decimal) Cmp(y *Decimal) int {
	panic("not implemented")
}

func (z *Decimal) Copy(x *Decimal) *Decimal {
	panic("not implemented")
}

func (x *Decimal) Float32() (float32, Accuracy) {
	panic("not implemented")
}

func (x *Decimal) Float64() (float64, Accuracy) {
	panic("not implemented")
}

func (x *Decimal) Format(s fmt.State, format rune) {
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
	panic("not implemented")
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
	return uint(x.dig)
}

// Mode returns the rounding mode of x.
func (x *Decimal) Mode() RoundingMode {
	return x.mode
}

func (z *Decimal) Mul(x, y *Decimal) *Decimal {
	panic("not implemented")
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

func (z *Decimal) Quo(x, y *Decimal) *Decimal {
	panic("not implemented")
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
		z.dig = x.dig
		if x.form == finite {
			z.exp = x.exp
			z.mant = z.mant.set(x.mant)
		}
		if z.prec == 0 {
			z.prec = x.prec
		} else if z.prec < x.prec {
			z.round()
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

const ln2_10 = math.Ln10 / math.Ln2

// SetInt sets z to the (possibly rounded) value of x and returns z.
// If z's precision is 0, it is changed to the larger of x.BitLen()
// or 64 (and rounding will have no effect).
func (z *Decimal) SetInt(x *big.Int) *Decimal {
	bits := uint32(x.BitLen())
	// estimate precision. May overshoot actual precision by 1.
	prec := uint32(float64(bits)/ln2_10) + 1
	if z.prec == 0 {
		z.prec = umax32(prec, _WD)
	}
	// TODO(db47h) truncating x could be more efficient if z.prec > 0
	// but small compared to the size of x, or if there
	// are many trailing 0's.
	z.acc = Exact
	z.neg = x.Sign() < 0
	if bits == 0 {
		z.form = zero
		return z
	}
	// x != 0
	exp := uint(0)
	z.mant = z.mant.make((int(prec) + _WD - 1) / _WD).setInt(x)
	z.mant, exp = z.mant.norm()
	z.dig = uint32(z.mant.digits())
	z.setExpAndRound(int64(exp) + int64(z.dig))
	return z
}

func (z *Decimal) SetInt64(x int64) *Decimal {
	panic("not implemented")
}

func (z *Decimal) setExpAndRound(exp int64) {
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
	z.round()
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

func (z *Decimal) SetPrec(prec uint) *Decimal {
	panic("not implemented")
}

func (z *Decimal) SetRat(x *big.Rat) *Decimal {
	panic("not implemented")
}

func (z *Decimal) SetUint64(x uint64) *Decimal {
	panic("not implemented")
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

func (x *Decimal) String() string {
	panic("not implemented")
}

func (z *Decimal) Sub(x, y *Decimal) *Decimal {
	panic("not implemented")
}

func (x *Decimal) Text(format byte, prec int) string {
	panic("not implemented")
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
	if x.mant[m-1] == 0 {
		panic(fmt.Sprintf("last word of %s is zero", x.Text('e', 0)))
	}
	if x.mant[0]%10 == 0 {
		panic(fmt.Sprintf("first word %d of %s is divisible by 10", x.mant[0], x.Text('e', 0)))
	}
	if d := uint32(x.mant.digits()); x.dig != d {
		panic(fmt.Sprintf("digit count %d != real digit count %d for %s", x.dig, d, x.Text('e', 0)))
	}
	if x.prec == 0 {
		panic("zero precision finite number")
	}

}

// round rounds z according to z.mode to z.prec digits and sets z.acc accordingly.
// z's mantissa must be normalized or empty.
//
// CAUTION: The rounding modes ToNegativeInf, ToPositiveInf are affected by the
// sign of z. For correct rounding, the sign of z must be set correctly before
// calling round.
func (z *Decimal) round() {
	var sbit bool
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
	// m := uint32(len(z.mant)) // present mantissa length in words
	if z.dig <= z.prec {
		// mantissa fits => nothing to do
		return
	}

	// digits > z.prec
	// r := uint(z.digits - z.prec - 1)
	// rd := z.mant.digit(r)

	var r Word
	z.mant, r, sbit = z.mant.shr10(uint(z.dig - z.prec))
	z.dig = z.prec

	if r != 0 || sbit {
		inc := false
		switch z.mode {
		case ToNegativeInf:
			inc = z.neg
		case ToZero:
			// nothing to do
		case ToNearestEven:
			inc = r > 5 || (r == 5 && (sbit || z.mant[0]&1 != 0))
		case ToNearestAway:
			inc = r >= 5
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
			if add10VW(z.mant, z.mant, 1) != 0 {

			}
		}
	}
	if debugDecimal {
		z.validate()
	}
}
