package parse

import "strings"

type Key = any

type Binding[K Key] struct {
	k     K
	c     TermFunc
	descr string
}

func Bind[K Key](key K, descr string, sequence ...any) *Binding[K] {
	return &Binding[K]{
		k:     key,
		c:     Sequence(sequence...),
		descr: descr,
	}
}

func Tokenize[T Key](buf []byte, bindings []*Binding[T], on_token func(k T, c *Context, lc LineCol)) error {
	lc := LineCol{}
	src := Static(buf, &lc)
	var ec ErrCode
	ctx := Context{}

outer:
	for !src.Done() {
		lc_orig := lc
		for _, binding := range bindings {
			ctx.Reset()
			ec = binding.c(src, &ctx)
			if ec == ErrCodeUnmatched {
				continue
			}
			if ec != ErrCodeNone {
				err := &ErrContent{Code: ec, What: binding.descr}
				return &ErrAtLineCol{Err: err, Loc: lc_orig}
			}
			on_token(binding.k, &ctx, lc_orig)
			continue outer
		}
		err := &ErrContent{ErrCodeUnexpected, "content"}
		return &ErrAtLineCol{Err: err, Loc: lc_orig}
	}
	return nil
}

type Context struct {
	strings.Builder
	Values []any
}

type TermFunc = func(Source, *Context) ErrCode

type Term interface {
	TermFunc | rune | string | func(rune) bool
}

func (c *Context) Reset() {
	c.Builder.Reset()
	c.Values = c.Values[:0]
}
