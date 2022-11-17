package parse

import (
	"math"
	"sort"
	"unicode/utf8"
)

type skipper struct {
	match func(rune) bool
}

func (s *skipper) Capture(src Source) (any, ErrCode) {
	_, size := src.Skip(s.match)
	if size > 0 {
		return "", ErrCodeNone
	} else {
		return nil, ErrCodeNone
	}
}

// Skipper creates Capturer that simply skips over the content matched by m.
// Note: it does not actually capture anything, returns an empty string instead.
func Skipper(m func(rune) bool) Capturer {
	return &skipper{match: m}
}

type string_capturer struct {
	quote   func(rune) bool // match begin/end quotes
	content func(rune) bool // match content chars
	escape  rune            // use 0 to disable
}

// CaptureString makes a simple string literal capturer where start/stop
// quotation mark is defined by the quote functor and inner content must pass
// the string_char func. Singleline/Multiline option is handled by allowing or
// disallowing '\n' with the string_char matcher.
func String(quote, content func(rune) bool, escape rune) Capturer {
	return &string_capturer{quote: quote, content: content, escape: escape}
}

func (sc *string_capturer) Capture(src Source) (any, ErrCode) {
	term, size := src.Fetch(sc.quote)
	if size == 0 {
		return nil, ErrCodeNone
	}
	rr := []rune{}
	escaped := false
	for {
		escaped = sc.escape > 0 && src.Hop(sc.escape)
		if escaped {
			rr = append(rr, sc.escape)
		}
		if !escaped && src.Hop(term) {
			return string(rr), ErrCodeNone
		} else if c, n := src.Fetch(sc.content); n > 0 {
			rr = append(rr, c)
		} else {
			return nil, ErrCodeUnterminated
		}
	}
}

type startcont_capturer struct {
	start func(rune) bool
	cont  func(rune) bool
}

func StartCont(start, cont func(rune) bool) Capturer {
	return &startcont_capturer{
		start: start,
		cont:  cont,
	}
}

func (sc *startcont_capturer) Capture(src Source) (any, ErrCode) {
	r, size := src.Fetch(sc.start)
	if size == 0 {
		return nil, ErrCodeNone
	}
	rr := []rune{r}
	for {
		r, n := src.Fetch(sc.cont)
		if n == 0 {
			return string(rr), ErrCodeNone
		} else {
			rr = append(rr, r)
		}
	}
}

type sequence_capturer struct {
	matchers map[rune][]string
}

func SequenceCapturer(m map[string]struct{}) Capturer {
	sc := sequence_capturer{}
	sc.matchers = make(map[rune][]string, len(m))
	for k := range m {
		r, size := utf8.DecodeLastRuneInString(k)
		if size == 0 || (size == 1 && r == utf8.RuneError) {
			panic("invalid sequence capturer key")
		}
		sc.matchers[r] = append(sc.matchers[r], k)
	}
	for _, kk := range sc.matchers {
		// sort longest first
		sort.Slice(kk, func(i, j int) bool {
			return len(kk[i]) > len(kk[j])
		})
	}
	return &sc
}

func (sc *sequence_capturer) Capture(src Source) (any, ErrCode) {
	c, size := src.Peek()
	if size == 0 {
		return nil, ErrCodeNone
	}
	mm, ok := sc.matchers[c]
	if !ok {
		return nil, ErrCodeNone
	}
	for _, m := range mm {
		if src.Leap(m) {
			return m, ErrCodeNone
		}
	}
	return nil, ErrCodeNone
}

type uint_capturer struct {
	prefix  string
	postfix string
	base    uint64
	match   func(rune) uint
}

func Uint(prefix, postfix string, base int) Capturer {
	return &uint_capturer{
		prefix:  prefix,
		postfix: postfix,
		base:    uint64(base),
		match:   make_digit_matcher(base),
	}
}

func (hc *uint_capturer) Capture(src Source) (any, ErrCode) {
	var v uint64
	var overflow bool

	handle_digit := func(r rune) bool {
		d := uint64(hc.match(r))
		if d >= hc.base {
			return false
		}
		if !overflow {
			overflow = v > math.MaxUint64/hc.base
			if !overflow {
				v *= hc.base
				overflow = v > math.MaxUint64-d
				v += d
			}
		}
		return true
	}

	if hc.prefix != "" {
		if !src.Leap(hc.prefix) {
			return nil, ErrCodeNone
		}
	}

	_, size := src.Skip(handle_digit)
	if size == 0 {
		if hc.prefix == "" {
			return nil, ErrCodeNone
		} else {
			return nil, ErrCodeInvalid
		}
	}
	if overflow {
		return nil, ErrCodeInvalid
	}

	if hc.postfix != "" && !src.Leap(hc.postfix) {
		return nil, ErrCodeUnterminated
	}

	return v, ErrCodeNone
}

func make_digit_matcher(base int) func(c rune) uint {
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

type sequence_terminated_capturer struct {
	prefix     string
	terminator string // if empty, assume EOL terminator
}

func StartStopSequence(prefix, terminator string) Capturer {
	return &sequence_terminated_capturer{
		prefix:     prefix,
		terminator: terminator,
	}
}

func (sc *sequence_terminated_capturer) Capture(src Source) (any, ErrCode) {
	if sc.prefix != "" {
		if !src.Leap(sc.prefix) {
			return nil, ErrCodeNone
		}
	}
	rr := []rune{}

	if sc.terminator == "" {
		// read everything until EOL or EOF
		for {
			r, size := src.Fetch(nil)
			if size == 0 {
				// reached end of file, which is also considered a EOL
				return string(rr), ErrCodeNone
			} else if r == '\n' {
				if len(rr) > 0 && rr[len(rr)-1] == '\r' {
					rr = rr[:len(rr)-1]
				}
				return string(rr), ErrCodeNone
			} else {
				rr = append(rr, r)
			}
		}
	} else {
		// read everything until sc.terminator
		for {
			if src.Leap(sc.terminator) {
				return string(rr), ErrCodeNone
			}
			r, size := src.Fetch(nil)
			if size == 0 {
				// eof
				return "", ErrCodeUnterminated
			} else {
				rr = append(rr, r)
			}
		}
	}
}
