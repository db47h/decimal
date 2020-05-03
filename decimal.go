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
	return x.mant.digits()
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
	// but small compared to the size of x, or if there are many trailing 0's.
	z.acc = Exact
	z.neg = x.Sign() < 0
	if bits == 0 {
		z.form = zero
		return z
	}
	// x != 0
	exp := uint(0)
	z.mant, exp = z.mant.make((int(prec) + _WD - 1) / _WD).setInt(x)
	z.setExpAndRound(int64(exp), 0)
	return z
}

func (z *Decimal) SetInt64(x int64) *Decimal {
	panic("not implemented")
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
	if msw := x.mant[m-1]; !(_BD/10 <= msw && msw < _BD) {
		panic(fmt.Sprintf("last word of %s is not within [%d %d)", x.Text('e', 0), uint(_BD/10), uint(_BD)))
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
	digits := m * _WD
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

	n := (z.prec + (_WD - 1)) / _WD // mantissa length in words for desired precision
	if m > n {
		copy(z.mant, z.mant[m-n:]) // move n last words to front
		z.mant = z.mant[:n]
	}

	// determine number of trailing zero digits (ntz) and compute lsd of mantissa's least-significant word
	ntz := uint(n*_WD - z.prec) // 0 <= ntz < _W
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
				z.mant[n-1] = _BD / 10
			}
		}
	}

	// zero out trailing digits in least-significant word
	z.mant[0] -= z.mant[0] % Word(lsd)

	if debugDecimal {
		z.validate()
	}
}
