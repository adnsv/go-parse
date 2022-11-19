package parse

import (
	"sort"
	"strings"
	"unicode/utf8"
)

func Codepoint(r rune) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		if src.Hop(r) {
			if dst != nil {
				dst.WriteRune(r)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func Literal(s string) StringCapturer {
	if len(s) == 0 {
		panic("empty literal capturer is not allowed")
	}
	return func(src Source, dst *strings.Builder) ErrCode {
		if src.Leap(s) {
			if dst != nil {
				dst.WriteString(s)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func CodepointFunc(m func(rune) bool) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		r, size := src.Fetch(m)
		if size > 0 {
			if dst != nil {
				dst.WriteRune(r)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func Sequence(args ...StringCapturer) StringCapturer {
	if len(args) == 0 {
		panic("empty sequence matcher is not allowed")
	}

	if len(args) == 1 {
		return args[0]
	} else {
		first := args[0]
		then := args[1:]
		return func(src Source, dst *strings.Builder) ErrCode {
			ec := first(src, dst)
			if ec == ErrCodeUnmatched {
				return ErrCodeUnmatched
			}
			for _, m := range then {
				ec = m(src, dst)
				if ec == ErrCodeUnmatched {
					return ErrCodeIncomplete
				} else if ec != ErrCodeNone {
					return ec
				}
			}
			return ErrCodeNone
		}
	}
}

// Optional matches zero or one: (a)?
func Optional(a StringCapturer) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		ec := a(src, dst)
		if ec == ErrCodeUnmatched {
			ec = ErrCodeNone
		}
		return ec
	}
}

// AnyOf matches and captures any of the provided literal sequences.
func AnyOf(args ...string) StringCapturer {
	match_empty := len(args) > 0
	matchers := map[rune][]string{}
	for _, arg := range args {
		r, size := utf8.DecodeRuneInString(arg)
		if size == 0 {
			match_empty = true
		} else if size == 1 && r == utf8.RuneError {
			panic("invalid sequence capturer key")
		} else {
			matchers[r] = append(matchers[r], arg)
		}
	}
	if len(matchers) == 0 {
		panic("empty literal capturer is not allowed")
	}
	for _, kk := range matchers {
		// sort longest first
		sort.Slice(kk, func(i, j int) bool {
			return len(kk[i]) > len(kk[j])
		})
	}

	return func(src Source, dst *strings.Builder) ErrCode {
		c, size := src.Peek()
		if size > 0 {
			if mm, ok := matchers[c]; ok {
				for _, m := range mm {
					if src.Leap(m) {
						if dst != nil {
							dst.WriteString(m)
						}
						return ErrCodeNone
					}
				}
			}
		}
		if match_empty {
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func OneOrMore(a StringCapturer) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		ec := a(src, dst)
		if ec != ErrCodeNone {
			return ec
		}
		for {
			ec = a(src, dst)
			if ec == ErrCodeUnmatched {
				return ErrCodeNone
			} else if ec != ErrCodeNone {
				return ec
			}
		}
	}
}

func ZeroOrMore(a StringCapturer) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		for {
			ec := a(src, dst)
			if ec == ErrCodeUnmatched {
				return ErrCodeNone
			} else if ec != ErrCodeNone {
				return ec
			}
		}
	}
}

func FirstOf(args ...StringCapturer) StringCapturer {
	switch len(args) {
	case 0:
		panic("empty literal capturer is not allowed")
	case 1:
		return args[0]
	default:
		return func(src Source, dst *strings.Builder) ErrCode {
			for _, a := range args {
				ec := a(src, dst)
				if ec == ErrCodeUnmatched {
					continue
				} else {
					return ec
				}
			}
			return ErrCodeUnmatched
		}
	}
}

func EOF(src Source, dst *strings.Builder) ErrCode {
	if src.Done() {
		return ErrCodeNone
	} else {
		return ErrCodeUnmatched
	}
}

func EOL(src Source, dst *strings.Builder) ErrCode {
	switch {
	case src.Done():
		return ErrCodeNone
	case src.Hop('\n'):
		if dst != nil {
			dst.WriteByte('\n')
		}
		return ErrCodeNone
	case src.Leap("\r\n"):
		if dst != nil {
			dst.WriteByte('\r')
			dst.WriteByte('\n')
		}
		return ErrCodeNone
	default:
		return ErrCodeUnmatched
	}
}

// HexCodeunit captures one UTF-8 codeunit by its numeric hexadecimal representation.
//
//   - when `allow_one_digit` == true: `[0-9A-Fa-f][0-9A-Fa-f]?`
//   - when `allow_one_digit` == false: `[0-9A-Fa-f][0-9A-Fa-f]`
func HexCodeunit(allow_one_digit bool) StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		v, n := ExtractHexN(src, 2)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n == 1 && !allow_one_digit {
			return ErrCodeInvalid
		} else if dst != nil {
			dst.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

func HexCodepoint_XXXX() StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		v, n := ExtractHexN(src, 4)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n < 4 {
			return ErrCodeInvalid
		} else if dst != nil {
			dst.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

func HexCodepoint_XXXXXXXX() StringCapturer {
	return func(src Source, dst *strings.Builder) ErrCode {
		v, n := ExtractHexN(src, 8)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n < 8 {
			return ErrCodeInvalid
		} else if dst != nil {
			dst.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

// HexCodeunit_XXXX this is a tricky one that is specialized for escape sequences
// that may decode into a utf-16 pair of surrogates which, in turn, needs to be
// re-assembled into a single codeunit. JSON is a good example.
func HexCodeunit_XXXX(prefix string) StringCapturer {
	if prefix == "" {
		panic("HexCodeunit_XXXX requires non-empty prefix")
	}
	return func(src Source, dst *strings.Builder) ErrCode {
		if !src.Leap(prefix) {
			return ErrCodeUnmatched
		}
		c, n := ExtractHexN(src, 4)
		if n < 4 {
			return ErrCodeInvalid
		}
		if c >= 0xD800 && c <= 0xDBFF {
			// try to avoid decoding into utf16 surrogate pairs
			if prefix == "" || src.Leap(prefix) {
				c2, n := ExtractHexN(src, 4)
				if n < 4 {
					return ErrCodeInvalid
				}
				if c2 >= 0xDC00 && c2 <= 0xDFFF {
					dst.WriteRune(rune((((c & 0x3ff) << 10) | (c2 & 0x3ff)) + 0x10000))
				} else {
					// decode as pair
					dst.WriteRune(rune(c))
					dst.WriteRune(rune(c2))
				}
				return ErrCodeNone
			}
		}
		dst.WriteRune(rune(c))
		return ErrCodeNone
	}
}

func Skip[C Capturer](c C) Skipper {
	switch c := any(c).(type) {
	case Skipper:
		return c
	case ValueCapturer:
		return func(src Source) ErrCode {
			_, ec := c(src)
			return ec
		}
	case StringCapturer:
		return func(src Source) ErrCode { return c(src, nil) }
	case rune:
		return func(src Source) ErrCode {
			if src.Hop(c) {
				return ErrCodeNone
			} else {
				return ErrCodeUnmatched
			}
		}
	case string:
		return func(src Source) ErrCode {
			if src.Leap(c) {
				return ErrCodeNone
			} else {
				return ErrCodeUnmatched
			}
		}
	default:
		return func(src Source) ErrCode {
			// noop
			return ErrCodeNone
		}
	}
}

func string_between[P Capturer, T Capturer](prefix P, content StringCapturer, terminator T, dst *strings.Builder) StringCapturer {
	if content == nil {
		panic("empty content capturer is not allowed")
	}

	prefix_skipper := Skip(prefix)
	terminator_skipper := Skip(terminator)

	return func(src Source, dst *strings.Builder) ErrCode {
		ec := prefix_skipper(src)
		if ec != ErrCodeNone {
			return ec
		}

		ec = content(src, dst) // capturing

		if ec == ErrCodeNone {
			ec = terminator_skipper(src)
		}

		return ec
	}
}

func value_between[P Capturer, T Capturer](prefix P, content ValueCapturer, terminator T, dst *strings.Builder) ValueCapturer {
	if content == nil {
		panic("empty content capturer is not allowed")
	}

	prefix_skipper := Skip(prefix)
	terminator_skipper := Skip(terminator)

	return func(src Source) (v any, ec ErrCode) {
		ec = prefix_skipper(src)
		if ec != ErrCodeNone {
			return
		}

		v, ec = content(src) // capturing

		if ec == ErrCodeNone {
			ec = terminator_skipper(src)
		}

		return v, ec
	}
}
