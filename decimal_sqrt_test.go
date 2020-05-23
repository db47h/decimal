// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package decimal

import (
	"fmt"
	"math"
	"testing"
)

func TestDecimalSqrt(t *testing.T) {
	for _, test := range []struct {
		x    string
		want string
	}{
		// Test values were generated on Wolfram Alpha using query
		//   'sqrt(N) to 350 digits'
		// 350 decimal digits give up to 1000 binary digits.
		{"0.03125", "0.17677669529663688110021109052621225982120898442211850914708496724884155980776337985629844179095519659187673077886403712811560450698134215158051518713749197892665283324093819909447499381264409775757143376369499645074628431682460775184106467733011114982619404115381053858929018135497032545349940642599871090667456829147610370507757690729404938184321879"},
		{"0.125", "0.35355339059327376220042218105242451964241796884423701829416993449768311961552675971259688358191039318375346155772807425623120901396268430316103037427498395785330566648187639818894998762528819551514286752738999290149256863364921550368212935466022229965238808230762107717858036270994065090699881285199742181334913658295220741015515381458809876368643757"},
		{"0.5", "0.70710678118654752440084436210484903928483593768847403658833986899536623923105351942519376716382078636750692311545614851246241802792536860632206074854996791570661133296375279637789997525057639103028573505477998580298513726729843100736425870932044459930477616461524215435716072541988130181399762570399484362669827316590441482031030762917619752737287514"},
		{"2.0", "1.4142135623730950488016887242096980785696718753769480731766797379907324784621070388503875343276415727350138462309122970249248360558507372126441214970999358314132226659275055927557999505011527820605714701095599716059702745345968620147285174186408891986095523292304843087143214508397626036279952514079896872533965463318088296406206152583523950547457503"},
		{"3.0", "1.7320508075688772935274463415058723669428052538103806280558069794519330169088000370811461867572485756756261414154067030299699450949989524788116555120943736485280932319023055820679748201010846749232650153123432669033228866506722546689218379712270471316603678615880190499865373798593894676503475065760507566183481296061009476021871903250831458295239598"},
		{"4.0", "2.0"},

		{"1e512", "1e256"},
		{"4e1024", "2e512"},
		{"9e2048", "3e1024"},

		{"1e-1024", "1e-512"},
		{"4e-2048", "2e-1024"},
		{"9e-4096", "3e-2048"},
	} {
		for _, prec := range []uint{9, 16, 19, 34, 50, 100, 150, 200, 250, 300, 350} {
			x := new(Decimal).SetPrec(prec)
			x.Parse(test.x, 10)

			got := new(Decimal).SetPrec(prec).Sqrt(x)
			want := new(Decimal).SetPrec(prec)
			want.Parse(test.want, 10)
			if got.Cmp(want) != 0 {
				t.Errorf("prec = %d, Sqrt(%v) =\ngot  %g;\nwant %g",
					prec, test.x, got, want)
			}

			// Square test.
			// If got holds the square root of x to precision p, then
			//   got = √x + k
			// for some k such that |k| < 10**(-p). Thus,
			//   got² = (√x + k)² = x + 2k√n + k²
			// and the error must satisfy
			//   err = |got² - x| ≈ | 2k√n | < 10**(-p+1)*√n
			// Ignoring the k² term for simplicity.

			// err = |got² - x|
			// (but do intermediate steps with 9 guard digits to
			// avoid introducing spurious rounding-related errors)
			sq := new(Decimal).SetPrec(prec+9).Mul(got, got)
			diff := new(Decimal).Sub(sq, x)
			err := diff.Abs(diff).SetPrec(prec)

			// maxErr = 10**(-p+1)*√x
			one := new(Decimal).SetPrec(prec).SetInt64(1)
			maxErr := new(Decimal).Mul(new(Decimal).SetMantExp(one, -int(prec)+1), got)

			if err.Cmp(maxErr) >= 0 {
				t.Errorf("prec = %d, Sqrt(%v) =\ngot err  %g;\nwant maxErr %g",
					prec, test.x, err, maxErr)
			}
		}
	}
}

func TestDecimalSqrtSpecial(t *testing.T) {
	for _, test := range []struct {
		x    *Decimal
		want *Decimal
	}{
		{new(Decimal).SetFloat64(+0), new(Decimal).SetFloat64(+0)},
		{new(Decimal).SetFloat64(-0), new(Decimal).SetFloat64(-0)},
		{new(Decimal).SetFloat64(math.Inf(+1)), new(Decimal).SetFloat64(math.Inf(+1))},
	} {
		got := new(Decimal).Sqrt(test.x)
		if got.neg != test.want.neg || got.form != test.want.form {
			t.Errorf("Sqrt(%v) = %v (neg: %v); want %v (neg: %v)",
				test.x, got, got.neg, test.want, test.want.neg)
		}
	}

}

// Benchmarks

func BenchmarkDecimalSqrt(b *testing.B) {
	for _, prec := range []uint{19, 38, 76, 3e2, 3e3, 3e4, 3e5} {
		x := NewDecimal(2, 0)
		z := new(Decimal).SetPrec(prec)
		b.Run(fmt.Sprintf("%v", prec), func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				z.Sqrt(x)
			}
		})
	}
}
