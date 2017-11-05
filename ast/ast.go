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
	CompileExp(*Compiler, ir.Register) ir.Register
}

func CompileExp(c *Compiler, e ExpNode) ir.Register {
	r1 := c.GetFreeRegister()
	r2 := e.CompileExp(c, r1)
	if r1 != r2 {
		return r2
	}
	return r1
}

// Var is an l-value
type Var interface {
	ExpNode
	CompileAssign(*Compiler, ir.Register)
}

type Float float64

func (f Float) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	EmitConstant(c, ir.Float(f), dst)
	return dst
}

func (f Float) HWrite(w HWriter) {
	w.Writef("%f", float64(f))
}

type Int int64

func (n Int) HWrite(w HWriter) {
	w.Writef("%d", int64(n))
}

func (n Int) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	EmitConstant(c, ir.Int(n), dst)
	return dst
}

type Bool bool

func (b Bool) HWrite(w HWriter) {
	w.Writef("%t", bool(b))
}

func (b Bool) CompilerExp(c *Compiler, dst ir.Register) ir.Register {
	EmitConstant(c, ir.Bool(b), dst)
	return dst
}

type String []byte

func (s String) HWrite(w HWriter) {
	w.Writef("%q", string(s))
}

func (s String) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	EmitConstant(c, ir.String(s), dst)
	return dst
}

type Name string

func (n Name) HWrite(w HWriter) {
	w.Writef(string(n))
}

func (n Name) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	reg, ok := c.GetRegister(n)
	if ok {
		return reg
	}
	return IndexExp{Name("_ENV"), String(n)}.CompileExp(c, dst)
}

func (n Name) CompileAssign(c *Compiler, src ir.Register) {
	reg, ok := c.GetRegister(n)
	if ok {
		EmitMove(c, reg, src)
		return
	}
	IndexExp{Name("_ENV"), String(n)}.CompileAssign(c, src)
}

type NilType struct{}

func (n NilType) HWrite(w HWriter) {
	w.Writef("nil")
}

func (n NilType) CompileExp(c *Compiler, dst ir.Register) ir.Register {
	EmitConstant(c, ir.NilType{}, dst)
	return dst
}

type EtcType struct{}

func (e EtcType) HWrite(w HWriter) {
	w.Writef("...")
}

func (e EtcType) CompileExp(c *Compiler, dst ir.Register) ir.Register {
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
