package math

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/db47h/decimal"
)

var pi100k *decimal.Decimal

func init() {
	pi100k = new(decimal.Decimal)

	f, err := os.Open("testdata/pi100000.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sz, err := f.Seek(0, io.SeekEnd)
	if err == nil {
		pi100k.SetPrec(uint(sz - 3)) // remove decimal . and last 2 digits
		_, _ = f.Seek(0, io.SeekStart)
	}
	_, err = fmt.Fscanf(f, "%g", pi100k)
	if err != nil {
		panic(err)
	}
}

func Test_computePi(t *testing.T) {
	// init _pi with defaults
	Pi(new(decimal.Decimal))
	// make sure that _pi is ok
	_piOK := new(decimal.Decimal).SetPrec(_pi.Prec()).Set(pi100k)
	if _pi.Cmp(_piOK) != 0 {
		t.Fatalf("Bad value for _pi\nGot : %g\nWant: %g", _pi, _piOK)
	}

	// test random pi values
	maxDigits := int(pi100k.Prec())
	if testing.Short() {
		maxDigits = 20000
	}
	seed := time.Now().UnixNano()
	rand.Seed(seed)
	cpus := runtime.GOMAXPROCS(-1)
	for cpu := 0; cpu < cpus; cpu++ {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			z := new(decimal.Decimal).SetPrec(uint(maxDigits))
			x := new(decimal.Decimal).SetPrec(uint(maxDigits))
			for i := 0; i < 10; i++ {
				// random prec in [maxDigits/2 + 1, maxDigits]
				prec := uint(rand.Intn(maxDigits/2) + maxDigits/2 + 1)
				x.SetPrec(prec).Set(pi100k)
				if __pi(z.SetPrec(prec)).Cmp(x) != 0 {
					t.Fatalf("SEED %x, bad π value for %d digits\nGot : %g\nWant: %g", seed, prec, z, x)
				}
			}
		})
	}
}

func Test_pi130641(t *testing.T) {
	// digits 130639... are 09050000... This may cause issues with
	// decimal.ToNearestEven if we do not compute enough extra digits:
	// pi(130641) may end with 090 instead of 091.
	if testing.Short() {
		t.SkipNow()
	}
	x := new(decimal.Decimal).SetPrec(130641).Set(pi100k)
	y := __pi(new(decimal.Decimal).SetPrec(130641))
	if x.Cmp(y) != 0 {
		xs := x.Text('g', -1)
		ys := x.Text('g', -1)
		t.Fatalf("pi(130641) produced an incorrect value. The last 10 digits are:\n%s\nExpected:%s\n",
			ys[len(ys)-10:],
			xs[len(xs)-10:])
	}
}

func Benchmark_pi(b *testing.B) {
	z := new(decimal.Decimal).SetPrec(1200)
	for i := 0; i < b.N; i++ {
		__pi(z)
	}
}

func Benchmark_Expm1(b *testing.B) {
	for _, prec := range []uint{34, 100, 200, 500, 1000} {
		b.Run(strconv.Itoa(int(prec)), func(b *testing.B) {
			z := new(decimal.Decimal).SetPrec(prec)
			x := decimal.NewDecimal(-373, 27)
			for i := 0; i < b.N; i++ {
				Expm1(z, x)
			}
		})
	}
}

func Benchmark_Log(b *testing.B) {
	z := new(decimal.Decimal).SetPrec(1200)
	x := decimal.NewDecimal(42, 0)
	for i := 0; i < b.N; i++ {
		Log(z, x)
	}
}

func Test_expm1T(t *testing.T) {
	const prec = 100 // decimal.DefaultDecimalPrec
	x := decimal.NewDecimal(-1, -100)
	z := new(decimal.Decimal).SetPrec(prec)
	Expm1(z, x)
	t.Logf("e^x-1 : %g", z)
	// x.Quo(x, two)
	// z.Add(expm1T(z.SetPrec(prec+decimal.DefaultDecimalPrec), x), one)
	// // z.Mul(z, z)
	// pow(z, z, 2)
	// t.Logf("e^x : %g", z.SetPrec(prec))

	// x.Quo(x, two)
	// z.Add(expm1T(z.SetPrec(prec+decimal.DefaultDecimalPrec), x), one)
	// z.Mul(z, z)
	// z.Mul(z, z)
	// t.Logf("e^x : %g", z.SetPrec(prec))

	// x.Quo(x, two)
	// z.Add(expm1T(z.SetPrec(prec+decimal.DefaultDecimalPrec), x), one)
	// z.Mul(z, z)
	// z.Mul(z, z)
	// z.Mul(z, z)
	// t.Logf("e^x : %g", z.SetPrec(prec))

	// e^(14×10^5) = (e^14)^10^5
	// x = decimal.NewDecimal(14, -2)
	// z.Add(expm1T(z.SetPrec(prec+decimal.DefaultDecimalPrec), x), one)
	// pow(z, z, 10000000)
	// t.Logf("e^x : %g", z.SetPrec(prec))
}

// z = Exp(x×10^n)
// Log(z) = x×10^n
// Log(z) / 10^n = x

// z = x^n
// Log(z) = n Log(x)
// z = Exp(n×Log(x))

// 2.7182818284590452353602874713526624977572470936999595749669676277240766303535475945713821785251664274274663919320030599218174135966290435729003342952605956307381323286279434907632338298807531952510190

// 2.7182818284590452353602874713526624977572470936999595749669676277240766303535606149554353915016794760416053540561963396558067135924747104121077290922735268587467124087345821321753200729805272128892931

// 1.648721270700128146848650787814163571653776100710148011575079311640661021194215608632776

// 1.648721270700128146848650787814163571653776100710148011575079311640661021194215608632776
