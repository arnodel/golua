package scanner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/arnodel/golua/token"
)

type tok struct {
	tp, lit   string
	pos, l, c int
}

func (t tok) Token() *token.Token {
	tp := t.tp
	if tp == "" {
		tp = t.lit
	}
	line := t.l
	if line == 0 {
		line = 1
	}
	col := t.c
	if col == 0 {
		col = 1
	}
	return &token.Token{
		Type: token.TokMap.Type(tp),
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
				{"ident", "hello", 0, 1, 1},
				{"ident", "there", 6, 1, 7},
				{"$", "", 11, 1, 12},
			},
			"",
		},
		{
			`123.45 "abc" -0xff`,
			[]tok{
				{"numdec", "123.45", 0, 1, 1},
				{"string", `"abc"`, 7, 1, 8},
				{"-", "-", 13, 1, 14},
				{"numhex", "0xff", 14, 1, 15},
				{"$", "", 18, 1, 19},
			},
			"",
		},
		{
			`...,:<=}///  `,
			[]tok{
				{"...", "...", 0, 1, 1},
				{",", ",", 3, 1, 4},
				{":", ":", 4, 1, 5},
				{"<=", "<=", 5, 1, 6},
				{"}", "}", 7, 1, 8},
				{"//", "//", 8, 1, 9},
				{"/", "/", 10, 1, 11},
				{"$", "", 13, 1, 14},
			},
			"",
		},
		// Token errors
		{
			`abc?xyz`,
			[]tok{
				{"ident", "abc", 0, 1, 1},
				{"INVALID", "?", 3, 1, 4},
			},
			"Illegal character",
		},
		// Number errors
		{
			"123zabc",
			[]tok{{"INVALID", "123", 0, 1, 1}},
			"Illegal character following number",
		},
		{
			"0x10abP(",
			[]tok{{"INVALID", "0x10abP", 0, 1, 1}},
			"Digit required after exponent",
		},
		//
		// Long brackets
		//
		{
			`if[[hello]"there]]`,
			[]tok{
				{"if", "if", 0, 1, 1},
				{"longstring", `[[hello]"there]]`, 2, 1, 3},
			},
			"",
		},
		{
			"[==[xy\n\n]]===]==]123.34",
			[]tok{
				{"longstring", "[==[xy\n\n]]===]==]", 0, 1, 1},
				{"numdec", "123.34", 17, 3, 10},
			},
			"",
		},
		{
			`1e3--[=[stuff--[[inner]] ]=]'"o"'`,
			[]tok{
				{"numdec", "1e3", 0, 1, 1},
				{"string", `'"o"'`, 28, 1, 29},
			},
			"",
		},
		// Long bracket errors
		{
			`[==!!!`,
			[]tok{{"INVALID", `[==!`, 0, 1, 1}},
			"Expected opening long bracket",
		},
		{
			`   [===[foo]==]`,
			[]tok{{"INVALID", `[===[foo]==]`, 3, 1, 4}},
			"Illegal EOF in long bracket of level 3",
		},
		//
		// Short strings
		//
		{
			"'\\u{123a}\\xff\\\n\\''",
			[]tok{
				{"string", "'\\u{123a}\\xff\\\n\\''", 0, 1, 1},
				{"$", "", 18, 2, 4},
			},
			"",
		},
		// Short string errors
		{
			`"\x2w"`,
			[]tok{{"INVALID", `"\x2`, 0, 1, 1}},
			`\x must be followed by 2 hex digits`,
		},
		{
			`'\uz`,
			[]tok{{"INVALID", `'\uz`, 0, 1, 1}},
			`\u must be followed by '{'`,
		},
		{
			`'  \u{l}''`,
			[]tok{{"INVALID", `'  \u{`, 0, 1, 1}},
			"At least 1 hex digit required",
		},
		{
			`"\u{1ef.}"`,
			[]tok{{"INVALID", `"\u{1ef.`, 0, 1, 1}},
			`Missing '}'`,
		},
		{
			"'hello\nthere'",
			[]tok{{"INVALID", "'hello\n", 0, 1, 1}},
			"Illegal new line in string literal",
		},
		{
			`"foo\"`,
			[]tok{{"INVALID", `"foo\"`, 0, 1, 1}},
			"Illegal EOF in string literal",
		},
	}
	for i, test := range tests {
		name := fmt.Sprintf("Test %d", i+1)
		t.Run(name, func(t *testing.T) {
			scanner := lex("test", []byte(test.text))
			for j, ts := range test.toks {
				next := scanner.Scan()
				fmt.Println("SCANNED", next)
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
