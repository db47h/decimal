package decimal

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"
)

var intData = []struct {
	s  string
	p  uint
	d  dec
	pr uint
	e  int32
}{
	{"1234567890123456789_0123456789012345678_9012345678901234567_8901234567890123456_78901234567890", 0,
		dec{7890123456789000000, 8901234567890123456, 9012345678901234567, 123456789012345678, 1234567890123456789},
		90, 90},
	{"1235", 3, dec{1240000000000000000}, 3, 4},
	{"1245", 3, dec{1240000000000000000}, 3, 4},
	{"12451", 3, dec{1250000000000000000}, 3, 5},
}

func TestDecimal_SetInt(t *testing.T) {
	for i, td := range intData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			b, _ := new(big.Int).SetString(td.s, 0)
			d := new(Decimal).SetMode(ToNearestEven).SetPrec(td.p).SetInt(b)
			if !reflect.DeepEqual(td.d, d.mant) {
				t.Fatalf("\nexpected mantissa %v\n              got %v", td.d, d.mant)
			}
			if td.pr != d.Prec() {
				t.Fatalf("\nexpected precision %v\n               got %v", td.pr, d.Prec())
			}
			if td.e != d.exp {
				t.Fatalf("\nexpected exponent %v\n              got %v", td.p, d.Prec())
			}
		})
	}
}

func TestDecimal_SetString(t *testing.T) {
	for i, td := range intData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			d, _ := new(Decimal).SetMode(ToNearestEven).SetPrec(td.p).SetString(td.s)
			if !reflect.DeepEqual(td.d, d.mant) {
				t.Fatalf("\nexpected mantissa %v\n              got %v", td.d, d.mant)
			}
			if td.pr != d.Prec() {
				t.Fatalf("\nexpected precision %v\n               got %v", td.pr, d.Prec())
			}
			if td.e != d.exp {
				t.Fatalf("\nexpected exponent %v\n              got %v", td.p, d.Prec())
			}
		})
	}
}
