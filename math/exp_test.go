package math_test

import (
	"strconv"
	"testing"

	"github.com/db47h/decimal"
	"github.com/db47h/decimal/math"
)

const maxPrec = 15000

// init _pi and _log10 to a sufficiently high precision
var _ = math.Pi(new(decimal.Decimal).SetPrec(maxPrec + decimal.DigitsPerWord*2))
var _ = math.Log(new(decimal.Decimal).SetPrec(maxPrec), decimal.NewDecimal(1, 1))

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
		x   string
		res string
	}{
		{"0", "0"},
		{"-Inf", "-1"},
		{"+Inf", "+Inf"},
		{"1",
			"1.7182818284590452353602874713526624977572470936999595749669676277240766303535475945713821785251664274274"},
		{"1e-33",
			"1.0000000000000000000000000000000005000000000000000000000000000000001666666666666666666666666666666667083e-33"},
		{"-1e-33",
			"-9.99999999999999999999999999999999500000000000000000000000000000000166666666666666666666666666666666625e-34"},
		{"-1e200", "0"},
		{"-1e-200", "-1e-200"},
		{"1e8",
			"1.5499767466484265044184585433334652127927121152116450579814179720474638638555243371803334216641703770775e43429448"},
		{"1e10", "+Inf"},
		{"1e-2",
			"1.0050167084168057542165456902860033807362201524292515164404031254374190731323852253210417020805424644822e-2"},
		{"9e-2",
			"9.4174283705210357872897623544886011846519908747085113495537273829677943694135052107150933181696135377161e-2"},
		{"-1e-2",
			"-9.950166250831946426094022819963442227920918746162533116121254706852272831254704981784469220616189007036e-3"},
		{"-9e-2",
			"-8.606881472877181325264645350047938978941480517319022333641173457709159144983607060549316409923275854118e-2"},
	}

	for _, d := range td {
		x, _, err := new(decimal.Decimal).Parse(d.x, 0)
		if err != nil {
			t.Fail()
			t.Logf("failed to parse %q: %s", d.x, err)
			continue
		}
		t.Run(x.Text('e', -1), func(t *testing.T) {
			z := new(decimal.Decimal)
			r := new(decimal.Decimal)
			for _, prec := range []uint{100, 4, 34, 50, 75} {
				math.Expm1(z.SetPrec(prec), x)
				if _, _, err := r.SetPrec(prec).Parse(d.res, 10); err != nil {
					t.Fatalf("failed to parse result: %s", err)
				}
				if z.Cmp(r) == 0 {
					continue
				}
				t.Fatalf("Error at precision %d: Expected:\n%g\nGot:\n%g", prec, r, z)
			}
		})
	}
}
