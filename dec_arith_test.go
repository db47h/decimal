package decimal

import (
	"fmt"
	"math/bits"
	"testing"
)

func TestDecDigits(t *testing.T) {
	for i := 0; i < 10000; i++ {
		n := uint(rnd.Uint64())
		d := uint(0)
		for m := n; m != 0; m /= 10 {
			d++
		}
		if dd := decDigits(n); dd != d {
			t.Fatalf("decDigits(%d) = %d, expected %d", n, dd, d)
		}
	}
}

func BenchmarkDecDigits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchH = Word(decDigits(uint(rnd.Uint64()) % _DB))
	}
}

func rnd10W() Word {
	return Word(rnd.Uint64() % _DB)
}

func rnd10V(n int) []Word {
	v := make([]Word, n)
	for i := range v {
		v[i] = rnd10W()
	}
	return v
}

func TestDecDiv10W(t *testing.T) {
	for i := 0; i < 1e7; i++ {
		h, l := rnd10W(), Word(rnd.Uint64())
		q, r := div10W(h, l)
		qq, rr := bits.Div(uint(h), uint(l), _DB)
		if q != Word(qq) || r != Word(rr) {
			t.Fatalf("Got (%d,%d)/_DB = %d, %d. Expected %d %d", h, l, q, r, qq, rr)
		}
	}
}

var benchH, benchL Word

func BenchmarkDecDiv10W_bits(b *testing.B) {
	h, l := rnd10W(), Word(rnd.Uint64())
	for i := 0; i < b.N; i++ {
		h, l := bits.Div(uint(h), uint(l), _DB)
		benchH, benchL = Word(h), Word(l)
	}
}

func BenchmarkDecDiv10W_mul(b *testing.B) {
	h, l := rnd10W(), Word(rnd.Uint64())
	for i := 0; i < b.N; i++ {
		benchH, benchL = div10W(h, l)
	}
}

///////////////////////////

type fun10VV func(z, x, y []Word) (c Word)
type arg10VV struct {
	z, x, y dec
	c       Word
}

var sum10VV = []arg10VV{
	{},
	{dec{0}, dec{0}, dec{0}, 0},
	{dec{1}, dec{1}, dec{0}, 0},
	{dec{0}, dec{_DMax}, dec{1}, 1},
	{dec{80235}, dec{12345}, dec{67890}, 0},
	{dec{_DMax - 1}, dec{_DMax}, dec{_DMax}, 1},
	{dec{0, 0, 0, 0}, dec{_DMax, _DMax, _DMax, _DMax}, dec{1, 0, 0, 0}, 1},
	{dec{0, 0, 0, _DMax}, dec{_DMax, _DMax, _DMax, _DMax - 1}, dec{1, 0, 0, 0}, 0},
	{dec{0, 0, 0, 0}, dec{_DMax, 0, _DMax, 0}, dec{1, _DMax, 0, _DMax}, 1},
}

func testFun10VV(t *testing.T, msg string, f fun10VV, a arg10VV) {
	z := make(dec, len(a.z))
	c := f(z, a.x, a.y)
	for i, zi := range z {
		if zi != a.z[i] {
			t.Errorf("%s%+v\n\tgot z[%d] = %d; want %d", msg, a, i, zi, a.z[i])
			break
		}
	}
	if c != a.c {
		t.Errorf("%s%+v\n\tgot c = %d; want %d", msg, a, c, a.c)
	}
}

func TestDecFun10VV(t *testing.T) {
	for _, a := range sum10VV {
		arg := a
		testFun10VV(t, "add10VV_g", add10VV_g, arg)
		testFun10VV(t, "add10VV", add10VV, arg)

		arg = arg10VV{a.z, a.y, a.x, a.c}
		testFun10VV(t, "add10VV_g symmetric", add10VV_g, arg)
		testFun10VV(t, "add10VV symmetric", add10VV, arg)

		arg = arg10VV{a.x, a.z, a.y, a.c}
		testFun10VV(t, "sub10VV_g", sub10VV_g, arg)
		testFun10VV(t, "sub10VV", sub10VV, arg)

		arg = arg10VV{a.y, a.z, a.x, a.c}
		testFun10VV(t, "sub10VV_g symmetric", sub10VV_g, arg)
		testFun10VV(t, "sub10VV symmetric", sub10VV, arg)
	}
}

func BenchmarkDecAdd10VV(b *testing.B) {
	for _, n := range benchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		x := rnd10V(n)
		y := rnd10V(n)
		z := make([]Word, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(n * _W))
			for i := 0; i < b.N; i++ {
				add10VV(z, x, y)
			}
		})
	}
}

func BenchmarkDecSub10VV(b *testing.B) {
	for _, n := range benchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		x := rnd10V(n)
		y := rnd10V(n)
		z := make([]Word, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(n * _W))
			for i := 0; i < b.N; i++ {
				sub10VV(z, x, y)
			}
		})
	}
}

type fun10VW func(z, x []Word, y Word) (c Word)
type arg10VW struct {
	z, x dec
	y    Word
	c    Word
}

var sum10VW = []arg10VW{
	{},
	{nil, nil, 2, 2},
	{dec{0}, dec{0}, 0, 0},
	{dec{1}, dec{0}, 1, 0},
	{dec{1}, dec{1}, 0, 0},
	{dec{0}, dec{_DMax}, 1, 1},
	{dec{0, 0, 0, 0, 0}, dec{_DMax, _DMax, _DMax, _DMax, _DMax}, 1, 1},
	{dec{585}, dec{314}, 271, 0},
}

var lsh10VW = []arg10VW{
	{},
	{dec{0}, dec{0}, 0, 0},
	{dec{0}, dec{0}, 1, 0},
	{dec{0}, dec{0}, 7, 0},

	{dec{_DMax}, dec{_DMax}, 0, 0},
	{dec{_DMax - _DMax%pow10(1)}, dec{_DMax}, 1, _DMax / pow10(_DW-1)},
	{dec{_DMax - _DMax%pow10(7)}, dec{_DMax}, 7, _DMax / pow10(_DW-7)},

	{dec{_DMax, _DMax, _DMax}, dec{_DMax, _DMax, _DMax}, 0, 0},
	{dec{_DMax - _DMax%pow10(1), _DMax, _DMax}, dec{_DMax, _DMax, _DMax}, 1, _DMax / pow10(_DW-1)},
	{dec{_DMax - _DMax%pow10(7), _DMax, _DMax}, dec{_DMax, _DMax, _DMax}, 7, _DMax / pow10(_DW-7)},
}

var rsh10VW = []arg10VW{
	{},
	{dec{0}, dec{0}, 0, 0},
	{dec{0}, dec{0}, 1, 0},
	{dec{0}, dec{0}, 7, 0},

	{dec{_DMax}, dec{_DMax}, 0, 0},
	{dec{_DMax / pow10(1)}, dec{_DMax}, 1, _DMax - _DMax%pow10(_DW-1)},
	{dec{_DMax / pow10(7)}, dec{_DMax}, 7, _DMax - _DMax%pow10(_DW-7)},

	{dec{_DMax, _DMax, _DMax}, dec{_DMax, _DMax, _DMax}, 0, 0},
	{dec{_DMax, _DMax, _DMax / pow10(1)}, dec{_DMax, _DMax, _DMax}, 1, _DMax - _DMax%pow10(_DW-1)},
	{dec{_DMax, _DMax, _DMax / pow10(7)}, dec{_DMax, _DMax, _DMax}, 7, _DMax - _DMax%pow10(_DW-7)},
}

func testFun10VW(t *testing.T, msg string, f fun10VW, a arg10VW) {
	n := len(a.z)
	z := make(nat, n+1)
	c := f(z[:n], a.x, a.y)
	for i, zi := range z[:n] {
		if zi != a.z[i] {
			t.Errorf("%s%+v\n\tgot z[%d] = %d; want %d", msg, a, i, zi, a.z[i])
			break
		}
	}
	if c != a.c {
		t.Errorf("%s%+v\n\tgot c = %d; want %d", msg, a, c, a.c)
	}
	// TestDecAddSub10VW sets x[len(x)] = some value
	// check that it does not get copied.
	if z[n] != 0 {
		panic("memcpy overflow")
	}
}

func makeFun10VW(f func(z, x []Word, s uint) (c Word)) fun10VW {
	return func(z, x []Word, s Word) (c Word) {
		return f(z, x, uint(s))
	}
}

func TestDecFun10VW(t *testing.T) {
	for _, a := range sum10VW {
		arg := a
		testFun10VW(t, "add10VW_g", add10VW_g, arg)
		testFun10VW(t, "add10VW", add10VW, arg)

		arg = arg10VW{a.x, a.z, a.y, a.c}
		testFun10VW(t, "sub10VW_g", sub10VW_g, arg)
		testFun10VW(t, "sub10VW", sub10VW, arg)
	}

	shl10VW_g := makeFun10VW(shl10VU_g)
	shl10VW := makeFun10VW(shl10VU)
	for _, a := range lsh10VW {
		arg := a
		testFun10VW(t, "shl10VU_g", shl10VW_g, arg)
		testFun10VW(t, "shl10VU", shl10VW, arg)
	}

	shr10VW_g := makeFun10VW(shr10VU_g)
	shr10VW := makeFun10VW(shr10VU)
	for _, a := range rsh10VW {
		arg := a
		testFun10VW(t, "shr10VU_g", shr10VW_g, arg)
		testFun10VW(t, "shr10VU", shr10VW, arg)
	}
}

// TestDecAddSub10VW tests proper behavior of assembly versions of add10Vw and
// sub10VW on edge cases.
func TestDecAddSub10VW(t *testing.T) {
	for n := 0; n < 10; n++ {
		for i := 0; i <= n; i++ {
			z := dec(nil).make(n)
			x := dec(nil).make(n + 1)
			// Bounds check. testFun10VW will allocate a larger result slice and
			// check that the higher Words are not overwritten.
			x[n] = 42
			x = x[:n]
			// test _DMax + 1 = 0
			// fill x[:j] with _DMax, z[:j] with 0
			for j := 0; j < i; j++ {
				x[j] = _DMax
			}
			// fill x[j:] and z[:j] with random-ish data
			for j := i; j < n; j++ {
				x[j] = Word(j + 1)
				z[j] = Word(j + 1)
				// add carry
				if j == i {
					z[j]++
				}
			}
			c := Word(0)
			if i == n {
				c = 1
			}
			testFun10VW(t, "add10VW_asm", add10VW, arg10VW{z, x, 1, c})
			testFun10VW(t, "sub10VW_asm", sub10VW, arg10VW{x, z, 1, c})
		}
	}
}

type arg10VU struct {
	d  []Word // d is a Word slice, the input parameters x and z come from this array.
	l  uint   // l is the length of the input parameters x and z.
	xp uint   // xp is the starting position of the input parameter x, x := d[xp:xp+l].
	zp uint   // zp is the starting position of the input parameter z, z := d[zp:zp+l].
	s  uint   // s is the shift number.
	r  []Word // r is the expected output result z.
	c  Word   // c is the expected return value.
	m  string // message.
}

var argshl10VU = []arg10VU{
	// test cases for shlVU
	{[]Word{1, _DMax, _DMax, _DMax, _DMax, _DMax, 99 * pow10(_DW-2), 0}, 7, 0, 0, 1, []Word{10, _DMax - _DMax%pow10(1), _DMax, _DMax, _DMax, _DMax, 9*pow10(_DW-1) + 9}, 9, "complete overlap of shlVU"},
	{[]Word{1, _DMax, _DMax, _DMax, _DMax, _DMax, 99 * pow10(_DW-2), 0, 0, 0, 0}, 7, 0, 3, 1, []Word{10, _DMax - _DMax%pow10(1), _DMax, _DMax, _DMax, _DMax, 9*pow10(_DW-1) + 9}, 9, "partial overlap by half of shlVU"},
	{[]Word{1, _DMax, _DMax, _DMax, _DMax, _DMax, 99 * pow10(_DW-2), 0, 0, 0, 0, 0, 0, 0}, 7, 0, 6, 1, []Word{10, _DMax - _DMax%pow10(1), _DMax, _DMax, _DMax, _DMax, 9*pow10(_DW-1) + 9}, 9, "partial overlap by 1 Word of shlVU"},
	{[]Word{1, _DMax, _DMax, _DMax, _DMax, _DMax, 99 * pow10(_DW-2), 0, 0, 0, 0, 0, 0, 0, 0}, 7, 0, 7, 1, []Word{10, _DMax - _DMax%pow10(1), _DMax, _DMax, _DMax, _DMax, 9*pow10(_DW-1) + 9}, 9, "no overlap of shlVU"},
}

var argshr10VU = []arg10VU{
	// test cases for shrVU
	{[]Word{0, 99, _DMax, _DMax, _DMax, _DMax, _DMax, 9 * pow10(_DW-1)}, 7, 1, 1, 1, []Word{9*pow10(_DW-1) + 9, _DMax, _DMax, _DMax, _DMax, _DMax / pow10(1), 9 * pow10(_DW-2)}, 9 * pow10(_DW-1), "complete overlap of shrVU"},
	{[]Word{0, 0, 0, 0, 99, _DMax, _DMax, _DMax, _DMax, _DMax, 9 * pow10(_DW-1)}, 7, 4, 1, 1, []Word{9*pow10(_DW-1) + 9, _DMax, _DMax, _DMax, _DMax, _DMax / pow10(1), 9 * pow10(_DW-2)}, 9 * pow10(_DW-1), "partial overlap by half of shrVU"},
	{[]Word{0, 0, 0, 0, 0, 0, 0, 99, _DMax, _DMax, _DMax, _DMax, _DMax, 9 * pow10(_DW-1)}, 7, 7, 1, 1, []Word{9*pow10(_DW-1) + 9, _DMax, _DMax, _DMax, _DMax, _DMax / pow10(1), 9 * pow10(_DW-2)}, 9 * pow10(_DW-1), "partial overlap by 1 Word of shrVU"},
	{[]Word{0, 0, 0, 0, 0, 0, 0, 0, 99, _DMax, _DMax, _DMax, _DMax, _DMax, 9 * pow10(_DW-1)}, 7, 8, 1, 1, []Word{9*pow10(_DW-1) + 9, _DMax, _DMax, _DMax, _DMax, _DMax / pow10(1), 9 * pow10(_DW-2)}, 9 * pow10(_DW-1), "no overlap of shrVU"},
}

func testShift10Func(t *testing.T, f func(z, x []Word, s uint) Word, a arg10VU) {
	// save a.d for error message, or it will be overwritten.
	b := make([]Word, len(a.d))
	copy(b, a.d)
	z := a.d[a.zp : a.zp+a.l]
	x := a.d[a.xp : a.xp+a.l]
	c := f(z, x, a.s)
	for i, zi := range z {
		if zi != a.r[i] {
			t.Errorf("d := %v, %s(d[%d:%d], d[%d:%d], %d)\n\tgot z[%d] = %d; want %d", b, a.m, a.zp, a.zp+a.l, a.xp, a.xp+a.l, a.s, i, zi, a.r[i])
			break
		}
	}
	if c != a.c {
		t.Errorf("d := %v, %s(d[%d:%d], d[%d:%d], %d)\n\tgot c = %d; want %d", b, a.m, a.zp, a.zp+a.l, a.xp, a.xp+a.l, a.s, c, a.c)
	}
}

func TestShift10Overlap(t *testing.T) {
	for _, a := range argshl10VU {
		arg := a
		testShift10Func(t, shl10VU, arg)
	}

	for _, a := range argshr10VU {
		arg := a
		testShift10Func(t, shr10VU, arg)
	}
}

func BenchmarkAdd10VW(b *testing.B) {
	for _, n := range benchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		x := rnd10V(n)
		y := rnd10W()
		z := make([]Word, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(n * _S))
			for i := 0; i < b.N; i++ {
				add10VW(z, x, y)
			}
		})
	}
}

func BenchmarkSub10VW(b *testing.B) {
	for _, n := range benchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		x := rnd10V(n)
		y := rnd10W()
		z := make([]Word, n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(n * _S))
			for i := 0; i < b.N; i++ {
				sub10VW(z, x, y)
			}
		})
	}
}

// TODO(db47h): complete port of the tests

// type funVWW func(z, x []Word, y, r Word) (c Word)
// type argVWW struct {
// 	z, x nat
// 	y, r Word
// 	c    Word
// }

// var prodVWW = []argVWW{
// 	{},
// 	{nat{0}, nat{0}, 0, 0, 0},
// 	{nat{991}, nat{0}, 0, 991, 0},
// 	{nat{0}, nat{_M}, 0, 0, 0},
// 	{nat{991}, nat{_M}, 0, 991, 0},
// 	{nat{0}, nat{0}, _M, 0, 0},
// 	{nat{991}, nat{0}, _M, 991, 0},
// 	{nat{1}, nat{1}, 1, 0, 0},
// 	{nat{992}, nat{1}, 1, 991, 0},
// 	{nat{22793}, nat{991}, 23, 0, 0},
// 	{nat{22800}, nat{991}, 23, 7, 0},
// 	{nat{0, 0, 0, 22793}, nat{0, 0, 0, 991}, 23, 0, 0},
// 	{nat{7, 0, 0, 22793}, nat{0, 0, 0, 991}, 23, 7, 0},
// 	{nat{0, 0, 0, 0}, nat{7893475, 7395495, 798547395, 68943}, 0, 0, 0},
// 	{nat{991, 0, 0, 0}, nat{7893475, 7395495, 798547395, 68943}, 0, 991, 0},
// 	{nat{0, 0, 0, 0}, nat{0, 0, 0, 0}, 894375984, 0, 0},
// 	{nat{991, 0, 0, 0}, nat{0, 0, 0, 0}, 894375984, 991, 0},
// 	{nat{_M << 1 & _M}, nat{_M}, 1 << 1, 0, _M >> (_W - 1)},
// 	{nat{_M<<1&_M + 1}, nat{_M}, 1 << 1, 1, _M >> (_W - 1)},
// 	{nat{_M << 7 & _M}, nat{_M}, 1 << 7, 0, _M >> (_W - 7)},
// 	{nat{_M<<7&_M + 1<<6}, nat{_M}, 1 << 7, 1 << 6, _M >> (_W - 7)},
// 	{nat{_M << 7 & _M, _M, _M, _M}, nat{_M, _M, _M, _M}, 1 << 7, 0, _M >> (_W - 7)},
// 	{nat{_M<<7&_M + 1<<6, _M, _M, _M}, nat{_M, _M, _M, _M}, 1 << 7, 1 << 6, _M >> (_W - 7)},
// }

// func testFunVWW(t *testing.T, msg string, f funVWW, a argVWW) {
// 	z := make(nat, len(a.z))
// 	c := f(z, a.x, a.y, a.r)
// 	for i, zi := range z {
// 		if zi != a.z[i] {
// 			t.Errorf("%s%+v\n\tgot z[%d] = %#x; want %#x", msg, a, i, zi, a.z[i])
// 			break
// 		}
// 	}
// 	if c != a.c {
// 		t.Errorf("%s%+v\n\tgot c = %#x; want %#x", msg, a, c, a.c)
// 	}
// }

// type funWVW func(z []Word, xn Word, x []Word, y Word) (r Word)
// type argWVW struct {
// 	z  nat
// 	xn Word
// 	x  nat
// 	y  Word
// 	r  Word
// }

// func testFunWVW(t *testing.T, msg string, f funWVW, a argWVW) {
// 	z := make(nat, len(a.z))
// 	r := f(z, a.xn, a.x, a.y)
// 	for i, zi := range z {
// 		if zi != a.z[i] {
// 			t.Errorf("%s%+v\n\tgot z[%d] = %#x; want %#x", msg, a, i, zi, a.z[i])
// 			break
// 		}
// 	}
// 	if r != a.r {
// 		t.Errorf("%s%+v\n\tgot r = %#x; want %#x", msg, a, r, a.r)
// 	}
// }

// func TestFunVWW(t *testing.T) {
// 	for _, a := range prodVWW {
// 		arg := a
// 		testFunVWW(t, "mulAddVWW_g", mulAddVWW_g, arg)
// 		testFunVWW(t, "mulAddVWW", mulAddVWW, arg)

// 		if a.y != 0 && a.r < a.y {
// 			arg := argWVW{a.x, a.c, a.z, a.y, a.r}
// 			testFunWVW(t, "divWVW_g", divWVW_g, arg)
// 			testFunWVW(t, "divWVW", divWVW, arg)
// 		}
// 	}
// }

// var mulWWTests = []struct {
// 	x, y Word
// 	q, r Word
// }{
// 	{_M, _M, _M - 1, 1},
// 	// 32 bit only: {0xc47dfa8c, 50911, 0x98a4, 0x998587f4},
// }

// func TestMulWW(t *testing.T) {
// 	for i, test := range mulWWTests {
// 		q, r := mulWW_g(test.x, test.y)
// 		if q != test.q || r != test.r {
// 			t.Errorf("#%d got (%x, %x) want (%x, %x)", i, q, r, test.q, test.r)
// 		}
// 	}
// }

// var mulAddWWWTests = []struct {
// 	x, y, c Word
// 	q, r    Word
// }{
// 	// TODO(db47h): These will only work on 64-bit platforms.
// 	// {15064310297182388543, 0xe7df04d2d35d5d80, 13537600649892366549, 13644450054494335067, 10832252001440893781},
// 	// {15064310297182388543, 0xdab2f18048baa68d, 13644450054494335067, 12869334219691522700, 14233854684711418382},
// 	{_M, _M, 0, _M - 1, 1},
// 	{_M, _M, _M, _M, 0},
// }

// func TestMulAddWWW(t *testing.T) {
// 	for i, test := range mulAddWWWTests {
// 		q, r := mulAddWWW_g(test.x, test.y, test.c)
// 		if q != test.q || r != test.r {
// 			t.Errorf("#%d got (%x, %x) want (%x, %x)", i, q, r, test.q, test.r)
// 		}
// 	}
// }

// func BenchmarkMulAddVWW(b *testing.B) {
// 	for _, n := range benchSizes {
// 		if isRaceBuilder && n > 1e3 {
// 			continue
// 		}
// 		z := make([]Word, n+1)
// 		x := rndV(n)
// 		y := rndW()
// 		r := rndW()
// 		b.Run(fmt.Sprint(n), func(b *testing.B) {
// 			b.SetBytes(int64(n * _W))
// 			for i := 0; i < b.N; i++ {
// 				mulAddVWW(z, x, y, r)
// 			}
// 		})
// 	}
// }

// func BenchmarkAddMulVVW(b *testing.B) {
// 	for _, n := range benchSizes {
// 		if isRaceBuilder && n > 1e3 {
// 			continue
// 		}
// 		x := rndV(n)
// 		y := rndW()
// 		z := make([]Word, n)
// 		b.Run(fmt.Sprint(n), func(b *testing.B) {
// 			b.SetBytes(int64(n * _W))
// 			for i := 0; i < b.N; i++ {
// 				addMulVVW(z, x, y)
// 			}
// 		})
// 	}
// }
