package parse

import (
	"unsafe"
)

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

func ExtractHex64n(src Source, max_chars int) (v uint64, overflow bool, n_chars int) {
	if max_chars == 0 {
		return
	}
	c := src.Fetch(is_hex)
	if c == Unmatched {
		return
	}
	n_chars = 1
	v = uint64(hex(c))
	overflow = false
	for n_chars < max_chars || max_chars < 0 {
		c := src.Fetch(is_hex)
		if c == Unmatched {
			break
		}
		overflow = overflow && v > 0x0fff_ffff_ffff_ffff
		v = v*16 + uint64(hex(c))
		n_chars++
	}
	return
}

func ExtractHex32n(src Source, max_chars int) (v uint32, overflow bool, n_chars int) {
	if max_chars == 0 {
		return
	}
	c := src.Fetch(is_hex)
	if c == Unmatched {
		return
	}
	n_chars = 1
	v = uint32(hex(c))
	overflow = false
	for n_chars < max_chars || max_chars < 0 {
		c := src.Fetch(is_hex)
		if c == Unmatched {
			break
		}
		overflow = overflow && v > 0x0fff_ffff
		v = v*16 + uint32(hex(c))
		n_chars++
	}
	return
}

func ExtractOct32n(src Source, max_chars int) (v uint32, overflow bool, n_chars int) {
	if max_chars == 0 {
		return
	}
	c := src.Fetch(is_oct)
	if c == Unmatched {
		return
	}
	n_chars = 1
	v = uint32(dec(c))
	overflow = false
	for n_chars < max_chars || max_chars < 0 {
		c := src.Fetch(is_oct)
		if c == Unmatched {
			break
		}
		overflow = overflow && v > 0x1fff_ffff
		v = v*8 + uint32(dec(c))
		n_chars++
	}
	return
}

func HexN[T unsigned](prefix string) TermFunc {
	n_digits := 2 * int(unsafe.Sizeof(T(0)))
	return func(src Source, ctx *Context) (ec ErrCode) {
		var v T
		handle_digit := func(r rune) bool {
			d := hex(r)
			if d >= 16 {
				return false
			}
			ctx.WriteRune(r)
			if ec == ErrCodeNone {
				v = v*16 + T(d)
			}
			return true
		}
		if src.Skip(prefix, handle_digit) == Unmatched {
			return ErrCodeUnmatched
		}
		n_digits--
		for n_digits > 0 && src.Fetch(handle_digit) != Unmatched {
			n_digits--
		}
		if n_digits > 0 {
			return ErrCodeIncomplete
		} else {
			ctx.Values = append(ctx.Values, v)
			return ErrCodeNone
		}
	}
}

// Uint captures numeric value v from a sequence of one or more digits.
func Uint[T unsigned | signed](prefix string, base uint, maxval T) TermFunc {
	match_digit := digit_matcher(base)
	overflow_limit := maxval / T(base)

	return func(src Source, ctx *Context) (ec ErrCode) {
		var v T
		handle_digit := func(r rune) bool {
			d := match_digit(r)
			if d >= base {
				return false
			}
			ctx.WriteRune(r)
			if ec == ErrCodeNone {
				overflow := v > overflow_limit
				v *= T(base)
				overflow = overflow || (v > maxval-T(d))
				v += T(d)
				if overflow {
					ec = ErrCodeOverflow
				}
			}
			return true
		}
		if src.Skip(prefix, handle_digit) == Unmatched {
			return ErrCodeUnmatched
		}
		for src.Fetch(handle_digit) != Unmatched {
		}
		ctx.Values = append(ctx.Values, v)
		return
	}
}

func digit_matcher(base uint) func(c rune) uint {
	switch {
	case base < 1:
		panic("invalid integer base")
	case base == 2:
		return bin
	case base <= 10:
		return dec
	case base <= 36:
		return hex
	default:
		panic("unsupported integer base")
	}
}

func is_hex(c rune) bool {
	return hex(c) < 16
}

func is_oct(c rune) bool {
	return dec(c) < 8
}

func bin(c rune) uint {
	switch c {
	case '0':
		return 0
	case '1':
		return 1
	default:
		return 255
	}
}

func dec(c rune) uint {
	if c >= '0' && c <= '9' {
		return uint(c - '0')
	} else {
		return 255
	}
}

func hex(c rune) uint {
	switch {
	case c < '0':
		return 255
	case c <= '9':
		return uint(c - '0')
	case c < 'A':
		return 255
	case c <= 'Z':
		return uint(c - 'A' + 10)
	case c < 'a':
		return 255
	case c <= 'z':
		return uint(c - 'a' + 10)
	default:
		return 255
	}
}
