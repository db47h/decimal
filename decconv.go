package decimal

import (
	"fmt"
	"io"
	"strings"
)

var decimalZero Decimal

// SetString sets z to the value of s and returns z and a boolean indicating
// success. s must be a floating-point number of the same format as accepted
// by Parse, with base argument 0. The entire string (not just a prefix) must
// be valid for success. If the operation failed, the value of z is undefined
// but the returned value is nil.
func (z *Decimal) SetString(s string) (*Decimal, bool) {
	if f, _, err := z.Parse(s, 0); err == nil {
		return f, true
	}
	return nil, false
}

// scan is like Parse but reads the longest possible prefix representing a valid
// floating point number from an io.ByteScanner rather than a string. It serves
// as the implementation of Parse. It does not recognize ±Inf and does not expect
// EOF at the end.
func (z *Decimal) scan(r io.ByteScanner, base int) (f *Decimal, b int, err error) {
	panic("not implemented")
}

// Parse parses s which must contain a text representation of a floating- point
// number with a mantissa in the given conversion base (the exponent is always a
// decimal number), or a string representing an infinite value.
//
// For base 0, an underscore character ``_'' may appear between a base prefix
// and an adjacent digit, and between successive digits; such underscores do not
// change the value of the number, or the returned digit count. Incorrect
// placement of underscores is reported as an error if there are no other
// errors. If base != 0, underscores are not recognized and thus terminate
// scanning like any other character that is not a valid radix point or digit.
//
// It sets z to the (possibly rounded) value of the corresponding floating-
// point value, and returns z, the actual base b, and an error err, if any. The
// entire string (not just a prefix) must be consumed for success. If z's
// precision is 0, it is changed to fit all digits of the mantissa before
// rounding takes effect. The number must be of the form:
//
//     number    = [ sign ] ( float | "inf" | "Inf" ) .
//     sign      = "+" | "-" .
//     float     = ( mantissa | prefix pmantissa ) [ exponent ] .
//     prefix    = "0" [ "b" | "B" | "o" | "O" | "x" | "X" ] .
//     mantissa  = digits "." [ digits ] | digits | "." digits .
//     pmantissa = [ "_" ] digits "." [ digits ] | [ "_" ] digits | "." digits .
//     exponent  = ( "e" | "E" | "p" | "P" ) [ sign ] digits .
//     digits    = digit { [ "_" ] digit } .
//     digit     = "0" ... "9" | "a" ... "z" | "A" ... "Z" .
//
// The base argument must be 0, 2, 8, 10, or 16. Providing an invalid base
// argument will lead to a run-time panic.
//
// For base 0, the number prefix determines the actual base: A prefix of ``0b''
// or ``0B'' selects base 2, ``0o'' or ``0O'' selects base 8, and ``0x'' or
// ``0X'' selects base 16. Otherwise, the actual base is 10 and no prefix is
// accepted. The octal prefix "0" is not supported (a leading "0" is simply
// considered a "0").
//
// A "p" or "P" exponent indicates a base 2 (rather then base 10) exponent; for
// instance, "0x1.fffffffffffffp1023" (using base 0) represents the maximum
// float64 value. For hexadecimal mantissae, the exponent character must be one
// of 'p' or 'P', if present (an "e" or "E" exponent indicator cannot be
// distinguished from a mantissa digit).
//
// Note that rounding only happens if z's precision is not zero and less than
// the number of digits in the mantissa or with a base 2 exponent, in which case
// it is best to use ParseFloat then z.SetFloat.
//
// The returned *Decimal f is nil and the value of z is valid but not defined if
// an error is reported.
//
func (z *Decimal) Parse(s string, base int) (f *Decimal, b int, err error) {
	// scan doesn't handle ±Inf
	if len(s) == 3 && (s == "Inf" || s == "inf") {
		f = z.SetInf(false)
		return
	}
	if len(s) == 4 && (s[0] == '+' || s[0] == '-') && (s[1:] == "Inf" || s[1:] == "inf") {
		f = z.SetInf(s[0] == '-')
		return
	}

	r := strings.NewReader(s)
	if f, b, err = z.scan(r, base); err != nil {
		return
	}

	// entire string must have been consumed
	if ch, err2 := r.ReadByte(); err2 == nil {
		err = fmt.Errorf("expected end of string, found %q", ch)
	} else if err2 != io.EOF {
		err = err2
	}

	return
}

// ParseDecimal is like f.Parse(s, base) with f set to the given precision
// and rounding mode.
func ParseDecimal(s string, base int, prec uint, mode RoundingMode) (f *Decimal, b int, err error) {
	return new(Decimal).SetPrec(prec).SetMode(mode).Parse(s, base)
}

var _ fmt.Scanner = &decimalZero // *Decimal must implement fmt.Scanner

// Scan is a support routine for fmt.Scanner; it sets z to the value of
// the scanned number. It accepts formats whose verbs are supported by
// fmt.Scan for floating point values, which are:
// 'b' (binary), 'e', 'E', 'f', 'F', 'g' and 'G'.
// Scan doesn't handle ±Inf.
func (z *Decimal) Scan(s fmt.ScanState, ch rune) error {
	s.SkipSpace()
	_, _, err := z.scan(byteReader{s}, 0)
	return err
}
