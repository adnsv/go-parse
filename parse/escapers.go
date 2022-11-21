package parse

import (
	"unicode"
	"unicode/utf16"
)

// Escaped creates a matcher for escape sequences (typically found inside string literals).
//
// Supported escaper types (assuming prefix is `\`):
//
//   - struct{}   self-mapping    \z    -> z
//   - byte       maps to byte    \z    -> byte code unit
//   - rune       maps to rune    \z    -> utf8-encoded codepoint
//   - string     maps to string  \z    -> literal string sequence
//   - TermFunc   uses termfunc   \z... -> envokes TermFunc to decode `...`
//
// A key value in the supplied map may be specified as Unmatched
func Escaped(prefix rune, escapers map[rune]any) TermFunc {

	literals := map[rune]string{}
	functors := map[rune]TermFunc{}

	for r, a := range escapers {
		switch v := a.(type) {
		case struct{}:
			literals[r] = string([]rune{r})
		case byte:
			literals[r] = string([]byte{v})
		case rune:
			literals[r] = string([]rune{v})
		case string:
			literals[r] = v
		case TermFunc:
			functors[r] = v
		default:
			panic("unsupported escaper type")
		}
	}

	return func(src Source, ctx *Context) ErrCode {
		if !src.Hop(prefix) {
			return ErrCodeUnmatched
		}
		c := src.Peek()
		if c != Unmatched {
			if lit, ok := literals[c]; ok {
				src.Hop(c)
				if ctx != nil {
					ctx.WriteString(lit)
				}
				return ErrCodeNone
			}
			if f, ok := functors[c]; ok {
				src.Hop(c)
				ec := f(src, ctx)
				if ec == ErrCodeUnmatched {
					ec = ErrCodeInvalid
				}
				return ec
			}
		}

		if lit, ok := literals[Unmatched]; ok {
			src.Hop(c)
			if ctx != nil {
				ctx.WriteString(lit)
			}
			return ErrCodeNone
		}
		if f, ok := functors[Unmatched]; ok {
			ec := f(src, ctx)
			if ec == ErrCodeUnmatched {
				ec = ErrCodeInvalid
			}
			return ec
		}
		return ErrCodeInvalid
	}
}

// HexCodeunit_Xn reads hexadecimal digits from src and inserts the
// corresponding numeric value into captured string as a UTF-8 codeunit. The
// codeunit is inserted as-is, without any validation.
//
// Returned values are:
//
//   - `ErrCodeUnmatched` if src does not start with the hex digit
//   - `ErrCodeInvalid` if the obtained value exceeds 255
//   - `ErrCodeNone` if src contains a value in [0..255] range
//
// This function consumes all the hex digits, regardless of overflow.
func HexCodeunit_Xn(src Source, ctx *Context) ErrCode {
	v, o, n := ExtractHex32n(src, -1)
	if n == 0 {
		return ErrCodeUnmatched
	} else if o || v > 255 {
		return ErrCodeInvalid
	} else if ctx != nil {
		ctx.WriteByte(byte(v))
	}
	return ErrCodeNone
}

// HexCodeunit_XX reads two hexadecimal digits from src and inserts the
// corresponding numeric value into captured string as a UTF-8 codeunit. The
// codeunit is inserted as-is, without any validation.
//
// Returned values are:
//
//   - `ErrCodeUnmatched` if src does not start with the hex digit
//   - `ErrCodeIncomplete` if src contains only one hex digit
//   - `ErrCodeNone` if src contains two hex digits
//
// If src contains more than two hex digits, this function consumes only
// the the first two of them.
func HexCodeunit_XX(src Source, ctx *Context) ErrCode {
	v, _, n := ExtractHex32n(src, 2)
	if n == 0 {
		return ErrCodeUnmatched
	} else if n == 1 {
		return ErrCodeIncomplete
	} else if ctx != nil {
		ctx.WriteByte(byte(v))
	}
	return ErrCodeNone
}

// HexCodepoint_XXXX captures four hexadecimal digits and interprets those as a
// UTF-16 codepoint. This codeunit is then converted to a UTF-8 sequence and
// inserted into the captured string.
//
// Returned values are:
//
//   - `ErrCodeUnmatched` if src does not start with the hex digit
//   - `ErrCodeIncomplete` if src contains less than 4 hex digits
//   - `ErrCodeInvalid` if src is a surrogate.
//   - `ErrCodeNone` if src contains 4 hex digits that represent a valid codepoint.
//
// If src contains more than 4 digits, this function consumes only
// the the first 4 them.
func HexCodepoint_XXXX(src Source, ctx *Context) ErrCode {
	v, _, n := ExtractHex32n(src, 4)
	if n == 0 {
		return ErrCodeUnmatched
	} else if n != 4 {
		return ErrCodeInvalid
	} else if utf16.IsSurrogate(rune(v)) {
		return ErrCodeInvalid
	} else if ctx != nil {
		ctx.WriteRune(rune(v))
	}
	return ErrCodeNone
}

// HexCodepoint_XXXXXXXX captures 8 hexadecimal digits and interprets those
// as a UTF-32 codeunit. This codeunit is then converted to a UTF-8 sequence and
// inserted into the captured string. This function does not perform any
// validation, neither does it check for surrogates.
//
// Returned values are:
//
//   - `ErrCodeUnmatched` if src does not start with the hex digit
//   - `ErrCodeIncomplete` if src contains less than 8 hex digits
//   - `ErrCodeNone` if src contains 8 hex digits
//
// If src contains more than 8 digits, this function consumes only the the first
// 8 them.
func HexCodepoint_XXXXXXXX(src Source, ctx *Context) ErrCode {
	v, _, n := ExtractHex32n(src, 8)
	if n == 0 {
		return ErrCodeUnmatched
	} else if n != 8 {
		return ErrCodeIncomplete
	} else if utf16.IsSurrogate(rune(v)) || rune(v) > unicode.MaxRune {
		return ErrCodeInvalid
	} else if ctx != nil {
		ctx.WriteRune(rune(v))
	}
	return ErrCodeNone
}

// HexCodeunit_XXXX this is a tricky one that is specialized for escape sequences
// that may decode into a utf-16 pair of surrogates which, in turn, needs to be
// re-assembled into a single codeunit. JSON is a good example.
func HexCodeunit_XXXX(first_prefix, second_prefix string) TermFunc {
	if second_prefix == "" {
		panic("HexCodeunit_XXXX requires non-empty second prefix")
	}
	return func(src Source, ctx *Context) ErrCode {
		if !src.Leap(first_prefix) {
			return ErrCodeUnmatched
		}
		c, _, n := ExtractHex32n(src, 4)
		if n < 4 {
			return ErrCodeInvalid
		}
		if c >= 0xD800 && c <= 0xDFFF {
			if c > 0xDC00 {
				return ErrCodeInvalid
			}
			if !src.Leap(second_prefix) {
				return ErrCodeInvalid
			}
			c2, _, n := ExtractHex32n(src, 4)
			if n < 4 {
				return ErrCodeInvalid
			}
			if ctx != nil {
				if c2 >= 0xDC00 && c2 <= 0xDFFF {
					ctx.WriteRune(rune((((c & 0x3ff) << 10) | (c2 & 0x3ff)) + 0x10000))
				} else {
					return ErrCodeInvalid
				}
			}
			return ErrCodeNone
		}
		if ctx != nil {
			ctx.WriteRune(rune(c))
		}
		return ErrCodeNone
	}
}

// OctCodeunit_X3n reads 1~3 octal digits from src and inserts the
// corresponding numeric value into captured string as a UTF-8 codeunit. The
// codeunit is inserted as-is, without any validation.
//
// Returned values are:
//
//   - `ErrCodeUnmatched` if src does not start with the hex digit
//   - `ErrCodeInvalid` if the obtained value exceeds 255
//   - `ErrCodeNone` if src contains a value in [0..255] range
func OctCodeunit_X3n(src Source, ctx *Context) ErrCode {
	v, _, n := ExtractOct32n(src, 3)
	if n == 0 {
		return ErrCodeUnmatched
	} else if v > 255 {
		return ErrCodeInvalid
	} else if ctx != nil {
		ctx.WriteByte(byte(v))
	}
	return ErrCodeNone
}
