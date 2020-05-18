package decimal

import (
	"math/big"
	"math/bits"
	"sync"
)

const debugDecimal = true

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
		return uint(i*_DW) + decDigits(uint(x[i]))
	}
	return 0
}

func (x dec) ntz() uint {
	for i, w := range x {
		if w != 0 {
			return uint(i)*_DW + decTrailingZeros(uint(w))
		}
	}
	return 0
}

func (x dec) digit(i uint) uint {
	j, i := bits.Div(0, i, _DW)
	if j >= uint(len(x)) {
		return 0
	}
	// 0 <= j < len(x)
	return uint(x[j]/pow10(i)) % 10
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
	if w := Word(x); uint64(w) == x && w < _DB {
		return z.setWord(w), dig
	}
	// x could be a 2 to 3 words value
	z = z.make(int(dig+_DW-1) / _DW)
	for i := 0; i < len(z); i++ {
		hi, lo := bits.Div64(0, x, _DB)
		z[i] = Word(lo)
		x = hi
	}
	return z.norm(), dig
}

// toUint64 returns the low 64 bits of z or MaxUint64 and true if z <= MaxUint64.
func (x dec) toUint64() (uint64, bool) {
	// using a decToNat style loop would modify x
	// so we unroll the loop and cache the values.
	if _W == 64 {
		var lo, hi Word
		switch l := len(x); l {
		case 2:
			lo = x[1]
			fallthrough
		case 1:
			hi, lo = mulAddWWW_g(lo, _DB, x[0])
			fallthrough
		case 0:
			return uint64(lo), hi == 0
		default:
			return ^uint64(0), false
		}
	}
	var z2, z1, z0, r, lo Word
	switch l := len(x); l {
	case 3:
		z2 = x[2]
		fallthrough
	case 2:
		z1 = x[1]
		fallthrough
	case 1:
		z0 = x[0]
	case 0:
		return 0, true
	default:
		return ^uint64(0), false
	}
	z1, r = mulAddWWW_g(z2, _DB, z1)
	z0, r = mulAddWWW_g(r, _DB, z0)
	lo = r // low 32 bits
	z0, r = mulAddWWW_g(z1, _DB, z0)
	return uint64(r)<<32 | uint64(lo), z0 == 0
}

func decToNat(z []big.Word, x dec) []big.Word {
	if len(x) == 0 {
		return z[:0]
	}
	if len(x) == 1 {
		z = makeNat(z, 1)
		z[0] = big.Word(x[0])
		return z
	}
	// bits = x.digits() * Log(10) / Log(2)  + 1
	// words = (bits + _W - 1)/_W
	z = makeNat(z, (int(float64(x.digits())*log2_10)+_W)/_W)

	zz := dec(nil).set(x)
	for i := 0; i < len(z); i++ {
		// r = zz & _B; zz = zz >> _W
		var r Word
		for j := len(zz) - 1; j >= 0; j-- {
			zz[j], r = mulAddWWW_g(r, _DB, zz[j])
		}
		zz = zz.norm()
		z[i] = big.Word(r)
	}
	// normalize
	i := len(z)
	for i > 0 && z[i-1] == 0 {
		i--
	}
	return z[0:i]
}

// setNat sets z = x.mant
func (z dec) setNat(x []big.Word) dec {
	// here we cannot directly copy(b, bb) because big.Word != decimal.Word.
	b := make([]Word, len(x))
	for i := 0; i < len(b) && i < len(x); i++ {
		b[i] = Word(x[i])
	}
	for i := 0; i < len(z); i++ {
		z[i] = divWVW(b, 0, b, _DB)
	}
	z = z.norm()
	return z
}

// sticky returns 1 if there's a non zero digit within the
// i least significant digits, otherwise it returns 0.
func (x dec) sticky(i uint) uint {
	j, i := bits.Div(0, i, _DW)
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
	if x[j]%pow10(i) != 0 {
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
	r = div10VWW(z, x, y, 0)
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
	d := _DB / (vIn[n-1] + 1)
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

	if n < divRecursiveThreshold {
		q.divBasic(u, v)
	} else {
		q.divRecursive(u, v)
	}
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
// - q is large enough to hold the quotient u / v
//   which has a maximum length of len(u)-len(v)+1.
// - v[len(v)-1] >= _DB/2
func (q dec) divBasic(u, v dec) {
	n := len(v)
	m := len(u) - n

	qhatvp := getDec(n + 1)
	qhatv := *qhatvp
	// D2.
	vn1 := v[n-1]
	for j := m; j >= 0; j-- {
		// D3.
		qhat := Word(_DMax)
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
		// Compute the remainder u - (q̂*v) * 10**(_DW*j).
		// The subtraction may overflow if q̂ estimate was off by one.
		qhatv[n] = mulAdd10VWW(qhatv[0:n], v, qhat, 0)
		qhl := len(qhatv)
		if j+qhl > len(u) && qhatv[n] == 0 {
			qhl--
		}
		c := sub10VV(u[j:j+qhl], u[j:], qhatv)
		if c != 0 {
			c := add10VV(u[j:j+n], u[j:], v)
			// If n == qhl, the carry from subVV and the carry from addVV
			// cancel out and don't affect u[j+n].
			if n < qhl {
				u[j+n] += c
			}
			qhat--
		}

		if j == m && m == len(q) && qhat == 0 {
			continue
		}
		q[j] = qhat
	}

	putDec(qhatvp)
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

	n := m + int(s/_DW)
	z = z.make(n + 1)
	z[n] = shl10VU(z[n-m:n], x, s%_DW)
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
	n := m - int(s/_DW)
	if n <= 0 {
		return z[:0]
	}
	// n > 0

	z = z.make(n)
	shr10VU(z, x[m-n:], s%_DW)

	return z.norm()
}

// Operands that are shorter than basicSqrThreshold are squared using
// "grade school" multiplication; for operands longer than karatsubaSqrThreshold
// we use the Karatsuba algorithm optimized for x == y.

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

	if n < basicSqrThreshold {
		z = z.make(2 * n)
		decBasicMul(z, x, x)
		return z.norm()
	}
	if n < karatsubaSqrThreshold {
		z = z.make(2 * n)
		decBasicSqr(z, x)
		return z.norm()
	}
	// Use Karatsuba multiplication optimized for x == y.
	// The algorithm and layout of z are the same as for mul.

	// z = (x1*b + x0)^2 = x1^2*b^2 + 2*x1*x0*b + x0^2

	k := karatsubaLen(n, karatsubaSqrThreshold)

	x0 := x[0:k]
	z = z.make(max(6*k, 2*n))
	decKaratsubaSqr(z, x0) // z = x0^2
	z = z[0 : 2*n]
	z[2*k:].clear()

	if k < n {
		tp := getDec(2 * k)
		t := *tp
		x0 := x0.norm()
		x1 := x[k:]
		t = t.mul(x0, x1)
		decAddAt(z, t, k)
		decAddAt(z, t, k) // z = 2*x1*x0*b + x0^2
		t = t.sqr(x1)
		decAddAt(z, t, 2*k) // z = x1^2*b^2 + 2*x1*x0*b + x0^2
		putDec(tp)
	}

	return z.norm()
}

// basicSqr sets z = x*x and is asymptotically faster than basicMul
// by about a factor of 2, but slower for small arguments due to overhead.
// Requirements: len(x) > 0, len(z) == 2*len(x)
// The (non-normalized) result is placed in z.
func decBasicSqr(z, x dec) {
	n := len(x)
	tp := getDec(2 * n)
	t := *tp // temporary variable to hold the products
	t.clear()
	z[1], z[0] = mul10WW(x[0], x[0]) // the initial square
	for i := 1; i < n; i++ {
		d := x[i]
		// z collects the squares x[i] * x[i]
		z[2*i+1], z[2*i] = mul10WW(d, d)
		// t collects the products x[i] * x[j] where j < i
		t[2*i] = addMul10VVW(t[i:2*i], x[0:i], d)
	}
	// t[2*n-1] = shlVU(t[1:2*n-1], t[1:2*n-1], 1) // double the j < i products
	t[2*n-1] = mulAdd10VWW(t[1:2*n-1], t[1:2*n-1], 2, 0)
	add10VV(z, z, t) // combine the result
	putDec(tp)
}

// decKaratsubaSqr squares x and leaves the result in z.
// len(x) must be a power of 2 and len(z) >= 6*len(x).
// The (non-normalized) result is placed in z[0 : 2*len(x)].
//
// The algorithm and the layout of z are the same as for karatsuba.
func decKaratsubaSqr(z, x dec) {
	n := len(x)

	if n&1 != 0 || n < karatsubaSqrThreshold || n < 2 {
		decBasicSqr(z[:2*n], x)
		return
	}

	n2 := n >> 1
	x1, x0 := x[n2:], x[0:n2]

	decKaratsubaSqr(z, x0)
	decKaratsubaSqr(z[n:], x1)

	// s = sign(xd*yd) == -1 for xd != 0; s == 1 for xd == 0
	xd := z[2*n : 2*n+n2]
	if sub10VV(xd, x1, x0) != 0 {
		sub10VV(xd, x0, x1)
	}

	p := z[n*3:]
	decKaratsubaSqr(p, xd)

	r := z[n*4:]
	copy(r, z[:n*2])

	decKaratsubaAdd(z[n2:], r, n)
	decKaratsubaAdd(z[n2:], r[n:], n)
	decKaratsubaSub(z[n2:], p, n) // s == -1 for p != 0; s == 1 for p == 0
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
	if n < karatsubaThreshold {
		z = z.make(m + n)
		decBasicMul(z, x, y)
		return z.norm()
	}
	// m >= n && n >= karatsubaThreshold && n >= 2

	// determine Karatsuba length k such that
	//
	//   x = xh*b + x0  (0 <= x0 < b)
	//   y = yh*b + y0  (0 <= y0 < b)
	//   b = 10**(_DW*k)  ("base" of digits xi, yi)
	//
	k := karatsubaLen(n, karatsubaThreshold)
	// k <= n

	// // multiply x0 and y0 via Karatsuba
	x0 := x[0:k]              // x0 is not normalized
	y0 := y[0:k]              // y0 is not normalized
	z = z.make(max(6*k, m+n)) // enough space for karatsuba of x0*y0 and full result of x*y
	decKaratsuba(z, x0, y0)
	z = z[0 : m+n]  // z has final length but may be incomplete
	z[2*k:].clear() // upper portion of z is garbage (and 2*k <= m+n since k <= n <= m)

	// If xh != 0 or yh != 0, add the missing terms to z. For
	//
	//   xh = xi*b^i + ... + x2*b^2 + x1*b (0 <= xi < b)
	//   yh =                         y1*b (0 <= y1 < b)
	//
	// the missing terms are
	//
	//   x0*y1*b and xi*y0*b^i, xi*y1*b^(i+1) for i > 0
	//
	// since all the yi for i > 1 are 0 by choice of k: If any of them
	// were > 0, then yh >= b^2 and thus y >= b^2. Then k' = k*2 would
	// be a larger valid threshold contradicting the assumption about k.
	//
	if k < n || m != n {
		tp := getDec(3 * k)
		t := *tp

		// add x0*y1*b
		x0 := x0.norm()
		y1 := y[k:]       // y1 is normalized because y is
		t = t.mul(x0, y1) // update t so we don't lose t's underlying array
		decAddAt(z, t, k)

		// add xi*y0<<i, xi*y1*b<<(i+k)
		y0 := y0.norm()
		for i := k; i < len(x); i += k {
			xi := x[i:]
			if len(xi) > k {
				xi = xi[:k]
			}
			xi = xi.norm()
			t = t.mul(xi, y0)
			decAddAt(z, t, i)
			t = t.mul(xi, y1)
			decAddAt(z, t, i+k)
		}

		putDec(tp)
	}

	return z.norm()
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
	// TODO(db47h): this is a quick hack to get expNN working.
	yy := decToNat(make([]big.Word, 1), y)

	v := yy[len(yy)-1] // v > 0 because yy is normalized and y > 0
	shift := nlz(Word(v)) + 1
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

// divRecursive performs word-by-word division of u by v.
// The quotient is written in pre-allocated z.
// The remainder overwrites input u.
//
// Precondition:
// - len(z) >= len(u)-len(v)
//
// See Burnikel, Ziegler, "Fast Recursive Division", Algorithm 1 and 2.
// TODO(db47h): review https://pure.mpg.de/rest/items/item_1819444_4/component/file_2599480/content
// and make sure that when calling divBasic, the preconditions are met.
func (z dec) divRecursive(u, v dec) {
	// Recursion depth is less than 2 log2(len(v))
	// Allocate a slice of temporaries to be reused across recursion.
	recDepth := 2 * bits.Len(uint(len(v)))
	// large enough to perform Karatsuba on operands as large as v
	tmp := getDec(3 * len(v))
	temps := make([]*dec, recDepth)
	z.clear()
	z.divRecursiveStep(u, v, 0, tmp, temps)
	for _, n := range temps {
		if n != nil {
			putDec(n)
		}
	}
	putDec(tmp)
}

// divRecursiveStep computes the division of u by v.
// - z must be large enough to hold the quotient
// - the quotient will overwrite z
// - the remainder will overwrite u
func (z dec) divRecursiveStep(u, v dec, depth int, tmp *dec, temps []*dec) {
	u = u.norm()
	v = v.norm()

	if len(u) == 0 {
		z.clear()
		return
	}
	n := len(v)
	if n < divRecursiveThreshold {
		z.divBasic(u, v)
		return
	}
	m := len(u) - n
	if m < 0 {
		return
	}

	// Produce the quotient by blocks of B words.
	// Division by v (length n) is done using a length n/2 division
	// and a length n/2 multiplication for each block. The final
	// complexity is driven by multiplication complexity.
	B := n / 2

	// Allocate a nat for qhat below.
	if temps[depth] == nil {
		temps[depth] = getDec(n)
	} else {
		*temps[depth] = temps[depth].make(B + 1)
	}

	j := m
	for j > B {
		// Divide u[j-B:j+n] by vIn. Keep remainder in u
		// for next block.
		//
		// The following property will be used (Lemma 2):
		// if u = u1 << s + u0
		//    v = v1 << s + v0
		// then floor(u1/v1) >= floor(u/v)
		//
		// Moreover, the difference is at most 2 if len(v1) >= len(u/v)
		// We choose s = B-1 since len(v)-B >= B+1 >= len(u/v)
		s := (B - 1)
		// Except for the first step, the top bits are always
		// a division remainder, so the quotient length is <= n.
		uu := u[j-B:]

		qhat := *temps[depth]
		qhat.clear()
		qhat.divRecursiveStep(uu[s:B+n], v[s:], depth+1, tmp, temps)
		qhat = qhat.norm()
		// Adjust the quotient:
		//    u = u_h << s + u_l
		//    v = v_h << s + v_l
		//  u_h = q̂ v_h + rh
		//    u = q̂ (v - v_l) + rh << s + u_l
		// After the above step, u contains a remainder:
		//    u = rh << s + u_l
		// and we need to subtract q̂ v_l
		//
		// But it may be a bit too large, in which case q̂ needs to be smaller.
		qhatv := tmp.make(3 * n)
		qhatv.clear()
		qhatv = qhatv.mul(qhat, v[:s])
		for i := 0; i < 2; i++ {
			e := qhatv.cmp(uu.norm())
			if e <= 0 {
				break
			}
			sub10VW(qhat, qhat, 1)
			c := sub10VV(qhatv[:s], qhatv[:s], v[:s])
			if len(qhatv) > s {
				sub10VW(qhatv[s:], qhatv[s:], c)
			}
			decAddAt(uu[s:], v[s:], 0)
		}
		if qhatv.cmp(uu.norm()) > 0 {
			panic("impossible")
		}
		c := sub10VV(uu[:len(qhatv)], uu[:len(qhatv)], qhatv)
		if c > 0 {
			sub10VW(uu[len(qhatv):], uu[len(qhatv):], c)
		}
		decAddAt(z, qhat, j-B)
		j -= B
	}

	// Now u < (v<<B), compute lower bits in the same way.
	// Choose shift = B-1 again.
	s := B
	qhat := *temps[depth]
	qhat.clear()
	qhat.divRecursiveStep(u[s:].norm(), v[s:], depth+1, tmp, temps)
	qhat = qhat.norm()
	qhatv := tmp.make(3 * n)
	qhatv.clear()
	qhatv = qhatv.mul(qhat, v[:s])
	// Set the correct remainder as before.
	for i := 0; i < 2; i++ {
		if e := qhatv.cmp(u.norm()); e > 0 {
			sub10VW(qhat, qhat, 1)
			c := sub10VV(qhatv[:s], qhatv[:s], v[:s])
			if len(qhatv) > s {
				sub10VW(qhatv[s:], qhatv[s:], c)
			}
			decAddAt(u[s:], v[s:], 0)
		}
	}
	if qhatv.cmp(u.norm()) > 0 {
		panic("impossible")
	}
	c := sub10VV(u[0:len(qhatv)], u[0:len(qhatv)], qhatv)
	if c > 0 {
		c = sub10VW(u[len(qhatv):], u[len(qhatv):], c)
	}
	if c > 0 {
		panic("impossible")
	}

	// Done!
	decAddAt(z, qhat.norm(), 0)
}

// addAt implements z += x*10**(_WD*i); z must be long enough.
// (we don't use dec.add because we need z to stay the same
// slice, and we don't need to normalize z after each addition)
func decAddAt(z, x dec, i int) {
	if n := len(x); n > 0 {
		if c := add10VV(z[i:i+n], z[i:], x); c != 0 {
			j := i + n
			if j < len(z) {
				add10VW(z[j:], z[j:], c)
			}
		}
	}
}

// Fast version of z[0:n+n>>1].add(z[0:n+n>>1], x[0:n]) w/o bounds checks.
// Factored out for readability - do not use outside karatsuba.
func decKaratsubaAdd(z, x dec, n int) {
	if c := add10VV(z[0:n], z, x); c != 0 {
		add10VW(z[n:n+n>>1], z[n:], c)
	}
}

// Like karatsubaAdd, but does subtract.
func decKaratsubaSub(z, x dec, n int) {
	if c := sub10VV(z[0:n], z, x); c != 0 {
		sub10VW(z[n:n+n>>1], z[n:], c)
	}
}

// karatsuba multiplies x and y and leaves the result in z.
// Both x and y must have the same length n and n must be a
// power of 2. The result vector z must have len(z) >= 6*n.
// The (non-normalized) result is placed in z[0 : 2*n].
func decKaratsuba(z, x, y dec) {
	n := len(y)

	// Switch to basic multiplication if numbers are odd or small.
	// (n is always even if karatsubaThreshold is even, but be
	// conservative)
	if n&1 != 0 || n < karatsubaThreshold || n < 2 {
		decBasicMul(z, x, y)
		return
	}
	// n&1 == 0 && n >= karatsubaThreshold && n >= 2

	// Karatsuba multiplication is based on the observation that
	// for two numbers x and y with:
	//
	//   x = x1*b + x0
	//   y = y1*b + y0
	//
	// the product x*y can be obtained with 3 products z2, z1, z0
	// instead of 4:
	//
	//   x*y = x1*y1*b*b + (x1*y0 + x0*y1)*b + x0*y0
	//       =    z2*b*b +              z1*b +    z0
	//
	// with:
	//
	//   xd = x1 - x0
	//   yd = y0 - y1
	//
	//   z1 =      xd*yd                    + z2 + z0
	//      = (x1-x0)*(y0 - y1)             + z2 + z0
	//      = x1*y0 - x1*y1 - x0*y0 + x0*y1 + z2 + z0
	//      = x1*y0 -    z2 -    z0 + x0*y1 + z2 + z0
	//      = x1*y0                 + x0*y1

	// split x, y into "digits"
	n2 := n >> 1              // n2 >= 1
	x1, x0 := x[n2:], x[0:n2] // x = x1*b + y0
	y1, y0 := y[n2:], y[0:n2] // y = y1*b + y0

	// z is used for the result and temporary storage:
	//
	//   6*n     5*n     4*n     3*n     2*n     1*n     0*n
	// z = [z2 copy|z0 copy| xd*yd | yd:xd | x1*y1 | x0*y0 ]
	//
	// For each recursive call of karatsuba, an unused slice of
	// z is passed in that has (at least) half the length of the
	// caller's z.

	// compute z0 and z2 with the result "in place" in z
	decKaratsuba(z, x0, y0)     // z0 = x0*y0
	decKaratsuba(z[n:], x1, y1) // z2 = x1*y1

	// compute xd (or the negative value if underflow occurs)
	s := 1 // sign of product xd*yd
	xd := z[2*n : 2*n+n2]
	if sub10VV(xd, x1, x0) != 0 { // x1-x0
		s = -s
		sub10VV(xd, x0, x1) // x0-x1
	}

	// compute yd (or the negative value if underflow occurs)
	yd := z[2*n+n2 : 3*n]
	if sub10VV(yd, y0, y1) != 0 { // y0-y1
		s = -s
		sub10VV(yd, y1, y0) // y1-y0
	}

	// p = (x1-x0)*(y0-y1) == x1*y0 - x1*y1 - x0*y0 + x0*y1 for s > 0
	// p = (x0-x1)*(y0-y1) == x0*y0 - x0*y1 - x1*y0 + x1*y1 for s < 0
	p := z[n*3:]
	decKaratsuba(p, xd, yd)

	// save original z2:z0
	// (ok to use upper half of z since we're done recursing)
	r := z[n*4:]
	copy(r, z[:n*2])

	// add up all partial products
	//
	//   2*n     n     0
	// z = [ z2  | z0  ]
	//   +    [ z0  ]
	//   +    [ z2  ]
	//   +    [  p  ]
	//
	decKaratsubaAdd(z[n2:], r, n)
	decKaratsubaAdd(z[n2:], r[n:], n)
	if s > 0 {
		decKaratsubaAdd(z[n2:], p, n)
	} else {
		decKaratsubaSub(z[n2:], p, n)
	}
}
