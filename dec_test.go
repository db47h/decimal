package decimal

import (
	"math/bits"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestDec_ntz(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
	again:
		w := uint(rand.Uint64()) % _BD
		// TODO: WTF? ignore anything divisible by ten since decDigits(10) = 2 but dec{100000000...}.digits() = 1
		if w%10 == 0 {
			goto again
		}
		e := uint(rand.Intn(_WD + 1))
		h, l := bits.Mul(w, pow10(e))
		h, l = bits.Div(h, l, _BD)
		d := dec{Word(l), Word(h)}.norm()
		if d.ntz() != e {
			t.Fatalf("dec{%v}.digits() = %d, expected %d", d, d.ntz(), e)
		}
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

func BenchmarkDec_Digits(b *testing.B) {
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
