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
			`123.45 "abc" -0xff`,
			[]tok{
				{token.NUMDEC, "123.45", 0, 1, 1},
				{token.STRING, `"abc"`, 7, 1, 8},
				{token.SgMinus, "-", 13, 1, 14},
				{token.NUMHEX, "0xff", 14, 1, 15},
				{token.EOF, "", 18, 1, 19},
			},
			"",
		},
		{
			`...,:<=}///  `,
			[]tok{
				{token.SgEtc, "...", 0, 1, 1},
				{token.SgComma, ",", 3, 1, 4},
				{token.SgColon, ":", 4, 1, 5},
				{token.SgLessEqual, "<=", 5, 1, 6},
				{token.SgCloseBrace, "}", 7, 1, 8},
				{token.SgSlashSlash, "//", 8, 1, 9},
				{token.SgSlash, "/", 10, 1, 11},
				{token.EOF, "", 13, 1, 14},
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
			"Illegal character",
		},
		// Number errors
		{
			"123zabc",
			[]tok{{token.INVALID, "123", 0, 1, 1}},
			"Illegal character following number",
		},
		{
			"0x10abP(",
			[]tok{{token.INVALID, "0x10abP", 0, 1, 1}},
			"Digit required after exponent",
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
			"Expected opening long bracket",
		},
		{
			`   [===[foo]==]`,
			[]tok{{token.INVALID, `[===[foo]==]`, 3, 1, 4}},
			"Illegal EOF in long bracket of level 3",
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
			"At least 1 hex digit required",
		},
		{
			`"\u{1ef.}"`,
			[]tok{{token.INVALID, `"\u{1ef.`, 0, 1, 1}},
			`Missing '}'`,
		},
		{
			"'hello\nthere'",
			[]tok{{token.INVALID, "'hello\n", 0, 1, 1}},
			"Illegal new line in string literal",
		},
		{
			`"foo\"`,
			[]tok{{token.INVALID, `"foo\"`, 0, 1, 1}},
			"Illegal EOF in string literal",
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
