package decimal

import (
	"math/bits"
	"reflect"
	"strconv"
	"testing"
)

func TestAdd10VW(t *testing.T) {
	td := []struct {
		i dec
		x Word
		o dec
		c Word
		s int64
	}{
		{dec{_DMax - 1, _DMax}, 2, dec{}, 1, 0},
		{dec{_DMax - 1, _DMax}, 1, dec{_DMax, _DMax}, 0, 0},
		{dec{_DMax - 1, _DMax - 1}, 2, dec{0, _DMax}, 0, 0},
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

func TestDivWDB(t *testing.T) {
	h, l := rnd10W(), Word(rnd.Uint64())
	for i := 0; i < 1e7; i++ {
		q, r := divWDB(h, l)
		qq, rr := bits.Div(uint(h), uint(l), _DB)
		if q != Word(qq) || r != Word(rr) {
			t.Fatalf("Got (%d,%d)/_DB = %d, %d. Expected %d %d", h, l, q, r, qq, rr)
		}
	}
}

var benchH, benchL Word

func BenchmarkDivWDB_bits(b *testing.B) {
	h, l := rnd10W(), Word(rnd.Uint64())
	for i := 0; i < b.N; i++ {
		h, l := bits.Div(uint(h), uint(l), _DB)
		benchH, benchL = Word(h), Word(l)
	}
}

func BenchmarkDivWDB_mul(b *testing.B) {
	h, l := rnd10W(), Word(rnd.Uint64())
	for i := 0; i < b.N; i++ {
		benchH, benchL = divWDB(h, l)
	}
}
