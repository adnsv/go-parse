package parse

import (
	"strings"
	"unicode/utf8"
)

type Source interface {
	// Done indicates that there is no more content available in the input.
	Done() bool

	NextIs(c rune) (size int)
	Peek() (c rune, size int)

	// Hop consumes one codepoint if it matches c.
	Hop(c rune) bool

	// Leap consumes len(s) bytes only if all the bytes match.
	Leap(seq string) bool

	// Fetch fetches and returns one codepoint. If its value is matched by f, the
	// corresponding sequence of bytes then is consumed and its size is returned.
	Fetch(f func(rune) bool) (c rune, size int)

	// Skip skips all codepoints matching f, returns the codepoint
	// following the last matched one and the total number of bytes consumed.
	Skip(f func(rune) bool) (c rune, size int)
}

type Skipper = func(Source) ErrCode

type ValueCapturer = func(Source) (any, ErrCode)

type StringCapturer = func(Source, *strings.Builder) ErrCode

type Capturer interface {
	StringCapturer | ValueCapturer | Skipper | string
}

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

func (r *static_impl) NextIs(c rune) (size int) {
	if r.pos >= r.end {
		return 0
	}
	have, size := rune(r.buf[r.pos]), 1
	if c >= utf8.RuneSelf && have >= utf8.RuneSelf {
		have, size = utf8.DecodeRune(r.buf[r.pos:])
		if size < 2 {
			// invalid codepoint
			if r.panic_on_invalid {
				panic(Invalid("utf-8 sequence"))
			} else {
				return 0
			}
		}
	}
	if have != c {
		size = 0
	}
	return
}

func (r *static_impl) Peek() (c rune, size int) {
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

func (r *static_impl) Hop(c rune) bool {
	sz := r.NextIs(c)
	if sz == 0 {
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

func (r *static_impl) Fetch(f func(rune) bool) (c rune, size int) {
	c, size = r.Peek()
	if f == nil || f(c) {
		r.pos += size
		if r.loc != nil {
			if c == '\n' {
				r.loc.LineIndex++
				r.loc.ColumnIndex = 0
			} else {
				r.loc.ColumnIndex++
			}
		}
	} else {
		size = 0
	}
	return
}

func (r *static_impl) Skip(f func(rune) bool) (c rune, size int) {
	var n int
	for {
		c, n = r.Fetch(f)
		if n > 0 {
			size += n
		} else {
			break
		}
	}
	return
}
