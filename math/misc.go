package math

import "github.com/db47h/decimal"

// constants
var (
	one     = new(decimal.Decimal).SetUint64(1)
	two     = new(decimal.Decimal).SetUint64(2)
	four    = new(decimal.Decimal).SetUint64(4)
	half    = decimal.NewDecimal(5, -1)
	quarter = decimal.NewDecimal(25, -2)
)

func precWords(prec uint) uint { return (prec+(decimal.DigitsPerWord-1))/decimal.DigitsPerWord + 1 }

// dec pre-allocates storage for a decimal of precision prec with one
// additional word of storage and returns the new decimal with its precision set
// to prec.
func dec(prec uint) *decimal.Decimal {
	return new(decimal.Decimal).SetPrec(prec).SetBitsExp(make([]decimal.Word, 0, precWords(prec)), 0)
}
