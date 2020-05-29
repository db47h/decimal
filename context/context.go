// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package context provides IEEE-754 style contexts for Decimals.
//
// All factory functions of the form
//
//    func (c *Context) NewT(x T) *decimal.Decimal
//
// create a new decimal.Decimal set to the value of x, and rounded using c's
// precision and rounding mode.
//
// Operators that set a receiver z to function of other decimal arguments like:
//
//    func (c *Context) UnaryOp(z, x *decimal.Decimal) *decimal.Decimal
//    func (c *Context) BinaryOp(z, x, y *decimal.Decimal) *decimal.Decimal
//
// set z to the result of z.Op(args), rounded using the c's precision and
// rounding mode and return z.
//
// A Context catches NaN errors: if an operation generates a NaN, the operation
// will silently succeed with an undefined result. Further operations with the
// context will be no-ops (they simply return the receiver z) until
// (*Context).Err is called to check for errors.
//
// Although it does not exactly provide IEEE-754 NaNs, it provides a form of
// support for quiet NaNs.
package context

import (
	"errors"
	"math/big"

	"github.com/db47h/decimal"
)

const handleNaNs = true

// A Context is a wrapper around Decimals that facilitates management of
// rounding modes, precision and error handling.
type Context struct {
	prec uint32
	mode decimal.RoundingMode
	err  error
}

// New creates a new context with the given precision and rounding mode. If prec
// is 0, it will be set to decimal.DefaultRoundingMode.
func New(prec uint, mode decimal.RoundingMode) *Context {
	return new(Context).SetMode(mode).SetPrec(prec)
}

// Mode returns the rounding mode of c.
func (c *Context) Mode() decimal.RoundingMode {
	return c.mode
}

// Prec returns the mantissa precision of x in decimal digits.
// The result may be 0 for |x| == 0 and |x| == Inf.
func (c *Context) Prec() uint {
	return uint(c.prec)
}

// SetMode sets c's rounding mode to mode and returns c.
func (c *Context) SetMode(mode decimal.RoundingMode) *Context {
	c.mode = mode
	return c
}

// SetPrec sets c's precision to prec and returns c.
//
// If prec > MaxPrec, it is set to MaxPrec. If prec == 0, it is set to
// decimal.DefaultDecimalPrec.
func (c *Context) SetPrec(prec uint) *Context {
	// special case
	if prec == 0 {
		prec = decimal.DefaultDecimalPrec
	}
	// general case
	if prec > decimal.MaxPrec {
		prec = decimal.MaxPrec
	}
	c.prec = uint32(prec)
	return c
}

// New returns a new decimal.Decimal with value 0, precision and rounding mode set
// to c's precision and rounding mode.
func (c *Context) New() *decimal.Decimal {
	return new(decimal.Decimal).SetMode(c.mode).SetPrec(uint(c.prec))
}

// NewInt returns a new *decimal.Decimal set to the (possibly rounded) value of
// x.
func (c *Context) NewInt(x *big.Int) *decimal.Decimal {
	return c.New().SetInt(x)
}

// NewInt64 returns a new *decimal.Decimal set to the (possibly rounded) value
// of x.
func (c *Context) NewInt64(x int64) *decimal.Decimal {
	return c.New().SetInt64(x)
}

// NewUint64 returns a new *decimal.Decimal set to the (possibly rounded) value
// of x.
func (c *Context) NewUint64(x uint64) *decimal.Decimal {
	return c.New().SetUint64(x)
}

// NewFloat returns a new *decimal.Decimal set to the (possibly rounded) value
// of x.
func (c *Context) NewFloat(x *big.Float) *decimal.Decimal {
	return c.New().SetFloat(x)
}

// NewFloat64 returns a new *decimal.Decimal set to the (possibly rounded) value
// of x.
func (c *Context) NewFloat64(x float64) *decimal.Decimal {
	return c.New().SetFloat64(x)
}

// NewRat returns a new *decimal.Decimal set to the (possibly rounded) value of
// x.
func (c *Context) NewRat(x *big.Rat) *decimal.Decimal {
	return c.New().SetRat(x)
}

// NewFromString returns a new Decimal with the value of s and a boolean
// indicating success. s must be a floating-point number of the same format as
// accepted by (*decimal.Decimal).Parse, with base argument 0. The entire string
// (not just a prefix) must be valid for success. If the operation failed, the
// value of d is undefined but the returned value is nil. d's precision and
// rounding mode are set to c's precision and rounding mode.
func (c *Context) NewString(s string) (d *decimal.Decimal, success bool) {
	return c.New().SetString(s)
}

// ParseDecimal is like d.Parse(s, base) with d set to the given precision and rounding mode.
func (c *Context) ParseDecimal(s string, base int) (f *decimal.Decimal, b int, err error) {
	return decimal.ParseDecimal(s, base, uint(c.prec), c.mode)
}

// Err returns the first error encountered since the last call to Err and clears
// the error state.
func (c *Context) Err() (err error) {
	err = c.err
	c.err = nil
	return
}

// Round sets z's to the value of x and returns z rounded using c's precision
// and rounding mode.
func (c *Context) Round(z, x *decimal.Decimal) *decimal.Decimal {
	if handleNaNs {
		if c.err != nil {
			return z
		}
	}
	return c.apply(z.Copy(x))
}

// apply applies c's precision and rounding mode to z and returns z.
func (c *Context) apply(z *decimal.Decimal) *decimal.Decimal {
	z.SetMode(c.mode)
	if z.Prec() != uint(c.prec) {
		z.SetPrec(0).SetPrec(uint(c.prec))
	}
	return z
}

// Add sets z to the rounded sum x+y and returns z.
func (c *Context) Add(z, x, y *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).Add(x, y)
}

// Sub sets z to the rounded difference x+y and returns z.
func (c *Context) Sub(z, x, y *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).Sub(x, y)
}

// FMA sets z to x * y + u, computed with only one rounding. That is, FMA
// performs the fused multiply-add of x, y, and u.
func (c *Context) FMA(z, x, y, u *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).FMA(x, y, u)
}

// Mul sets z to the rounded product x×y and returns z.
func (c *Context) Mul(z, x, y *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).Mul(x, y)
}

// Quo sets z to the rounded quotient x/y and returns z.
func (c *Context) Quo(z, x, y *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).Quo(x, y)
}

// Neg sets z to the (possibly rounded) value of x with its sign negated,
// and returns z.
func (c *Context) Neg(z, x *decimal.Decimal) *decimal.Decimal {
	if handleNaNs {
		if c.err != nil {
			return z
		}
	}
	return c.apply(z).Neg(x)
}

// Abs sets z to the (possibly rounded) value |x| (the absolute value of x)
// and returns z.
func (c *Context) Abs(z, x *decimal.Decimal) *decimal.Decimal {
	if handleNaNs {
		if c.err != nil {
			return z
		}
	}
	return c.apply(z).Abs(x)
}

// Sqrt sets z to the rounded square root of x, and returns z.
//
func (c *Context) Sqrt(z, x *decimal.Decimal) (r *decimal.Decimal) {
	if handleNaNs {
		if c.err != nil {
			return z
		}
		defer func() {
			if err := recover(); err != nil {
				if !errors.As(err.(error), &c.err) {
					panic(err)
				}
				r = z
			}
		}()
	}
	return c.apply(z).Sqrt(x)
}
