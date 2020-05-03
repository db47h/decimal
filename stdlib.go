// This file mirrors types and constants from math/big.

package decimal

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/bits"
	"strconv"
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
func divWW(u1, u0, v big.Word) (q, r big.Word) {
	qq, rr := bits.Div(uint(u1), uint(u0), uint(v))
	return big.Word(qq), big.Word(rr)
}

func divWVW(z []big.Word, xn big.Word, x []big.Word, y big.Word) (r big.Word) {
	r = xn
	for i := len(z) - 1; i >= 0; i-- {
		z[i], r = divWW(r, x[i], y)
	}
	return r
}

func same(x, y []Word) bool {
	return len(x) == len(y) && len(x) > 0 && &x[0] == &y[0]
}

// scan errors
var (
	errNoDigits = errors.New("number has no digits")
	errInvalSep = errors.New("'_' must separate successive digits")
)

func scanSign(r io.ByteScanner) (neg bool, err error) {
	var ch byte
	if ch, err = r.ReadByte(); err != nil {
		return false, err
	}
	switch ch {
	case '-':
		neg = true
	case '+':
		// nothing to do
	default:
		r.UnreadByte()
	}
	return
}

func scanExponent(r io.ByteScanner, base2ok, sepOk bool) (exp int64, base int, err error) {
	// one char look-ahead
	ch, err := r.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return 0, 10, err
	}

	// exponent char
	switch ch {
	case 'e', 'E':
		base = 10
	case 'p', 'P':
		if base2ok {
			base = 2
			break // ok
		}
		fallthrough // binary exponent not permitted
	default:
		r.UnreadByte() // ch does not belong to exponent anymore
		return 0, 10, nil
	}

	// sign
	var digits []byte
	ch, err = r.ReadByte()
	if err == nil && (ch == '+' || ch == '-') {
		if ch == '-' {
			digits = append(digits, '-')
		}
		ch, err = r.ReadByte()
	}

	// prev encodes the previously seen char: it is one
	// of '_', '0' (a digit), or '.' (anything else). A
	// valid separator '_' may only occur after a digit.
	prev := '.'
	invalSep := false

	// exponent value
	hasDigits := false
	for err == nil {
		if '0' <= ch && ch <= '9' {
			digits = append(digits, ch)
			prev = '0'
			hasDigits = true
		} else if ch == '_' && sepOk {
			if prev != '0' {
				invalSep = true
			}
			prev = '_'
		} else {
			r.UnreadByte() // ch does not belong to number anymore
			break
		}
		ch, err = r.ReadByte()
	}

	if err == io.EOF {
		err = nil
	}
	if err == nil && !hasDigits {
		err = errNoDigits
	}
	if err == nil {
		exp, err = strconv.ParseInt(string(digits), 10, 64)
	}
	// other errors take precedence over invalid separators
	if err == nil && (invalSep || prev == '_') {
		err = errInvalSep
	}

	return
}
