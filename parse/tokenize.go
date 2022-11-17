package parse

type Key = any

type Binding[K Key] struct {
	k     K
	c     Capturer
	descr string
}

func Bind[K Key](key K, c Capturer, descr string) *Binding[K] {
	return &Binding[K]{
		k:     key,
		c:     c,
		descr: descr,
	}
}

func Tokenize[T Key](buf []byte, bindings []*Binding[T], on_token func(k T, v any, lc LineCol)) error {
	lc := LineCol{}
	src := Static(buf, &lc)

outer:
	for !src.Done() {
		lc_orig := lc
		for _, binding := range bindings {
			v, ec := binding.c.Capture(src)
			if ec != ErrCodeNone {
				err := &ErrContent{Code: ec, What: binding.descr}
				return &ErrAtLineCol{Err: err, Loc: lc_orig}
			}
			if v == nil {
				continue
			}
			on_token(binding.k, v, lc_orig)
			continue outer
		}
		err := &ErrContent{ErrCodeUnexpected, "content"}
		return &ErrAtLineCol{Err: err, Loc: lc_orig}
	}
	return nil
}
