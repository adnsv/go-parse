package parse

import (
	"fmt"
	"testing"
)

func TestTokenize(t *testing.T) {

	ws := func(c rune) bool { return c <= ' ' }

	//	id_start := func(c rune) bool {
	//		return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || c == '_'
	//	}
	//	id_cont := func(c rune) bool {
	//		return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || c == '_'
	//	}
	//
	//	str_quote := func(c rune) bool {
	//		return c == '\''
	//	}
	//	str_content := func(c rune) bool {
	//		return c >= ' '
	//	}

	bb := []*Binding[string]{
		Bind("ws", OneOrMore(ws), "white space"),
		Bind("id", StartCont(id_start, id_cont), "ident"),
		//Bind("hex", Uint(`\x`, ";", 16), "char code sequence"),
		//Bind("hex", Uint(`\x`, ";", 16), "char code sequence"),
		//Bind("dec", Uint("", "", 10), "digit sequence"),
		//Bind("str", String(str_quote, str_content, 0), "string"),
		//Bind("slc", Between("//", ""), "single-line comment"),
		//Bind("mlc", Between("/*", "*/"), "multi-line comment"),
		//Bind("punct", AnyOf("+", "+=", "="), "punct"),
	}

	tests := []struct {
		src  string
		want string
	}{
		{"", ""},
		//{"abc", "<id:abc>"},
		//{"abc ", "<id:abc> "},
		//{" abc ", " <id:abc> "},
		//{"abc def", "<id:abc> <id:def>"},
		//{"42", "<dec:42>"},
		//{"00042", "<dec:42>"},
		//{"42mm", "<dec:42><id:mm>"},
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
			err := Tokenize([]byte(tt.src), bb, func(k string, v any, _ LineCol) {
				if k == "ws" {
					got += " "
				} else {
					got += fmt.Sprintf("<%s:%v>", k, v)
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
