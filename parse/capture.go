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
		if r := src.Fetch(m); r == Unmatched {
			return ErrCodeUnmatched
		} else if ctx != nil {
			ctx.WriteRune(r)
		}
		return ErrCodeNone
	}
}

func Literal(s string) TermFunc {
	if len(s) == 0 {
		panic("empty literal term is not allowed")
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

func asTermFunc(a any) TermFunc {
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
		panic("unsupported term type")
	}
}

func asOptTermFunc(a any) TermFunc {
	if a == nil {
		return func(Source, *Context) ErrCode { return ErrCodeNone }
	} else {
		return asTermFunc(a)
	}
}

func asTermFuncs(args ...any) []TermFunc {
	r := make([]TermFunc, 0, len(args))
	for _, a := range args {
		if a == nil {
			continue
		}
		r = append(r, asTermFunc(a))
	}
	return r
}

func Sequence(args ...any) TermFunc {
	if len(args) == 0 {
		panic("empty sequence term is not allowed")
	}

	if len(args) == 1 {
		return asTermFunc(args[0])
	} else {
		first := asTermFunc(args[0])
		rest := asTermFuncs(args[1:]...)
		return func(src Source, ctx *Context) ErrCode {
			ec := first(src, ctx)
			if ec == ErrCodeUnmatched {
				return ErrCodeUnmatched
			}
			for _, m := range rest {
				ec = m(src, ctx)
				if ec == ErrCodeUnmatched {
					return ErrCodeNone
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
	v := asTermFunc(a)
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
	if len(args) == 0 {
		panic("empty literal term is not allowed")
	}
	matchers := map[rune][]string{}
	for _, arg := range args {
		r, size := utf8.DecodeRuneInString(arg)
		if size == 0 {
			panic("empty literal term is not allowed")
		} else if size == 1 && r == utf8.RuneError {
			panic("invalid literal term")
		} else {
			matchers[r] = append(matchers[r], arg)
		}
	}
	for _, kk := range matchers {
		// sort longest first
		sort.Slice(kk, func(i, j int) bool {
			return len(kk[i]) > len(kk[j])
		})
	}

	return func(src Source, ctx *Context) ErrCode {
		if c := src.Peek(); c != Unmatched {
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
		return ErrCodeUnmatched
	}
}

func OneOrMore[T Term](a T) TermFunc {
	v := asTermFunc(a)
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
	v := asTermFunc(a)
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
		panic("empty literal term is not allowed")
	case 1:
		return asTermFunc(args[0])
	default:
		vv := asTermFuncs(args...)
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

func Between(prefix, terminator any, content ...any) TermFunc {
	prefix_v := asOptTermFunc(prefix)

	if len(content) == 0 {
		terminator_v := asTermFunc(terminator)
		return func(src Source, ctx *Context) ErrCode {
			ec := prefix_v(src, nil)
			if ec != ErrCodeNone {
				return ec
			}
			for {
				ec = terminator_v(src, nil)
				if ec == ErrCodeNone {
					return ec
				} else if r := src.Fetch(nil); r == Unmatched {
					return ErrCodeUnterminated
				} else {
					ctx.WriteRune(r)
				}
			}
		}
	} else {
		terminator_v := asOptTermFunc(terminator)
		content_v := Sequence(content...)
		return func(src Source, ctx *Context) ErrCode {
			ec := prefix_v(src, nil)
			if ec != ErrCodeNone {
				return ec
			}

			ec = content_v(src, ctx) // capturing

			if ec == ErrCodeNone {
				ec = terminator_v(src, nil)
				if ec == ErrCodeUnmatched {
					ec = ErrCodeUnterminated
				}
			}

			return ec
		}
	}
}

func Skip(content ...any) TermFunc {
	v := Sequence(content...)
	return func(src Source, ctx *Context) ErrCode {
		return v(src, nil) // not capturing
	}
}
