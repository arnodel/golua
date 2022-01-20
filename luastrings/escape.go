package luastrings

import (
	"strconv"
	"strings"
	"unicode"
)

var names = []byte("abtnvf")

// Quote a string so that it is a valid Lua string literal
func Quote(s string, quote byte) string {
	var b strings.Builder
	b.WriteByte(quote)
	for _, c := range []byte(s) {
		switch {
		case c == quote:
			b.WriteByte('\\')
			b.WriteByte(c)
		case unicode.IsGraphic(rune(c)):
			b.WriteByte(c)
		default:
			b.WriteByte('\\')
			if c >= 7 && c <= 13 {
				b.WriteByte(names[c-7])
			} else {
				b.WriteString(strconv.FormatInt(int64(c), 10))
			}
		}
	}
	b.WriteByte(quote)
	return b.String()
}
