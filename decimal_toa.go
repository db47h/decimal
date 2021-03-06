// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements Decimal-to-string conversion functions. It is closely
// following the corresponding implementation in math/big/ftoa.go.

package decimal

import (
	"bytes"
	"fmt"
	"strconv"
)

// Text converts the decimal floating-point number x to a string according to
// the given format and precision prec. The format is one of:
//
//  'e' -d.dddde±dd, decimal exponent, at least two (possibly 0) exponent digits
//  'E' -d.ddddE±dd, decimal exponent, at least two (possibly 0) exponent digits
//  'f' -ddddd.dddd, no exponent
//  'g' like 'e' for large exponents, like 'f' otherwise
//  'G' like 'E' for large exponents, like 'f' otherwise
//  'p' -0.dddde±dd, decimal mantissa, decimal exponent (non-standard)
//  'b' -dddddde±dd, decimal mantissa, decimal exponent (non-standard)
//
// For non-standard formats, the mantissa is printed in normalized form:
//
//  'p' decimal mantissa in [0.1, 1), or 0
//  'b' decimal integer mantissa using x.Prec() digits, or 0
//
// Note that the 'b' and 'p' formats differ from big.Float: an hexadecimal
// representation does not make sense for decimals. These formats use a full
// decimal representation instead.
//
// If format is a different character, Text returns a "%" followed by the
// unrecognized format character.
//
// The precision prec controls the number of digits (excluding the exponent)
// printed by the 'e', 'E', 'f', 'g', and 'G' formats. For 'e', 'E', and 'f', it
// is the number of digits after the decimal point. For 'g' and 'G' it is the
// total number of digits. A negative precision selects the smallest number of
// decimal digits necessary to identify the value x uniquely using x.Prec()
// mantissa digits. The prec value is ignored for the 'b' and 'p' formats.
func (x *Decimal) Text(format byte, prec int) string {
	return string(x.Append(nil, format, prec))
}

// String formats x like x.Text('g', 10).
// (String must be called explicitly, Decimal.Format does not support %s verb.)
func (x *Decimal) String() string {
	return x.Text('g', 10)
}

// Append appends to buf the string form of the floating-point number x,
// as generated by x.Text, and returns the extended buffer.
func (x *Decimal) Append(buf []byte, fmt byte, prec int) []byte {
	if cap(buf) == 0 {
		buf = make([]byte, 0, x.bufSizeForFmt(fmt, prec))
	}

	// sign
	if x.neg {
		buf = append(buf, '-')
	}

	// Inf
	if x.form == inf {
		if !x.neg {
			buf = append(buf, '+')
		}
		return append(buf, "Inf"...)
	}

	// pick off easy formats
	switch fmt {
	case 'b':
		return x.fmtB(buf)
	case 'p':
		return x.fmtP(buf)
	}

	// Algorithm:
	//   1) round to desired precision
	//   2) read digits out and format

	// 1) round to desired precision
	shortest := false
	digits := int(x.MinPrec())
	if prec < 0 {
		shortest = true
		// no rounding necessary
		switch fmt {
		case 'e', 'E':
			prec = digits - 1
		case 'f':
			prec = max(digits-int(x.exp), 0)
		case 'g', 'G':
			prec = digits
		}
	} else {
		// round appropriately
		rnd := 0
		switch fmt {
		case 'e', 'E':
			// one digit before and number of digits after decimal point
			rnd = 1 + prec
		case 'f':
			// number of digits before and after decimal point
			rnd = max(int(x.exp)+prec, 0)
		case 'g', 'G':
			if prec == 0 {
				prec = 1
			}
			rnd = prec
		}
		if rnd < digits {
			x = new(Decimal).SetMode(x.mode).SetPrec(uint(rnd)).Set(x)
			digits = int(x.MinPrec())
		}
	}

	// 2) read digits out and format
	switch fmt {
	case 'e', 'E':
		return x.fmtE(buf, fmt, prec)
	case 'f':
		return x.fmtF(buf, prec)
	case 'g', 'G':
		// trim trailing fractional zeros in %e format
		eprec := prec
		if eprec > digits && digits >= int(x.exp) {
			eprec = digits
		}
		// %e is used if the exponent is less than -4 or greater than or
		// equal to the precision. If precision was the shortest possible,
		// use eprec = 6 for this decision.
		if shortest {
			eprec = 6
		}
		exp := int(x.exp) - 1
		if exp < -4 || exp >= eprec {
			if prec > digits {
				prec = digits
			}
			return x.fmtE(buf, fmt+'e'-'g', prec-1)
		}
		if prec > int(x.exp) {
			prec = digits
		}
		return x.fmtF(buf, max(prec-int(x.exp), 0))
	}

	// unknown format
	if x.neg {
		buf = buf[:len(buf)-1] // sign was added prematurely - remove it again
	}
	return append(buf, '%', fmt)
}

// digitsForFmt returns the estimated buffer size required to represent x in
// format fmt with precision prec.
func (x *Decimal) bufSizeForFmt(fmt byte, prec int) int {
	digits := int(x.MinPrec())
	exp := x.MantExp(nil)
	if digits == 0 {
		digits = 1
	}
	var sz int
	if x.neg {
		sz++
	}
	switch fmt {
	case 'e', 'E':
		sz += 2 + expSz(exp)
		if prec < 0 {
			sz += digits
		} else {
			sz += prec + 1
		}
	case 'f':
		sz++
		if prec < 0 {
			sz += digits
			if exp < 0 || exp > digits {
				sz += abs(exp)
			}
		} else {
			sz += max(int(exp), 1) + prec
		}
	case 'g':
		sz += 2 + expSz(exp)
		if prec < 0 {
			sz += digits
		} else {
			sz += prec
		}
	case 'b':
		// -ddddde±dd
		sz += len(x.mant)*_DW + 1 + expSz(exp)
	case 'p':
		// -0.ddde±dd
		sz += 2 + digits + 1 + expSz(exp)
	default:
		sz = prec
	}

	return max(4, sz)
}

func expSz(exp int) int {
	var n int
	if exp < 0 {
		n = int(decDigits(uint(-exp))) + 1
	} else {
		n = int(decDigits(uint(exp)))
	}
	return min(2, n)
}

// %f: ddddddd.ddddd
// prec is # of digits after decimal point
func (x *Decimal) fmtF(buf []byte, prec int) []byte {
	mant, exp := x.toa(10)
	// integer, padded with zeros as needed
	if exp > 0 {
		m := min(int(x.MinPrec()), exp)
		buf = append(buf, mant[:m]...)
		for ; m < exp; m++ {
			buf = append(buf, '0')
		}
	} else {
		buf = append(buf, '0')
	}

	// fraction
	if prec > 0 {
		buf = append(buf, '.')
		for i := 0; i < prec; i++ {
			n := exp + i
			var ch byte = '0'
			if 0 <= n && n < len(mant) {
				ch = mant[n]
			}
			buf = append(buf, ch)
		}
	}

	return buf
}

// %e: d.ddddde±dd
// prec is # of digits after decimal point
func (x *Decimal) fmtE(buf []byte, fmt byte, prec int) []byte {
	mant, ex := x.toa(10)
	// trim trailing zeros
	n := len(mant)
	for n > 0 && mant[n-1] == '0' {
		n--
	}
	mant = mant[:n]

	// first digit
	ch := byte('0')
	if len(mant) > 0 {
		ch = mant[0]
	}
	buf = append(buf, ch)

	// .moredigits
	if prec > 0 {
		buf = append(buf, '.')
		i := 1
		m := min(len(mant), prec+1)
		if i < m {
			buf = append(buf, mant[i:m]...)
			i = m
		}
		for ; i <= prec; i++ {
			buf = append(buf, '0')
		}
	}

	// e±
	buf = append(buf, fmt)
	var exp int64
	if len(mant) > 0 {
		exp = int64(ex) - 1 // -1 because first digit was printed before '.'
	}
	if exp < 0 {
		ch = '-'
		exp = -exp
	} else {
		ch = '+'
	}
	buf = append(buf, ch)

	// dd...d
	if exp < 10 {
		buf = append(buf, '0') // at least 2 exponent digits
	}
	return strconv.AppendInt(buf, exp, 10)
}

// -dddddde±dd
func (x *Decimal) fmtB(buf []byte) []byte {
	if x.form == zero {
		return append(buf, '0')
	}

	if debugDecimal && x.form != finite {
		panic("non-finite decimal")
	}
	// x != 0

	// adjust mantissa to use exactly x.prec bits
	m, exp := x.toa(10)
	if int(x.prec) < len(m) {
		m = m[:x.prec]
	}

	buf = append(buf, m...)
	for i := len(m); i < int(x.prec); i++ {
		buf = append(buf, '0')
	}
	buf = append(buf, 'e')
	e := int64(exp) - int64(x.prec)
	if e >= 0 {
		buf = append(buf, '+')
	}
	return strconv.AppendInt(buf, e, 10)
}

// -0.dddde±dd
func (x *Decimal) fmtP(buf []byte) []byte {
	if x.form == zero {
		return append(buf, '0')
	}

	if debugDecimal && x.form != finite {
		panic("non-finite decimal")
	}
	// x != 0

	buf = append(buf, "0."...)
	mant, exp := x.toa(10)
	buf = append(buf, bytes.TrimRight(mant, "0")...)
	buf = append(buf, 'e')
	if exp >= 0 {
		buf = append(buf, '+')
	}
	return strconv.AppendInt(buf, int64(exp), 10)
}

// Format implements fmt.Formatter. It accepts the regular formats for
// floating-point numbers 'e', 'E', 'f', 'F', 'g', and 'G', as well as 'b', 'p'
// and 'v'. See (*Decimal).Text for the interpretation of 'b' and 'p'. The 'v'
// format is handled like 'g'. Format also supports specification of the minimum
// precision in digits, the output field width, as well as the format flags '+'
// and ' ' for sign control, '0' for space or zero padding, and '-' for left or
// right justification. See the fmt package for details.
func (x *Decimal) Format(s fmt.State, format rune) {
	prec, hasPrec := s.Precision()
	if !hasPrec {
		prec = 6 // default precision for 'e', 'f'
	}

	switch format {
	case 'e', 'E', 'f', 'b', 'p':
		// nothing to do
	case 'F':
		// (*Decimal).Text doesn't support 'F'; handle like 'f'
		format = 'f'
	case 's':
		format = 'g'
		if !hasPrec {
			prec = 10
		}
	case 'v':
		// handle like 'g'
		format = 'g'
		fallthrough
	case 'g', 'G':
		if !hasPrec {
			prec = -1 // default precision for 'g', 'G'
		}
	default:
		fmt.Fprintf(s, "%%!%c(*decimal.Decimal=%s)", format, x.String())
		return
	}
	var buf []byte
	buf = x.Append(buf, byte(format), prec)
	if len(buf) == 0 {
		buf = []byte("?") // should never happen, but don't crash
	}
	// len(buf) > 0

	var sign string
	switch {
	case buf[0] == '-':
		sign = "-"
		buf = buf[1:]
	case buf[0] == '+':
		// +Inf
		sign = "+"
		if s.Flag(' ') {
			sign = " "
		}
		buf = buf[1:]
	case s.Flag('+'):
		sign = "+"
	case s.Flag(' '):
		sign = " "
	}

	var padding int
	if width, hasWidth := s.Width(); hasWidth && width > len(sign)+len(buf) {
		padding = width - len(sign) - len(buf)
	}

	switch {
	case s.Flag('0') && !x.IsInf():
		// 0-padding on left
		writeMultiple(s, sign, 1)
		writeMultiple(s, "0", padding)
		s.Write(buf)
	case s.Flag('-'):
		// padding on right
		writeMultiple(s, sign, 1)
		s.Write(buf)
		writeMultiple(s, " ", padding)
	default:
		// padding on left
		writeMultiple(s, " ", padding)
		writeMultiple(s, sign, 1)
		s.Write(buf)
	}
}

// toa returns x.mant.utoa(base) and x.exp with least significant zero Words removed
// this function returns nil, 0 for non-finite numbers.
func (x *Decimal) toa(base int) ([]byte, int) {
	if x.form == finite {
		m := x.mant
		i := 0
		for i < len(m) && m[i] == 0 {
			i++
		}
		return m[i:].utoa(base), int(x.exp)
	}
	return nil, 0
}
