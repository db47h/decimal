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

// The resulting carry c is either 0 or 1.
func add10VW(z, x []Word, y Word) (c Word) {
	c = y
	for i := 0; i < len(z) && i < len(x); i++ {
		zi, cc := bits.Add(uint(x[i]), uint(c), 0)
		if zi >= _BD {
			zi -= _BD
			c = 1
		}
		z[i] = Word(zi)
		c = Word(cc)
	}
	return
}
