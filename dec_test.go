package decimal

import (
	"math/big"
	"math/bits"
	"math/rand"
	"testing"
	"time"
)

func Test_decNorm(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
	again:
		w := uint(rand.Uint64()) % _BD
		if w%10 == 0 {
			goto again
		}
		e := uint(rand.Intn(_WD + 1))
		h, l := bits.Mul(w, pow10(e))
		// convert h, l from base _B (2**64) to base _BD (10**19) or 2**32 -> 10**9
		h, l = bits.Div(h, l, _BD)
		d, s := dec{Word(l), Word(h)}.norm()
		if len(d) > 1 || d[0] != Word(w) || s != e {
			t.Fatalf("%ve%v => dec{%v, %v}.norm() = %v, %v --- Expected [%d], %d",
				w, e, l, h, d, s, w, e)
		}
	}
}

func Test_decDigits(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
		n := uint(rand.Uint64())
		d := uint(0)
		for m := n; m != 0; m /= 10 {
			d++
		}
		if dd := decDigits(n); dd != d {
			t.Fatalf("decDigits(%d) = %d, expected %d", n, dd, d)
		}
	}
}

func Test_decShr10(t *testing.T) {
	d := dec{1234, 0, 1234567890123456789}
	d, r, tnz := d.shr10(20)
	t.Log(d, r, tnz)
}

func Test_decSetInt(t *testing.T) {
	b, _ := new(big.Int).SetString("123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", 0)
	prec := uint32(float64(b.BitLen())/ln2_10) + 1
	d := dec{}.make((int(prec) + _WD - 1) / _WD)
	d = d.setInt(b)
	t.Log(d, len(d))
}

var (
	benchD dec
	benchU uint
)

func Benchmark_decNorm(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		w := uint(rand.Uint64()) % _BD
		e := uint(rand.Intn(_WD))
		h, l := w/pow10(_WD-e), (w%pow10(_WD-e))*pow10(e)
		benchD, benchU = dec{Word(l), Word(h)}.norm()
	}
}

func Benchmark_decDigits(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		benchU = decDigits(uint(rand.Uint64()) % _BD)
	}
}

func BenchmarkDecDigits(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		benchU = dec{Word(rand.Uint64()) % _BD}.digits()
	}
}
