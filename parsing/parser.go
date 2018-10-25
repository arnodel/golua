package parsing

import (
	"github.com/arnodel/golua/ops"
	"github.com/arnodel/golua/token"

	"github.com/arnodel/golua/ast"
)

type Parser struct {
	getToken func() *token.Token
}

func (p *Parser) Scan() *token.Token {
	return p.getToken()
}

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
		expectType(closer, token.KwEnd)
		return stat, p.Scan()
	case token.KwWhile:
		cond, doTok := p.Exp(p.Scan())
		expectType(doTok, token.KwDo)
		body, endTok := p.Block(p.Scan())
		expectType(endTok, token.KwEnd)
		return ast.NewWhileStat(t, endTok, cond, body), p.Scan()
	case token.KwRepeat:
		body, untilTok := p.Block(p.Scan())
		expectType(untilTok, token.KwUntil)
		cond, next := p.Exp(p.Scan())
		return ast.NewRepeatStat(t, body, cond), next
	case token.KwIf:
		return p.If(t)
	case token.KwFor:
		return p.For(t)
	case token.KwFunction:
		name, t := p.Name(p.Scan())
		var v ast.Var = name
		var method ast.Name
		for t.Type == token.SgDot {
			name, t = p.Name(t)
			v = ast.NewIndexExp(v, name.AstString())
		}
		if t.Type == token.SgColon {
			method, t = p.Name(p.Scan())
		}
		fx, t := p.FunctionDef(t)
		return ast.NewFunctionStat(name, method, fx), t
	case token.KwLocal:
		return p.Local(t)
	case token.SgDoubleColon:
		name, t := p.Name(p.Scan())
		expectType(t, token.SgDoubleColon)
		return ast.NewLabelStat(name), p.Scan()
	default:
		exp, t := p.PrefixExp(t)
		switch e := exp.(type) {
		case ast.Stat:
			// This is a function call
			return e, t
		case ast.Var:
			// This should be the start of 'varlist = explist'
			vars := []ast.Var{e}
			var pexp ast.ExpNode
			for t.Type == token.SgComma {
				pexp, t = p.PrefixExp(t)
				if v, ok := pexp.(ast.Var); ok {
					vars = append(vars, v)
				} else {
					panic("Expected var")
				}
			}
			expectType(t, token.SgAssign)
			exps, t := p.ExpList(t)
			return ast.NewAssignStat(vars, exps), t
		default:
			panic("Expected something else") // TODO
		}
	}
}

// If parses an if / then / else statement.  It assumes that t is the "if"
// token.
func (p *Parser) If(t *token.Token) (ast.IfStat, *token.Token) {
	ifStat := ast.NewIfStat(nil)
	cond, thenTok := p.Exp(p.Scan())
	expectType(thenTok, token.KwThen)
	thenBlock, endTok := p.Block(p.Scan())
	ifStat = ifStat.AddIf(t, cond, thenBlock)
	for {
		switch endTok.Type {
		case token.KwElseIf:
			cond, thenTok = p.Exp(p.Scan())
			expectType(thenTok, token.KwThen)
			thenBlock, endTok = p.Block(p.Scan())
			ifStat = ifStat.AddElseIf(cond, thenBlock)
		case token.KwEnd:
			return ifStat, p.Scan()
		case token.KwElse:
			elseBlock, elseTok := p.Block(p.Scan())
			expectType(elseTok, token.KwEnd)
			ifStat = ifStat.AddElse(endTok, elseBlock)
			return ifStat, p.Scan()
		default:
			panic("Expected elseif, end or else")
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
		expectType(nextTok, token.SgComma)
		params[1], nextTok = p.Exp(p.Scan())
		if nextTok.Type == token.SgComma {
			params[2], nextTok = p.Exp(p.Scan())
		}
		expectType(nextTok, token.KwDo)
		body, endTok := p.Block(p.Scan())
		expectType(endTok, token.KwEnd)
		forStat := ast.NewForStat(t, endTok, name, params, body)
		return forStat, p.Scan()
	}
	// Parse for namelist in explist ...
	names := []ast.Name{name}
	for nextTok.Type == token.SgComma {
		name, nextTok = p.Name(p.Scan())
		names = append(names, name)
	}
	expectType(nextTok, token.KwIn)
	exp, nextTok := p.Exp(p.Scan())
	params := []ast.ExpNode{exp}
	for nextTok.Type == token.SgComma {
		exp, nextTok = p.Exp(p.Scan())
		params = append(params, exp)
	}
	expectType(nextTok, token.KwDo)
	body, endTok := p.Block(p.Scan())
	expectType(endTok, token.KwEnd)
	forInStat := ast.NewForInStat(t, endTok, names, params, body)
	return forInStat, p.Scan()

}

// Local parses a "local" statement (function definition of variable
// declaration).  It assumes that t is the "local" token.
func (p *Parser) Local(t *token.Token) (ast.Stat, *token.Token) {
	t = p.Scan()
	if t.Type == token.KwFunction {
		name, t := p.Name(p.Scan())
		fx, t := p.FunctionDef(t)
		return ast.NewLocalFunctionStat(name, fx), t
	}
	// local namelist ['=' explist]
	name, t := p.Name(t)
	names := []ast.Name{name}
	for t.Type == token.SgComma {
		name, t = p.Name(p.Scan())
		names = append(names, name)
	}
	var values []ast.ExpNode
	if t.Type == token.SgAssign {
		values, t = p.ExpList(p.Scan())
	}
	return ast.NewLocalStat(names, values), t
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
func (p *Parser) Return(t *token.Token) ([]ast.ExpNode, *token.Token) {
	t = p.Scan()
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
}

func mergepop(stack []item, it item) ([]item, item) {
	i := len(stack) - 1
	top := stack[i]
	top.exp = ast.NewBinOp(top.exp, it.op, it.exp)
	return stack[:i], top
}

// Exp parses any expression.
func (p *Parser) Exp(t *token.Token) (ast.ExpNode, *token.Token) {
	var exp ast.ExpNode
	exp, t = p.ShortExp(t)
	var op ops.Op
	var stack []item
	last := item{exp, op}
	for t.Type.IsBinOp() {
		op = binopMap[t.Type]
		exp, t = p.ShortExp(p.Scan())
		for len(stack) > 0 {
			pdiff := op.Precedence() - last.op.Precedence()
			if pdiff > 0 || (pdiff == 0 && op == ops.OpConcat) {
				break
			}
			stack, last = mergepop(stack, last)
		}
		stack = append(stack, last)
		last = item{exp, op}
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
		exp, t = ast.Nil(t), p.Scan()
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
	case token.SgEtc:
		exp, t = ast.Etc(t), p.Scan()
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
		exp = ast.NewBinOp(exp, ops.OpPow, pow)
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
	expectType(startTok, token.SgOpenBkt)
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
			panic("invalid param list")
		}
	}
	expectType(t, token.SgCloseBkt)
	body, endTok := p.Block(p.Scan())
	expectType(endTok, token.KwEnd)
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
		// TODO: put function call in brackets
		expectType(t, token.SgCloseBkt)
	case token.IDENT:
		exp = ast.NewName(t)
	default:
		panic("Expected '(' or name")
	}
	t = p.Scan()
	for {
		switch t.Type {
		case token.SgOpenSquareBkt:
			var idxExp ast.ExpNode
			idxExp, t = p.Exp(p.Scan())
			expectType(t, token.SgCloseSquareBkt)
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
				panic("Expected arguments")
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
		expectType(t, token.SgCloseBkt)
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
	expectType(t, token.SgCloseBrace)
	return ast.NewTableConstructor(opTok, t, fields), p.Scan()
}

// Field parses a table constructor field.
func (p *Parser) Field(t *token.Token) (ast.TableField, *token.Token) {
	var key ast.ExpNode = ast.NoTableKey{}
	var val ast.ExpNode
	switch t.Type {
	case token.SgOpenSquareBkt:
		key, t = p.Exp(p.Scan())
		expectType(t, token.SgCloseSquareBkt)
		expectType(p.Scan(), token.SgAssign)
		val, t = p.Exp(p.Scan())
	case token.IDENT:
		var name ast.Name
		name, t = p.Name(t)
		val = name.AstString()
		if t.Type == token.SgAssign {
			key = val
			val, t = p.Exp(p.Scan())
		}
	default:
		val, t = p.Exp(t)
	}
	return ast.NewTableField(key, val), t
}

// Name parses a name.
func (p *Parser) Name(t *token.Token) (ast.Name, *token.Token) {
	expectIdent(t)
	return ast.NewName(t), p.Scan()
}

func expectIdent(t *token.Token) {
	if t.Type != token.IDENT {
		panic("Expected ident")
	}
}

func expectType(t *token.Token, tp token.Type) {
	if t.Type != tp {
		panic("Expected other type")
	}
}
