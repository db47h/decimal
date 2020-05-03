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
	// words. TODO(db47h): We want this value to be a const. This is a dirty
	// hack to avoid conditional compilation that will break if bits.UintSize>64
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

// Returns z with leading zeros truncated and left shifted (in 10 base) such
// that the most significant digit is >= 1. Returns z and the left shift amount.
func (z dec) norm() (dec, uint) {
	var ls uint
	// find first non-zero word
	i := len(z)
	for i > 0 && z[i-1] == 0 {
		i--
		ls += _WD
	}
	z = z[:i]
	if len(z) == 0 {
		return z, 0
	}
	// partial shift
	if s := _WD - mag(uint(z[len(z)-1])); s != 0 {
		ls += s
		r := shl10VU(z, z, s)
		if debugDecimal && r != 0 {
			panic("shl10VU returned non zero carry")
		}
	}
	// remove trailing zeros
	for i, w := range z {
		if w != 0 {
			copy(z, z[i:])
			z = z[:len(z)-i]
			break
		}
	}
	return z, ls
}

// shr10 shifts z right by s decimal places. Returns
// z and the most significant digit removed and a boolean
// indicating if there were any non-zero digits following r
func (z dec) shr10(s uint) (d dec, r Word, tnz bool) {
	nw, s := s/_WD, s%_WD
	if nw > 0 {
		// save rounding digit
		r = z[nw-1]
		for _, w := range z[:nw-1] {
			tnz = tnz || w != 0
		}
		copy(z, z[nw:])
		z = z[:len(z)-int(nw)]
	}
	if s == 0 {
		r, m := r/(_BD-1), r%(_BD-1)
		return z, r, m != 0
	}
	tnz = tnz || r != 0
	// shift right by 0 < s < _WD
	r = shr10VU(z, z, s)
	p := Word(pow10(s - 1))
	r, m := r/p, r%p
	return z, r, tnz || m != 0
}

func (x dec) digits() uint {
	for i, w := range x {
		if w != 0 {
			return uint(len(x)-i)*_WD - decTrailingZeros(uint(w))
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

func (z dec) set(x dec) dec {
	z = z.make(len(x))
	copy(z, x)
	return z
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

// setInt sets z such that z*10**exp = x with 0 < z <= 1.
// Returns z and exp.
func (z dec) setInt(x *big.Int) (dec, uint) {
	b := new(big.Int).Set(x).Bits()
	var i int
	for i = 0; i < len(z) && len(b) > 0; i++ {
		z[i] = Word(divWVW(b, 0, b, big.Word(_BD)))
	}
	z = z[:i]
	z, s := z.norm()
	return z, uint(i)*_WD - s
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
