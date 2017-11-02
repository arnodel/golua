package ast

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/arnodel/golua/ir"
	"github.com/arnodel/golua/token"
)

type ExpNode interface {
	Node
	CompileExp(c *Compiler) ir.Register
}

// Var is an l-value
type Var interface {
	ExpNode
}

type Float float64

func (f Float) CompileExp(c *Compiler) ir.Register {
	return EmitConstant(c, ir.Float(f))
}

func (f Float) HWrite(w HWriter) {
	w.Writef("%f", float64(f))
}

type Int int64

func (n Int) HWrite(w HWriter) {
	w.Writef("%d", int64(n))
}

func (n Int) CompileExp(c *Compiler) ir.Register {
	return EmitConstant(c, ir.Int(n))
}

type Bool bool

func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", bool(b))
}

func (b Bool) CompilerExp(c *Compiler) ir.Register {
	return EmitConstant(c, ir.Bool(b))
}

type String []byte

func (s String) HWrite(w HWriter) {
	w.Writef("%q", string(s))
}

func (s String) CompileExp(c *Compiler) ir.Register {
	return EmitConstant(c, ir.String(s))
}

type Name string

func (n Name) HWrite(w HWriter) {
	w.Writef(string(n))
}

func (n Name) CompileExp(c *Compiler) ir.Register {
	reg, ok := c.GetRegister(n)
	if ok {
		return reg
	}
	env, ok := c.GetRegister("_ENV")
	if !ok {
		panic("Cannot find _ENV")
	}
	idx := EmitConstant(c, ir.String(n))
	reg = c.NewRegister()
	c.Emit(ir.Lookup{
		Dst:   reg,
		Table: env,
		Index: idx,
	})
	return reg
}

type NilType struct{}

func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

func (n NilType) CompileExp(c *Compiler) ir.Register {
	return EmitConstant(c, ir.NilType{})
}

type EtcType struct{}

func (e EtcType) HWrite(w HWriter) {
	w.Writef("...")
}

func (e EtcType) CompileExp(c *Compiler) ir.Register {
	reg, ok := c.GetRegister("...")
	if ok {
		return reg
	}
	panic("... not defined")
}

var True Bool = true
var False Bool = false
var Nil NilType
var Etc EtcType

func NewName(id *token.Token) (Name, error) {
	return Name(id.Lit), nil
}

func NewNumber(id *token.Token) (ExpNode, error) {
	nstring := string(id.Lit)
	if strings.ContainsAny(nstring, ".eE") {
		f, err := strconv.ParseFloat(nstring, 64)
		if err != nil {
			return nil, err
		}
		return Float(f), nil
	}
	n, err := strconv.ParseInt(nstring, 0, 64)
	if err != nil {
		return nil, err
	}
	return Int(n), nil
}

var escapeSeqs = regexp.MustCompile(`\\\d{1,3}|\\[xX][0-9a-fA-F]{2}|\\[abtnvfr]|\\z\w*|\[uU]\{[0-9a-fA-F]+\}|\\.`)

func replaceEscapeSeq(e []byte) []byte {
	switch e[1] {
	case 'a':
		return []byte{7}
	case 'b':
		return []byte{8}
	case 't':
		return []byte{9}
	case 'n':
		return []byte{10}
	case 'v':
		return []byte{11}
	case 'f':
		return []byte{12}
	case 'r':
		return []byte{13}
	case 'z':
		return []byte{}
	case 'x', 'X':
		b, err := strconv.ParseInt(string(e[2:]), 16, 64)
		if err != nil {
			panic(err)
		}
		return []byte{byte(b)}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		b, err := strconv.ParseInt(string(e[2:]), 10, 64)
		if err != nil {
			panic(err)
		}
		return []byte{byte(b)}
	case 'u', 'U':
		i, err := strconv.ParseInt(string(e[3:len(e)-1]), 16, 32)
		if err != nil {
			panic(err)
		}
		return []byte(string(i))
	default:
		return e[1:]
	}
}

func NewString(id *token.Token) String {
	s := id.Lit
	return String(escapeSeqs.ReplaceAllFunc(s[1:len(s)-1], replaceEscapeSeq))
}
