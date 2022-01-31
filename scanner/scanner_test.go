package scanner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/arnodel/golua/token"
)

type tok struct {
	tp        token.Type
	lit       string
	pos, l, c int
}

func (t tok) Token() *token.Token {
	line := t.l
	if line == 0 {
		line = 1
	}
	col := t.c
	if col == 0 {
		col = 1
	}
	return &token.Token{
		Type: t.tp,
		Lit:  []byte(t.lit),
		Pos:  token.Pos{Offset: t.pos, Line: line, Column: col},
	}
}

func tokenString(t *token.Token) string {
	return fmt.Sprintf("Type:%d, Lit:%q, Pos:%s", t.Type, t.Lit, t.Pos)
}

func TestScanner(t *testing.T) {
	tests := []struct {
		text string
		toks []tok
		err  string
	}{
		//
		// Tokens
		//
		{
			`hello there`,
			[]tok{
				{token.IDENT, "hello", 0, 1, 1},
				{token.IDENT, "there", 6, 1, 7},
				{token.EOF, "", 11, 1, 12},
			},
			"",
		},
		{
			`123.45 "abc" -0xff .5`,
			[]tok{
				{token.NUMDEC, "123.45", 0, 1, 1},
				{token.STRING, `"abc"`, 7, 1, 8},
				{token.SgMinus, "-", 13, 1, 14},
				{token.NUMHEX, "0xff", 14, 1, 15},
				{token.NUMDEC, ".5", 19, 1, 20},
				{token.EOF, "", 21, 1, 22},
			},
			"",
		},
		{
			"...,:<=}///[]>=~~====  --hi\n--[bye",
			[]tok{
				{token.SgEtc, "...", 0, 1, 1},
				{token.SgComma, ",", 3, 1, 4},
				{token.SgColon, ":", 4, 1, 5},
				{token.SgLessEqual, "<=", 5, 1, 6},
				{token.SgCloseBrace, "}", 7, 1, 8},
				{token.SgSlashSlash, "//", 8, 1, 9},
				{token.SgSlash, "/", 10, 1, 11},
				{token.SgOpenSquareBkt, "[", 11, 1, 12},
				{token.SgCloseSquareBkt, "]", 12, 1, 13},
				{token.SgGreaterEqual, ">=", 13, 1, 14},
				{token.SgTilde, "~", 15, 1, 16},
				{token.SgNotEqual, "~=", 16, 1, 17},
				{token.SgEqual, "==", 18, 1, 19},
				{token.SgAssign, "=", 20, 1, 21},
				{token.EOF, "", 34, 2, 7},
			},
			"",
		},
		// Token errors
		{
			`abc?xyz`,
			[]tok{
				{token.IDENT, "abc", 0, 1, 1},
				{token.INVALID, "?", 3, 1, 4},
			},
			"illegal character",
		},
		// Number errors
		{
			"123zabc",
			[]tok{
				{token.NUMDEC, "123", 0, 1, 1},
				{token.INVALID, "z", 3, 1, 4},
			},
			"illegal character following number",
		},
		{
			"0x10abP(",
			[]tok{{token.INVALID, "0x10abP", 0, 1, 1}},
			"digit required after exponent",
		},
		//
		// Long brackets
		//
		{
			`if[[hello]"there]]`,
			[]tok{
				{token.KwIf, "if", 0, 1, 1},
				{token.LONGSTRING, `[[hello]"there]]`, 2, 1, 3},
			},
			"",
		},
		{
			"[==[xy\n\n]]===]==]123.34",
			[]tok{
				{token.LONGSTRING, "[==[xy\n\n]]===]==]", 0, 1, 1},
				{token.NUMDEC, "123.34", 17, 3, 10},
			},
			"",
		},
		{
			`1e3--[=[stuff--[[inner]] ]=]'"o"'`,
			[]tok{
				{token.NUMDEC, "1e3", 0, 1, 1},
				{token.STRING, `'"o"'`, 28, 1, 29},
			},
			"",
		},
		// Long bracket errors
		{
			`[==!!!`,
			[]tok{{token.INVALID, `[==!`, 0, 1, 1}},
			"expected opening long bracket",
		},
		{
			`   [===[foo]==]`,
			[]tok{{token.INVALID, `[===[foo]==]`, 3, 1, 4}},
			"illegal <eof> in long bracket of level 3",
		},
		//
		// Short strings
		//
		{
			"'\\u{123a}\\xff\\\n\\''",
			[]tok{
				{token.STRING, "'\\u{123a}\\xff\\\n\\''", 0, 1, 1},
				{token.EOF, "", 18, 2, 4},
			},
			"",
		},
		{
			"'abc\\z\n  \r\\65x'",
			[]tok{
				{token.STRING, "'abc\\z\n  \r\\65x'", 0, 1, 1},
				{token.EOF, "", 15, 3, 6},
			},
			"",
		},
		// Short string errors
		{
			`"\x2w"`,
			[]tok{{token.INVALID, `"\x2`, 0, 1, 1}},
			`\x must be followed by 2 hex digits`,
		},
		{
			`'\uz`,
			[]tok{{token.INVALID, `'\uz`, 0, 1, 1}},
			`\u must be followed by '{'`,
		},
		{
			`'  \u{l}''`,
			[]tok{{token.INVALID, `'  \u{`, 0, 1, 1}},
			"at least 1 hex digit required",
		},
		{
			`"\u{1ef.}"`,
			[]tok{{token.INVALID, `"\u{1ef.`, 0, 1, 1}},
			`missing '}'`,
		},
		{
			"'hello\nthere'",
			[]tok{{token.INVALID, "'hello\n", 0, 1, 1}},
			"illegal new line in string literal",
		},
		{
			`"foo\"`,
			[]tok{{token.INVALID, `"foo\"`, 0, 1, 1}},
			"illegal <eof> in string literal",
		},
		{
			`"\o"`,
			[]tok{{token.INVALID, `"\o`, 0, 1, 1}},
			"illegal escaped character",
		},
	}
	for i, test := range tests {
		name := fmt.Sprintf("Test %d", i+1)
		t.Run(name, func(t *testing.T) {
			scanner := New("test", []byte(test.text))
			for j, ts := range test.toks {
				next := scanner.Scan()
				if next == nil {
					t.Fatalf("Token %d: scan returns nil", j+1)
				}
				if !reflect.DeepEqual(next, ts.Token()) {
					t.Fatalf("Token %d: expected <%s>, got <%s>", j+1, tokenString(ts.Token()), tokenString(next))
				}
			}
			if scanner.ErrorMsg() != test.err {
				t.Fatalf("Wrong error message: expected %q, got %q", test.err, scanner.ErrorMsg())
			}
		})
	}
}
