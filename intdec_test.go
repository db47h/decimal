// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the IntDec type used for testing Decimal operations
// via an independent (albeit slower) representation for decimal floating-point
// numbers.

package decimal

import (
	"math/big"
	"strconv"
	"strings"
	"testing"
)

// IDec is a decimal floating-point number. Its value is mant x 10**exp.
type IDec struct {
	mant big.Int
	exp  int32
}

var (
	iTen = big.NewInt(10)
	iOne = big.NewInt(1)
)

func (z *IDec) norm() {
	if z.mant.Sign() == 0 {
		z.exp = 0
		return
	}
	var q, r big.Int
	for {
		q.QuoRem(&z.mant, iTen, &r)
		if r.Sign() != 0 {
			break
		}
		z.mant.Set(&q)
		z.exp++
	}
}

func (x *IDec) Mul(y *IDec) *IDec {
	z := new(IDec)
	z.mant.Mul(&x.mant, &y.mant)
	z.exp = x.exp + y.exp
	z.norm()
	return z
}

func (z *IDec) SetString(s string) *IDec {
	var e string = "0"
	if i := strings.Index(s, "e"); i >= 0 {
		e = s[i+1:]
		s = s[:i]
	}
	z.exp = 0
	if i := strings.Index(s, "."); i >= 0 {
		z.exp = int32(i - len(s) + 1)
		s = s[:i] + s[i+1:]
	}
	_, ok := z.mant.SetString(s, 0)
	if !ok {
		panic("(*IDec).SetString: (*big.Int).SetString failed")
	}
	exp, err := strconv.ParseInt(e, 0, 32)
	if err != nil {
		panic(err)
	}
	z.exp += int32(exp)
	z.norm()
	return z
}

func (x *IDec) String() string {
	b := x.mant.Append(nil, 10)
	if x.exp != 0 {
		b = append(b, 'e')
		b = strconv.AppendInt(b, int64(x.exp), 10)
	}
	return string(b)
}

func TestIDecMul(t *testing.T) {
	for _, d := range []struct {
		x string
		y string
		z string
	}{
		{"0", "0", "0"},
		{"12.1e-1", "1", "121e-2"},
		{"120e-1", "1", "12"},
		{"12", "1", "12"},
		{"-12e-1", "100", "-12e1"},
	} {
		var x, y, want IDec
		x.SetString(d.x)
		y.SetString(d.y)
		want.SetString(d.z)
		z := x.Mul(&y)
		if z.mant.Cmp(&want.mant) != 0 || z.exp != want.exp {
			t.Errorf("%v * %v = %v. Want %v", &x, &y, &z, &want)
		}
	}
}

func (z *IDec) Set(x *IDec) *IDec {
	z.mant.Set(&x.mant)
	z.exp = x.exp
	return z
}

func (z *IDec) SetMantExp(m *big.Int, e int32) *IDec {
	z.mant.Set(m)
	z.exp = e
	return z
}

func intPow10(z *big.Int, exp int32) *big.Int {
	if exp < 0 {
		panic("intPow10: negative exponent")
	}
	if z == nil {
		z = new(big.Int)
	}
	if exp < _DW {
		return z.SetUint64(uint64(pow10(uint(exp))))
	}
	z.SetUint64(uint64(exp))
	return z.Exp(iTen, z, nil)
}

func (x *IDec) Shl(exp int32) *big.Int {
	return new(big.Int).Mul(&x.mant, intPow10(nil, exp))
}

func (x *IDec) Add(y *IDec) *IDec {
	z := new(IDec)
	switch {
	case x.exp > y.exp:
		z.exp = y.exp
		t := x.Shl(x.exp - y.exp)
		z.mant.Add(t, &y.mant)
	default:
		z.exp = x.exp
		z.mant.Add(&x.mant, &y.mant)
	case x.exp < y.exp:
		z.exp = x.exp
		t := y.Shl(y.exp - x.exp)
		z.mant.Add(&x.mant, t)
	}
	z.norm()
	return z
}

func TestIDecAdd(t *testing.T) {
	for _, d := range []struct {
		x string
		y string
		z string
	}{
		{"0", "0", "0"},
		{"120e-1", "1", "13"},
		{"12", "1", "13"},
		{"12e-1", "100", "1012e-1"},
		{"10", "21e-1", "121e-1"},
	} {
		var x, y, want IDec
		x.SetString(d.x)
		y.SetString(d.y)
		want.SetString(d.z)
		z := x.Add(&y)
		if z.mant.Cmp(&want.mant) != 0 || z.exp != want.exp {
			x.SetString(d.x)
			t.Errorf("%v + %v = %v. Want %v", &x, &y, z, &want)
		}
	}
}

func (z *IDec) magnitude() uint {
	m := uint(float64(z.mant.BitLen()) * log10_2)
	// m <= magnitude <= m+1
	if z.mant.Cmp(intPow10(nil, int32(m))) >= 0 {
		return m + 1
	}
	return m
}

func (x *IDec) Round(prec uint, mode RoundingMode) *Decimal {
	m := x.magnitude()
	if m <= prec {
		return x.Decimal()
	}

	// save mantissa sign
	sign := x.mant.Sign()

	var q, r big.Int

	// digits following rounding digit
	q.QuoRem(&x.mant, intPow10(nil, int32(m-prec-1)), &r)
	var sbit uint64
	if r.Sign() != 0 {
		sbit = 1
	}
	// rounding digit and abs of truncated mantissa
	q.QuoRem(&q, iTen, &r)
	rdigit := r.Uint64()
	q.Abs(&q)
	exp := x.exp + int32(m-prec)

	if rdigit|sbit != 0 {
		var inc bool
		switch mode {
		case ToNearestEven: // round do nearest, ties to even
			inc = rdigit > 5 || (rdigit == 5 && (sbit != 0 || (q.Sign() != 0 && q.Bits()[0]&1 != 0)))
		case ToNearestAway: // round do nearest, ties away from 0
			inc = rdigit >= 5
		case ToZero: // truncate
			// nothing to do
		case AwayFromZero: // round away from zero
			inc = true
		case ToNegativeInf: // round towards -Inf
			inc = sign <= 0
		case ToPositiveInf: // round towards +Inf
			inc = sign >= 0
		}
		if inc {
			q.Add(&q, iOne)
		}
	}
	// restore sign
	if sign <= 0 {
		q.Neg(&q)
	}
	z := new(Decimal).SetInt(&q)
	return z.SetMantExp(z, int(exp))
}

func (x *IDec) Decimal() *Decimal {
	z := new(Decimal).SetInt(&x.mant)
	return z.SetMantExp(z, int(x.exp))
}
