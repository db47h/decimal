# decimal

[![godocb]][godoc]

Package decimal implements arbitrary-precision decimal floating-point arithmetic
for Go.

## Rationale

How computers represent numbers internally is of no import for most
applications. However numerical applications that interract with humans must use
the same number base as humans. A very simple example is this:
https://play.golang.org/p/CVjiDCdhyoR where `1.1 + 0.11 = 1.2100000000000002`.
Not quite what one would expect.

There are other arbitrary-precision decimal floating-point libraries available,
but most use a binray representation of the mantissa (building on top of
`big.Int`), use external C libraries or are just plain slow.

This started out as an experiment on how an implementation using a pure decimal
representation would fare performance-wise. Secondary goals were to implement it
with an API that would match the Go standard library's math/big API (for better
or worse), and build it in such a way that it could some day be integrated into
the Go standard library.

Admittedly, this actually started out while doing some yak shaving, combined
with an NIH syndrom, but the result was well worth it.

## Features

The implementation is in essence a port of `big.Float` to decimal
floating-point, and the API is identical to that of `*big.Float` with the
exception of a few additional getters and setters, an FMA operation, and helper
functions to support implementation of missing low-level Decimal functionality
outside this package.

Unlike big.Float, the mantissa of a Decimal is stored in a little-endian Word
slice as "declets" of 9 or 19 decimal digits per 32 or 64 bits Word. All
arithmetic operations are performed directly in base 10**9 or 10**19 without
conversion to/from binary (see Performance below for a more in-depth discussion
of this choice).

While basic operations are slower than with a binary representation, some
operations like rounding (happening after every other operation!), or aligning
mantissae (in add/subtract) are much cheaper.

### Decimal and IEEE-754

The decimal package supports IEEE-754 rounding modes, signed zeros, infinity,
and an exactly rounded `Sqrt`. Other functions like Log will be implemented in a
future "math" sub-package. All results are rounded to the desired precision (no
manual rounding).

NaN values are not directly supported (like in `big.Float`). They can be
considered as "signaling NaNs" in IEEE-754 terminology, that is when a NaN is
generated as a result of an operation, it causes a panic. Applications that need
to handle NaNs gracefully can use Go's built-in panic/recover machanism to
handle these efficiently: NaNs cause a panic with an ErrNaN which can be tested
to distinguish NaNs from other causes of panic.

Mantissae are always normalized, as a result, Decimals have a single possible
representation:

    0.1 <= mantissa < 1; d = mantissa × 10**exponent

so there is no notion of scale and no Quantize operation.

There is no notion of "context" either. Contexts have not been implemented in
this package simply to keep the API in-line with math/big. They are however so
useful that they will be provided by a future sub-package.

## TODO's and upcoming features

- Some math primitives are implemented in assembler. Right now only the amd64
  version is implemented, so we're still missing i386, arm, mips, power, riscV,
  and s390. The amd64 version could also use a good review (my assembly days
  date back to the Motorola MC68000). HELP WANTED!
- Complete decimal conversion tests
- A math sub-package that will provide at least the functions required by
  IEEE-754
- A context sub-package

The decimal API is frozen, that is, any additional features will be added in
sub-packages.

Well, with the exception of `NewDecimal`: The current `integer mantissa` ×
`10**exp`, works well enough, but I'm not truly happy with it. Early versions
were using a float64 value as initializer, but that lead to unexpected side
effects where one would expect the number to be exact; not quite so as it turned
out, so it's not an option either.

## Performance

There are other full-featured arbitrary-precision decimal-floating point
libraries for Go out there, like [Eric Lagergren's decimal][eldecimal],
[CockroachDB's apd][apd], or [Spring's decimal][spdec].

For users only interested in performance here are the benchmark results of this
package versus the others using Eric's Pi test (times are in ns/op sorted from
fastest to slowest at 38 digits of precision):

| digits | 9 | 19 | 38 | 100 | 500 | 5000 |
|--------|--:|---:|---:|----:|----:|-----:|
| Eric's decimal (Go) | 6415 | 30254 | 65171 | 194263 | 1731528 | 89841923 |
| decimal | 12887 | 42720 | 100878 | 348865 | 4212811 | 342349031| 
| Eric's decimal (GDA) | 7124 | 39357 | 107720 | 392453 | 5421146 | 1175936547 |
| Spring's decimal | 39528 | 96261 | 204017 | 561321 | 3402562 | 97370022 |
| apd | 70833 | 301098 | 1262021 | 9859180 | 716558666 | ??? |

Note that Eric's decimal uses a separate logic for decimals < 1e19 (mantissa
stored in a single uint64), which explains its impressive perfomance for low
precisions.

In additions and subtractions the operands' mantissae need to be aligned
(shifted), this results in an additional multiplication by 10**shift. In
implementations that use a binary representation of the matissa, this is faster
for shifts < 19, but performance degrades as shifts get higher. With a decimal
representation, this requires a multiplication as well but always by a single
Word, regardless of precision. 

Rounding happens after every operation in decimal and Eric's decimal in GDA mode
(not in Go mode, which explains its speed). Rounding requires a decimal shift
right, which translates to a division by 10**shift. Again for small shifts,
binary representations are faster, but degrades even faster as precision gets
higher. On decimal implementations, this operation is quite fast since it
translates to a memcpy and a divmod of the least significant Word.

This explains why decimal's performace degrades slower than Eric's decimal-GDA
as precision increases, and why Eric's decimal in Go mode is so fast (no
rounding, which surprisingly counter-balances the high cost of mantissae
alignment).

## Caveats

The Float <-> Decimal conversion code needs some love

The math/big API is designed to keep memory allocations to a minimum, but some
people find it cumbersome. Indeed it requires some practice to get used to it, 
so here's a quick rundown of what to do and not do:

Most APIs look like:

    func (z *Decimal) Add(x, y *Decimal) *Decimal

where the function sets the receiver `z` to the result of `a + b` and returns
'z'. The fact that the function returns the receiver is meant to allow chaining
of operations:

    s := new(Decimal).Mul(new(Decimal).Mul(r, r), pi) // d = r**2 * pi

If we don't care about what happens to `r`, we can just:

    s := new(Decimal).Mul(r.Mul(r, r), pi)            // r *= r; d = r * pi

and save one memory allocation.

However, NEVER assign the result to a variable:

    d := new(Decimal).SetUint(4)
    d2 := d.Mul(d, d) // d2 == 16, but d == 16 as well!

Again, the sole intent behind returning the receiver is chaining of operations.
By assigning it to a variable, you will shoot yourself in the foot and kill
puppies in some far away land, so never assign the result of an operation!

However, feel free do do this:

    d.Mul(d, d) // d = d*d

The code will properly detect that the receiver is also one of the arguments
(possibly both), and allocate temporary storage space if (and only if)
necessary. Should this kind of construct fail, please file an issue.

## License

Simplified BSD license. See the [LICENSE] file.

The decimal package reuses a lot of code from the Go standard library, governed
by a 3-Clause BSD license. See the [LICENSE-go] file.

I'm aware that software using this package might have to include both licenses,
which might be a hassle; tracking licenses from dependencies is hard enough as
it is. I'd love to have a single license and hand over copyright to "The Go
authors", but the clause restricting use of the names of contributors for
endorsement of a derived work in the 3-Clause BSD license that Go uses is
problematic. i.e. I can't just use it as-is, mentioning Google Inc., as that
would be an infringement in itself (well, that's the way I see it, but IANAL).
On the other hand, any piece of software written in Go should include the Go
license anyway...

Any helpful insights are welcome.

[godoc]: https://pkg.go.dev/github.com/db47h/decimal?tab=doc
[godocb]: https://img.shields.io/badge/go.dev-reference-blue
[eldecimal]: https://github.com/ericlagergren/decimal
[apd]: github.com/cockroachdb/apd
[spdec]: github.com/shopspring/decimal
[LICENSE]: https://github.com/db47h/decimal/blob/master/LICENSE
[LICENSE-go]: https://github.com/db47h/decimal/blob/master/LICENSE-go