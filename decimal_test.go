// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"encoding"
	"encoding/gob"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkDecimal_dnorm(b *testing.B) {
	d := dec(nil).make(1000)
	for i := range d {
		d[i] = Word(rand.Uint64()) % _DB
	}
	for i := 0; i < b.N; i++ {
		d[0] = Word(rand.Uint64()) % _DB
		d[len(d)-1] = Word(rand.Uint64()) % _DB
		_ = uint(dnorm(d))
	}
}
func BenchmarkDecimal_Sqrt(b *testing.B) {
	x := new(Decimal).SetUint64(2)
	z := new(Decimal).SetPrec(34)
	for i := 0; i < b.N; i++ {
		z.Sqrt(x)
	}
}

func BenchmarkDecimal_Float(b *testing.B) {
	d := new(Decimal).SetPrec(100).SetUint64(2)
	d.Sqrt(d)
	f := d.Float(nil)
	for i := 0; i < b.N; i++ {
		d.Float(f)
	}
}

func TestDecimal_FMA(t *testing.T) {
	x := NewDecimal(123, -2).SetPrec(3)
	y := NewDecimal(227, -2).SetPrec(3)
	u := NewDecimal(3, -3).SetPrec(3)
	z := new(Decimal).SetPrec(3).Mul(x, y) // == 2.7921
	z.Add(z, u)
	if s := z.String(); s != "2.79" {
		t.Fatalf("Precision 3, 2.79 + 0.003 = %s, want 2.79", s)
	}
	z = z.FMA(x, y, u)
	if s := z.String(); s != "2.8" {
		t.Fatalf("Precision 3, %v * %v + 0.003 = %s, want 2.8", x, y, s)
	}
	// test aliasing z and u
	u.FMA(x, y, u)
	if s := z.String(); s != "2.8" {
		t.Fatalf("Aliasing z & u, %v * %v + 0.003 = %s, want 2.8", x, y, s)
	}
}

var decimalZero Decimal

var (
	// required implemented interfaces
	_ fmt.Stringer             = &decimalZero
	_ fmt.Scanner              = &decimalZero
	_ fmt.Formatter            = &decimalZero
	_ encoding.TextMarshaler   = &decimalZero
	_ encoding.TextUnmarshaler = &decimalZero
	_ gob.GobEncoder           = &decimalZero
	_ gob.GobDecoder           = &decimalZero
)

// Verify that ErrNaN implements the error interface.
var _ error = ErrNaN{}

func (x *Decimal) uint64() uint64 {
	u, acc := x.Uint64()
	if acc != Exact {
		panic(fmt.Sprintf("%s is not a uint64", x.Text('g', 10)))
	}
	return u
}

func (x *Decimal) int64() int64 {
	i, acc := x.Int64()
	if acc != Exact {
		panic(fmt.Sprintf("%s is not an int64", x.Text('g', 10)))
	}
	return i
}

func TestDecimalZeroValue(t *testing.T) {
	// zero (uninitialized) value is a ready-to-use 0.0
	var x Decimal
	if s := x.Text('f', 1); s != "0.0" {
		t.Errorf("zero value = %s; want 0.0", s)
	}

	// zero value has precision 0
	if prec := x.Prec(); prec != 0 {
		t.Errorf("prec = %d; want 0", prec)
	}

	// zero value can be used in any and all positions of binary operations
	make := func(x int) *Decimal {
		var f Decimal
		if x != 0 {
			f.SetInt64(int64(x))
		}
		// x == 0 translates into the zero value
		return &f
	}
	for _, test := range []struct {
		z, x, y, want int
		opname        rune
		op            func(z, x, y *Decimal) *Decimal
	}{
		{0, 0, 0, 0, '+', (*Decimal).Add},
		{0, 1, 2, 3, '+', (*Decimal).Add},
		{1, 2, 0, 2, '+', (*Decimal).Add},
		{2, 0, 1, 1, '+', (*Decimal).Add},

		{0, 0, 0, 0, '-', (*Decimal).Sub},
		{0, 1, 2, -1, '-', (*Decimal).Sub},
		{1, 2, 0, 2, '-', (*Decimal).Sub},
		{2, 0, 1, -1, '-', (*Decimal).Sub},

		{0, 0, 0, 0, '*', (*Decimal).Mul},
		{0, 1, 2, 2, '*', (*Decimal).Mul},
		{1, 2, 0, 0, '*', (*Decimal).Mul},
		{2, 0, 1, 0, '*', (*Decimal).Mul},

		// {0, 0, 0, 0, '/', (*Decimal).Quo}, // panics
		{0, 2, 1, 2, '/', (*Decimal).Quo},
		{1, 2, 0, 0, '/', (*Decimal).Quo}, // = +Inf
		{2, 0, 1, 0, '/', (*Decimal).Quo},
	} {
		z := make(test.z)
		test.op(z, make(test.x), make(test.y))
		got := 0
		if !z.IsInf() {
			got = int(z.int64())
		}
		if got != test.want {
			t.Errorf("%d %c %d = %d; want %d", test.x, test.opname, test.y, got, test.want)
		}
	}

	// TODO(db47h) test how precision is set for zero value results
}

func makeDecimal(s string) *Decimal {
	x, _, err := ParseDecimal(s, 0, 350, ToNearestEven)
	if err != nil {
		panic(err)
	}
	return x
}

func TestDecimalSetPrec(t *testing.T) {
	for _, test := range []struct {
		x    string
		prec uint
		want string
		acc  Accuracy
	}{
		// prec 0
		{"0", 0, "0", Exact},
		{"-0", 0, "-0", Exact},
		{"-Inf", 0, "-Inf", Exact},
		{"+Inf", 0, "+Inf", Exact},
		{"123", 0, "0", Below},
		{"-123", 0, "-0", Above},

		// prec at upper limit
		{"0", MaxPrec, "0", Exact},
		{"-0", MaxPrec, "-0", Exact},
		{"-Inf", MaxPrec, "-Inf", Exact},
		{"+Inf", MaxPrec, "+Inf", Exact},

		// just a few regular cases - general rounding is tested elsewhere
		{"1.5", 1, "2", Above},
		{"-1.5", 1, "-2", Below},
		{"123", 1e6, "123", Exact},
		{"-123", 1e6, "-123", Exact},
	} {
		x := makeDecimal(test.x).SetPrec(test.prec)
		prec := test.prec
		if prec > MaxPrec {
			prec = MaxPrec
		}
		if got := x.Prec(); got != prec {
			t.Errorf("%s.SetPrec(%d).Prec() == %d; want %d", test.x, test.prec, got, prec)
		}
		if got, acc := x.String(), x.Acc(); got != test.want || acc != test.acc {
			t.Errorf("%s.SetPrec(%d) = %s (%s); want %s (%s)", test.x, test.prec, got, acc, test.want, test.acc)
		}
	}
}

func TestDecimalMinPrec(t *testing.T) {
	const max = 30
	for _, test := range []struct {
		x    string
		want uint
	}{
		{"0", 0},
		{"-0", 0},
		{"+Inf", 0},
		{"-Inf", 0},
		{"1", 1},
		{"9", 1},
		{"999", 3},
		{"123456", 6},
		{"123456e-1000", 6},
		{"123456e+1000", 6},
		// {"0.1", max},
	} {
		x := makeDecimal(test.x).SetPrec(max)
		if got := x.MinPrec(); got != test.want {
			t.Errorf("%s.MinPrec() = %d; want %d", test.x, got, test.want)
		}
	}
}

func TestDecimalSign(t *testing.T) {
	for _, test := range []struct {
		x string
		s int
	}{
		{"-Inf", -1},
		{"-1", -1},
		{"-0", 0},
		{"+0", 0},
		{"+1", +1},
		{"+Inf", +1},
	} {
		x := makeDecimal(test.x)
		s := x.Sign()
		if s != test.s {
			t.Errorf("%s.Sign() = %d; want %d", test.x, s, test.s)
		}
	}
}

// alike(x, y) is like x.Cmp(y) == 0 but also considers the sign of 0 (0 != -0).
func alike(x, y *Decimal) bool {
	return x.Cmp(y) == 0 && x.Signbit() == y.Signbit()
}

func alike32(x, y float32) bool {
	// we can ignore NaNs
	return x == y && math.Signbit(float64(x)) == math.Signbit(float64(y))

}

func alike64(x, y float64) bool {
	// we can ignore NaNs
	return x == y && math.Signbit(x) == math.Signbit(y)

}

func TestDecimalMantExp(t *testing.T) {
	for _, test := range []struct {
		x    string
		mant string
		exp  int
	}{
		{"0", "0", 0},
		{"+0", "0", 0},
		{"-0", "-0", 0},
		{"Inf", "+Inf", 0},
		{"+Inf", "+Inf", 0},
		{"-Inf", "-Inf", 0},
		{"1.5", "0.15", 1},
		{"1.024e3", "0.1024", 4},
		{"-0.00125", "-0.125", -2},
	} {
		x := makeDecimal(test.x)
		mant := makeDecimal(test.mant)
		m := new(Decimal)
		e := x.MantExp(m)
		if !alike(m, mant) || e != test.exp {
			t.Errorf("%s.MantExp() = %s, %d; want %s, %d", test.x, m.Text('g', 10), e, test.mant, test.exp)
		}
	}
}

func TestDecimalMantExpAliasing(t *testing.T) {
	x := makeDecimal("0.5e10")
	if e := x.MantExp(x); e != 10 {
		t.Fatalf("Decimal.MantExp aliasing error: got %d; want 10", e)
	}
	if want := makeDecimal("0.5"); !alike(x, want) {
		t.Fatalf("Decimal.MantExp aliasing error: got %s; want %s", x.Text('g', 10), want.Text('g', 10))
	}
}

func TestDecimalSetMantExp(t *testing.T) {
	for _, test := range []struct {
		frac string
		exp  int
		z    string
	}{
		{"0", 0, "0"},
		{"+0", 0, "0"},
		{"-0", 0, "-0"},
		{"Inf", 1234, "+Inf"},
		{"+Inf", -1234, "+Inf"},
		{"-Inf", -1234, "-Inf"},
		{"0", MinExp, "0"},
		{"0.01", MinExp, "+0"},     // exponent underflow
		{"-0.01", MinExp, "-0"},    // exponent underflow
		{"1", MaxExp, "+Inf"},      // exponent overflow
		{"10", MaxExp - 1, "+Inf"}, // exponent overflow
		{"1", MaxExp - 1, "1e2147483646"},
		{"0.75", 1, "7.5"},
		{"0.5", 4, "5000"},
		{"-0.5", -2, "-0.005"},
		{"32", 2, "3200"},
		{"1024", -5, "0.01024"},
	} {
		frac := makeDecimal(test.frac)
		want := makeDecimal(test.z)
		var z Decimal
		z.SetMantExp(frac, test.exp)
		if !alike(&z, want) {
			t.Errorf("SetMantExp(%s, %d) = %s; want %s", test.frac, test.exp, z.Text('g', 10), test.z)
		}
		// test inverse property
		mant := new(Decimal)
		if z.SetMantExp(mant, want.MantExp(mant)).Cmp(want) != 0 {
			t.Errorf("Inverse property not satisfied: got %s; want %s", z.Text('g', 10), test.z)
		}
	}
}

func TestDecimalPredicates(t *testing.T) {
	for _, test := range []struct {
		x            string
		sign         int
		signbit, inf bool
	}{
		{x: "-Inf", sign: -1, signbit: true, inf: true},
		{x: "-1", sign: -1, signbit: true},
		{x: "-0", signbit: true},
		{x: "0"},
		{x: "1", sign: 1},
		{x: "+Inf", sign: 1, inf: true},
	} {
		x := makeDecimal(test.x)
		if got := x.Signbit(); got != test.signbit {
			t.Errorf("(%s).Signbit() = %v; want %v", test.x, got, test.signbit)
		}
		if got := x.Sign(); got != test.sign {
			t.Errorf("(%s).Sign() = %d; want %d", test.x, got, test.sign)
		}
		if got := x.IsInf(); got != test.inf {
			t.Errorf("(%s).IsInf() = %v; want %v", test.x, got, test.inf)
		}
	}
}

func TestDecimalIsInt(t *testing.T) {
	for _, test := range []string{
		"0 int",
		"-0 int",
		"1 int",
		"-1 int",
		"0.5",
		"1.23",
		"1.23e1",
		"1.23e2 int",
		"0.000000001e+8",
		"0.000000001e+9 int",
		"1.2345e200 int",
		"Inf",
		"+Inf",
		"-Inf",
	} {
		s := strings.TrimSuffix(test, " int")
		want := s != test
		if got := makeDecimal(s).IsInt(); got != want {
			t.Errorf("%s.IsInt() == %t", s, got)
		}
	}
}

func testDecimalRound(t *testing.T, x, r int64, prec uint, mode RoundingMode) {
	// verify test data
	var ok bool
	switch mode {
	case ToNearestEven, ToNearestAway:
		ok = true // nothing to do for now
	case ToZero:
		if x < 0 {
			ok = r >= x
		} else {
			ok = r <= x
		}
	case AwayFromZero:
		if x < 0 {
			ok = r <= x
		} else {
			ok = r >= x
		}
	case ToNegativeInf:
		ok = r <= x
	case ToPositiveInf:
		ok = r >= x
	default:
		panic("unreachable")
	}
	if !ok {
		t.Fatalf("incorrect test data for prec = %d, %s: x = %d, r = %d", prec, mode, x, r)
	}

	// compute expected accuracy
	a := Exact
	switch {
	case r < x:
		a = Below
	case r > x:
		a = Above
	}

	// round
	f := new(Decimal).SetMode(mode).SetInt64(x).SetPrec(prec)

	// check result
	r1 := f.int64()
	p1 := f.Prec()
	a1 := f.Acc()
	if r1 != r || p1 != prec || a1 != a {
		t.Errorf("round %d (%d digits, %s) incorrect: got %d (%d digits, %s); want %d (%d digits, %s)",
			x, prec, mode,
			r1, p1, a1,
			r, prec, a)
		return
	}

	// g and f should be the same
	// (rounding by SetPrec after SetInt64 using default precision
	// should be the same as rounding by SetInt64 after setting the
	// precision)
	g := new(Decimal).SetMode(mode).SetPrec(prec).SetInt64(x)
	if !alike(g, f) {
		t.Errorf("round %d (%d digits, %s) not symmetric: got %d and %d; want %d",
			x, prec, mode,
			g.int64(),
			r1,
			r,
		)
		return
	}

	// h and f should be the same
	// (repeated rounding should be idempotent)
	h := new(Decimal).SetMode(mode).SetPrec(prec).Set(f)
	if !alike(h, f) {
		t.Errorf("round %d (%d digits, %s) not idempotent: got %d and %d; want %d",
			x, prec, mode,
			h.int64(),
			r1,
			r,
		)
		return
	}
}

func fromBase10(s string) int64 {
	x, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return x
}

// TestDecimalRound tests basic rounding.
func TestDecimalRound(t *testing.T) {
	for _, test := range []struct {
		prec                        uint
		x, zero, neven, naway, away string // input, results rounded to prec bits
	}{
		{5, "5000", "5000", "5000", "5000", "5000"},
		{5, "5005", "5005", "5005", "5005", "5005"},
		{5, "5050", "5050", "5050", "5050", "5050"},
		{5, "5055", "5055", "5055", "5055", "5055"},
		{5, "5500", "5500", "5500", "5500", "5500"},
		{5, "5505", "5505", "5505", "5505", "5505"},
		{5, "5550", "5550", "5550", "5550", "5550"},
		{5, "5555", "5555", "5555", "5555", "5555"},

		{4, "5000", "5000", "5000", "5000", "5000"},
		{4, "5005", "5005", "5005", "5005", "5005"},
		{4, "5050", "5050", "5050", "5050", "5050"},
		{4, "5055", "5055", "5055", "5055", "5055"},
		{4, "5500", "5500", "5500", "5500", "5500"},
		{4, "5505", "5505", "5505", "5505", "5505"},
		{4, "5550", "5550", "5550", "5550", "5550"},
		{4, "5555", "5555", "5555", "5555", "5555"},

		{3, "5000", "5000", "5000", "5000", "5000"},
		{3, "5005", "5000", "5000", "5010", "5010"},
		{3, "5050", "5050", "5050", "5050", "5050"},
		{3, "5055", "5050", "5060", "5060", "5060"},
		{3, "5500", "5500", "5500", "5500", "5500"},
		{3, "5505", "5500", "5500", "5510", "5510"},
		{3, "5550", "5550", "5550", "5550", "5550"},
		{3, "9995", "9990", "10000", "10000", "10000"},

		{3, "5000005", "5000000", "5000000", "5000000", "5010000"},
		{3, "5005005", "5000000", "5010000", "5010000", "5010000"},
		{3, "5050005", "5050000", "5050000", "5050000", "5060000"},
		{3, "5055005", "5050000", "5060000", "5060000", "5060000"},
		{3, "5500005", "5500000", "5500000", "5500000", "5510000"},
		{3, "5505005", "5500000", "5510000", "5510000", "5510000"},
		{3, "9990005", "9990000", "9990000", "9990000", "10000000"},
		{3, "9995005", "9990000", "10000000", "10000000", "10000000"},

		{2, "5000", "5000", "5000", "5000", "5000"},
		{2, "5005", "5000", "5000", "5000", "5100"},
		{2, "5050", "5000", "5000", "5100", "5100"},
		{2, "5055", "5000", "5100", "5100", "5100"},
		{2, "5500", "5500", "5500", "5500", "5500"},
		{2, "9905", "9900", "9900", "9900", "10000"},
		{2, "9950", "9900", "10000", "10000", "10000"},
		{2, "9955", "9900", "10000", "10000", "10000"},

		{2, "5000005", "5000000", "5000000", "5000000", "5100000"},
		{2, "5005005", "5000000", "5000000", "5000000", "5100000"},
		{2, "5050005", "5000000", "5100000", "5100000", "5100000"},
		{2, "5055005", "5000000", "5100000", "5100000", "5100000"},
		{2, "9900005", "9900000", "9900000", "9900000", "10000000"},
		{2, "9905005", "9900000", "9900000", "9900000", "10000000"},
		{2, "9950005", "9900000", "10000000", "10000000", "10000000"},
		{2, "9955005", "9900000", "10000000", "10000000", "10000000"},

		{1, "9000", "9000", "9000", "9000", "9000"},
		{1, "9005", "9000", "9000", "9000", "10000"},
		{1, "9050", "9000", "9000", "9000", "10000"},
		{1, "9055", "9000", "9000", "9000", "10000"},
		{1, "9500", "9000", "10000", "10000", "10000"},
		{1, "9505", "9000", "10000", "10000", "10000"},
		{1, "9550", "9000", "10000", "10000", "10000"},
		{1, "9555", "9000", "10000", "10000", "10000"},

		{1, "9000005", "9000000", "9000000", "9000000", "10000000"},
		{1, "9005005", "9000000", "9000000", "9000000", "10000000"},
		{1, "9050005", "9000000", "9000000", "9000000", "10000000"},
		{1, "9055005", "9000000", "9000000", "9000000", "10000000"},
		{1, "9500005", "9000000", "10000000", "10000000", "10000000"},
		{1, "9505005", "9000000", "10000000", "10000000", "10000000"},
		{1, "9550005", "9000000", "10000000", "10000000", "10000000"},
		{1, "9555005", "9000000", "10000000", "10000000", "10000000"},
	} {
		x := fromBase10(test.x)
		z := fromBase10(test.zero)
		e := fromBase10(test.neven)
		n := fromBase10(test.naway)
		a := fromBase10(test.away)
		prec := test.prec

		testDecimalRound(t, x, z, prec, ToZero)
		testDecimalRound(t, x, e, prec, ToNearestEven)
		testDecimalRound(t, x, n, prec, ToNearestAway)
		testDecimalRound(t, x, a, prec, AwayFromZero)

		testDecimalRound(t, x, z, prec, ToNegativeInf)
		testDecimalRound(t, x, a, prec, ToPositiveInf)

		testDecimalRound(t, -x, -a, prec, ToNegativeInf)
		testDecimalRound(t, -x, -z, prec, ToPositiveInf)
	}
}

// TestFloatRound24 tests that rounding a float64 to 24 bits
// matches IEEE-754 rounding to nearest when converting a
// float64 to a float32 (excluding denormal numbers).
// TODO(db47h): implement this for decimal64/128
// func TestFloatRound24(t *testing.T) {
// 	const x0 = 1<<26 - 0x10 // 11...110000 (26 bits)
// 	for d := 0; d <= 0x10; d++ {
// 		x := float64(x0 + d)
// 		f := new(Float).SetPrec(24).SetFloat64(x)
// 		got, _ := f.Float32()
// 		want := float32(x)
// 		if got != want {
// 			t.Errorf("Round(%g, 24) = %g; want %g", x, got, want)
// 		}
// 	}
// }

func TestDecimalSetUint64(t *testing.T) {
	for _, want := range []uint64{
		0,
		1,
		2,
		10,
		100,
		1<<32 - 1,
		1 << 32,
		1<<64 - 1,
	} {
		var f Decimal
		f.SetUint64(want)
		if got := f.uint64(); got != want {
			t.Errorf("got %#x (%s); want %#x", got, f.Text('p', 0), want)
		}
	}

	// test basic rounding behavior (exhaustive rounding testing is done elsewhere)
	const x uint64 = 0x8765432187654321 // 64 bits needed
	mp := decDigits64(uint64(x))
	for prec := uint(1); prec <= mp; prec++ {
		f := new(Decimal).SetPrec(prec).SetMode(ToZero).SetUint64(x)
		got := f.uint64()
		want := x - x%uint64(pow10(mp-prec)) // cut off (round to zero) low prec digits
		if got != want {
			t.Errorf("got %#x (%s); want %#x", got, f.Text('p', 0), want)
		}
	}
}

func TestDecimalSetInt64(t *testing.T) {
	for _, want := range []int64{
		0,
		1,
		2,
		10,
		100,
		1<<32 - 1,
		1 << 32,
		1<<63 - 1,
	} {
		for i := range [2]int{} {
			if i&1 != 0 {
				want = -want
			}
			var f Decimal
			f.SetInt64(want)
			if got := f.int64(); got != want {
				t.Errorf("got %#x (%s); want %#x", got, f.Text('p', 0), want)
			}
		}
	}

	// test basic rounding behavior (exhaustive rounding testing is done elsewhere)
	const x int64 = 0x7654321076543210 // 63 bits needed
	mp := decDigits64(uint64(x))
	for prec := uint(1); prec <= mp; prec++ {
		f := new(Decimal).SetPrec(prec).SetMode(ToZero).SetInt64(x)
		got := f.int64()
		want := x - x%int64(pow10(mp-prec)) // cut off (round to zero) low prec digits
		if got != want {
			t.Errorf("got %#x (%s); want %#x", got, f.Text('p', 0), want)
		}
	}
}

func TestDecimalSetFloat64(t *testing.T) {
	for _, want := range []float64{
		0,
		1,
		2,
		12345,
		1e10,
		1e100,
		3.14159265e10,
		2.718281828e-123,
		1.0 / 3,
		1.1,
		math.MaxFloat32,
		math.MaxFloat64,
		math.SmallestNonzeroFloat32,
		math.SmallestNonzeroFloat64,
		math.Inf(-1),
		math.Inf(0),
		-math.Inf(1),
	} {
		for i := range [2]int{} {
			if i&1 != 0 {
				want = -want
			}
			var f Decimal
			f.SetFloat64(want)
			if got, _ := f.Float64(); got != want {
				t.Errorf("got %g (%s); want %g", got, f.Text('p', 0), want)
			}
		}
	}

	// test NaN
	defer func() {
		if p, ok := recover().(ErrNaN); !ok {
			t.Errorf("got %v; want ErrNaN panic", p)
		}
	}()
	var f Decimal
	f.SetFloat64(math.NaN())
	// should not reach here
	t.Errorf("got %s; want ErrNaN panic", f.Text('p', 0))
}

func TestDecimalSetInt(t *testing.T) {
	for _, want := range []string{
		"0",
		"1",
		"-1",
		"1234567890",
		"123456789012345678901234567890",
		"123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
	} {
		var x big.Int
		_, ok := x.SetString(want, 0)
		if !ok {
			t.Errorf("invalid integer %s", want)
			continue
		}

		var f Decimal
		f.SetInt(&x)

		// check precision
		n := len(strings.TrimRight(want, "0"))
		if n < DefaultDecimalPrec {
			n = DefaultDecimalPrec
		}
		if prec := f.Prec(); prec != uint(n) {
			t.Errorf("%s: got prec = %d; want %d", want, prec, n)
		}

		// check value
		got := f.Text('g', 100)
		if got != want {
			t.Errorf("got %s (%s); want %s", got, f.Text('p', 0), want)
		}
	}
}

func TestDecimalSetRat(t *testing.T) {
	for _, want := range []struct {
		v string
		p uint
	}{
		{"0", DefaultDecimalPrec},
		{"1", DefaultDecimalPrec},
		{"-1", DefaultDecimalPrec},
		{"1234567890", DefaultDecimalPrec},
		{"123456789012345678901234567890", DefaultDecimalPrec},
		{"123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", 89},
		{"1.2", DefaultDecimalPrec},
		{"3.14159265", DefaultDecimalPrec},
		// TODO(db47h) expand
	} {
		var x big.Rat
		_, ok := x.SetString(want.v)
		if !ok {
			t.Errorf("invalid fraction %s", want.v)
			continue
		}

		var f1, f2 Decimal
		f2.SetPrec(1000)
		f1.SetRat(&x)
		f2.SetRat(&x)

		// check precision when set automatically
		if want.p < DefaultDecimalPrec {
			want.p = DefaultDecimalPrec
		}
		if prec := f1.Prec(); prec != want.p {
			t.Errorf("got prec = %d; want %d", prec, want.p)
		}

		got := f2.Text('g', 100)
		if got != want.v {
			t.Errorf("got %s (%s); want %s", got, f2.Text('p', 0), want.v)
		}
	}
}

func TestFloatSetInf(t *testing.T) {
	var f Decimal
	for _, test := range []struct {
		signbit bool
		prec    uint
		want    string
	}{
		{false, 0, "+Inf"},
		{true, 0, "-Inf"},
		{false, 10, "+Inf"},
		{true, 30, "-Inf"},
	} {
		x := f.SetPrec(test.prec).SetInf(test.signbit)
		if got := x.String(); got != test.want || x.Prec() != test.prec {
			t.Errorf("SetInf(%v) = %s (prec = %d); want %s (prec = %d)", test.signbit, got, x.Prec(), test.want, test.prec)
		}
	}
}

func TestDecimalUint64(t *testing.T) {
	for _, test := range []struct {
		x   string
		out uint64
		acc Accuracy
	}{
		{"-Inf", 0, Above},
		{"-1", 0, Above},
		{"-1e-1000", 0, Above},
		{"-0", 0, Exact},
		{"0", 0, Exact},
		{"1e-1000", 0, Below},
		{"1", 1, Exact},
		{"1.000000000000000000001", 1, Below},
		{"12345.0", 12345, Exact},
		{"12345.000000000000000000001", 12345, Below},
		{"18446744073709551615", 18446744073709551615, Exact},
		{"18446744073709551615.000000000000000000001", math.MaxUint64, Below},
		{"18446744073709551616", math.MaxUint64, Below},
		{"1e10000", math.MaxUint64, Below},
		{"+Inf", math.MaxUint64, Below},
	} {
		x := makeDecimal(test.x)
		out, acc := x.Uint64()
		if out != test.out || acc != test.acc {
			t.Errorf("%s: got %d (%s); want %d (%s)", test.x, out, acc, test.out, test.acc)
		}
	}
}

func TestDecimalInt64(t *testing.T) {
	for _, test := range []struct {
		x   string
		out int64
		acc Accuracy
	}{
		{"-Inf", math.MinInt64, Above},
		{"-1e10000", math.MinInt64, Above},
		{"-9223372036854775809", math.MinInt64, Above},
		{"-9223372036854775808.000000000000000000001", math.MinInt64, Above},
		{"-9223372036854775808", -9223372036854775808, Exact},
		{"-9223372036854775807.000000000000000000001", -9223372036854775807, Above},
		{"-9223372036854775807", -9223372036854775807, Exact},
		{"-12345.000000000000000000001", -12345, Above},
		{"-12345.0", -12345, Exact},
		{"-1.000000000000000000001", -1, Above},
		{"-1.5", -1, Above},
		{"-1", -1, Exact},
		{"-1e-1000", 0, Above},
		{"0", 0, Exact},
		{"1e-1000", 0, Below},
		{"1", 1, Exact},
		{"1.000000000000000000001", 1, Below},
		{"1.5", 1, Below},
		{"12345.0", 12345, Exact},
		{"12345.000000000000000000001", 12345, Below},
		{"9223372036854775807", 9223372036854775807, Exact},
		{"9223372036854775807.000000000000000000001", math.MaxInt64, Below},
		{"9223372036854775808", math.MaxInt64, Below},
		{"1e10000", math.MaxInt64, Below},
		{"+Inf", math.MaxInt64, Below},
	} {
		x := makeDecimal(test.x)
		out, acc := x.Int64()
		if out != test.out || acc != test.acc {
			t.Errorf("%s: got %d (%s); want %d (%s)", test.x, out, acc, test.out, test.acc)
		}
	}
}

func TestDecimalFloat32(t *testing.T) {
	for _, test := range []struct {
		x   string
		out float32
		acc Accuracy
	}{
		{"0", 0, Exact},

		// underflow to zero
		{"1e-1000", 0, Below},
		{"0x0.000002p-127", 0, Below},
		{"0x.0000010p-126", 0, Below},

		// denormals
		{"1.401298464e-45", math.SmallestNonzeroFloat32, Above}, // rounded up to smallest denormal
		{"0x.ffffff8p-149", math.SmallestNonzeroFloat32, Above}, // rounded up to smallest denormal
		{"0x.0000018p-126", math.SmallestNonzeroFloat32, Above}, // rounded up to smallest denormal
		{"0x.0000020p-126", math.SmallestNonzeroFloat32, Exact},
		{"0x.8p-148", math.SmallestNonzeroFloat32, Exact},
		{"1p-149", math.SmallestNonzeroFloat32, Exact},
		{"0x.fffffep-126", math.Float32frombits(0x7fffff), Exact}, // largest denormal

		// special denormal cases (see issues 14553, 14651)
		{"0x0.0000001p-126", math.Float32frombits(0x00000000), Below}, // underflow to zero
		{"0x0.0000008p-126", math.Float32frombits(0x00000000), Below}, // underflow to zero
		{"0x0.0000010p-126", math.Float32frombits(0x00000000), Below}, // rounded down to even
		{"0x0.0000011p-126", math.Float32frombits(0x00000001), Above}, // rounded up to smallest denormal
		{"0x0.0000018p-126", math.Float32frombits(0x00000001), Above}, // rounded up to smallest denormal

		{"0x1.0000000p-149", math.Float32frombits(0x00000001), Exact}, // smallest denormal
		{"0x0.0000020p-126", math.Float32frombits(0x00000001), Exact}, // smallest denormal
		{"0x0.fffffe0p-126", math.Float32frombits(0x007fffff), Exact}, // largest denormal
		{"0x1.0000000p-126", math.Float32frombits(0x00800000), Exact}, // smallest normal

		{"0x0.8p-149", math.Float32frombits(0x000000000), Below}, // rounded down to even
		{"0x0.9p-149", math.Float32frombits(0x000000001), Above}, // rounded up to smallest denormal
		{"0x0.ap-149", math.Float32frombits(0x000000001), Above}, // rounded up to smallest denormal
		{"0x0.bp-149", math.Float32frombits(0x000000001), Above}, // rounded up to smallest denormal
		{"0x0.cp-149", math.Float32frombits(0x000000001), Above}, // rounded up to smallest denormal

		{"0x1.0p-149", math.Float32frombits(0x000000001), Exact}, // smallest denormal
		{"0x1.7p-149", math.Float32frombits(0x000000001), Below},
		{"0x1.8p-149", math.Float32frombits(0x000000002), Above},
		{"0x1.9p-149", math.Float32frombits(0x000000002), Above},

		{"0x2.0p-149", math.Float32frombits(0x000000002), Exact},
		{"0x2.8p-149", math.Float32frombits(0x000000002), Below}, // rounded down to even
		{"0x2.9p-149", math.Float32frombits(0x000000003), Above},

		{"0x3.0p-149", math.Float32frombits(0x000000003), Exact},
		{"0x3.7p-149", math.Float32frombits(0x000000003), Below},
		{"0x3.8p-149", math.Float32frombits(0x000000004), Above}, // rounded up to even

		{"0x4.0p-149", math.Float32frombits(0x000000004), Exact},
		{"0x4.8p-149", math.Float32frombits(0x000000004), Below}, // rounded down to even
		{"0x4.9p-149", math.Float32frombits(0x000000005), Above},

		// specific case from issue 14553
		{"0x7.7p-149", math.Float32frombits(0x000000007), Below},
		{"0x7.8p-149", math.Float32frombits(0x000000008), Above},
		{"0x7.9p-149", math.Float32frombits(0x000000008), Above},

		// normals
		{"0x.ffffffp-126", math.Float32frombits(0x00800000), Above}, // rounded up to smallest normal
		{"1p-126", math.Float32frombits(0x00800000), Exact},         // smallest normal
		{"0x1.fffffep-126", math.Float32frombits(0x00ffffff), Exact},
		{"0x1.ffffffp-126", math.Float32frombits(0x01000000), Above}, // rounded up
		{"1", 1, Exact},
		{"1.000000000000000000001", 1, Below},
		{"12345.0", 12345, Exact},
		{"12345.000000000000000000001", 12345, Below},
		{"0x1.fffffe0p127", math.MaxFloat32, Exact},
		{"0x1.fffffe8p127", math.MaxFloat32, Below},

		// overflow
		{"0x1.ffffff0p127", float32(math.Inf(+1)), Above},
		{"0x1p128", float32(math.Inf(+1)), Above},
		{"1e10000", float32(math.Inf(+1)), Above},
		{"0x1.ffffff0p2147483646", float32(math.Inf(+1)), Above}, // overflow in rounding

		// inf
		{"Inf", float32(math.Inf(+1)), Exact},
	} {
		for i := 0; i < 2; i++ {
			// test both signs
			tx, tout, tacc := test.x, test.out, test.acc
			if i != 0 {
				tx = "-" + tx
				tout = -tout
				tacc = -tacc
			}

			// conversion should match strconv where syntax is agreeable
			if f, err := strconv.ParseFloat(tx, 32); err == nil && !alike32(float32(f), tout) {
				t.Errorf("%s: got %g; want %g (incorrect test data)", tx, f, tout)
			}

			x := makeDecimal(tx)
			out, acc := x.Float32()
			// TODO(db47h): need to check accuracy, somehow.
			if !alike32(out, tout) /* || acc != tacc */ { // do not check accuracy
				t.Errorf("%s: got %g (%#08x, %s); want %g (%#08x, %s)", tx, out, math.Float32bits(out), acc, test.out, math.Float32bits(test.out), tacc)
			}

			// test that x.SetFloat64(float64(f)).Float32() == f
			var x2 Decimal
			out2, acc2 := x2.SetFloat64(float64(out)).Float32()
			if !alike32(out2, out) /* || acc2 != Exact */ {
				t.Errorf("idempotency test: got %g (%s); want %g (Exact)", out2, acc2, out)
			}
		}
	}
}

func TestDecimalFloat64(t *testing.T) {
	const smallestNormalFloat64 = 2.2250738585072014e-308 // 1p-1022
	for _, test := range []struct {
		x   string
		out float64
		acc Accuracy
	}{
		{"0", 0, Exact},

		// underflow to zero
		{"1e-1000", 0, Below},
		{"0x0.0000000000001p-1023", 0, Below},
		{"0x0.00000000000008p-1022", 0, Below},

		// denormals
		{"0x0.0000000000000cp-1022", math.SmallestNonzeroFloat64, Above}, // rounded up to smallest denormal
		{"0x0.00000000000010p-1022", math.SmallestNonzeroFloat64, Exact}, // smallest denormal
		{"0x.8p-1073", math.SmallestNonzeroFloat64, Exact},
		{"1p-1074", math.SmallestNonzeroFloat64, Exact},
		{"0x.fffffffffffffp-1022", math.Float64frombits(0x000fffffffffffff), Exact}, // largest denormal

		// special denormal cases (see issues 14553, 14651)
		{"0x0.00000000000001p-1022", math.Float64frombits(0x00000000000000000), Below}, // underflow to zero
		{"0x0.00000000000004p-1022", math.Float64frombits(0x00000000000000000), Below}, // underflow to zero
		{"0x0.00000000000008p-1022", math.Float64frombits(0x00000000000000000), Below}, // rounded down to even
		{"0x0.00000000000009p-1022", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal
		{"0x0.0000000000000ap-1022", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal

		{"0x0.8p-1074", math.Float64frombits(0x00000000000000000), Below}, // rounded down to even
		{"0x0.9p-1074", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal
		{"0x0.ap-1074", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal
		{"0x0.bp-1074", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal
		{"0x0.cp-1074", math.Float64frombits(0x00000000000000001), Above}, // rounded up to smallest denormal

		{"0x1.0p-1074", math.Float64frombits(0x00000000000000001), Exact},
		{"0x1.7p-1074", math.Float64frombits(0x00000000000000001), Below},
		{"0x1.8p-1074", math.Float64frombits(0x00000000000000002), Above},
		{"0x1.9p-1074", math.Float64frombits(0x00000000000000002), Above},

		{"0x2.0p-1074", math.Float64frombits(0x00000000000000002), Exact},
		{"0x2.8p-1074", math.Float64frombits(0x00000000000000002), Below}, // rounded down to even
		{"0x2.9p-1074", math.Float64frombits(0x00000000000000003), Above},

		{"0x3.0p-1074", math.Float64frombits(0x00000000000000003), Exact},
		{"0x3.7p-1074", math.Float64frombits(0x00000000000000003), Below},
		{"0x3.8p-1074", math.Float64frombits(0x00000000000000004), Above}, // rounded up to even

		{"0x4.0p-1074", math.Float64frombits(0x00000000000000004), Exact},
		{"0x4.8p-1074", math.Float64frombits(0x00000000000000004), Below}, // rounded down to even
		{"0x4.9p-1074", math.Float64frombits(0x00000000000000005), Above},

		// normals
		{"0x.fffffffffffff8p-1022", math.Float64frombits(0x0010000000000000), Above}, // rounded up to smallest normal
		{"1p-1022", math.Float64frombits(0x0010000000000000), Exact},                 // smallest normal
		{"1", 1, Exact},
		{"1.000000000000000000001", 1, Below},
		{"12345.0", 12345, Exact},
		{"12345.000000000000000000001", 12345, Below},
		{"0x1.fffffffffffff0p1023", math.MaxFloat64, Exact},
		{"0x1.fffffffffffff4p1023", math.MaxFloat64, Below},

		// overflow
		{"0x1.fffffffffffff8p1023", math.Inf(+1), Above},
		{"0x1p1024", math.Inf(+1), Above},
		{"1e10000", math.Inf(+1), Above},
		{"0x1.fffffffffffff8p2147483646", math.Inf(+1), Above}, // overflow in rounding
		{"Inf", math.Inf(+1), Exact},

		// selected denormalized values that were handled incorrectly in the past
		{"0x.fffffffffffffp-1022", smallestNormalFloat64 - math.SmallestNonzeroFloat64, Exact},
		{"4503599627370495p-1074", smallestNormalFloat64 - math.SmallestNonzeroFloat64, Exact},

		// https://www.exploringbinary.com/php-hangs-on-numeric-value-2-2250738585072011e-308/
		{"2.2250738585072011e-308", 2.225073858507201e-308, Below},
		// https://www.exploringbinary.com/java-hangs-when-converting-2-2250738585072012e-308/
		{"2.2250738585072012e-308", 2.2250738585072014e-308, Above},
	} {
		for i := 0; i < 2; i++ {
			// test both signs
			tx, tout, tacc := test.x, test.out, test.acc
			if i != 0 {
				tx = "-" + tx
				tout = -tout
				tacc = -tacc
			}

			// conversion should match strconv where syntax is agreeable
			if f, err := strconv.ParseFloat(tx, 64); err == nil && !alike64(f, tout) {
				t.Errorf("%s: got %g; want %g (incorrect test data)", tx, f, tout)
			}

			x := makeDecimal(tx)
			out, acc := x.Float64()
			// TODO(db47h): need to check accuracy, somehow.
			if !alike64(out, tout) /*|| acc != tacc */ {
				t.Errorf("%s: got %g (%#016x, %s); want %g (%#016x, %s)", tx, out, math.Float64bits(out), acc, test.out, math.Float64bits(test.out), tacc)
			}

			// test that x.SetFloat64(f).Float64() == f
			var x2 Decimal
			out2, acc2 := x2.SetFloat64(out).Float64()
			if !alike64(out2, out) /* || acc2 != Exact */ {
				t.Errorf("idempotency test: got %g (%s); want %g (Exact)", out2, acc2, out)
			}
		}
	}
}

func TestDecimalInt(t *testing.T) {
	for _, test := range []struct {
		x    string
		want string
		acc  Accuracy
	}{
		{"0", "0", Exact},
		{"+0", "0", Exact},
		{"-0", "0", Exact},
		{"Inf", "nil", Below},
		{"+Inf", "nil", Below},
		{"-Inf", "nil", Above},
		{"1", "1", Exact},
		{"-1", "-1", Exact},
		{"1.23", "1", Below},
		{"-1.23", "-1", Above},
		{"123e-2", "1", Below},
		{"123e-3", "0", Below},
		{"123e-4", "0", Below},
		{"1e-1000", "0", Below},
		{"-1e-1000", "0", Above},
		{"1e+10", "10000000000", Exact},
		{"1e+100", "10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", Exact},
	} {
		x := makeDecimal(test.x)
		res, acc := x.Int(nil)
		got := "nil"
		if res != nil {
			got = res.String()
		}
		if got != test.want || acc != test.acc {
			t.Errorf("%s: got %s (%s); want %s (%s)", test.x, got, acc, test.want, test.acc)
		}
	}

	// check that supplied *Int is used
	for _, f := range []string{"0", "1", "-1", "1234"} {
		x := makeDecimal(f)
		i := new(big.Int)
		if res, _ := x.Int(i); res != i {
			t.Errorf("(%s).Int is not using supplied *Int", f)
		}
	}
}

func TestDecimalRat(t *testing.T) {
	for _, test := range []struct {
		x, want string
		acc     Accuracy
	}{
		{"0", "0/1", Exact},
		{"+0", "0/1", Exact},
		{"-0", "0/1", Exact},
		{"Inf", "nil", Below},
		{"+Inf", "nil", Below},
		{"-Inf", "nil", Above},
		{"1", "1/1", Exact},
		{"-1", "-1/1", Exact},
		{"1.25", "5/4", Exact},
		{"-1.25", "-5/4", Exact},
		{"1e10", "10000000000/1", Exact},
		{"1p10", "1024/1", Exact},
		{"-1p-10", "-1/1024", Exact},
		{"3.14159265", "62831853/20000000", Exact},
	} {
		x := makeDecimal(test.x).SetPrec(34)
		res, acc := x.Rat(nil)
		got := "nil"
		if res != nil {
			got = res.String()
		}
		if got != test.want {
			t.Errorf("%s: got %s; want %s", test.x, got, test.want)
			continue
		}
		if acc != test.acc {
			t.Errorf("%s: got %s; want %s", test.x, acc, test.acc)
			continue
		}

		// inverse conversion
		if res != nil {
			got := new(Decimal).SetPrec(34).SetRat(res)
			if got.Cmp(x) != 0 {
				t.Errorf("%s: got %s; want %s", test.x, got, x)
			}
		}
	}

	// check that supplied *Rat is used
	for _, f := range []string{"0", "1", "-1", "1234"} {
		x := makeDecimal(f)
		r := new(big.Rat)
		if res, _ := x.Rat(r); res != r {
			t.Errorf("(%s).Rat is not using supplied *Rat", f)
		}
	}
}

func TestDecimalAbs(t *testing.T) {
	for _, test := range []string{
		"0",
		"1",
		"1234",
		"1.23e-2",
		"1e-1000",
		"1e1000",
		"Inf",
	} {
		p := makeDecimal(test)
		a := new(Decimal).Abs(p)
		if !alike(a, p) {
			t.Errorf("%s: got %s; want %s", test, a.Text('g', 10), test)
		}

		n := makeDecimal("-" + test)
		a.Abs(n)
		if !alike(a, p) {
			t.Errorf("-%s: got %s; want %s", test, a.Text('g', 10), test)
		}
	}
}

func TestDecimalNeg(t *testing.T) {
	for _, test := range []string{
		"0",
		"1",
		"1234",
		"1.23e-2",
		"1e-1000",
		"1e1000",
		"Inf",
	} {
		p1 := makeDecimal(test)
		n1 := makeDecimal("-" + test)
		n2 := new(Decimal).Neg(p1)
		p2 := new(Decimal).Neg(n2)
		if !alike(n2, n1) {
			t.Errorf("%s: got %s; want %s", test, n2.Text('g', 10), n1.Text('g', 10))
		}
		if !alike(p2, p1) {
			t.Errorf("%s: got %s; want %s", test, p2.Text('g', 10), p1.Text('g', 10))
		}
	}
}

func TestDecimalInc(t *testing.T) {
	const n = 10
	for _, prec := range precList {
		var x, one Decimal
		x.SetPrec(prec)
		one.SetInt64(1)
		for i := 0; i < n; i++ {
			x.Add(&x, &one)
		}
		if x.Cmp(new(Decimal).SetInt64(n)) != 0 {
			t.Errorf("prec = %d: got %s; want %d", prec, &x, n)
		}
	}
}

// Selected precisions with which to run various tests.
var precList = [...]uint{1, 2, 5, 8, 10, 16, 17, 19, 34, 38, 50, 100, 250, 300, 3000}

// // Selected bits with which to run various tests.
// // Each entry is a list of bits representing a floating-point number (see fromBits).
// var bitsList = [...]Bits{
// 	{},           // = 0
// 	{0},          // = 1
// 	{1},          // = 2
// 	{-1},         // = 1/2
// 	{10},         // = 2**10 == 1024
// 	{-10},        // = 2**-10 == 1/1024
// 	{100, 10, 1}, // = 2**100 + 2**10 + 2**1
// 	{0, -1, -2, -10},
// 	// TODO(gri) add more test cases
// }

// // TestFloatAdd tests Float.Add/Sub by comparing the result of a "manual"
// // addition/subtraction of arguments represented by Bits values with the
// // respective Float addition/subtraction for a variety of precisions
// // and rounding modes.
// func TestFloatAdd(t *testing.T) {
// 	for _, xbits := range bitsList {
// 		for _, ybits := range bitsList {
// 			// exact values
// 			x := xbits.Float()
// 			y := ybits.Float()
// 			zbits := xbits.add(ybits)
// 			z := zbits.Float()

// 			for i, mode := range [...]RoundingMode{ToZero, ToNearestEven, AwayFromZero} {
// 				for _, prec := range precList {
// 					got := new(Float).SetPrec(prec).SetMode(mode)
// 					got.Add(x, y)
// 					want := zbits.round(prec, mode)
// 					if got.Cmp(want) != 0 {
// 						t.Errorf("i = %d, prec = %d, %s:\n\t     %s %v\n\t+    %s %v\n\t=    %s\n\twant %s",
// 							i, prec, mode, x, xbits, y, ybits, got, want)
// 					}

// 					got.Sub(z, x)
// 					want = ybits.round(prec, mode)
// 					if got.Cmp(want) != 0 {
// 						t.Errorf("i = %d, prec = %d, %s:\n\t     %s %v\n\t-    %s %v\n\t=    %s\n\twant %s",
// 							i, prec, mode, z, zbits, x, xbits, got, want)
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// // TestFloatAddRoundZero tests Float.Add/Sub rounding when the result is exactly zero.
// // x + (-x) or x - x for non-zero x should be +0 in all cases except when
// // the rounding mode is ToNegativeInf in which case it should be -0.
// func TestFloatAddRoundZero(t *testing.T) {
// 	for _, mode := range [...]RoundingMode{ToNearestEven, ToNearestAway, ToZero, AwayFromZero, ToPositiveInf, ToNegativeInf} {
// 		x := NewFloat(5.0)
// 		y := new(Float).Neg(x)
// 		want := NewFloat(0.0)
// 		if mode == ToNegativeInf {
// 			want.Neg(want)
// 		}
// 		got := new(Float).SetMode(mode)
// 		got.Add(x, y)
// 		if got.Cmp(want) != 0 || got.neg != (mode == ToNegativeInf) {
// 			t.Errorf("%s:\n\t     %v\n\t+    %v\n\t=    %v\n\twant %v",
// 				mode, x, y, got, want)
// 		}
// 		got.Sub(x, x)
// 		if got.Cmp(want) != 0 || got.neg != (mode == ToNegativeInf) {
// 			t.Errorf("%v:\n\t     %v\n\t-    %v\n\t=    %v\n\twant %v",
// 				mode, x, x, got, want)
// 		}
// 	}
// }

// // TestFloatAdd32 tests that Float.Add/Sub of numbers with
// // 24bit mantissa behaves like float32 addition/subtraction
// // (excluding denormal numbers).
// func TestFloatAdd32(t *testing.T) {
// 	// chose base such that we cross the mantissa precision limit
// 	const base = 1<<26 - 0x10 // 11...110000 (26 bits)
// 	for d := 0; d <= 0x10; d++ {
// 		for i := range [2]int{} {
// 			x0, y0 := float64(base), float64(d)
// 			if i&1 != 0 {
// 				x0, y0 = y0, x0
// 			}

// 			x := NewFloat(x0)
// 			y := NewFloat(y0)
// 			z := new(Float).SetPrec(24)

// 			z.Add(x, y)
// 			got, acc := z.Float32()
// 			want := float32(y0) + float32(x0)
// 			if got != want || acc != Exact {
// 				t.Errorf("d = %d: %g + %g = %g (%s); want %g (Exact)", d, x0, y0, got, acc, want)
// 			}

// 			z.Sub(z, y)
// 			got, acc = z.Float32()
// 			want = float32(want) - float32(y0)
// 			if got != want || acc != Exact {
// 				t.Errorf("d = %d: %g - %g = %g (%s); want %g (Exact)", d, x0+y0, y0, got, acc, want)
// 			}
// 		}
// 	}
// }

// // TestFloatAdd64 tests that Float.Add/Sub of numbers with
// // 53bit mantissa behaves like float64 addition/subtraction.
// func TestFloatAdd64(t *testing.T) {
// 	// chose base such that we cross the mantissa precision limit
// 	const base = 1<<55 - 0x10 // 11...110000 (55 bits)
// 	for d := 0; d <= 0x10; d++ {
// 		for i := range [2]int{} {
// 			x0, y0 := float64(base), float64(d)
// 			if i&1 != 0 {
// 				x0, y0 = y0, x0
// 			}

// 			x := NewFloat(x0)
// 			y := NewFloat(y0)
// 			z := new(Float).SetPrec(53)

// 			z.Add(x, y)
// 			got, acc := z.Float64()
// 			want := x0 + y0
// 			if got != want || acc != Exact {
// 				t.Errorf("d = %d: %g + %g = %g (%s); want %g (Exact)", d, x0, y0, got, acc, want)
// 			}

// 			z.Sub(z, y)
// 			got, acc = z.Float64()
// 			want -= y0
// 			if got != want || acc != Exact {
// 				t.Errorf("d = %d: %g - %g = %g (%s); want %g (Exact)", d, x0+y0, y0, got, acc, want)
// 			}
// 		}
// 	}
// }

// func TestIssue20490(t *testing.T) {
// 	var tests = []struct {
// 		a, b float64
// 	}{
// 		{4, 1},
// 		{-4, 1},
// 		{4, -1},
// 		{-4, -1},
// 	}

// 	for _, test := range tests {
// 		a, b := NewFloat(test.a), NewFloat(test.b)
// 		diff := new(Float).Sub(a, b)
// 		b.Sub(a, b)
// 		if b.Cmp(diff) != 0 {
// 			t.Errorf("got %g - %g = %g; want %g\n", a, NewFloat(test.b), b, diff)
// 		}

// 		b = NewFloat(test.b)
// 		sum := new(Float).Add(a, b)
// 		b.Add(a, b)
// 		if b.Cmp(sum) != 0 {
// 			t.Errorf("got %g + %g = %g; want %g\n", a, NewFloat(test.b), b, sum)
// 		}

// 	}
// }

// // TestFloatMul tests Float.Mul/Quo by comparing the result of a "manual"
// // multiplication/division of arguments represented by Bits values with the
// // respective Float multiplication/division for a variety of precisions
// // and rounding modes.
// func TestFloatMul(t *testing.T) {
// 	for _, xbits := range bitsList {
// 		for _, ybits := range bitsList {
// 			// exact values
// 			x := xbits.Float()
// 			y := ybits.Float()
// 			zbits := xbits.mul(ybits)
// 			z := zbits.Float()

// 			for i, mode := range [...]RoundingMode{ToZero, ToNearestEven, AwayFromZero} {
// 				for _, prec := range precList {
// 					got := new(Float).SetPrec(prec).SetMode(mode)
// 					got.Mul(x, y)
// 					want := zbits.round(prec, mode)
// 					if got.Cmp(want) != 0 {
// 						t.Errorf("i = %d, prec = %d, %s:\n\t     %v %v\n\t*    %v %v\n\t=    %v\n\twant %v",
// 							i, prec, mode, x, xbits, y, ybits, got, want)
// 					}

// 					if x.Sign() == 0 {
// 						continue // ignore div-0 case (not invertable)
// 					}
// 					got.Quo(z, x)
// 					want = ybits.round(prec, mode)
// 					if got.Cmp(want) != 0 {
// 						t.Errorf("i = %d, prec = %d, %s:\n\t     %v %v\n\t/    %v %v\n\t=    %v\n\twant %v",
// 							i, prec, mode, z, zbits, x, xbits, got, want)
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// // TestFloatMul64 tests that Float.Mul/Quo of numbers with
// // 53bit mantissa behaves like float64 multiplication/division.
// func TestFloatMul64(t *testing.T) {
// 	for _, test := range []struct {
// 		x, y float64
// 	}{
// 		{0, 0},
// 		{0, 1},
// 		{1, 1},
// 		{1, 1.5},
// 		{1.234, 0.5678},
// 		{2.718281828, 3.14159265358979},
// 		{2.718281828e10, 3.14159265358979e-32},
// 		{1.0 / 3, 1e200},
// 	} {
// 		for i := range [8]int{} {
// 			x0, y0 := test.x, test.y
// 			if i&1 != 0 {
// 				x0 = -x0
// 			}
// 			if i&2 != 0 {
// 				y0 = -y0
// 			}
// 			if i&4 != 0 {
// 				x0, y0 = y0, x0
// 			}

// 			x := NewFloat(x0)
// 			y := NewFloat(y0)
// 			z := new(Float).SetPrec(53)

// 			z.Mul(x, y)
// 			got, _ := z.Float64()
// 			want := x0 * y0
// 			if got != want {
// 				t.Errorf("%g * %g = %g; want %g", x0, y0, got, want)
// 			}

// 			if y0 == 0 {
// 				continue // avoid division-by-zero
// 			}
// 			z.Quo(z, y)
// 			got, _ = z.Float64()
// 			want /= y0
// 			if got != want {
// 				t.Errorf("%g / %g = %g; want %g", x0*y0, y0, got, want)
// 			}
// 		}
// 	}
// }

// func TestIssue6866(t *testing.T) {
// 	for _, prec := range precList {
// 		two := new(Float).SetPrec(prec).SetInt64(2)
// 		one := new(Float).SetPrec(prec).SetInt64(1)
// 		three := new(Float).SetPrec(prec).SetInt64(3)
// 		msix := new(Float).SetPrec(prec).SetInt64(-6)
// 		psix := new(Float).SetPrec(prec).SetInt64(+6)

// 		p := new(Float).SetPrec(prec)
// 		z1 := new(Float).SetPrec(prec)
// 		z2 := new(Float).SetPrec(prec)

// 		// z1 = 2 + 1.0/3*-6
// 		p.Quo(one, three)
// 		p.Mul(p, msix)
// 		z1.Add(two, p)

// 		// z2 = 2 - 1.0/3*+6
// 		p.Quo(one, three)
// 		p.Mul(p, psix)
// 		z2.Sub(two, p)

// 		if z1.Cmp(z2) != 0 {
// 			t.Fatalf("prec %d: got z1 = %v != z2 = %v; want z1 == z2\n", prec, z1, z2)
// 		}
// 		if z1.Sign() != 0 {
// 			t.Errorf("prec %d: got z1 = %v; want 0", prec, z1)
// 		}
// 		if z2.Sign() != 0 {
// 			t.Errorf("prec %d: got z2 = %v; want 0", prec, z2)
// 		}
// 	}
// }

// func TestFloatQuo(t *testing.T) {
// 	// TODO(gri) make the test vary these precisions
// 	preci := 200 // precision of integer part
// 	precf := 20  // precision of fractional part

// 	for i := 0; i < 8; i++ {
// 		// compute accurate (not rounded) result z
// 		bits := Bits{preci - 1}
// 		if i&3 != 0 {
// 			bits = append(bits, 0)
// 		}
// 		if i&2 != 0 {
// 			bits = append(bits, -1)
// 		}
// 		if i&1 != 0 {
// 			bits = append(bits, -precf)
// 		}
// 		z := bits.Float()

// 		// compute accurate x as z*y
// 		y := NewFloat(3.14159265358979323e123)

// 		x := new(Float).SetPrec(z.Prec() + y.Prec()).SetMode(ToZero)
// 		x.Mul(z, y)

// 		// leave for debugging
// 		// fmt.Printf("x = %s\ny = %s\nz = %s\n", x, y, z)

// 		if got := x.Acc(); got != Exact {
// 			t.Errorf("got acc = %s; want exact", got)
// 		}

// 		// round accurate z for a variety of precisions and
// 		// modes and compare against result of x / y.
// 		for _, mode := range [...]RoundingMode{ToZero, ToNearestEven, AwayFromZero} {
// 			for d := -5; d < 5; d++ {
// 				prec := uint(preci + d)
// 				got := new(Float).SetPrec(prec).SetMode(mode).Quo(x, y)
// 				want := bits.round(prec, mode)
// 				if got.Cmp(want) != 0 {
// 					t.Errorf("i = %d, prec = %d, %s:\n\t     %s\n\t/    %s\n\t=    %s\n\twant %s",
// 						i, prec, mode, x, y, got, want)
// 				}
// 			}
// 		}
// 	}
// }

// var long = flag.Bool("long", false, "run very long tests")

// // TestFloatQuoSmoke tests all divisions x/y for values x, y in the range [-n, +n];
// // it serves as a smoke test for basic correctness of division.
// func TestFloatQuoSmoke(t *testing.T) {
// 	n := 10
// 	if *long {
// 		n = 1000
// 	}

// 	const dprec = 3         // max. precision variation
// 	const prec = 10 + dprec // enough bits to hold n precisely
// 	for x := -n; x <= n; x++ {
// 		for y := -n; y < n; y++ {
// 			if y == 0 {
// 				continue
// 			}

// 			a := float64(x)
// 			b := float64(y)
// 			c := a / b

// 			// vary operand precision (only ok as long as a, b can be represented correctly)
// 			for ad := -dprec; ad <= dprec; ad++ {
// 				for bd := -dprec; bd <= dprec; bd++ {
// 					A := new(Float).SetPrec(uint(prec + ad)).SetFloat64(a)
// 					B := new(Float).SetPrec(uint(prec + bd)).SetFloat64(b)
// 					C := new(Float).SetPrec(53).Quo(A, B) // C has float64 mantissa width

// 					cc, acc := C.Float64()
// 					if cc != c {
// 						t.Errorf("%g/%g = %s; want %.5g\n", a, b, C.Text('g', 5), c)
// 						continue
// 					}
// 					if acc != Exact {
// 						t.Errorf("%g/%g got %s result; want exact result", a, b, acc)
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// // TestFloatArithmeticSpecialValues tests that Float operations produce the
// // correct results for combinations of zero (±0), finite (±1 and ±2.71828),
// // and infinite (±Inf) operands.
// func TestFloatArithmeticSpecialValues(t *testing.T) {
// 	zero := 0.0
// 	args := []float64{math.Inf(-1), -2.71828, -1, -zero, zero, 1, 2.71828, math.Inf(1)}
// 	xx := new(Float)
// 	yy := new(Float)
// 	got := new(Float)
// 	want := new(Float)
// 	for i := 0; i < 4; i++ {
// 		for _, x := range args {
// 			xx.SetFloat64(x)
// 			// check conversion is correct
// 			// (no need to do this for y, since we see exactly the
// 			// same values there)
// 			if got, acc := xx.Float64(); got != x || acc != Exact {
// 				t.Errorf("Float(%g) == %g (%s)", x, got, acc)
// 			}
// 			for _, y := range args {
// 				yy.SetFloat64(y)
// 				var (
// 					op string
// 					z  float64
// 					f  func(z, x, y *Float) *Float
// 				)
// 				switch i {
// 				case 0:
// 					op = "+"
// 					z = x + y
// 					f = (*Float).Add
// 				case 1:
// 					op = "-"
// 					z = x - y
// 					f = (*Float).Sub
// 				case 2:
// 					op = "*"
// 					z = x * y
// 					f = (*Float).Mul
// 				case 3:
// 					op = "/"
// 					z = x / y
// 					f = (*Float).Quo
// 				default:
// 					panic("unreachable")
// 				}
// 				var errnan bool // set if execution of f panicked with ErrNaN
// 				// protect execution of f
// 				func() {
// 					defer func() {
// 						if p := recover(); p != nil {
// 							_ = p.(ErrNaN) // re-panic if not ErrNaN
// 							errnan = true
// 						}
// 					}()
// 					f(got, xx, yy)
// 				}()
// 				if math.IsNaN(z) {
// 					if !errnan {
// 						t.Errorf("%5g %s %5g = %5s; want ErrNaN panic", x, op, y, got)
// 					}
// 					continue
// 				}
// 				if errnan {
// 					t.Errorf("%5g %s %5g panicked with ErrNan; want %5s", x, op, y, want)
// 					continue
// 				}
// 				want.SetFloat64(z)
// 				if !alike(got, want) {
// 					t.Errorf("%5g %s %5g = %5s; want %5s", x, op, y, got, want)
// 				}
// 			}
// 		}
// 	}
// }

// func TestFloatArithmeticOverflow(t *testing.T) {
// 	for _, test := range []struct {
// 		prec       uint
// 		mode       RoundingMode
// 		op         byte
// 		x, y, want string
// 		acc        Accuracy
// 	}{
// 		{4, ToNearestEven, '+', "0", "0", "0", Exact},                   // smoke test
// 		{4, ToNearestEven, '+', "0x.8p+0", "0x.8p+0", "0x.8p+1", Exact}, // smoke test

// 		{4, ToNearestEven, '+', "0", "0x.8p2147483647", "0x.8p+2147483647", Exact},
// 		{4, ToNearestEven, '+', "0x.8p2147483500", "0x.8p2147483647", "0x.8p+2147483647", Below}, // rounded to zero
// 		{4, ToNearestEven, '+', "0x.8p2147483647", "0x.8p2147483647", "+Inf", Above},             // exponent overflow in +
// 		{4, ToNearestEven, '+', "-0x.8p2147483647", "-0x.8p2147483647", "-Inf", Below},           // exponent overflow in +
// 		{4, ToNearestEven, '-', "-0x.8p2147483647", "0x.8p2147483647", "-Inf", Below},            // exponent overflow in -

// 		{4, ToZero, '+', "0x.fp2147483647", "0x.8p2147483643", "0x.fp+2147483647", Below}, // rounded to zero
// 		{4, ToNearestEven, '+', "0x.fp2147483647", "0x.8p2147483643", "+Inf", Above},      // exponent overflow in rounding
// 		{4, AwayFromZero, '+', "0x.fp2147483647", "0x.8p2147483643", "+Inf", Above},       // exponent overflow in rounding

// 		{4, AwayFromZero, '-', "-0x.fp2147483647", "0x.8p2147483644", "-Inf", Below},        // exponent overflow in rounding
// 		{4, ToNearestEven, '-', "-0x.fp2147483647", "0x.8p2147483643", "-Inf", Below},       // exponent overflow in rounding
// 		{4, ToZero, '-', "-0x.fp2147483647", "0x.8p2147483643", "-0x.fp+2147483647", Above}, // rounded to zero

// 		{4, ToNearestEven, '+', "0", "0x.8p-2147483648", "0x.8p-2147483648", Exact},
// 		{4, ToNearestEven, '+', "0x.8p-2147483648", "0x.8p-2147483648", "0x.8p-2147483647", Exact},

// 		{4, ToNearestEven, '*', "1", "0x.8p2147483647", "0x.8p+2147483647", Exact},
// 		{4, ToNearestEven, '*', "2", "0x.8p2147483647", "+Inf", Above},  // exponent overflow in *
// 		{4, ToNearestEven, '*', "-2", "0x.8p2147483647", "-Inf", Below}, // exponent overflow in *

// 		{4, ToNearestEven, '/', "0.5", "0x.8p2147483647", "0x.8p-2147483646", Exact},
// 		{4, ToNearestEven, '/', "0x.8p+0", "0x.8p2147483647", "0x.8p-2147483646", Exact},
// 		{4, ToNearestEven, '/', "0x.8p-1", "0x.8p2147483647", "0x.8p-2147483647", Exact},
// 		{4, ToNearestEven, '/', "0x.8p-2", "0x.8p2147483647", "0x.8p-2147483648", Exact},
// 		{4, ToNearestEven, '/', "0x.8p-3", "0x.8p2147483647", "0", Below}, // exponent underflow in /
// 	} {
// 		x := makeFloat(test.x)
// 		y := makeFloat(test.y)
// 		z := new(Float).SetPrec(test.prec).SetMode(test.mode)
// 		switch test.op {
// 		case '+':
// 			z.Add(x, y)
// 		case '-':
// 			z.Sub(x, y)
// 		case '*':
// 			z.Mul(x, y)
// 		case '/':
// 			z.Quo(x, y)
// 		default:
// 			panic("unreachable")
// 		}
// 		if got := z.Text('p', 0); got != test.want || z.Acc() != test.acc {
// 			t.Errorf(
// 				"prec = %d (%s): %s %c %s = %s (%s); want %s (%s)",
// 				test.prec, test.mode, x.Text('p', 0), test.op, y.Text('p', 0), got, z.Acc(), test.want, test.acc,
// 			)
// 		}
// 	}
// }

// // TODO(gri) Add tests that check correctness in the presence of aliasing.

// // For rounding modes ToNegativeInf and ToPositiveInf, rounding is affected
// // by the sign of the value to be rounded. Test that rounding happens after
// // the sign of a result has been set.
// // This test uses specific values that are known to fail if rounding is
// // "factored" out before setting the result sign.
// func TestFloatArithmeticRounding(t *testing.T) {
// 	for _, test := range []struct {
// 		mode       RoundingMode
// 		prec       uint
// 		x, y, want int64
// 		op         byte
// 	}{
// 		{ToZero, 3, -0x8, -0x1, -0x8, '+'},
// 		{AwayFromZero, 3, -0x8, -0x1, -0xa, '+'},
// 		{ToNegativeInf, 3, -0x8, -0x1, -0xa, '+'},

// 		{ToZero, 3, -0x8, 0x1, -0x8, '-'},
// 		{AwayFromZero, 3, -0x8, 0x1, -0xa, '-'},
// 		{ToNegativeInf, 3, -0x8, 0x1, -0xa, '-'},

// 		{ToZero, 3, -0x9, 0x1, -0x8, '*'},
// 		{AwayFromZero, 3, -0x9, 0x1, -0xa, '*'},
// 		{ToNegativeInf, 3, -0x9, 0x1, -0xa, '*'},

// 		{ToZero, 3, -0x9, 0x1, -0x8, '/'},
// 		{AwayFromZero, 3, -0x9, 0x1, -0xa, '/'},
// 		{ToNegativeInf, 3, -0x9, 0x1, -0xa, '/'},
// 	} {
// 		var x, y, z Float
// 		x.SetInt64(test.x)
// 		y.SetInt64(test.y)
// 		z.SetPrec(test.prec).SetMode(test.mode)
// 		switch test.op {
// 		case '+':
// 			z.Add(&x, &y)
// 		case '-':
// 			z.Sub(&x, &y)
// 		case '*':
// 			z.Mul(&x, &y)
// 		case '/':
// 			z.Quo(&x, &y)
// 		default:
// 			panic("unreachable")
// 		}
// 		if got, acc := z.Int64(); got != test.want || acc != Exact {
// 			t.Errorf("%s, %d bits: %d %c %d = %d (%s); want %d (Exact)",
// 				test.mode, test.prec, test.x, test.op, test.y, got, acc, test.want,
// 			)
// 		}
// 	}
// }

// // TestFloatCmpSpecialValues tests that Cmp produces the correct results for
// // combinations of zero (±0), finite (±1 and ±2.71828), and infinite (±Inf)
// // operands.
// func TestFloatCmpSpecialValues(t *testing.T) {
// 	zero := 0.0
// 	args := []float64{math.Inf(-1), -2.71828, -1, -zero, zero, 1, 2.71828, math.Inf(1)}
// 	xx := new(Float)
// 	yy := new(Float)
// 	for i := 0; i < 4; i++ {
// 		for _, x := range args {
// 			xx.SetFloat64(x)
// 			// check conversion is correct
// 			// (no need to do this for y, since we see exactly the
// 			// same values there)
// 			if got, acc := xx.Float64(); got != x || acc != Exact {
// 				t.Errorf("Float(%g) == %g (%s)", x, got, acc)
// 			}
// 			for _, y := range args {
// 				yy.SetFloat64(y)
// 				got := xx.Cmp(yy)
// 				want := 0
// 				switch {
// 				case x < y:
// 					want = -1
// 				case x > y:
// 					want = +1
// 				}
// 				if got != want {
// 					t.Errorf("(%g).Cmp(%g) = %v; want %v", x, y, got, want)
// 				}
// 			}
// 		}
// 	}
// }

// func BenchmarkFloatAdd(b *testing.B) {
// 	x := new(Float)
// 	y := new(Float)
// 	z := new(Float)

// 	for _, prec := range []uint{10, 1e2, 1e3, 1e4, 1e5} {
// 		x.SetPrec(prec).SetRat(NewRat(1, 3))
// 		y.SetPrec(prec).SetRat(NewRat(1, 6))
// 		z.SetPrec(prec)

// 		b.Run(fmt.Sprintf("%v", prec), func(b *testing.B) {
// 			b.ReportAllocs()
// 			for i := 0; i < b.N; i++ {
// 				z.Add(x, y)
// 			}
// 		})
// 	}
// }

// func BenchmarkFloatSub(b *testing.B) {
// 	x := new(Float)
// 	y := new(Float)
// 	z := new(Float)

// 	for _, prec := range []uint{10, 1e2, 1e3, 1e4, 1e5} {
// 		x.SetPrec(prec).SetRat(NewRat(1, 3))
// 		y.SetPrec(prec).SetRat(NewRat(1, 6))
// 		z.SetPrec(prec)

// 		b.Run(fmt.Sprintf("%v", prec), func(b *testing.B) {
// 			b.ReportAllocs()
// 			for i := 0; i < b.N; i++ {
// 				z.Sub(x, y)
// 			}
// 		})
// 	}
// }
