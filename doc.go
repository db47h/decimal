// Copyright 2020 Denis Bernard <db047h@gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package decimal implements arbitrary-precision decimal floating-point
arithmetic.

The implementation is heavily based on big.Float and besides a few additional
getters and setters for other math/big types, the API is identical to that of
*big.Float.

Howvever, and unlike big.Float, the mantissa of a decimal is stored in a
little-endian Word slice as "declets" of 9 or 19 decimal digits per 32 or 64
bits Word. All arithmetic operations are performed directly in base 10**9 or
10**19 without conversion to/from binary.

The zero value for a Decimal corresponds to 0. Thus, new values can be declared
in the usual ways and denote 0 without further initialization:

    x := new(Decimal)  // x is a *Decimal of value 0

Alternatively, new Decimal values can be allocated and initialized with the
function:

    func NewDecimal(f float64) *Decimal

For instance, NewDecimal(x) returns a *Decimal set to the value of the float64
argument f. More flexibility is provided with explicit setters, for instance:

    z := new(Float).SetUint64(123)    // z3 := 123.0

Setters, numeric operations and predicates are represented as methods of the
form:

    func (z *Decimal) SetV(v V) *Decimal                // z = v
    func (z *Decimal) Unary(x *Decimal) *Decimal        // z = unary x
    func (z *Decimal) Binary(x, y *Decimal) *Decimal    // z = x binary y
    func (x *Decimal) Pred() P                          // p = pred(x)

For unary and binary operations, the result is the receiver (usually named z in
that case; see below); if it is one of the operands x or y it may be safely
overwritten (and its memory reused).

Arithmetic expressions are typically written as a sequence of individual method
calls, with each call corresponding to an operation. The receiver denotes the
result and the method arguments are the operation's operands. For instance,
given three *Decimal values a, b and c, the invocation

    c.Add(a, b)

computes the sum a + b and stores the result in c, overwriting whatever value
was held in c before. Unless specified otherwise, operations permit aliasing of
parameters, so it is perfectly ok to write

    sum.Add(sum, x)

to accumulate values x in a sum.

(By always passing in a result value via the receiver, memory use can be much
better controlled. Instead of having to allocate new memory for each result, an
operation can reuse the space allocated for the result value, and overwrite that
value with the new result in the process.)

Notational convention: Incoming method parameters (including the receiver) are
named consistently in the API to clarify their use. Incoming operands are
usually named x, y, a, b, and so on, but never z. A parameter specifying the
result is named z (typically the receiver).

For instance, the arguments for (*Decimal).Add are named x and y, and because
the receiver specifies the result destination, it is called z:

    func (z *Decimal) Add(x, y *Decimal) *Decimal

Methods of this form typically return the incoming receiver as well, to enable
simple call chaining.

Methods which don't require a result value to be passed in (for instance,
Decimal.Sign), simply return the result. In this case, the receiver is typically
the first operand, named x:

    func (x *Decimal) Sign() int

Various methods support conversions between strings and corresponding numeric
values, and vice versa: Decimal implements the Stringer interface for a
(default) string representation of the value, but also provides SetString
methods to initialize a Decimal value from a string in a variety of supported
formats (see the SetString documentation).

Finally, *Decimal satisfies the fmt package's Scanner interface for scanning and
the Formatter interface for formatted printing.
*/
package decimal
