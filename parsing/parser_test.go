package parsing

import (
	"reflect"
	"testing"

	"github.com/arnodel/golua/ast"
	"github.com/arnodel/golua/ops"
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

func TestParser_TableConstructor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.TableConstructor
		want1 *token.Token
	}{
		{
			name:  "Empty table",
			input: "{}",
			want:  ast.TableConstructor{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Single element table",
			input: "{1}",
			want: ast.TableConstructor{
				Fields: []ast.TableField{
					{Key: ast.NoTableKey{}, Value: ast.NewInt(1)},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Single element table with terminating semicolon",
			input: "{1;}",
			want: ast.TableConstructor{
				Fields: []ast.TableField{
					{Key: ast.NoTableKey{}, Value: ast.NewInt(1)},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Single element table with terminating comma",
			input: "{1,}",
			want: ast.TableConstructor{
				Fields: []ast.TableField{
					{Key: ast.NoTableKey{}, Value: ast.NewInt(1)},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Comma separated elements table",
			input: "{x=1,y=2}",
			want: ast.TableConstructor{
				Fields: []ast.TableField{
					{Key: str("x"), Value: ast.NewInt(1)},
					{Key: str("y"), Value: ast.NewInt(2)},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Semicolon separated elements table",
			input: `{x=1;y=2;}`,
			want: ast.TableConstructor{
				Fields: []ast.TableField{
					{Key: str("x"), Value: ast.NewInt(1)},
					{Key: str("y"), Value: ast.NewInt(2)},
				},
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.TableConstructor(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.TableConstructor() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.TableConstructor() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_ShortExp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.ExpNode
		want1 *token.Token
	}{
		{
			name:  "nil",
			input: "nil and",
			want:  ast.NilType{},
			want1: tok(token.KwAnd, "and"),
		},
		{
			name:  "true",
			input: "true or",
			want:  ast.Bool{Val: true},
			want1: tok(token.KwOr, "or"),
		},
		{
			name:  "false",
			input: "false or",
			want:  ast.Bool{},
			want1: tok(token.KwOr, "or"),
		},
		{
			name:  "decimal int",
			input: "1234",
			want:  ast.NewInt(1234),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "hex int",
			input: "0x1234",
			want:  ast.NewInt(0x1234),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "decimal float",
			input: "0.125",
			want:  ast.NewFloat(0.125),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "single quoted string",
			input: `'abc'`,
			want:  str("abc"),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "double quoted string",
			input: `"hello there"`,
			want:  str("hello there"),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "long string",
			input: `[=[[[hello]]]=]`,
			want:  str("[[hello]]"),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "...",
			input: `...`,
			want:  ast.EtcType{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "function",
			input: `function(x) end`,
			want: ast.Function{
				ParList: ast.ParList{Params: []ast.Name{name("x")}},
				Body:    ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "unop shortexp",
			input: `not true`,
			want:  &ast.UnOp{Op: ops.OpNot, Operand: ast.Bool{Val: true}},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "unop unop shortexp",
			input: `-#x`,
			want: &ast.UnOp{
				Op:      ops.OpNeg,
				Operand: &ast.UnOp{Op: ops.OpLen, Operand: name("x")},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "prefix exp",
			input: "(x)+2",
			want:  name("x"),
			want1: tok(token.SgPlus, "+"),
		},
		{
			name:  "power",
			input: "x^y+z",
			want:  ast.NewBinOp(name("x"), ops.OpPow, name("y")),
			want1: tok(token.SgPlus, "+"),
		},
		{
			name:  "-power (power tighter than unary op)",
			input: "-x^y+z",
			want: &ast.UnOp{
				Op:      ops.OpNeg,
				Operand: ast.NewBinOp(name("x"), ops.OpPow, name("y")),
			},
			want1: tok(token.SgPlus, "+"),
		},
		{
			name:  "x^y^z (right associative)",
			input: "x^y^z/",
			want: ast.NewBinOp(
				name("x"),
				ops.OpPow,
				ast.NewBinOp(name("y"), ops.OpPow, name("z")),
			),
			want1: tok(token.SgSlash, "/"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.ShortExp(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.ShortExp() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.ShortExp() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_FunctionDef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Function
		want1 *token.Token
	}{
		{
			name:  "no arguments",
			input: "() end",
			want: ast.Function{
				ParList: ast.ParList{},
				Body:    ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "etc only",
			input: "(...) end",
			want: ast.Function{
				ParList: ast.ParList{HasDots: true},
				Body:    ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "one arg",
			input: "(x) end",
			want: ast.Function{
				ParList: ast.ParList{
					Params: []ast.Name{name("x")},
				},
				Body: ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "two args",
			input: "(x, y) end",
			want: ast.Function{
				ParList: ast.ParList{
					Params: []ast.Name{name("x"), name("y")},
				},
				Body: ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "two args and etc",
			input: "(x, y, ...) end",
			want: ast.Function{
				ParList: ast.ParList{
					Params:  []ast.Name{name("x"), name("y")},
					HasDots: true,
				},
				Body: ast.NewBlockStat(nil, []ast.ExpNode{}),
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.FunctionDef(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.FunctionDef() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.FunctionDef() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Args(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ast.ExpNode
		want1 *token.Token
	}{
		{
			name:  "Empty brackets",
			input: "()",
			want:  []ast.ExpNode{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "1 arg in brackets",
			input: "(x)",
			want:  []ast.ExpNode{name("x")},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "2 args in brackets",
			input: "(x, y)",
			want:  []ast.ExpNode{name("x"), name("y")},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "table arg",
			input: "{x=1}",
			want: []ast.ExpNode{
				ast.TableConstructor{
					Fields: []ast.TableField{{Key: str("x"), Value: ast.NewInt(1)}},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "string arg",
			input: `"hello"`,
			want:  []ast.ExpNode{str("hello")},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "long string arg",
			input: `[[coucou]]`,
			want:  []ast.ExpNode{str("coucou")},
			want1: tok(token.EOF, ""),
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Args(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Args() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Args() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Field(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.TableField
		want1 *token.Token
	}{
		{
			name:  "[x]=y",
			input: "[false]=1,",
			want:  ast.TableField{Key: ast.Bool{}, Value: ast.NewInt(1)},
			want1: tok(token.SgComma, ","),
		},
		{
			name:  "x=y",
			input: "bar=42;",
			want:  ast.TableField{Key: str("bar"), Value: ast.NewInt(42)},
			want1: tok(token.SgSemicolon, ";"),
		},
		{
			name:  "exp",
			input: `'bonjour'`,
			want:  ast.TableField{Key: ast.NoTableKey{}, Value: str("bonjour")},
			want1: tok(token.EOF, ""),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Field(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Field() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Field() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Exp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.ExpNode
		want1 *token.Token
	}{
		{
			name:  "short expression",
			input: "-x.y)",
			want: &ast.UnOp{
				Op: ops.OpNeg,
				Operand: ast.IndexExp{
					Coll: name("x"),
					Idx:  str("y"),
				},
			},
			want1: tok(token.SgCloseBkt, ")"),
		},
		{
			name:  "binop",
			input: "x + y)",
			want:  ast.NewBinOp(name("x"), ops.OpAdd, name("y")),
			want1: tok(token.SgCloseBkt, ")"),
		},
		{
			name:  "2 binops of precedence",
			input: "x + y - z then",
			want: ast.NewBinOp(
				ast.NewBinOp(name("x"), ops.OpAdd, name("y")),
				ops.OpSub,
				name("z"),
			),
			want1: tok(token.KwThen, "then"),
		},
		{
			name:  "2 binops of decreasing precedence",
			input: "x * y + z then",
			want: ast.NewBinOp(
				ast.NewBinOp(name("x"), ops.OpMul, name("y")),
				ops.OpAdd,
				name("z"),
			),
			want1: tok(token.KwThen, "then"),
		},
		{
			name:  "2 binops of increasing precedence",
			input: "x | y + z then",
			want: ast.NewBinOp(
				name("x"),
				ops.OpBitOr,
				ast.NewBinOp(name("y"), ops.OpAdd, name("z")),
			),
			want1: tok(token.KwThen, "then"),
		},
		{
			name:  "3 binops of decreasing precedence",
			input: "x * y + z or t then",
			want: ast.NewBinOp(
				ast.NewBinOp(
					ast.NewBinOp(name("x"), ops.OpMul, name("y")),
					ops.OpAdd,
					name("z"),
				),
				ops.OpOr,
				name("t"),
			),
			want1: tok(token.KwThen, "then"),
		},
		{
			name:  "3 binops of increasing precedence",
			input: "x << y .. z % t]",
			want: ast.NewBinOp(
				name("x"),
				ops.OpShiftL,
				ast.NewBinOp(
					name("y"),
					ops.OpConcat,
					ast.NewBinOp(name("z"), ops.OpMod, name("t")),
				),
			),
			want1: tok(token.SgCloseSquareBkt, "]"),
		},
		{
			name:  "concat right associative",
			input: "x .. y .. z .. t until",
			want: ast.NewBinOp(
				name("x"),
				ops.OpConcat,
				ast.NewBinOp(
					name("y"),
					ops.OpConcat,
					ast.NewBinOp(name("z"), ops.OpConcat, name("t")),
				),
			),
			want1: tok(token.KwUntil, "until"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Exp(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Exp() got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Exp() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Block(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.BlockStat
		want1 *token.Token
	}{
		{
			name:  "Empty block ending in 'end'",
			input: "end",
			want:  ast.NewBlockStat(nil, nil),
			want1: tok(token.KwEnd, "end"),
		},
		{
			name:  "Empty block ending in 'else'",
			input: "else",
			want:  ast.NewBlockStat(nil, nil),
			want1: tok(token.KwElse, "else"),
		},
		{
			name:  "Empty block ending in 'elsif'",
			input: "elseif",
			want:  ast.NewBlockStat(nil, nil),
			want1: tok(token.KwElseIf, "elseif"),
		},
		{
			name:  "Empty block ending in 'until'",
			input: "until",
			want:  ast.NewBlockStat(nil, nil),
			want1: tok(token.KwUntil, "until"),
		},
		{
			name:  "Empty block ending in EOF",
			input: "",
			want:  ast.NewBlockStat(nil, nil),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Block with return",
			input: "break return 1",
			want: ast.NewBlockStat(
				[]ast.Stat{ast.BreakStat{}},
				[]ast.ExpNode{ast.NewInt(1)},
			),
			want1: tok(token.EOF, ""),
		},
		{
			name:  "Block with empty return",
			input: "break return end",
			want: ast.NewBlockStat(
				[]ast.Stat{ast.BreakStat{}},
				[]ast.ExpNode{},
			),
			want1: tok(token.KwEnd, "end"),
		},
		{
			name:  "Block without return",
			input: "break end",
			want: ast.NewBlockStat(
				[]ast.Stat{ast.BreakStat{}},
				nil,
			),
			want1: tok(token.KwEnd, "end"),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Block(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Block() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Block() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_If(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.IfStat
		want1 *token.Token
	}{
		{
			name:  "plain if ... then ... end",
			input: "if true then ; end",
			want: ast.IfStat{
				If: ast.CondStat{
					Cond: ast.Bool{Val: true},
					Body: ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "if ... then ... else ... end",
			input: "if cond then ; else ; end",
			want: ast.IfStat{
				If: ast.CondStat{
					Cond: name("cond"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				},
				Else: &ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "if ... then ... elseif ... then ... end",
			input: "if a then ; elseif b then ; end",
			want: ast.IfStat{
				If: ast.CondStat{
					Cond: name("a"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				},
				ElseIfs: []ast.CondStat{{
					Cond: name("b"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "if ... then ... elseif ... then ... else ... end",
			input: "if a then ; elseif b then ; else ; end",
			want: ast.IfStat{
				If: ast.CondStat{
					Cond: name("a"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				},
				ElseIfs: []ast.CondStat{{
					Cond: name("b"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				}},
				Else: &ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "if ... then ... elseif ... then ... elseif ... then ... end",
			input: "if a then ; elseif b then ; elseif c then ; end",
			want: ast.IfStat{
				If: ast.CondStat{
					Cond: name("a"),
					Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
				},
				ElseIfs: []ast.CondStat{
					{
						Cond: name("b"),
						Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
					},
					{
						Cond: name("c"),
						Body: ast.NewBlockStat([]ast.Stat{ast.EmptyStat{}}, nil),
					},
				},
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.If(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.If() got = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.If() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_For(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Stat
		want1 *token.Token
	}{
		{
			name:  "for x = 1, 2 do ; end",
			input: "for x = 1, 2 do ; end",
			want: &ast.ForStat{
				Var:   name("x"),
				Start: ast.NewInt(1),
				Stop:  ast.NewInt(2),
				Body:  ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "for x = 1, 2, 3 do ; end",
			input: "for x = 1, 2, 3 do ; end",
			want: &ast.ForStat{
				Var:   name("x"),
				Start: ast.NewInt(1),
				Stop:  ast.NewInt(2),
				Step:  ast.NewInt(3),
				Body:  ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "for in with one variable",
			input: "for i in X do ; end",
			want: &ast.ForInStat{
				Vars:   []ast.Name{name("i")},
				Params: []ast.ExpNode{name("X")},
				Body:   ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "for in with 3 variables",
			input: "for i, j, k in X do ; end",
			want: &ast.ForInStat{
				Vars:   []ast.Name{name("i"), name("j"), name("k")},
				Params: []ast.ExpNode{name("X")},
				Body:   ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "for in with 3 parameters",
			input: "for i in X,Y,Z do ; end",
			want: &ast.ForInStat{
				Vars:   []ast.Name{name("i")},
				Params: []ast.ExpNode{name("X"), name("Y"), name("Z")},
				Body:   ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.For(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.For() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.For() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Local(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Stat
		want1 *token.Token
	}{
		{
			name:  "local function definition",
			input: "local function f(x) return x end",
			want: ast.LocalFunctionStat{
				Name: name("f"),
				Function: ast.Function{
					Name: "f",
					ParList: ast.ParList{
						Params: []ast.Name{name("x")},
					},
					Body: ast.BlockStat{
						Return: []ast.ExpNode{name("x")},
					},
				},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "local single variable declaration with no value",
			input: "local x z",
			want:  ast.LocalStat{Names: []ast.Name{name("x")}},
			want1: tok(token.IDENT, "z"),
		},
		{
			name:  "local 3 variables declaration with no value",
			input: "local x, y, z",
			want:  ast.LocalStat{Names: []ast.Name{name("x"), name("y"), name("z")}},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "local 3 variables declaration with 1 value",
			input: "local x, y, z = 123",
			want: ast.LocalStat{
				Names:  []ast.Name{name("x"), name("y"), name("z")},
				Values: []ast.ExpNode{ast.NewInt(123)},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "local 2 variables declaration with 3 value",
			input: `local x, y = 123, "a", 'b'`,
			want: ast.LocalStat{
				Names:  []ast.Name{name("x"), name("y")},
				Values: []ast.ExpNode{ast.NewInt(123), str("a"), str("b")},
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Local(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Local() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Local() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_FunctionStat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Stat
		want1 *token.Token
	}{
		{
			name:  "function with plain name",
			input: "function foo() end",
			want: ast.AssignStat{
				Dest: []ast.Var{name("foo")},
				Src: []ast.ExpNode{ast.Function{
					Name: "foo",
					Body: ast.BlockStat{Return: []ast.ExpNode{}},
				}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "function with dotted name",
			input: "function foo.bar.baz() end",
			want: ast.AssignStat{
				Dest: []ast.Var{
					ast.IndexExp{
						Coll: ast.IndexExp{
							Coll: name("foo"),
							Idx:  str("bar"),
						},
						Idx: str("baz"),
					}},
				Src: []ast.ExpNode{ast.Function{
					Name: "baz",
					Body: ast.BlockStat{Return: []ast.ExpNode{}},
				}},
			},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "method",
			input: "function dog:bark(at) end",
			want: ast.AssignStat{
				Dest: []ast.Var{
					ast.IndexExp{
						Coll: name("dog"),
						Idx:  str("bark"),
					}},
				Src: []ast.ExpNode{ast.Function{
					Name:    "bark",
					ParList: ast.ParList{Params: []ast.Name{name("self"), name("at")}},
					Body:    ast.BlockStat{Return: []ast.ExpNode{}},
				}},
			},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.FunctionStat(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.FunctionStat() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.FunctionStat() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestParser_Stat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Stat
		want1 *token.Token
	}{
		{
			name:  "empty statement",
			input: ";",
			want:  ast.EmptyStat{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "break statement",
			input: "break",
			want:  ast.BreakStat{},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "goto statement",
			input: "goto start",
			want:  ast.GotoStat{Label: name("start")},
			want1: tok(token.EOF, ""),
		},
		{
			name:  "do ... end block",
			input: "do ; end bye",
			want:  ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			want1: tok(token.IDENT, "bye"),
		},
		{
			name:  "while block",
			input: "while true do ; end",
			want: ast.WhileStat{CondStat: ast.CondStat{
				Cond: ast.Bool{Val: true},
				Body: ast.BlockStat{Stats: []ast.Stat{ast.EmptyStat{}}},
			}},
			want1: tok(token.EOF, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{getToken: testScanner(tt.input)}
			got, got1 := p.Stat(p.Scan())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.Stat() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Parser.Stat() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
