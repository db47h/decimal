package decimal

import (
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
		benchU = decDigits(uint(rnd.Uint64()) % _BD)
	}
}

func rnd10W() Word {
	return Word(rnd.Uint64() % _BD)
}

func rnd10V(n int) []Word {
	v := make([]Word, n)
	for i := range v {
		v[i] = rnd10W()
	}
	return v
}
