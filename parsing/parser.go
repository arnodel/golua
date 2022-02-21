package parsing

import (
	"errors"
	"fmt"

	"github.com/arnodel/golua/luastrings"
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"

	"github.com/arnodel/golua/ast"
)

// Parser can parse lua statements or expressions
type Parser struct {
	scanner Scanner
}

type Scanner interface {
	Scan() *token.Token
	ErrorMsg() string
}

type Error struct {
	Got      *token.Token
	Expected string
}

func (e Error) Error() string {
	expected := e.Expected
	if e.Got.Type == token.INVALID {
		expected = "invalid token: " + expected
	} else if e.Got.Type == token.UNFINISHED {
		expected = "unexpected <eof>"
	} else if expected == "" {
		expected = "unexpected symbol"
	} else {
		expected = "expected " + expected
	}
	var tok string
	if e.Got.Type == token.EOF {
		tok = "<eof>"
	} else {
		tok = luastrings.Quote(string(e.Got.Lit), '\'')
	}
	return fmt.Sprintf("%d:%d: %s near %s", e.Got.Line, e.Got.Column, expected, tok)
}

// ParseExp takes in a function that returns tokens and builds an ExpNode for it
// (or returns an error).
func ParseExp(scanner Scanner) (exp ast.ExpNode, err error) {
	defer func() {
		if r := recover(); r != nil {
			exp = nil
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = errors.New("Unknown error")
			}
		}
	}()
	parser := &Parser{scanner}
	var t *token.Token
	exp, t = parser.Exp(parser.Scan())
	expectType(t, token.EOF, "<eof>")
	return
}

// ParseChunk takes in a function that returns tokens and builds a BlockStat for it
// (or returns an error).
func ParseChunk(scanner Scanner) (stat ast.BlockStat, err error) {
	defer func() {
		if r := recover(); r != nil {
			stat = ast.BlockStat{}
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = errors.New("Unknown error")
			}
		}
	}()
	parser := &Parser{scanner}
	var t *token.Token
	stat, t = parser.Block(parser.Scan())
	expectType(t, token.EOF, "<eof>")
	return
}

// Scan returns the next token.
func (p *Parser) Scan() *token.Token {
	tok := p.scanner.Scan()
	if tok.Type == token.INVALID {
		panic(Error{Got: tok, Expected: p.scanner.ErrorMsg()})
	}
	return tok
}

// Stat parses any statement.
func (p *Parser) Stat(t *token.Token) (ast.Stat, *token.Token) {
	switch t.Type {
	case token.SgSemicolon:
		return ast.NewEmptyStat(t), p.Scan()
	case token.KwBreak:
		return ast.NewBreakStat(t), p.Scan()
	case token.KwGoto:
		dest := p.Scan()
		expectIdent(dest)
		return ast.NewGotoStat(t, ast.NewName(dest)), p.Scan()
	case token.KwDo:
		stat, closer := p.Block(p.Scan())
		expectType(closer, token.KwEnd, "'end'")
		return stat, p.Scan()
	case token.KwWhile:
		cond, doTok := p.Exp(p.Scan())
		expectType(doTok, token.KwDo, "'do'")
		body, endTok := p.Block(p.Scan())
		expectType(endTok, token.KwEnd, "'end'")
		return ast.NewWhileStat(t, endTok, cond, body), p.Scan()
	case token.KwRepeat:
		body, untilTok := p.Block(p.Scan())
		expectType(untilTok, token.KwUntil, "'until'")
		cond, next := p.Exp(p.Scan())
		return ast.NewRepeatStat(t, body, cond), next
	case token.KwIf:
		return p.If(t)
	case token.KwFor:
		return p.For(t)
	case token.KwFunction:
		return p.FunctionStat(t)
	case token.KwLocal:
		return p.Local(t)
	case token.SgDoubleColon:
		name, t := p.Name(p.Scan())
		expectType(t, token.SgDoubleColon, "'::'")
		return ast.NewLabelStat(name), p.Scan()
	default:
		var exp ast.ExpNode
		exp, t = p.PrefixExp(t)
		switch e := exp.(type) {
		case ast.Stat:
			// This is a function call
			return e, t
		case ast.Var:
			// This should be the start of 'varlist = explist'
			vars := []ast.Var{e}
			var pexp ast.ExpNode
			for t.Type == token.SgComma {
				pexp, t = p.PrefixExp(p.Scan())
				if v, ok := pexp.(ast.Var); ok {
					vars = append(vars, v)
				} else {
					tokenError(t, "expected variable")
				}
			}
			expectType(t, token.SgAssign, "'='")
			exps, t := p.ExpList(p.Scan())
			return ast.NewAssignStat(vars, exps), t
		default:
			tokenError(t, "")
		}
	}
	return nil, nil
}

// If parses an if / then / else statement.  It assumes that t is the "if"
// token.
func (p *Parser) If(t *token.Token) (ast.IfStat, *token.Token) {
	cond, thenTok := p.Exp(p.Scan())
	expectType(thenTok, token.KwThen, "'then'")
	thenBlock, endTok := p.Block(p.Scan())
	ifStat := ast.NewIfStat(t, cond, thenBlock)
	for {
		switch endTok.Type {
		case token.KwElseIf:
			cond, thenTok = p.Exp(p.Scan())
			expectType(thenTok, token.KwThen, "'then'")
			thenBlock, endTok = p.Block(p.Scan())
			ifStat = ifStat.AddElseIf(cond, thenBlock)
		case token.KwEnd:
			return ifStat, p.Scan()
		case token.KwElse:
			elseBlock, elseTok := p.Block(p.Scan())
			expectType(elseTok, token.KwEnd, "'end'")
			ifStat = ifStat.WithElse(endTok, elseBlock)
			return ifStat, p.Scan()
		default:
			tokenError(t, "'elseif' or 'end' or 'else'")
		}
	}
}

// For parses a for in / for = statement.  It assumes that t is the "for" token.
func (p *Parser) For(t *token.Token) (ast.Stat, *token.Token) {
	name, nextTok := p.Name(p.Scan())
	if nextTok.Type == token.SgAssign {
		// Parse for Name = ...
		params := make([]ast.ExpNode, 3)
		params[0], nextTok = p.Exp(p.Scan())
		expectType(nextTok, token.SgComma, "','")
		params[1], nextTok = p.Exp(p.Scan())
		if nextTok.Type == token.SgComma {
			params[2], nextTok = p.Exp(p.Scan())
		} else {
			params[2] = ast.NewInt(1)
		}
		expectType(nextTok, token.KwDo, "'do'")
		body, endTok := p.Block(p.Scan())
		expectType(endTok, token.KwEnd, "'end'")
		forStat := ast.NewForStat(t, endTok, name, params, body)
		return forStat, p.Scan()
	}
	// Parse for namelist in explist ...
	names := []ast.Name{name}
	for nextTok.Type == token.SgComma {
		name, nextTok = p.Name(p.Scan())
		names = append(names, name)
	}
	expected := "'in'"
	if len(names) == 1 {
		expected = "'=' or 'in'"
	}
	expectType(nextTok, token.KwIn, expected)
	exp, nextTok := p.Exp(p.Scan())
	params := []ast.ExpNode{exp}
	for nextTok.Type == token.SgComma {
		exp, nextTok = p.Exp(p.Scan())
		params = append(params, exp)
	}
	expectType(nextTok, token.KwDo, "'do'")
	body, endTok := p.Block(p.Scan())
	expectType(endTok, token.KwEnd, "'end'")
	forInStat := ast.NewForInStat(t, endTok, names, params, body)
	return forInStat, p.Scan()

}

// Local parses a "local" statement (function definition of variable
// declaration).  It assumes that t is the "local" token.
func (p *Parser) Local(*token.Token) (ast.Stat, *token.Token) {
	t := p.Scan()
	if t.Type == token.KwFunction {
		name, t := p.Name(p.Scan())
		fx, t := p.FunctionDef(t)
		return ast.NewLocalFunctionStat(name, fx), t
	}
	// local namelist ['=' explist]
	nameAttrib, t := p.NameAttrib(t)
	nameAttribs := []ast.NameAttrib{nameAttrib}
	for t.Type == token.SgComma {
		nameAttrib, t = p.NameAttrib(p.Scan())
		nameAttribs = append(nameAttribs, nameAttrib)
	}
	var values []ast.ExpNode
	if t.Type == token.SgAssign {
		values, t = p.ExpList(p.Scan())
	}
	return ast.NewLocalStat(nameAttribs, values), t
}

// FunctionStat parses a function definition statement. It assumes that t is the
// "function" token.
func (p *Parser) FunctionStat(*token.Token) (ast.Stat, *token.Token) {
	name, t := p.Name(p.Scan())
	var v ast.Var = name
	var method ast.Name
	for t.Type == token.SgDot {
		name, t = p.Name(p.Scan())
		v = ast.NewIndexExp(v, name.AstString())
	}
	if t.Type == token.SgColon {
		method, t = p.Name(p.Scan())
	}
	fx, t := p.FunctionDef(t)
	return ast.NewFunctionStat(v, method, fx), t
}

// Block parses a block whose starting token (e.g. "do") has already been
// consumed. Returns the token that closes the block (e.g. "end"). So the caller
// should check that this is the right kind of closing token.
func (p *Parser) Block(t *token.Token) (ast.BlockStat, *token.Token) {
	var stats []ast.Stat
	var next ast.Stat
	for {
		switch t.Type {
		case token.KwReturn:
			ret, t := p.Return(t)
			return ast.NewBlockStat(stats, ret), t
		case token.KwEnd, token.KwElse, token.KwElseIf, token.KwUntil, token.EOF:
			return ast.NewBlockStat(stats, nil), t
		default:
			next, t = p.Stat(t)
			stats = append(stats, next)
		}
	}
}

// Return parses a return statement.
func (p *Parser) Return(*token.Token) ([]ast.ExpNode, *token.Token) {
	t := p.Scan()
	switch t.Type {
	case token.SgSemicolon:
		return []ast.ExpNode{}, p.Scan()
	case token.KwEnd, token.KwElse, token.KwElseIf, token.KwUntil, token.EOF:
		return []ast.ExpNode{}, t
	default:
		exps, t := p.ExpList(t)
		if t.Type == token.SgSemicolon {
			t = p.Scan()
		}
		return exps, t
	}
}

type item struct {
	exp ast.ExpNode
	op  ops.Op
	tok *token.Token
}

func mergepop(stack []item, it item) ([]item, item) {
	i := len(stack) - 1
	top := stack[i]
	top.exp = ast.NewBinOp(top.exp, it.op, it.tok, it.exp)
	return stack[:i], top
}

// Exp parses any expression.
func (p *Parser) Exp(t *token.Token) (ast.ExpNode, *token.Token) {
	var exp ast.ExpNode
	exp, t = p.ShortExp(t)
	var op ops.Op
	var opTok *token.Token
	var stack []item
	last := item{exp: exp}
	for t.Type.IsBinOp() {
		op = binopMap[t.Type]
		opTok = t
		exp, t = p.ShortExp(p.Scan())
		for len(stack) > 0 {
			pdiff := op.Precedence() - last.op.Precedence()
			if pdiff > 0 || (pdiff == 0 && op == ops.OpConcat) {
				break
			}
			stack, last = mergepop(stack, last)
		}
		stack = append(stack, last)
		last = item{exp: exp, op: op, tok: opTok}
	}
	// We are left with a stack of strictly increasing precedence
	for len(stack) > 0 {
		stack, last = mergepop(stack, last)
	}
	return last.exp, t
}

// ShortExp parses an expression which is either atomic, a unary operation, a
// prefix expression or a power operation (right associatively composed). In
// other words, any expression that doesn't contain a binary operator.
func (p *Parser) ShortExp(t *token.Token) (ast.ExpNode, *token.Token) {
	var exp ast.ExpNode
	switch t.Type {
	case token.KwNil:
		exp, t = ast.NewNil(t), p.Scan()
	case token.KwTrue:
		exp, t = ast.True(t), p.Scan()
	case token.KwFalse:
		exp, t = ast.False(t), p.Scan()
	case token.NUMDEC, token.NUMHEX:
		n, err := ast.NewNumber(t)
		if err != nil {
			panic(err)
		}
		exp, t = n, p.Scan()
	case token.STRING:
		s, err := ast.NewString(t)
		if err != nil {
			panic(err)
		}
		exp, t = s, p.Scan()
	case token.LONGSTRING:
		exp, t = ast.NewLongString(t), p.Scan()
	case token.SgOpenBrace:
		exp, t = p.TableConstructor(t)
	case token.SgEtc:
		exp, t = ast.NewEtc(t), p.Scan()
	case token.KwFunction:
		exp, t = p.FunctionDef(p.Scan())
	case token.SgMinus, token.KwNot, token.SgHash, token.SgTilde:
		// A unary operator!
		opTok := t
		exp, t = p.ShortExp(p.Scan())
		exp = ast.NewUnOp(opTok, unopMap[opTok.Type], exp)
	default:
		exp, t = p.PrefixExp(t)
	}
	if t.Type == token.SgHat {
		var pow ast.ExpNode
		pow, t = p.ShortExp(p.Scan())
		exp = ast.NewBinOp(exp, ops.OpPow, t, pow)
	}
	return exp, t
}

var unopMap = map[token.Type]ops.Op{
	token.SgMinus: ops.OpNeg,
	token.KwNot:   ops.OpNot,
	token.SgHash:  ops.OpLen,
	token.SgTilde: ops.OpBitNot,
}

var binopMap = map[token.Type]ops.Op{
	token.KwOr:  ops.OpOr,
	token.KwAnd: ops.OpAnd,

	token.SgLess:         ops.OpLt,
	token.SgLessEqual:    ops.OpLeq,
	token.SgGreater:      ops.OpGt,
	token.SgGreaterEqual: ops.OpGeq,
	token.SgEqual:        ops.OpEq,
	token.SgNotEqual:     ops.OpNeq,

	token.SgPipe:      ops.OpBitOr,
	token.SgTilde:     ops.OpBitXor,
	token.SgAmpersand: ops.OpBitAnd,

	token.SgShiftLeft:  ops.OpShiftL,
	token.SgShiftRight: ops.OpShiftR,

	token.SgConcat: ops.OpConcat,

	token.SgPlus:  ops.OpAdd,
	token.SgMinus: ops.OpSub,

	token.SgStar:       ops.OpMul,
	token.SgSlash:      ops.OpDiv,
	token.SgSlashSlash: ops.OpFloorDiv,
	token.SgPct:        ops.OpMod,

	token.SgHat: ops.OpPow,
}

// FunctionDef parses a function definition expression.
func (p *Parser) FunctionDef(startTok *token.Token) (ast.Function, *token.Token) {
	expectType(startTok, token.SgOpenBkt, "'('")
	t := p.Scan()
	var names []ast.Name
	hasEtc := false
ParamsLoop:
	for {
		switch t.Type {
		case token.IDENT:
			names = append(names, ast.NewName(t))
			t = p.Scan()
			if t.Type != token.SgComma {
				break ParamsLoop
			}
			t = p.Scan()
		case token.SgEtc:
			hasEtc = true
			t = p.Scan()
			break ParamsLoop
		case token.SgCloseBkt:
			break ParamsLoop
		default:
			tokenError(t, "")
		}
	}
	expectType(t, token.SgCloseBkt, "')'")
	body, endTok := p.Block(p.Scan())
	expectType(endTok, token.KwEnd, "'end'")
	def := ast.NewFunction(startTok, endTok, ast.NewParList(names, hasEtc), body)
	return def, p.Scan()
}

// PrefixExp parses an expression made of a name or and expression in brackets
// followed by zero or more indexing operations or function applications.
func (p *Parser) PrefixExp(t *token.Token) (ast.ExpNode, *token.Token) {
	var exp ast.ExpNode
	switch t.Type {
	case token.SgOpenBkt:
		exp, t = p.Exp(p.Scan())
		if f, ok := exp.(ast.FunctionCall); ok {
			exp = f.InBrackets()
		}
		expectType(t, token.SgCloseBkt, "')'")
	case token.IDENT:
		exp = ast.NewName(t)
	default:
		tokenError(t, "")
	}
	t = p.Scan()
	for {
		switch t.Type {
		case token.SgOpenSquareBkt:
			var idxExp ast.ExpNode
			idxExp, t = p.Exp(p.Scan())
			expectType(t, token.SgCloseSquareBkt, "']'")
			t = p.Scan()
			exp = ast.NewIndexExp(exp, idxExp)
		case token.SgDot:
			var name ast.Name
			name, t = p.Name(p.Scan())
			exp = ast.NewIndexExp(exp, name.AstString())
		case token.SgColon:
			var name ast.Name
			var args []ast.ExpNode
			name, t = p.Name(p.Scan())
			args, t = p.Args(t)
			if args == nil {
				tokenError(t, "expected function arguments")
			}
			exp = ast.NewFunctionCall(exp, name, args)
		default:
			var args []ast.ExpNode
			args, t = p.Args(t)
			if args == nil {
				return exp, t
			}
			exp = ast.NewFunctionCall(exp, ast.Name{}, args)
		}
	}
}

// Args parses the arguments of a function call. It returns nil rather than
// panicking if it couldn't parse arguments.
func (p *Parser) Args(t *token.Token) ([]ast.ExpNode, *token.Token) {
	switch t.Type {
	case token.SgOpenBkt:
		t = p.Scan()
		if t.Type == token.SgCloseBkt {
			return []ast.ExpNode{}, p.Scan()
		}
		args, t := p.ExpList(t)
		expectType(t, token.SgCloseBkt, "')'")
		return args, p.Scan()
	case token.SgOpenBrace:
		arg, t := p.TableConstructor(t)
		return []ast.ExpNode{arg}, t
	case token.STRING:
		arg, err := ast.NewString(t)
		if err != nil {
			panic(err)
		}
		return []ast.ExpNode{arg}, p.Scan()
	case token.LONGSTRING:
		return []ast.ExpNode{ast.NewLongString(t)}, p.Scan()
	}
	return nil, t
}

// ExpList parses a comma separated list of expressions.
func (p *Parser) ExpList(t *token.Token) ([]ast.ExpNode, *token.Token) {
	var exp ast.ExpNode
	exp, t = p.Exp(t)
	exps := []ast.ExpNode{exp}
	for t.Type == token.SgComma {
		exp, t = p.Exp(p.Scan())
		exps = append(exps, exp)
	}
	return exps, t
}

// TableConstructor parses a table constructor.
func (p *Parser) TableConstructor(opTok *token.Token) (ast.TableConstructor, *token.Token) {
	t := p.Scan()
	var fields []ast.TableField
	if t.Type != token.SgCloseBrace {
		var field ast.TableField
		field, t = p.Field(t)
		fields = []ast.TableField{field}
		for t.Type == token.SgComma || t.Type == token.SgSemicolon {
			t = p.Scan()
			if t.Type == token.SgCloseBrace {
				break
			}
			field, t = p.Field(t)
			fields = append(fields, field)
		}
	}
	expectType(t, token.SgCloseBrace, "'}'")
	return ast.NewTableConstructor(opTok, t, fields), p.Scan()
}

// Field parses a table constructor field.
func (p *Parser) Field(t *token.Token) (ast.TableField, *token.Token) {
	var key ast.ExpNode = ast.NoTableKey{}
	var val ast.ExpNode
	if t.Type == token.SgOpenSquareBkt {
		key, t = p.Exp(p.Scan())
		expectType(t, token.SgCloseSquareBkt, "']'")
		expectType(p.Scan(), token.SgAssign, "'='")
		val, t = p.Exp(p.Scan())
	} else {
		val, t = p.Exp(t)
		if t.Type == token.SgAssign {
			if name, ok := val.(ast.Name); !ok {
				tokenError(t, "")
			} else {
				key = name.AstString()
				val, t = p.Exp(p.Scan())
			}
		}
	}
	return ast.NewTableField(key, val), t
}

// Name parses a name.
func (p *Parser) Name(t *token.Token) (ast.Name, *token.Token) {
	expectIdent(t)
	return ast.NewName(t), p.Scan()
}

func (p *Parser) NameAttrib(t *token.Token) (ast.NameAttrib, *token.Token) {
	name, t := p.Name(t)
	attrib := ast.NoAttrib
	var attribName *ast.Name
	if t.Type == token.SgLess {
		attribTok := p.Scan()
		attribName = new(ast.Name)
		*attribName, t = p.Name(attribTok)
		switch attribName.Val {
		case "const":
			attrib = ast.ConstAttrib
		case "close":
			attrib = ast.CloseAttrib
		default:
			tokenError(attribTok, "'const' or 'close'")
		}
		expectType(t, token.SgGreater, "'>'")
		t = p.Scan()
	}
	return ast.NewNameAttrib(name, attribName, attrib), t
}

func expectIdent(t *token.Token) {
	expectType(t, token.IDENT, "name")
}

func expectType(t *token.Token, tp token.Type, expected string) {
	if t.Type != tp {
		panic(Error{Got: t, Expected: expected})
	}
}

func tokenError(t *token.Token, expected string) {
	panic(Error{Got: t, Expected: expected})
}
