// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"math/bits"
)

const (
	// _W * log10(2) = decimal digits per word. 9 decimal digits per 32 bits
	// word and 19 per 64 bits word.
	_DW = _W * 30103 / 100000
	// Decimal base for a word. 1e9 for 32 bits words and 1e19 for 64 bits
	// words.
	// We want this value to be a const. This is a dirty hack to avoid
	// conditional compilation; it will break if bits.UintSize != 32 or 64
	_DB = 9999999998000000000*(_DW/19) + 1000000000*(_DW/9)
	// Maximum value of a decimal Word
	_DMax = _DB - 1
	// Bits per decimal Word: Log2(_DB)+1 = _DW * Log2(10) + 1
	_DWb = _DW*100000/30103 + 1
)

var pow10tab = [...]uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000,
	10000000000, 100000000000, 1000000000000, 10000000000000, 100000000000000, 1000000000000000,
	10000000000000000, 100000000000000000, 1000000000000000000, 10000000000000000000,
}

func pow10(n uint) Word {
	if debugDecimal && _W == 32 && n > 9 {
		panic("pow10: overflow")
	}
	return Word(pow10tab[n])
}

var pow2digitsTab = [...]uint{
	1, 1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 4, 5, 5,
	5, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 9, 9, 9, 10, 10,
	10, 10, 11, 11, 11, 12, 12, 12, 13, 13, 13, 13, 14, 14, 14, 15,
	15, 15, 16, 16, 16, 16, 17, 17, 17, 18, 18, 18, 19, 19, 19, 20, 20,
}

// decDigits returns n such that of x such that 10**(n-1) <= x < 10**n.
// In other words, n the number of digits required to represent n.
// Returns 0 for x == 0.
func decDigits(x uint) (n uint) {
	if bits.UintSize == 32 {
		return decDigits32(uint32(x))
	}
	return decDigits64(uint64(x))
}

func decDigits64(x uint64) (n uint) {
	n = pow2digitsTab[bits.Len64(x)]
	if x < pow10tab[n-1] {
		n--
	}
	return n
}

func decDigits32(x uint32) (n uint) {
	n = pow2digitsTab[bits.Len32(x)]
	if x < uint32(pow10tab[n-1]) {
		n--
	}
	return n
}

func nlz10(x Word) uint {
	return _DW - decDigits(uint(x))
}

func trailingZeroDigits(n uint) uint {
	var d uint
	if bits.UintSize > 32 {
		if uint64(n)%10000000000000000 == 0 {
			n = uint(uint64(n) / uint64(10000000000000000))
			d += 16
		}
	}
	if n%100000000 == 0 {
		n /= 100000000
		d += 8
	}
	if n%10000 == 0 {
		n /= 10000
		d += 4
	}
	if n%100 == 0 {
		n /= 100
		d += 2
	}
	if n%10 == 0 {
		d += 1
	}
	return d
}

var decMaxPow32 = [...]uint32{
	0, 0, 0, 0, 536870912, 29, 387420489, 18, 268435456, 14, 244140625, 12, 362797056, 11,
	282475249, 10, 134217728, 9, 387420489, 9, 1000000000, 9, 214358881, 8, 429981696, 8, 815730721, 8,
	105413504, 7, 170859375, 7, 268435456, 7, 410338673, 7, 612220032, 7, 893871739, 7, 64000000, 6,
	85766121, 6, 113379904, 6, 148035889, 6, 191102976, 6, 244140625, 6, 308915776, 6, 387420489, 6,
	481890304, 6, 594823321, 6, 729000000, 6, 887503681, 6, 33554432, 5, 39135393, 5, 45435424, 5,
	52521875, 5, 60466176, 5, 69343957, 5, 79235168, 5, 90224199, 5, 102400000, 5, 115856201, 5,
	130691232, 5, 147008443, 5, 164916224, 5, 184528125, 5, 205962976, 5, 229345007, 5, 254803968, 5,
	282475249, 5, 312500000, 5, 345025251, 5, 380204032, 5, 418195493, 5, 459165024, 5, 503284375, 5,
	550731776, 5, 601692057, 5, 656356768, 5, 714924299, 5, 777600000, 5, 844596301, 5, 916132832, 5,
}

var decMaxPow64 = [...]uint64{
	0, 0, 0, 0, 9223372036854775808, 63, 4052555153018976267, 39, 4611686018427387904, 31, 7450580596923828125, 27, 4738381338321616896, 24,
	3909821048582988049, 22, 9223372036854775808, 21, 1350851717672992089, 19, 10000000000000000000, 19, 5559917313492231481, 18, 2218611106740436992, 17, 8650415919381337933, 17,
	2177953337809371136, 16, 6568408355712890625, 16, 1152921504606846976, 15, 2862423051509815793, 15, 6746640616477458432, 15, 799006685782884121, 14, 1638400000000000000, 14,
	3243919932521508681, 14, 6221821273427820544, 14, 504036361936467383, 13, 876488338465357824, 13, 1490116119384765625, 13, 2481152873203736576, 13, 4052555153018976267, 13,
	6502111422497947648, 13, 353814783205469041, 12, 531441000000000000, 12, 787662783788549761, 12, 1152921504606846976, 12, 1667889514952984961, 12, 2386420683693101056, 12,
	3379220508056640625, 12, 4738381338321616896, 12, 6582952005840035281, 12, 9065737908494995456, 12, 317475837322472439, 11, 419430400000000000, 11, 550329031716248441, 11,
	717368321110468608, 11, 929293739471222707, 11, 1196683881290399744, 11, 1532278301220703125, 11, 1951354384207722496, 11, 2472159215084012303, 11, 3116402981210161152, 11,
	3909821048582988049, 11, 4882812500000000000, 11, 6071163615208263051, 11, 7516865509350965248, 11, 9269035929372191597, 11, 210832519264920576, 10, 253295162119140625, 10,
	303305489096114176, 10, 362033331456891249, 10, 430804206899405824, 10, 511116753300641401, 10, 604661760000000000, 10, 713342911662882601, 10, 839299365868340224, 10,
}

// decMaxPow returns (b**n, n) with n the largest power of b such that (b**n) <= _BD.
// For instance decMaxPow(10) == (1e19 - 1, 19) for 19 decimal digits in a 64bit Word.
// In other words, at most n digits in base b fit into a decimal Word.
func decMaxPow(b Word) (p Word, n int) {
	i := b * 2
	if bits.UintSize == 32 {
		return Word(decMaxPow32[i]), int(decMaxPow32[i+1])
	}
	return Word(decMaxPow64[i]), int(decMaxPow64[i+1])
}

// pow10DivTab64 contains the "magic" numbers for fast division by 10**n
// where 1 <= n < 19, x / 10**n = ((x >> pre) * m) >> (_W + post).
// See https://gmplib.org/~tege/divcnst-pldi94.pdf
// generated using Go's src/cmd/compile/internal/ssa/magic.go and rewritegeneric.go rules
var pow10DivTab64 = [...]magic{
	{10, 0xcccccccccccccccd, 0, 3},
	{100, 0xa3d70a3d70a3d70b, 1, 5},
	{1000, 0x83126e978d4fdf3c, 1, 8},
	{10000, 0xd1b71758e219652c, 0, 13},
	{100000, 0xa7c5ac471b478424, 1, 15},
	{1000000, 0x8637bd05af6c69b6, 0, 19},
	{10000000, 0xd6bf94d5e57a42bd, 1, 22},
	{100000000, 0xabcc77118461cefd, 0, 26},
	{1000000000, 0x89705f4136b4a598, 1, 28},
	{10000000000, 0xdbe6fecebdedd5bf, 0, 33},
	{100000000000, 0xafebff0bcb24aaff, 0, 36},
	{1000000000000, 0x8cbccc096f5088cc, 0, 39},
	{10000000000000, 0xe12e13424bb40e14, 1, 42},
	{100000000000000, 0xb424dc35095cd810, 1, 45},
	{1000000000000000, 0x901d7cf73ab0acda, 1, 48},
	{10000000000000000, 0xe69594bec44de15c, 1, 52},
	{100000000000000000, 0xb877aa3236a4b44a, 1, 55},
	{1000000000000000000, 0x9392ee8e921d5d08, 1, 58},
	{10000000000000000000, 0xec1e4a7db69561a6, 1, 62},
}

var pow10DivTab32 = [...]magic{
	{10, 0xcccccccd, 0, 3},
	{100, 0xa3d70a3e, 1, 5},
	{1000, 0x83126e98, 0, 9},
	{10000, 0xd1b71759, 0, 13},
	{100000, 0xa7c5ac48, 1, 15},
	{1000000, 0x8637bd06, 0, 19},
	{10000000, 0xd6bf94d6, 0, 23},
	{100000000, 0xabcc7712, 0, 26},
	{1000000000, 0x89705f42, 1, 28},
}

type magic struct {
	d    uint64 // divisor
	m    uint64 // multiplier
	pre  byte   // pre-shift
	post byte   // post-shift
}

func divisorPow10(n uint) magic {
	if debugDecimal && n == 0 {
		panic("divisorPow10: 10**0 is not a valid divisor")
	}
	if _W == 32 {
		return pow10DivTab32[n-1]
	}
	return pow10DivTab64[n-1]
}

func (m magic) div(n Word) (q, r Word) {
	h, _ := bits.Mul(uint(n)>>m.pre, uint(m.m))
	q = Word(h) >> m.post
	return q, n - q*Word(m.d)
}

//-----------------------------------------------------------------------------
// Arithmetic primitives
//

// z1<<_W + z0 = x*y
func mul10WW_g(x, y Word) (z1, z0 Word) {
	hi, lo := bits.Mul(uint(x), uint(y))
	return div10W_g(Word(hi), Word(lo))
}

// q = (u1<<_W + u0 - r)/v
func div10WW_g(u1, u0, v Word) (q, r Word) {
	// convert to base 2
	hi, lo := mulAddWWW_g(u1, _DB, u0)
	// q = (u-r)/v. Since v < _BD => r < _BD
	return divWW_g(hi, lo, v)
}

func add10WWW_g(x, y, cIn Word) (s, c Word) {
	r, cc := bits.Add(uint(x), uint(y), uint(cIn))
	var c1 uint
	// this simple if statement is compiled without jumps
	// at least on amd64.
	if r >= _DB {
		c1 = 1
	}
	cc |= c1
	r -= _DB & -cc
	return Word(r), Word(cc)
}

// The resulting carry c is either 0 or 1.
func add10VV_g(z, x, y []Word) (c Word) {
	for i := 0; i < len(z) && i < len(x) && i < len(y); i++ {
		z[i], c = add10WWW_g(x[i], y[i], c)
	}
	return
}

func sub10WWW_g(x, y, b Word) (d, c Word) {
	dd, cc := bits.Sub(uint(x), uint(y), uint(b))
	if cc != 0 {
		dd += _DB
	}
	return Word(dd), Word(cc)
}

// The resulting carry c is either 0 or 1.
func sub10VV_g(z, x, y []Word) (c Word) {
	for i := 0; i < len(z) && i < len(x) && i < len(y); i++ {
		z[i], c = sub10WWW_g(x[i], y[i], c)
	}
	return
}

// add10VW adds y to x. The resulting carry c is either 0 or 1.
func add10VW_g(z, x []Word, y Word) (c Word) {
	if len(z) == 0 {
		return y
	}
	z[0], c = add10WWW_g(x[0], y, 0)
	// propagate carry
	for i := 1; i < len(z) && i < len(x); i++ {
		s := x[i] + c
		if s < _DB {
			z[i] = s
			// copy remaining digits
			copy(z[i+1:], x[i+1:])
			return 0
		}
		z[i] = 0
	}
	return
}

func sub10VW_g(z, x []Word, y Word) (c Word) {
	c = y
	for i := 0; i < len(z) && i < len(x); i++ {
		zi, cc := bits.Sub(uint(x[i]), uint(c), 0)
		c = Word(cc)
		if c == 0 {
			z[i] = Word(zi)
			copy(z[i+1:], x[i+1:])
			return
		}
		z[i] = Word(zi + _DB)
	}
	return
}

// shl10VU sets z to x*(10**s), s < _WD
func shl10VU_g(z, x []Word, s uint) (r Word) {
	if s == 0 {
		copy(z, x)
		return
	}
	if len(z) == 0 || len(x) == 0 {
		return
	}
	d, m := divisorPow10(_DW-s), pow10(s)
	var h, l Word
	r, l = d.div(x[len(x)-1])
	for i := len(z) - 1; i > 0; i-- {
		t := l
		h, l = d.div(x[i-1])
		z[i] = t*m + h
	}
	z[0] = l * m

	return r
}

// shr10VU sets z to x/(10**s)
func shr10VU_g(z, x []Word, s uint) (r Word) {
	if s == 0 {
		copy(z, x)
		return
	}
	if len(z) == 0 || len(x) == 0 {
		return
	}

	var h, l Word
	d, m := divisorPow10(s), pow10(_DW-s)
	h, r = d.div(x[0])
	for i := 1; i < len(z) && i < len(x); i++ {
		t := h
		h, l = d.div(x[i])
		z[i-1] = t + l*m
	}
	z[len(z)-1] = h
	return r * m
}

func mulAdd10VWW_g(z, x []Word, y, r Word) (c Word) {
	c = r
	// The comment near the top of this file discusses this for loop condition.
	for i := 0; i < len(z) && i < len(x); i++ {
		hi, lo := mulAddWWW_g(x[i], y, c)
		c, z[i] = div10W_g(hi, lo)
	}
	return
}

func addMul10VVW_g(z, x []Word, y Word) (c Word) {
	for i := 0; i < len(z) && i < len(x); i++ {
		// do x[i] * y + c in base 2 => (hi+cc) * 2**_W + lo
		hi, z0 := mulAddWWW_g(x[i], y, z[i])
		lo, cc := bits.Add(uint(z0), uint(c), 0)
		c, z[i] = div10W_g(hi+Word(cc), Word(lo))
	}
	return
}

func div10VWW_g(z, x []Word, y, xn Word) (r Word) {
	r = xn
	for i := len(z) - 1; i >= 0; i-- {
		z[i], r = div10WW_g(r, x[i], y)
	}
	return
}

// div10W_g returns the quotient and remainder of a double-Word n divided by _DB:
//
// q = n/_DB, r = n%_DB
//
// with the dividend bits' upper half in parameter n1 and the lower half in
// parameter n0. divDecBase panics if n1 > _DMax (quotient overflow).
//
// This function uses the algorithm from "Division by invariant integers using
// multiplication" by Torbjörn Granlund & Peter L. Montgomery.
//
// See https://gmplib.org/~tege/divcnst-pldi94.pdf, section 8, Dividing udword
// by uword.
//
// In the article, some equations show an addition or subtraction of 2**N, which
// is a no-op. In the comments below, these have been removed for the sake of
// clarity.
//
func div10W_g(n1, n0 Word) (q, r Word) {
	const (
		N     = _W
		d     = _DB
		l     = _DWb
		mP    = (1<<(N+l)-1)/d - 1<<N // m'
		dNorm = d << (N - l)
	)
	if debugDecimal && n1 > _DMax {
		panic("decimal: integer overflow")
	}

	// if N == 64, N == l => n2 == n1 && n10 == n0
	// go vet complains, but this is optimized out.
	n2 := n1<<(N-l) + n0>>l
	n10 := n0 << (N - l)
	// -n1 = (n10 < 0 ? -1 : 0)
	_n1 := Word(int(n10) >> (N - 1))
	nAdj := n10 + (_n1 & dNorm)

	// q1 = n2 + HIGH(mP * (n2-_n1) + nAdj)
	q1, _ := mulAddWWW_g(mP, n2-_n1, nAdj)
	q1 += n2
	// dr = 2**N*n1 + n0 - 2**N*d + (-1-q1)*d
	//    = (-1-q1) * d + n0 +           (1)
	//      2**N * (n1 - d)              (2)
	// let t = -1 - q1 = (^q1 + 1) - 1 = ^q1
	t := ^q1
	drHi, drLo := mulAddWWW_g(t, d, n0) // (1)
	drHi += n1 - d                      // (2)
	// q = drHi - (-1-q1)
	// r = drLow + (d & drHi)
	return drHi - t, drLo + d&drHi
}

func mulAdd10WWW_g(x, y, c Word) (hi, lo Word) {
	return div10W(mulAddWWW_g(x, y, c))
}
