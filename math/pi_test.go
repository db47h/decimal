package math

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/db47h/decimal"
)

func Test_pi(t *testing.T) {
	pi100k := new(decimal.Decimal)

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

	// make sure that _pi is ok
	_piOK := new(decimal.Decimal).SetPrec(_pi.Prec()).Set(pi100k)
	if _pi.Cmp(_piOK) != 0 {
		t.Fatalf("Bad value for _pi\nGot : %g\nWant: %g", _pi, _piOK)
	}

	// test random pi values
	// don't go overboard with the precision. It takes an AMD FX6300 60s to compute 50K digits of pi.
	maxDigits := int(pi100k.Prec() / 2)
	if testing.Short() {
		maxDigits = 4000
	}
	for cpu := 0; cpu < runtime.GOMAXPROCS(-1); cpu++ {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 10; i++ {
				// random prec in [maxDigits/2 + 1, maxDigits]
				// prec := uint(rand.Intn(maxDigits/2) + maxDigits/2 + 1)
				prec := uint(maxDigits)
				x := new(decimal.Decimal).SetPrec(prec).Set(pi100k)
				z := new(decimal.Decimal).SetPrec(prec)
				if pi(z).Cmp(x) != 0 {
					t.Fatalf("Bad π value for %d digits\nGot : %g\nWant: %g", prec, z, x)
				}
			}
		})
	}
}

func Benchmark_Pi(b *testing.B) {
	z := new(decimal.Decimal).SetPrec(256)
	for i := 0; i < b.N; i++ {
		pi(z)
	}
}
