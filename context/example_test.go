package context_test

import (
	"errors"
	"fmt"

	"github.com/db47h/decimal"
	"github.com/db47h/decimal/context"
)

var _four = new(decimal.Decimal).SetPrec(1).SetInt64(-4)
var two = new(decimal.Decimal).SetPrec(1).SetInt64(2)

// solve solves the quadratic equation ax² + bx + c = 0, using ctx's rounding
// mode and precision. It can fail with various combinations of inputs, for
// example a = 0, b = 2, c = -3 will result in dividing zero by zero when
// computing x0. So we need to check errors.
func solve(ctx context.Context, a, b, c *decimal.Decimal) (x0, x1 *decimal.Decimal, err error) {
	d := ctx.New()
	// compute discriminant
	ctx.Mul(d, a, _four) // d = a × -4
	ctx.Mul(d, d, c)     //     × c
	ctx.FMA(d, b, b, d)  //     + b × b
	if err != nil {
		return nil, nil, fmt.Errorf("error computing discriminant: %w", err)
	}
	if d.Sign() < 0 {
		return nil, nil, errors.New("no real roots")
	}
	// d = √d
	ctx.Sqrt(d, d)
	twoA := ctx.Mul(ctx.New(), a, two)
	negB := ctx.Neg(ctx.New(), b)

	x0 = ctx.Add(ctx.New(), negB, d)
	ctx.Quo(x0, x0, twoA)
	x1 = ctx.Sub(ctx.New(), negB, d)
	ctx.Quo(x1, x1, twoA)

	if err = ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("error computing roots: %w", err)
	}
	return
}

// Example demonstrates various features of Contexts.
func Example() {
	ctx := context.New(0, decimal.ToNearestEven)
	a, b, c := ctx.NewInt64(1), ctx.NewInt64(2), ctx.NewInt64(-3)
	x0, x1, err := solve(ctx, a, b, c)
	if err != nil {
		fmt.Printf("failed to solve %g×x²%+gx%+g: %v\n", a, b, c, err)
		return
	}
	fmt.Printf("roots of %g×x²%+gx%+g: %g, %g\n", a, b, c, x0, x1)

	a = ctx.New() // zero
	x0, x1, err = solve(ctx, a, b, c)
	if err != nil {
		// obviously, our solve() algorithm cannot handle a == 0
		fmt.Printf("failed to solve %g×x²%+gx%+g: %v\n", a, b, c, err)
		return
	}
	fmt.Printf("roots of %g×x²%+gx%+g: %g, %g\n", a, b, c, x0, x1)
	//
	// Output:
	// roots of 1×x²+2x-3: 1, -3
	// failed to solve 0×x²+2x-3: error computing roots: division of zero by zero or infinity by infinity
}
