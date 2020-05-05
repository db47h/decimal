package decimal

import (
	"math/bits"
)

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

// decDigits returns n such that of x such that 10**(n-1) <= x < 10**n.
// In other words, n the number of digits required to represent n.
// Returns 0 for x == 0.
func decDigits(x uint) (n uint) {
	n = maxDigits[bits.Len(x)]
	if x < pow10(n-1) {
		n--
	}
	return n
}

// shl10VU sets z to x*(10**s), s < _WD
func shl10VU(z, x dec, s uint) (r Word) {
	if s == 0 {
		copy(z, x)
		return
	}
	if len(z) == 0 || len(x) == 0 {
		return
	}
	d, m := Word(pow10(_WD-s)), Word(pow10(s))
	var h, l Word
	r, l = divWW(0, x[len(x)-1], d)
	for i := len(z) - 1; i > 0; i-- {
		t := l
		h, l = divWW(0, x[i-1], d)
		z[i] = t*m + h
	}
	z[0] = l * m

	return r
}

// shr10VU sets z to x/(10**s)
func shr10VU(z, x dec, s uint) (r Word) {
	if s == 0 {
		copy(z, x)
		return
	}
	if len(z) == 0 || len(x) == 0 {
		return
	}

	var h, l Word
	d, m := Word(pow10(s)), Word(pow10(_WD-s))
	h, r = divWW(0, x[0], Word(d))
	for i := 1; i < len(z) && i < len(x); i++ {
		t := h
		h, l = divWW(0, x[i], d)
		z[i-1] = t + l*m
	}
	z[len(z)-1] = h
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
	0, 0, 0, 0, 9223372036854775808, 63, 4052555153018976267, 39, 4611686018427387904, 31, 7450580596923828125, 27, 9983543956220149760, 25,
	8922003266371364727, 23, 9223372036854775808, 21, 1350851717672992089, 19, 10000000000000000000, 19, 8667208279016151025, 20, 8176589207175692288, 18, 8650415919381337933, 17,
	2177953337809371136, 16, 6568408355712890625, 16, 1152921504606846976, 15, 2862423051509815793, 15, 6746640616477458432, 15, 799006685782884121, 14, 1638400000000000000, 14,
	3243919932521508681, 14, 7752859499445190656, 15, 504036361936467383, 13, 6795192965888212992, 15, 1490116119384765625, 13, 9169742482168496128, 14, 4052555153018976267, 13,
	6502111422497947648, 13, 353814783205469041, 12, 531441000000000000, 12, 5970802223735490975, 13, 1152921504606846976, 12, 1667889514952984961, 12, 7351326950727229440, 13,
	7592253339725112179, 13, 4738381338321616896, 12, 6582952005840035281, 12, 9065737908494995456, 12, 317475837322472439, 11, 419430400000000000, 11, 4116746226656634465, 12,
	717368321110468608, 11, 3066142649843473169, 12, 1196683881290399744, 11, 1532278301220703125, 11, 1951354384207722496, 11, 5511018666691268545, 12, 3116402981210161152, 11,
	7113790643470898241, 12, 4882812500000000000, 11, 6071163615208263051, 11, 7516865509350965248, 11, 9269035929372191597, 11, 210832519264920576, 10, 253295162119140625, 10,
	303305489096114176, 10, 2189155819333249577, 11, 6539899926455986176, 11, 511116753300641401, 10, 604661760000000000, 10, 6620429464016735429, 11, 839299365868340224, 10,
}

// decMaxPow returns (b**n, n) such that b**n is the largest power b**n such that (b**n) <= _BD.
// For instance decMaxPow(10) == (1e19 - 1, 19) for 19 decimal digits in a 64bit Word.
// In other words, at most n digits in base b fit into a decimal Word.
func decMaxPow(b Word) (p Word, n int) {
	i := b * 2
	if bits.UintSize == 32 {
		return Word(decMaxPow32[i]), int(decMaxPow32[i+1])
	}
	return Word(decMaxPow64[i]), int(decMaxPow64[i+1])
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

func div10WVW(z []Word, xn Word, x []Word, y Word) (r Word) {
	r = xn
	for i := len(z) - 1; i >= 0; i-- {
		h, l := mulAddWWW(r, _BD, x[i])
		z[i], r = divWW(h, l, y)
	}
	return
}

func mulAdd10VWW(z, x []Word, y, r Word) (c Word) {
	c = r
	// The comment near the top of this file discusses this for loop condition.
	for i := 0; i < len(z) && i < len(x); i++ {
		c, z[i] = mulAdd10WWW(x[i], y, c)
	}
	return
}

// z1*_BD + z0 = x*y + c
func mulAdd10WWW(x, y, c Word) (z1, z0 Word) {
	hi, lo := bits.Mul(uint(x), uint(y))
	var cc uint
	lo, cc = bits.Add(lo, uint(c), 0)
	hi, lo = bits.Div(hi+cc, lo, _BD)
	return Word(hi), Word(lo)
}
