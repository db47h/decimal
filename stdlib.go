// This file mirrors types and constants from math/big.

package decimal

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
)

// MaxBase is the largest number base accepted for string conversions.
const MaxBase = 10 + ('z' - 'a' + 1) + ('Z' - 'A' + 1)

// Exponent and precision limits.
const (
	MaxExp  = math.MaxInt32  // largest supported exponent
	MinExp  = math.MinInt32  // smallest supported exponent
	MaxPrec = math.MaxUint32 // largest (theoretically) supported precision; likely memory-limited
)

// Internal representation: The mantissa bits x.mant of a nonzero finite
// Decimal x are stored in a dec slice long enough to hold up to x.prec digits;
//
// A zero or non-finite Decimal x ignores x.mant and x.exp.
//
// x                 form      neg      mant         exp
// ----------------------------------------------------------
// ±0                zero      sign     -            -
// 0 < |x| < +Inf    finite    sign     mantissa     exponent
// ±Inf              inf       sign     -            -

// A form value describes the internal representation.
type form byte

// The form value order is relevant - do not change!
const (
	zero form = iota
	finite
	inf
)

// RoundingMode determines how a Float value is rounded to the
// desired precision. Rounding may change the Float value; the
// rounding error is described by the Float's Accuracy.
type RoundingMode byte

// These constants define supported rounding modes.
const (
	ToNearestEven RoundingMode = iota // == IEEE 754-2008 roundTiesToEven
	ToNearestAway                     // == IEEE 754-2008 roundTiesToAway
	ToZero                            // == IEEE 754-2008 roundTowardZero
	AwayFromZero                      // no IEEE 754-2008 equivalent
	ToNegativeInf                     // == IEEE 754-2008 roundTowardNegative
	ToPositiveInf                     // == IEEE 754-2008 roundTowardPositive
)

//go:generate stringer -type=RoundingMode

// Accuracy describes the rounding error produced by the most recent
// operation that generated a Float value, relative to the exact value.
type Accuracy int8

// Constants describing the Accuracy of a Float.
const (
	Below Accuracy = -1
	Exact Accuracy = 0
	Above Accuracy = +1
)

//go:generate stringer -type=Accuracy

func makeAcc(above bool) Accuracy {
	if above {
		return Above
	}
	return Below
}

// A Word represents a single digit of a multi-precision unsigned integer.
type Word uint

const (
	// _S = _W / 8 // word size in bytes

	_W = bits.UintSize // word size in bits
	// _B = 1 << _W       // digit base
	// _M = _B - 1        // digit mask
)

// byteReader is a local wrapper around fmt.ScanState;
// it implements the ByteReader interface.
type byteReader struct {
	fmt.ScanState
}

func (r byteReader) ReadByte() (byte, error) {
	ch, size, err := r.ReadRune()
	if size != 1 && err == nil {
		err = fmt.Errorf("invalid rune %#U", ch)
	}
	return byte(ch), err
}

func (r byteReader) UnreadByte() error {
	return r.UnreadRune()
}

func umax32(x, y uint32) uint32 {
	if x > y {
		return x
	}
	return y
}

// q = (u1<<_W + u0 - r)/v
func divWW_g(u1, u0, v big.Word) (q, r big.Word) {
	qq, rr := bits.Div(uint(u1), uint(u0), uint(v))
	return big.Word(qq), big.Word(rr)
}

func divWVW_g(z []big.Word, xn big.Word, x []big.Word, y big.Word) (r big.Word) {
	r = xn
	for i := len(z) - 1; i >= 0; i-- {
		z[i], r = divWW_g(r, x[i], y)
	}
	return r
}
