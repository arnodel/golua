package luastrings

import (
	"strconv"
	"strings"
	"unicode"
)

var names = []byte("abtnvf")

func Quote(s string, quote rune) string {
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote:
			b.WriteByte('\\')
			b.WriteRune(r)
		case unicode.IsGraphic(r):
			b.WriteRune(r)
		case r < 256:
			b.WriteByte('\\')
			if r >= 7 && r <= 13 {
				b.WriteByte(names[r-7])
			}
			b.WriteString(strconv.FormatInt(int64(r), 10))
		default:
			b.Write([]byte("\\x"))
			b.WriteRune(r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}
