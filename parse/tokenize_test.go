package parse

import (
	"fmt"
	"testing"
)

func TestTokenize(t *testing.T) {

	ws := func(c rune) bool { return c <= ' ' }

	id_start := func(c rune) bool {
		return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || c == '_'
	}
	id_cont := func(c rune) bool {
		return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || c == '_'
	}
	str_content := func(c rune) bool {
		return c >= ' ' && c != 127 && c != '\''
	}
	xstr_content := func(c rune) bool {
		return c >= ' ' && c != 127 && c != '"'
	}

	bb := []*Binding[string]{
		Bind("ws", "whitespace", Skip(OneOrMore(ws))),
		Bind("id", "ident", id_start, ZeroOrMore(id_cont)),
		Bind("str", "string", Between('\'', '\'', ZeroOrMore(str_content))),
		Bind("hex32", "hex", Uint[uint32]("0x", 16, 0xffffffff)),
		Bind("dec32", "decimal", Uint[uint32]("", 10, 0xffffffff)),
		Bind("slc", "single-line comment", Between("//", EOL)),
		Bind("mlc", "multi-line comment", Between("/*", "*/")),
		Bind("punct", "punct", AnyOf("+", "+=", "=")),
		Bind("xstr", "string", Between('"', '"',
			ZeroOrMore(FirstOf(
				Escaped('\\', map[rune]any{
					'x':  HexCodeunit_Xn,
					'u':  HexCodepoint_XXXX,
					'U':  HexCodepoint_XXXXXXXX,
					'\'': '\'',
					'"':  '"',
					'?':  '?',
					'\\': '\\',
					'a':  '\a',
					'b':  '\b',
					'f':  '\f',
					'n':  '\n',
					'r':  '\r',
					't':  '\t',
					'v':  '\v',
					// ['0'..'7']
					Unmatched: OctCodeunit_X3n,
				}),
				xstr_content,
			)))),
	}

	tests := []struct {
		src  string
		want string
	}{
		{"", ""},
		{" ", " "},
		{"    ", " "},
		{"\n\n\n\t \t", " "},
		{"abc", "<id:abc>"},
		{"abc ", "<id:abc> "},
		{" abc ", " <id:abc> "},
		{"abc def", "<id:abc> <id:def>"},
		{"42", "42"},
		{"1000", "1000"},
		{"4294967295", "4294967295"},
		{"00004294967295", "4294967295"},
		{"4294967296", "<!ERR:[1:1] overflow decimal>"},
		{"00042", "42"},
		{"42mm", "42<id:mm>"},
		{"42 mm", "42 <id:mm>"},
		{`0xff`, "0xFF"},
		{`0xffffffff`, "0xFFFFFFFF"},
		{`0x1ffffffff`, "<!ERR:[1:1] overflow hex>"},
		{`0x0000ff`, "0xFF"},
		{`0xxyz`, "0<id:xxyz>"},
		{`;`, "<!ERR:[1:1] unexpected content>"},
		{`42 0xxxyz;`, "42 0<id:xxxyz><!ERR:[1:10] unexpected content>"},
		{`42 0xff`, "42 0xFF"},
		{"''", "<str:>"},
		{"'abc'", "<str:abc>"},
		{"'abc''def'", "<str:abc><str:def>"},
		{"'abc", "<!ERR:[1:1] unterminated string>"},
		{"'abc\n'", "<!ERR:[1:1] unterminated string>"},
		{"//", "<slc:>"},
		{"//abc", "<slc:abc>"},
		{"// abc", "<slc: abc>"},
		{"// abc\nxyz", "<slc: abc><id:xyz>"},
		{"/**/", "<mlc:>"},
		{"/*abc*/", "<mlc:abc>"},
		{"/*abc\n*/", "<mlc:abc\n>"},
		{"/*abc", "<!ERR:[1:1] unterminated multi-line comment>"},
		{"/*abc\n", "<!ERR:[1:1] unterminated multi-line comment>"},
		{"// abc /*xyz*/", "<slc: abc /*xyz*/>"},
		{"// abc\n/*xyz*/", "<slc: abc><mlc:xyz>"},
		{"/*xyz \n// abc*/", "<mlc:xyz \n// abc>"},
		{"+", "<punct:+>"},
		{"+=", "<punct:+=>"},
		{"+ =", "<punct:+> <punct:=>"},
		{`""`, "<xstr:>"},
		{`" "`, "<xstr: >"},
		{`"\50"`, "<xstr:(>"},
		{`"\508"`, "<xstr:(8>"},
		{`"\050"`, "<xstr:(>"},
		{`"\x21"`, "<xstr:!>"},
		{`"\x000021"`, "<xstr:!>"},
		{`"\u0021"`, "<xstr:!>"},
		{`"\uDC00"`, "<!ERR:[1:1] invalid string>"},
		{`"\Z"`, "<!ERR:[1:1] invalid string>"},
		{`"\\"`, "<xstr:\\>"},
		{`"abc""xyz"`, "<xstr:abc><xstr:xyz>"},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("tokenize %q", tt.src)
		t.Run(name, func(t *testing.T) {
			got := ""
			err := Tokenize([]byte(tt.src), bb, func(k string, c *Context, _ LineCol) {
				switch k {
				case "ws":
					if c.String() == "" {
						got += " "
					} else {
						got += "#ERR"
					}

				case "dec32":
					if len(c.Values) == 1 {
						if v, ok := c.Values[0].(uint32); ok {
							got += fmt.Sprintf("%d", v)
						} else {
							got += "#OVERFLOW"
						}
					} else {
						got += "?"
					}
				case "hex32":
					if len(c.Values) == 1 {
						if v, ok := c.Values[0].(uint32); ok {
							got += fmt.Sprintf("0x%X", v)
						} else {
							got += "#OVERFLOW"
						}
					} else {
						got += "?"
					}

				default:
					got += fmt.Sprintf("<%s:%v>", k, c.String())

				}

			})
			if err != nil {
				got += fmt.Sprintf("<!ERR:%s>", err.Error())
			}

			if got != tt.want {
				t.Errorf("Tokenize() = %v, want %s", got, tt.want)
			}
		})
	}
}
