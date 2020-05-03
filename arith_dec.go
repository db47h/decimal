package decimal

import "math/bits"

var pow10s = [...]uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000,
	10000000000, 100000000000, 1000000000000, 10000000000000, 100000000000000, 1000000000000000,
	10000000000000000, 100000000000000000, 1000000000000000000, 10000000000000000000,
}

func pow10(n uint) uint { return uint(pow10s[n]) }

var maxDigits = [...]uint{
	1, 1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 4, 5, 5,
	5, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 9, 9, 9, 10, 10,
	10, 10, 11, 11, 11, 12, 12, 12, 13, 13, 13, 13, 14, 14, 14, 15,
	15, 15, 16, 16, 16, 16, 17, 17, 17, 18, 18, 18, 19, 19, 19, 20, 20,
}

// mag returns the magnitude of x such that 10**(mag-1) <= x < 10**mag.
// Returns 0 for x == 0.
func mag(x uint) uint {
	d := maxDigits[bits.Len(x)]
	if x < pow10(d-1) {
		d--
	}
	return d
}

// shl10VU sets z to x*(10**s), s < _WD
func shl10VU(z, x dec, s uint) (r Word) {
	m := pow10(s)
	for i := 0; i < len(z) && i < len(x); i++ {
		h, l := bits.Mul(uint(x[i]), m)
		h, l = bits.Div(h, l, _BD)
		z[i] = Word(l) + r
		r = Word(h)
	}
	return r
}

// shr10VU sets z to x/(10**s)
func shr10VU(z, x dec, s uint) (r Word) {
	d, m := Word(pow10(s)), Word(pow10(_WD-s))
	for i := len(x) - 1; i >= 0; i-- {
		var q Word
		rm := r * m
		q, r = x[i]/d, x[i]%d
		z[i] = rm + q
	}
	return r
}

func decTrailingZeros(n uint) uint {
	if bits.UintSize == 32 {
		return dec32TrailingZeros(n)
	}
	return dec64TrailingZeros(uint64(n))
}

func dec32TrailingZeros(n uint) uint {
	var d uint
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

func dec64TrailingZeros(n uint64) uint {
	var d uint
	if n%10000000000000000 == 0 {
		n /= 10000000000000000
		d += 16
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

// addW adds y to x. The resulting carry c is either 0 or 1.
func add10VW(z, x dec, y Word) (c Word) {
	s := x[0] + y
	if (s < y) || s >= _BD {
		z[0] = s - _BD
		c = 1
	} else {
		z[0] = s
	}
	// propagate carry
	for i := 1; i < len(z) && i < len(x); i++ {
		s = x[i] + c
		if s == _BD {
			z[i] = 0
			continue
		}
		// c = 0 from this point
		z[i] = s
		// copy remaining digits if not adding in-place
		if !same(z, x) {
			copy(z[i+1:], x[i+1:])
		}
		return 0
	}
	return c
}
