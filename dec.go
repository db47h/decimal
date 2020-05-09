package decimal

import (
	"math/big"
	"math/bits"
	"sync"
)

const debugDecimal = true

const (
	// _W * log10(2) = decimal digits per word. 9 decimal digits per 32 bits
	// word and 19 per 64 bits word.
	_WD = _W * 30103 / 100000
	// Decimal base for a word. 1e9 for 32 bits words and 1e19 for 64 bits
	// words.
	// We want this value to be a const. This is a dirty hack to avoid
	// conditional compilation; it will break if bits.UintSize != 32 or 64
	_BD = 9999999998000000000*(_WD/19) + 1000000000*(_WD/9)
	_MD = _BD - 1
)

// dec is an unsigned integer x of the form
//
//   x = x[n-1]*_BD^(n-1) + x[n-2]*_BD^(n-2) + ... + x[1]*_BD + x[0]
//
// with 0 <= x[i] < _B and 0 <= i < n is stored in a slice of length n,
// with the digits x[i] as the slice elements.
//
// A number is normalized if the slice contains no leading 0 digits.
// During arithmetic operations, denormalized values may occur but are
// always normalized before returning the final result. The normalized
// representation of 0 is the empty or nil slice (length = 0).
type dec []Word

var (
	decOne = dec{1}
	decTwo = dec{2}
	decTen = dec{10}
)

func (z dec) clear() {
	for i := range z {
		z[i] = 0
	}
}

func (z dec) norm() dec {
	i := len(z)
	for i > 0 && z[i-1] == 0 {
		i--
	}
	return z[0:i]
}

// digits returns the number of digits of x.
func (x dec) digits() uint {
	if i := len(x) - 1; i >= 0 {
		return uint(i*_WD) + decDigits(uint(x[i]))
	}
	return 0
}

func (x dec) ntz() uint {
	for i, w := range x {
		if w != 0 {
			return uint(i)*_WD + decTrailingZeros(uint(w))
		}
	}
	return 0
}

func (x dec) digit(i uint) uint {
	j, i := bits.Div(0, i, _WD)
	if j >= uint(len(x)) {
		return 0
	}
	// 0 <= j < len(x)
	return (uint(x[j]) / pow10(i)) % 10
}

func (z dec) make(n int) dec {
	if n <= cap(z) {
		return z[:n] // reuse z
	}
	if n == 1 {
		// Most decs start small and stay that way; don't over-allocate.
		return make(dec, 1)
	}
	// Choosing a good value for e has significant performance impact
	// because it increases the chance that a value can be reused.
	const e = 4 // extra capacity
	return make(dec, n, n+e)
}

func (z dec) set(x dec) dec {
	z = z.make(len(x))
	copy(z, x)
	return z
}

func (z dec) setWord(x Word) dec {
	if x == 0 {
		return z[:0]
	}
	z = z.make(1)
	z[0] = x
	return z
}

func (z dec) setUint64(x uint64) (dec, int32) {
	dig := int32(decDigits64(x))
	if w := Word(x); uint64(w) == x && w < _BD {
		return z.setWord(w), dig
	}
	// x could be a 2 to 3 words value
	z = z.make(int(dig+_WD-1) / _WD)
	for i := 0; i < len(z); i++ {
		hi, lo := bits.Div64(0, x, _BD)
		z[i] = Word(lo)
		x = hi
	}
	return z, dig
}

func (x dec) toNat(z []Word) []Word {
	if len(x) == 0 {
		return dec(z)[:0]
	}
	if len(x) == 1 {
		return dec(z).setWord(x[0])
	}
	// bits = x.digits() * Log10 / Log2  + 1
	// words = (bits + _W - 1)/_W
	z = dec(z).make((int(float64(x.digits())*log2_10) + _W) / _W)
	zz := dec(nil).set(x)
	for i := 0; i < len(z); i++ {
		// r = zz & _B; zz = zz >> _W
		var r Word
		for j := len(zz) - 1; j >= 0; j-- {
			zz[j], r = mulAddWWW(r, _BD, zz[j])
		}
		zz = zz.norm()
		z[i] = r
	}
	return dec(z).norm()
}

// setInt sets z = x.mant
func (z dec) setInt(x *big.Int) dec {
	bb := x.Bits()
	// TODO(db47h): here we cannot directly copy(b, bb)
	// because big.Word != decimal.Word
	b := make([]Word, len(bb))
	for i := 0; i < len(b) && i < len(bb); i++ {
		b[i] = Word(bb[i])
	}
	for i := 0; i < len(z); i++ {
		z[i] = divWVW(b, 0, b, _BD)
	}
	z = z.norm()
	return z
}

// sticky returns 1 if there's a non zero digit within the
// i least significant digits, otherwise it returns 0.
func (x dec) sticky(i uint) uint {
	j, i := bits.Div(0, i, _WD)
	if j >= uint(len(x)) {
		if len(x) == 0 {
			return 0
		}
		return 1
	}
	// 0 <= j < len(x)
	for _, x := range x[:j] {
		if x != 0 {
			return 1
		}
	}
	if uint(x[j])%pow10(i) != 0 {
		return 1
	}
	return 0
}

func (z dec) add(x, y dec) dec {
	m := len(x)
	n := len(y)

	switch {
	case m < n:
		return z.add(y, x)
	case m == 0:
		// n == 0 because m >= n; result is 0
		return z[:0]
	case n == 0:
		// result is x
		return z.set(x)
	}
	// m > 0

	z = z.make(m + 1)
	c := add10VV(z[0:n], x, y)
	if m > n {
		c = add10VW(z[n:m], x[n:], c)
	}
	z[m] = c

	return z.norm()
}

func (z dec) sub(x, y dec) dec {
	m := len(x)
	n := len(y)

	switch {
	case m < n:
		panic("underflow")
	case m == 0:
		// n == 0 because m >= n; result is 0
		return z[:0]
	case n == 0:
		// result is x
		return z.set(x)
	}
	// m > 0

	z = z.make(m)
	c := sub10VV(z[0:n], x, y)
	if m > n {
		c = sub10VW(z[n:], x[n:], c)
	}
	if c != 0 {
		panic("underflow")
	}

	return z.norm()
}

func (x dec) cmp(y dec) (r int) {
	m := len(x)
	n := len(y)
	if m != n || m == 0 {
		switch {
		case m < n:
			r = -1
		case m > n:
			r = 1
		}
		return
	}

	i := m - 1
	for i > 0 && x[i] == y[i] {
		i--
	}

	switch {
	case x[i] < y[i]:
		r = -1
	case x[i] > y[i]:
		r = 1
	}
	return
}

// q = (x-r)/y, with 0 <= r < y
func (z dec) divW(x dec, y Word) (q dec, r Word) {
	m := len(x)
	switch {
	case y == 0:
		panic("division by zero")
	case y == 1:
		q = z.set(x) // result is x
		return
	case m == 0:
		q = z[:0] // result is 0
		return
	}
	// m > 0
	z = z.make(m)
	r = div10WVW(z, 0, x, y)
	q = z.norm()
	return
}

func (z dec) div(z2, u, v dec) (q, r dec) {
	if len(v) == 0 {
		panic("division by zero")
	}

	if u.cmp(v) < 0 {
		q = z[:0]
		r = z2.set(u)
		return
	}

	if len(v) == 1 {
		var r2 Word
		q, r2 = z.divW(u, v[0])
		r = z2.setWord(r2)
		return
	}

	q, r = z.divLarge(z2, u, v)
	return
}

// getDec returns a *dec of len n. The contents may not be zero.
// The pool holds *dec to avoid allocation when converting to interface{}.
func getDec(n int) *dec {
	var z *dec
	if v := decPool.Get(); v != nil {
		z = v.(*dec)
	}
	if z == nil {
		z = new(dec)
	}
	*z = z.make(n)
	return z
}

func putDec(x *dec) {
	decPool.Put(x)
}

var decPool sync.Pool

// q = (uIn-r)/vIn, with 0 <= r < vIn
// Uses z as storage for q, and u as storage for r if possible.
// See Knuth, Volume 2, section 4.3.1, Algorithm D.
// Preconditions:
//    len(vIn) >= 2
//    len(uIn) >= len(vIn)
//    u must not alias z
func (z dec) divLarge(u, uIn, vIn dec) (q, r dec) {
	n := len(vIn)
	m := len(uIn)

	// D1.
	d := _BD / (vIn[n-1] + 1)
	// do not modify vIn, it may be used by another goroutine simultaneously
	vp := getDec(n)
	v := *vp
	mulAdd10VWW(v, vIn, d, 0)

	// u may safely alias uIn or vIn, the value of uIn is used to set u and vIn was already used
	u = u.make(m + 1)
	u[m] = mulAdd10VWW(u[:m], uIn, d, 0)

	// z may safely alias uIn or vIn, both values were used already
	if alias(z, u) {
		z = nil // z is an alias for u - cannot reuse
	}
	q = z.make(m - n + 1)

	// TODO(db47h): implement divRecursive
	// if n < divRecursiveThreshold {
	q.divBasic(u, v)
	// } else {
	// 	q.divRecursive(u, v)
	// }
	putDec(vp)

	q = q.norm()
	r, _ = u.divW(u, d)
	r = r.norm()
	return q, r
}

// divBasic performs word-by-word division of u by v.
// The quotient is written in pre-allocated q.
// The remainder overwrites input u.
//
// Precondition:
// - len(q) >= len(u)-len(v)
func (q dec) divBasic(u, v dec) {
	n := len(v)
	m := len(u) - n

	qhatvp := getDec(n + 1)
	qhatv := *qhatvp
	// D2.
	vn1 := v[n-1]
	for j := m; j >= 0; j-- {
		// D3.
		qhat := Word(_MD)
		var ujn Word
		if j+n < len(u) {
			ujn = u[j+n]
		}
		if ujn != vn1 {
			var rhat Word
			qhat, rhat = div10WW(ujn, u[j+n-1], vn1)
			// x1 | x2 = q̂v_{n-2}
			vn2 := v[n-2]
			x1, x2 := mul10WW(qhat, vn2)
			// test if q̂v_{n-2} > br̂ + u_{j+n-2}
			ujn2 := u[j+n-2]
			for greaterThan(x1, x2, rhat, ujn2) {
				qhat--
				prevRhat := rhat
				rhat += vn1
				// v[n-1] >= 0, so this tests for overflow.
				if rhat < prevRhat {
					break
				}
				x1, x2 = mul10WW(qhat, vn2)
			}
		}

		// D4.
		qhatv[n] = mulAdd10VWW(qhatv[0:n], v, qhat, 0)
		qhl := len(qhatv)
		if j+qhl > len(u) && qhatv[n] == 0 {
			qhl--
		}
		c := sub10VV(u[j:j+qhl], u[j:], qhatv)
		if c != 0 {
			c := add10VV(u[j:j+n], u[j:], v)
			u[j+n] += c
			qhat--
		}

		if j == m && m == len(q) && qhat == 0 {
			continue
		}
		q[j] = qhat
	}

	putDec(qhatvp)
}

// greaterThan reports whether (x1*_BD + x2) > (y1*_BD + y2)
func greaterThan(x1, x2, y1, y2 Word) bool {
	return x1 > y1 || x1 == y1 && x2 > y2
}

// modW returns x % d.
func (x dec) modW(d Word) (r Word) {
	for i := len(x) - 1; i >= 0; i-- {
		_, r = div10WW(r, x[i], d)
	}
	return r
}

func (z dec) mulAddWW(x dec, y, r Word) dec {
	m := len(x)
	if m == 0 || y == 0 {
		return z.setWord(r) // result is r
	}
	// m > 0

	z = z.make(m + 1)
	z[m] = mulAdd10VWW(z[0:m], x, y, r)

	return z.norm()
}

// z = x * 10**s
func (z dec) shl(x dec, s uint) dec {
	if s == 0 {
		if same(z, x) {
			return z
		}
		if !alias(z, x) {
			return z.set(x)
		}
	}

	m := len(x)
	if m == 0 {
		return z[:0]
	}
	// m > 0

	n := m + int(s/_WD)
	z = z.make(n + 1)
	z[n] = shl10VU(z[n-m:n], x, s%_WD)
	z[0 : n-m].clear()

	return z.norm()
}

// z = x >> s
func (z dec) shr(x dec, s uint) dec {
	if s == 0 {
		if same(z, x) {
			return z
		}
		if !alias(z, x) {
			return z.set(x)
		}
	}

	m := len(x)
	n := m - int(s/_WD)
	if n <= 0 {
		return z[:0]
	}
	// n > 0

	z = z.make(n)
	shr10VU(z, x[m-n:], s%_WD)

	return z.norm()
}

// Operands that are shorter than basicSqrThreshold are squared using
// "grade school" multiplication; for operands longer than karatsubaSqrThreshold
// we use the Karatsuba algorithm optimized for x == y.
var decBasicSqrThreshold = 20      // computed by calibrate_test.go
var decKaratsubaSqrThreshold = 260 // computed by calibrate_test.go

// z = x*x
func (z dec) sqr(x dec) dec {
	n := len(x)
	switch {
	case n == 0:
		return z[:0]
	case n == 1:
		d := x[0]
		z = z.make(2)
		z[1], z[0] = mul10WW(d, d)
		return z.norm()
	}

	if alias(z, x) {
		z = nil // z is an alias for x - cannot reuse
	}

	// if n < decBasicSqrThreshold {
	z = z.make(2 * n)
	decBasicMul(z, x, x)
	return z.norm()
	// }
	// TODO(db47h): implement basicSqr
	// if n < decKaratsubaSqrThreshold {
	// 	z = z.make(2 * n)
	// 	basicSqr(z, x)
	// 	return z.norm()
	// }
	// TODO(db47h): implement karatsuba algorithm
	// Use Karatsuba multiplication optimized for x == y.
	// The algorithm and layout of z are the same as for mul.

	// z = (x1*b + x0)^2 = x1^2*b^2 + 2*x1*x0*b + x0^2

	// k := karatsubaLen(n, karatsubaSqrThreshold)

	// x0 := x[0:k]
	// z = z.make(max(6*k, 2*n))
	// karatsubaSqr(z, x0) // z = x0^2
	// z = z[0 : 2*n]
	// z[2*k:].clear()

	// if k < n {
	// 	tp := getNat(2 * k)
	// 	t := *tp
	// 	x0 := x0.norm()
	// 	x1 := x[k:]
	// 	t = t.mul(x0, x1)
	// 	addAt(z, t, k)
	// 	addAt(z, t, k) // z = 2*x1*x0*b + x0^2
	// 	t = t.sqr(x1)
	// 	addAt(z, t, 2*k) // z = x1^2*b^2 + 2*x1*x0*b + x0^2
	// 	putNat(tp)
	// }

	// return z.norm()
}

// decBasicMul multiplies x and y and leaves the result in z.
// The (non-normalized) result is placed in z[0 : len(x) + len(y)].
func decBasicMul(z, x, y dec) {
	z[0 : len(x)+len(y)].clear() // initialize z
	for i, d := range y {
		if d != 0 {
			z[len(x)+i] = addMul10VVW(z[i:i+len(x)], x, d)
		}
	}
}

func (z dec) mul(x, y dec) dec {
	m := len(x)
	n := len(y)

	switch {
	case m < n:
		return z.mul(y, x)
	case m == 0 || n == 0:
		return z[:0]
	case n == 1:
		return z.mulAddWW(x, y[0], 0)
	}
	// m >= n > 1

	// determine if z can be reused
	if alias(z, x) || alias(z, y) {
		z = nil // z is an alias for x or y - cannot reuse
	}

	// use basic multiplication if the numbers are small
	// if n < karatsubaThreshold {
	z = z.make(m + n)
	decBasicMul(z, x, y)
	return z.norm()
	// }
	// m >= n && n >= karatsubaThreshold && n >= 2

	// // determine Karatsuba length k such that
	// //
	// //   x = xh*b + x0  (0 <= x0 < b)
	// //   y = yh*b + y0  (0 <= y0 < b)
	// //   b = 1<<(_W*k)  ("base" of digits xi, yi)
	// //
	// k := karatsubaLen(n, karatsubaThreshold)
	// // k <= n

	// // multiply x0 and y0 via Karatsuba
	// x0 := x[0:k]              // x0 is not normalized
	// y0 := y[0:k]              // y0 is not normalized
	// z = z.make(max(6*k, m+n)) // enough space for karatsuba of x0*y0 and full result of x*y
	// karatsuba(z, x0, y0)
	// z = z[0 : m+n]  // z has final length but may be incomplete
	// z[2*k:].clear() // upper portion of z is garbage (and 2*k <= m+n since k <= n <= m)

	// // If xh != 0 or yh != 0, add the missing terms to z. For
	// //
	// //   xh = xi*b^i + ... + x2*b^2 + x1*b (0 <= xi < b)
	// //   yh =                         y1*b (0 <= y1 < b)
	// //
	// // the missing terms are
	// //
	// //   x0*y1*b and xi*y0*b^i, xi*y1*b^(i+1) for i > 0
	// //
	// // since all the yi for i > 1 are 0 by choice of k: If any of them
	// // were > 0, then yh >= b^2 and thus y >= b^2. Then k' = k*2 would
	// // be a larger valid threshold contradicting the assumption about k.
	// //
	// if k < n || m != n {
	// 	tp := getNat(3 * k)
	// 	t := *tp

	// 	// add x0*y1*b
	// 	x0 := x0.norm()
	// 	y1 := y[k:]       // y1 is normalized because y is
	// 	t = t.mul(x0, y1) // update t so we don't lose t's underlying array
	// 	addAt(z, t, k)

	// 	// add xi*y0<<i, xi*y1*b<<(i+k)
	// 	y0 := y0.norm()
	// 	for i := k; i < len(x); i += k {
	// 		xi := x[i:]
	// 		if len(xi) > k {
	// 			xi = xi[:k]
	// 		}
	// 		xi = xi.norm()
	// 		t = t.mul(xi, y0)
	// 		addAt(z, t, i)
	// 		t = t.mul(xi, y1)
	// 		addAt(z, t, i+k)
	// 	}

	// 	putNat(tp)
	// }

	// return z.norm()
}

// If m != 0 (i.e., len(m) != 0), expNN sets z to x**y mod m;
// otherwise it sets z to x**y. The result is the value of z.
func (z dec) expNN(x, y, m dec) dec {
	if alias(z, x) || alias(z, y) {
		// We cannot allow in-place modification of x or y.
		z = nil
	}

	// x**y mod 1 == 0
	if len(m) == 1 && m[0] == 1 {
		return z.setWord(0)
	}
	// m == 0 || m > 1

	// x**0 == 1
	if len(y) == 0 {
		return z.setWord(1)
	}
	// y > 0

	// x**1 mod m == x mod m
	if len(y) == 1 && y[0] == 1 && len(m) != 0 {
		_, z = dec(nil).div(z, x, m)
		return z
	}
	// y > 1

	if len(m) != 0 {
		// We likely end up being as long as the modulus.
		z = z.make(len(m))
	}

	// If the base is non-trivial and the exponent is large, we use
	// 4-bit, windowed exponentiation. This involves precomputing 14 values
	// (x^2...x^15) but then reduces the number of multiply-reduces by a
	// third. Even for a 32-bit exponent, this reduces the number of
	// operations. Uses Montgomery method for odd moduli.
	// TODO(db47h): implement montgomery & windowed algorithms
	// if x.cmp(decOne) > 0 && len(y) > 1 && len(m) > 0 {
	// 	if m[0]&1 == 1 {
	// 		return z.expNNMontgomery(x, y, m)
	// 	}
	// 	return z.expNNWindowed(x, y, m)
	// }

	// convert y from dec to base2 nat
	yy := y.toNat(make([]Word, 1))

	v := yy[len(yy)-1] // v > 0 because yy is normalized and y > 0
	shift := nlz(v) + 1
	v <<= shift

	// zz and r are used to avoid allocating in mul and div as
	// otherwise the arguments would alias.
	var zz, r dec

	// set x = x % m, this speeds up cases with large x even if len(y) == 1
	if len(m) != 0 {
		zz, r = zz.div(r, x, m)
		x = dec(nil).set(r)
	}
	z = z.set(x)

	const mask = 1 << (_W - 1)

	// We walk through the bits of the exponent one by one. Each time we
	// see a bit, we square, thus doubling the power. If the bit is a one,
	// we also multiply by x, thus adding one to the power.

	w := _W - int(shift)

	for j := 0; j < w; j++ {
		zz = zz.sqr(z)
		zz, z = z, zz

		if v&mask != 0 {
			zz = zz.mul(z, x)
			zz, z = z, zz
		}

		if len(m) != 0 {
			zz, r = zz.div(r, z, m)
			z, r = r, z
		}

		v <<= 1
	}

	for i := len(yy) - 2; i >= 0; i-- {
		v = yy[i]

		for j := 0; j < _W; j++ {
			zz = zz.sqr(z)
			zz, z = z, zz

			if v&mask != 0 {
				zz = zz.mul(z, x)
				zz, z = z, zz
			}

			if len(m) != 0 {
				zz, r = zz.div(r, z, m)
				z, r = r, z
			}

			v <<= 1
		}
	}

	return z.norm()
}
