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
		return c >= ' ' && c != 127
	}

	bb := []*Binding[string]{
		Bind("ws", "whitespace", OneOrMore(ws)),
		Bind("id", "ident", id_start, ZeroOrMore(id_cont)),
		Bind("str", "string", Between('\'', '\'', str_content)),
		Bind("dec", "decimal", Uint[uint16](10, 1000)),
		//Bind("hex", Uint(`\x`, ";", 16), "char code sequence"),
		//Bind("hex", Uint(`\x`, ";", 16), "char code sequence"),
		//Bind("slc", Between("//", ""), "single-line comment"),
		//Bind("mlc", Between("/*", "*/"), "multi-line comment"),
		//Bind("punct", AnyOf("+", "+=", "="), "punct"),
	}

	tests := []struct {
		src  string
		want string
	}{
		{"", ""},
		{" ", " "},
		{"    ", " "},
		{"abc", "<id:abc>"},
		{"abc ", "<id:abc> "},
		{" abc ", " <id:abc> "},
		{"abc def", "<id:abc> <id:def>"},
		{"42", "42"},
		{"1000", "1000"},
		{"1001", "<!ERR:[1:1] overflow decimal>"},
		{"00042", "42"},
		{"42mm", "42<id:mm>"},
		{"42 mm", "42 <id:mm>"},
		//{`\xff;`, "<hex:255>"},
		//{`\xxyz;`, "<!ERR:[1:1] invalid char code sequence>"},
		//{`42 \xxyz;`, "<dec:42> <!ERR:[1:4] invalid char code sequence>"},
		//{`42 \xff`, "<dec:42> <!ERR:[1:4] unterminated char code sequence>"},
		//{"42\n\\xxyz;", "<dec:42> <!ERR:[2:1] invalid char code sequence>"},
		//{"42\n\\xxyz;", "<dec:42> <!ERR:[2:1] invalid char code sequence>"},
		//{"''", "<str:>"},
		//{"'abc'", "<str:abc>"},
		//{"'abc", "<!ERR:[1:1] unterminated string>"},
		//{"'abc\n'", "<!ERR:[1:1] unterminated string>"},
		//{"//", "<slc:>"},
		//{"//abc", "<slc:abc>"},
		//{"// abc", "<slc: abc>"},
		//{"// abc\nxyz", "<slc: abc><id:xyz>"},
		//{"/**/", "<mlc:>"},
		//{"/*abc*/", "<mlc:abc>"},
		//{"/*abc\n*/", "<mlc:abc\n>"},
		//{"/*abc", "<!ERR:[1:1] unterminated multi-line comment>"},
		//{"/*abc\n", "<!ERR:[1:1] unterminated multi-line comment>"},
		//{"// abc /*xyz*/", "<slc: abc /*xyz*/>"},
		//{"// abc\n/*xyz*/", "<slc: abc><mlc:xyz>"},
		//{"/*xyz \n// abc*/", "<mlc:xyz \n// abc>"},
		//{"+", "<punct:+>"},
		//{"+=", "<punct:+=>"},
		//{"+ =", "<punct:+> <punct:=>"},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("tokenize %q", tt.src)
		t.Run(name, func(t *testing.T) {
			got := ""
			err := Tokenize([]byte(tt.src), bb, func(k string, c *Context, _ LineCol) {
				switch k {
				case "ws":
					got += " "
				case "dec":
					if len(c.Values) == 1 {
						if v, ok := c.Values[0].(uint16); ok {
							got += fmt.Sprintf("%d", v)
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
