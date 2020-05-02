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
	d := dec{}.make(2)[:0]
	for i := 0; i < b.N; i++ {
		w := uint(rand.Uint64()) % _BD
		e := uint(rand.Intn(_WD))
		h, l := w/pow10(_WD-e), (w%pow10(_WD-e))*pow10(e)
		d = d.make(2)
		d[0], d[1] = Word(l), Word(h)
		benchD, benchU = d.norm()
	}
}

func Benchmark_mag(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		benchU = mag(uint(rand.Uint64()) % _BD)
	}
}

func Benchmark_decDigits(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	d := dec{}.make(1)
	for i := 0; i < b.N; i++ {
		w := uint(rand.Uint64()) % _BD
		e := uint(rand.Intn(_WD))
		h, l := bits.Mul(w, pow10(e))
		_, l = bits.Div(h, l, _BD)
		d[0] = Word(l)
		benchU = d.digits()
	}
}
