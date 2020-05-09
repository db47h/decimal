// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

var decCmpTests = []struct {
	x, y dec
	r    int
}{
	{nil, nil, 0},
	{nil, dec(nil), 0},
	{dec(nil), nil, 0},
	{dec(nil), dec(nil), 0},
	{dec{0}, dec{0}, 0},
	{dec{0}, dec{1}, -1},
	{dec{1}, dec{0}, 1},
	{dec{1}, dec{1}, 0},
	{dec{0, _MD}, dec{1}, 1},
	{dec{1}, dec{0, _MD}, -1},
	{dec{1, _MD}, dec{0, _MD}, 1},
	{dec{0, _MD}, dec{1, _MD}, -1},
	{dec{16, 571956, 8794, 68}, dec{837, 9146, 1, 754489}, -1},
	{dec{34986, 41, 105, 1957}, dec{56, 7458, 104, 1957}, 1},
}

func TestDecCmp(t *testing.T) {
	for i, a := range decCmpTests {
		r := a.x.cmp(a.y)
		if r != a.r {
			t.Errorf("#%d got r = %v; want %v", i, r, a.r)
		}
	}
}

type decFunNN func(z, x, y dec) dec
type decArgNN struct {
	z, x, y dec
}

var decSumNN = []decArgNN{
	{},
	{dec{1}, nil, dec{1}},
	{dec{1111111110}, dec{123456789}, dec{987654321}},
	{dec{0, 0, 0, 1}, nil, dec{0, 0, 0, 1}},
	{dec{0, 0, 0, 1111111110}, dec{0, 0, 0, 123456789}, dec{0, 0, 0, 987654321}},
	{dec{0, 0, 0, 1}, dec{0, 0, _MD}, dec{0, 0, 1}},
}

var decProdNN = []decArgNN{
	{},
	{nil, nil, nil},
	{nil, dec{991}, nil},
	{dec{991}, dec{991}, dec{1}},
	{dec{991 * 991}, dec{991}, dec{991}},
	{dec{0, 0, 991 * 991}, dec{0, 991}, dec{0, 991}},
	{dec{1 * 991, 2 * 991, 3 * 991, 4 * 991}, dec{1, 2, 3, 4}, dec{991}},
	{dec{4, 11, 20, 30, 20, 11, 4}, dec{1, 2, 3, 4}, dec{4, 3, 2, 1}},
	// 3^100 * 3^28 = 3^128
	{
		decFromString("11790184577738583171520872861412518665678211592275841109096961"),
		decFromString("515377520732011331036461129765621272702107522001"),
		decFromString("22876792454961"),
	},
	// z = 111....1 (70000 digits)
	// x = 10^(99*700) + ... + 10^1400 + 10^700 + 1
	// y = 111....1 (700 digits, larger than Karatsuba threshold on 32-bit and 64-bit)
	{
		decFromString(strings.Repeat("1", 70000)),
		decFromString("1" + strings.Repeat(strings.Repeat("0", 699)+"1", 99)),
		decFromString(strings.Repeat("1", 700)),
	},
	// z = 111....1 (20000 digits)
	// x = 10^10000 + 1
	// y = 111....1 (10000 digits)
	{
		decFromString(strings.Repeat("1", 20000)),
		decFromString("1" + strings.Repeat("0", 9999) + "1"),
		decFromString(strings.Repeat("1", 10000)),
	},
}

func decFromString(s string) dec {
	x, _, _, err := dec(nil).scan(strings.NewReader(s), 0, false)
	if err != nil {
		panic(err)
	}
	return x
}

func TestDecSet(t *testing.T) {
	for _, a := range decSumNN {
		z := dec(nil).set(a.z)
		if z.cmp(a.z) != 0 {
			t.Errorf("got z = %v; want %v", z, a.z)
		}
	}
}

func decTestFunNN(t *testing.T, msg string, f decFunNN, a decArgNN) {
	z := f(nil, a.x, a.y)
	if z.cmp(a.z) != 0 {
		t.Errorf("%s%+v\n\tgot z = %v; want %v", msg, a, z, a.z)
	}
}

func TestDecFunNN(t *testing.T) {
	for _, a := range decSumNN {
		arg := a
		decTestFunNN(t, "add", dec.add, arg)

		arg = decArgNN{a.z, a.y, a.x}
		decTestFunNN(t, "add symmetric", dec.add, arg)

		arg = decArgNN{a.x, a.z, a.y}
		decTestFunNN(t, "sub", dec.sub, arg)

		arg = decArgNN{a.y, a.z, a.x}
		decTestFunNN(t, "sub symmetric", dec.sub, arg)
	}

	for _, a := range decProdNN {
		arg := a
		decTestFunNN(t, "mul", dec.mul, arg)

		arg = decArgNN{a.z, a.y, a.x}
		decTestFunNN(t, "mul symmetric", dec.mul, arg)
	}
}

// var mulRangesN = []struct {
// 	a, b uint64
// 	prod string
// }{
// 	{0, 0, "0"},
// 	{1, 1, "1"},
// 	{1, 2, "2"},
// 	{1, 3, "6"},
// 	{10, 10, "10"},
// 	{0, 100, "0"},
// 	{0, 1e9, "0"},
// 	{1, 0, "1"},                    // empty range
// 	{100, 1, "1"},                  // empty range
// 	{1, 10, "3628800"},             // 10!
// 	{1, 20, "2432902008176640000"}, // 20!
// 	{1, 100,
// 		"933262154439441526816992388562667004907159682643816214685929" +
// 			"638952175999932299156089414639761565182862536979208272237582" +
// 			"51185210916864000000000000000000000000", // 100!
// 	},
// }

// func TestMulRangeN(t *testing.T) {
// 	for i, r := range mulRangesN {
// 		prod := string(nat(nil).mulRange(r.a, r.b).utoa(10))
// 		if prod != r.prod {
// 			t.Errorf("#%d: got %s; want %s", i, prod, r.prod)
// 		}
// 	}
// }

// decAllocBytes returns the number of bytes allocated by invoking f.
func decAllocBytes(f func()) uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t := stats.TotalAlloc
	f()
	runtime.ReadMemStats(&stats)
	return stats.TotalAlloc - t
}

// TestDecMulUnbalanced tests that multiplying numbers of different lengths
// does not cause deep recursion and in turn allocate too much memory.
// Test case for issue 3807.
func TestDecMulUnbalanced(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	x := rndDec(50000)
	y := rndDec(40)
	allocSize := decAllocBytes(func() {
		dec(nil).mul(x, y)
	})
	inputSize := uint64(len(x)+len(y)) * _S
	if ratio := allocSize / uint64(inputSize); ratio > 10 {
		t.Errorf("multiplication uses too much memory (%d > %d times the size of inputs)", allocSize, ratio)
	}
}

// rndDec returns a random dec value >= 0 of (usually) n words in length.
// In extremely unlikely cases it may be smaller than n words if the top-
// most words are 0.
func rndDec(n int) dec {
	return dec(rnd10V(n)).norm()
}

// rndDec1 is like rndDec but the result is guaranteed to be > 0.
func rndDec1(n int) dec {
	x := dec(rnd10V(n)).norm()
	if len(x) == 0 {
		x.setWord(1)
	}
	return x
}

func BenchmarkDecMul1e4(b *testing.B) {
	mulx := rndDec(1e4)
	muly := rndDec(1e4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var z dec
		z.mul(mulx, muly)
	}
}

func benchmarkDecMul(b *testing.B, nwords int) {
	x := rndDec(nwords)
	y := rndDec(nwords)
	var z dec
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		z.mul(x, y)
	}
}

var decMulBenchSizes = []int{10, 100, 1000, 10000, 100000}

func BenchmarkDecMul(b *testing.B) {
	for _, n := range decMulBenchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			benchmarkDecMul(b, n)
		})
	}
}

func TestDecNLZ10(t *testing.T) {
	var x Word = _MD
	for i := 0; i <= _WD; i++ {
		if int(nlz10(x)) != i {
			t.Errorf("failed at %x: got %d want %d", x, nlz10(x), i)
		}
		x /= 10
	}
}

type decShiftTest struct {
	in    dec
	shift uint
	out   dec
}

var decLeftShiftTests = []decShiftTest{
	{nil, 0, nil},
	{nil, 1, nil},
	{decOne, 0, decOne},
	{decOne, 1, decTen},
	{dec{_BD / 10}, 1, dec{0}},
	{dec{_BD / 10, 0}, 1, dec{0, 1}},
}

func TestDecShiftLeft(t *testing.T) {
	for i, test := range decLeftShiftTests {
		var z dec
		z = z.shl(test.in, test.shift)
		for j, d := range test.out {
			if j >= len(z) || z[j] != d {
				t.Errorf("#%d: got: %v want: %v", i, z, test.out)
				break
			}
		}
	}
}

var decRightShiftTests = []decShiftTest{
	{nil, 0, nil},
	{nil, 1, nil},
	{decOne, 0, decOne},
	{decOne, 1, nil},
	{decTen, 1, decOne},
	{dec{0, 1}, 1, dec{_BD / 10}},
	{dec{10, 1, 1}, 1, dec{_BD/10 + 1, _BD / 10}},
}

func TestDecShiftRight(t *testing.T) {
	for i, test := range decRightShiftTests {
		var z dec
		z = z.shr(test.in, test.shift)
		for j, d := range test.out {
			if j >= len(z) || z[j] != d {
				t.Errorf("#%d: got: %v want: %v", i, z, test.out)
				break
			}
		}
	}
}

func BenchmarkDecZeroShifts(b *testing.B) {
	x := rndDec(800)

	b.Run("Shl", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var z dec
			z.shl(x, 0)
		}
	})
	b.Run("ShlSame", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x.shl(x, 0)
		}
	})

	b.Run("Shr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var z dec
			z.shr(x, 0)
		}
	})
	b.Run("ShrSame", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			x.shr(x, 0)
		}
	})
}

type decModWTest struct {
	in       string
	dividend string
	out      string
}

var decModWTests32 = []decModWTest{
	{"23492635982634928349238759823742", "252341", "220170"},
}

var decModWTests64 = []decModWTest{
	{"6527895462947293856291561095690465243862946", "524326975699234", "375066989628668"},
}

func runModWTests(t *testing.T, tests []decModWTest) {
	for i, test := range tests {
		in := decFromString(test.in)
		d := decFromString(test.dividend)
		out := decFromString(test.out)

		r := in.modW(d[0])
		if r != out[0] {
			t.Errorf("#%d failed: got %d want %s", i, r, out.utoa(10))
		}
	}
}

func TestDecModW(t *testing.T) {
	if _W >= 32 {
		runModWTests(t, decModWTests32)
	}
	if _W >= 64 {
		runModWTests(t, decModWTests64)
	}
}

// var montgomeryTests = []struct {
// 	x, y, m      string
// 	k0           uint64
// 	out32, out64 string
// }{
// 	{
// 		"0xffffffffffffffffffffffffffffffffffffffffffffffffe",
// 		"0xffffffffffffffffffffffffffffffffffffffffffffffffe",
// 		"0xfffffffffffffffffffffffffffffffffffffffffffffffff",
// 		1,
// 		"0x1000000000000000000000000000000000000000000",
// 		"0x10000000000000000000000000000000000",
// 	},
// 	{
// 		"0x000000000ffffff5",
// 		"0x000000000ffffff0",
// 		"0x0000000010000001",
// 		0xff0000000fffffff,
// 		"0x000000000bfffff4",
// 		"0x0000000003400001",
// 	},
// 	{
// 		"0x0000000080000000",
// 		"0x00000000ffffffff",
// 		"0x1000000000000001",
// 		0xfffffffffffffff,
// 		"0x0800000008000001",
// 		"0x0800000008000001",
// 	},
// 	{
// 		"0x0000000080000000",
// 		"0x0000000080000000",
// 		"0xffffffff00000001",
// 		0xfffffffeffffffff,
// 		"0xbfffffff40000001",
// 		"0xbfffffff40000001",
// 	},
// 	{
// 		"0x0000000080000000",
// 		"0x0000000080000000",
// 		"0x00ffffff00000001",
// 		0xfffffeffffffff,
// 		"0xbfffff40000001",
// 		"0xbfffff40000001",
// 	},
// 	{
// 		"0x0000000080000000",
// 		"0x0000000080000000",
// 		"0x0000ffff00000001",
// 		0xfffeffffffff,
// 		"0xbfff40000001",
// 		"0xbfff40000001",
// 	},
// 	{
// 		"0x3321ffffffffffffffffffffffffffff00000000000022222623333333332bbbb888c0",
// 		"0x3321ffffffffffffffffffffffffffff00000000000022222623333333332bbbb888c0",
// 		"0x33377fffffffffffffffffffffffffffffffffffffffffffff0000000000022222eee1",
// 		0xdecc8f1249812adf,
// 		"0x04eb0e11d72329dc0915f86784820fc403275bf2f6620a20e0dd344c5cd0875e50deb5",
// 		"0x0d7144739a7d8e11d72329dc0915f86784820fc403275bf2f61ed96f35dd34dbb3d6a0",
// 	},
// 	{
// 		"0x10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffffffffffffffffffffffffffffff00000000000022222223333333333444444444",
// 		"0x10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffffffffffffffffffffffffffffff999999999999999aaabbbbbbbbcccccccccccc",
// 		"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff33377fffffffffffffffffffffffffffffffffffffffffffff0000000000022222eee1",
// 		0xdecc8f1249812adf,
// 		"0x5c0d52f451aec609b15da8e5e5626c4eaa88723bdeac9d25ca9b961269400410ca208a16af9c2fb07d7a11c7772cba02c22f9711078d51a3797eb18e691295293284d988e349fa6deba46b25a4ecd9f715",
// 		"0x92fcad4b5c0d52f451aec609b15da8e5e5626c4eaa88723bdeac9d25ca9b961269400410ca208a16af9c2fb07d799c32fe2f3cc5422f9711078d51a3797eb18e691295293284d8f5e69caf6decddfe1df6",
// 	},
// }

// func TestMontgomery(t *testing.T) {
// 	one := NewInt(1)
// 	_B := new(Int).Lsh(one, _W)
// 	for i, test := range montgomeryTests {
// 		x := decFromString(test.x)
// 		y := decFromString(test.y)
// 		m := decFromString(test.m)
// 		for len(x) < len(m) {
// 			x = append(x, 0)
// 		}
// 		for len(y) < len(m) {
// 			y = append(y, 0)
// 		}

// 		if x.cmp(m) > 0 {
// 			_, r := nat(nil).div(nil, x, m)
// 			t.Errorf("#%d: x > m (0x%s > 0x%s; use 0x%s)", i, x.utoa(16), m.utoa(16), r.utoa(16))
// 		}
// 		if y.cmp(m) > 0 {
// 			_, r := nat(nil).div(nil, x, m)
// 			t.Errorf("#%d: y > m (0x%s > 0x%s; use 0x%s)", i, y.utoa(16), m.utoa(16), r.utoa(16))
// 		}

// 		var out nat
// 		if _W == 32 {
// 			out = decFromString(test.out32)
// 		} else {
// 			out = decFromString(test.out64)
// 		}

// 		// t.Logf("#%d: len=%d\n", i, len(m))

// 		// check output in table
// 		xi := &Int{abs: x}
// 		yi := &Int{abs: y}
// 		mi := &Int{abs: m}
// 		p := new(Int).Mod(new(Int).Mul(xi, new(Int).Mul(yi, new(Int).ModInverse(new(Int).Lsh(one, uint(len(m))*_W), mi))), mi)
// 		if out.cmp(p.abs.norm()) != 0 {
// 			t.Errorf("#%d: out in table=0x%s, computed=0x%s", i, out.utoa(16), p.abs.norm().utoa(16))
// 		}

// 		// check k0 in table
// 		k := new(Int).Mod(&Int{abs: m}, _B)
// 		k = new(Int).Sub(_B, k)
// 		k = new(Int).Mod(k, _B)
// 		k0 := Word(new(Int).ModInverse(k, _B).Uint64())
// 		if k0 != Word(test.k0) {
// 			t.Errorf("#%d: k0 in table=%#x, computed=%#x\n", i, test.k0, k0)
// 		}

// 		// check montgomery with correct k0 produces correct output
// 		z := nat(nil).montgomery(x, y, m, k0, len(m))
// 		z = z.norm()
// 		if z.cmp(out) != 0 {
// 			t.Errorf("#%d: got 0x%s want 0x%s", i, z.utoa(16), out.utoa(16))
// 		}
// 	}
// }

var decExpNNTests = []struct {
	x, y, m string
	out     string
}{
	{"0", "0", "0", "1"},
	{"0", "0", "1", "0"},
	{"1", "1", "1", "0"},
	{"2", "1", "1", "0"},
	{"2", "2", "1", "0"},
	{"10", "100000000000", "1", "0"},
	{"0x8000000000000000", "2", "", "0x40000000000000000000000000000000"},
	{"0x8000000000000000", "2", "6719", "4944"},
	{"0x8000000000000000", "3", "6719", "5447"},
	{"0x8000000000000000", "1000", "6719", "1603"},
	{"0x8000000000000000", "1000000", "6719", "3199"},
	{
		"2938462938472983472983659726349017249287491026512746239764525612965293865296239471239874193284792387498274256129746192347",
		"298472983472983471903246121093472394872319615612417471234712061",
		"29834729834729834729347290846729561262544958723956495615629569234729836259263598127342374289365912465901365498236492183464",
		"23537740700184054162508175125554701713153216681790245129157191391322321508055833908509185839069455749219131480588829346291",
	},
	{
		"11521922904531591643048817447554701904414021819823889996244743037378330903763518501116638828335352811871131385129455853417360623007349090150042001944696604737499160174391019030572483602867266711107136838523916077674888297896995042968746762200926853379",
		"426343618817810911523",
		"444747819283133684179",
		"42",
	},
}

func TestDecExpNN(t *testing.T) {
	for i, test := range decExpNNTests {
		x := decFromString(test.x)
		y := decFromString(test.y)
		out := decFromString(test.out)

		var m dec
		if len(test.m) > 0 {
			m = decFromString(test.m)
		}

		z := dec(nil).expNN(x, y, m)
		if z.cmp(out) != 0 {
			t.Errorf("#%d got %s want %s", i, z.utoa(10), out.utoa(10))
		}
	}
}

func BenchmarkDecExpNN(b *testing.B) {
	for i, test := range decExpNNTests {
		x := decFromString(test.x)
		y := decFromString(test.y)
		out := decFromString(test.out)

		var m dec
		if len(test.m) > 0 {
			m = decFromString(test.m)
		}

		b.Run(fmt.Sprintf("#%d", i), func(b *testing.B) {
			var z dec
			for it := 0; it < b.N; it++ {
				z = z.expNN(x, y, m)
				if z.cmp(out) != 0 {
					b.Fatalf("#%d got %s want %s", i, z.utoa(10), out.utoa(10))
				}
			}
		})
	}
}

func BenchmarkDecExp3Power(b *testing.B) {
	const x = 3
	for _, y := range []Word{
		0x10, 0x40, 0x100, 0x400, 0x1000, 0x4000, 0x10000, 0x40000, 0x100000, 0x400000,
	} {
		b.Run(fmt.Sprintf("%#x", y), func(b *testing.B) {
			var z dec
			for i := 0; i < b.N; i++ {
				z.expWW(x, y)
			}
		})
	}
}

func decFibo(n int) dec {
	switch n {
	case 0:
		return nil
	case 1:
		return dec{1}
	}
	f0 := decFibo(0)
	f1 := decFibo(1)
	var f2 dec
	for i := 1; i < n; i++ {
		f2 = f2.add(f0, f1)
		f0, f1, f2 = f1, f2, f0
	}
	return f1
}

var decFiboNums = []string{
	"0",
	"55",
	"6765",
	"832040",
	"102334155",
	"12586269025",
	"1548008755920",
	"190392490709135",
	"23416728348467685",
	"2880067194370816120",
	"354224848179261915075",
}

func TestDecFibo(t *testing.T) {
	for i, want := range decFiboNums {
		n := i * 10
		got := string(decFibo(n).utoa(10))
		if got != want {
			t.Errorf("fibo(%d) failed: got %s want %s", n, got, want)
		}
	}
}

func BenchmarkFibo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decFibo(1e0)
		decFibo(1e1)
		decFibo(1e2)
		decFibo(1e3)
		decFibo(1e4)
		decFibo(1e5)
	}
}

var decDigitTests = []struct {
	x    string
	i    uint
	want uint
}{
	{"0", 0, 0},
	{"0", 1, 0},
	{"0", 1000, 0},

	{"1", 0, 1},
	{"10000", 0, 0},
	{"10000", 3, 0},
	{"10000", 4, 1},
	{"10000", 5, 0},

	{"100000000000000000", 16, 0},
	{"100000000000000000", 17, 1},
	{"100000000000000000", 18, 0},

	{"37" + strings.Repeat("0", 32), 31, 0},
	{"37" + strings.Repeat("0", 32), 32, 7},
	{"37" + strings.Repeat("0", 32), 33, 3},
	{"37" + strings.Repeat("0", 32), 34, 0},
}

func TestDecDigit(t *testing.T) {
	for i, test := range decDigitTests {
		x := decFromString(test.x)
		if got := x.digit(test.i); got != test.want {
			t.Errorf("#%d: %s.bit(%d) = %v; want %v", i, test.x, test.i, got, test.want)
		}
	}
}

var decStickyTests = []struct {
	x    string
	i    uint
	want uint
}{
	{"0", 0, 0},
	{"0", 1, 0},
	{"0", 1000, 0},

	{"1", 0, 0},
	{"1", 1, 1},

	{"1001101010000", 0, 0},
	{"1001101010000", 4, 0},
	{"1001101010000", 5, 1},

	{"1000000000000000000", 18, 0},
	{"1000000000000000000", 29, 1},

	{"1" + strings.Repeat("0", 100), 100, 0},
	{"1" + strings.Repeat("0", 100), 101, 1},
}

func TestDecSticky(t *testing.T) {
	for i, test := range decStickyTests {
		x := decFromString(test.x)
		if got := x.sticky(test.i); got != test.want {
			t.Errorf("#%d: %s.sticky(%d) = %v; want %v", i, test.x, test.i, got, test.want)
		}
		if test.want == 1 {
			// all subsequent i's should also return 1
			for d := uint(1); d <= 3; d++ {
				if got := x.sticky(test.i + d); got != 1 {
					t.Errorf("#%d: %s.sticky(%d) = %v; want %v", i, test.x, test.i+d, got, 1)
				}
			}
		}
	}
}

func testDecSqr(t *testing.T, x dec) {
	got := make(dec, 2*len(x))
	want := make(dec, 2*len(x))
	got = got.sqr(x)
	want = want.mul(x, x)
	if got.cmp(want) != 0 {
		t.Errorf("basicSqr(%v), got %v, want %v", x, got, want)
	}
}

func TestDecSqr(t *testing.T) {
	for _, a := range decProdNN {
		if a.x != nil {
			testDecSqr(t, a.x)
		}
		if a.y != nil {
			testDecSqr(t, a.y)
		}
		if a.z != nil {
			testDecSqr(t, a.z)
		}
	}
}

func benchmarkDecSqr(b *testing.B, nwords int) {
	x := rndDec(nwords)
	var z dec
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		z.sqr(x)
	}
}

var decSqrBenchSizes = []int{
	1, 2, 3, 5, 8, 10, 20, 30, 50, 80,
	100, 200, 300, 500, 800,
	1000, 10000, 100000,
}

func BenchmarkDecSqr(b *testing.B) {
	for _, n := range decSqrBenchSizes {
		if isRaceBuilder && n > 1e3 {
			continue
		}
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			benchmarkDecSqr(b, n)
		})
	}
}

// func BenchmarkNatSetBytes(b *testing.B) {
// 	const maxLength = 128
// 	lengths := []int{
// 		// No remainder:
// 		8, 24, maxLength,
// 		// With remainder:
// 		7, 23, maxLength - 1,
// 	}
// 	n := make(nat, maxLength/_W) // ensure n doesn't need to grow during the test
// 	buf := make([]byte, maxLength)
// 	for _, l := range lengths {
// 		b.Run(fmt.Sprint(l), func(b *testing.B) {
// 			for i := 0; i < b.N; i++ {
// 				n.setBytes(buf[:l])
// 			}
// 		})
// 	}
// }

func TestDecDiv(t *testing.T) {
	sizes := []int{
		1, 2, 5, 8, 15, 25, 40, 65, 100,
		200, 500, 800, 1500, 2500, 4000, 6500, 10000,
	}
	for _, i := range sizes {
		for _, j := range sizes {
			a := rndDec1(i)
			b := rndDec1(j)
			// the test requires b >= 2
			if len(b) == 1 && b[0] == 1 {
				b[0] = 2
			}
			// choose a remainder c < b
			c := rndDec1(len(b))
			if len(c) == len(b) && c[len(c)-1] >= b[len(b)-1] {
				c[len(c)-1] = 0
				c = c.norm()
			}
			// compute x = a*b+c
			x := dec(nil).mul(a, b)
			x = x.add(x, c)

			var q, r dec
			q, r = q.div(r, x, b)
			if q.cmp(a) != 0 {
				t.Fatalf("wrong quotient: got %s; want %s for %s/%s", q.utoa(10), a.utoa(10), x.utoa(10), b.utoa(10))
			}
			if r.cmp(c) != 0 {
				t.Fatalf("wrong remainder: got %s; want %s for %s/%s", r.utoa(10), c.utoa(10), x.utoa(10), b.utoa(10))
			}
		}
	}
}

// TODO(bd47h): move this to decimal_test
func benchmarkDiv(b *testing.B, aSize, bSize int) {
	aa := rndDec1(aSize)
	// bb := rndDec1(bSize)
	bb := dec(nil).setWord(Word(rnd.Intn(_W-1)) + 1)
	if aa.cmp(bb) < 0 {
		aa, bb = bb, aa
	}
	x := dec(nil)
	y := dec(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.div(y, aa, bb)
	}
}

func BenchmarkDiv(b *testing.B) {
	sizes := []int{
		10, 20, 50, 100, 200, 500, 1000,
		// 1e4, 1e5, 1e6, 1e7,
	}
	for _, i := range sizes {
		j := 2 * i
		b.Run(fmt.Sprintf("%d/%d", j, i), func(b *testing.B) {
			benchmarkDiv(b, j, i)
		})
	}
}
