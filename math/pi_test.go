package math

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
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
				if computePi(z.SetPrec(prec)).Cmp(x) != 0 {
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
	y := computePi(new(decimal.Decimal).SetPrec(130641))
	if x.Cmp(y) != 0 {
		xs := x.Text('g', -1)
		ys := x.Text('g', -1)
		t.Fatalf("pi(130641) produced an incorrect value. The last 10 digits are:\n%s\nExpected:%s\n",
			ys[len(ys)-10:],
			xs[len(xs)-10:])
	}
}

func Benchmark_computePi(b *testing.B) {
	z := new(decimal.Decimal).SetPrec(1200)
	for i := 0; i < b.N; i++ {
		computePi(z)
	}
}
