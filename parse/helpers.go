package parse

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func ExtractUnsigned[T unsigned](src Source, base uint, maxval T) (v T, ec ErrCode) {
	match_digit := digit_matcher(base)
	overflow_limit := maxval / T(base)
	handle_digit := func(r rune) bool {
		d := match_digit(r)
		if d >= base {
			return false
		}
		if ec != ErrCodeNone {
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
	_, size := src.Skip(handle_digit)
	if size == 0 {
		ec = ErrCodeUnmatched
	}
	return
}

func ExtractHexN(src Source, max_chars uint) (v uint, n_chars uint) {
	for n_chars < max_chars {
		_, size := src.Fetch(func(r rune) bool {
			d := hex(r)
			if d >= 16 {
				return false
			} else {
				v = v*16 + d
				return true
			}
		})
		if size > 0 {
			n_chars++
		} else {
			break
		}
	}
	return
}

// UintCapturer captures numeric value v from a sequence of one or more hex digits.
func UintCapturer[T unsigned](base uint, maxval T) func(src Source) (v T, ec ErrCode) {
	match_digit := digit_matcher(base)
	overflow_limit := maxval / T(base)

	return func(src Source) (v T, ec ErrCode) {
		handle_digit := func(r rune) bool {
			d := match_digit(r)
			if d >= base {
				return false
			}
			if ec != ErrCodeNone {
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
		_, size := src.Skip(handle_digit)
		if size == 0 {
			ec = ErrCodeUnmatched
		}
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
