package parsing

import (
	"reflect"
	"testing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/scanner"
	"github.com/arnodel/golua/token"
)

func testScanner(src string) func() *token.Token {
	s := scanner.New("test", []byte(src))
	return func() *token.Token {
		tok := s.Scan()
		if tok != nil {
			tok.Pos = token.Pos{Offset: -1}
		}
		return tok
	}
}

func tok(tp token.Type, lit string) *token.Token {
	return &token.Token{
		Type: tp,
		Lit:  []byte(lit),
		Pos:  token.Pos{Offset: -1},
	}
}

func name(s string) ast.Name {
	return ast.Name{Val: s}
}

func str(s string) ast.String {
	return ast.String{Val: []byte(s)}
}
func TestParser_Return(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ast.ExpNode
		want1 *token.Token
	}{
		{
			name:  "Bare return without semicolon",
			input: "return",
			want:  []ast.ExpNode{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Bare return with semicolon",
			input: "return;",
			want:  []ast.ExpNode{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Single return without semicolon",
			input: "return 1",
			want:  []ast.ExpNode{ast.NewInt(1)},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Single return with semicolon",
			input: "return 42; end",
			want:  []ast.ExpNode{ast.NewInt(42)},
			want1: tok(token.KwEnd, "end"),
		},
		{
			name:  "Double return with semicolon",
			input: "return 42, true end",
			want:  []ast.ExpNode{ast.NewInt(42), ast.Bool{Val: true}},
			want1: tok(token.KwEnd, "end"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Return(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Return() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Return() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_PrefixExp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.ExpNode
		want1 *token.Token
	}{
		{
			name:  "name",
			input: "abc",
			want:  ast.Name{Val: "abc"},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "name followed by binop",
			input: "abc +",
			want:  ast.Name{Val: "abc"},
			want1: tok(token.SgPlus, "+"),
		},
		{
			name:  "Exp in brackets",
			input: "(false)",
			want:  ast.Bool{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "index",
			input: "x[1] then",
			want:  ast.IndexExp{Coll: ast.Name{Val: "x"}, Idx: ast.NewInt(1)},
			want1: tok(token.KwThen, "then"),
		},
		{
			name:  "dot",
			input: "foo.bar ..",
			want:  ast.IndexExp{Coll: name("foo"), Idx: str("bar")},
			want1: tok(token.SgConcat, ".."),
		},
		{
			name:  "method call",
			input: "foo:bar(1)",
			want:  ast.NewFunctionCall(name("foo"), name("bar"), []ast.ExpNode{ast.NewInt(1)}),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "call",
			input: `f(x, "y") /`,
			want:  ast.NewFunctionCall(name("f"), ast.Name{}, []ast.ExpNode{name("x"), str("y")}),
			want1: tok(token.SgSlash, "/"),
		},
		{
			name:  "chain index, dot, function call, method call",
			input: "x[1].abc():meth(1)",
			want: ast.NewFunctionCall(
				ast.NewFunctionCall(
					ast.IndexExp{
						Coll: ast.IndexExp{Coll: name("x"), Idx: ast.NewInt(1)},
						Idx:  str("abc"),
					},
					ast.Name{},
					[]ast.ExpNode{},
				),
				name("meth"),
				[]ast.ExpNode{ast.NewInt(1)},
			),
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.PrefixExp(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.PrefixExp() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.PrefixExp() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
