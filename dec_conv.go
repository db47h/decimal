package decimal

import (
	"fmt"
	"io"
	"math"
)

func (z dec) scan(r io.ByteScanner, base int, fracOk bool) (res dec, b, count int, err error) {
	// reject invalid bases
	baseOk := base == 0 ||
		!fracOk && 2 <= base && base <= MaxBase ||
		fracOk && (base == 2 || base == 8 || base == 10 || base == 16)
	if !baseOk {
		panic(fmt.Sprintf("invalid number base %d", base))
	}

	// prev encodes the previously seen char: it is one
	// of '_', '0' (a digit), or '.' (anything else). A
	// valid separator '_' may only occur after a digit
	// and if base == 0.
	prev := '.'
	invalSep := false

	// one char look-ahead
	ch, err := r.ReadByte()

	// determine actual base
	b, prefix := base, 0
	if base == 0 {
		// actual base is 10 unless there's a base prefix
		b = 10
		if err == nil && ch == '0' {
			prev = '0'
			count = 1
			ch, err = r.ReadByte()
			if err == nil {
				// possibly one of 0b, 0B, 0o, 0O, 0x, 0X
				switch ch {
				case 'b', 'B':
					b, prefix = 2, 'b'
				case 'o', 'O':
					b, prefix = 8, 'o'
				case 'x', 'X':
					b, prefix = 16, 'x'
				default:
					if !fracOk {
						b, prefix = 8, '0'
					}
				}
				if prefix != 0 {
					count = 0 // prefix is not counted
					if prefix != '0' {
						ch, err = r.ReadByte()
					}
				}
			}
		}
	}

	// convert string
	// Algorithm: Collect digits in groups of at most n digits in di
	// and then use mulAddWW for every such group to add them to the
	// result.
	z = z[:0]
	b1 := Word(b)
	bn, n := decMaxPow(b1) // at most n+1 digits in base b1 fit into Word
	di := Word(0)          // 0 <= di < b1**i < bn
	i := 0                 // 0 <= i < n
	dp := -1               // position of decimal point
	for err == nil {
		if ch == '.' && fracOk {
			fracOk = false
			if prev == '_' {
				invalSep = true
			}
			prev = '.'
			dp = count
		} else if ch == '_' && base == 0 {
			if prev != '0' {
				invalSep = true
			}
			prev = '_'
		} else {
			// convert rune into digit value d1
			var d1 Word
			switch {
			case '0' <= ch && ch <= '9':
				d1 = Word(ch - '0')
			case 'a' <= ch && ch <= 'z':
				d1 = Word(ch - 'a' + 10)
			case 'A' <= ch && ch <= 'Z':
				if b <= maxBaseSmall {
					d1 = Word(ch - 'A' + 10)
				} else {
					d1 = Word(ch - 'A' + maxBaseSmall)
				}
			default:
				d1 = MaxBase + 1
			}
			if d1 >= b1 {
				err = r.UnreadByte() // ch does not belong to number anymore
				break
			}
			prev = '0'
			count++

			// collect d1 in di
			di = di*b1 + d1
			i++

			// if di is "full", add it to the result
			if i == n {
				if bn == _BD {
					// shift left by _WD digits
					t := z.make(len(z) + 1)
					copy(t[1:], z)
					z = t
					z[0] = di
				} else {
					z = z.mulAddWW(z, Word(bn), di)
				}
				di = 0
				i = 0
			}
		}

		ch, err = r.ReadByte()
	}

	if err == io.EOF {
		err = nil
	}

	// other errors take precedence over invalid separators
	if err == nil && (invalSep || prev == '_') {
		err = errInvalSep
	}

	if count == 0 {
		// no digits found
		if prefix == '0' {
			// there was only the octal prefix 0 (possibly followed by separators and digits > 7);
			// interpret as decimal 0
			return z[:0], 10, 1, err
		}
		err = errNoDigits // fall through; result will be 0
	}

	// add remaining digits to result
	if i > 0 {
		z = z.mulAddWW(z, pow(b1, i), di)
	}
	res = z.norm()

	// adjust count for fraction, if any
	if dp >= 0 {
		// 0 <= dp <= count
		count = dp - count
	}

	return
}

// utoa converts x to an ASCII representation in the given base;
// base must be between 2 and MaxBase, inclusive.
func (x dec) utoa(base int) []byte {
	return x.itoa(false, base)
}

// itoa is like utoa but it prepends a '-' if neg && x != 0.
func (x dec) itoa(neg bool, base int) []byte {
	if base < 2 || base > MaxBase {
		panic("invalid base")
	}

	// x == 0
	if len(x) == 0 {
		return []byte("0")
	}
	// len(x) > 0

	// allocate buffer for conversion
	i := int(float64(x.digits())/math.Log10(float64(base))) + 1 // off by 1 at most
	if neg {
		i++
	}
	s := make([]byte, i)

	b := Word(base)
	bb, ndigits := decMaxPow(b)

	// construct table of successive squares of bb*leafSize to use in subdivisions
	// result (table != nil) <=> (len(x) > leafSize > 0)
	table := divisors(len(x), b, ndigits, bb)

	// preserve x, create local copy for use by convertWords
	q := dec(nil).set(x)

	// convert q to string s in base b
	q.convertWords(s, b, ndigits, bb, table)

	// strip leading zeros
	// (x != 0; thus s must contain at least one non-zero digit
	// and the loop will terminate)
	i = 0
	for s[i] == '0' {
		i++
	}

	if neg {
		i--
		s[i] = '-'
	}

	return s[i:]
}

type divisor struct{}

// TODO(db47h): implement recursive algorithm
func (q dec) convertWords(s []byte, b Word, ndigits int, bb Word, table []decDivisor) {
	// split larger blocks recursively
	if table != nil {
		panic("not implemented")
	}

	// having split any large blocks now process the remaining (small) block iteratively
	i := len(s)
	var r Word
	if b == 10 {
		// hard-coding for 10 here speeds this up by 1.25x (allows for / and % by constants)
		for len(q) > 0 {
			// extract least significant, base bb "digit"
			r = q[0]
			q = q.shr(q, _WD)
			for j := 0; j < ndigits && i > 0; j++ {
				i--
				// avoid % computation since r%10 == r - int(r/10)*10;
				// this appears to be faster for BenchmarkString10000Base10
				// and smaller strings (but a bit slower for larger ones)
				t := r / 10
				s[i] = '0' + byte(r-t*10)
				r = t
			}
		}
	} else {
		for len(q) > 0 {
			// extract least significant, base bb "digit"
			q, r = q.divW(q, bb)
			for j := 0; j < ndigits && i > 0; j++ {
				i--
				s[i] = digits[r%b]
				r /= b
			}
		}
	}

	// prepend high-order zeros
	for i > 0 { // while need more leading zeros
		i--
		s[i] = '0'
	}
}

var decLeafSize int = 8 // number of Word-size binary values treat as a monolithic block

type decDivisor struct {
	bbb     dec // divisor
	nbits   int // bit length of divisor (discounting leading zeros) ~= log2(bbb)
	ndigits int // digit length of divisor in terms of output base digits
}

func divisors(m int, b Word, ndigits int, bb Word) []decDivisor {
	return nil
}
