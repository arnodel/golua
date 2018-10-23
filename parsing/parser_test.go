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

func TestParser_Return(t *testing.T) {
	type fields struct {
		getToken func() *token.Token
	}
	type args struct {
		t *token.Token
	}
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
		// TODO: Add test cases.
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
