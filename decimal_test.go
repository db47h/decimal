// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"math/big"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

var benchU uint

var intData = []struct {
	s  string
	b  int
	p  uint
	d  dec
	pr uint
	e  int32
}{
	{"00000000000000000001232", 10, 0, dec{1232000000000000000}, DefaultDecimalPrec, 4},
	{"1234567890123456789_0123456789012345678_9012345678901234567_8901234567890123456_78901234567890", 0, 90,
		dec{7890123456789000000, 8901234567890123456, 9012345678901234567, 123456789012345678, 1234567890123456789},
		90, 90},
	{"1235", 0, 0, dec{1235000000000000000}, DefaultDecimalPrec, 4},
	{"1235", 0, 3, dec{1240000000000000000}, 3, 4},
	{"1245", 0, 3, dec{1240000000000000000}, 3, 4},
	{"12451", 0, 3, dec{1250000000000000000}, 3, 5},
	{"0", 0, 0, nil, DefaultDecimalPrec, 0},
}

func TestDecimal_dnorm(t *testing.T) {
	for i := 0; i < 10000; i++ {
	again:
		w := uint(rand.Uint64()) % _DB
		e := uint(rand.Intn(_DW + 1))
		h, l := mulWW(Word(w), pow10(e))
		// convert h, l from base _B (2**64) to base _BD (10**19) or 2**32 -> 10**9
		h, l = div10W(h, l)
		d := dec{Word(l), Word(h)}.norm()
		if len(d) == 0 {
			if w == 0 {
				goto again
			}
			t.Fatalf("dec{%v, %v}).norm() returned dec{} for word %d", l, h, w)
		}
		dd := dec(nil).set(d)
		s := dnorm(dd)
		// d should now have a single element with e shifted left
		ew := w * uint(pow10(_DW-decDigits(w)))
		es := int64(uint(len(d)*_DW) - (decDigits(w) + e))
		if dd[len(dd)-1] != Word(ew) || s != es {
			t.Fatalf("%ve%v => dnorm(%v) = %v, %v --- Expected %d, %d",
				w, e, d, dd, s, w, es)
		}
	}
}

func TestDecimal_SetInt(t *testing.T) {
	for i, td := range intData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			b, _ := new(big.Int).SetString(td.s, td.b)
			d := new(Decimal).SetMode(ToNearestEven).SetPrec(td.p).SetInt(b)
			ep := td.pr
			if td.p == 0 && ep < DefaultDecimalPrec {
				ep = DefaultDecimalPrec
			}
			if !reflect.DeepEqual(td.d, d.mant) {
				t.Fatalf("\nexpected mantissa %v\n              got %v", td.d, d.mant)
			}
			if ep != d.Prec() {
				t.Fatalf("\nexpected precision %v\n               got %v", ep, d.Prec())
			}
			if td.e != d.exp {
				t.Fatalf("\nexpected exponent %v\n              got %v", td.p, d.Prec())
			}
		})
	}
}

func TestDecimal_SetString(t *testing.T) {
	for i, td := range intData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			d, _ := new(Decimal).SetMode(ToNearestEven).SetPrec(td.p).SetString(td.s)
			if !reflect.DeepEqual(td.d, d.mant) {
				t.Fatalf("\nexpected mantissa %v\n              got %v", td.d, d.mant)
			}
			if td.pr != d.Prec() {
				t.Fatalf("\nexpected precision %v\n               got %v", td.pr, d.Prec())
			}
			if td.e != d.exp {
				t.Fatalf("\nexpected exponent %v\n              got %v", td.p, d.Prec())
			}
		})
	}
}

func BenchmarkDecimal_dnorm(b *testing.B) {
	d := dec(nil).make(1000)
	for i := range d {
		d[i] = Word(rand.Uint64()) % _DB
	}
	for i := 0; i < b.N; i++ {
		d[0] = Word(rand.Uint64()) % _DB
		d[len(d)-1] = Word(rand.Uint64()) % _DB
		benchU = uint(dnorm(d))
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
	x := NewDecimal(1.23).SetPrec(3)
	y := NewDecimal(2.27).SetPrec(3)
	u := NewDecimal(0.003).SetPrec(3)
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
