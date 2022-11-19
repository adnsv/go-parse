package parse

import "strings"

type Key = any

type Binding[K Key] struct {
	k     K
	c     any
	descr string
}

func Bind[K Key, C Capturer](key K, c C, descr string) *Binding[K] {
	return &Binding[K]{
		k:     key,
		c:     c,
		descr: descr,
	}
}

func Tokenize[T Key](buf []byte, bindings []*Binding[T], on_token func(k T, v any, lc LineCol)) error {
	lc := LineCol{}
	src := Static(buf, &lc)
	var captured any
	var ec ErrCode
	sb := strings.Builder{}

outer:
	for !src.Done() {
		lc_orig := lc
		for _, binding := range bindings {
			sb.Reset()
			switch c := binding.c.(type) {
			case ValueCapturer:
				captured, ec = c(src)
			case StringCapturer:
				ec = c(src, &sb)
				if ec == ErrCodeNone {
					captured = sb.String()
				}
			default:
				ec = ErrCodeUnmatched
			}
			if ec == ErrCodeUnmatched {
				continue
			}
			if ec != ErrCodeNone {
				err := &ErrContent{Code: ec, What: binding.descr}
				return &ErrAtLineCol{Err: err, Loc: lc_orig}
			}
			on_token(binding.k, captured, lc_orig)
			continue outer
		}
		err := &ErrContent{ErrCodeUnexpected, "content"}
		return &ErrAtLineCol{Err: err, Loc: lc_orig}
	}
	return nil
}
