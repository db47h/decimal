package math_test

import (
	"strconv"
	"testing"

	"github.com/db47h/decimal"
	"github.com/db47h/decimal/math"
)

const maxPrec = 15000

// init _pi and _log10 to a sufficiently high precision
// var _ = math.Pi(new(decimal.Decimal).SetPrec(maxPrec + decimal.DigitsPerWord*2))
// var _ = math.Log(new(decimal.Decimal).SetPrec(maxPrec), decimal.NewDecimal(1, 1))

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
			"1.718281828459045235360287471352662497757247093699959574966967627724076630353547594571382178525166427427"},
		{"1e-33",
			"1.000000000000000000000000000000000500000000000000000000000000000000166666666666666666666666666666666708e-33"},
		{"0.1e-34",
			"1.000000000000000000000000000000000005000000000000000000000000000000000016666666666666666666666666667e-35"},
		{"9e-33",
			"9.000000000000000000000000000000040500000000000000000000000000000121500000000000000000000000000000273375e-33"},
		{"-1e-33",
			"-9.99999999999999999999999999999999500000000000000000000000000000000166666666666666666666666666666666625e-34"},
		{"-1e200", "-1"},
		{"-1e10", "-1"},
		{"-1e9", "-1"},
		{"1e9",
			"8.002981770660972533041909374365000688782314997176374565356445473341386965534987717522905014823536972023e+434294481"},
		{"5e9", "+Inf"},
		{"-1e8", "-1"},
		{"1e-100", "1e-100"},
		{"-1e-200", "-1e-200"},
		{"1e8",
			"1.549976746648426504418458543333465212792712115211645057981417972047463863855524337180333421664170377077e43429448"},
		{"1e10", "+Inf"},
		{"1e-2",
			"1.005016708416805754216545690286003380736220152429251516440403125437419073132385225321041702080542464482e-2"},
		{"9e-2",
			"9.417428370521035787289762354488601184651990874708511349553727382967794369413505210715093318169613537716e-2"},
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
