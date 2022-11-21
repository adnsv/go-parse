package parse

import (
	"strings"
	"unicode/utf8"
)

type Source interface {
	// Done indicates that there is no more content available in the input.
	Done() bool

	// Peek previews the codepoint without consuming it. Returns the Unmatched
	// sentinel if the source is at the end of input.
	Peek() rune

	// Hop consumes one codepoint if it matches c.
	Hop(c rune) bool

	// Leap consumes len(seq) bytes only if all the bytes match.
	Leap(seq string) bool

	// Fetch consumes and returns one codepoint if its value is matched by f.
	// Otherwise, it returns the Unmatched sentinel.
	Fetch(f func(rune) bool) rune

	// Skip consumes len(seq) bytes only if all the bytes match and the codepoint
	// that follows matches the term. This is similar to Leap followed by Fetch.
	Skip(seq string, term func(rune) bool) rune
}

const Unmatched = rune(0x7fffffff)

type static_impl struct {
	buf              []byte
	pos              int
	end              int
	panic_on_invalid bool
	loc              *LineCol
}

// Static implements Source that reads content from memory-loaded data.
func Static(buf []byte, lc *LineCol) *static_impl {
	return &static_impl{
		buf: buf,
		end: len(buf),
		loc: lc,
	}
}

func (r *static_impl) Offset() int {
	return r.pos
}

func (r *static_impl) Done() bool {
	return r.pos >= r.end
}

func (r *static_impl) next() (c rune, size int) {
	if r.pos >= r.end {
		return 0, 0
	}
	c, size = rune(r.buf[r.pos]), 1
	if c >= utf8.RuneSelf {
		c, size = utf8.DecodeRune(r.buf[r.pos:])
		if size < 2 {
			// invalid codepoint
			if r.panic_on_invalid {
				panic(Invalid("utf-8 sequence"))
			}

			// zip through the rest of the invalids
			for r.pos+size < r.end && ((r.buf[r.pos+size] & 0b11000000) == 0b10000000) {
				size++
			}
			return utf8.RuneError, size
		}
	}
	return c, size
}

func (r *static_impl) Peek() rune {
	c, sz := r.next()
	if sz > 0 {
		return c
	} else {
		return Unmatched
	}
}

func (r *static_impl) Hop(c rune) bool {
	have, sz := r.next()
	if sz == 0 || c != have {
		return false
	}
	r.pos += sz
	if r.loc != nil {
		if c == '\n' {
			r.loc.LineIndex++
			r.loc.ColumnIndex = 0
		} else {
			r.loc.ColumnIndex++
		}
	}
	return true
}

func (r *static_impl) Leap(seq string) bool {
	n := len(seq)
	ok := r.pos+n <= r.end && string(r.buf[r.pos:r.pos+n]) == seq
	if ok {
		r.pos += n
		if r.loc != nil {
			for {
				if i := strings.IndexByte(seq, '\n'); i >= 0 {
					r.loc.LineIndex++
					r.loc.ColumnIndex = 0
					seq = seq[i+1:]
				} else {
					break
				}
			}
			r.loc.ColumnIndex += utf8.RuneCountInString(seq)
		}
	}
	return ok
}

func (r *static_impl) Fetch(f func(rune) bool) rune {
	c, size := r.next()
	if size > 0 && (f == nil || f(c)) {
		r.pos += size
		if r.loc != nil {
			if c == '\n' {
				r.loc.LineIndex++
				r.loc.ColumnIndex = 0
			} else {
				r.loc.ColumnIndex++
			}
		}
		return c
	} else {
		return Unmatched
	}
}

func (r *static_impl) Skip(seq string, term func(rune) bool) rune {
	n := len(seq)
	if n == 0 {
		return r.Fetch(term)
	}
	seq_ok := r.pos+n < r.end && string(r.buf[r.pos:r.pos+n]) == seq
	if !seq_ok {
		return Unmatched
	}
	t, t_size := rune(r.buf[r.pos+n]), 1
	if t >= utf8.RuneSelf {
		t, t_size = utf8.DecodeRune(r.buf[r.pos+n:])
		if t_size < 2 {
			return Unmatched
		}
	}
	if !term(t) {
		return Unmatched
	}
	r.pos += n + t_size
	if r.loc != nil {
		for {
			if i := strings.IndexByte(seq, '\n'); i >= 0 {
				r.loc.LineIndex++
				r.loc.ColumnIndex = 0
				seq = seq[i+1:]
			} else {
				break
			}
		}
		r.loc.ColumnIndex += utf8.RuneCountInString(seq)
		if t == '\n' {
			r.loc.LineIndex++
			r.loc.ColumnIndex = 0
		} else {
			r.loc.ColumnIndex++
		}
	}
	return t
}
