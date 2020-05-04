package decimal

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"
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

func BenchmarkDecDigits(b *testing.B) {
	rand.Seed(0xdeadbeefbadf00d)
	for i := 0; i < b.N; i++ {
		benchU = decDigits(uint(rand.Uint64()) % _BD)
	}
}
