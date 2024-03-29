// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"fmt"
	"io"
	"strings"
)

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
	prec := z.prec
	if prec == 0 {
		prec = DefaultDecimalPrec
	}

	// A reasonable value in case of an error.
	z.form = zero

	// sign
	z.neg, err = scanSign(r)
	if err != nil {
		return
	}

	// mantissa
	var fcount int // fractional digit count; valid if <= 0
	z.mant, b, fcount, err = z.mant.scan(r, base, true)
	if err != nil {
		return
	}

	// exponent
	var exp int64
	var ebase int
	exp, ebase, err = scanExponent(r, true, base == 0)
	if err != nil {
		return
	}

	// special-case 0
	if len(z.mant) == 0 {
		z.prec = prec
		z.acc = Exact
		z.form = zero
		f = z
		return
	}
	// len(z.mant) > 0

	// The mantissa may have a radix point (fcount <= 0) and there
	// may be a nonzero exponent exp. The radix point amounts to a
	// division by b**(-fcount). An exponent means multiplication by
	// ebase**exp. Finally, mantissa normalization (shift left) requires
	// a correcting multiplication by 2**(-shiftcount). Multiplications
	// are commutative, so we can apply them in any order as long as there
	// is no loss of precision. We only have powers of 2 and 10, and
	// we split powers of 10 into the product of the same powers of
	// 2 and 5. This reduces the size of the multiplication factor
	// needed for base-10 exponents.

	// normalize mantissa and determine initial exponent contributions
	exp2 := int64(0)
	exp10 := int64(len(z.mant))*_DW - dnorm(z.mant)

	// determine binary or decimal exponent contribution of radix point
	if fcount < 0 {
		// The mantissa has a radix point ddd.dddd; and
		// -fcount is the number of digits to the right
		// of '.'. Adjust relevant exponent accordingly.
		d := int64(fcount)
		switch b {
		case 10:
			exp10 += d
		case 2:
			exp2 += d
		case 8:
			exp2 += d * 3 // octal digits are 3 bits each
		case 16:
			exp2 += d * 4 // hexadecimal digits are 4 bits each
		default:
			panic("unexpected mantissa base")
		}
		// fcount consumed - not needed anymore
	}

	// take actual exponent into account
	switch ebase {
	case 10:
		exp10 += exp
	case 2:
		exp2 += exp
	default:
		panic("unexpected exponent base")
	}
	// exp consumed - not needed anymore

	// apply 10**exp10
	if MinExp <= exp10 && exp10 <= MaxExp {
		z.prec = prec
		z.form = finite
		z.exp = int32(exp10)
		f = z
	} else {
		err = fmt.Errorf("exponent overflow")
		return
	}

	if exp2 == 0 {
		// no binary exponent contribution
		z.round(0)
		return
	}
	// exp2 != 0

	// // apply 2**exp2
	p := new(Decimal).SetPrec(z.Prec() + _DW) // use more bits for p -- TODO(db47h) what is the right number?
	if exp2 < 0 {
		z.Quo(z, p.pow2(uint64(-exp2)))
	} else {
		z.Mul(z, p.pow2(uint64(exp2)))
	}

	return
}

// pow2 sets z to 2**n and returns z.
// n must not be negative.
func (z *Decimal) pow2(n uint64) *Decimal {
	const m = _DWb - 1 // maximum exponent such that 2**m < _BD
	if n < _W {
		return z.SetUint64(1 << n)
	}
	// n > m

	z.SetUint64(1 << m)
	n -= m

	// use more bits for f than for z
	// TODO(db47h) what is the right number?
	f := new(Decimal).SetPrec(z.Prec() + _DW).SetUint64(2)

	for n > 0 {
		if n&1 != 0 {
			z.Mul(z, f)
			if n == 1 {
				break
			}
		}
		f.Mul(f, f)
		n >>= 1
	}

	return z
}

// Parse parses s which must contain a text representation of a floating-point
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
// precision is 0, it is changed to DefaultDecimalPrec before rounding takes
// effect. The number must be of the form:
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
// The returned *Decimal d is nil and the value of z is valid but not defined if
// an error is reported.
//
func (z *Decimal) Parse(s string, base int) (d *Decimal, b int, err error) {
	// scan doesn't handle ±Inf
	if len(s) == 3 && (s == "Inf" || s == "inf") {
		d = z.SetInf(false)
		return
	}
	if len(s) == 4 && (s[0] == '+' || s[0] == '-') && (s[1:] == "Inf" || s[1:] == "inf") {
		d = z.SetInf(s[0] == '-')
		return
	}

	r := strings.NewReader(s)
	if d, b, err = z.scan(r, base); err != nil {
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

// ParseDecimal is like d.Parse(s, base) with d set to the given precision
// and rounding mode.
func ParseDecimal(s string, base int, prec uint, mode RoundingMode) (d *Decimal, b int, err error) {
	return new(Decimal).SetPrec(prec).SetMode(mode).Parse(s, base)
}

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
