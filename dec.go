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
	// TODO(db47h): We want this value to be a const. This is a dirty hack to
	// avoid conditional compilation that will break if bits.UintSize>64
	_BD = 9999999998000000000*(_WD/19) + 1000000000*(_WD/9)
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
	if w := Word(x); uint64(w) == x {
		return z.setWord(w), dig
	}
	// x could be a 2 to 3 words value
	z = z.make(int(dig+_WD-1) / _WD)
	for i := 0; i < len(z) && x != 0; i++ {
		hi, lo := bits.Div64(0, x, _BD)
		z[i] = Word(lo)
		x = hi
	}
	return z, dig
}

// setInt sets z = x.mant
func (z dec) setInt(x *big.Int) dec {
	bb := x.Bits()
	// TODO(db47h): here we cannot directly copy(b, bb)
	b := make([]Word, len(bb))
	for i := 0; i < len(b) && i < len(bb); i++ {
		b[i] = Word(bb[i])
	}
	var i int
	for i = 0; i < len(z); i++ {
		z[i] = Word(divWVW(b, 0, b, _BD))
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
	// TODO(db47h): optimize and bench shifts when s%_WD == 0
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
	// TODO(db47h): implement aratsuba algorithm
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
