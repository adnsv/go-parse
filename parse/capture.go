package parse

import (
	"sort"
	"unicode/utf8"
)

func Codepoint(r rune) TermFunc {
	return func(src Source, ctx *Context) ErrCode {
		if src.Hop(r) {
			if ctx != nil {
				ctx.WriteRune(r)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func CodepointFunc(m func(rune) bool) TermFunc {
	return func(src Source, ctx *Context) ErrCode {
		r, size := src.Fetch(m)
		if size > 0 {
			if ctx != nil {
				ctx.WriteRune(r)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func Literal(s string) TermFunc {
	if len(s) == 0 {
		panic("empty literal capturer is not allowed")
	}
	return func(src Source, ctx *Context) ErrCode {
		if src.Leap(s) {
			if ctx != nil {
				ctx.WriteString(s)
			}
			return ErrCodeNone
		} else {
			return ErrCodeUnmatched
		}
	}
}

func asValueCapturer(a any) TermFunc {
	switch v := a.(type) {
	case TermFunc:
		return v
	case rune:
		return Codepoint(v)
	case string:
		return Literal(v)
	case func(rune) bool:
		return CodepointFunc(v)
	default:
		panic("unsupported capturer type")
	}
}

func asOptValueCapturer(a any) TermFunc {
	if a == nil {
		return nil
	} else {
		return asValueCapturer(a)
	}
}

func asValueCapturers(args ...any) []TermFunc {
	r := make([]TermFunc, 0, len(args))
	for _, a := range args {
		if a == nil {
			continue
		}
		r = append(r, asValueCapturer(a))
	}
	return r
}

func Sequence(args ...any) TermFunc {
	if len(args) == 0 {
		panic("empty sequence matcher is not allowed")
	}

	if len(args) == 1 {
		return asValueCapturer(args[0])
	} else {
		first := asValueCapturer(args[0])
		rest := asValueCapturers(args[1:]...)
		return func(src Source, ctx *Context) ErrCode {
			ec := first(src, ctx)
			if ec == ErrCodeUnmatched {
				return ErrCodeUnmatched
			}
			for _, m := range rest {
				ec = m(src, ctx)
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
func Optional[T Term](a T) TermFunc {
	v := asValueCapturer(a)
	return func(src Source, ctx *Context) ErrCode {
		ec := v(src, ctx)
		if ec == ErrCodeUnmatched {
			ec = ErrCodeNone
		}
		return ec
	}
}

// AnyOf matches and captures any of the provided literal sequences.
func AnyOf(args ...string) TermFunc {
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

	return func(src Source, ctx *Context) ErrCode {
		c, size := src.Peek()
		if size > 0 {
			if mm, ok := matchers[c]; ok {
				for _, m := range mm {
					if src.Leap(m) {
						if ctx != nil {
							ctx.WriteString(m)
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

func OneOrMore[T Term](a T) TermFunc {
	v := asValueCapturer(a)
	return func(src Source, ctx *Context) ErrCode {
		ec := v(src, ctx)
		if ec != ErrCodeNone {
			return ec
		}
		for {
			ec = v(src, ctx)
			if ec == ErrCodeUnmatched {
				return ErrCodeNone
			} else if ec != ErrCodeNone {
				return ec
			}
		}
	}
}

func ZeroOrMore[T Term](a T) TermFunc {
	v := asValueCapturer(a)
	return func(src Source, ctx *Context) ErrCode {
		for {
			ec := v(src, ctx)
			if ec == ErrCodeUnmatched {
				return ErrCodeNone
			} else if ec != ErrCodeNone {
				return ec
			}
		}
	}
}

func FirstOf(args ...any) TermFunc {
	switch len(args) {
	case 0:
		panic("empty literal capturer is not allowed")
	case 1:
		return asValueCapturer(args[0])
	default:
		vv := asValueCapturers(args...)
		return func(src Source, ctx *Context) ErrCode {
			for _, v := range vv {
				ec := v(src, ctx)
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

func EOF(src Source, ctx *Context) ErrCode {
	if src.Done() {
		return ErrCodeNone
	} else {
		return ErrCodeUnmatched
	}
}

func EOL(src Source, ctx *Context) ErrCode {
	switch {
	case src.Done():
		return ErrCodeNone
	case src.Hop('\n'):
		if ctx != nil {
			ctx.WriteByte('\n')
		}
		return ErrCodeNone
	case src.Leap("\r\n"):
		if ctx != nil {
			ctx.WriteString("\r\n")
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
func HexCodeunit(allow_one_digit bool) TermFunc {
	return func(src Source, ctx *Context) ErrCode {
		v, n := ExtractHexN(src, 2)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n == 1 && !allow_one_digit {
			return ErrCodeInvalid
		} else if ctx != nil {
			ctx.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

func HexCodepoint_XXXX() TermFunc {
	return func(src Source, ctx *Context) ErrCode {
		v, n := ExtractHexN(src, 4)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n < 4 {
			return ErrCodeInvalid
		} else if ctx != nil {
			ctx.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

func HexCodepoint_XXXXXXXX() TermFunc {
	return func(src Source, ctx *Context) ErrCode {
		v, n := ExtractHexN(src, 8)
		if n == 0 {
			return ErrCodeUnmatched
		} else if n < 8 {
			return ErrCodeInvalid
		} else if ctx != nil {
			ctx.WriteByte(byte(v))
		}
		return ErrCodeNone
	}
}

// HexCodeunit_XXXX this is a tricky one that is specialized for escape sequences
// that may decode into a utf-16 pair of surrogates which, in turn, needs to be
// re-assembled into a single codeunit. JSON is a good example.
func HexCodeunit_XXXX(prefix string) TermFunc {
	if prefix == "" {
		panic("HexCodeunit_XXXX requires non-empty prefix")
	}
	return func(src Source, ctx *Context) ErrCode {
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
				if ctx != nil {
					if c2 >= 0xDC00 && c2 <= 0xDFFF {
						ctx.WriteRune(rune((((c & 0x3ff) << 10) | (c2 & 0x3ff)) + 0x10000))
					} else {
						// decode as pair
						ctx.WriteRune(rune(c))
						ctx.WriteRune(rune(c2))
					}
				}
				return ErrCodeNone
			}
		}
		if ctx != nil {
			ctx.WriteRune(rune(c))
		}
		return ErrCodeNone
	}
}

func Between(prefix, terminator any, content ...any) TermFunc {
	if content == nil {
		panic("empty content capturer is not allowed")
	}

	prefix_v := asValueCapturer(prefix)
	terminator_v := asValueCapturer(terminator)
	content_v := Sequence(content...)

	return func(src Source, ctx *Context) ErrCode {
		ec := prefix_v(src, nil)
		if ec != ErrCodeNone {
			return ec
		}

		ec = content_v(src, ctx) // capturing

		if ec == ErrCodeNone {
			ec = terminator_v(src, nil)
		}

		return ec
	}
}
