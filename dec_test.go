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
		w := uint(rand.Uint64()) % _BD
		e := uint(rand.Intn(_WD + 1))
		h, l := bits.Mul(w, pow10(e))
		h, l = bits.Div(h, l, _BD)
		d := dec{Word(l), Word(h)}.norm()
		// adjust e if w == 0 or w%10 == 0
		if w == 0 {
			e = 0
		} else {
			e += decTrailingZeros(w)
		}
		if d.ntz() != e {
			t.Fatalf("dec{%v}.ntz() = %d, expected %d", d, d.ntz(), e)
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
