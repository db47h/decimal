package decimal

import (
	"fmt"
	"io"
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
	bn, n := decMaxPow(uint(b1)) // at most n+1 digits in base b1 fit into Word
	di := Word(0)                // 0 <= di < b1**i < bn
	i := 0                       // 0 <= i < n
	dp := -1                     // position of decimal point
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
				r.UnreadByte() // ch does not belong to number anymore
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
					z = z.shl(z, _WD)
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
