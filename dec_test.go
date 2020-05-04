package decimal

import (
	"math/bits"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"
)

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
		d := dec{Word(l), Word(h)}.norm()
		dnorm(d)
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

// TODO(db47h): remove this function
func Test_dec_setInt(t *testing.T) {
	// // TODO(db47h): next step
	// b, _ := new(big.Int).SetString("12345678901234567890", 0)
	// d, exp := dec{}.make(3).setInt(b)
	// t.Log(d, exp)
	// t.Log(string(dtoa(d, 10)))
}

func Test_add10VW(t *testing.T) {
	td := []struct {
		i dec
		x Word
		o dec
		c Word
		s int64
	}{
		{dec{_BD - 2, _BD - 1}, 2, dec{}, 1, 0},
		{dec{_BD - 2, _BD - 1}, 1, dec{_BD - 1, _BD - 1}, 0, 0},
		{dec{_BD - 2, _BD - 2}, 2, dec{0, _BD - 1}, 0, 0},
	}
	for i, d := range td {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			z := d.i
			c := add10VW(z, z, d.x)
			var s int64
			z = z.norm()
			if len(z) > 0 {
				s = dnorm(z)
			}
			if !reflect.DeepEqual(z, d.o) || s != d.s || c != d.c {
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

// func Benchmark_dnorm(b *testing.B) {
// 	rand.Seed(0xdeadbeefbadf00d)
// 	d := dec{}.make(10000)
// 	for i := range d {
// 		d[i] = Word(rand.Uint64()) % _BD
// 	}
// 	for i := 0; i < b.N; i++ {
// 		d[0] = Word(rand.Uint64()) % _BD
// 		d[len(d)-1] = Word(rand.Uint64()) % _BD
// 		benchD, benchU = d.dnorm()
// 	}
// }

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
