package math_test

import (
	"strconv"
	"testing"

	"github.com/db47h/decimal"
	"github.com/db47h/decimal/math"
)

func Benchmark_Expm1(b *testing.B) {
	for _, prec := range []uint{34, 100, 200, 500, 1000} {
		b.Run(strconv.Itoa(int(prec)), func(b *testing.B) {
			z := new(decimal.Decimal).SetPrec(prec)
			x := decimal.NewDecimal(373, -2)
			for i := 0; i < b.N; i++ {
				math.Expm1(z, x)
			}
		})
	}
}

func Test_Expm1(t *testing.T) {
	td := []struct {
		xm  int64
		xe  int
		res string
	}{
		{1, 0,
			"1.7182818284590452353602874713526624977572470936999595749669676277240766303535475945713821785251664274274"},
		{1, -33,
			"1.0000000000000000000000000000000005000000000000000000000000000000001666666666666666666666666666666667083e-33"},
		{-1, -33,
			"-9.99999999999999999999999999999999500000000000000000000000000000000166666666666666666666666666666666625e-34"},
		{-1, 200, "0"},
		{-1, -200, "-1e-200"},
		{1, 8,
			"1.5499767466484265044184585433334652127927121152116450579814179720474638638555243371803334216641703770775e43429448"},
		{1, 10, "+Inf"},
		{0, 0, "0"},
	}

	for _, d := range td {
		x := decimal.NewDecimal(d.xm, d.xe)
		t.Run(x.Text('e', -1), func(t *testing.T) {
			z := new(decimal.Decimal)
			r := new(decimal.Decimal)
			for _, prec := range []uint{100, 4, 34, 50, 75} {
				math.Expm1(z.SetPrec(prec), x)
				_, _, err := r.SetPrec(prec).Parse(d.res, 10)
				if err != nil {
					t.Fatal(err)
				}
				if z.Cmp(r) == 0 {
					continue
				}
				t.Fatalf("Error at precision %d: Expected:\n%g\nGot:\n%g", prec, r, z)
			}
		})
	}
}
