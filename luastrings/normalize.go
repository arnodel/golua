package luastrings

import (
	"bytes"
	"regexp"
)

var newLines = regexp.MustCompile(`(?s)\r\n|\n\r|\r`)

func NormalizeNewLines(b []byte) []byte {
	if bytes.IndexByte(b, '\r') == -1 {
		return b
	}
	return newLines.ReplaceAllLiteral(b, []byte{'\n'})
}
