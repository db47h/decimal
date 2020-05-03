package decimal

import (
	"bytes"
	"math/big"
	"math/bits"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func Test_dec_norm(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
		w := uint(rand.Uint64()) % _BD
		e := uint(rand.Intn(_WD + 1))
		h, l := bits.Mul(w, pow10(e))
		// convert h, l from base _B (2**64) to base _BD (10**19) or 2**32 -> 10**9
		h, l = bits.Div(h, l, _BD)
		d, s := dec{0, Word(l), Word(h), 0}.norm()
		// d should now have a single element with e shifted left
		ew := w * pow10(_WD-mag(w))
		// expected shift
		// _WD :   _WD  :  _WD  : ...
		// _WD : S + mag(w) + e : ...
		es := _WD*2 - (mag(w) + e) + _WD
		if len(d) > 1 || d[0] != Word(ew) || s != es {
			t.Fatalf("%ve%v => dec{0, %v, %v, 0}.norm() = %v, %v --- Expected [%d], %d",
				w, e, l, h, d, s, w, es)
		}
	}
}

func Test_dec_digits(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
	again:
		w := uint(rand.Uint64()) % _BD
		// ignore anything divisible by ten since mag(10) = 2 but dec{100000000...}.digits() = 1
		if w%10 == 0 {
			goto again
		}
		e := uint(rand.Intn(_WD + 1))
		h, l := bits.Mul(w, pow10(e))
		h, l = bits.Div(h, l, _BD)
		d, _ := dec{Word(l), Word(h)}.norm()
		if d.digits() != mag(w) {
			t.Fatalf("dec{%d}.digits() = %d, expected %d", d[0], d.digits(), mag(w))
		}
	}
}

func Test_mag(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
		n := uint(rand.Uint64())
		d := uint(0)
		for m := n; m != 0; m /= 10 {
			d++
		}
		if dd := mag(n); dd != d {
			t.Fatalf("mag(%d) = %d, expected %d", n, dd, d)
		}
	}
}

func Test_dec_setInt(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1000; i++ {
		ns := make([]byte, rand.Intn(100)+1)
		for i := 0; i < len(ns); i++ {
			ns[i] = '0' + byte(rand.Intn(10))
		}
		b, _ := new(big.Int).SetString(string(ns), 10)
		// remove trailing 0s
		ns = bytes.TrimLeft(ns, "0")
		prec := uint32(float64(b.BitLen())/ln2_10) + 1
		d, exp := dec{}.make((int(prec) + _WD - 1) / _WD).setInt(b)
		if exp != uint(len(ns)) {
			t.Fatalf("%s -> %v. Expected exponent %d, got %d.", ns, d, len(ns), exp)
		}
		b2 := new(big.Int)
		bd := new(big.Int).SetUint64(_BD)
		x := new(big.Int)
		for i := len(d) - 1; i >= 0; i-- {
			b2.Mul(b2, bd).Add(b2, x.SetUint64(uint64(d[i])))
		}
		shr := len(d)*_WD - int(exp)
		if shr > 0 {
			b2.Div(b2, x.SetUint64(uint64(pow10(uint(shr)))))
		} else {
			b2.Mul(b2, x.SetUint64(uint64(pow10(uint(-shr)))))
		}
		if b.Cmp(b2) != 0 {
			t.Fatalf("Got %s -> %v x 10**%d. Bad conversion back to Int: %s", b, d, exp, b2)
		}
	}
	b, _ := new(big.Int).SetString("12345678901234567890000000000000000000", 0)
	d, exp := dec{}.make(3).setInt(b)
	t.Log(d, exp)
}

func Test_add10VW(t *testing.T) {
	td := []struct {
		i dec
		x Word
		o dec
		c Word
		s uint
	}{
		{dec{_BD - 2, _BD - 1}, 2, nil, 1, 0},
		{dec{_BD - 2, _BD - 1}, 1, dec{_BD - 1, _BD - 1}, 0, 0},
		{dec{_BD - 2, _BD - 2}, 2, dec{_BD - 1}, 0, 0},
	}
	for i, d := range td {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			z := d.i
			c := add10VW(z, z, d.x)
			z, s := z.norm()
			ok := true
			if len(z) != len(d.o) {
				ok = false
			} else {
				for i := 0; i < len(z) && i < len(d.o); i++ {
					if z[i] != d.o[i] {
						ok = false
					}
				}
			}
			if !ok || s != d.s || c != d.c {
				t.Fatalf("addW failed: expected z = %v, s = %d, c = %d, got d = %v, s = %v, c = %v", d.o, d.s, d.c, z, s, c)
			}

		})
	}
}

func TestDec_digit(t *testing.T) {
	data := []struct {
		d dec
		n uint
		r uint
	}{
		{dec{123}, 0, 3},
		{dec{123}, 2, 1},
		{dec{123}, 3, 0},
		{dec{0, 1234567891234567891}, 37, 1},
		{dec{0, 1234567891234567891}, 36, 2},
		{dec{0, 1234567891234567891}, 38, 0},
	}
	for di, d := range data {
		t.Run(strconv.Itoa(di), func(t *testing.T) {
			if dig := d.d.digit(d.n); dig != d.r {
				t.Fatalf("%v.digit(%d) = %d, expected %d", d.d, d.n, dig, d.r)
			}
		})
	}
}

var (
	benchD dec
	benchU uint
)

func Benchmark_dec_norm(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	d := dec{}.make(10000)
	for i := range d {
		d[i] = Word(rand.Uint64()) % _BD
	}
	for i := 0; i < b.N; i++ {
		d[0] = Word(rand.Uint64()) % _BD
		d[len(d)-1] = Word(rand.Uint64()) % _BD
		benchD, benchU = d.norm()
	}
}

func Benchmark_mag(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		benchU = mag(uint(rand.Uint64()) % _BD)
	}
}

func Benchmark_dec_Digits(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	d := dec{}.make(10000)
	for i := range d {
		d[i] = Word(rand.Uint64()) % _BD
	}
	for i := 0; i < b.N; i++ {
		d[0] = Word(rand.Uint64()) % _BD
		d[len(d)-1] = Word(rand.Uint64()) % _BD
		benchU = d.digits()
	}
}
